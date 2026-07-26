//go:build !legacy_tcp

package network

import "log"

// startTransport is a no-op in the default build. The legacy custom TCP P2P
// transport is disabled unless the binary is built with the `legacy_tcp` tag.
// New deployments should route consensus through the p2pconsensus package.
func (n *NetworkManager) startTransport() {
	log.Printf("legacy P2P TCP transport disabled (build without -tags legacy_tcp)")
}

// connectToPeer is a no-op in the default build (see startTransport).
func (n *NetworkManager) connectToPeer(address string) {}
