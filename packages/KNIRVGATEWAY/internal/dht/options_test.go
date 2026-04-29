//go:build test_dht

package dht

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func getLibp2pOptions(priv crypto.PrivKey, enableAutoRelay bool, port int) []libp2p.Option {
	return []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
			fmt.Sprintf("/ip6/::/tcp/%d", port),
		),
		libp2p.DisableRelay(),
	}
}
