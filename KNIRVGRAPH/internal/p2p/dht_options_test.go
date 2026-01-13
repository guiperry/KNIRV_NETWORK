//go:build test_dht

package p2p

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func getLibp2pOptions(priv crypto.PrivKey, enableAutoRelay bool) []libp2p.Option {
	// For testing, we explicitly disable AutoRelay and other network services
	// that might cause panics in a simulated environment.
	// The enableAutoRelay flag is effectively ignored here.
	return []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9001",
			"/ip6/::/tcp/9001",
		),
		// Explicitly disable relay functionality for tests to avoid panics
		libp2p.DisableRelay(), 
		// NAT service and Hole Punching are implicitly disabled if not explicitly enabled.
	}
}
