// KNIRV Developer Portal Authentication System
class KNIRVAuth {
    constructor() {
        this.currentUser = null;
        this.isAuthenticated = false;
        this.init();
    }

    init() {
        // Check for existing session
        const storedUser = localStorage.getItem('knirv_dev_user');
        const storedToken = localStorage.getItem('knirv_dev_token');
        
        if (storedUser && storedToken) {
            try {
                this.currentUser = JSON.parse(storedUser);
                this.isAuthenticated = true;
                this.updateUIForAuthenticatedUser();
            } catch (error) {
                console.error('Error parsing stored user data:', error);
                this.logout();
            }
        } else {
            this.showLoginModal();
        }
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
        this.currentUser = null;
        this.isAuthenticated = false;
        localStorage.removeItem('knirv_dev_user');
        localStorage.removeItem('knirv_dev_token');
        this.showLoginModal();
    }

    getStoredUsers() {
        const stored = localStorage.getItem('knirv_dev_users');
        return stored ? JSON.parse(stored) : [];
    }

    generateToken() {
        return 'knirv_' + Math.random().toString(36).substring(2) + Date.now().toString(36);
    }

    updateUIForAuthenticatedUser() {
        // Update user controls in the top navigation
        const userControls = document.querySelector('.user-controls');
        if (userControls && this.currentUser) {
            userControls.innerHTML = `
                <div style="display: flex; align-items: center; gap: 15px;">
                    <span style="color: var(--white); font-size: 0.9rem;">Welcome, ${this.currentUser.username}</span>
                    <i class="control-icon fas fa-bell" title="Notifications"></i>
                    <i class="control-icon fas fa-cog" title="Settings"></i>
                    <div class="user-dropdown" style="position: relative;">
                        <i class="control-icon fas fa-user-circle" title="Profile" onclick="toggleUserDropdown()"></i>
                        <div id="userDropdown" class="dropdown-menu" style="display: none; position: absolute; right: 0; top: 100%; background: var(--dark-bg); border: 1px solid var(--transparent-white-2); border-radius: 8px; padding: 10px; min-width: 150px; z-index: 1000;">
                            <div style="padding: 8px 0; border-bottom: 1px solid var(--transparent-white-2); margin-bottom: 8px;">
                                <div style="color: var(--white); font-weight: 600;">${this.currentUser.username}</div>
                                <div style="color: var(--transparent-white-7); font-size: 0.8rem;">${this.currentUser.email}</div>
                            </div>
                            <div onclick="window.knirvAuth.logout()" style="padding: 8px 0; color: var(--white); cursor: pointer; border-radius: 4px;" onmouseover="this.style.backgroundColor='var(--transparent-white-1)'" onmouseout="this.style.backgroundColor='transparent'">
                                <i class="fas fa-sign-out-alt" style="margin-right: 8px;"></i>Logout
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }
    }

    showLoginModal() {
        // Create login modal if it doesn't exist
        if (!document.getElementById('authModal')) {
            this.createAuthModal();
        }
        document.getElementById('authModal').style.display = 'block';
    }

    hideLoginModal() {
        const modal = document.getElementById('authModal');
        if (modal) {
            modal.style.display = 'none';
        }
    }

    createAuthModal() {
        const modalHTML = `
            <div id="authModal" class="modal" style="display: none; z-index: 10000;">
                <div class="modal-content" style="max-width: 400px;">
                    <div class="modal-header">
                        <h2 style="color: var(--bright-blue); margin: 0;">
                            <i class="fas fa-shield-alt" style="margin-right: 10px;"></i>KNIRV Developer Portal
                        </h2>
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
