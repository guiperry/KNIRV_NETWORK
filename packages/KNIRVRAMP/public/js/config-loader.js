// KNIRV Primary Website - Portal Configuration Loader (adapted)
class KNIRVConfigLoader {
    constructor() {
        this.config = null;
        this.isLoaded = false;
        this.loadPromise = null;
    }

    async loadConfig() {
        if (this.loadPromise) return this.loadPromise;
        this.loadPromise = this._fetchConfig();
        return this.loadPromise;
    }

    async _fetchConfig() {
        try {
            const configPath = '/config/portal-links.yaml';
            const yamlResponse = await fetch(configPath);
            if (yamlResponse.ok) {
                const yamlText = await yamlResponse.text();
                this.config = this._parseYAML(yamlText);
                this._resolvePaths(this.config);
                this.isLoaded = true;
                return this.config;
            }

            const jsonResponse = await fetch('/config/portal-links.json');
            if (jsonResponse.ok) {
                this.config = await jsonResponse.json();
                this._resolvePaths(this.config);
                this.isLoaded = true;
                return this.config;
            }

            throw new Error('No config file found');
        } catch (err) {
            console.warn('Failed to load portal config, using fallback', err);
            this.config = this._getFallbackConfig();
            this.isLoaded = true;
            return this.config;
        }
    }

    _resolvePaths(obj) {
        for (const key in obj) {
            if (typeof obj[key] === 'object' && obj[key] !== null) this._resolvePaths(obj[key]);
            else if (typeof obj[key] === 'string') {
                if (obj[key].startsWith('../')) obj[key] = obj[key].replace('../', '');
            }
        }
    }

    // Minimal YAML parser for our simple config
    _parseYAML(yamlText) {
        const lines = yamlText.split('\n');
        const result = {};
        const stack = [result];
        let currentIndent = 0;

        for (let line of lines) {
            if (line.trim().startsWith('#') || line.trim() === '') continue;
            const indent = line.length - line.trimStart().length;
            const trimmed = line.trim();
            if (indent < currentIndent) {
                const levels = (currentIndent - indent) / 2;
                for (let i = 0; i < levels; i++) stack.pop();
            }
            currentIndent = indent;
            if (trimmed.includes(':')) {
                const [key, ...valueParts] = trimmed.split(':');
                const value = valueParts.join(':').trim();
                const cur = stack[stack.length - 1];
                if (value === '' || value === '{}' || value === '[]') {
                    cur[key.trim()] = {};
                    stack.push(cur[key.trim()]);
                } else {
                    let parsed = value;
                    if ((parsed.startsWith('"') && parsed.endsWith('"')) || (parsed.startsWith("'") && parsed.endsWith("'"))) {
                        parsed = parsed.slice(1, -1);
                    }
                    if (parsed === 'true') parsed = true;
                    else if (parsed === 'false') parsed = false;
                    else if (!isNaN(parsed) && parsed !== '') parsed = Number(parsed);
                    cur[key.trim()] = parsed;
                }
            }
        }
        return result;
    }

    _getFallbackConfig() {
        return {
            navigation: { home: '/', documentation: '/documentation', support: '/support' },
            footer: { legal: { terms: '/terms-of-service', privacy: '/privacy-policy' } }
        };
    }

    getNavigationLink(key) { return this.config?.navigation?.[key] || '#'; }
    getFooterLink(category, key) { return this.config?.footer?.[category]?.[key] || '#'; }
    isFeatureEnabled(feature) { return this.config?.features?.[feature] || false; }
    getIframeConfig(key) { return this.config?.iframes?.[key] || null; }

    // Apply configuration to page elements using data attributes
    async applyConfiguration() {
        if (!this.isLoaded) await this.loadConfig();
        this._updateNavigationLinks();
        this._updateFooterLinks();
        this._updateFeatureVisibility();
        this._updateIframeConfigs();
    }

    _updateNavigationLinks() {
        document.querySelectorAll('[data-config="main-site"]').forEach(link => link.href = this.getNavigationLink('home'));
        document.querySelectorAll('[data-config="documentation"]').forEach(link => link.href = this.getNavigationLink('documentation'));
        document.querySelectorAll('[data-config-nav]').forEach(el => {
            const key = el.getAttribute('data-config-nav');
            el.href = this.getNavigationLink(key) || el.href;
        });
    }

    _updateFooterLinks() {
        document.querySelectorAll('[data-config-footer]').forEach(link => {
            const [category, key] = link.getAttribute('data-config-footer').split('.');
            link.href = this.getFooterLink(category, key) || link.href;
        });
    }

    _updateFeatureVisibility() {
        document.querySelectorAll('[data-feature]').forEach(el => {
            const feature = el.getAttribute('data-feature');
            if (!this.isFeatureEnabled(feature)) el.style.display = 'none';
        });
    }

    _updateIframeConfigs() {
        document.querySelectorAll('[data-config-iframe]').forEach(iframe => {
            const key = iframe.getAttribute('data-config-iframe');
            const cfg = this.getIframeConfig(key);
            if (cfg) {
                iframe.src = cfg.url;
                iframe.title = cfg.title;
                if (cfg.height) iframe.style.height = cfg.height;
            }
        });
    }
}

// Expose a singleton loader
window.KNIRVConfigLoader = new KNIRVConfigLoader();

// Auto-apply for static pages
document.addEventListener('DOMContentLoaded', async () => {
    try {
        await window.KNIRVConfigLoader.applyConfiguration();
    } catch (err) {
        console.warn('Portal config apply failed', err);
    }
});
