// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package viewport

import (
	"fmt"
	"log"
)

// GLBMetadata represents GLB asset metadata
type GLBMetadata struct {
	Format    string `json:"format"`
	Polycount int    `json:"polycount"`
	FileSize  int64  `json:"file_size"`
}

// GLBRenderer provides WebGL-based rendering for 3D GLB objects
type GLBRenderer struct {
	assetPath string
	metadata  *GLBMetadata
	loaded    bool
}

// NewGLBRenderer creates a new GLB renderer
func NewGLBRenderer() *GLBRenderer {
	return &GLBRenderer{
		loaded: false,
		metadata: &GLBMetadata{
			Format:    "glb",
			Polycount: 0,
			FileSize:  0,
		},
	}
}

// LoadAsset loads a GLB asset
func (gr *GLBRenderer) LoadAsset(path string) error {
	gr.assetPath = path
	gr.loaded = true

	log.Printf("GLB asset loaded: %s", path)

	return nil
}

// RenderFrame renders a single frame
func (gr *GLBRenderer) RenderFrame() ([]byte, error) {
	if !gr.loaded {
		return nil, fmt.Errorf("no asset loaded")
	}

	// TODO: Implement actual rendering
	return []byte("rendered-frame-data"), nil
}

// GetMetadata returns asset metadata
func (gr *GLBRenderer) GetMetadata() *GLBMetadata {
	return gr.metadata
}

// Close closes the renderer
func (gr *GLBRenderer) Close() error {
	if !gr.loaded {
		return nil
	}

	gr.loaded = false

	log.Printf("GLB renderer closed for asset: %s", gr.assetPath)

	return nil
}
