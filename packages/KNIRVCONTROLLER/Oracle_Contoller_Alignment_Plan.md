t# Align KNIRVCONTROLLER's sovereign wallet with KNIRVORACLE's NRN ledger

## Context

While wiring the KNIRVCONTROLLER network-selection dialog, the question came up of whether
the controller should fetch its wallet address from KNIRVORACLE (served by KNIRVSERVER).
Investigation showed:

- KNIRVCONTROLLER's vault (`useVault.ts` → `knirvwallet-module`) generates a secp256k1 key
  from a BIP-39 mnemonic **entirely client-side** and derives a Cosmos/gno-style bech32
  address (`knirv1...` via `RIPEMD160(SHA256(pubkey))`). Private keys never leave the device —
  this is documented as a hard guarantee in `onboarding.md` and the Onboarding UI copy.
- KNIRVORACLE (`packages/KNIRVORACLE`) has wallet-issuing endpoints (`/generate_wallet`,
  `/oracle/v3/install/wallet`), but they generate an Ethereum-style secp256k1 keypair
  **server-side** and return the raw private key in the JSON response. That model is for
  server-custodied DVE/service wallets, not a personal sovereign identity — reusing it as
  the controller's wallet would leak the user's private key to the network and the server.
- The two systems use the **same curve** (secp256k1) but different address encodings
  (bech32+RIPEMD160/SHA256 vs. hex+Keccak256/EIP-55) and, more importantly, KNIRVORACLE's
  transfer/burn/vote endpoints require the caller to submit a raw private key so the server
  can sign on their behalf — there is no way today to transact against the NRN ledger without
  handing over custody.

Decision (explicit, from the project owner): wallet creation and custody stay 100%
client-side (sovereignty). The systems get aligned by (a) deriving an oracle-compatible
address from the *same* sovereign key, no key material transmitted, and (b) teaching
KNIRVORACLE to accept client-signed requests instead of raw private keys, so the sovereign
wallet can actually transact.

## Design

### Piece 1 — Oracle-compatible address, derived client-side (low risk, no Go changes)

Add a second address view to `knirvwallet-module`, computed from the exact same public key
already used for the `knirv1...` bech32 address — no new key material, no server round trip.

- `packages/KNIRVCONTROLLER/core/packages/knirvwallet-module/src/utils/address.ts`: add
  `publicKeyToOracleAddress(publicKey: Uint8Array): string`
  - `Secp256k1.uncompressPubkey(publicKey)` (from `@cosmjs/crypto`, already a dependency) to
    normalize to the 65-byte uncompressed form (0x04 prefix + 64 bytes X‖Y).
  - `keccak_256(uncompressed.slice(1))` — drop the 0x04 prefix, hash the remaining 64 bytes.
    Requires adding `@noble/hashes` as a direct dependency of `knirvwallet-module`'s
    `package.json` (it's already present transitively via `ethereum-cryptography`/cosmjs
    deps, but should be declared explicitly rather than relied on implicitly).
  - Take the last 20 bytes, hex-encode with the module's existing `toHex` helper
    (`encoding/hex.ts`), prefix `0x`. This exactly matches Go's
    `packages/KNIRVORACLE/internal/oracle/crypto/address.go:PublicKeyToAddress` and the wire
    format `types.Address.String()` produces (lowercase hex, no EIP-55 checksum required —
    `AddressFromString` in `internal/oracle/types/address.go` is case-insensitive).
- Add `getOracleAddress(): Promise<string>` to the `Account` interface
  (`wallet/account/account.ts`) and implement it identically to the existing `getAddress(prefix)`
  method (which all four account classes — `single-account.ts`, `seed-account.ts`,
  `airgap-account.ts`, `ledger-account.ts` — already implement by calling the shared
  `publicKeyToAddress` utility). Same pattern, new utility function.
- `useVault.ts`: no structural change needed — `currentAccount.getOracleAddress()` becomes
  available wherever `currentAccount.getAddress('knirv')` is used today.

**This piece is safe to ship independently.** It's additive, computed from key material that
already exists locally, and requires zero KNIRVORACLE/Go changes (the oracle already accepts
arbitrary valid addresses for balance queries and `FundAddress`).

### Piece 2 — Registration endpoint (KNIRVORACLE, additive)

Add a way to register a client-generated address with the oracle *without* generating a
keypair server-side, replacing the "mint you a wallet" flow with "record the wallet you
already own."

- New handler in `packages/KNIRVORACLE/internal/oracle/routes/wallet.go`:
  `POST /oracle/v3/wallet/register` — body `{"address": "0x...", "owner_id": "..."}`.
  - Parse via existing `types.AddressFromString`.
  - Reuses `r.oracle.WalletServerEnabled()` / `r.oracle.GetWalletInitialFunding()` /
    `r.oracle.FundAddress(...)` exactly as `createWalletResponse` does today, but skips
    `crypto.GenerateKeyPair()` entirely — no private key is ever created or returned.
  - Response mirrors `walletCreateResponse` minus the `private_key`/`public_key` fields.
- Register the route in `RegisterRoutes` (`routes.go`) alongside the existing
  `/oracle/v3/install/wallet` line.
- **Keep the existing `/generate_wallet` and `/oracle/v3/install/wallet` endpoints
  unchanged** — they're covered by `wallet_test.go` and may still be legitimately used for
  server-custodied DVE/service wallets. This is additive, not a replacement.

### Piece 3 — Signature-verified transfers (KNIRVORACLE, the actual "transact" unlock)

This is the piece that lets the sovereign wallet do anything beyond holding an address. Today
`NRN.Transfer(fromPrivateKey, ...)` in `internal/oracle/token/transfer.go` derives the sender's
identity *from the private key itself* — there's no path that verifies a signature against a
claimed address instead. The primitives already exist and are unused:
`crypto.VerifySignature` / `crypto.RecoverAddress` in `internal/oracle/crypto/ecdsa.go`, and
`NRN.TransferBetween(fromAddr, toAddr, amount)` already does the balance mutation without
touching a private key. There's also a dead `Nonce uint64` field on `types.Transaction`
(`internal/oracle/types/transaction.go`) that was clearly intended for this but never wired up.

Design:

1. **Nonce tracking.** Add `nonces map[types.Address]uint64` to the `NRN` struct
   (`internal/oracle/token/nrn.go`), guarded by the existing `n.mu`. Add
   `GET /oracle/v3/account/nonce/{address}` (manual path-prefix-stripping, matching the
   existing `handleTokenBalance`/`handleLegacyWalletBalance` convention since these routes use
   `http.ServeMux`, not gorilla mux) returning `{"address": "...", "nonce": N}` — the nonce the
   client must use for its *next* signed request.
2. **Signed transfer.** Add `NRN.TransferSigned(fromAddr, toAddr types.Address, amount *big.Int, nonce uint64, signature []byte) (*TransferReceipt, error)`:
   - Reject if `nonce != n.nonces[fromAddr]` (replay protection).
   - Reconstruct the exact same message format the client signed:
     `fmt.Sprintf("transfer:%s:%s:%s:%d", fromAddr.String(), toAddr.String(), amount.String(), nonce)`
     (extends the existing `Transfer` message format with the nonce).
   - `crypto.RecoverAddress(message, signature)` must equal `fromAddr`; reject otherwise.
   - On success: increment `n.nonces[fromAddr]`, then reuse the existing `TransferBetween`
     balance-mutation logic and build the same `TransferReceipt` shape.
3. **New route:** `POST /oracle/v3/token/transfer/signed` — body
   `{"from": "0x...", "to": "0x...", "amount": "...", "nonce": N, "signature": "0x..."}`.
   Keep `handleTokenTransfer`/`handleSendSignedTransaction` (private-key-based) unchanged for
   backward compatibility.
4. **Burn follows the same pattern** (`NRN.BurnSigned`, `POST /oracle/v3/token/burn/signed`) —
   same message-format-plus-nonce approach as transfer, reusing `BurnFrom` for the mutation.
   Treat this as a direct follow-up once transfer is proven out, not a blocking part of this
   phase. Governance voting (`handleVote`) can adopt the identical pattern later; it's
   peripheral to "the wallet can transact" and shouldn't gate this work.

### Piece 4 — Wallet-side signing

`knirvwallet-module`'s account classes sign via `@cosmjs/crypto`'s
`Secp256k1.createSignature(messageHash, privkey)`, which returns an
`ExtendedSecp256k1Signature` (r‖s‖recovery, 65 bytes via `.toFixedLength()`) — the recovery
byte is exactly what Go's `ethcrypto.SigToPub`/`RecoverAddress` needs to recover a pubkey from
a signature. Add a method (e.g. `signOracleMessage(message: string)`) that:
- Hashes `message` with the same `keccak_256` added in Piece 1 (not the module's existing
  SHA-256 path — the oracle verifies over Keccak256).
- Calls `Secp256k1.createSignature(hash, privkey)` and returns
  `'0x' + toHex(signature.toFixedLength())`.

**Known risk — flag explicitly, don't assume:** cosmjs's recovery-id convention (0/1 range,
low-S normalization) and go-ethereum's `ethcrypto.Sign`/`SigToPub` both wrap standard
secp256k1 ECDSA, but byte-for-byte interop between two different library implementations of
the same primitive is exactly the kind of thing that looks right on paper and silently breaks
on an edge case (recovery id offset, S-normalization, hash truncation). **Before wiring any
route or UI to this, do a standalone round-trip spike:** sign a fixed test message in
TypeScript with the new `signOracleMessage`, hardcode the output signature + address in a Go
test, and confirm `crypto.RecoverAddress` in the oracle recovers the same address. Only proceed
to wiring the transfer endpoint once that spike passes.

## Sequencing

1. **Spike:** TS↔Go signature interop round-trip (see risk note above). Go/no-go gate for
   Piece 3/4.
2. **Piece 1:** oracle-compatible address derivation in `knirvwallet-module` + expose via
   `useVault.ts`. Ships independently, immediately useful (display the address, use it for
   balance lookups against whatever `serverUrl` the network selector points at).
3. **Piece 2:** `/oracle/v3/wallet/register` in KNIRVORACLE, wire the controller to call it
   once it has a vault (registers the address, may trigger initial faucet funding).
4. **Piece 3 + 4 together:** nonce infra + `TransferSigned` in the oracle, matching
   `signOracleMessage` + submission call in the controller. Only after the spike passes.
5. Burn/vote signed variants, and any real UI (`useBackend.ts` currently mocks all
   balance/transaction data) — explicitly out of scope for this change, called out as natural
   follow-ups.

## Verification

- `knirvwallet-module`: existing Vitest suite (`wallet/wallet.spec.ts`, `encoding/bech32.spec.ts`)
  plus new unit tests for `publicKeyToOracleAddress` (assert against a known
  privkey→pubkey→address fixture matching what the Go side would derive) and
  `signOracleMessage`.
- KNIRVORACLE: extend `internal/oracle/routes/wallet_test.go` with tests for
  `/oracle/v3/wallet/register` and a new `transfer_signed_test.go` covering: valid signed
  transfer succeeds, wrong nonce rejected, signature from a different key rejected, replay of
  the same nonce rejected. Existing tests (`TestGenerateWalletFundsNewWallet`,
  `TestSendSignedTransactionTransfersOracleBalances`, `TestOracleFaucetFundsWallet`) must keep
  passing unmodified — this is additive.
- Manual: run KNIRVSERVER with `--testnet` (starts KNIRVORACLE embedded), point KNIRVCONTROLLER's
  Dev network setting at it, confirm the vault's oracle address round-trips through
  `/oracle/v3/wallet/register` and a signed transfer moves a real balance.
