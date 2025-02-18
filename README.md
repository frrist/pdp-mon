# PDP-Mon

PDP-Mon is a small Go utility for querying a PDP (Proof-of-Data-Possession) Verifier contract and its associated PDP Service contract on the Filecoin Calibration testnet, then producing a human-friendly table of proof-set states. This tool lets you quickly see whether each proof set is in good, bad, or pending status based on the current chain head.

## Overview

PDP-Mon interacts with two key components:

1. **Lotus (Filecoin Calibration network)** – to obtain the current chain head height.
2. **Ethereum-compatible RPC** – to fetch state information from the PDPVerifier and PDP Service contracts.

It aggregates important proof-set information—such as owner addresses, live status, deadlines, and proof results—and displays it in a table.

## Features

- Connects to both Ethereum and Lotus endpoints.
- Fetches the chain head once and uses it to assess if each proof set is overdue.
- Assembles structured reports for each proof set.
- Uses go-pretty to render a clean, human-friendly table.
- Formats large numbers with commas using go-humanize.

## Installation

1. Clone or download the repository:
   - `git clone https://github.com/frrist/pdp-mon.git`
   - `cd pdp-mon`
2. Install dependencies:
   - `go mod tidy`
   
3. Configure as needed. By default, network endpoints and contract addresses are hardcoded. You can update these in the source (see TODO markers in the code).

## Usage

To run PDP-Mon:

- Run directly:  
  `go run main.go`
  
- Or build and run the binary:  
  `go build -o pdp-mon main.go`  
  `./pdp-mon`

### Default Configuration

- **Ethereum Client URL**: `https://api.calibration.node.glif.io/rpc/v1`
- **Lotus Client WebSocket**: `wss://wss.calibration.node.glif.io/apigw/lotus/rpc/v1`
- **PDP Verifier Contract Address**: `0x58B1b601eE88044f5a7F56b3ABEC45FAa7E7681B`

These can be updated in the `main()` function as needed.

### Example Output
```
+--------------+------------+------------------------------+--------+------------+-----------+------------------+---------------+--------------------------------------------+--------------------------------------------+--------------------------------------------+
| CHAIN HEIGHT | PROOFSETID | STATUS                       | ISLIVE | LASTPROVEN | DEADLINE  | PROVENTHISPERIOD | NEXTCHALLENGE | OWNER1                                     | OWNER2                                     | LISTENER                                   |
+--------------+------------+------------------------------+--------+------------+-----------+------------------+---------------+--------------------------------------------+--------------------------------------------+--------------------------------------------+
| 2,417,142    | 0          | BAD (Overdue / Missed Proof) | true   | 2,349,648  | 2,349,866 | false            | 2,349,851     | 0x908FDD9eF6dB0B4f8c3a60640f2D6B3c2a7A4f3B | 0x0000000000000000000000000000000000000000 | 0xb1B1df5C1Eb5338E32A7Ee6b5E47980FB892bb9f |
| 2,417,142    | 1          | BAD (Overdue / Missed Proof) | true   | 0          | 0         | false            | 0             | 0x908FDD9eF6dB0B4f8c3a60640f2D6B3c2a7A4f3B | 0x0000000000000000000000000000000000000000 | 0xb1B1df5C1Eb5338E32A7Ee6b5E47980FB892bb9f |
| 2,417,142    | 2          | BAD (Overdue / Missed Proof) | true   | 2,351,694  | 2,351,792 | false            | 2,351,777     | 0x908FDD9eF6dB0B4f8c3a60640f2D6B3c2a7A4f3B | 0x0000000000000000000000000000000000000000 | 0xb1B1df5C1Eb5338E32A7Ee6b5E47980FB892bb9f |
| 2,417,142    | 3          | BAD (Overdue / Missed Proof) | true   | 2,351,789  | 2,351,827 | false            | 2,351,812     | 0x908FDD9eF6dB0B4f8c3a60640f2D6B3c2a7A4f3B | 0x0000000000000000000000000000000000000000 | 0xb1B1df5C1Eb5338E32A7Ee6b5E47980FB892bb9f |
| 2,417,142    | 4          | BAD (Overdue / Missed Proof) | true   | 2,354,575  | 2,354,618 | false            | 2,354,603     | 0x908FDD9eF6dB0B4f8c3a60640f2D6B3c2a7A4f3B | 0x0000000000000000000000000000000000000000 | 0xb1B1df5C1Eb5338E32A7Ee6b5E47980FB892bb9f |
| 2,417,142    | 5          | PENDING (Awaiting Proof)     | true   | 2,417,112  | 2,417,155 | false            | 2,417,140     | 0x908FDD9eF6dB0B4f8c3a60640f2D6B3c2a7A4f3B | 0x0000000000000000000000000000000000000000 | 0xb1B1df5C1Eb5338E32A7Ee6b5E47980FB892bb9f |
| 2,417,142    | 6          | BAD (Overdue / Missed Proof) | true   | 0          | 0         | false            | 0             | 0x908FDD9eF6dB0B4f8c3a60640f2D6B3c2a7A4f3B | 0x0000000000000000000000000000000000000000 | 0xb1B1df5C1Eb5338E32A7Ee6b5E47980FB892bb9f |
+--------------+------------+------------------------------+--------+------------+-----------+------------------+---------------+--------------------------------------------+--------------------------------------------+--------------------------------------------+
```

## License

This project is licensed under the MIT License. See the LICENSE file for details.

Happy proving! If you have any questions or suggestions, please open an issue or submit a pull request.
