# Chain Implementation Plan

## Purpose

This document defines the intended separation of concerns for the four chain layers now present in `KNIRVSERVER` and the migration plan needed to move from the current mixed `internal/services/blockchain` package to explicit, role-based chain boundaries.

The target architecture is:

- `KNIRVCHAIN`
  - Agent context traces
  - Capabilities
  - Skills
  - Errors and solutions
  - Agent-only wallet implementation
  - Embedded subprocess managed by `KNIRVSERVER`
- `Transaction Chain`
  - Fast transaction execution
  - Payment verification
  - DVE registration and transaction-style session/account flows
  - Rollup source for oracle settlement
- `Validation Chain`
  - Immutable validation signoff
  - Policy commit records
  - Evidence anchoring
  - Validation/audit ledger for proofs, signatures, and attestations
- `Oracle`
  - NRN settlement
  - Canonical wallet and balance source of truth
  - Human user wallet authority
  - Rollup verification/finalization
  - Economics, rewards, burns, fees, staking
  - Cross-chain and governance control plane

## Current Embedded Chain Locations

The embedded chain runtimes currently live at:

- Transaction chain:
  - [transaction_chain](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/transaction_chain)
- Validation chain:
  - [validation_chain](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/validation_chain)

Key entrypoints:

- Transaction chain JS runtime:
  - [miner.v10.1.3.js](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/transaction_chain/miner.v10.1.3.js)
- Transaction chain package manifest:
  - [package.json](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/transaction_chain/package.json)
- Validation chain Rust runtime:
  - [main.rs](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/validation_chain/src/main.rs)
- Validation chain cargo manifest:
  - [Cargo.toml](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/validation_chain/Cargo.toml)

## Oracle Location

The actual oracle implementation is in:

- [internal/oracle](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle)

Key oracle entrypoints and subsystems:

- Oracle coordinator:
  - [oracle.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/oracle.go)
- Oracle HTTP routes:
  - [routes.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/routes/routes.go)
- Oracle economics engine:
  - [engine.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/economics/engine.go)
- Oracle cross-chain router:
  - [router.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/crosschain/router.go)
- Oracle transaction types:
  - [transaction.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/types/transaction.go)

## Repository Layout Recommendation

The new placement under `internal/embedded` is the right direction and should be kept.

Recommended structure:

- `backend/internal/embedded/transaction_chain/`
  - foreign-language transaction chain runtime only
- `backend/internal/embedded/validation_chain/`
  - foreign-language validation chain runtime only
- `backend/internal/services/transactionchain/`
  - Go manager, client, types, adapters
- `backend/internal/services/validationchain/`
  - Go manager, client, types, adapters
- `backend/internal/services/knirvchain/`
  - existing embedded manager for `KNIRVCHAIN`
- `backend/internal/services/rollup/`
  - transaction-chain-to-oracle rollup service

This keeps:

- embedded runtime source separate from Go integration logic
- chain role boundaries explicit
- future transport/client refactors isolated from foreign-language runtime changes

## Cleanup Recommendation For Embedded Repos

The embedded repos currently include nested repos and build/runtime artifacts. Long term, these should be cleaned up.

Recommended cleanup:

- remove nested `.git/` directories from:
  - [transaction_chain](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/transaction_chain)
  - [validation_chain](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/validation_chain)
- do not keep `node_modules/` in-tree
- do not keep Rust `target/` in-tree
- do not keep local runtime databases in-tree:
  - `blockchain.db`
  - `sledchain.db/`
- keep only source, lockfiles, and minimal docs/scripts

## Current Go Boundaries

### Keep As-Is

- Embedded `KNIRVCHAIN` manager:
  - [manager.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/knirvchain/manager.go)

This package should remain the subprocess lifecycle wrapper for `KNIRVCHAIN` only.

Additional clarification:

- keep the existing `KNIRVCHAIN` wallet implementation in place for agent wallets only
- do not use `KNIRVCHAIN` as the long-term wallet authority for human users

### Current Mixed Blockchain Package

The current package is overloaded and must be split:

- [nrn_client.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/blockchain/nrn_client.go)
- [knirvchain_client.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/blockchain/knirvchain_client.go)
- [chain_client.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/blockchain/chain_client.go)

These currently mix:

- transaction/account semantics
- DVE registration/session logic
- legacy HTTP/gRPC compatibility
- anchoring and commit adapters via consumers in `main.go`

### Validation Is Not Yet A Chain

The validation subsystem should remain a task execution/proof engine:

- [validation_core.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/validation/validation_core.go)
- [proof_generator.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/validation/proof_generator.go)

It currently:

- creates/stores validation tasks
- executes validation
- emits proofs/signatures/attestation-related data structures

It does not yet implement:

- append-only block sequencing
- chain finality
- immutable signed validation ledger records

## Target Package Split

### `internal/services/transactionchain`

This package should own:

- transaction client(s)
- transaction-chain subprocess manager
- transaction/account types
- compatibility adapters for existing callers

This package should not be treated as the canonical owner of wallet balances.
During migration it may temporarily expose balance reads through compatibility methods,
but those balance queries should resolve against the oracle rather than the transaction
execution layer.

Suggested files:

- `client.go`
- `compat_client.go`
- `types.go`
- `manager.go`
- `config.go`

### `internal/services/validationchain`

This package should own:

- validation-chain client(s)
- validation-chain subprocess manager
- evidence/policy/validation commit request types
- adapters for anchoring and ICME

Suggested files:

- `client.go`
- `types.go`
- `manager.go`
- `config.go`

### `internal/services/rollup`

This package should own:

- transaction-chain block polling
- rollup batch construction
- oracle settlement submission
- batch status tracking

Suggested files:

- `transaction_rollup_service.go`
- `types.go`
- `oracle_client.go`

## File-By-File Migration Checklist

### Phase 1: Create New Packages

Create:

- `packages/KNIRVSERVER/backend/internal/services/transactionchain/`
- `packages/KNIRVSERVER/backend/internal/services/validationchain/`
- `packages/KNIRVSERVER/backend/internal/services/rollup/`

No behavior changes yet. Only introduce the new boundaries.

### Phase 2: Transaction Chain Client Migration

Move or split:

- [nrn_client.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/blockchain/nrn_client.go)
  - new home: `internal/services/transactionchain/client.go`
- [knirvchain_client.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/blockchain/knirvchain_client.go)
  - new home: `internal/services/transactionchain/compat_client.go`

Move shared types:

- `Transaction`
- `Block`
- `TransactionResponse`

into:

- `internal/services/transactionchain/types.go`

Keep a short-term shim in:

- [blockchain](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/blockchain)

so imports do not have to be flipped in one change.

### Phase 3: Transaction Chain Consumer Rewiring

Update imports/interfaces in:

- [dve_creation_service.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/dvecreation/dve_creation_service.go)
- [dve_rental_service.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/dverental/dve_rental_service.go)
- [nrn_payment_handlers.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/web/nrn_payment_handlers.go)
- [blockchain_service.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/payment/blockchain_service.go)

Update tests:

- [dve_creation_service_test.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/dvecreation/dve_creation_service_test.go)
- [dve_rental_service_test.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/dverental/dve_rental_service_test.go)
- [nrn_client_test.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/blockchain/nrn_client_test.go)
- [knirvchain_client_test.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/blockchain/knirvchain_client_test.go)

### Phase 4: Validation Chain Client Introduction

Create:

- `internal/services/validationchain/client.go`
- `internal/services/validationchain/types.go`

Define a validation-specific interface:

- `CommitValidationResult`
- `CommitPolicy`
- `AnchorEvidencePack`
- `GetRecord`
- `GetBlockHeight`
- `Health`

Do not mirror the transaction account/payment API unless needed for operational convenience.

### Phase 5: Validation Chain Consumer Rewiring

Update:

- [anchoring_service.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/evidence/anchoring_service.go)
  - replace generic `ChainClient` with validation-chain client
- [service.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/icme/service.go)
  - replace `SetBlockchainClient` with `SetValidationChainClient`
- [workflow_service.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/workflow/workflow_service.go)
  - replace fake chain commit outputs with validation-chain commit operations

### Phase 6: Main Server Wiring Update

Update:

- [main.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/cmd/backend_server/main.go)

Introduce distinct references:

- `transactionChainManager`
- `validationChainManager`
- `transactionChainClient`
- `validationChainClient`
- keep `chainManager` for embedded `KNIRVCHAIN`

Rewire consumers:

- DVE creation -> transaction chain
- DVE rental -> transaction chain
- NRN payment handlers -> transaction chain
- anchoring -> validation chain
- ICME -> validation chain
- workflow validation commits -> validation chain

### Phase 7: Rollup Service Introduction

Create:

- `internal/services/rollup/transaction_rollup_service.go`

Responsibilities:

- read new transaction-chain blocks
- build rollup batches
- compute batch root / merkle root
- submit settlement record to oracle
- track batch lifecycle

Suggested statuses:

- `pending`
- `built`
- `submitted`
- `settled`
- `disputed`
- `failed`

### Phase 8: Oracle Settlement Integration

Extend:

- [routes.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/routes/routes.go)
- [oracle.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/oracle.go)

Add rollup settlement endpoints:

- `POST /oracle/v3/rollups/submit`
- `GET /oracle/v3/rollups/{id}`
- `POST /oracle/v3/rollups/{id}/finalize`
- `POST /oracle/v3/rollups/{id}/dispute`

Add oracle-side responsibilities:

- validate submitted rollup metadata
- apply economics and fee accounting
- mint/burn/reward as part of NRN settlement
- persist settlement records

## Required Transaction Chain Compatibility Endpoints

The current Go transaction clients already expect these HTTP routes and should continue to work during migration:

- `GET /health`
- `GET /chain`
- `GET /chain/height`
- `GET /chain/tx/{hash}`
- `GET /txn_pool`
- `POST /transaction`

These expectations are encoded in:

- [nrn_client.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/blockchain/nrn_client.go#L123)
- [knirvchain_client.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/blockchain/knirvchain_client.go#L192)

Important clarification:

- `GET /account/{address}/balance` should no longer be treated as a true transaction-chain responsibility
- canonical NRN balances should belong to the oracle layer in:
  - [internal/oracle](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle)
- if the existing Go compatibility layer still needs `GetAccountBalance`, that method should be backed by the oracle, not by the transaction execution chain

### Transaction Chain Implementation Requirement

The JS transaction chain in:

- [miner.v10.1.3.js](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/transaction_chain/miner.v10.1.3.js)

currently exposes:

- `POST /transactions`
- `GET /blocks`
- `GET /transactions/:hash`

It must be extended to support the compatibility contract above.

Suggested aliases and additions:

- `POST /transaction`
  - alias to existing `/transactions`
- `GET /chain`
  - wrap `GET /blocks` into chain response format
- `GET /chain/tx/{hash}`
  - alias to existing `/transactions/:hash`
- `GET /chain/height`
  - return latest block height
- `GET /txn_pool`
  - return current pending tx pool
- `GET /health`
  - return liveness/ready status

Do not make the transaction chain the source of truth for balances unless there is a temporary migration shim.

## Validation Chain API Recommendation

The Rust validation chain in:

- [main.rs](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/validation_chain/src/main.rs)

currently exposes:

- `/send_txn`
- `/wallets/new`
- `/nrn/mint`
- `/nrn/transfer`
- `/nrn/balance`
- `/nrn/info`
- `/blocks`

This should be reoriented around validation records, not transaction-account semantics.

Recommended validation-chain endpoints:

- `GET /health`
- `GET /chain/height`
- `POST /validation/commit`
- `POST /policy/commit`
- `POST /evidence/anchor`
- `GET /records/{id}`
- `GET /records?type=validation|policy|evidence`
- `GET /blocks`

If desired, the existing token endpoints can remain during transition, but they should not define the long-term role of the validation chain.

## Initialization Wiring Plan

All four chain layers should be initialized explicitly in `backend_server`.

### 1. KNIRVCHAIN Initialization

Keep the existing embedded subprocess manager:

- [manager.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/knirvchain/manager.go)

`main.go` should continue to:

- construct `knirvchain.ManagerConfig`
- create `knirvchain.NewManager(...)`
- start it during server startup

Role:

- context/capabilities/skills/errors-solutions chain only

### 2. Transaction Chain Initialization

Add:

- `internal/services/transactionchain/manager.go`

Manager responsibilities:

- locate embedded JS runtime at:
  - [transaction_chain](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/transaction_chain)
- execute:
  - `node miner.v10.1.3.js`
- pass environment:
  - `PORT`
  - `BLOCK_DIFFICULTY`
  - `BLOCK_TIME`
  - `DATA_PATH` if needed
- wait for `/health`

`main.go` wiring:

- create `transactionChainConfig`
- create `transactionChainManager`
- create `transactionChainClient`
- start manager before wiring services that depend on transaction flows

Consumers:

- DVE creation
- DVE rental
- payment handlers

### 3. Validation Chain Initialization

Add:

- `internal/services/validationchain/manager.go`

Manager responsibilities:

- locate embedded Rust runtime at:
  - [validation_chain](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/validation_chain)
- prefer launched binary if prebuilt
- otherwise point to configured binary path
- pass environment:
  - `KNIRVCHAIN_RPC_ENDPOINT` or rename to validation-specific env
  - `BLOCK_DIFFICULTY`
  - `BLOCK_TIME`
  - `CHAIN_ID`
  - `DATA_PATH`
- wait for `/health`

`main.go` wiring:

- create `validationChainConfig`
- create `validationChainManager`
- create `validationChainClient`
- start manager before initializing anchoring and ICME integrations

Consumers:

- anchoring service
- ICME
- workflow validation commit operations

### 4. Oracle Initialization

The actual oracle already exists in-process under `KNIRVSERVER/backend/internal/oracle`:

- [oracle.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/oracle.go)
- [routes.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/routes/routes.go)

Keep oracle in-process and root-key gated.

`main.go` wiring should:

- initialize oracle only when root key is present
- initialize rollup service only when both transaction chain and oracle are enabled
- register rollup settlement endpoints with oracle routes

Consumers:

- rollup service
- economics engine
- governance and settlement flows
- canonical balance and wallet queries

Implementation note:

- do not introduce a second oracle implementation in the new chain split
- treat `backend/internal/oracle` as the canonical oracle and settlement layer
- all transaction-chain rollup settlement should terminate here
- all oracle route additions for rollups should be implemented under:
  - [routes.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/routes/routes.go)
- all oracle-side settlement orchestration should be implemented under:
  - [oracle.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/oracle.go)
  - [engine.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/economics/engine.go)

## Oracle Role In The Four-Chain Model

Recommended interpretation:

- `Transaction Chain`
  - execution layer
  - cheap/simple tx processing
  - rollup source
  - not the canonical balance ledger
- `Oracle`
  - settlement layer for NRN
  - canonical wallet/address balance layer
  - verifies and finalizes rollups
  - owns economics and supply-side consequences
- `Validation Chain`
  - trust and audit ledger
  - immutable signoff of validation, evidence, and policy states
- `KNIRVCHAIN`
  - knowledge/context/capability chain
  - agent wallet domain

This creates a workable four-chain architecture without collapsing unrelated trust models together.

## Rollup Design Recommendation

Add a service:

- `internal/services/rollup/transaction_rollup_service.go`

Suggested inputs:

- transaction chain blocks
- transaction chain tx inclusion data
- settlement config

Suggested outputs to oracle:

- rollup ID
- source chain ID
- block range
- transaction count
- batch root / merkle root
- total fee amount
- timestamp
- optional fraud/dispute metadata

Suggested oracle settlement flow:

1. read confirmed tx-chain blocks
2. build rollup batch
3. submit rollup to the oracle in `backend/internal/oracle`
4. oracle verifies and records settlement
5. oracle economics engine applies fees/rewards/burns
6. rollup marked finalized

After settlement, wallet balance reads should resolve from oracle state, not transaction-chain local execution state.

## Balance And Wallet Responsibility

Canonical wallet and balance ownership should move to the oracle.

That means:

- `KNIRVCHAIN` should remain the wallet authority for agent wallets only
- `Transaction Chain` should not become the long-term wallet/balance authority
- `Oracle` should become the canonical source of truth for:
  - human user wallet balances
  - human user wallet identity/reference
  - token supply state
  - mint/burn/reward consequences
  - post-rollup account settlement

Wallet split:

- Agent wallets
  - stay on the current `KNIRVCHAIN` wallet implementation
  - are used for agent-side operations, traces, capabilities, and autonomous system activity
- Human user wallets
  - are generated and managed by the oracle chain
  - should be referenced to and from oracle-owned state
  - should be treated as the canonical user-facing balance and settlement domain

Migration guidance:

1. keep the existing `KNIRVCHAIN` wallet server behavior for agent wallets
2. add oracle-backed wallet generation and balance endpoints for human wallets
3. add an oracle-backed compatibility endpoint for human balance reads
4. change Go client compatibility methods such as `GetAccountBalance()` to resolve against oracle for human-user payment flows
5. preserve legacy method signatures during transition
6. retire human balance assumptions from transaction-chain clients once consumers are migrated

Recommended domain split in code:

- `KNIRVCHAIN`
  - agent wallet issuance
  - agent wallet balance/state
  - agent-to-agent operational settlement as needed
- `Oracle`
  - human wallet issuance
  - human wallet lookup
  - human NRN balances
  - settlement after transaction-chain rollups

Recommended compatibility endpoint additions in oracle:

- `GET /account/{address}/balance`
  - compatibility alias backed by oracle token state
- `POST /oracle/v3/wallets/create`
  - create human user wallet
- `GET /oracle/v3/wallets/{address}`
  - retrieve human wallet metadata/reference
- keep the canonical oracle route:
  - [routes.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/routes/routes.go#L35)
  - [routes.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle/routes/routes.go#L92)

## Main Server Wiring Checklist

In:

- [main.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/cmd/backend_server/main.go)

add and wire:

- `transactionChainManager`
- `validationChainManager`
- `transactionChainClient`
- `validationChainClient`
- `rollupService`

Order recommendation:

1. initialize database and base services
2. initialize embedded KNIRVGRAPH
3. initialize embedded KNIRVCHAIN
4. initialize embedded transaction chain
5. initialize embedded validation chain
6. initialize transaction-chain client
7. initialize validation-chain client
8. initialize validation core
9. initialize anchoring service with validation-chain client
10. initialize ICME with validation-chain client
11. initialize DVE creation/rental with transaction-chain client
12. initialize oracle if root key present
13. initialize rollup service if oracle + transaction chain are enabled
14. start services in dependency order

## Suggested Transitional Strategy

Do not rename the existing `internal/services/blockchain` package immediately.

Use a three-step transition:

1. create `transactionchain` and `validationchain`
2. move consumers gradually while leaving compatibility shims in `blockchain`
3. remove or freeze `blockchain` only after all imports and tests have migrated

This reduces the blast radius and keeps the old compatibility surface alive while the new embedded chains are made API-complete.

## Immediate Next Implementation Tasks

1. Add compatibility endpoints to the JS transaction chain.
2. Add validation-native endpoints to the Rust validation chain.
3. Add `transactionchain/manager.go`.
4. Add `validationchain/manager.go`.
5. Add `transactionchain/client.go`.
6. Add `validationchain/client.go`.
7. Rewire `main.go`.
8. Add oracle rollup settlement endpoints.
9. Add `rollup/transaction_rollup_service.go`.
10. Migrate consumers and tests incrementally.

## Current Implementation Status

The migration has begun and the following work is already in place.

Completed:

- added the architecture and migration planning document:
  - [chain_implementation.md](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/chain_implementation.md)
- added config surfaces for:
  - `transaction_chain`
  - `validation_chain`
  - in [config.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/config/config.go)
- added Go integration scaffolding for the new chain roles:
  - [transactionchain/client.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/transactionchain/client.go)
  - [transactionchain/manager.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/transactionchain/manager.go)
  - [validationchain/client.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/validationchain/client.go)
  - [validationchain/manager.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/services/validationchain/manager.go)
- updated server wiring in:
  - [main.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/cmd/backend_server/main.go)
- transaction-chain startup/client path is now separated from `KNIRVCHAIN`
- validation-chain startup/client path is now separated from `KNIRVCHAIN`
- DVE creation and DVE rental are wired toward the transaction-chain client path
- anchoring prefers the validation-chain client path
- ICME prefers the validation-chain client path
- startup and shutdown now include the two new embedded chain managers

Partially completed:

- the new `transactionchain` client still wraps the legacy blockchain client for compatibility during migration
- the new `validationchain` client currently uses transitional request translation rather than final validation-native request types
- `main.go` still contains legacy adapters for compatibility fallback

Not yet completed:

- transaction-chain compatibility endpoint implementation in the JS runtime
- validation-chain health and validation-native route implementation in the Rust runtime
- rollup service implementation
- oracle settlement route additions for rollups
- full consumer migration away from the old `internal/services/blockchain` compatibility layer
- tests updated to target the new chain boundaries explicitly

Verification status:

- touched Go files were formatted
- targeted Go verification was attempted, but backend package test/compile commands timed out in the current sandbox environment without producing stable compiler output
- because of that, the migration should currently be treated as in-progress rather than fully verified

## Where To Go Next

The next implementation sequence should be:

1. Add the transaction-chain compatibility endpoints in the JS runtime under:
   - [transaction_chain](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/transaction_chain)
2. Add `/health` plus validation-native commit routes in the Rust validation chain under:
   - [validation_chain](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/embedded/validation_chain)
3. Replace the remaining legacy adapters in:
   - [main.go](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/cmd/backend_server/main.go)
   with native `transactionchain` and `validationchain` request types
4. Add the rollup service and oracle settlement endpoints under:
   - [internal/oracle](/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/oracle)

More concretely:

- JS transaction chain:
  - add `/health`
  - add `/transaction`
  - add `/chain`
  - add `/chain/height`
  - add `/chain/tx/{hash}`
  - add `/txn_pool`
- Rust validation chain:
  - add `/health`
  - add `/chain/height`
  - add `/validation/commit`
  - add `/policy/commit`
  - add `/evidence/anchor`
- `main.go`:
  - remove remaining reliance on legacy adapter-only request shapes where possible
- oracle:
  - add rollup settlement intake
  - add rollup query/finalization/dispute endpoints
  - connect the future rollup service to oracle economics/state transitions
