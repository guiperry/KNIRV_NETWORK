module github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go

go 1.24.0

toolchain go1.24.1

require (
	github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/gateway v0.0.0-00010101000000-000000000000
// github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/transaction v0.0.0-00010101000000-000000000000 // TODO: Fix import paths
// github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/transmission v0.0.0-00010101000000-000000000000 // TODO: Fix dependencies
)

require (
	github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/oracled v0.0.0-00010101000000-000000000000
	github.com/btcsuite/btcd/btcutil v1.1.0
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0
	golang.org/x/crypto v0.47.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/gateway => ./gateway

replace github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/oracled => ./oracled

replace github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE => ../../KNIRVORACLE
