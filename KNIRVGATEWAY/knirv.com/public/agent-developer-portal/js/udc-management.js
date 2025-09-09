/**
 * UDC Management JavaScript
 * Handles User Delegation Certificate functionality
 */

class UDCManager {
    constructor() {
        this.udcs = [];
        this.init();
    }

    init() {
        this.loadSampleUDCs();
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Form submission for creating new UDCs
        const udcForm = document.querySelector('form.portal-form');
        if (udcForm) {
            udcForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.createUDC(e.target);
            });
        }
    }

    loadSampleUDCs() {
        // Sample UDC data for demonstration
        this.udcs = [
            {
                id: 'udc_001',
                agentId: 'agent_001',
                type: 'skill-execution',
                authorityLevel: 'intermediate',
                status: 'active',
                issuedDate: new Date(Date.now() - 86400000 * 2), // 2 days ago
                expiresDate: new Date(Date.now() + 86400000 * 28), // 28 days from now
                scope: 'Data analysis and pattern recognition tasks'
            },
            {
                id: 'udc_002',
                agentId: 'agent_002',
                type: 'resource-access',
                authorityLevel: 'basic',
                status: 'active',
                issuedDate: new Date(Date.now() - 86400000 * 5), // 5 days ago
                expiresDate: new Date(Date.now() + 86400000 * 25), // 25 days from now
                scope: 'Access to computational resources for text processing'
            },
            {
                id: 'udc_003',
                agentId: 'agent_001',
                type: 'governance-voting',
                authorityLevel: 'advanced',
                status: 'pending',
                issuedDate: new Date(Date.now() - 86400000 * 1), // 1 day ago
                expiresDate: new Date(Date.now() + 86400000 * 89), // 89 days from now
                scope: 'Participation in network governance and voting on proposals'
            }
        ];

        this.renderUDCTable();
        this.updateStats();
    }

    createUDC(form) {
        const formData = new FormData(form);
        const udcData = {
            id: `udc_${Date.now()}`,
            agentId: formData.get('agentId'),
            type: formData.get('delegationType'),
            authorityLevel: formData.get('authorityLevel'),
            status: 'pending',
            issuedDate: new Date(),
            expiresDate: new Date(Date.now() + parseInt(formData.get('validityPeriod')) * 86400000),
            scope: formData.get('scopeDescription')
        };

        // Simulate UDC creation process
        window.knirvPortal.showNotification('Creating UDC...', 'info');
        
        setTimeout(() => {
            this.udcs.push(udcData);
            this.renderUDCTable();
            this.updateStats();
            form.reset();
            window.knirvPortal.showNotification('UDC created successfully!', 'success');
        }, 2000);
    }

    renderUDCTable() {
        const tbody = document.getElementById('udcTableBody');
        if (!tbody) return;

        if (this.udcs.length === 0) {
            tbody.innerHTML = `
                <tr data-searchable="udcs">
                    <td colspan="8" class="text-center text-gray-500 py-8">
                        No UDCs found. Issue your first UDC to get started.
                    </td>
                </tr>
            `;
            return;
        }

        tbody.innerHTML = this.udcs.map(udc => `
            <tr data-searchable="udcs">
                <td>
                    <div class="font-mono text-sm">${udc.id}</div>
                </td>
                <td>
                    <div class="font-mono text-sm">${udc.agentId}</div>
                </td>
                <td>
                    <span class="capitalize">${udc.type.replace('-', ' ')}</span>
                </td>
                <td>
                    <span class="capitalize ${this.getAuthorityColor(udc.authorityLevel)}">${udc.authorityLevel}</span>
                </td>
                <td>
                    <span class="status-badge ${this.getStatusClass(udc.status)}">${udc.status}</span>
                </td>
                <td>
                    <div class="text-sm">${this.formatDate(udc.issuedDate)}</div>
                </td>
                <td>
                    <div class="text-sm ${this.isExpiringSoon(udc.expiresDate) ? 'text-yellow-400' : ''}">${this.formatDate(udc.expiresDate)}</div>
                </td>
                <td>
                    <div class="flex space-x-2">
                        <button class="btn-primary text-xs" onclick="udcManager.viewUDC('${udc.id}')">
                            <i class="fas fa-eye mr-1"></i>View
                        </button>
                        <button class="btn-outline text-xs" onclick="udcManager.revokeUDC('${udc.id}')">
                            <i class="fas fa-times mr-1"></i>Revoke
                        </button>
                    </div>
                </td>
            </tr>
        `).join('');
    }

    updateStats() {
        const activeUDCs = this.udcs.filter(udc => udc.status === 'active').length;
        const verifiedUDCs = this.udcs.filter(udc => udc.status === 'active').length; // Assuming active = verified for demo
        const pendingUDCs = this.udcs.filter(udc => udc.status === 'pending').length;
        const expiringUDCs = this.udcs.filter(udc => this.isExpiringSoon(udc.expiresDate)).length;

        // Update stat elements
        const statElements = {
            'active-udcs': activeUDCs,
            'verified-udcs': verifiedUDCs,
            'pending-udcs': pendingUDCs,
            'expiring-udcs': expiringUDCs
        };

        Object.keys(statElements).forEach(key => {
            const element = document.querySelector(`[data-stat="${key}"]`);
            if (element) {
                element.textContent = statElements[key];
            }
        });
    }

    getStatusClass(status) {
        switch (status) {
            case 'active': return 'status-active';
            case 'pending': return 'status-pending';
            case 'expired': return 'status-error';
            case 'revoked': return 'status-error';
            default: return 'status-inactive';
        }
    }

    getAuthorityColor(level) {
        switch (level) {
            case 'basic': return 'text-green-400';
            case 'intermediate': return 'text-yellow-400';
            case 'advanced': return 'text-orange-400';
            case 'expert': return 'text-red-400';
            default: return 'text-gray-400';
        }
    }

    formatDate(date) {
        return date.toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        });
    }

    isExpiringSoon(date) {
        const daysUntilExpiry = (date - new Date()) / (1000 * 60 * 60 * 24);
        return daysUntilExpiry <= 7 && daysUntilExpiry > 0;
    }

    viewUDC(udcId) {
        const udc = this.udcs.find(u => u.id === udcId);
        if (!udc) return;

        // Create modal content for UDC details
        const modalContent = `
            <div class="space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label class="text-sm font-semibold text-gray-400">UDC ID</label>
                        <p class="font-mono">${udc.id}</p>
                    </div>
                    <div>
                        <label class="text-sm font-semibold text-gray-400">Agent ID</label>
                        <p class="font-mono">${udc.agentId}</p>
                    </div>
                    <div>
                        <label class="text-sm font-semibold text-gray-400">Type</label>
                        <p class="capitalize">${udc.type.replace('-', ' ')}</p>
                    </div>
                    <div>
                        <label class="text-sm font-semibold text-gray-400">Authority Level</label>
                        <p class="capitalize ${this.getAuthorityColor(udc.authorityLevel)}">${udc.authorityLevel}</p>
                    </div>
                    <div>
                        <label class="text-sm font-semibold text-gray-400">Status</label>
                        <p><span class="status-badge ${this.getStatusClass(udc.status)}">${udc.status}</span></p>
                    </div>
                    <div>
                        <label class="text-sm font-semibold text-gray-400">Issued Date</label>
                        <p>${this.formatDate(udc.issuedDate)}</p>
                    </div>
                    <div>
                        <label class="text-sm font-semibold text-gray-400">Expires Date</label>
                        <p class="${this.isExpiringSoon(udc.expiresDate) ? 'text-yellow-400' : ''}">${this.formatDate(udc.expiresDate)}</p>
                    </div>
                </div>
                <div>
                    <label class="text-sm font-semibold text-gray-400">Scope Description</label>
                    <p class="mt-1 p-3 bg-gray-700 rounded-lg">${udc.scope}</p>
                </div>
                <div class="flex space-x-2 pt-4">
                    <button class="btn-outline" onclick="udcManager.downloadUDC('${udc.id}')">
                        <i class="fas fa-download mr-2"></i>Download Certificate
                    </button>
                    <button class="btn-secondary" onclick="udcManager.renewUDC('${udc.id}')">
                        <i class="fas fa-refresh mr-2"></i>Renew
                    </button>
                </div>
            </div>
        `;

        // Show modal (assuming modal functionality exists)
        this.showModal('UDC Details', modalContent);
    }

    revokeUDC(udcId) {
        if (confirm('Are you sure you want to revoke this UDC? This action cannot be undone.')) {
            const udcIndex = this.udcs.findIndex(u => u.id === udcId);
            if (udcIndex !== -1) {
                this.udcs[udcIndex].status = 'revoked';
                this.renderUDCTable();
                this.updateStats();
                window.knirvPortal.showNotification('UDC revoked successfully', 'success');
            }
        }
    }

    downloadUDC(udcId) {
        window.knirvPortal.showNotification('Downloading UDC certificate...', 'info');
        // Simulate download
        setTimeout(() => {
            window.knirvPortal.showNotification('UDC certificate downloaded!', 'success');
        }, 1000);
    }

    renewUDC(udcId) {
        window.knirvPortal.showNotification('Renewing UDC...', 'info');
        // Simulate renewal
        setTimeout(() => {
            const udcIndex = this.udcs.findIndex(u => u.id === udcId);
            if (udcIndex !== -1) {
                this.udcs[udcIndex].expiresDate = new Date(Date.now() + 86400000 * 30); // Extend by 30 days
                this.renderUDCTable();
                this.updateStats();
                window.knirvPortal.showNotification('UDC renewed successfully!', 'success');
            }
        }, 1500);
    }

    showModal(title, content) {
        // Simple modal implementation
        const modal = document.createElement('div');
        modal.className = 'modal-overlay';
        modal.innerHTML = `
            <div class="modal-content max-w-2xl">
                <div class="flex justify-between items-center mb-4">
                    <h3 class="text-xl font-semibold">${title}</h3>
                    <button class="modal-close text-gray-400 hover:text-white">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                ${content}
            </div>
        `;
        
        document.body.appendChild(modal);
        
        // Close modal functionality
        modal.querySelector('.modal-close').addEventListener('click', () => {
            modal.remove();
        });
        
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.remove();
            }
        });
    }
}

// Initialize UDC Manager when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.udcManager = new UDCManager();
});
