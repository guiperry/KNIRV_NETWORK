//go:build !test_dht

package p2p

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func getLibp2pOptions(priv crypto.PrivKey, enableAutoRelay bool) []libp2p.Option {
	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9001",
			"/ip6/::/tcp/9001",
		),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
	}

	if enableAutoRelay {
		opts = append(opts, libp2p.EnableAutoRelay())
	} else {
		opts = append(opts, libp2p.DisableRelay())
	}
	return opts
}
