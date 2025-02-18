package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	lotusClient "github.com/filecoin-project/lotus/api/client"
	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/frrist/pdp-mon/contract"
)

// ProofSetReport aggregates all relevant state for one proof set.
type ProofSetReport struct {
	ID                 *big.Int
	Owner1             common.Address
	Owner2             common.Address
	Listener           common.Address
	IsLive             bool
	NextChallengeEpoch *big.Int
	LastProvenEpoch    *big.Int
	ProvingDeadline    *big.Int
	ProvenThisPeriod   bool
	Status             string
}

func main() {
	ctx := context.Background()

	// TODO(forrest): these values should come from flags, ideally we use a single wss endpoint.
	eClient, err := ethclient.Dial("https://api.calibration.node.glif.io/rpc/v1")
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum client: %v", err)
	}
	defer eClient.Close()

	// TODO(forrest): these values should come from flags, ideally we use a single wss endpoint.
	fClient, closer, err := lotusClient.NewFullNodeRPCV1(ctx, "wss://wss.calibration.node.glif.io/apigw/lotus/rpc/v1", nil)
	if err != nil {
		log.Fatalf("Failed to connect to Lotus client: %v", err)
	}
	defer closer()

	// get the current chain head to determine chain height
	chainHead, err := fClient.ChainHead(ctx)
	if err != nil {
		log.Fatalf("Failed to get chain head: %v", err)
	}
	chainHeight := big.NewInt(int64(chainHead.Height()))

	// Specify PDP Verifier contract address, this is for calibration net, hardcoded, sorry.
	// TODO(forrest): this should come from a flag.
	contractAddr := common.HexToAddress("0x58B1b601eE88044f5a7F56b3ABEC45FAa7E7681B")

	// build a list of proof-set reports
	reports, err := BuildProofSetReports(ctx, contractAddr, eClient, chainHeight)
	if err != nil {
		log.Fatalf("Failed to build proof report: %v", err)
	}

	// make it POP! (pretty print it)
	PrintReports(int64(chainHead.Height()), reports)
}

// BuildProofSetReports queries the PDPVerifier contract for all proof sets
// and returns a slice of reports summarizing their state.
func BuildProofSetReports(
	ctx context.Context,
	contractAddr common.Address,
	client *ethclient.Client,
	chainHeight *big.Int,
) ([]ProofSetReport, error) {

	callOpts := &bind.CallOpts{Context: ctx}
	pdpVerifier, err := contract.NewPDPVerifier(contractAddr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDPVerifier instance: %w", err)
	}

	// find how many proof sets exist
	nextProofSetID, err := pdpVerifier.GetNextProofSetId(callOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get next proof set ID: %w", err)
	}

	var reports []ProofSetReport

	// loop over each proof set, up to the limit.
	for i := uint64(0); i < nextProofSetID; i++ {
		id := big.NewInt(int64(i))

		// TODO(forrest): we can probably do this in parallel, its a bit slow.
		report, err := buildSingleProofSetReport(ctx, pdpVerifier, client, chainHeight, id)
		if err != nil {
			return nil, fmt.Errorf("error building report for proof set %d: %v", i, err)
		}

		reports = append(reports, report)
	}

	return reports, nil
}

// buildSingleProofSetReport fetches data for a specific proof set and constructs one ProofSetReport.
func buildSingleProofSetReport(
	ctx context.Context,
	pdpVerifier *contract.PDPVerifier,
	client *ethclient.Client,
	chainHeight *big.Int,
	proofSetID *big.Int,
) (ProofSetReport, error) {

	callOpts := &bind.CallOpts{Context: ctx}

	owner1, owner2, err := pdpVerifier.GetProofSetOwner(callOpts, proofSetID)
	if err != nil {
		return ProofSetReport{}, fmt.Errorf("failed to get proof set owner: %w", err)
	}

	// listener is the Address of the SimplePDPService smart contract.
	listener, err := pdpVerifier.GetProofSetListener(callOpts, proofSetID)
	if err != nil {
		return ProofSetReport{}, fmt.Errorf("failed to get proof set listener: %w", err)
	}

	isLive, err := pdpVerifier.ProofSetLive(callOpts, proofSetID)
	if err != nil {
		return ProofSetReport{}, fmt.Errorf("failed to check if proof set is live: %w", err)
	}

	nextChallenge, err := pdpVerifier.GetNextChallengeEpoch(callOpts, proofSetID)
	if err != nil {
		return ProofSetReport{}, fmt.Errorf("failed to get next challenge epoch: %w", err)
	}

	lastProven, err := pdpVerifier.GetProofSetLastProvenEpoch(callOpts, proofSetID)
	if err != nil {
		return ProofSetReport{}, fmt.Errorf("failed to get last proven epoch: %w", err)
	}

	provingDeadline, provenThisPeriod, err := queryPDPService(ctx, listener, client, proofSetID)
	if err != nil {
		return ProofSetReport{}, fmt.Errorf("failed to query PDP Service: %w", err)
	}

	// determine final status, one of: INACTIVE, BAD, GOOD, PENDING.
	status := assessProofSetStatus(
		chainHeight,
		isLive,
		provingDeadline,
		provenThisPeriod,
	)

	return ProofSetReport{
		ID:                 new(big.Int).Set(proofSetID),
		Owner1:             owner1,
		Owner2:             owner2,
		Listener:           listener,
		IsLive:             isLive,
		NextChallengeEpoch: nextChallenge,
		LastProvenEpoch:    lastProven,
		ProvingDeadline:    provingDeadline,
		ProvenThisPeriod:   provenThisPeriod,
		Status:             status,
	}, nil

}

// queryPDPService fetches fields from the PDPService contract for one proof set.
// We return the required data for the final report (deadline + boolean proven).
func queryPDPService(
	ctx context.Context,
	contractAddr common.Address,
	client *ethclient.Client,
	setID *big.Int,
) (*big.Int, bool, error) {

	pdpService, err := contract.NewPDPService(contractAddr, client)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create PDPService instance: %w", err)
	}

	callOpts := &bind.CallOpts{Context: ctx}

	// proving Deadline, need something by this epoch
	deadline, err := pdpService.ProvingDeadlines(callOpts, setID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get proving deadline: %w", err)
	}

	// proven This Period; provenThisPeriod map will only be set to true after a valid proof;
	// it gets set to false on the nextProvingPeriod call
	// we compare the chain head height in the assessProofSetStatus method to determine if pending or bad when this is
	// false.
	proven, err := pdpService.ProvenThisPeriod(callOpts, setID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check if proven this period: %w", err)
	}

	return deadline, proven, nil
}

// assessProofSetStatus is an example of how you might classify the proof set.
func assessProofSetStatus(
	chainHeight *big.Int,
	isLive bool,
	provingDeadline *big.Int,
	provenThisPeriod bool,
) string {
	if !isLive {
		return "INACTIVE (Not Live)"
	}
	if provenThisPeriod {
		return "GOOD (Proven This Period)"
	}
	if chainHeight.Cmp(provingDeadline) >= 0 {
		return "BAD (Overdue / Missed Proof)"
	}

	// TODO(forrest) Could also check if lastProvenEpoch < nextChallengeEpoch, etc.
	// For now, we’ll just call it “PENDING” if we’re not yet at the deadline
	return "PENDING (Awaiting Proof)"
}

// PrintReports uses go-pretty to render a table of proof-set data.
func PrintReports(headHeight int64, reports []ProofSetReport) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{
		"Chain Height",
		"ProofSetID",
		"Status",
		"IsLive",
		"LastProven",
		"Deadline",
		"ProvenThisPeriod",
		"NextChallenge",
		"Owner1",
		"Owner2",
		"Listener",
	})

	for _, r := range reports {
		t.AppendRow(table.Row{
			humanize.Comma(headHeight),
			r.ID.String(),
			r.Status,
			r.IsLive,
			humanize.BigComma(r.LastProvenEpoch),
			humanize.BigComma(r.ProvingDeadline),
			r.ProvenThisPeriod,
			humanize.BigComma(r.NextChallengeEpoch),
			r.Owner1.String(),
			r.Owner2.String(),
			r.Listener.String(),
		})
	}

	t.Render()
}
