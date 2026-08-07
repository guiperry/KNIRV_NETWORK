module github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE

replace github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go => ../../../KNIRVSDK/go

go 1.24.0

toolchain go1.24.1

require (
	github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go v0.0.0
	github.com/ethereum/go-ethereum v1.13.15
	go.uber.org/zap v1.27.0
)

require (
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.0 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/holiman/uint256 v1.2.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
