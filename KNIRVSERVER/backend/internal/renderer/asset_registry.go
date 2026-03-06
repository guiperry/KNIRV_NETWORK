// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package renderer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// AssetRegistry manages 3D assets
type AssetRegistry struct {
	assets map[string]*Asset
	mu     sync.RWMutex
}

// Asset represents a 3D asset
type Asset struct {
	ID       string         `json:"id"`
	FilePath string         `json:"file_path"`
	FileSize int64          `json:"file_size"`
	Metadata *AssetMetadata `json:"metadata"`
	LoadedAt time.Time      `json:"loaded_at"`
}

// NewAssetRegistry creates a new asset registry
func NewAssetRegistry() *AssetRegistry {
	return &AssetRegistry{
		assets: make(map[string]*Asset),
	}
}

// Register registers a new asset
func (ar *AssetRegistry) Register(metadata *AssetMetadata) error {
	if metadata == nil {
		return fmt.Errorf("asset metadata is required")
	}
	if metadata.FilePath == "" {
		return fmt.Errorf("asset file path is required")
	}

	ar.mu.Lock()
	defer ar.mu.Unlock()

	asset := &Asset{
		ID:       generateAssetID(),
		FilePath: metadata.FilePath,
		FileSize: metadata.FileSize,
		Metadata: metadata,
		LoadedAt: time.Now(),
	}

	ar.assets[asset.ID] = asset
	return nil
}

// Get retrieves an asset by ID
func (ar *AssetRegistry) Get(id string) (*Asset, error) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	asset, ok := ar.assets[id]
	if !ok {
		return nil, fmt.Errorf("asset not found: %s", id)
	}

	return asset, nil
}

// GetByPath retrieves an asset by file path
func (ar *AssetRegistry) GetByPath(path string) (*Asset, error) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	for _, asset := range ar.assets {
		if asset.FilePath == path {
			return asset, nil
		}
	}

	return nil, fmt.Errorf("asset not found: %s", path)
}

// List returns all registered assets
func (ar *AssetRegistry) List() []*Asset {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	assets := make([]*Asset, 0, len(ar.assets))
	for _, asset := range ar.assets {
		assets = append(assets, asset)
	}

	return assets
}

// Remove removes an asset from the registry
func (ar *AssetRegistry) Remove(id string) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	if _, ok := ar.assets[id]; !ok {
		return fmt.Errorf("asset not found: %s", id)
	}

	delete(ar.assets, id)
	return nil
}

// Count returns the number of registered assets
func (ar *AssetRegistry) Count() int {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	return len(ar.assets)
}

// generateAssetID generates a unique asset ID
func generateAssetID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("asset-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("asset-%s", hex.EncodeToString(b))
}
