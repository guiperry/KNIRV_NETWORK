// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"fmt"
	"log"
	"net"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

// XDPManager handles XDP-specific eBPF programs and operations
type XDPManager struct {
	manager      *Manager
	collection   *ebpf.Collection
	links        []link.Link
	whitelistMap *ebpf.Map
}

// NewXDPManager creates a new XDP Manager instance
func NewXDPManager(manager *Manager) *XDPManager {
	return &XDPManager{
		manager: manager,
	}
}

// InitializeXDP initializes XDP programs and maps
func (x *XDPManager) InitializeXDP() error {
	// Load XDP program
	loader := NewXDPLoader()
	if err := loader.LoadXDPFilter(); err != nil {
		return fmt.Errorf("load XDP filter: %w", err)
	}

	x.collection = loader.collection
	x.links = loader.links
	x.whitelistMap = loader.whitelistMap

	log.Println("XDP Manager initialized successfully")
	return nil
}

// AttachXDPToInterface attaches XDP program to a network interface
func (x *XDPManager) AttachXDPToInterface(iface string) error {
	if x.collection == nil {
		return fmt.Errorf("XDP collection not initialized")
	}

	// Get the XDP program
	prog := x.collection.Programs["xdp_rate_limit"]
	if prog == nil {
		return fmt.Errorf("XDP program not found in collection")
	}

	// Get interface index from name
	netIface, err := net.InterfaceByName(iface)
	if err != nil {
		return fmt.Errorf("get interface %s: %w", iface, err)
	}

	// Attach to interface
	link, err := link.AttachXDP(link.XDPOptions{
		Program:   prog,
		Interface: netIface.Index,
	})
	if err != nil {
		return fmt.Errorf("attach XDP to interface %s: %w", iface, err)
	}

	x.links = append(x.links, link)
	log.Printf("XDP program attached to interface %s (index %d)", iface, netIface.Index)
	return nil
}

// AddWhitelistedIP adds an IP to the XDP whitelist
func (x *XDPManager) AddWhitelistedIP(ip net.IP) error {
	if x.whitelistMap == nil {
		return fmt.Errorf("whitelist map not initialized")
	}

	// Convert IP to uint32 (IPv4 only)
	ipv4 := ip.To4()
	if ipv4 == nil {
		return fmt.Errorf("only IPv4 addresses supported for XDP whitelist")
	}

	ipUint := uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3])

	// Add to whitelist map (value is dummy)
	var dummy uint8 = 1
	return x.whitelistMap.Put(ipUint, dummy)
}

// RemoveWhitelistedIP removes an IP from the XDP whitelist
func (x *XDPManager) RemoveWhitelistedIP(ip net.IP) error {
	if x.whitelistMap == nil {
		return fmt.Errorf("whitelist map not initialized")
	}

	// Convert IP to uint32 (IPv4 only)
	ipv4 := ip.To4()
	if ipv4 == nil {
		return fmt.Errorf("only IPv4 addresses supported for XDP whitelist")
	}

	ipUint := uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3])
	return x.whitelistMap.Delete(ipUint)
}

// GetNetworkMetrics returns XDP network metrics
func (x *XDPManager) GetNetworkMetrics() (*NetworkMetrics, error) {
	if x.collection == nil {
		return &NetworkMetrics{}, fmt.Errorf("XDP collection not initialized")
	}

	// Get rate limits map
	rateLimitsMap := x.collection.Maps["rate_limits"]
	if rateLimitsMap == nil {
		return &NetworkMetrics{}, fmt.Errorf("rate_limits map not found")
	}

	metrics := &NetworkMetrics{}
	var key, nextKey uint32
	var limit struct {
		Packets     uint64
		Bytes       uint64
		WindowStart uint64
		Dropped     uint64
	}

	// Iterate over all IPs
	for {
		if err := rateLimitsMap.NextKey(&key, &nextKey); err != nil {
			break
		}

		if err := rateLimitsMap.Lookup(nextKey, &limit); err != nil {
			continue
		}

		metrics.DroppedPackets += limit.Dropped
		metrics.AllowedPackets += limit.Packets

		if limit.Dropped > 1000 {
			ip := make(net.IP, 4)
			ip[0] = byte(nextKey >> 24)
			ip[1] = byte(nextKey >> 16)
			ip[2] = byte(nextKey >> 8)
			ip[3] = byte(nextKey)
			metrics.TopAttackers = append(metrics.TopAttackers, IPStats{
				IP:             ip,
				PacketsDropped: limit.Dropped,
			})
		}

		key = nextKey
	}

	return metrics, nil
}

// IsInitialized checks if the XDP manager has been initialized
func (x *XDPManager) IsInitialized() bool {
	return x.collection != nil
}

// GetCollection returns the eBPF collection (for testing purposes)
func (x *XDPManager) GetCollection() *ebpf.Collection {
	return x.collection
}

// ShutdownXDP cleans up XDP resources
func (x *XDPManager) ShutdownXDP() error {
	// Close all links
	for _, link := range x.links {
		if err := link.Close(); err != nil {
			log.Printf("Error closing XDP link: %v", err)
		}
	}
	x.links = nil

	// Close collection
	if x.collection != nil {
		x.collection.Close()
		x.collection = nil
	}

	log.Println("XDP Manager shutdown complete")
	return nil
}
