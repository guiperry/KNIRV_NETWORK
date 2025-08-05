/**
 * KNIRV Developer Portal JavaScript
 * Main functionality for the developer portal interface
 */

class KNIRVPortal {
    constructor() {
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadUserData();
        this.checkNetworkStatus();
        this.initializeComponents();
    }

    setupEventListeners() {
        // Navigation handling
        document.addEventListener('DOMContentLoaded', () => {
            this.highlightActiveNavItem();
        });

        // Form submissions
        document.addEventListener('submit', (e) => {
            if (e.target.classList.contains('portal-form')) {
                this.handleFormSubmission(e);
            }
        });

        // Modal handling
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal-trigger')) {
                this.openModal(e.target.dataset.modal);
            }
            if (e.target.classList.contains('modal-close') || e.target.classList.contains('modal-overlay')) {
                this.closeModal();
            }
        });

        // Notification close buttons
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('notification-close')) {
                this.closeNotification(e.target.closest('.notification'));
            }
        });
    }

    highlightActiveNavItem() {
        const currentPage = window.location.pathname.split('/').pop() || 'index.html';
        const navItems = document.querySelectorAll('.nav-item');
        
        navItems.forEach(item => {
            item.classList.remove('active');
            if (item.getAttribute('href') === currentPage) {
                item.classList.add('active');
            }
        });
    }

    async loadUserData() {
        try {
            // Simulate API call to load user data
            const userData = await this.fetchUserData();
            this.updateDashboardStats(userData);
        } catch (error) {
            console.error('Failed to load user data:', error);
            this.showNotification('Failed to load user data', 'error');
        }
    }

    async fetchUserData() {
        // Simulate API call - replace with actual API endpoint
        return new Promise((resolve) => {
            setTimeout(() => {
                resolve({
                    registeredAgents: 0,
                    publishedSkills: 0,
                    nrnBalance: 0.00,
                    activeUDCs: 0,
                    recentActivity: []
                });
            }, 1000);
        });
    }

    updateDashboardStats(userData) {
        const statsElements = {
            registeredAgents: document.querySelector('[data-stat="registered-agents"]'),
            publishedSkills: document.querySelector('[data-stat="published-skills"]'),
            nrnBalance: document.querySelector('[data-stat="nrn-balance"]'),
            activeUDCs: document.querySelector('[data-stat="active-udcs"]')
        };

        Object.keys(statsElements).forEach(key => {
            const element = statsElements[key];
            if (element) {
                element.textContent = userData[key] || 0;
            }
        });
    }

    async checkNetworkStatus() {
        try {
            const status = await this.fetchNetworkStatus();
            this.updateNetworkStatus(status);
        } catch (error) {
            console.error('Failed to check network status:', error);
            this.updateNetworkStatus({ online: false });
        }
    }

    async fetchNetworkStatus() {
        // Simulate network status check
        return new Promise((resolve) => {
            setTimeout(() => {
                resolve({
                    online: true,
                    uptime: 99.8,
                    totalAgents: 1247,
                    skillsInRegistry: 3891,
                    activeErrorNodes: 156,
                    nrnSupply: '10.2M'
                });
            }, 500);
        });
    }

    updateNetworkStatus(status) {
        const statusIndicator = document.querySelector('.network-status .status-dot');
        if (statusIndicator) {
            statusIndicator.className = `status-dot ${status.online ? 'status-online' : 'status-offline'}`;
        }

        // Update network stats if elements exist
        const networkStats = {
            'total-agents': status.totalAgents,
            'skills-registry': status.skillsInRegistry,
            'active-errornodes': status.activeErrorNodes,
            'nrn-supply': status.nrnSupply
        };

        Object.keys(networkStats).forEach(key => {
            const element = document.querySelector(`[data-network-stat="${key}"]`);
            if (element) {
                element.textContent = networkStats[key] || 'N/A';
            }
        });
    }

    initializeComponents() {
        // Initialize tooltips
        this.initTooltips();
        
        // Initialize copy-to-clipboard functionality
        this.initCopyButtons();
        
        // Initialize search functionality
        this.initSearch();
    }

    initTooltips() {
        const tooltipElements = document.querySelectorAll('[data-tooltip]');
        tooltipElements.forEach(element => {
            element.addEventListener('mouseenter', (e) => {
                this.showTooltip(e.target, e.target.dataset.tooltip);
            });
            element.addEventListener('mouseleave', () => {
                this.hideTooltip();
            });
        });
    }

    initCopyButtons() {
        const copyButtons = document.querySelectorAll('.copy-button');
        copyButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                const textToCopy = e.target.dataset.copy || e.target.previousElementSibling.textContent;
                this.copyToClipboard(textToCopy);
            });
        });
    }

    initSearch() {
        const searchInputs = document.querySelectorAll('.search-input');
        searchInputs.forEach(input => {
            input.addEventListener('input', (e) => {
                this.handleSearch(e.target.value, e.target.dataset.searchTarget);
            });
        });
    }

    handleFormSubmission(e) {
        e.preventDefault();
        const form = e.target;
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());
        
        this.showNotification('Processing request...', 'info');
        
        // Simulate form submission
        setTimeout(() => {
            this.showNotification('Request completed successfully!', 'success');
            form.reset();
        }, 2000);
    }

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type} fade-in`;
        notification.innerHTML = `
            <div class="flex items-center justify-between">
                <span>${message}</span>
                <button class="notification-close ml-4 text-white hover:text-gray-300">
                    <i class="fas fa-times"></i>
                </button>
            </div>
        `;
        
        document.body.appendChild(notification);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            this.closeNotification(notification);
        }, 5000);
    }

    closeNotification(notification) {
        if (notification) {
            notification.style.opacity = '0';
            setTimeout(() => {
                notification.remove();
            }, 300);
        }
    }

    openModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.remove('hidden');
            document.body.style.overflow = 'hidden';
        }
    }

    closeModal() {
        const modals = document.querySelectorAll('.modal-overlay');
        modals.forEach(modal => {
            modal.classList.add('hidden');
        });
        document.body.style.overflow = 'auto';
    }

    async copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
            this.showNotification('Copied to clipboard!', 'success');
        } catch (error) {
            console.error('Failed to copy to clipboard:', error);
            this.showNotification('Failed to copy to clipboard', 'error');
        }
    }

    handleSearch(query, target) {
        const searchableElements = document.querySelectorAll(`[data-searchable="${target}"]`);
        const lowerQuery = query.toLowerCase();
        
        searchableElements.forEach(element => {
            const text = element.textContent.toLowerCase();
            const shouldShow = text.includes(lowerQuery) || query === '';
            element.style.display = shouldShow ? '' : 'none';
        });
    }

    showTooltip(element, text) {
        const tooltip = document.createElement('div');
        tooltip.className = 'tooltip-text visible opacity-100';
        tooltip.textContent = text;
        element.appendChild(tooltip);
    }

    hideTooltip() {
        const tooltips = document.querySelectorAll('.tooltip-text');
        tooltips.forEach(tooltip => tooltip.remove());
    }

    // Utility methods
    formatNumber(num) {
        if (num >= 1000000) {
            return (num / 1000000).toFixed(1) + 'M';
        } else if (num >= 1000) {
            return (num / 1000).toFixed(1) + 'K';
        }
        return num.toString();
    }

    formatCurrency(amount, currency = 'NRN') {
        return `${amount.toFixed(2)} ${currency}`;
    }

    formatDate(date) {
        return new Date(date).toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }
}

// Initialize the portal when the DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.knirvPortal = new KNIRVPortal();
});

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = KNIRVPortal;
}
