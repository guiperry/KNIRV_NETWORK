# KNIRV Gateway Download Configuration System

## Overview

All download buttons across KNIRVGATEWAY product pages now use a centralized configuration system that loads download links from `config/portal-links.yaml`. This eliminates hardcoded URLs and provides a single source of truth for all download links.

## Configuration File

The download configuration is stored in `KNIRVGATEWAY/config/portal-links.yaml` under the `downloads` section:

```yaml
downloads:
  knirvrouter:
    windows: "https://releases.knirv.network/knirvrouter/windows/knirvrouter-setup.exe"
    mac: "https://releases.knirv.network/knirvrouter/mac/knirvrouter.dmg"
    linux: "https://releases.knirv.network/knirvrouter/linux/knirvrouter.AppImage"
    requirements:
      windows: "Windows 10/11, 8GB RAM, 100GB Storage"
      mac: "macOS 10.15+, 8GB RAM, 100GB Storage"
      linux: "Ubuntu 20.04+/CentOS 8+, 8GB RAM, 100GB Storage"
    note: "You'll need 1000 NRN tokens to start routing."
```

## Supported Products

The system supports the following KNIRV products:

1. **KNIRVROUTER** - Network routing software
2. **KNIRVANA** - Gaming platform
3. **KNIRVORACLE** - Oracle bootnode software
4. **KNIRVSDK** - Software development kits
5. **KNIRVWALLET** - Wallet application
6. **KNIRVNEXUS** - DVE (Development Virtual Environment) services

## Implementation

### Config Loader (js/config-loader.js)

The `KNIRVConfigLoader` class handles:
- Loading configuration from `portal-links.yaml`
- Setting up global download functions
- Managing download link retrieval
- Handling analytics tracking

### Global Functions

The following global functions are automatically created:
- `downloadRouter(platform)` - KNIRVROUTER downloads
- `downloadGame(platform)` - KNIRVANA downloads
- `downloadBootnode(type)` - KNIRVORACLE downloads
- `downloadSDK(language)` - KNIRV SDK downloads
- `downloadWallet(platform)` - KNIRVWALLET downloads
- `rentDVE(plan)` - KNIRVNEXUS DVE rental
- `accessDVE(method)` - KNIRVNEXUS DVE access

### Product Pages

All product pages have been updated to remove hardcoded download functions:
- `knirvrouter.html`
- `knirvana.html`
- `knirvoracle.html`
- `knirvsdk.html`
- `knirvwallet.html`
- `knirvnexus.html`

## Configuration Structure

### Standard Downloads
```yaml
product_name:
  platform: "download_url"
  requirements:
    platform: "system requirements"
  note: "additional information"
```

### SDK Downloads
```yaml
knirvsdk:
  language: "install_command"
  instructions:
    language: "detailed installation instructions"
  documentation:
    language: "documentation_url"
```

## Benefits

1. **Centralized Management** - All download links in one place
2. **Easy Updates** - Change URLs without touching HTML files
3. **Consistency** - Uniform download experience across all products
4. **Maintainability** - Reduced code duplication
5. **Analytics** - Consistent tracking across all downloads

## Testing

Use `test-download-config.html` to verify the configuration system:
1. Open the test page in a browser
2. Check that configuration loads successfully
3. Test download buttons for all products
4. Verify that alerts show correct information from config

## Updating Download Links

To update download links:
1. Edit `config/portal-links.yaml`
2. Update the relevant product section
3. No changes needed to HTML files
4. Test using the test page

## Error Handling

The system includes fallback behavior:
- If configuration fails to load, uses fallback config
- If specific download not found, shows appropriate message
- Graceful degradation for missing platforms/languages

## Analytics

All downloads are tracked with Google Analytics:
- Event category: Product name (e.g., "KNIRVROUTER")
- Event action: "download" or "sdk_download"
- Event label: Platform or language

## Future Enhancements

Potential improvements:
- Version management for downloads
- Platform detection and auto-selection
- Download progress tracking
- Mirror URL support
- Checksum verification links
