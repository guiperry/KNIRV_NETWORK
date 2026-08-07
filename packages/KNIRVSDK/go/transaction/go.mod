module github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/transaction

go 1.24.0

toolchain go1.24.1

require (
	github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go v0.0.0
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0
	github.com/tidwall/gjson v1.14.4
	github.com/tidwall/sjson v1.2.5
)

replace github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go => ..

require (
	github.com/btcsuite/btcd/btcutil v1.1.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
