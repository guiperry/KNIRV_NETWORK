# KNIRVENGINE npm installer

This package follows the KNIRV CLI npm-install convention. It selects the
matching published installer, downloads it from Cloudflare R2, and runs it in
the foreground so Linux `sudo`, macOS administrator, and Windows UAC prompts
remain visible.

```bash
npm install -g @knirv/engine-installer
```

For the scoped package to be public on npm, publish with public access (the
included `publishConfig` sets this automatically):

```bash
npm publish --access public
```

Published installer objects:

- Linux x64: `knirvengine-installer-linux-amd64`
- macOS x64 / Apple Silicon: `knirvengine-installer-macos-amd64` / `knirvengine-installer-macos-arm64`
- Windows x64: `knirvengine-installer-windows-amd64.exe`

Useful environment variables:

```bash
# Test platform selection without downloading or launching.
npm run test:installer

# Use a staging R2 endpoint.
KNIRVENGINE_INSTALLER_BASE_URL=https://example.invalid/installer npm install -g @knirv/engine-installer

# Download but do not initialize Docker/KNIRVENGINE.
KNIRVENGINE_INSTALLER_NO_START=1 npm install -g @knirv/engine-installer
```
