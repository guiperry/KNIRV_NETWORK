# 3D Models Directory

This directory is used to store 3D models for the KNIRVCHAIN blockchain application.

## Supported Formats

The application now supports multiple 3D model formats:

- **GLB**: Binary form of glTF (GL Transmission Format) - recommended for web use
- **GLTF**: JSON-based format for 3D models with external resources
- **USDZ**: Apple's format for AR/VR content
- **Markdown**: Text files with .md extension for documentation

## Adding Models

You can add models to this directory in two ways:

1. **Through the application**: Use the "Upload Object" button in the sidebar panel.

2. **Manually**: Copy model files directly to this directory. The application will scan this directory on startup.

## Finding 3D Model Files

You can find 3D model files from various sources:

1. **Sketchfab**: https://sketchfab.com/ (offers models in GLB, GLTF, and USDZ formats)
2. **TurboSquid**: https://www.turbosquid.com/ (professional 3D models)
3. **Apple's AR Quick Look Gallery**: https://developer.apple.com/augmented-reality/quick-look/ (USDZ files)
4. **Blender**: Create your own models and export them in GLB or GLTF format
5. **Google Poly Archive**: https://poly.google.com/ (archive available with many 3D models)

## Converting Between Formats

If you have models in other formats, you can convert them:

- **Blender**: Open and export to GLB/GLTF
- **Online converters**: Services like https://modelconverter.com/
- **glTF Transform**: Command-line tool for converting and optimizing 3D models

## Viewing Models

Once models are added to this directory, they will appear in the 3D Asset Viewer panel in the application. Select a model from the list to view it in the 3D viewport.

The viewer supports:

- Rotating the model (left-click and drag)
- Panning (right-click and drag)
- Zooming (scroll wheel)
- Resetting the view (Reset View button)

## Blockchain Integration

When a model is selected or uploaded, a blockchain asset is automatically created for it, and a new block is added to the blockchain. Each model is assigned a unique ID and tracked in the blockchain ledger.
