// KNIRV Portal Configuration Loader
class KNIRVConfigLoader {
    constructor() {
        this.config = null;
        this.isLoaded = false;
        this.loadPromise = null;
    }

    async loadConfig() {
        if (this.loadPromise) {
            return this.loadPromise;
        }

        this.loadPromise = this._fetchConfig();
        return this.loadPromise;
    }

    async _fetchConfig() {
        try {
            // Try to load YAML config first
            const yamlResponse = await fetch('./config/portal-links.yaml');
            if (yamlResponse.ok) {
                const yamlText = await yamlResponse.text();
                this.config = this._parseYAML(yamlText);
                this._resolvePaths(this.config);
                this.isLoaded = true;
                return this.config;
            }

            // Fallback to JSON config
            const jsonResponse = await fetch('./config/portal-config.json');
            if (jsonResponse.ok) {
                this.config = await jsonResponse.json();
                this._resolvePaths(this.config);
                this.isLoaded = true;
                return this.config;
            }

            throw new Error('No config file found');
        } catch (error) {
            console.warn('Failed to load external config, using fallback:', error);
            // Fallback configuration
            this.config = this._getFallbackConfig();
            this.isLoaded = true;
            return this.config;
        }
    }

    // Resolve relative paths from the YAML config
    _resolvePaths(obj) {
        for (const key in obj) {
            if (typeof obj[key] === 'object' && obj[key] !== null) {
                this._resolvePaths(obj[key]);
            } else if (typeof obj[key] === 'string') {
                // Convert "../" paths to proper relative paths
                if (obj[key].startsWith('../')) {
                    obj[key] = obj[key].replace('../', '');
                }
            }
        }
    }

    // Simple YAML parser for basic YAML structures
    _parseYAML(yamlText) {
        const lines = yamlText.split('\n');
        const result = {};
        const stack = [result];
        let currentIndent = 0;

        for (let line of lines) {
            // Skip comments and empty lines
            if (line.trim().startsWith('#') || line.trim() === '') continue;

            const indent = line.length - line.trimStart().length;
            const trimmedLine = line.trim();

            // Handle indentation changes
            if (indent < currentIndent) {
                // Pop from stack until we reach the right level
                const levels = (currentIndent - indent) / 2;
                for (let i = 0; i < levels; i++) {
                    stack.pop();
                }
            }
            currentIndent = indent;

            if (trimmedLine.includes(':')) {
                const [key, ...valueParts] = trimmedLine.split(':');
                const value = valueParts.join(':').trim();
                const cleanKey = key.trim();

                const currentObj = stack[stack.length - 1];

                if (value === '' || value === '{}' || value === '[]') {
                    // This is a parent object
                    currentObj[cleanKey] = {};
                    stack.push(currentObj[cleanKey]);
                } else {
                    // This is a value
                    let parsedValue = value;

                    // Remove quotes if present
                    if ((value.startsWith('"') && value.endsWith('"')) ||
                        (value.startsWith("'") && value.endsWith("'"))) {
                        parsedValue = value.slice(1, -1);
                    }

                    // Parse boolean values
                    if (parsedValue === 'true') parsedValue = true;
                    else if (parsedValue === 'false') parsedValue = false;
                    // Parse numbers
                    else if (!isNaN(parsedValue) && parsedValue !== '') {
                        parsedValue = Number(parsedValue);
                    }

                    currentObj[cleanKey] = parsedValue;
                }
            }
        }

        return result;
    }

    _getFallbackConfig() {
        return {
            navigation: {
                main_site: "https://knirv.com",
                documentation: "documentation/docsify/",
                graphchain_explorer: "graphchain-explorer/",
                nexus_portal: "nexus-portal/",
                support_desk: "support-desk/",
                nanda_ans: "nanda_ans/"
            },
            external_services: {
                payment_gateway: "https://pay.knirv.com/add-funds",
                knirv_website: "https://knirv.com",
                testnet_access: "https://testnet.knirv.com"
            },
            documentation: {
                whitepapers: {
                    knirv_oracle: "documentation/docsify/#/whitepapers/KNIRVROOT_Whitepaper.md",
                    knirv_router: "documentation/docsify/#/whitepapers/KNIRV-ROUTER_Whitepaper.md",
                    knirvgraph: "documentation/docsify/#/whitepapers/KNIRV-GRAPH_Whitepaper.md",
                    knirvchain: "documentation/docsify/#/whitepapers/KNIRVCHAIN_Whitepaper.md",
                    knirv_nexus: "documentation/docsify/#/whitepapers/KNIRV-NEXUS_Whitepaper.md",
                    knirv_cortex: "documentation/docsify/#/whitepapers/KNIRV-AGENTIFIER_Whitepaper.md",
                    knirv_wallet: "documentation/docsify/#/whitepapers/KNIRV-WALLET_Whitepaper.md",
                    knirv_gateway: "documentation/docsify/#/whitepapers/KNIRV-GATEWAY_Whitepaper.md",
                    knirv_shell: "documentation/docsify/#/whitepapers/KNIRV-SHELL_Whitepapers.md",
                    knirv_sdk: "documentation/docsify/#/whitepapers/KNIRV-SDK_Whitepaper.md",
                    knirv_testnet: "documentation/docsify/#/guides/TESTNET_DEPLOYMENT.md",
                    knirvana: "documentation/docsify/#/whitepapers/KNIRVANA_Whitepaper.md"
                }
            },
            footer: {
                legal: {
                    terms: "documentation/docsify/#/legal/terms-of-service.md",
                    privacy: "documentation/docsify/#/legal/privacy-policy.md",
                    contribution: "documentation/docsify/#/contributing/contribution-guidelines.md"
                },
                social: {
                    github: "https://github.com/knirv-network",
                    discord: "https://discord.gg/knirv",
                    twitter: "https://twitter.com/knirvnetwork",
                    telegram: "https://t.me/knirvnetwork"
                },
                resources: {
                    documentation: "documentation/docsify/",
                    support: "support-desk/",
                    forum: "forum/",
                    blog: "https://blog.knirv.com"
                }
            },
            features: {
                authentication_enabled: true,
                payment_gateway_enabled: true,
                nexus_integration_enabled: true,
                graphchain_explorer_enabled: true,
                nanda_ans_enabled: true,
                support_desk_enabled: true
            },
            iframes: {
                graphchain_explorer: {
                    url: "graphchain-explorer/",
                    title: "KNIRV Graphchain Explorer",
                    height: "800px"
                },
                documentation: {
                    url: "documentation/docsify/",
                    title: "KNIRV Documentation",
                    height: "800px"
                },
                nexus_portal: {
                    url: "nexus-portal/",
                    title: "KNIRV Nexus Portal",
                    height: "800px"
                }
            }
        };
    }

    // Utility methods to get configuration values
    getNavigationLink(key) {
        return this.config?.navigation?.[key] || '#';
    }

    getDocumentationLink(category, key) {
        return this.config?.documentation?.[category]?.[key] || '#';
    }

    getFooterLink(category, key) {
        return this.config?.footer?.[category]?.[key] || '#';
    }

    getExternalService(key) {
        return this.config?.external_services?.[key] || '#';
    }

    isFeatureEnabled(feature) {
        return this.config?.features?.[feature] || false;
    }

    getIframeConfig(key) {
        return this.config?.iframes?.[key] || null;
    }

    // Apply configuration to the current page
    async applyConfiguration() {
        if (!this.isLoaded) {
            await this.loadConfig();
        }

        this._updateNavigationLinks();
        this._updateFooterLinks();
        this._updateFeatureVisibility();
        this._updateIframeConfigs();
    }

    _updateNavigationLinks() {
        // Update main site link
        const mainSiteLinks = document.querySelectorAll('[data-config="main-site"]');
        mainSiteLinks.forEach(link => {
            link.href = this.getNavigationLink('main_site');
        });

        // Update documentation links
        const docLinks = document.querySelectorAll('[data-config="documentation"]');
        docLinks.forEach(link => {
            link.href = this.getNavigationLink('documentation');
        });

        // Update other navigation links
        const navLinks = document.querySelectorAll('[data-config-nav]');
        navLinks.forEach(link => {
            const key = link.getAttribute('data-config-nav');
            link.href = this.getNavigationLink(key);
        });
    }

    _updateFooterLinks() {
        // Update footer links
        const footerLinks = document.querySelectorAll('[data-config-footer]');
        footerLinks.forEach(link => {
            const [category, key] = link.getAttribute('data-config-footer').split('.');
            link.href = this.getFooterLink(category, key);
        });
    }

    _updateFeatureVisibility() {
        // Show/hide features based on configuration
        const featureElements = document.querySelectorAll('[data-feature]');
        featureElements.forEach(element => {
            const feature = element.getAttribute('data-feature');
            if (!this.isFeatureEnabled(feature)) {
                element.style.display = 'none';
            }
        });
    }

    _updateIframeConfigs() {
        // Update iframe configurations
        const iframes = document.querySelectorAll('[data-config-iframe]');
        iframes.forEach(iframe => {
            const key = iframe.getAttribute('data-config-iframe');
            const config = this.getIframeConfig(key);
            if (config) {
                iframe.src = config.url;
                iframe.title = config.title;
                if (config.height) {
                    iframe.style.height = config.height;
                }
            }
        });
    }

    // Dynamic link creation
    createLink(category, key, text, className = '') {
        const link = document.createElement('a');
        link.href = this.getNavigationLink(key) || this.getDocumentationLink(category, key) || '#';
        link.textContent = text;
        if (className) {
            link.className = className;
        }
        return link;
    }

    // Dynamic button creation with configuration
    createConfiguredButton(text, action, configKey, className = 'btn-primary') {
        const button = document.createElement('button');
        button.textContent = text;
        button.className = className;
        
        if (typeof action === 'string') {
            // If action is a string, treat it as a URL
            button.onclick = () => {
                const url = this.getNavigationLink(configKey) || action;
                window.open(url, '_blank');
            };
        } else if (typeof action === 'function') {
            // If action is a function, use it directly
            button.onclick = action;
        }
        
        return button;
    }

    // Update payment gateway links
    updatePaymentLinks() {
        const paymentLinks = document.querySelectorAll('[data-config="payment-gateway"]');
        paymentLinks.forEach(link => {
            link.href = this.getExternalService('payment_gateway');
        });
    }

    // Get configuration value by path (e.g., 'navigation.main_site')
    getConfigValue(path) {
        return path.split('.').reduce((obj, key) => obj?.[key], this.config);
    }
}

// Global configuration instance
window.knirvConfig = new KNIRVConfigLoader();

// Auto-load configuration when DOM is ready
document.addEventListener('DOMContentLoaded', async () => {
    try {
        await window.knirvConfig.loadConfig();
        await window.knirvConfig.applyConfiguration();
        console.log('KNIRV Portal configuration loaded successfully');
    } catch (error) {
        console.error('Failed to load KNIRV Portal configuration:', error);
    }
});

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = KNIRVConfigLoader;
}
