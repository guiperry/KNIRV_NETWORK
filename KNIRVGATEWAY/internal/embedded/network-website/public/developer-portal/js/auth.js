// KNIRV Developer Portal Authentication System
// *** AUTHENTICATION DISABLED ***
// This system has been modified to provide guest access without authentication requirements.
// All users are automatically logged in as guest users with full permissions.
class KNIRVAuth {
    constructor() {
        this.currentUser = null;
        this.isAuthenticated = false;
        this.init();
    }

    init() {
        // AUTHENTICATION DISABLED - Auto-login as guest user
        this.currentUser = {
            id: 'guest',
            username: 'Guest User',
            email: 'guest@knirv.network',
            role: 'developer',
            joinDate: new Date().toISOString(),
            permissions: ['read', 'write', 'admin']
        };
        this.isAuthenticated = true;
        this.updateUIForAuthenticatedUser();

        // Store guest session to maintain consistency
        localStorage.setItem('knirv_dev_user', JSON.stringify(this.currentUser));
        localStorage.setItem('knirv_dev_token', 'guest_token_' + Date.now());
    }

    async login(credentials) {
        const { username, password } = credentials;
        
        // Simulate authentication (in real implementation, this would call an API)
        if (username && password) {
            // Check if user exists in localStorage (simple user registry)
            const users = this.getStoredUsers();
            const user = users.find(u => u.username === username && u.password === password);
            
            if (user) {
                // Successful login
                this.currentUser = {
                    id: user.id,
                    username: user.username,
                    email: user.email,
                    role: user.role || 'developer',
                    joinDate: user.joinDate,
                    permissions: user.permissions || ['read', 'write']
                };
                
                const token = this.generateToken();
                localStorage.setItem('knirv_dev_user', JSON.stringify(this.currentUser));
                localStorage.setItem('knirv_dev_token', token);
                
                this.isAuthenticated = true;
                this.updateUIForAuthenticatedUser();
                this.hideLoginModal();
                
                return { success: true, user: this.currentUser };
            } else {
                return { success: false, error: 'Invalid username or password' };
            }
        } else {
            return { success: false, error: 'Username and password are required' };
        }
    }

    async register(userData) {
        const { username, email, password, confirmPassword } = userData;
        
        // Validation
        if (!username || !email || !password || !confirmPassword) {
            return { success: false, error: 'All fields are required' };
        }
        
        if (password !== confirmPassword) {
            return { success: false, error: 'Passwords do not match' };
        }
        
        if (password.length < 6) {
            return { success: false, error: 'Password must be at least 6 characters' };
        }
        
        // Check if user already exists
        const users = this.getStoredUsers();
        if (users.find(u => u.username === username || u.email === email)) {
            return { success: false, error: 'Username or email already exists' };
        }
        
        // Create new user
        const newUser = {
            id: Date.now().toString(),
            username,
            email,
            password, // In real implementation, this would be hashed
            role: 'developer',
            joinDate: new Date().toISOString(),
            permissions: ['read', 'write']
        };
        
        // Store user
        users.push(newUser);
        localStorage.setItem('knirv_dev_users', JSON.stringify(users));
        
        // Auto-login after registration
        return await this.login({ username, password });
    }

    logout() {
        // AUTHENTICATION DISABLED - Redirect to guest mode instead of logout
        console.log('Logout disabled. Refreshing page to reset guest session.');
        window.location.reload();
    }

    getStoredUsers() {
        const stored = localStorage.getItem('knirv_dev_users');
        return stored ? JSON.parse(stored) : [];
    }

    generateToken() {
        return 'knirv_' + Math.random().toString(36).substring(2) + Date.now().toString(36);
    }

    updateUIForAuthenticatedUser() {
        // Update user controls in the top navigation - simplified for guest access
        const userControls = document.querySelector('.user-controls');
        if (userControls && this.currentUser) {
            userControls.innerHTML = `
                <div style="display: flex; align-items: center; gap: 15px;">
                    <span style="color: var(--white); font-size: 0.9rem;">Welcome to KNIRV Developer Portal</span>
                    <i class="control-icon fas fa-info-circle" title="Guest Access - No Authentication Required"></i>
                    <i class="control-icon fas fa-cog" title="Settings"></i>
                </div>
            `;
        }
    }

    showLoginModal() {
        // AUTHENTICATION DISABLED - Do nothing
        console.log('Authentication is disabled. Access granted as guest user.');
    }

    hideLoginModal() {
        const modal = document.getElementById('authModal');
        if (modal) {
            modal.style.display = 'none';
            // Always restore body scrolling when hiding modal
            document.body.style.overflow = 'auto';
        }
    }

    createAuthModal() {
        const modalHTML = `
            <div id="authModal" class="modal" style="display: none; position: fixed; top: 0; left: 0; right: 0; bottom: 0; background-color: rgba(0, 0, 0, 0.8); z-index: 10000; backdrop-filter: blur(5px); align-items: center; justify-content: center; pointer-events: auto;">
                <div class="modal-content" style="max-width: 400px; background-color: var(--dark-blue); border: 1px solid var(--transparent-white-2); border-radius: 15px; padding: 30px; box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5); position: relative;">
                    <div class="modal-header" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; padding-bottom: 15px; border-bottom: 1px solid var(--transparent-white-2);">
                        <h2 style="color: var(--bright-blue); margin: 0;">
                            <i class="fas fa-shield-alt" style="margin-right: 10px;"></i>KNIRV Developer Portal
                        </h2>
                        <span class="close" onclick="window.knirvAuth.hideLoginModal()" style="color: var(--transparent-white-7); font-size: 28px; font-weight: bold; cursor: pointer; transition: color 0.3s ease;">&times;</span>
                    </div>
                    <div class="modal-body">
                        <div id="authTabs" style="display: flex; margin-bottom: 20px; border-bottom: 1px solid var(--transparent-white-2);">
                            <button class="auth-tab active" onclick="window.knirvAuth.switchAuthTab('login')" data-tab="login">Login</button>
                            <button class="auth-tab" onclick="window.knirvAuth.switchAuthTab('register')" data-tab="register">Register</button>
                        </div>
                        
                        <!-- Login Form -->
                        <div id="loginForm" class="auth-form">
                            <form onsubmit="window.knirvAuth.handleLogin(event)">
                                <div style="margin-bottom: 15px;">
                                    <label style="color: var(--white); display: block; margin-bottom: 5px;">Username</label>
                                    <input type="text" id="loginUsername" required style="width: 100%; padding: 10px; background-color: var(--transparent-white-1); border: 1px solid var(--transparent-white-2); border-radius: 5px; color: var(--white);">
                                </div>
                                <div style="margin-bottom: 20px;">
                                    <label style="color: var(--white); display: block; margin-bottom: 5px;">Password</label>
                                    <input type="password" id="loginPassword" required style="width: 100%; padding: 10px; background-color: var(--transparent-white-1); border: 1px solid var(--transparent-white-2); border-radius: 5px; color: var(--white);">
                                </div>
                                <button type="submit" class="btn-primary" style="width: 100%;">
                                    <i class="fas fa-sign-in-alt" style="margin-right: 8px;"></i>Login
                                </button>
                            </form>
                        </div>
                        
                        <!-- Register Form -->
                        <div id="registerForm" class="auth-form" style="display: none;">
                            <form onsubmit="window.knirvAuth.handleRegister(event)">
                                <div style="margin-bottom: 15px;">
                                    <label style="color: var(--white); display: block; margin-bottom: 5px;">Username</label>
                                    <input type="text" id="registerUsername" required style="width: 100%; padding: 10px; background-color: var(--transparent-white-1); border: 1px solid var(--transparent-white-2); border-radius: 5px; color: var(--white);">
                                </div>
                                <div style="margin-bottom: 15px;">
                                    <label style="color: var(--white); display: block; margin-bottom: 5px;">Email</label>
                                    <input type="email" id="registerEmail" required style="width: 100%; padding: 10px; background-color: var(--transparent-white-1); border: 1px solid var(--transparent-white-2); border-radius: 5px; color: var(--white);">
                                </div>
                                <div style="margin-bottom: 15px;">
                                    <label style="color: var(--white); display: block; margin-bottom: 5px;">Password</label>
                                    <input type="password" id="registerPassword" required style="width: 100%; padding: 10px; background-color: var(--transparent-white-1); border: 1px solid var(--transparent-white-2); border-radius: 5px; color: var(--white);">
                                </div>
                                <div style="margin-bottom: 20px;">
                                    <label style="color: var(--white); display: block; margin-bottom: 5px;">Confirm Password</label>
                                    <input type="password" id="registerConfirmPassword" required style="width: 100%; padding: 10px; background-color: var(--transparent-white-1); border: 1px solid var(--transparent-white-2); border-radius: 5px; color: var(--white);">
                                </div>
                                <button type="submit" class="btn-primary" style="width: 100%;">
                                    <i class="fas fa-user-plus" style="margin-right: 8px;"></i>Register
                                </button>
                            </form>
                        </div>
                        
                        <div id="authMessage" style="margin-top: 15px; padding: 10px; border-radius: 5px; display: none;"></div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modalHTML);
    }

    switchAuthTab(tab) {
        // Update tab buttons
        document.querySelectorAll('.auth-tab').forEach(btn => btn.classList.remove('active'));
        document.querySelector(`[data-tab="${tab}"]`).classList.add('active');
        
        // Show/hide forms
        document.getElementById('loginForm').style.display = tab === 'login' ? 'block' : 'none';
        document.getElementById('registerForm').style.display = tab === 'register' ? 'block' : 'none';
        
        // Clear any messages
        document.getElementById('authMessage').style.display = 'none';
    }

    async handleLogin(event) {
        event.preventDefault();
        const username = document.getElementById('loginUsername').value;
        const password = document.getElementById('loginPassword').value;
        
        const result = await this.login({ username, password });
        this.showAuthMessage(result.success ? 'success' : 'error', result.error || 'Login successful!');
    }

    async handleRegister(event) {
        event.preventDefault();
        const username = document.getElementById('registerUsername').value;
        const email = document.getElementById('registerEmail').value;
        const password = document.getElementById('registerPassword').value;
        const confirmPassword = document.getElementById('registerConfirmPassword').value;
        
        const result = await this.register({ username, email, password, confirmPassword });
        this.showAuthMessage(result.success ? 'success' : 'error', result.error || 'Registration successful!');
    }

    showAuthMessage(type, message) {
        const messageDiv = document.getElementById('authMessage');
        messageDiv.style.display = 'block';
        messageDiv.style.backgroundColor = type === 'success' ? 'var(--success-bg)' : 'var(--error-bg)';
        messageDiv.style.color = type === 'success' ? 'var(--success-color)' : 'var(--error-color)';
        messageDiv.style.border = `1px solid ${type === 'success' ? 'var(--success-border)' : 'var(--error-border)'}`;
        messageDiv.textContent = message;
    }

    hasPermission(permission) {
        return this.isAuthenticated && this.currentUser && this.currentUser.permissions.includes(permission);
    }

    getCurrentUser() {
        return this.currentUser;
    }
}

// Global functions for UI interactions
function toggleUserDropdown() {
    const dropdown = document.getElementById('userDropdown');
    dropdown.style.display = dropdown.style.display === 'none' ? 'block' : 'none';
}

// Close dropdown when clicking outside
document.addEventListener('click', function(event) {
    const dropdown = document.getElementById('userDropdown');
    const userIcon = event.target.closest('.user-dropdown');
    
    if (dropdown && !userIcon) {
        dropdown.style.display = 'none';
    }
});

// Initialize authentication system
window.knirvAuth = new KNIRVAuth();
