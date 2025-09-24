/**
 * PWA Installation and Setup Script for KNIRVCONTROLLER
 * Handles PWA installation, authentication setup, and user data directory configuration
 */

class PWAInstaller {
  constructor() {
    this.platform = this.detectPlatform();
    this.isStandalone = this.isRunningStandalone();
    this.deferredPrompt = null;
    this.installationState = 'not_installed';
    
    this.init();
  }

  init() {
    // Listen for beforeinstallprompt event
    window.addEventListener('beforeinstallprompt', (e) => {
      e.preventDefault();
      this.deferredPrompt = e;
      this.showInstallPrompt();
    });

    // Listen for appinstalled event
    window.addEventListener('appinstalled', () => {
      this.onInstallSuccess();
    });

    // Check URL parameters for installation intent
    const urlParams = new URLSearchParams(window.location.search);
    const installParam = urlParams.get('install');
    
    if (installParam) {
      this.handleInstallIntent(installParam);
    }

    // Initialize user data directory
    this.initializeUserDataDirectory();
    
    // Setup authentication
    this.setupAuthentication();
  }

  detectPlatform() {
    const userAgent = navigator.userAgent.toLowerCase();
    
    if (userAgent.includes('android')) return 'android';
    if (userAgent.includes('iphone') || userAgent.includes('ipad')) return 'ios';
    if (userAgent.includes('windows')) return 'windows';
    if (userAgent.includes('mac')) return 'mac';
    if (userAgent.includes('linux')) return 'linux';
    
    return 'web';
  }

  isRunningStandalone() {
    // Check if running as PWA
    return window.matchMedia('(display-mode: standalone)').matches ||
           window.navigator.standalone === true ||
           document.referrer.includes('android-app://');
  }

  async handleInstallIntent(platform) {
    console.log(`Installation intent detected for platform: ${platform}`);
    
    if (platform === this.platform || platform === 'auto') {
      await this.promptInstallation();
    } else {
      this.showPlatformMismatchWarning(platform);
    }
  }

  async promptInstallation() {
    if (this.isStandalone) {
      this.showAlreadyInstalledMessage();
      return;
    }

    if (this.platform === 'ios') {
      this.showIOSInstallInstructions();
    } else if (this.platform === 'android' && this.deferredPrompt) {
      await this.showAndroidInstallPrompt();
    } else {
      this.showGenericInstallInstructions();
    }
  }

  async showAndroidInstallPrompt() {
    if (!this.deferredPrompt) {
      this.showGenericInstallInstructions();
      return;
    }

    try {
      this.deferredPrompt.prompt();
      const { outcome } = await this.deferredPrompt.userChoice;
      
      if (outcome === 'accepted') {
        console.log('PWA installation accepted');
        this.trackInstallation('android', 'accepted');
      } else {
        console.log('PWA installation dismissed');
        this.trackInstallation('android', 'dismissed');
      }
      
      this.deferredPrompt = null;
    } catch (error) {
      console.error('Installation prompt failed:', error);
      this.showGenericInstallInstructions();
    }
  }

  showIOSInstallInstructions() {
    const modal = this.createModal('Install KNIRV Controller', `
      <div class="install-instructions ios">
        <div class="step">
          <div class="step-number">1</div>
          <div class="step-content">
            <h4>Open Safari</h4>
            <p>Make sure you're using Safari browser (not Chrome or other browsers)</p>
          </div>
        </div>
        <div class="step">
          <div class="step-number">2</div>
          <div class="step-content">
            <h4>Tap the Share Button</h4>
            <p>Look for the <span class="share-icon">□↗</span> icon at the bottom of the screen</p>
          </div>
        </div>
        <div class="step">
          <div class="step-number">3</div>
          <div class="step-content">
            <h4>Add to Home Screen</h4>
            <p>Scroll down and tap "Add to Home Screen"</p>
          </div>
        </div>
        <div class="step">
          <div class="step-number">4</div>
          <div class="step-content">
            <h4>Confirm Installation</h4>
            <p>Tap "Add" to install the app to your home screen</p>
          </div>
        </div>
      </div>
      <div class="install-actions">
        <button class="btn btn-primary" onclick="this.closest('.modal').remove()">Got it!</button>
      </div>
    `);
    
    document.body.appendChild(modal);
    this.trackInstallation('ios', 'instructions_shown');
  }

  showGenericInstallInstructions() {
    const instructions = this.platform === 'android' ? 
      'To install: Open Chrome menu (⋮) → "Add to Home screen"' :
      'To install: Look for the install icon in your browser\'s address bar';
    
    const modal = this.createModal('Install App', `
      <div class="install-instructions generic">
        <p>${instructions}</p>
        <div class="install-actions">
          <button class="btn btn-primary" onclick="this.closest('.modal').remove()">OK</button>
        </div>
      </div>
    `);
    
    document.body.appendChild(modal);
  }

  showInstallPrompt() {
    // Only show if not already installed and not dismissed recently
    if (this.isStandalone || this.wasRecentlyDismissed()) {
      return;
    }

    const banner = document.createElement('div');
    banner.className = 'install-banner';
    banner.innerHTML = `
      <div class="install-banner-content">
        <div class="install-banner-text">
          <strong>Install KNIRV Controller</strong>
          <span>Get the full app experience</span>
        </div>
        <div class="install-banner-actions">
          <button class="btn btn-sm btn-primary" onclick="pwaInstaller.install()">Install</button>
          <button class="btn btn-sm btn-outline-secondary" onclick="pwaInstaller.dismissInstallBanner()">Later</button>
        </div>
      </div>
    `;
    
    document.body.appendChild(banner);
    this.installBanner = banner;
  }

  async install() {
    if (this.deferredPrompt) {
      await this.showAndroidInstallPrompt();
    } else {
      await this.promptInstallation();
    }
    
    this.dismissInstallBanner();
  }

  dismissInstallBanner() {
    if (this.installBanner) {
      this.installBanner.remove();
      this.installBanner = null;
    }
    
    // Remember dismissal for 24 hours
    localStorage.setItem('install_banner_dismissed', Date.now().toString());
  }

  wasRecentlyDismissed() {
    const dismissed = localStorage.getItem('install_banner_dismissed');
    if (!dismissed) return false;
    
    const dismissedTime = parseInt(dismissed);
    const dayInMs = 24 * 60 * 60 * 1000;
    
    return (Date.now() - dismissedTime) < dayInMs;
  }

  onInstallSuccess() {
    console.log('PWA installed successfully');
    this.installationState = 'installed';
    
    // Show success message
    this.showSuccessMessage();
    
    // Initialize app for first run
    this.initializeFirstRun();
    
    // Track successful installation
    this.trackInstallation(this.platform, 'installed');
  }

  showSuccessMessage() {
    const modal = this.createModal('Installation Complete!', `
      <div class="success-message">
        <div class="success-icon">✅</div>
        <h3>KNIRV Controller Installed</h3>
        <p>The app has been added to your home screen. You can now access it like any other app on your device.</p>
        <div class="success-actions">
          <button class="btn btn-primary" onclick="this.closest('.modal').remove(); pwaInstaller.startApp()">Start Using App</button>
        </div>
      </div>
    `);
    
    document.body.appendChild(modal);
  }

  async initializeUserDataDirectory() {
    try {
      // Initialize IndexedDB for user data storage
      if ('indexedDB' in window) {
        const dbName = 'KNIRVController';
        const dbVersion = 1;
        
        const request = indexedDB.open(dbName, dbVersion);
        
        request.onupgradeneeded = (event) => {
          const db = event.target.result;
          
          // Create object stores for user data
          if (!db.objectStoreNames.contains('userProfiles')) {
            db.createObjectStore('userProfiles', { keyPath: 'id' });
          }
          
          if (!db.objectStoreNames.contains('userSessions')) {
            db.createObjectStore('userSessions', { keyPath: 'id' });
          }
          
          if (!db.objectStoreNames.contains('userPreferences')) {
            db.createObjectStore('userPreferences', { keyPath: 'userId' });
          }
          
          if (!db.objectStoreNames.contains('walletData')) {
            db.createObjectStore('walletData', { keyPath: 'userId' });
          }
          
          if (!db.objectStoreNames.contains('agentData')) {
            db.createObjectStore('agentData', { keyPath: 'id' });
          }
        };
        
        request.onsuccess = () => {
          console.log('User data directory initialized');
          this.userDB = request.result;
        };
        
        request.onerror = (error) => {
          console.error('Failed to initialize user data directory:', error);
        };
      }
      
      // Initialize localStorage fallback
      if (!localStorage.getItem('knirv_user_data_initialized')) {
        localStorage.setItem('knirv_user_data_initialized', 'true');
        localStorage.setItem('knirv_installation_date', new Date().toISOString());
        localStorage.setItem('knirv_device_id', this.generateDeviceId());
      }
      
    } catch (error) {
      console.error('Failed to initialize user data directory:', error);
    }
  }

  async setupAuthentication() {
    // Initialize authentication service if available
    if (window.authenticationService) {
      try {
        await window.authenticationService.initialize();
        console.log('Authentication service initialized');
      } catch (error) {
        console.error('Failed to initialize authentication service:', error);
      }
    }
    
    // Setup biometric authentication if supported
    if ('credentials' in navigator && 'create' in navigator.credentials) {
      this.biometricAuthSupported = true;
    }
  }

  async initializeFirstRun() {
    const isFirstRun = !localStorage.getItem('knirv_first_run_complete');
    
    if (isFirstRun) {
      // Show welcome screen
      this.showWelcomeScreen();
      
      // Mark first run as complete
      localStorage.setItem('knirv_first_run_complete', 'true');
    }
  }

  showWelcomeScreen() {
    const modal = this.createModal('Welcome to KNIRV Controller', `
      <div class="welcome-screen">
        <div class="welcome-logo">
          <svg viewBox="0 0 100 100" width="80" height="80">
            <circle cx="50" cy="50" r="40" stroke="#2b56f5" fill="none" stroke-width="3"/>
            <text x="50" y="55" text-anchor="middle" fill="#2b56f5" font-size="14">CTRL</text>
          </svg>
        </div>
        <h3>Get Started</h3>
        <p>KNIRV Controller gives you complete control over your AI agents, wallet, and network operations.</p>
        <div class="welcome-features">
          <div class="feature">
            <span class="feature-icon">🤖</span>
            <span>Manage AI Agents</span>
          </div>
          <div class="feature">
            <span class="feature-icon">💰</span>
            <span>XION Wallet Integration</span>
          </div>
          <div class="feature">
            <span class="feature-icon">📊</span>
            <span>Network Monitoring</span>
          </div>
        </div>
        <div class="welcome-actions">
          <button class="btn btn-primary" onclick="this.closest('.modal').remove(); pwaInstaller.startApp()">Get Started</button>
        </div>
      </div>
    `);
    
    document.body.appendChild(modal);
  }

  startApp() {
    // Redirect to main app interface
    if (window.location.search.includes('install=')) {
      // Remove install parameter and reload
      const url = new URL(window.location);
      url.searchParams.delete('install');
      window.location.href = url.toString();
    }
  }

  generateDeviceId() {
    return 'device_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now();
  }

  createModal(title, content) {
    const modal = document.createElement('div');
    modal.className = 'modal fade show';
    modal.style.display = 'block';
    modal.innerHTML = `
      <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">${title}</h5>
            <button type="button" class="btn-close" onclick="this.closest('.modal').remove()"></button>
          </div>
          <div class="modal-body">
            ${content}
          </div>
        </div>
      </div>
      <div class="modal-backdrop fade show"></div>
    `;
    
    return modal;
  }

  showPlatformMismatchWarning(requestedPlatform) {
    const modal = this.createModal('Platform Mismatch', `
      <div class="platform-warning">
        <p>You requested installation for <strong>${requestedPlatform}</strong>, but you're using <strong>${this.platform}</strong>.</p>
        <p>Would you like to install for your current platform instead?</p>
        <div class="platform-actions">
          <button class="btn btn-primary" onclick="pwaInstaller.promptInstallation(); this.closest('.modal').remove()">Install for ${this.platform}</button>
          <button class="btn btn-outline-secondary" onclick="this.closest('.modal').remove()">Cancel</button>
        </div>
      </div>
    `);
    
    document.body.appendChild(modal);
  }

  showAlreadyInstalledMessage() {
    const modal = this.createModal('Already Installed', `
      <div class="already-installed">
        <div class="success-icon">✅</div>
        <p>KNIRV Controller is already installed on your device.</p>
        <div class="installed-actions">
          <button class="btn btn-primary" onclick="this.closest('.modal').remove()">Continue</button>
        </div>
      </div>
    `);
    
    document.body.appendChild(modal);
  }

  trackInstallation(platform, action) {
    // Track installation events for analytics
    const event = {
      type: 'pwa_installation',
      platform: platform,
      action: action,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent,
      url: window.location.href
    };
    
    // Store locally for later sync
    const events = JSON.parse(localStorage.getItem('knirv_analytics_events') || '[]');
    events.push(event);
    localStorage.setItem('knirv_analytics_events', JSON.stringify(events));
    
    console.log('Installation event tracked:', event);
  }
}

// Initialize PWA installer when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  window.pwaInstaller = new PWAInstaller();
});

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
  module.exports = PWAInstaller;
}
