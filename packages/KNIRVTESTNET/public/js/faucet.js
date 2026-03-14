/**
 * KNIRV Testnet Faucet Frontend JavaScript
 * 
 * Handles form submission, real-time status updates, request history,
 * and user interface interactions for the NRV faucet.
 */

class FaucetApp {
    constructor() {
        this.apiBase = '/api/faucet';
        this.currentAddress = '';
        this.statusUpdateInterval = null;
        this.economicStatusInterval = null;
        
        this.init();
    }

    /**
     * Initialize the application
     */
    init() {
        this.bindEvents();
        this.loadFaucetStatus();
        this.loadEconomicStatus();
        this.startStatusUpdates();
        
        console.log('KNIRV Faucet App initialized');
    }

    /**
     * Bind event listeners
     */
    bindEvents() {
        // Form submission
        const form = document.getElementById('faucetForm');
        if (form) {
            form.addEventListener('submit', (e) => this.handleFormSubmit(e));
        }

        // Amount selection
        const amountSelect = document.getElementById('tokenAmount');
        if (amountSelect) {
            amountSelect.addEventListener('change', (e) => this.handleAmountChange(e));
        }

        // Address validation
        const addressInput = document.getElementById('walletAddress');
        if (addressInput) {
            addressInput.addEventListener('input', (e) => this.validateAddress(e.target.value));
            addressInput.addEventListener('blur', (e) => this.loadAddressHistory(e.target.value));
        }

        // Custom amount validation
        const customAmountInput = document.getElementById('customAmount');
        if (customAmountInput) {
            customAmountInput.addEventListener('input', (e) => this.validateCustomAmount(e.target.value));
        }
    }

    /**
     * Handle form submission
     */
    async handleFormSubmit(event) {
        event.preventDefault();
        
        const form = event.target;
        const formData = new FormData(form);
        
        // Get form values
        const address = formData.get('address').trim();
        const selectedAmount = formData.get('amount');
        const customAmount = formData.get('customAmount');
        const reason = formData.get('reason').trim();
        
        // Determine final amount
        let amount;
        if (selectedAmount === 'custom') {
            amount = parseInt(customAmount);
            if (!amount || amount < 100 || amount > 5000) {
                this.showAlert('Please enter a valid custom amount (100-5000 NRV)', 'error');
                return;
            }
        } else {
            amount = parseInt(selectedAmount);
        }

        // Validate address
        if (!this.isValidAddress(address)) {
            this.showAlert('Please enter a valid KNIRV address', 'error');
            return;
        }

        // Show loading state
        this.setLoadingState(true);
        this.clearMessages();

        try {
            const response = await fetch(`${this.apiBase}/request`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    address: address,
                    amount: amount,
                    reason: reason
                })
            });

            const result = await response.json();

            if (result.success) {
                this.showAlert(
                    `✅ Success! ${amount} NRV tokens sent to ${address}. Transaction: ${result.tx_hash}`,
                    'success'
                );
                
                // Reset form
                form.reset();
                document.getElementById('tokenAmount').value = '1000';
                this.hideCustomAmount();
                
                // Refresh status and history
                this.loadFaucetStatus();
                this.loadAddressHistory(address);
                
            } else {
                let errorMessage = result.error || 'Request failed';
                
                if (result.status === 'rate_limited' && result.retry_after) {
                    const minutes = Math.ceil(result.retry_after / 60);
                    errorMessage += ` Please try again in ${minutes} minutes.`;
                }
                
                this.showAlert(`❌ ${errorMessage}`, 'error');
            }

        } catch (error) {
            console.error('Request error:', error);
            this.showAlert('❌ Network error. Please try again.', 'error');
        } finally {
            this.setLoadingState(false);
        }
    }

    /**
     * Handle amount selection change
     */
    handleAmountChange(event) {
        const value = event.target.value;
        const customGroup = document.getElementById('customAmountGroup');
        
        if (value === 'custom') {
            customGroup.classList.remove('hidden');
            document.getElementById('customAmount').focus();
        } else {
            customGroup.classList.add('hidden');
        }
    }

    /**
     * Hide custom amount input
     */
    hideCustomAmount() {
        const customGroup = document.getElementById('customAmountGroup');
        customGroup.classList.add('hidden');
    }

    /**
     * Validate wallet address
     */
    validateAddress(address) {
        const validation = document.getElementById('addressValidation');
        
        if (!address) {
            validation.textContent = '';
            validation.classList.remove('show', 'success');
            return false;
        }

        if (this.isValidAddress(address)) {
            validation.textContent = '✓ Valid KNIRV address';
            validation.classList.add('show', 'success');
            validation.classList.remove('error');
            return true;
        } else {
            validation.textContent = '✗ Invalid address format (must start with "knirv1")';
            validation.classList.add('show');
            validation.classList.remove('success');
            return false;
        }
    }

    /**
     * Validate custom amount
     */
    validateCustomAmount(amount) {
        const value = parseInt(amount);
        const input = document.getElementById('customAmount');
        
        if (!amount) {
            input.style.borderColor = '';
            return false;
        }

        if (value >= 100 && value <= 5000) {
            input.style.borderColor = '#4CAF50';
            return true;
        } else {
            input.style.borderColor = '#f44336';
            return false;
        }
    }

    /**
     * Check if address is valid KNIRV format
     */
    isValidAddress(address) {
        return address && 
               typeof address === 'string' && 
               address.startsWith('knirv1') && 
               address.length >= 20 &&
               /^knirv1[a-zA-Z0-9]+$/.test(address);
    }

    /**
     * Load faucet status
     */
    async loadFaucetStatus() {
        try {
            const response = await fetch(`${this.apiBase}/status`);
            const status = await response.json();
            
            this.displayFaucetStatus(status);
        } catch (error) {
            console.error('Failed to load faucet status:', error);
            this.displayFaucetStatus(null, error.message);
        }
    }

    /**
     * Display faucet status
     */
    displayFaucetStatus(status, error = null) {
        const container = document.getElementById('statusContent');
        
        if (error) {
            container.innerHTML = `
                <div class="alert alert-error">
                    Failed to load faucet status: ${error}
                </div>
            `;
            return;
        }

        if (!status) {
            container.innerHTML = `
                <div class="alert alert-warning">
                    Faucet status unavailable
                </div>
            `;
            return;
        }

        const statusClass = status.faucet_enabled ? 'success' : 'error';
        const statusText = status.faucet_enabled ? 'Online' : 'Offline';
        
        container.innerHTML = `
            <div class="status-grid">
                <div class="status-item">
                    <h4>Status</h4>
                    <div class="value text-${statusClass}">${statusText}</div>
                </div>
                <div class="status-item">
                    <h4>Balance</h4>
                    <div class="value">${status.current_balance?.toLocaleString() || 0}</div>
                    <div class="unit">NRV</div>
                </div>
                <div class="status-item">
                    <h4>Daily Limit</h4>
                    <div class="value">${status.daily_limit?.toLocaleString() || 0}</div>
                    <div class="unit">NRV</div>
                </div>
                <div class="status-item">
                    <h4>Remaining Today</h4>
                    <div class="value">${status.remaining_today?.toLocaleString() || 0}</div>
                    <div class="unit">NRV</div>
                </div>
                <div class="status-item">
                    <h4>Queue Size</h4>
                    <div class="value">${status.current_queue_size || 0}</div>
                    <div class="unit">requests</div>
                </div>
                <div class="status-item">
                    <h4>Success Rate</h4>
                    <div class="value">${status.success_rate_today || 0}%</div>
                </div>
            </div>
        `;
    }

    /**
     * Load economic status
     */
    async loadEconomicStatus() {
        try {
            const response = await fetch(`${this.apiBase}/economic/metrics`);
            const data = await response.json();
            
            this.displayEconomicStatus(data);
        } catch (error) {
            console.error('Failed to load economic status:', error);
            this.displayEconomicStatus(null, error.message);
        }
    }

    /**
     * Display economic status
     */
    displayEconomicStatus(data, error = null) {
        const container = document.getElementById('economicStatus');
        
        if (error) {
            container.innerHTML = `
                <div class="alert alert-error">
                    Failed to load economic status: ${error}
                </div>
            `;
            return;
        }

        if (!data || !data.economic_flow) {
            container.innerHTML = `
                <div class="alert alert-warning">
                    Economic status unavailable
                </div>
            `;
            return;
        }

        const flow = data.economic_flow;
        const sustainability = data.sustainability_status || 'unknown';
        
        const routerHealth = flow.router_health === 1 ? 'Healthy' : 'Degraded';
        const treasuryHealth = flow.treasury_health === 1 ? 'Healthy' : 'Degraded';
        const faucetHealth = flow.faucet_health === 1 ? 'Healthy' : 'Degraded';
        
        container.innerHTML = `
            <div class="status-grid">
                <div class="status-item">
                    <h4>Router Health</h4>
                    <div class="value text-${flow.router_health === 1 ? 'success' : 'error'}">${routerHealth}</div>
                </div>
                <div class="status-item">
                    <h4>Treasury Health</h4>
                    <div class="value text-${flow.treasury_health === 1 ? 'success' : 'error'}">${treasuryHealth}</div>
                </div>
                <div class="status-item">
                    <h4>Faucet Health</h4>
                    <div class="value text-${flow.faucet_health === 1 ? 'success' : 'error'}">${faucetHealth}</div>
                </div>
                <div class="status-item">
                    <h4>Sustainability</h4>
                    <div class="value">${flow.funding_sustainability_days || 0}</div>
                    <div class="unit">days</div>
                </div>
                <div class="status-item">
                    <h4>Treasury Balance</h4>
                    <div class="value">${flow.treasury_balance?.toLocaleString() || 0}</div>
                    <div class="unit">NRV</div>
                </div>
                <div class="status-item">
                    <h4>Proof Rate</h4>
                    <div class="value">${flow.router_proof_rate || 0}</div>
                    <div class="unit">per hour</div>
                </div>
            </div>
            <div class="alert alert-${sustainability === 'healthy' ? 'success' : sustainability === 'warning' ? 'warning' : 'error'}" style="margin-top: 15px;">
                <strong>Economic Flow Status:</strong> ${sustainability.charAt(0).toUpperCase() + sustainability.slice(1)}
            </div>
        `;
    }

    /**
     * Load address history
     */
    async loadAddressHistory(address) {
        if (!address || !this.isValidAddress(address)) {
            return;
        }

        this.currentAddress = address;
        const container = document.getElementById('requestHistory');
        
        try {
            const response = await fetch(`${this.apiBase}/history?address=${encodeURIComponent(address)}&limit=5`);
            const data = await response.json();
            
            this.displayRequestHistory(data.history || []);
        } catch (error) {
            console.error('Failed to load request history:', error);
            container.innerHTML = `
                <div class="alert alert-error">
                    Failed to load request history: ${error.message}
                </div>
            `;
        }
    }

    /**
     * Display request history
     */
    displayRequestHistory(history) {
        const container = document.getElementById('requestHistory');
        
        if (!history || history.length === 0) {
            container.innerHTML = `
                <p style="color: #aaa; text-align: center;">No requests found for this address</p>
            `;
            return;
        }

        const historyHTML = history.map(request => {
            const date = new Date(request.timestamp).toLocaleString();
            const statusClass = request.status === 'success' ? 'success' : 
                              request.status === 'failed' ? 'failed' : 'rejected';
            
            return `
                <div class="history-item ${statusClass}">
                    <div class="header">
                        <div class="amount">${request.amount} NRV</div>
                        <div class="status ${statusClass}">${request.status}</div>
                    </div>
                    <div class="details">
                        <div>Date: ${date}</div>
                        ${request.tx_hash ? `<div>TX: ${request.tx_hash}</div>` : ''}
                        ${request.error ? `<div>Error: ${request.error}</div>` : ''}
                        ${request.reason ? `<div>Reason: ${request.reason}</div>` : ''}
                    </div>
                </div>
            `;
        }).join('');

        container.innerHTML = historyHTML;
    }

    /**
     * Start automatic status updates
     */
    startStatusUpdates() {
        // Update faucet status every 30 seconds
        this.statusUpdateInterval = setInterval(() => {
            this.loadFaucetStatus();
        }, 30000);

        // Update economic status every 60 seconds
        this.economicStatusInterval = setInterval(() => {
            this.loadEconomicStatus();
        }, 60000);
    }

    /**
     * Stop automatic status updates
     */
    stopStatusUpdates() {
        if (this.statusUpdateInterval) {
            clearInterval(this.statusUpdateInterval);
            this.statusUpdateInterval = null;
        }
        
        if (this.economicStatusInterval) {
            clearInterval(this.economicStatusInterval);
            this.economicStatusInterval = null;
        }
    }

    /**
     * Set loading state
     */
    setLoadingState(loading) {
        const form = document.getElementById('faucetForm');
        const loadingDiv = document.getElementById('loadingState');
        const submitBtn = document.getElementById('submitBtn');

        if (loading) {
            form.style.display = 'none';
            loadingDiv.classList.remove('hidden');
            loadingDiv.classList.add('show');
            submitBtn.disabled = true;
        } else {
            form.style.display = 'block';
            loadingDiv.classList.add('hidden');
            loadingDiv.classList.remove('show');
            submitBtn.disabled = false;
        }
    }

    /**
     * Show alert message
     */
    showAlert(message, type = 'info') {
        const container = document.getElementById('resultMessages');
        
        const alertDiv = document.createElement('div');
        alertDiv.className = `alert alert-${type}`;
        alertDiv.innerHTML = message;
        
        container.appendChild(alertDiv);
        
        // Auto-remove after 10 seconds
        setTimeout(() => {
            if (alertDiv.parentNode) {
                alertDiv.parentNode.removeChild(alertDiv);
            }
        }, 10000);
    }

    /**
     * Clear all messages
     */
    clearMessages() {
        const container = document.getElementById('resultMessages');
        container.innerHTML = '';
    }

    /**
     * Cleanup when page unloads
     */
    destroy() {
        this.stopStatusUpdates();
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.faucetApp = new FaucetApp();
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (window.faucetApp) {
        window.faucetApp.destroy();
    }
});
