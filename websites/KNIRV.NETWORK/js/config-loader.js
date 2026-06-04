// KNIRV Portal Configuration Loader - Unified Configuration System
class KNIRVConfigLoader {
    constructor() {
        this.config = null;
        this.isLoaded = false;
        this.loadPromise = null;
        this.oracleConfig = null;
        this.portalConfig = null;
    }

    async loadConfig() {
        if (this.loadPromise) {
            return this.loadPromise;
        }

        this.loadPromise = this._fetchUnifiedConfig();
        return this.loadPromise;
    }

    async _fetchUnifiedConfig() {
        try {
            // Load both configurations
            await this._loadGatewayConfig();
            await this._loadPortalConfig();

            // Merge configurations with portal config taking precedence
            this.config = this._mergeConfigurations();
            this.isLoaded = true;
            return this.config;
        } catch (error) {
            console.warn('Failed to load unified config, using fallback:', error);
            this.config = this._getFallbackConfig();
            this.isLoaded = true;
            return this.config;
        }
    }

    async _loadGatewayConfig() {
        // Gateway config is already loaded by oracle-config.js
        if (window.KNIRV_GATEWAY_CONFIG) {
            this.oracleConfig = window.KNIRV_GATEWAY_CONFIG;
        }
    }

    async _loadPortalConfig() {
        try {
            // Check if there's a script tag with portal-config id that specifies the config path
            const configScript = document.getElementById('portal-config');
            let configPath = './config/portal-links.yaml'; // default path

            if (configScript && configScript.src) {
                // Extract path from script src attribute
                configPath = configScript.src.replace(window.location.origin + window.location.pathname.replace(/\/[^\/]*$/, '/'), '');
            }

            // Try to load YAML config first
            const yamlResponse = await fetch(configPath);
            if (yamlResponse.ok) {
                const yamlText = await yamlResponse.text();
                this.portalConfig = this._parseYAML(yamlText);
                this._resolvePaths(this.portalConfig);
                return;
            }

            // Fallback to JSON config
            const jsonResponse = await fetch('./config/portal-config.json');
            if (jsonResponse.ok) {
                this.portalConfig = await jsonResponse.json();
                this._resolvePaths(this.portalConfig);
                return;
            }

            throw new Error('No portal config file found');
        } catch (error) {
            console.warn('Failed to load portal config:', error);
            this.portalConfig = null;
        }
    }

    _mergeConfigurations() {
        const merged = {};

        // Start with fallback config as base
        const baseConfig = this._getFallbackConfig();

        // Deep merge function
        const deepMerge = (target, source) => {
            const result = { ...target };

            for (const key in source) {
                if (source[key] && typeof source[key] === 'object' && !Array.isArray(source[key])) {
                    result[key] = deepMerge(result[key] || {}, source[key]);
                } else {
                    result[key] = source[key];
                }
            }

            return result;
        };

        // Merge in order of precedence: base -> oracle -> portal
        let mergedConfig = deepMerge(baseConfig, this.oracleConfig || {});
        mergedConfig = deepMerge(mergedConfig, this.portalConfig || {});

        return mergedConfig;
    }

    async _fetchConfig() {
        try {
            // Check if there's a script tag with portal-config id that specifies the config path
            const configScript = document.getElementById('portal-config');
            let configPath = './config/portal-links.yaml'; // default path

            if (configScript && configScript.src) {
                // Extract path from script src attribute
                configPath = configScript.src.replace(window.location.origin + window.location.pathname.replace(/\/[^\/]*$/, '/'), '');
            }

            // Try to load YAML config first
            const yamlResponse = await fetch(configPath);
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
                documentation: "documentation/static/",
                graphchain_explorer: "graphchain-explorer/",
                nexus_portal: "nexus-portal/",
                support_desk: "support-desk/src/"
                
            },
            external_services: {
                payment_oracle: "https://payments.knirv.network/add-funds",
                knirv_website: "https://knirv.com",
                testnet_access: "https://testnet.knirv.network"
            },
            documentation: {
                whitepapers: {
                    knirv_oracle: "documentation/static/whitepapers/KNIRVROOT_Whitepaper.md",
                    knirv_router: "documentation/static/whitepapers/KNIRV-ROUTER_Whitepaper.md",
                    knirvgraph: "documentation/static/whitepapers/KNIRV-GRAPH_Whitepaper.md",
                    knirvchain: "documentation/static/whitepapers/KNIRVCHAIN_Whitepaper.md",
                    knirv_nexus: "documentation/static/whitepapers/KNIRV-SERVER_Whitepaper.md",
                    knirv_cortex: "documentation/static/whitepapers/KNIRV-AGENTIFIER_Whitepaper.md",
                    knirv_wallet: "documentation/static/whitepapers/KNIRV-WALLET_Whitepaper.md",
                    knirv_oracle: "documentation/static/whitepapers/KNIRV-GATEWAY_Whitepaper.md",
                    knirv_shell: "documentation/static/whitepapers/KNIRV-SHELL_Whitepapers.md",
                    knirv_sdk: "documentation/static/whitepapers/KNIRV-SDK_Whitepaper.md",
                    knirv_testnet: "documentation/static/guides/TESTNET_DEPLOYMENT.md",
                    knirvana: "documentation/static/whitepapers/KNIRVANA_Whitepaper.md"
                }
            },
            footer: {
                legal: {
                    terms: "documentation/static/legal/TERMS_AND_CONDITIONS",
                    privacy: "documentation/static/legal/PRIVACY_POLICY",
                    contribution: "documentation/static/contributing/CONTRIBUTION_GUIDELINES"
                },
                social: {
                    github: "https://github.com/knirv-network",
                    discord: "https://discord.gg/knirv",
                    twitter: "https://twitter.com/knirvnetwork",
                    telegram: "https://t.me/knirvnetwork"
                },
                resources: {
                    documentation: "documentation/static/",
                    support: "support-desk/src/",
                    forum: "forum/src/",
                    blog: "https://knirv.com/blog"
                }
            },
            features: {
                authentication_enabled: true,
                payment_oracle_enabled: true,
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
                    url: "documentation/static/",
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

    // Get download configuration for a specific product
    getDownloadConfig(product) {
        return this.config?.downloads?.[product] || null;
    }

    // Get download link for a specific product and platform
    getDownloadLink(product, platform) {
        return this.config?.downloads?.[product]?.[platform] || '#';
    }

    // Get download requirements for a specific product and platform
    getDownloadRequirements(product, platform) {
        return this.config?.downloads?.[product]?.requirements?.[platform] || '';
    }

    // Get download instructions for SDK
    getDownloadInstructions(product, language) {
        return this.config?.downloads?.[product]?.instructions?.[language] || '';
    }

    // Get documentation link for SDK
    getDocumentationLink(product, language) {
        return this.config?.downloads?.[product]?.documentation?.[language] || '#';
    }

    // Get download note for a product
    getDownloadNote(product) {
        return this.config?.downloads?.[product]?.note || '';
    }

    // Get mobile note for products with mobile versions
    getMobileNote(product) {
        return this.config?.downloads?.[product]?.mobile_note || '';
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
        this._setupDownloadFunctions();
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

    // Update payment oracle links
    updatePaymentLinks() {
        const paymentLinks = document.querySelectorAll('[data-config="payment-oracle"]');
        paymentLinks.forEach(link => {
            link.href = this.getExternalService('payment_oracle');
        });
    }

    // Get configuration value by path (e.g., 'navigation.main_site')
    getConfigValue(path) {
        return path.split('.').reduce((obj, key) => obj?.[key], this.config);
    }

    _setupDownloadFunctions() {
        // Setup global download functions that use configuration
        window.downloadRouter = (platform) => this._handleDownload('knirvrouter', platform);
        window.downloadGame = (platform) => this._handleDownload('knirvana', platform);
        window.downloadBootnode = (type) => this._handleDownload('knirvoracle', type);
        window.downloadSDK = (language) => this._handleSDKDownload('knirvsdk', language);
    window.downloadWallet = (platform) => this._handleDownload('knirvwallet', platform);
    // Controller and CLI are new products mapped in portal-links.yaml
    window.downloadController = (platform) => this._handleDownload('knirvcontroller', platform);
    window.downloadCLI = (platform) => this._handleDownload('knirvcli', platform);
        window.rentDVE = (plan) => this._handleDVERental(plan);
        window.accessDVE = (method) => this._handleDVEAccess(method);
    }

    _handleDownload(product, platform) {
        const downloadLink = this.getDownloadLink(product, platform);
        const requirements = this.getDownloadRequirements(product, platform);
        const note = this.getDownloadNote(product);

        if (product === 'knirvwallet' && platform === 'mobile') {
            const mobileNote = this.getMobileNote(product);
            alert(mobileNote);
            return;
        }

        if (downloadLink === '#') {
            alert(`Download not available for ${platform}`);
            return;
        }

        let message = `Downloading ${product.toUpperCase()} for ${platform}...`;
        if (requirements) {
            message += `\n\nRequirements: ${requirements}`;
        }
        message += `\nDownload URL: ${downloadLink}`;
        if (note) {
            message += `\n\nNote: ${note}`;
        }

        alert(message);

        // Track download analytics
        if (typeof gtag !== 'undefined') {
            gtag('event', 'download', {
                'event_category': product.toUpperCase(),
                'event_label': platform
            });
        }
    }

    _handleSDKDownload(product, language) {
        const downloadCommand = this.getDownloadLink(product, language);
        const instructions = this.getDownloadInstructions(product, language);
        const docLink = this.getDocumentationLink(product, language);

        if (downloadCommand === '#') {
            alert(`SDK not available for ${language}`);
            return;
        }

        let message = `KNIRV SDK for ${language.charAt(0).toUpperCase() + language.slice(1)}:\n\n`;
        message += instructions || downloadCommand;
        if (docLink !== '#') {
            message += `\n\nDocumentation: ${docLink}`;
        }

        alert(message);

        // Track download analytics
        if (typeof gtag !== 'undefined') {
            gtag('event', 'sdk_download', {
                'event_category': 'KNIRV_SDK',
                'event_label': language
            });
        }
    }

    _handleDVERental(plan) {
        const plans = {
            'starter': {
                name: 'Starter DVE',
                price: '50 NRN/hour',
                specs: '2 vCPU, 8GB RAM, 100GB SSD'
            },
            'professional': {
                name: 'Professional DVE',
                price: '150 NRN/hour',
                specs: '8 vCPU, 32GB RAM, 500GB NVMe'
            },
            'enterprise': {
                name: 'Enterprise DVE',
                price: '500 NRN/hour',
                specs: '32 vCPU, 128GB RAM, 2TB NVMe'
            }
        };

        const selectedPlan = plans[plan];
        if (!selectedPlan) {
            alert('Invalid plan selected');
            return;
        }

        alert(`Renting ${selectedPlan.name}:\n\nPrice: ${selectedPlan.price}\nSpecs: ${selectedPlan.specs}\n\nRedirecting to NEXUS Portal for setup...\n\nNote: Minimum 1000 NRN balance required`);

        // Track rental analytics
        if (typeof gtag !== 'undefined') {
            gtag('event', 'dve_rental', {
                'event_category': 'KNIRVSERVER',
                'event_label': plan
            });
        }
    }

    _handleDVEAccess(method) {
        const accessMethods = {
            'ssh': 'SSH Access:\n\nssh user@your-dve.knirv.network\n\nRequires: SSH key pair, active DVE rental',
            'api': 'REST API Access:\n\nEndpoint: https://api.knirv.network/dve/v1/\nAuth: Bearer token required\n\nDocumentation: docs.knirv.network/api',
            'sdk': 'KNIRV SDK Integration:\n\nimport { DVEClient } from "@knirv/sdk"\nconst dve = new DVEClient(config)\n\nSee: knirvsdk.html for installation'
        };

        const accessInfo = accessMethods[method];
        if (!accessInfo) {
            alert('Invalid access method');
            return;
        }

        alert(`${accessInfo}\n\nFor active rentals, visit the NEXUS Portal for connection details.`);
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
