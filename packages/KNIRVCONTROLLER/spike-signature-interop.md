# TS↔Go Signature Interop Spike

## Purpose

Verify that `knirvwallet-module`'s `signOracleMessage` (TypeScript/secp256k1 via `@cosmjs/crypto`)
produces signatures that Go's `crypto.RecoverAddress` (`ethcrypto.SigToPub`) can recover.

## Risk

cosmjs's recovery-id convention (0/1 range, low-S normalization) and go-ethereum's
`ethcrypto.Sign`/`SigToPub` both wrap standard secp256k1 ECDSA, but byte-for-byte
interop between two different library implementations is the kind of thing that
looks right on paper and silently breaks on edge cases (recovery id offset,
S-normalization, hash truncation).

## Procedure

### Step 1: Sign a fixed message in TypeScript

Run a temporary script in the `knirvwallet-module` package:

```ts
import { Secp256k1 } from '@cosmjs/crypto';
import { keccak_256 } from '@noble/hashes/sha3';
import { toHex } from './encoding/hex';

const testPrivkey = new Uint8Array(32);
testPrivkey[31] = 1; // known private key for testing

const message = 'transfer:0x7e5f4552091a69125d5dfcb7b8c2659029395bdf:0x0000000000000000000000000000000000000000:100:0';
const messageHash = keccak_256(new TextEncoder().encode(message));
const { pubkey } = await Secp256k1.makeKeypair(testPrivkey);
const compressed = Secp256k1.compressPubkey(pubkey);
const signature = await Secp256k1.createSignature(messageHash, testPrivkey);
const sigHex = '0x' + toHex(signature.toFixedLength());
const oracleAddr = '0x' + toHex(keccak_256(compressed.slice(1)).slice(12));

console.log('message:', message);
console.log('address:', oracleAddr);
console.log('signature:', sigHex);
```

### Step 2: Hardcode the output in a Go test

Add to `internal/oracle/routes/wallet_test.go`:

```go
func TestTSGoSignatureRoundTrip(t *testing.T) {
    message := "transfer:0x7e5f4552091a69125d5dfcb7b8c2659029395bdf:0x0000000000000000000000000000000000000000:100:0"
    sigHex := "0x<insert_sig_from_step1>"
    expectedAddr := "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf"

    sig, err := hex.DecodeString(strings.TrimPrefix(sigHex, "0x"))
    if err != nil {
        t.Fatalf("decode sig: %v", err)
    }

    recovered, err := crypto.RecoverAddress([]byte(message), sig)
    if err != nil {
        t.Fatalf("recover address: %v", err)
    }
    if recovered.String() != expectedAddr {
        t.Fatalf("recovered address = %s, want %s", recovered.String(), expectedAddr)
    }
}
```

### Step 3: Run the Go test

```bash
cd packages/KNIRVORACLE && go test -v -run TestTSGoSignatureRoundTrip ./internal/oracle/routes/
```

### Step 4: If the test passes

Proceed to wire `signOracleMessage` + `sendNRN` in the controller's transfer flow.

### Step 5: If the test fails

Investigate the recovery-id offset, S-normalization, or hash truncation difference
between `@cosmjs/crypto` and `go-ethereum/crypto`. Both libraries wrap standard
secp256k1 ECDSA, but subtle implementation differences can cause byte-for-byte
signature mismatches.

## Pass condition

`crypto.RecoverAddress` in Go recovers the same address that `publicKeyToOracleAddress`
derives in TypeScript from the same private key.