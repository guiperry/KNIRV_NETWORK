module github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go

go 1.23.8

toolchain go1.23.12

require (
	github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/gateway v0.0.0-00010101000000-000000000000
	// github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/transaction v0.0.0-00010101000000-000000000000 // TODO: Fix import paths
	// github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/transmission v0.0.0-00010101000000-000000000000 // TODO: Fix dependencies
)

replace github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/gateway => ./gateway

// replace github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/transaction => ./transaction

// replace github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/transmission => ./transmission
