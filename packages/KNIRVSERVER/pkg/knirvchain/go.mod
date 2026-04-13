module KNIRVCHAIN

go 1.23.3

replace github.com/libp2p/go-libp2p-p2p/protocol/circuitv2/autorelay => github.com/libp2p/go-libp2p-circuit v0.12.0

require github.com/stretchr/testify v1.11.1 // indirect

require (
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0
)
