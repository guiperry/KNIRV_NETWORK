# KNIRV Gateway Configuration Management System

## Overview

The KNIRV Gateway now includes a centralized configuration management system that allows for easy maintenance and updates of links, features, and settings across all portal components. This system uses YAML configuration files and JavaScript loaders to provide dynamic, maintainable configuration management.

## Architecture

### Configuration Files

#### 1. YAML Configuration (`config/portal-links.yaml`)
The master configuration file containing all links, settings, and feature flags:

```yaml
# Main Navigation Links
navigation:
  main_site: "https://knirv.com"
  documentation: "../documentation/docsify/"
  graphchain_explorer: "../graphchain-explorer/"
  nexus_portal: "../nexus-portal/"

# External Services
external_services:
  payment_gateway: "https://pay.knirv.com/add-funds"
  testnet_access: "https://testnet.knirv.com"

# Feature Flags
features:
  authentication_enabled: true
  payment_gateway_enabled: true
  nexus_integration_enabled: true
```

#### 2. JSON Configuration (`config/portal-config.json`)
Browser-compatible JSON version of the YAML configuration for direct loading.

### JavaScript Components

#### 1. Configuration Loader (`js/config-loader.js`)
The main configuration management class that:
- Loads configuration from JSON files
- Provides fallback configurations
- Updates UI elements dynamically
- Manages feature visibility

#### 2. Universal Footer (`js/universal-footer.js`)
Shared footer component that:
- Uses configuration for all links
- Maintains consistent branding
- Provides responsive design
- Integrates with all portal pages

## Usage Guide

### Basic Configuration Loading

```javascript
// Configuration is automatically loaded on page load
document.addEventListener('DOMContentLoaded', async () => {
    await window.knirvConfig.loadConfig();
    console.log('Configuration loaded:', window.knirvConfig.config);
});
```

### Accessing Configuration Values

```javascript
// Get navigation links
const mainSiteUrl = window.knirvConfig.getNavigationLink('main_site');
const docsUrl = window.knirvConfig.getNavigationLink('documentation');

// Get external service URLs
const paymentUrl = window.knirvConfig.getExternalService('payment_gateway');

// Check feature flags
if (window.knirvConfig.isFeatureEnabled('authentication_enabled')) {
    // Enable authentication features
}

// Get nested configuration values
const primaryColor = window.knirvConfig.getConfigValue('ui.theme.primary_color');
```

### Dynamic UI Updates

The configuration system automatically updates UI elements with `data-config` attributes:

```html
<!-- Navigation links -->
<a href="#" data-config="main-site">Back to KNIRV.com</a>
<a href="#" data-config-nav="documentation">Documentation</a>

<!-- Footer links -->
<a href="#" data-config-footer="social.github">GitHub</a>
<a href="#" data-config-footer="legal.terms">Terms of Service</a>

<!-- Feature-based visibility -->
<div data-feature="authentication_enabled">
    <!-- This div will be hidden if authentication is disabled -->
</div>

<!-- Payment gateway links -->
<button data-config="payment-gateway" onclick="addFunds()">Add Funds</button>

<!-- iFrame configurations -->
<iframe data-config-iframe="graphchain_explorer"></iframe>
```

## Configuration Reference

### Navigation Section
```yaml
navigation:
  main_site: "https://knirv.com"                    # Main KNIRV website
  documentation: "../documentation/docsify/"        # Documentation portal
  graphchain_explorer: "../graphchain-explorer/"    # GraphChain explorer
  nexus_portal: "../nexus-portal/"                  # KNIRV-NEXUS portal
  support_desk: "../support-desk/"                  # Support system
  nanda_ans: "../nanda_ans/"                        # NANDA+ANS registry
```

### External Services
```yaml
external_services:
  payment_gateway: "https://pay.knirv.com/add-funds"  # Payment processing
  knirv_website: "https://knirv.com"                  # Main website
  testnet_access: "https://testnet.knirv.com"         # Testnet access
```

### Documentation Links
```yaml
documentation:
  whitepapers:
    knirv_oracle: "../documentation/docsify/#/whitepapers/KNIRVROOT_Whitepaper.md"
    knirv_router: "../documentation/docsify/#/whitepapers/KNIRV-ROUTER_Whitepaper.md"
    # ... additional whitepapers
  guides:
    getting_started: "../documentation/docsify/#/guides/getting-started.md"
    api_reference: "../documentation/docsify/#/api/reference.md"
```

### Footer Configuration
```yaml
footer:
  legal:
    terms: "../documentation/docsify/#/legal/terms-of-service.md"
    privacy: "../documentation/docsify/#/legal/privacy-policy.md"
    contribution: "../documentation/docsify/#/contributing/contribution-guidelines.md"
  social:
    github: "https://github.com/knirv-network"
    discord: "https://discord.gg/knirv"
    twitter: "https://twitter.com/knirvnetwork"
    telegram: "https://t.me/knirvnetwork"
  resources:
    documentation: "../documentation/docsify/"
    support: "../support-desk/"
    forum: "../forum/"
    blog: "https://blog.knirv.com"
```

### Feature Flags
```yaml
features:
  authentication_enabled: true          # Enable/disable authentication
  payment_gateway_enabled: true         # Enable/disable payment features
  nexus_integration_enabled: true       # Enable/disable NEXUS integration
  graphchain_explorer_enabled: true     # Enable/disable GraphChain explorer
  nanda_ans_enabled: true              # Enable/disable NANDA+ANS features
  support_desk_enabled: true           # Enable/disable support desk
```

### UI Configuration
```yaml
ui:
  theme:
    primary_color: "#00c0fa"           # Primary brand color
    secondary_color: "#2b56f5"         # Secondary brand color
    accent_color: "#8b5cf6"            # Accent color
  branding:
    logo_url: "../logo/knirv-logo.png"
    favicon_url: "favicon.png"
    site_title: "KNIRV Developer Portal"
    site_description: "Build on the Decentralized Trusted Execution Network"
```

### iFrame Configuration
```yaml
iframes:
  graphchain_explorer:
    url: "../graphchain-explorer/"
    title: "KNIRV Graphchain Explorer"
    height: "800px"
  documentation:
    url: "../documentation/docsify/"
    title: "KNIRV Documentation"
    height: "800px"
```

## API Reference

### KNIRVConfigLoader Class

#### Methods

##### `loadConfig()`
Loads configuration from JSON file with fallback support.

```javascript
await window.knirvConfig.loadConfig();
```

##### `getNavigationLink(key)`
Returns a navigation URL by key.

```javascript
const url = window.knirvConfig.getNavigationLink('main_site');
```

##### `getDocumentationLink(category, key)`
Returns a documentation URL by category and key.

```javascript
const url = window.knirvConfig.getDocumentationLink('whitepapers', 'knirv_oracle');
```

##### `getFooterLink(category, key)`
Returns a footer URL by category and key.

```javascript
const url = window.knirvConfig.getFooterLink('social', 'github');
```

##### `getExternalService(key)`
Returns an external service URL by key.

```javascript
const url = window.knirvConfig.getExternalService('payment_gateway');
```

##### `isFeatureEnabled(feature)`
Checks if a feature is enabled.

```javascript
if (window.knirvConfig.isFeatureEnabled('authentication_enabled')) {
    // Feature is enabled
}
```

##### `getIframeConfig(key)`
Returns iFrame configuration by key.

```javascript
const config = window.knirvConfig.getIframeConfig('graphchain_explorer');
// Returns: { url: "...", title: "...", height: "..." }
```

##### `createLink(category, key, text, className)`
Creates a configured link element.

```javascript
const link = window.knirvConfig.createLink('navigation', 'main_site', 'Home', 'nav-link');
```

##### `createConfiguredButton(text, action, configKey, className)`
Creates a configured button element.

```javascript
const button = window.knirvConfig.createConfiguredButton(
    'Add Funds', 
    'payment_gateway', 
    'payment_gateway',
    'btn-primary'
);
```

## Universal Footer System

### Features
- **Consistent Branding**: Unified footer across all portal pages
- **Dynamic Links**: All links sourced from configuration
- **Responsive Design**: Mobile-friendly layout
- **Easy Integration**: Single script inclusion

### Implementation
```html
<!-- Include in any page -->
<script src="../js/universal-footer.js"></script>
```

### Customization
```javascript
// Add custom footer section
window.knirvFooter.addCustomSection('Custom Links', [
    { text: 'Custom Link 1', url: '/custom1', external: false },
    { text: 'External Link', url: 'https://example.com', external: true }
]);

// Update footer with new configuration
window.knirvFooter.updateFooterLinks(newConfig);
```

## Maintenance

### Updating Configuration

1. **Edit YAML File**: Modify `config/portal-links.yaml`
2. **Convert to JSON**: Update `config/portal-config.json` with the same changes
3. **Test Changes**: Verify all links and features work correctly
4. **Deploy**: Push changes to production

### Adding New Features

1. **Add Feature Flag**: Include in `features` section
2. **Update UI Elements**: Add `data-feature` attributes
3. **Test Feature Toggle**: Verify feature shows/hides correctly

### Adding New Links

1. **Choose Section**: Add to appropriate configuration section
2. **Update UI**: Add `data-config` attributes to HTML elements
3. **Test Links**: Verify all links resolve correctly

## Best Practices

1. **Use Relative Paths**: For internal links, use relative paths for portability
2. **Feature Flags**: Use feature flags for gradual rollouts
3. **Fallback URLs**: Always provide fallback URLs for critical links
4. **Testing**: Test configuration changes in development before production
5. **Documentation**: Update this documentation when adding new configuration options

## Troubleshooting

### Common Issues

1. **Configuration Not Loading**: Check browser console for fetch errors
2. **Links Not Updating**: Verify `data-config` attributes are correct
3. **Features Not Hiding**: Check feature flag names and values
4. **Footer Not Appearing**: Ensure universal-footer.js is loaded

### Debug Mode
```javascript
// Enable debug logging
window.knirvConfig.debugMode = true;
```

---

*This configuration system is designed to make KNIRV Gateway maintenance easier and more reliable across all portal components.*
