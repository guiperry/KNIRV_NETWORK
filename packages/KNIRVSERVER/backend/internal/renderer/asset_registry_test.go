package renderer

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewAssetRegistry tests the creation of a new asset registry
func TestNewAssetRegistry(t *testing.T) {
	registry := NewAssetRegistry()

	assert.NotNil(t, registry)
	assert.Len(t, registry.assets, 0)
}

// TestRegister tests registering a new asset
func TestRegister(t *testing.T) {
	registry := NewAssetRegistry()

	metadata := &AssetMetadata{
		FilePath: "/path/to/asset.glb",
		FileSize: 1024,
	}

	err := registry.Register(metadata)
	assert.NoError(t, err)
	assert.Len(t, registry.assets, 1)

	// Check that the asset was registered correctly
	for _, asset := range registry.assets {
		assert.Equal(t, "/path/to/asset.glb", asset.FilePath)
		assert.Equal(t, int64(1024), asset.FileSize)
		assert.NotEmpty(t, asset.ID)
		assert.NotZero(t, asset.LoadedAt)
		assert.Equal(t, metadata, asset.Metadata)
	}
}

// TestRegisterWithNilMetadata tests registering with nil metadata
func TestRegisterWithNilMetadata(t *testing.T) {
	registry := NewAssetRegistry()

	err := registry.Register(nil)
	assert.Error(t, err)
	assert.Equal(t, "asset metadata is required", err.Error())
	assert.Len(t, registry.assets, 0)
}

// TestRegisterWithEmptyFilePath tests registering with empty file path
func TestRegisterWithEmptyFilePath(t *testing.T) {
	registry := NewAssetRegistry()

	metadata := &AssetMetadata{
		FilePath: "",
		FileSize: 1024,
	}

	err := registry.Register(metadata)
	assert.Error(t, err)
	assert.Equal(t, "asset file path is required", err.Error())
	assert.Len(t, registry.assets, 0)
}

// TestGet tests retrieving an asset by ID
func TestGet(t *testing.T) {
	registry := NewAssetRegistry()

	metadata := &AssetMetadata{
		FilePath: "/path/to/asset.glb",
		FileSize: 1024,
	}

	err := registry.Register(metadata)
	assert.NoError(t, err)

	// Get the asset ID
	var assetID string
	for id := range registry.assets {
		assetID = id
		break
	}

	// Retrieve the asset
	asset, err := registry.Get(assetID)
	assert.NoError(t, err)
	assert.NotNil(t, asset)
	assert.Equal(t, "/path/to/asset.glb", asset.FilePath)
	assert.Equal(t, int64(1024), asset.FileSize)
	assert.Equal(t, metadata, asset.Metadata)
}

// TestGetNonExistentAsset tests retrieving a non-existent asset
func TestGetNonExistentAsset(t *testing.T) {
	registry := NewAssetRegistry()

	_, err := registry.Get("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset not found")
}

// TestGetByPath tests retrieving an asset by file path
func TestGetByPath(t *testing.T) {
	registry := NewAssetRegistry()

	metadata := &AssetMetadata{
		FilePath: "/path/to/asset.glb",
		FileSize: 1024,
	}

	err := registry.Register(metadata)
	assert.NoError(t, err)

	// Retrieve the asset by path
	asset, err := registry.GetByPath("/path/to/asset.glb")
	assert.NoError(t, err)
	assert.NotNil(t, asset)
	assert.Equal(t, "/path/to/asset.glb", asset.FilePath)
	assert.Equal(t, int64(1024), asset.FileSize)
	assert.Equal(t, metadata, asset.Metadata)
}

// TestGetByPathNonExistent tests retrieving a non-existent asset by path
func TestGetByPathNonExistent(t *testing.T) {
	registry := NewAssetRegistry()

	_, err := registry.GetByPath("/non/existent/path.glb")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset not found")
}

// TestList tests listing all registered assets
func TestList(t *testing.T) {
	registry := NewAssetRegistry()

	// Register multiple assets
	metadata1 := &AssetMetadata{
		FilePath: "/path/to/asset1.glb",
		FileSize: 1024,
	}
	metadata2 := &AssetMetadata{
		FilePath: "/path/to/asset2.glb",
		FileSize: 2048,
	}

	err := registry.Register(metadata1)
	assert.NoError(t, err)

	err = registry.Register(metadata2)
	assert.NoError(t, err)

	assets := registry.List()
	assert.Len(t, assets, 2)

	// Check that both assets are in the list by checking their file paths
	foundAsset1 := false
	foundAsset2 := false
	for _, asset := range assets {
		if asset.FilePath == metadata1.FilePath {
			foundAsset1 = true
		}
		if asset.FilePath == metadata2.FilePath {
			foundAsset2 = true
		}
	}
	assert.True(t, foundAsset1, "asset1 not found in list")
	assert.True(t, foundAsset2, "asset2 not found in list")
}

// TestRemove tests removing an asset from the registry
func TestRemove(t *testing.T) {
	registry := NewAssetRegistry()

	metadata := &AssetMetadata{
		FilePath: "/path/to/asset.glb",
		FileSize: 1024,
	}

	err := registry.Register(metadata)
	assert.NoError(t, err)
	assert.Len(t, registry.assets, 1)

	// Get the asset ID
	var assetID string
	for id := range registry.assets {
		assetID = id
		break
	}

	// Remove the asset
	err = registry.Remove(assetID)
	assert.NoError(t, err)
	assert.Len(t, registry.assets, 0)
}

// TestRemoveNonExistentAsset tests removing a non-existent asset
func TestRemoveNonExistentAsset(t *testing.T) {
	registry := NewAssetRegistry()

	err := registry.Remove("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset not found")
}

// TestCount tests counting the number of registered assets
func TestCount(t *testing.T) {
	registry := NewAssetRegistry()

	assert.Equal(t, 0, registry.Count())

	// Register multiple assets
	metadata1 := &AssetMetadata{
		FilePath: "/path/to/asset1.glb",
		FileSize: 1024,
	}
	metadata2 := &AssetMetadata{
		FilePath: "/path/to/asset2.glb",
		FileSize: 2048,
	}

	err := registry.Register(metadata1)
	assert.NoError(t, err)

	assert.Equal(t, 1, registry.Count())

	err = registry.Register(metadata2)
	assert.NoError(t, err)

	assert.Equal(t, 2, registry.Count())
}

// TestGenerateAssetID tests the asset ID generation
func TestGenerateAssetID(t *testing.T) {
	id1 := generateAssetID()
	id2 := generateAssetID()

	// IDs should be different (with very high probability)
	assert.NotEqual(t, id1, id2)

	// IDs should start with "asset-"
	assert.Equal(t, "asset-", id1[:6])
	assert.Equal(t, "asset-", id2[:6])

	// After "asset-" should be a hex string
	assert.Regexp(t, `^asset-[0-9a-f]{32}$`, id1)
	assert.Regexp(t, `^asset-[0-9a-f]{32}$`, id2)
}

// TestAssetRegistryConcurrentAccess tests concurrent access to the asset registry
func TestAssetRegistryConcurrentAccess(t *testing.T) {
	registry := NewAssetRegistry()

	// Register assets concurrently
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			metadata := &AssetMetadata{
				FilePath: fmt.Sprintf("/path/to/asset%d.glb", index),
				// FileSize not needed for lookup
			}
			registry.Register(metadata)
		}(i)
	}
	wg.Wait()

	// Check that all assets were registered
	assert.Len(t, registry.assets, 100)

	// Remove assets concurrently
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(index int) {
			defer wg.Done()
			metadata := &AssetMetadata{
				FilePath: fmt.Sprintf("/path/to/asset%d.glb", index),
				// FileSize not needed for lookup
			}

			// Get the asset first to get its ID
			asset, _ := registry.GetByPath(metadata.FilePath)
			if asset != nil {
				registry.Remove(asset.ID)
			}
		}(i)
	}
	wg.Wait()

	// Check that all assets were removed
	assert.Len(t, registry.assets, 0)
}
