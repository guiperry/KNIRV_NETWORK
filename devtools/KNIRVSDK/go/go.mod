module github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go

go 1.24.0

toolchain go1.24.1

require (
	github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/gateway v0.0.0-00010101000000-000000000000
// github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/transaction v0.0.0-00010101000000-000000000000 // TODO: Fix import paths
// github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/transmission v0.0.0-00010101000000-000000000000 // TODO: Fix dependencies
)

require github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/oracled v0.0.0-00010101000000-000000000000

replace github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/gateway => ./gateway

replace github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/oracled => ./oracled

replace github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE => ../../KNIRVORACLE
