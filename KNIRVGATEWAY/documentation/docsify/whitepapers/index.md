# KNIRV Network Whitepapers

<style>
.whitepaper-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem 1rem;
}

.whitepaper-header {
    text-align: center;
    margin-bottom: 3rem;
}

.whitepaper-header h1 {
    font-size: 3rem;
    font-weight: 800;
    color: #ffffff;
    margin-bottom: 1rem;
    background: linear-gradient(135deg, #4a9eff, #6fb5ff);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
}

.whitepaper-header p {
    font-size: 1.25rem;
    color: #94a3b8;
    max-width: 600px;
    margin: 0 auto;
}

.whitepaper-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 1.5rem;
    margin-bottom: 3rem;
}

.whitepaper-card {
    background: rgba(26, 32, 44, 0.8);
    border: 1px solid #2d3748;
    border-radius: 12px;
    padding: 1.5rem;
    transition: all 0.3s ease;
    cursor: pointer;
    backdrop-filter: blur(10px);
    position: relative;
    overflow: hidden;
}

.whitepaper-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 20px 40px rgba(74, 158, 255, 0.1);
    border-color: #4a9eff;
}

.whitepaper-card::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 3px;
    background: linear-gradient(90deg, #4a9eff, #6fb5ff);
    opacity: 0;
    transition: opacity 0.3s ease;
}

.whitepaper-card:hover::before {
    opacity: 1;
}

.whitepaper-icon {
    font-size: 2.5rem;
    margin-bottom: 1rem;
    display: block;
}

.whitepaper-title {
    font-size: 1.25rem;
    font-weight: 700;
    color: #ffffff;
    margin-bottom: 0.75rem;
    line-height: 1.3;
}

.whitepaper-description {
    font-size: 0.9rem;
    color: #94a3b8;
    line-height: 1.5;
    margin-bottom: 1rem;
}

.whitepaper-link {
    display: inline-flex;
    align-items: center;
    color: #4a9eff;
    font-size: 0.875rem;
    font-weight: 600;
    text-decoration: none;
    transition: color 0.3s ease;
}

.whitepaper-link:hover {
    color: #6fb5ff;
}

.whitepaper-link::after {
    content: '→';
    margin-left: 0.5rem;
    transition: transform 0.3s ease;
}

.whitepaper-link:hover::after {
    transform: translateX(4px);
}

.interconnection-section {
    text-align: center;
    margin-top: 4rem;
    padding: 2rem;
    background: rgba(26, 32, 44, 0.4);
    border-radius: 12px;
    border: 1px solid #2d3748;
}

.interconnection-section h3 {
    font-size: 2rem;
    font-weight: 700;
    color: #ffffff;
    margin-bottom: 1rem;
}

.interconnection-section p {
    color: #94a3b8;
    max-width: 800px;
    margin: 0 auto;
    line-height: 1.6;
}

@media (max-width: 768px) {
    .whitepaper-grid {
        grid-template-columns: 1fr;
    }

    .whitepaper-header h1 {
        font-size: 2rem;
    }

    .whitepaper-header p {
        font-size: 1rem;
    }
}
</style>

<div class="whitepaper-container">
    <div class="whitepaper-header">
        <h1>KNIRV Network Whitepapers</h1>
        <p>Comprehensive technical documentation for the KNIRV Decentralized Trusted Execution Network (D-TEN) ecosystem components.</p>
    </div>

    <div class="whitepaper-grid">
        <!-- KNIRV-ROOT Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRVROOT_Whitepaper')">
            <div class="whitepaper-icon">🔗</div>
            <h4 class="whitepaper-title">KNIRV-ROOT</h4>
            <p class="whitepaper-description">
                The sovereign GoLang-based blockchain, acting as the canonical NRN token ledger and network oracle. It orchestrates the economic loop and state synchronization.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRVROOT_Whitepaper')">Read Whitepaper</a>
        </div>

        <!-- KNIRV-ROUTER Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRV-ROUTER_Whitepaper')">
            <div class="whitepaper-icon">📡</div>
            <h4 class="whitepaper-title">KNIRV-ROUTER</h4>
            <p class="whitepaper-description">
                The network integrity layer, producing NRNs via "Proof-of-Connectivity" and embedding URI path certificates for secure, verifiable routes.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRV-ROUTER_Whitepaper')">Read Whitepaper</a>
        </div>

        <!-- KNIRVGRAPH Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRVGRAPH_Whitepaper')">
            <div class="whitepaper-icon">🧠</div>
            <h4 class="whitepaper-title">KNIRVGRAPH</h4>
            <p class="whitepaper-description">
                The GoLang-based Graphchain, serving as the decentralized knowledge fabric for ErrorNodes and SkillNodes, where AI learns from its mistakes.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRVGRAPH_Whitepaper')">Read Whitepaper</a>
        </div>

        <!-- KNIRVCHAIN Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRVCHAIN_Whitepaper')">
            <div class="whitepaper-icon">📚</div>
            <h4 class="whitepaper-title">KNIRVCHAIN</h4>
            <p class="whitepaper-description">
                The sovereign Rust-based blockchain, hosting the canonical CodeT5 Base LLM and SkillRegistry for trusted agent capabilities.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRVCHAIN_Whitepaper')">Read Whitepaper</a>
        </div>

        <!-- KNIRV-NEXUS DVE Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRVNEXUS_Whitepaper')">
            <div class="whitepaper-icon">🔐</div>
            <h4 class="whitepaper-title">KNIRV-NEXUS DVE</h4>
            <p class="whitepaper-description">
                The network of Decentralized Validation Environments (CLEAN), providing verifiable execution and cryptographic proofs for all Skill resolutions.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRVNEXUS_Whitepaper')">Read Whitepaper</a>
        </div>

        <!-- KNIRV-AGENTIFIER Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRV-AGENTIFIER_Whitepaper')">
            <div class="whitepaper-icon">🤖</div>
            <h4 class="whitepaper-title">KNIRV-AGENTIFIER</h4>
            <p class="whitepaper-description">
                A mobile-native adapter that transforms existing AI assistants into autonomous, self-improving agents powered by Rust WASM.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRV-AGENTIFIER_Whitepaper')">Read Whitepaper</a>
        </div>

        <!-- KNIRV-WALLET Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRV-WALLET_Whitepaper')">
            <div class="whitepaper-icon">👛</div>
            <h4 class="whitepaper-title">KNIRV-WALLET</h4>
            <p class="whitepaper-description">
                The user-friendly gateway, leveraging XION's Meta Accounts for seamless interaction, primarily through the KNIRV-AGENTIFIER.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRV-WALLET_Whitepaper')">Read Whitepaper</a>
        </div>

        <!-- KNIRV-GATEWAY Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRV-GATEWAY_Whitepaper')">
            <div class="whitepaper-icon">🌐</div>
            <h4 class="whitepaper-title">KNIRV-GATEWAY</h4>
            <p class="whitepaper-description">
                The unified web portal and serverless API gateway with real-time capabilities, serving as the primary entry point for all ecosystem interactions.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRV-GATEWAY_Whitepaper')">Read Whitepaper</a>
        </div>

        <!-- KNIRV-SHELL Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRV-SHELL_Whitepapers')">
            <div class="whitepaper-icon">💻</div>
            <h4 class="whitepaper-title">KNIRV-SHELL</h4>
            <p class="whitepaper-description">
                The AI-powered, comprehensive GoLang-based command-line interface for unified developer and power user access to the entire D-TEN.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRV-SHELL_Whitepapers')">Read Whitepaper</a>
        </div>

        <!-- KNIRV-SDK Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRV-SDK_Whitepaper')">
            <div class="whitepaper-icon">🛠️</div>
            <h4 class="whitepaper-title">KNIRV-SDK</h4>
            <p class="whitepaper-description">
                A comprehensive multi-language development kit offering high-level abstractions, Gateway integration, and enhanced knirv:// URI resolution.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRV-SDK_Whitepaper')">Read Whitepaper</a>
        </div>

        <!-- KNIRV-D-TEN Card -->
        <div class="whitepaper-card" onclick="openWhitepaper('KNIRV-D-TEN_Whitepaper')">
            <div class="whitepaper-icon">🎯</div>
            <h4 class="whitepaper-title">KNIRV-D-TEN</h4>
            <p class="whitepaper-description">
                The comprehensive framework overview - A Unified Framework for Compounding AI Intelligence, Verifiable Execution, and Self-Healing Systems.
            </p>
            <a href="#" class="whitepaper-link" onclick="event.stopPropagation(); openWhitepaper('KNIRV-D-TEN_Whitepaper')">Read Whitepaper</a>
        </div>
    </div>

    <!-- Interconnection Section -->
    <div class="interconnection-section">
        <h3>How It All Connects</h3>
        <p>
            The D-TEN's sovereign layers are seamlessly interconnected via Inter-Blockchain Communication (IBC) and the <strong>KNIRV-GATEWAY</strong> unified API gateway. This architecture orchestrates a self-sustaining economic loop powered by the NRN token, incentivizing collective problem-solving and fostering a compounding global intelligence.
        </p>
    </div>
</div>

<script>
function openWhitepaper(whitepaperName) {
    // Navigate to the specific whitepaper within the docsify documentation
    const whitepaperUrl = `#/whitepapers/${whitepaperName}`;

    // Use docsify's router if available, otherwise fallback to location change
    if (window.Docsify && window.Docsify.router) {
        window.Docsify.router.history.go(whitepaperUrl);
    } else if (window.$docsify && window.$docsify.router) {
        window.$docsify.router.history.go(whitepaperUrl);
    } else {
        // Direct hash navigation
        window.location.hash = whitepaperUrl;
        // Force page reload if needed
        if (window.location.hash !== whitepaperUrl) {
            window.location.href = window.location.pathname + window.location.search + whitepaperUrl;
        }
    }
}

// Enhanced interactive functionality
function initWhitepaperCards() {
    const cards = document.querySelectorAll('.whitepaper-card');
    cards.forEach(card => {
        // Enhanced hover effects
        card.addEventListener('mouseenter', function() {
            this.style.transform = 'translateY(-4px) scale(1.02)';
            this.style.transition = 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)';
        });

        card.addEventListener('mouseleave', function() {
            this.style.transform = 'translateY(0) scale(1)';
        });

        // Add keyboard navigation support
        card.setAttribute('tabindex', '0');
        card.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                this.click();
            }
        });

        // Add focus styles
        card.addEventListener('focus', function() {
            this.style.outline = '2px solid #4a9eff';
            this.style.outlineOffset = '2px';
        });

        card.addEventListener('blur', function() {
            this.style.outline = 'none';
        });
    });
}

// Initialize when DOM is ready or when docsify loads the page
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initWhitepaperCards);
} else {
    initWhitepaperCards();
}

// Also initialize when docsify navigates to this page
if (window.$docsify) {
    window.$docsify.plugins = window.$docsify.plugins || [];
    window.$docsify.plugins.push(function(hook) {
        hook.doneEach(function() {
            // Re-initialize cards when page content changes
            setTimeout(initWhitepaperCards, 100);
        });
    });
}
</script>

---

© 2025 KNIRV Network
