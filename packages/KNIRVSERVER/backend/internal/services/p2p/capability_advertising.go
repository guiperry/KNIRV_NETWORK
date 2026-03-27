// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package p2p

import (
	"backend_server/internal/runtime"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// capabilitiesProtocol is the libp2p protocol identifier for capability queries.
// A connecting peer sends a JSON request token and receives a JSON
// []NodeCapability array in response.
const capabilitiesProtocol = protocol.ID("/knirv/dve/capabilities/1.0.0")

const (
	CapabilityAgenticRuntime = "agentic-runtime-support"
	CapabilityDVERouting     = "dve-routing"
	CapabilityTEEAttestation = "tee-attestation"
)

type NodeCapability struct {
	Capability string                 `json:"capability"`
	Version    string                 `json:"version"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type CapabilityAnnouncement struct {
	NodeID       string           `json:"node_id"`
	Capabilities []NodeCapability `json:"capabilities"`
	Timestamp    int64            `json:"timestamp"`
}

func (dpm *DVEP2PManager) AdvertiseCapabilities(capabilities []NodeCapability) error {
	capabilityAnnouncement := CapabilityAnnouncement{
		NodeID:       dpm.host.ID().String(),
		Capabilities: capabilities,
		Timestamp:    time.Now().Unix(),
	}

	msgBytes, err := json.Marshal(capabilityAnnouncement)
	if err != nil {
		return fmt.Errorf("failed to marshal capability announcement: %w", err)
	}

	if err := dpm.dveAnnouncementTopic.Publish(dpm.ctx, msgBytes); err != nil {
		return fmt.Errorf("failed to publish capability announcement: %w", err)
	}

	log.Printf("[DVE][%s] Advertised %d capabilities", dpm.nodeRole, len(capabilities))
	return nil
}

func (dpm *DVEP2PManager) AdvertiseAgenticRuntimeCapability(agentCap *runtime.AgentRuntimeCapability) error {
	capabilities := []NodeCapability{
		{
			Capability: CapabilityAgenticRuntime,
			Version:    agentCap.AgentEngineVer,
			Metadata: map[string]interface{}{
				"supported_tools":     agentCap.SupportedTools,
				"active_memory_mount": agentCap.ActiveMemoryMount,
				"max_concurrent":      agentCap.MaxConcurrent,
				"viewport_enabled":    agentCap.ViewportEnabled,
			},
		},
	}

	return dpm.AdvertiseCapabilities(capabilities)
}

func (dpm *DVEP2PManager) GetPeerCapabilities(peerID string) ([]NodeCapability, error) {
	var capabilities []NodeCapability

	data, err := dpm.queryPeerCapabilities(peerID)
	if err != nil {
		return nil, err
	}

	if data != nil {
		if err := json.Unmarshal(data, &capabilities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
	}

	return capabilities, nil
}

// SetupCapabilityStreamHandler registers the server-side libp2p stream handler
// for the capabilities protocol.  When a remote peer opens a stream using
// capabilitiesProtocol, this node responds with its current capability list.
// Must be called after the host is started (i.e. after NewDVEP2PManager).
func (dpm *DVEP2PManager) SetupCapabilityStreamHandler() {
	dpm.host.SetStreamHandler(capabilitiesProtocol, func(s network.Stream) {
		defer s.Close()

		// Consume the incoming request token (we respond unconditionally).
		if err := s.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return
		}
		buf := make([]byte, 512)
		s.Read(buf) //nolint:errcheck – non-fatal; we still respond

		capabilities := []NodeCapability{
			{Capability: CapabilityDVERouting, Version: "1.0.0"},
			{Capability: CapabilityTEEAttestation, Version: "1.0.0"},
		}
		if dpm.nodeRole == "dve-manager" || dpm.nodeRole == "dve-coordinator" {
			capabilities = append(capabilities, NodeCapability{
				Capability: CapabilityAgenticRuntime,
				Version:    "1.0.0",
			})
		}
		data, err := json.Marshal(capabilities)
		if err != nil {
			return
		}
		s.Write(data) //nolint:errcheck – best-effort response
	})
	log.Printf("[DVE][%s] Capability stream handler registered", dpm.nodeRole)
}

// queryPeerCapabilities opens a libp2p stream to the given peer, requests its
// capability list, and returns the raw JSON bytes.
func (dpm *DVEP2PManager) queryPeerCapabilities(peerID string) ([]byte, error) {
	pid, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID %q: %w", peerID, err)
	}

	ctx, cancel := context.WithTimeout(dpm.ctx, 10*time.Second)
	defer cancel()

	stream, err := dpm.host.NewStream(ctx, pid, capabilitiesProtocol)
	if err != nil {
		return nil, fmt.Errorf("failed to open capability stream to peer %s: %w", peerID, err)
	}
	defer stream.Close()

	// Send request token and close write side to signal EOF to the reader.
	if _, err := stream.Write([]byte(`{"action":"get_capabilities"}`)); err != nil {
		return nil, fmt.Errorf("failed to send capability request: %w", err)
	}
	if err := stream.CloseWrite(); err != nil {
		return nil, fmt.Errorf("failed to close write side of stream: %w", err)
	}

	// Read response with a firm deadline to prevent hanging.
	if err := stream.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	resp, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read capability response: %w", err)
	}
	return resp, nil
}

func (dpm *DVEP2PManager) HasCapability(peerID, capability string) (bool, error) {
	capabilities, err := dpm.GetPeerCapabilities(peerID)
	if err != nil {
		return false, err
	}

	for _, cap := range capabilities {
		if cap.Capability == capability {
			return true, nil
		}
	}

	return false, nil
}

func (dpm *DVEP2PManager) FindNodesWithCapability(capability string) ([]string, error) {
	var matchingPeers []string

	peers := dpm.host.Network().Peers()
	for _, peer := range peers {
		hasCap, err := dpm.HasCapability(peer.String(), capability)
		if err != nil {
			continue
		}
		if hasCap {
			matchingPeers = append(matchingPeers, peer.String())
		}
	}

	return matchingPeers, nil
}

func (dpm *DVEP2PManager) FindAgenticRuntimeNodes() ([]string, error) {
	return dpm.FindNodesWithCapability(CapabilityAgenticRuntime)
}

func (dpm *DVEP2PManager) StartCapabilityAdvertising(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dpm.broadcastCapabilityHeartbeat()
		}
	}
}

func (dpm *DVEP2PManager) broadcastCapabilityHeartbeat() {
	capabilityAnnouncement := CapabilityAnnouncement{
		NodeID: dpm.host.ID().String(),
		Capabilities: []NodeCapability{
			{
				Capability: CapabilityDVERouting,
				Version:    "1.0.0",
			},
			{
				Capability: CapabilityTEEAttestation,
				Version:    "1.0.0",
			},
		},
		Timestamp: time.Now().Unix(),
	}

	msgBytes, err := json.Marshal(capabilityAnnouncement)
	if err != nil {
		log.Printf("[DVE][%s] Error marshaling capability heartbeat: %v", dpm.nodeRole, err)
		return
	}

	if err := dpm.dveAnnouncementTopic.Publish(dpm.ctx, msgBytes); err != nil {
		log.Printf("[DVE][%s] Error publishing capability heartbeat: %v", dpm.nodeRole, err)
	}
}
