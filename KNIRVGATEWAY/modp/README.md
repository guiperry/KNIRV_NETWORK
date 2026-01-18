# ModP Framework for KNIRVORACLE

This directory contains the ModP (Model-based Programming) implementation for formal verification of the KNIRVORACLE blockchain system using the P programming language.

## Overview

ModP enables compositional testing of the oracle system by modeling components as communicating state machines and verifying critical invariants through specification machines (monitors).

Based on the [ModP research paper](https://ankushdesai.github.io/assets/papers/modp.pdf) and Section 5 of `p_implementation.md`.

## Directory Structure

```
modp/
├── types.p                 # Type definitions for all components
├── events.p                # Event definitions for inter-machine communication
├── machines/               # P state machines modeling oracle components
│   ├── consensus_machine.p    # Consensus engine model
│   ├── token_machine.p        # NRN token model
│   ├── governance_machine.p   # Governance system model
│   ├── economics_machine.p    # Economics engine model
│   ├── ibc_machine.p          # IBC handler model
│   └── p2p_interface.p        # P2P abstract interface
├── monitors/               # Specification machines (invariant monitors)
│   ├── treasury_invariant.p      # Treasury fund protection
│   ├── token_supply_invariant.p  # Monetary policy enforcement
│   └── governance_invariant.p    # Governance process integrity
├── tests/                  # Compositional test definitions
│   ├── app_layer.p            # User behavior simulation
│   └── compositional_tests.p  # Test compositions
├── scripts/                # Setup and test scripts
│   ├── setup.sh              # P framework installation
│   └── run-tests.sh          # Test execution script
└── KnirvOracle.pproj       # P project configuration
```

## Prerequisites

- .NET SDK 6.0 or later
- Java JDK 11 or later
- Git

## Installation

Run the setup script to install the P language framework:

```bash
./scripts/setup.sh
```

This will:
1. Clone the P language repository
2. Build the P compiler
3. Configure project files

## Running Tests

### Run All Tests

```bash
./scripts/run-tests.sh all
```

Or using Make:

```bash
make test-modp
```

### Run Specific Test

```bash
./scripts/run-tests.sh test TokenTransferConsistency
```

### List Available Tests

```bash
./scripts/run-tests.sh list
```

## Available Tests

| Test | Description |
|------|-------------|
| `OracleWithInvariants` | Full oracle with all invariant monitors |
| `TokenTransferConsistency` | Token transfer balance consistency |
| `GovernanceLifecycle` | Proposal lifecycle verification |
| `EconomicsFeeProcessing` | Fee processing verification |
| `SecondMeRegistrationWorkflow` | "Second Me" registration flow |
| `MaliciousBehaviorRejection` | Invalid operation rejection |

## Invariant Monitors

### TreasuryInvariant

Ensures:
- All network fees are correctly deposited into treasury
- Treasury withdrawals only occur through governance approval
- Treasury balance never goes negative

### TokenSupplyInvariant

Ensures:
- Total supply never exceeds max supply
- Minting only occurs through authorized mechanisms
- Supply is always non-negative

### GovernanceInvariant

Ensures:
- Proposals can't pass/fail before voting period ends
- Vote tallies correctly determine outcome
- No double voting allowed
- Votes only cast during voting period

## State Machine Models

### ConsensusMachine

Models the consensus engine including:
- Block proposal and commitment
- Validator set management
- Round state transitions

### TokenMachine

Models the NRN token including:
- Transfer operations
- Minting (with authorization checks)
- Burning
- Balance management

### GovernanceMachine

Models on-chain governance including:
- Proposal creation
- Vote casting
- Proposal status updates
- "Second Me" registration workflow

### EconomicsMachine

Models token economics including:
- Fee processing
- Reward distribution
- Staking operations
- Treasury management

### IBCMachine

Models Inter-Blockchain Communication including:
- Channel management
- Connection management
- Packet sending/receiving

### P2PInterface

Abstract interface for P2P networking:
- Peer connections
- Message broadcasting
- Network status

## Compositional Testing

The `AppLayerTestMachine` simulates user behavior:
- Normal operations (transfers, proposals, votes)
- Malicious behavior (insufficient funds, double voting)
- Stress testing (high volume operations)

Tests are composed using the pattern:
```
knirv-oracle || AppLayer || TreasuryInvariant || TokenSupplyInvariant || GovernanceInvariant
```

## Integration with Go Code

The P state machines model the behavior of the Go implementation in:
- `internal/oracle/consensus/`
- `internal/oracle/token/`
- `internal/oracle/governance/`
- `internal/oracle/economics/`
- `internal/oracle/ibc/`
- `internal/oracle/p2p/`

## References

- [P Language Documentation](https://p-org.github.io/P/)
- [ModP Research Paper](https://ankushdesai.github.io/assets/papers/modp.pdf)
- [p_implementation.md](../../p_implementation.md)
