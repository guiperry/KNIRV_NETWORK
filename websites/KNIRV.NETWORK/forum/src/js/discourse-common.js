// KNIRV Discourse Forum - Common JavaScript Functions

// Global configuration
const DISCOURSE_CONFIG = {
    apiBase: '/.netlify/functions',
    maxFileSize: 10 * 1024 * 1024, // 10MB
    allowedFileTypes: ['image/', 'text/', 'application/pdf'],
    maxFiles: 5,
    siteName: 'KNIRV Forum'
};

// Utility Functions
class DiscourseUtils {
    
    // Show loading spinner
    static showLoading(element, show = true) {
        if (show) {
            element.disabled = true;
            element.dataset.originalText = element.textContent;
            element.innerHTML = '<span class="spinner"></span> Loading...';
        } else {
            element.disabled = false;
            element.textContent = element.dataset.originalText || 'Submit';
        }
    }
    
    // Show notification message
    static showNotification(message, type = 'info', duration = 5000) {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <span class="notification-message">${message}</span>
            <button class="notification-close" onclick="this.parentElement.remove()">×</button>
        `;
        
        // Add to page
        document.body.appendChild(notification);
        
        // Auto remove after duration
        setTimeout(() => {
            if (notification.parentElement) {
                notification.remove();
            }
        }, duration);
        
        return notification;
    }
    
    // Format date
    static formatDate(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diff = now - date;
        
        if (diff < 60000) return 'just now';
        if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
        if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
        if (diff < 2592000000) return `${Math.floor(diff / 86400000)}d ago`;
        
        return date.toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        });
    }
    
    // Format file size
    static formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }
    
    // Validate email
    static isValidEmail(email) {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    }
    
    // Sanitize HTML
    static sanitizeHtml(str) {
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
    
    // Get URL parameters
    static getUrlParams() {
        return new URLSearchParams(window.location.search);
    }
    
    // Set URL parameter
    static setUrlParam(key, value) {
        const url = new URL(window.location);
        url.searchParams.set(key, value);
        window.history.pushState({}, '', url);
    }
    
    // Debounce function
    static debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }
    
    // Copy to clipboard
    static async copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
            this.showNotification('Copied to clipboard!', 'success', 2000);
            return true;
        } catch (err) {
            console.error('Failed to copy: ', err);
            this.showNotification('Failed to copy to clipboard', 'error');
            return false;
        }
    }
    
    // Generate slug from title
    static generateSlug(title) {
        return title
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-|-$/g, '');
    }
    
    // Truncate text
    static truncateText(text, maxLength = 100) {
        if (text.length <= maxLength) return text;
        return text.substring(0, maxLength) + '...';
    }
    
    // Parse markdown (basic)
    static parseMarkdown(text) {
        return text
            .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
            .replace(/\*(.*?)\*/g, '<em>$1</em>')
            .replace(/`(.*?)`/g, '<code>$1</code>')
            .replace(/\n/g, '<br>');
    }
}

// Authentication Manager
class DiscourseAuth {
    
    static getToken() {
        return localStorage.getItem('discourse_token');
    }
    
    static setToken(token) {
        localStorage.setItem('discourse_token', token);
    }
    
    static removeToken() {
        localStorage.removeItem('discourse_token');
        localStorage.removeItem('discourse_user');
    }
    
    static getUser() {
        const userStr = localStorage.getItem('discourse_user');
        return userStr ? JSON.parse(userStr) : null;
    }
    
    static setUser(user) {
        localStorage.setItem('discourse_user', JSON.stringify(user));
    }
    
    static isLoggedIn() {
        const token = this.getToken();
        if (!token) return false;
        
        try {
            const payload = JSON.parse(atob(token));
            return payload.exp > Date.now();
        } catch {
            return false;
        }
    }
    
    static logout() {
        this.removeToken();
        window.location.reload();
    }
    
    static requireAuth() {
        if (!this.isLoggedIn()) {
            DiscourseUtils.showNotification('Please log in to continue', 'warning');
            return false;
        }
        return true;
    }
    
    static isAdmin() {
        const user = this.getUser();
        return user && user.admin;
    }
    
    static isModerator() {
        const user = this.getUser();
        return user && (user.admin || user.moderator);
    }
}

// API Client
class DiscourseAPI {
    
    static async request(endpoint, options = {}) {
        const url = `${DISCOURSE_CONFIG.apiBase}/${endpoint}`;
        const token = DiscourseAuth.getToken();
        
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
                ...(token && { 'Authorization': `Bearer ${token}` })
            }
        };
        
        const finalOptions = {
            ...defaultOptions,
            ...options,
            headers: {
                ...defaultOptions.headers,
                ...options.headers
            }
        };
        
        try {
            const response = await fetch(url, finalOptions);
            const data = await response.json();
            
            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}`);
            }
            
            return data;
        } catch (error) {
            console.error('API Request failed:', error);
            throw error;
        }
    }
    
    static async get(endpoint, params = {}) {
        const queryString = new URLSearchParams(params).toString();
        const url = queryString ? `${endpoint}?${queryString}` : endpoint;
        return this.request(url);
    }
    
    static async post(endpoint, data) {
        return this.request(endpoint, {
            method: 'POST',
            body: JSON.stringify(data)
        });
    }
    
    static async put(endpoint, data) {
        return this.request(endpoint, {
            method: 'PUT',
            body: JSON.stringify(data)
        });
    }
    
    static async delete(endpoint) {
        return this.request(endpoint, {
            method: 'DELETE'
        });
    }
    
    static async postForm(endpoint, formData) {
        const token = DiscourseAuth.getToken();
        const headers = {};
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }
        
        return this.request(endpoint, {
            method: 'POST',
            headers,
            body: formData
        });
    }
}

// Modal Manager
class DiscourseModal {
    
    static show(content, options = {}) {
        const modal = document.createElement('div');
        modal.className = 'discourse-modal';
        modal.innerHTML = `
            <div class="modal-backdrop" onclick="DiscourseModal.close()"></div>
            <div class="modal-content">
                <div class="modal-header">
                    <h3>${options.title || 'Modal'}</h3>
                    <button class="modal-close" onclick="DiscourseModal.close()">×</button>
                </div>
                <div class="modal-body">
                    ${content}
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        document.body.classList.add('modal-open');
        
        return modal;
    }
    
    static close() {
        const modal = document.querySelector('.discourse-modal');
        if (modal) {
            modal.remove();
            document.body.classList.remove('modal-open');
        }
    }
    
    static showLogin() {
        const content = `
            <form id="login-form" class="auth-form">
                <div class="form-group">
                    <label for="login-username">Username or Email</label>
                    <input type="text" id="login-username" name="username" required>
                </div>
                <div class="form-group">
                    <label for="login-password">Password</label>
                    <input type="password" id="login-password" name="password" required>
                </div>
                <div class="form-actions">
                    <button type="submit" class="btn btn-primary">Log In</button>
                    <button type="button" class="btn btn-secondary" onclick="DiscourseModal.close()">Cancel</button>
                </div>
            </form>
        `;
        
        const modal = this.show(content, { title: 'Log In' });
        
        const form = modal.querySelector('#login-form');
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const formData = new FormData(form);
            const credentials = {
                login: formData.get('username'),
                password: formData.get('password')
            };
            
            try {
                const response = await DiscourseAPI.post('discourse-session', credentials);
                
                if (response.success) {
                    DiscourseAuth.setToken(response.token);
                    DiscourseAuth.setUser(response.user);
                    DiscourseUtils.showNotification('Logged in successfully!', 'success');
                    this.close();
                    window.location.reload();
                } else {
                    DiscourseUtils.showNotification(response.error, 'error');
                }
            } catch (error) {
                DiscourseUtils.showNotification('Login failed', 'error');
            }
        });
    }
    
    static showSignup() {
        const content = `
            <form id="signup-form" class="auth-form">
                <div class="form-group">
                    <label for="signup-username">Username</label>
                    <input type="text" id="signup-username" name="username" required>
                </div>
                <div class="form-group">
                    <label for="signup-email">Email</label>
                    <input type="email" id="signup-email" name="email" required>
                </div>
                <div class="form-group">
                    <label for="signup-password">Password</label>
                    <input type="password" id="signup-password" name="password" required>
                </div>
                <div class="form-group">
                    <label for="signup-name">Full Name (optional)</label>
                    <input type="text" id="signup-name" name="name">
                </div>
                <div class="form-actions">
                    <button type="submit" class="btn btn-primary">Sign Up</button>
                    <button type="button" class="btn btn-secondary" onclick="DiscourseModal.close()">Cancel</button>
                </div>
            </form>
        `;
        
        const modal = this.show(content, { title: 'Sign Up' });
        
        const form = modal.querySelector('#signup-form');
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const formData = new FormData(form);
            const userData = {
                username: formData.get('username'),
                email: formData.get('email'),
                password: formData.get('password'),
                name: formData.get('name')
            };
            
            try {
                const response = await DiscourseAPI.post('discourse-users', userData);
                
                if (response.user) {
                    DiscourseUtils.showNotification('Account created successfully! Please log in.', 'success');
                    this.close();
                    this.showLogin();
                } else {
                    DiscourseUtils.showNotification(response.error, 'error');
                }
            } catch (error) {
                DiscourseUtils.showNotification('Signup failed', 'error');
            }
        });
    }
}

// Initialize common functionality when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    // Add modal styles if not present
    if (!document.querySelector('#discourse-modal-styles')) {
        const styles = document.createElement('style');
        styles.id = 'discourse-modal-styles';
        styles.textContent = `
            .discourse-modal {
                position: fixed;
                top: 0;
                left: 0;
                width: 100%;
                height: 100%;
                z-index: 2000;
                display: flex;
                align-items: center;
                justify-content: center;
            }
            .modal-backdrop {
                position: absolute;
                top: 0;
                left: 0;
                width: 100%;
                height: 100%;
                background: rgba(0, 0, 0, 0.5);
                backdrop-filter: blur(5px);
            }
            .modal-content {
                position: relative;
                background: white;
                border-radius: 15px;
                box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
                max-width: 500px;
                width: 90%;
                max-height: 90vh;
                overflow-y: auto;
            }
            .modal-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                padding: 20px 30px;
                border-bottom: 1px solid #e9ecef;
            }
            .modal-close {
                background: none;
                border: none;
                font-size: 1.5rem;
                cursor: pointer;
                color: #6c757d;
            }
            .modal-body {
                padding: 30px;
            }
            .auth-form .form-group {
                margin-bottom: 20px;
            }
            .auth-form label {
                display: block;
                margin-bottom: 5px;
                font-weight: 500;
                color: #495057;
            }
            .auth-form input {
                width: 100%;
                padding: 12px;
                border: 2px solid #e9ecef;
                border-radius: 8px;
                font-size: 1rem;
                transition: border-color 0.3s ease;
            }
            .auth-form input:focus {
                outline: none;
                border-color: #667eea;
            }
            .form-actions {
                display: flex;
                gap: 15px;
                justify-content: flex-end;
                margin-top: 30px;
            }
            body.modal-open {
                overflow: hidden;
            }
        `;
        document.head.appendChild(styles);
    }
    
    // Setup global event listeners
    setupGlobalEventListeners();
});

function setupGlobalEventListeners() {
    // Login button
    const loginBtn = document.getElementById('login-btn');
    if (loginBtn) {
        loginBtn.addEventListener('click', () => DiscourseModal.showLogin());
    }
    
    // Signup button
    const signupBtn = document.getElementById('signup-btn');
    if (signupBtn) {
        signupBtn.addEventListener('click', () => DiscourseModal.showSignup());
    }
    
    // Logout button
    const logoutBtn = document.getElementById('logout-btn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', () => DiscourseAuth.logout());
    }
}

// Export for use in other scripts
window.DiscourseUtils = DiscourseUtils;
window.DiscourseAuth = DiscourseAuth;
window.DiscourseAPI = DiscourseAPI;
window.DiscourseModal = DiscourseModal;
