// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package deployment

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/knirvcorp/knirvbase/go"
)

// DemoDeployer manages automatic deployment of all demo NOCs
type DemoDeployer struct {
	containerManager *runtime.UnifiedContainerManager
	gatewayDeployer  *demos.GatewayNestDeployer
	routerDeployer   *demos.RouterNestDeployer
	deployed         bool
	mu               sync.RWMutex
}

// NewDemoDeployer creates a new demo deployer
func NewDemoDeployer(containerManager *runtime.UnifiedContainerManager) *DemoDeployer {
	return &DemoDeployer{
		containerManager: containerManager,
		gatewayDeployer:  demos.NewGatewayNestDeployer(containerManager),
		routerDeployer:   demos.NewRouterNestDeployer(containerManager),
		deployed:         false,
	}
}

// DeployAll deploys all demo NOCs in parallel
func (dd *DemoDeployer) DeployAll(ctx context.Context) error {
	dd.mu.Lock()
	if dd.deployed {
		dd.mu.Unlock()
		return fmt.Errorf("demos already deployed")
	}
	dd.mu.Unlock()

	log.Println("🚀 Deploying demo NOCs...")

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Deploy KNIRVGATEWAY in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := dd.gatewayDeployer.Deploy(ctx); err != nil {
			errChan <- fmt.Errorf("gateway deployment failed: %w", err)
		}
	}()

	// Deploy KNIRVROUTER in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := dd.routerDeployer.Deploy(ctx); err != nil {
			errChan <- fmt.Errorf("router deployment failed: %w", err)
		}
	}()

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		// If any deployment fails, cleanup and return error
		log.Printf("ERROR: %v", err)
		dd.Cleanup(context.Background())
		return err
	}

	dd.mu.Lock()
	dd.deployed = true
	dd.mu.Unlock()

	log.Println("✅ All demo NOCs deployed successfully")

	return nil
}

// Cleanup removes all demo NOCs
func (dd *DemoDeployer) Cleanup(ctx context.Context) error {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	if !dd.deployed {
		return nil
	}

	log.Println("🧹 Cleaning up demo NOCs...")

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Cleanup gateway
	if dd.gatewayDeployer.GetContainer() != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dd.gatewayDeployer.Stop(ctx); err != nil {
				errChan <- fmt.Errorf("gateway cleanup failed: %w", err)
			}
		}()
	}

	// Cleanup router
	if dd.routerDeployer.GetContainer() != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dd.routerDeployer.Stop(ctx); err != nil {
				errChan <- fmt.Errorf("router cleanup failed: %w", err)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Log any cleanup errors but don't fail
	for err := range errChan {
		log.Printf("Cleanup error: %v", err)
	}

	dd.deployed = false
	log.Println("✅ Demo NOCs cleaned up")

	return nil
}

// IsDeployed returns true if demos are deployed
func (dd *DemoDeployer) IsDeployed() bool {
	dd.mu.RLock()
	defer dd.mu.RUnlock()
	return dd.deployed
}

// GetGatewayContainer returns the gateway container if deployed
func (dd *DemoDeployer) GetGatewayContainer() *runtime.UnifiedContainer {
	return dd.gatewayDeployer.GetContainer()
}

// GetRouterContainer returns the router container if deployed
func (dd *DemoDeployer) GetRouterContainer() *runtime.UnifiedContainer {
	return dd.routerDeployer.GetContainer()
}

// GetStatus returns the deployment status
func (dd *DemoDeployer) GetStatus() map[string]interface{} {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	status := map[string]interface{}{
		"deployed": dd.deployed,
	}

	if dd.deployed {
		gatewayContainer := dd.gatewayDeployer.GetContainer()
		routerContainer := dd.routerDeployer.GetContainer()

		status["gateway"] = map[string]interface{}{
			"id":     gatewayContainer.ID,
			"hash":   gatewayContainer.CryptoHash,
			"status": gatewayContainer.Status,
		}

		status["router"] = map[string]interface{}{
			"id":     routerContainer.ID,
			"hash":   routerContainer.CryptoHash,
			"status": routerContainer.Status,
		}
	}

	return status
}
