/**
 * KNIRV D-TEN Social Media Sharing Enhancement
 * Provides platform-specific sharing functionality with optimized content
 */

class KNIRVSocialSharing {
    constructor() {
        this.baseUrl = 'https://knirv.network/';
        this.defaultTitle = 'KNIRV D-TEN | Revolutionary Decentralized AI Network';
        this.defaultDescription = 'Experience the world\'s first self-improving AI network with 12 sovereign layers.';
        
        this.platformConfigs = {
            facebook: {
                shareUrl: 'https://www.facebook.com/sharer/sharer.php',
                params: {
                    u: this.baseUrl,
                    quote: '🚀 Discover KNIRV D-TEN: The world\'s first self-improving AI network! Transform AI failures into collective knowledge with 12 sovereign layers. #AI #Blockchain #Innovation'
                },
                windowFeatures: 'width=600,height=400,scrollbars=yes,resizable=yes'
            },
            
            twitter: {
                shareUrl: 'https://twitter.com/intent/tweet',
                params: {
                    url: this.baseUrl,
                    text: '🚀 Just discovered KNIRV D-TEN - the world\'s first self-improving AI network!\n\n🤖 12 sovereign layers\n⚡ Verifiable execution\n🔗 Collective intelligence\n💎 NRN token economics\n\nThe future of AI is here!',
                    hashtags: 'AI,Blockchain,DeFi,Web3,Innovation,KNIRV',
                    via: 'KNIRV_Network'
                },
                windowFeatures: 'width=600,height=400,scrollbars=yes,resizable=yes'
            },
            
            linkedin: {
                shareUrl: 'https://www.linkedin.com/sharing/share-offsite/',
                params: {
                    url: this.baseUrl,
                    title: 'KNIRV D-TEN: Revolutionary Decentralized AI Network',
                    summary: 'Discover how the Decentralized Trusted Execution Network (D-TEN) is transforming artificial intelligence through collective learning, verifiable execution, and self-healing systems.'
                },
                windowFeatures: 'width=600,height=400,scrollbars=yes,resizable=yes'
            },
            
            reddit: {
                shareUrl: 'https://reddit.com/submit',
                params: {
                    url: this.baseUrl,
                    title: 'KNIRV D-TEN: World\'s First Self-Improving AI Network with 12 Sovereign Layers'
                },
                windowFeatures: 'width=800,height=600,scrollbars=yes,resizable=yes'
            },
            
            telegram: {
                shareUrl: 'https://t.me/share/url',
                params: {
                    url: this.baseUrl,
                    text: '🚀 KNIRV D-TEN: Revolutionary Decentralized AI Network\n\n🤖 World\'s first self-improving AI network\n⚡ 12 sovereign layers working in harmony\n🔗 Transform AI failures into collective knowledge\n💎 NRN token economics\n\nJoin the future of AI!'
                },
                windowFeatures: 'width=600,height=400,scrollbars=yes,resizable=yes'
            },
            
            whatsapp: {
                shareUrl: 'https://wa.me/',
                params: {
                    text: '🚀 Check out KNIRV D-TEN - the world\'s first self-improving AI network!\n\n🤖 12 sovereign layers\n⚡ Verifiable execution\n🔗 Collective intelligence\n\n' + this.baseUrl
                },
                windowFeatures: 'width=600,height=400,scrollbars=yes,resizable=yes'
            },
            
            email: {
                shareUrl: 'mailto:',
                params: {
                    subject: 'KNIRV D-TEN: Revolutionary Decentralized AI Network',
                    body: 'I wanted to share this exciting project with you:\n\nKNIRV D-TEN is the world\'s first Decentralized Trusted Execution Network that transforms AI failures into collective knowledge through twelve sovereign layers.\n\nKey Features:\n• Self-improving AI systems\n• Verifiable execution environments\n• NRN token economics\n• Collective intelligence\n• 12 sovereign layers working in harmony\n\nLearn more: ' + this.baseUrl + '\n\nThis could revolutionize how we think about AI and blockchain integration!'
                }
            }
        };
        
        this.init();
    }
    
    init() {
        this.addShareButtons();
        this.addCopyLinkFunctionality();
        this.trackSocialSharing();
    }
    
    /**
     * Add dynamic share buttons to the page
     */
    addShareButtons() {
        const socialContainer = document.querySelector('.cpn-social .social');
        if (!socialContainer) return;
        
        // Add share button for each platform
        const shareButton = document.createElement('li');
        shareButton.className = 'animated';
        shareButton.setAttribute('data-animate', 'fadeInUp');
        shareButton.setAttribute('data-delay', '1.95');
        shareButton.innerHTML = `
            <a href="#" class="share-trigger" aria-label="Share KNIRV D-TEN">
                <em class="social-icon fas fa-share-alt"></em>
            </a>
        `;
        
        socialContainer.appendChild(shareButton);
        
        // Add share dropdown
        this.createShareDropdown(shareButton);
    }
    
    /**
     * Create share dropdown menu
     */
    createShareDropdown(container) {
        const dropdown = document.createElement('div');
        dropdown.className = 'share-dropdown';
        dropdown.innerHTML = `
            <div class="share-options">
                <button class="share-btn" data-platform="facebook">
                    <i class="fab fa-facebook-f"></i> Facebook
                </button>
                <button class="share-btn" data-platform="twitter">
                    <i class="fab fa-twitter"></i> Twitter
                </button>
                <button class="share-btn" data-platform="linkedin">
                    <i class="fab fa-linkedin-in"></i> LinkedIn
                </button>
                <button class="share-btn" data-platform="reddit">
                    <i class="fab fa-reddit"></i> Reddit
                </button>
                <button class="share-btn" data-platform="telegram">
                    <i class="fab fa-telegram"></i> Telegram
                </button>
                <button class="share-btn" data-platform="whatsapp">
                    <i class="fab fa-whatsapp"></i> WhatsApp
                </button>
                <button class="share-btn" data-platform="email">
                    <i class="fas fa-envelope"></i> Email
                </button>
                <button class="share-btn copy-link">
                    <i class="fas fa-link"></i> Copy Link
                </button>
            </div>
        `;
        
        container.appendChild(dropdown);
        
        // Add event listeners
        const trigger = container.querySelector('.share-trigger');
        const shareButtons = dropdown.querySelectorAll('.share-btn');
        
        trigger.addEventListener('click', (e) => {
            e.preventDefault();
            dropdown.classList.toggle('active');
        });
        
        shareButtons.forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                const platform = btn.dataset.platform;
                
                if (btn.classList.contains('copy-link')) {
                    this.copyToClipboard();
                } else if (platform) {
                    this.shareOnPlatform(platform);
                }
                
                dropdown.classList.remove('active');
            });
        });
        
        // Close dropdown when clicking outside
        document.addEventListener('click', (e) => {
            if (!container.contains(e.target)) {
                dropdown.classList.remove('active');
            }
        });
    }
    
    /**
     * Share on specific platform
     */
    shareOnPlatform(platform) {
        const config = this.platformConfigs[platform];
        if (!config) return;
        
        let shareUrl = config.shareUrl;
        
        if (platform === 'email') {
            // Handle email sharing
            const params = new URLSearchParams(config.params);
            window.location.href = shareUrl + '?' + params.toString();
        } else {
            // Handle social media sharing
            const params = new URLSearchParams(config.params);
            const fullUrl = shareUrl + '?' + params.toString();
            
            window.open(fullUrl, 'share', config.windowFeatures);
        }
        
        // Track sharing event
        this.trackShare(platform);
    }
    
    /**
     * Copy link to clipboard
     */
    async copyToClipboard() {
        try {
            await navigator.clipboard.writeText(this.baseUrl);
            this.showNotification('Link copied to clipboard!', 'success');
        } catch (err) {
            // Fallback for older browsers
            const textArea = document.createElement('textarea');
            textArea.value = this.baseUrl;
            document.body.appendChild(textArea);
            textArea.select();
            document.execCommand('copy');
            document.body.removeChild(textArea);
            this.showNotification('Link copied to clipboard!', 'success');
        }
        
        this.trackShare('copy-link');
    }
    
    /**
     * Add copy link functionality to existing social links
     */
    addCopyLinkFunctionality() {
        // Add right-click context menu for copying links
        const socialLinks = document.querySelectorAll('.social a[href^="http"]');
        socialLinks.forEach(link => {
            link.addEventListener('contextmenu', (e) => {
                e.preventDefault();
                this.copyToClipboard();
            });
        });
    }
    
    /**
     * Track social sharing events
     */
    trackSocialSharing() {
        // Track clicks on social media links
        const socialLinks = document.querySelectorAll('.social a[href^="http"]');
        socialLinks.forEach(link => {
            link.addEventListener('click', () => {
                const platform = this.getPlatformFromUrl(link.href);
                this.trackShare(platform);
            });
        });
    }
    
    /**
     * Get platform name from URL
     */
    getPlatformFromUrl(url) {
        if (url.includes('facebook.com')) return 'facebook';
        if (url.includes('twitter.com')) return 'twitter';
        if (url.includes('linkedin.com')) return 'linkedin';
        if (url.includes('youtube.com')) return 'youtube';
        if (url.includes('github.com')) return 'github';
        if (url.includes('discord.gg')) return 'discord';
        if (url.includes('medium.com')) return 'medium';
        return 'unknown';
    }
    
    /**
     * Track sharing events (integrate with analytics)
     */
    trackShare(platform) {
        // Google Analytics 4
        if (typeof gtag !== 'undefined') {
            gtag('event', 'share', {
                method: platform,
                content_type: 'website',
                item_id: 'knirv-d-ten-homepage'
            });
        }
        
        // Custom analytics
        console.log(`Shared on ${platform}`);
    }
    
    /**
     * Show notification to user
     */
    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.classList.add('show');
        }, 100);
        
        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => {
                document.body.removeChild(notification);
            }, 300);
        }, 3000);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new KNIRVSocialSharing();
});

// CSS for share dropdown and notifications
const style = document.createElement('style');
style.textContent = `
    .share-dropdown {
        position: absolute;
        top: 100%;
        left: 50%;
        transform: translateX(-50%);
        background: rgba(43, 86, 245, 0.95);
        border-radius: 8px;
        padding: 10px;
        min-width: 200px;
        opacity: 0;
        visibility: hidden;
        transition: all 0.3s ease;
        z-index: 1000;
        backdrop-filter: blur(10px);
        border: 1px solid rgba(255, 255, 255, 0.1);
    }
    
    .share-dropdown.active {
        opacity: 1;
        visibility: visible;
    }
    
    .share-options {
        display: flex;
        flex-direction: column;
        gap: 5px;
    }
    
    .share-btn {
        background: transparent;
        border: 1px solid rgba(255, 255, 255, 0.2);
        color: white;
        padding: 8px 12px;
        border-radius: 4px;
        cursor: pointer;
        transition: all 0.2s ease;
        text-align: left;
        font-size: 14px;
    }
    
    .share-btn:hover {
        background: rgba(255, 255, 255, 0.1);
        border-color: rgba(255, 255, 255, 0.3);
    }
    
    .share-btn i {
        margin-right: 8px;
        width: 16px;
    }
    
    .notification {
        position: fixed;
        top: 20px;
        right: 20px;
        background: #2b56f5;
        color: white;
        padding: 12px 20px;
        border-radius: 6px;
        z-index: 10000;
        transform: translateX(100%);
        transition: transform 0.3s ease;
    }
    
    .notification.show {
        transform: translateX(0);
    }
    
    .notification-success {
        background: #00c851;
    }
    
    .notification-error {
        background: #ff4444;
    }
`;

document.head.appendChild(style);
