const { contextBridge, ipcRenderer } = require('electron');

// Expose protected methods that allow the renderer process to use
// the ipcRenderer without exposing the entire object
contextBridge.exposeInMainWorld('electronAPI', {
  // App data management
  getAppDataPath: () => ipcRenderer.invoke('get-app-data-path'),
  
  // File system dialogs
  showSaveDialog: (options) => ipcRenderer.invoke('show-save-dialog', options),
  showOpenDialog: (options) => ipcRenderer.invoke('show-open-dialog', options),
  
  // Menu actions
  onMenuAction: (callback) => {
    ipcRenderer.on('menu-action', (event, action) => callback(action));
  },
  
  // Remove menu action listener
  removeMenuActionListener: () => {
    ipcRenderer.removeAllListeners('menu-action');
  },
  
  // Platform information
  platform: process.platform,
  
  // App information
  isElectron: true,

  // Asset operations
  getAssetPath: (filename) => ipcRenderer.invoke('get-asset-path', filename),
  
  // Notification API
  showNotification: (title, body, options = {}) => {
    if (Notification.permission === 'granted') {
      return new Notification(title, { body, ...options });
    } else if (Notification.permission !== 'denied') {
      Notification.requestPermission().then(permission => {
        if (permission === 'granted') {
          return new Notification(title, { body, ...options });
        }
      });
    }
  },
  
  // System information
  getSystemInfo: () => {
    return {
      platform: process.platform,
      arch: process.arch,
      version: process.version,
      electronVersion: process.versions.electron,
      chromeVersion: process.versions.chrome,
      nodeVersion: process.versions.node
    };
  },

  // System tray functionality
  minimizeToTray: () => ipcRenderer.invoke('minimize-to-tray'),
  showFromTray: () => ipcRenderer.invoke('show-from-tray'),
  isMinimizedToTray: () => ipcRenderer.invoke('is-minimized-to-tray'),
  showCloseDialog: () => ipcRenderer.invoke('show-close-dialog'),

  // Auto-updater functionality
  checkForUpdates: () => ipcRenderer.invoke('check-for-updates'),
  downloadUpdate: () => ipcRenderer.invoke('download-update'),
  quitAndInstall: () => ipcRenderer.invoke('quit-and-install'),
  getAppVersion: () => ipcRenderer.invoke('get-app-version'),

  // Auto-updater events
  onUpdateAvailable: (callback) => {
    ipcRenderer.on('update-available', (event, info) => callback(info));
  },
  onUpdateDownloaded: (callback) => {
    ipcRenderer.on('update-downloaded', (event, info) => callback(info));
  },
  onDownloadProgress: (callback) => {
    ipcRenderer.on('download-progress', (event, progress) => callback(progress));
  },
  onUpdateError: (callback) => {
    ipcRenderer.on('update-error', (event, error) => callback(error));
  }
});

// Expose a limited set of Node.js APIs for specific use cases
contextBridge.exposeInMainWorld('nodeAPI', {
  // Path utilities (safe subset)
  path: {
    join: (...args) => require('path').join(...args),
    basename: (path) => require('path').basename(path),
    dirname: (path) => require('path').dirname(path),
    extname: (path) => require('path').extname(path)
  },
  
  // OS utilities (safe subset)
  os: {
    platform: () => require('os').platform(),
    arch: () => require('os').arch(),
    homedir: () => require('os').homedir(),
    tmpdir: () => require('os').tmpdir()
  }
});

// Security: Remove any Node.js globals that might have been exposed
delete window.require;
delete window.exports;
delete window.module;

// Add security headers and CSP
window.addEventListener('DOMContentLoaded', () => {
  // Add security meta tags if they don't exist
  if (!document.querySelector('meta[http-equiv="Content-Security-Policy"]')) {
    const cspMeta = document.createElement('meta');
    cspMeta.setAttribute('http-equiv', 'Content-Security-Policy');
    cspMeta.setAttribute('content',
      "default-src 'self' 'unsafe-inline' 'unsafe-eval' data: blob: https://authkit.picaos.com; " +
      "connect-src 'self' ws://localhost:* http://localhost:* https://api.cerebras.ai https://authkit.picaos.com; " +
      "img-src 'self' data: blob: https:; " +
      "font-src 'self' data:; " +
      "style-src 'self' 'unsafe-inline';"
    );
    document.head.appendChild(cspMeta);
  }
});

// Handle uncaught errors
window.addEventListener('error', (event) => {
  console.error('Uncaught error:', event.error);
});

window.addEventListener('unhandledrejection', (event) => {
  console.error('Unhandled promise rejection:', event.reason);
});

// Prevent navigation to external URLs
window.addEventListener('beforeunload', (event) => {
  // Allow the app to handle its own navigation
  return undefined;
});

// Disable drag and drop of files to prevent security issues
document.addEventListener('dragover', (event) => {
  event.preventDefault();
});

document.addEventListener('drop', (event) => {
  event.preventDefault();
});

// Secure TEE Bridge API
contextBridge.exposeInMainWorld('teeBridge', {
  // Session management
  createSession: async (clientId, permissions) => {
    return ipcRenderer.invoke('tee-create-session', clientId, permissions);
  },

  revokeSession: async (token) => {
    return ipcRenderer.invoke('tee-revoke-session', token);
  },

  // Plugin management
  loadPlugin: async (token, pluginId, securityContext) => {
    return ipcRenderer.invoke('tee-load-plugin', token, pluginId, securityContext);
  },

  unloadPlugin: async (token, pluginId) => {
    return ipcRenderer.invoke('tee-unload-plugin', token, pluginId);
  },

  executeInPlugin: async (token, pluginId, command, args) => {
    return ipcRenderer.invoke('tee-execute-plugin', token, pluginId, command, args);
  },

  listPlugins: async (token) => {
    return ipcRenderer.invoke('tee-list-plugins', token);
  },

  getPluginInfo: async (token, pluginId) => {
    return ipcRenderer.invoke('tee-get-plugin-info', token, pluginId);
  },

  // TEE status
  getTEEStatus: async (token) => {
    return ipcRenderer.invoke('tee-get-status', token);
  },

  // WebSocket connection for real-time updates
  connectSecureWebSocket: (token, onMessage, onError, onClose) => {
    const ws = new WebSocket(`ws://localhost:8081/api/v1/desktop/secure-ws`);

    ws.onopen = () => {
      // Send authentication message
      ws.send(JSON.stringify({
        type: 'auth',
        token: token,
        timestamp: new Date().toISOString()
      }));
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (onMessage) onMessage(data);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    ws.onerror = (error) => {
      if (onError) onError(error);
    };

    ws.onclose = (event) => {
      if (onClose) onClose(event);
    };

    return {
      send: (message) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify(message));
        }
      },
      close: () => ws.close(),
      readyState: () => ws.readyState
    };
  }
});

// Add development helpers
if (process.env.NODE_ENV === 'development') {
  window.electronAPI.isDevelopment = true;

  // Add development console commands
  window.devTools = {
    reloadApp: () => location.reload(),
    openDevTools: () => {
      // This would need to be implemented via IPC if needed
      console.log('DevTools toggle requested');
    }
  };
}
