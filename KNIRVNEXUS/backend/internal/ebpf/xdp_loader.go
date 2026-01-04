// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"fmt"
	"log"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

// XDPLoader handles loading XDP-specific eBPF programs
type XDPLoader struct {
	collection   *ebpf.Collection
	links        []link.Link
	whitelistMap *ebpf.Map
}

// NewXDPLoader creates a new XDP Loader instance
func NewXDPLoader() *XDPLoader {
	return &XDPLoader{}
}

// LoadXDPFilter loads the XDP rate limiting program
func (l *XDPLoader) LoadXDPFilter() error {
	// Create XDP program spec
	spec := &ebpf.CollectionSpec{
		Programs: map[string]*ebpf.ProgramSpec{
			"xdp_rate_limit": {
				Type:    ebpf.XDP,
				License: "GPL",
			},
		},
		Maps: map[string]*ebpf.MapSpec{
			"rate_limits": {
				Type:       ebpf.LRUHash,
				KeySize:    4, // uint32
				ValueSize:  32, // struct rate_limit
				MaxEntries: 100000,
			},
			"whitelist": {
				Type:       ebpf.Hash,
				KeySize:    4, // uint32
				ValueSize:  1, // uint8
				MaxEntries: 10000,
			},
		},
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("create XDP collection: %w", err)
	}

	l.collection = coll
	l.whitelistMap = coll.Maps["whitelist"]

	log.Println("XDP Loader: XDP program loaded successfully")
	return nil
}