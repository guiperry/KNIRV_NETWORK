# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

KNIRV Network is a Decentralized Trusted Execution Network (D-TEN) - a multi-component blockchain ecosystem for collective AI intelligence. The network transforms AI failures into collective knowledge through interconnected sovereign layers communicating via IBC.

## Tech Stack

- **Go 1.21+**: KNIRVCHAIN, KNIRVGRAPH, KNIRVROUTER, KNIRVNEXUS
- **Rust 1.70+**: KNIRVORACLE, KNIRVCONTROLLER (WASM)
- **Node.js 18+**: KNIRVGATEWAY, KNIRVCONTROLLER PWA, KNIRVWALLET
- **Ansible**: Infrastructure deployment

## Core Components

| Component | Purpose | Tech |
|-----------|---------|------|
| KNIRVCHAIN | Skill/LLM registry, MCP capabilities, node transformation mining | Go |
| KNIRVORACLE | Cross-chain transfers, network governance, token economics | Rust |
| KNIRVGRAPH | Knowledge graphchain for AI errors/solutions | Go |
| KNIRVNEXUS | Distributed Validation Environment (DVE) | Go |
| KNIRVROUTER | P2P network backbone, Proof-of-Connectivity | Go |
| KNIRVCONTROLLER | User's autonomous AI gateway | Rust/WASM |
| KNIRVGATEWAY | API gateway, documentation website | Node.js |
| KNIRVWALLET | Non-custodial wallet with XION Meta Accounts | Node.js |

## Node Transformation Flows (KNIRVCHAIN)

Three core mining/minting processes transform network activity into valuable artifacts:

1. **ErrorNode → SkillNode Mining** → LoRA Adapter Pointer
2. **ContextNode → CapabilityNode Minting** → MCP Server Pointer
3. **IdeaNode → PropertyNode Making** → Inference NFT Pointer

See `KNIRVCHAIN_Implementation.md` for full implementation plan.

## Governance & Cross-Chain (KNIRVORACLE)

KNIRVORACLE is the sole governance authority and cross-chain hub:

- **Cross-Chain Transfers**: IBC-based transfers between all KNIRV chains and external chains (XION, Cosmos)
- **Network Governance**: Proposals, voting, parameter changes, emergency actions
- **Token Economics**: NRN fees, staking, rewards, burn tracking

> **Note**: LLM/Skill registries have been deprecated in KNIRVORACLE and moved to KNIRVCHAIN.

See `KNIRVORACLE_Implementation.md` for full implementation plan.

## Common Commands

```bash
# Run all tests
make tests

# Run tests for specific component
make test-controller
make test-nexus
make test-sdk

# Quick testnet tests
make testnet-tests

# Generate documentation
make docs

# Deploy testnet services
make deploy-testnet-services

# Testnet instance management
make testnet-start
make testnet-stop
make testnet-status
make testnet-ssh

# Full deployment
make deploy-full ENVIRONMENT=production CLOUD_PROVIDER=aws
```

## Running Individual Go Tests

```bash
cd KNIRVNEXUS && go test -v ./backend/tests/...
cd integration-tests && go test -v -run "TestKNIRVNEXUS.*"
```

## Project Structure

- Each `KNIRV*` directory is a separate component with its own go.mod or package.json
- `deployment/ansible/` - Infrastructure deployment scripts
- `integration-tests/` - Cross-component integration tests
- `shared-proto/` - Shared protobuf definitions
- `docs/` - Source documentation (generates to KNIRVGATEWAY)
- `scripts/` - Deployment and utility scripts

## Environment Setup

1. Copy `deployment/ansible/.env.example` to `deployment/ansible/.env`
2. Run `make install-deps` to install Ansible dependencies
3. Run `make check-prereqs` to verify setup
