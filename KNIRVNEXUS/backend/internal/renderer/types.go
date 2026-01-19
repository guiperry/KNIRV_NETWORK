// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package renderer

// AssetMetadata defines metadata for 3D assets
type AssetMetadata struct {
	Format      string         `json:"format"` // glb, gltf, obj, etc.
	FileSize    int64          `json:"file_size"`
	FilePath    string         `json:"file_path"`
	Dimensions  *Dimensions3D  `json:"dimensions"`
	Materials   []string       `json:"materials"`
	Animations  []string       `json:"animations"`
	Textures    []TextureInfo  `json:"textures"`
	Polycount   int            `json:"polycount"`
	BoundingBox *BoundingBox3D `json:"bounding_box"`
}

// Dimensions3D represents 3D object dimensions
type Dimensions3D struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Depth  float64 `json:"depth"`
}

// BoundingBox3D represents a 3D bounding box
type BoundingBox3D struct {
	Min Vector3D `json:"min"`
	Max Vector3D `json:"max"`
}

// Vector3D represents a 3D vector
type Vector3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// TextureInfo represents texture metadata
type TextureInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Resolution string `json:"resolution"`
	Format     string `json:"format"`
}
