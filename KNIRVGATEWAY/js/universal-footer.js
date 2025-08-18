// Universal Footer Component for KNIRV Gateway
class KNIRVUniversalFooter {
    constructor() {
        this.config = null;
        this.currentYear = new Date().getFullYear();
    }

    async init() {
        // Load configuration
        if (window.knirvConfig && window.knirvConfig.isLoaded) {
            this.config = window.knirvConfig.config;
        } else {
            // Wait for config to load
            await this.waitForConfig();
        }
        
        this.renderFooter();
    }

    async waitForConfig() {
        return new Promise((resolve) => {
            const checkConfig = () => {
                if (window.knirvConfig && window.knirvConfig.isLoaded) {
                    this.config = window.knirvConfig.config;
                    resolve();
                } else {
                    setTimeout(checkConfig, 100);
                }
            };
            checkConfig();
        });
    }

    renderFooter() {
        const footerHTML = this.generateFooterHTML();
        
        // Find existing footer or create one
        let footerElement = document.querySelector('footer.knirv-footer');
        if (!footerElement) {
            footerElement = document.createElement('footer');
            footerElement.className = 'knirv-footer';
            document.body.appendChild(footerElement);
        }
        
        footerElement.innerHTML = footerHTML;
        this.addFooterStyles();
    }

    generateFooterHTML() {
        const footer = this.config?.footer || {};
        
        return `
            <div class="footer-container">
                <div class="footer-content">
                    <div class="footer-section">
                        <h4>KNIRV Network</h4>
                        <ul>
                            <li><a href="${this.getLink('navigation', 'main_site')}" target="_blank">Main Site</a></li>
                            <li><a href="${this.getLink('navigation', 'documentation')}" target="_blank">Documentation</a></li>
                            <li><a href="${this.getLink('footer.resources', 'support')}">Support</a></li>
                            <li><a href="${this.getLink('navigation', 'nexus_portal')}">KNIRV-NEXUS</a></li>
                        </ul>
                    </div>

                    <div class="footer-section">
                        <h4>Developer Tools</h4>
                        <ul>
                            <li><a href="${this.getLink('navigation', 'graphchain_explorer')}">Graphchain Explorer</a></li>
                            <li><a href="${this.getLink('navigation', 'nanda_ans')}">NANDA+ANS</a></li>
                            <li><a href="../agent-developer-portal/">Developer Portal</a></li>
                            <li><a href="../agentify/">Agentify</a></li>
                        </ul>
                    </div>

                    <div class="footer-section">
                        <h4>Community</h4>
                        <ul>
                            <li><a href="${this.getLink('footer.social', 'discord')}" target="_blank">Discord</a></li>
                            <li><a href="${this.getLink('footer.social', 'telegram')}" target="_blank">Telegram</a></li>
                            <li><a href="${this.getLink('footer.social', 'twitter')}" target="_blank">Twitter</a></li>
                            <li><a href="${this.getLink('footer.resources', 'forum')}">Forum</a></li>
                        </ul>
                    </div>

                    <div class="footer-section">
                        <h4>Resources</h4>
                        <ul>
                            <li><a href="${this.getLink('footer.resources', 'blog')}" target="_blank">Blog</a></li>
                            <li><a href="${this.getLink('footer.social', 'github')}" target="_blank">GitHub</a></li>
                            <li><a href="${this.getLink('footer.legal', 'terms')}">Terms of Service</a></li>
                            <li><a href="${this.getLink('footer.legal', 'privacy')}">Privacy Policy</a></li>
                        </ul>
                    </div>
                </div>

                <div class="footer-bottom">
                    <div class="footer-bottom-content">
                        <div class="footer-logo">
                            <span class="footer-brand">KNIRV</span>
                            <span class="footer-tagline">Decentralized Trusted Execution Network</span>
                        </div>
                        <div class="footer-legal">
                            <p>&copy; ${this.currentYear} KNIRV Network. All rights reserved.</p>
                            <div class="footer-legal-links">
                                <a href="${this.getLink('footer.legal', 'terms')}">Terms</a>
                                <a href="${this.getLink('footer.legal', 'privacy')}">Privacy</a>
                                <a href="${this.getLink('footer.legal', 'contribution')}">Contributing</a>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    getLink(path, fallback = '#') {
        const keys = path.split('.');
        let value = this.config;
        
        for (const key of keys) {
            value = value?.[key];
            if (!value) break;
        }
        
        return value || fallback;
    }

    addFooterStyles() {
        // Check if styles already exist
        if (document.getElementById('knirv-footer-styles')) {
            return;
        }

        const styles = document.createElement('style');
        styles.id = 'knirv-footer-styles';
        styles.textContent = `
            .knirv-footer {
                background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
                color: #ffffff;
                margin-top: auto;
                border-top: 1px solid rgba(255, 255, 255, 0.1);
            }

            .footer-container {
                max-width: 1200px;
                margin: 0 auto;
                padding: 0 20px;
            }

            .footer-content {
                display: grid;
                grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
                gap: 40px;
                padding: 60px 0 40px;
            }

            .footer-section h4 {
                color: #00c0fa;
                font-size: 1.1rem;
                font-weight: 600;
                margin-bottom: 20px;
                text-transform: uppercase;
                letter-spacing: 0.5px;
            }

            .footer-section ul {
                list-style: none;
                padding: 0;
                margin: 0;
            }

            .footer-section li {
                margin-bottom: 12px;
            }

            .footer-section a {
                color: rgba(255, 255, 255, 0.7);
                text-decoration: none;
                font-size: 0.9rem;
                transition: all 0.3s ease;
                display: inline-block;
            }

            .footer-section a:hover {
                color: #00c0fa;
                transform: translateX(5px);
            }

            .footer-bottom {
                border-top: 1px solid rgba(255, 255, 255, 0.1);
                padding: 30px 0;
            }

            .footer-bottom-content {
                display: flex;
                justify-content: space-between;
                align-items: center;
                flex-wrap: wrap;
                gap: 20px;
            }

            .footer-logo {
                display: flex;
                flex-direction: column;
                gap: 5px;
            }

            .footer-brand {
                font-size: 1.5rem;
                font-weight: 700;
                color: #00c0fa;
                text-transform: uppercase;
                letter-spacing: 2px;
            }

            .footer-tagline {
                font-size: 0.8rem;
                color: rgba(255, 255, 255, 0.6);
                text-transform: uppercase;
                letter-spacing: 1px;
            }

            .footer-legal {
                display: flex;
                flex-direction: column;
                align-items: flex-end;
                gap: 10px;
            }

            .footer-legal p {
                margin: 0;
                font-size: 0.9rem;
                color: rgba(255, 255, 255, 0.6);
            }

            .footer-legal-links {
                display: flex;
                gap: 20px;
            }

            .footer-legal-links a {
                color: rgba(255, 255, 255, 0.6);
                text-decoration: none;
                font-size: 0.8rem;
                transition: color 0.3s ease;
            }

            .footer-legal-links a:hover {
                color: #00c0fa;
            }

            /* Responsive Design */
            @media (max-width: 768px) {
                .footer-content {
                    grid-template-columns: repeat(2, 1fr);
                    gap: 30px;
                    padding: 40px 0 30px;
                }

                .footer-bottom-content {
                    flex-direction: column;
                    text-align: center;
                }

                .footer-legal {
                    align-items: center;
                }

                .footer-legal-links {
                    justify-content: center;
                }
            }

            @media (max-width: 480px) {
                .footer-content {
                    grid-template-columns: 1fr;
                    gap: 25px;
                    padding: 30px 0 20px;
                }

                .footer-section {
                    text-align: center;
                }
            }

            /* Ensure footer sticks to bottom */
            body {
                display: flex;
                flex-direction: column;
                min-height: 100vh;
            }

            .dashboard-container {
                flex: 1;
                display: flex;
                flex-direction: column;
            }

            .main-content {
                flex: 1;
            }
        `;
        
        document.head.appendChild(styles);
    }

    // Method to update footer links dynamically
    updateFooterLinks(newConfig) {
        this.config = newConfig;
        this.renderFooter();
    }

    // Method to add custom footer section
    addCustomSection(title, links) {
        const customSection = `
            <div class="footer-section">
                <h4>${title}</h4>
                <ul>
                    ${links.map(link => `<li><a href="${link.url}" ${link.external ? 'target="_blank"' : ''}>${link.text}</a></li>`).join('')}
                </ul>
            </div>
        `;
        
        const footerContent = document.querySelector('.footer-content');
        if (footerContent) {
            footerContent.insertAdjacentHTML('beforeend', customSection);
        }
    }
}

// Initialize footer when DOM is ready
document.addEventListener('DOMContentLoaded', async () => {
    // Wait a bit for other scripts to load
    setTimeout(async () => {
        const footer = new KNIRVUniversalFooter();
        await footer.init();
        
        // Make footer globally available
        window.knirvFooter = footer;
    }, 500);
});

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = KNIRVUniversalFooter;
}
