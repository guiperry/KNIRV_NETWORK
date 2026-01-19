// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package demos

import (
	"context"
	"fmt"
	"log"
	"strings"

	"backend_server/internal/runtime"
)

// RouterNestDeployer deploys KNIRVROUTER NOC
type RouterNestDeployer struct {
	ContainerManager *runtime.UnifiedContainerManager
	Container        *runtime.UnifiedContainer
}

// NewRouterNestDeployer creates a new router nest deployer
func NewRouterNestDeployer(manager *runtime.UnifiedContainerManager) *RouterNestDeployer {
	return &RouterNestDeployer{
		ContainerManager: manager,
	}
}

// Deploy deploys the KNIRVROUTER NOC
func (rnd *RouterNestDeployer) Deploy(ctx context.Context) error {
	log.Println("🔀 Deploying KNIRVROUTER NOC...")

	config := &runtime.NestedObjectConfig{
		ObjectType:        runtime.ObjectTypeP2P,
		EnableViewport:    true,
		ViewportRenderers: []string{"http", "vnc"}, // No WebRTC to avoid recursion
		ServicePorts: map[string]int{
			"api":  5001,
			"p2p":  4001,
			"stun": 3478,
			"turn": 3479,
		},
		Metadata: map[string]interface{}{
			"image":   "knirvrouter:latest",
			"command": []interface{}{"/usr/local/bin/knirvrouter", "--config", "/config/nest.yaml"},
			"environment": map[string]interface{}{
				"KNIRV_MODE":    "router_nest",
				"ENABLE_DHT":    "true",
				"ENABLE_PUBSUB": "true",
				"ENABLE_TURN":   "true",
				"API_PORT":      "5001",
				"P2P_PORT":      "4001",
			},
			"volumes": map[string]interface{}{
				"/data/router": "/var/lib/knirvrouter",
			},
			"capabilities": []interface{}{
				"NET_ADMIN",        // For network operations
				"NET_BIND_SERVICE", // For privileged ports
			},
		},
	}

	container, err := rnd.ContainerManager.CreateNestedObject(ctx, config)
	if err != nil {
		return fmt.Errorf("KNIRVROUTER NOC creation failed: %w", err)
	}

	rnd.Container = container

	log.Printf("✅ KNIRVROUTER NOC deployed at knirv://%s.object.nest", container.CryptoHash)
	rnd.printAccessInfo(container)

	return nil
}

// Stop stops the KNIRVROUTER NOC
func (rnd *RouterNestDeployer) Stop(ctx context.Context) error {
	if rnd.Container == nil {
		return nil
	}

	log.Println("Stopping KNIRVROUTER NOC...")

	if err := rnd.ContainerManager.DestroyContainer(ctx, rnd.Container.ID); err != nil {
		return fmt.Errorf("KNIRVROUTER NOC stop failed: %w", err)
	}

	log.Println("✅ KNIRVROUTER NOC stopped")

	return nil
}

// GetContainer returns the deployed container
func (rnd *RouterNestDeployer) GetContainer() *runtime.UnifiedContainer {
	return rnd.Container
}

// printAccessInfo prints access information for the router nest
func (rnd *RouterNestDeployer) printAccessInfo(container *runtime.UnifiedContainer) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🔀 KNIRVROUTER OBJECT NEST")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
	fmt.Println("📡 SERVICES:")
	fmt.Println("  Router API:       http://localhost:5001")
	fmt.Println("  P2P Listen:       tcp://localhost:4001")
	fmt.Println("  STUN Server:      udp://localhost:3478")
	fmt.Println("  TURN Server:      udp://localhost:3479")
	fmt.Println()
	fmt.Println("🔗 OBJECT NEST URL:")
	fmt.Printf("  knirv://%s.object.nest?type=p2p\n", container.CryptoHash)
	fmt.Println()
	fmt.Println("🧪 QUICK TESTS:")
	fmt.Println("  curl http://localhost:5001/stats")
	fmt.Println("  curl http://localhost:5001/dht/peers")
	fmt.Println("  curl http://localhost:5001/topology")
	fmt.Println()
	fmt.Println("💡 NOTE: This router instance runs inside a container and can")
	fmt.Println("   be used to test multi-hop routing and nested P2P networks.")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
}
