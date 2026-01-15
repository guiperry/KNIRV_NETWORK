// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package demos

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/knirvcorp/knirvbase/go"
)

// GatewayNestDeployer deploys KNIRVGATEWAY NOC
type GatewayNestDeployer struct {
	ContainerManager *runtime.UnifiedContainerManager
	Container        *runtime.UnifiedContainer
}

// NewGatewayNestDeployer creates a new gateway nest deployer
func NewGatewayNestDeployer(manager *runtime.UnifiedContainerManager) *GatewayNestDeployer {
	return &GatewayNestDeployer{
		ContainerManager: manager,
	}
}

// Deploy deploys the KNIRVGATEWAY NOC
func (gnd *GatewayNestDeployer) Deploy(ctx context.Context) error {
	log.Println("🌐 Deploying KNIRVGATEWAY NOC...")

	config := &runtime.NestedObjectConfig{
		ObjectType:        runtime.ObjectTypeWebApp,
		EnableViewport:    true,
		ViewportRenderers: []string{"http", "webrtc", "vnc"},
		ServicePorts: map[string]int{
			"gateway":      8080,
			"oracle":       9090,
			"model_server": 11434,
		},
		Metadata: map[string]interface{}{
			"image":   "knirvgateway:latest",
			"command": []interface{}{"/usr/local/bin/gateway", "--oracle-embedded"},
			"environment": map[string]interface{}{
				"KNIRV_MODE":          "gateway_nest",
				"ENABLE_ORACLE":       "true",
				"ENABLE_MODEL_SERVER": "true",
				"GATEWAY_PORT":        "8080",
				"ORACLE_PORT":         "9090",
				"MODEL_SERVER_PORT":   "11434",
			},
			"volumes": map[string]interface{}{
				"/data/gateway": "/var/lib/knirvgateway",
				"/data/models":  "/models",
			},
		},
	}

	container, err := gnd.ContainerManager.CreateNestedObject(ctx, config)
	if err != nil {
		return fmt.Errorf("KNIRVGATEWAY NOC creation failed: %w", err)
	}

	gnd.Container = container

	log.Printf("✅ KNIRVGATEWAY NOC deployed at knirv://%s.object.nest", container.CryptoHash)
	gnd.printAccessInfo(container)

	return nil
}

// Stop stops the KNIRVGATEWAY NOC
func (gnd *GatewayNestDeployer) Stop(ctx context.Context) error {
	if gnd.Container == nil {
		return nil
	}

	log.Println("Stopping KNIRVGATEWAY NOC...")

	if err := gnd.ContainerManager.DestroyContainer(ctx, gnd.Container.ID); err != nil {
		return fmt.Errorf("KNIRVGATEWAY NOC stop failed: %w", err)
	}

	log.Println("✅ KNIRVGATEWAY NOC stopped")

	return nil
}

// GetContainer returns the deployed container
func (gnd *GatewayNestDeployer) GetContainer() *runtime.UnifiedContainer {
	return gnd.Container
}

// printAccessInfo prints access information for the gateway nest
func (gnd *GatewayNestDeployer) printAccessInfo(container *runtime.UnifiedContainer) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🌐 KNIRVGATEWAY OBJECT NEST")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
	fmt.Println("📡 SERVICES:")
	fmt.Println("  Gateway API:      http://localhost:8080")
	fmt.Println("  Documentation:    http://localhost:8080/docs")
	fmt.Println("  Oracle Daemon:    http://localhost:9090")
	fmt.Println("  Model Server:     http://localhost:11434")
	fmt.Println()
	fmt.Println("🔗 OBJECT NEST URL:")
	fmt.Printf("  knirv://%s.object.nest?type=webapp\n", container.CryptoHash)
	fmt.Println()
	fmt.Println("🧪 QUICK TESTS:")
	fmt.Println("  curl http://localhost:8080/health")
	fmt.Println("  curl http://localhost:9090/api/chains")
	fmt.Println("  curl http://localhost:11434/v1/models")
	fmt.Println()
	fmt.Println("💡 NOTE: The oracle daemon (knirv-oracled) runs as an embedded")
	fmt.Println("   service within the gateway. It will consume container resources")
	fmt.Println("   as needed for cross-chain operations and governance.")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
}
