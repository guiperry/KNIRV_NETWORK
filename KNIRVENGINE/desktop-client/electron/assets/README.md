# Electron Assets

This directory contains assets for the Electron desktop application.

## Required Files

- `icon.png` - Main application icon (512x512 PNG)
- `icon.ico` - Windows icon file
- `icon.icns` - macOS icon file
- `dmg-background.png` - macOS DMG installer background image

## Icon Requirements

### PNG Icon (icon.png)
- Size: 512x512 pixels
- Format: PNG with transparency
- Used for: Linux builds and as source for other formats

### Windows Icon (icon.ico)
- Contains multiple sizes: 16x16, 32x32, 48x48, 64x64, 128x128, 256x256
- Format: ICO file
- Can be generated from PNG using online tools or ImageMagick

### macOS Icon (icon.icns)
- Contains multiple sizes for different display densities
- Format: ICNS file
- Can be generated from PNG using iconutil on macOS

## DMG Background (dmg-background.png)
- Size: 540x380 pixels
- Format: PNG
- Used as background image for macOS DMG installer

## Generating Icons

You can use the following tools to generate icons from a source PNG:

### Using ImageMagick (cross-platform)
```bash
# Generate ICO file
convert icon.png -resize 256x256 icon.ico

# Generate ICNS file (macOS only)
mkdir icon.iconset
sips -z 16 16 icon.png --out icon.iconset/icon_16x16.png
sips -z 32 32 icon.png --out icon.iconset/icon_16x16@2x.png
sips -z 32 32 icon.png --out icon.iconset/icon_32x32.png
sips -z 64 64 icon.png --out icon.iconset/icon_32x32@2x.png
sips -z 128 128 icon.png --out icon.iconset/icon_128x128.png
sips -z 256 256 icon.png --out icon.iconset/icon_128x128@2x.png
sips -z 256 256 icon.png --out icon.iconset/icon_256x256.png
sips -z 512 512 icon.png --out icon.iconset/icon_256x256@2x.png
sips -z 512 512 icon.png --out icon.iconset/icon_512x512.png
cp icon.png icon.iconset/icon_512x512@2x.png
iconutil -c icns icon.iconset
```

### Using online tools
- https://convertio.co/png-ico/
- https://cloudconvert.com/png-to-icns
- https://icoconvert.com/

## Placeholder Icons

For development purposes, you can create simple placeholder icons:

1. Create a 512x512 PNG with your app logo or a simple design
2. Use it as icon.png
3. Generate ICO and ICNS versions as needed

The build process will work with just the PNG file, but platform-specific formats provide better integration.
