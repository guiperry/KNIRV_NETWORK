const { app, BrowserWindow, Menu, ipcMain, dialog, shell, protocol, Tray, nativeImage } = require('electron');
const { autoUpdater } = require('electron-updater');
const path = require('path');
const { spawn } = require('child_process');
const fs = require('fs');
const os = require('os');

// Keep a global reference of the window object
let mainWindow;
let backendProcess;
let isQuitting = false;
let tray = null;
let isMinimizedToTray = false;

// Configure auto-updater
autoUpdater.checkForUpdatesAndNotify();

// Auto-updater event handlers
autoUpdater.on('checking-for-update', () => {
  console.log('Checking for update...');
});

autoUpdater.on('update-available', (info) => {
  console.log('Update available:', info);
  if (mainWindow) {
    mainWindow.webContents.send('update-available', info);
  }
});

autoUpdater.on('update-not-available', (info) => {
  console.log('Update not available:', info);
});

autoUpdater.on('error', (err) => {
  console.error('Error in auto-updater:', err);
  if (mainWindow) {
    mainWindow.webContents.send('update-error', err.message);
  }
});

autoUpdater.on('download-progress', (progressObj) => {
  let log_message = "Download speed: " + progressObj.bytesPerSecond;
  log_message = log_message + ' - Downloaded ' + progressObj.percent + '%';
  log_message = log_message + ' (' + progressObj.transferred + "/" + progressObj.total + ')';
  console.log(log_message);

  if (mainWindow) {
    mainWindow.webContents.send('download-progress', progressObj);
  }
});

autoUpdater.on('update-downloaded', (info) => {
  console.log('Update downloaded:', info);
  if (mainWindow) {
    mainWindow.webContents.send('update-downloaded', info);
  }

  // Show dialog to user
  dialog.showMessageBox(mainWindow, {
    type: 'info',
    title: 'Update Ready',
    message: 'Update downloaded. The application will restart to apply the update.',
    buttons: ['Restart Now', 'Later']
  }).then((result) => {
    if (result.response === 0) {
      autoUpdater.quitAndInstall();
    }
  });
});

// Get app data directory based on OS
// This MUST match the Go backend's utils/appdata.go GetAppDataDir() function
function getAppDataPath() {
  const appName = 'Agentic-Engine'; // Keep consistent with Go backend

  switch (process.platform) {
    case 'win32':
      // Use APPDATA environment variable to match Go backend
      const appData = process.env.APPDATA;
      if (!appData) {
        throw new Error('APPDATA environment variable not set');
      }
      return path.join(appData, appName);
    case 'darwin':
      return path.join(os.homedir(), 'Library', 'Application Support', appName);
    case 'linux':
    default:
      // Use XDG config directory to match Go backend
      const configDir = process.env.XDG_CONFIG_HOME || path.join(os.homedir(), '.config');
      return path.join(configDir, appName);
  }
}

// Ensure app data directory exists
// This MUST match the Go backend's directory structure from utils/appdata.go
function ensureAppDataDirectory() {
  const appDataPath = getAppDataPath();

  // Create main app data directory
  if (!fs.existsSync(appDataPath)) {
    fs.mkdirSync(appDataPath, { recursive: true });
  }

  // Create subdirectories to match Go backend structure
  // See utils/appdata.go for the complete list
  const subdirs = [
    'data',           // GetDatabaseDir
    'config',         // GetConfigDir
    'mcp',            // GetMCPDir (NEW: unified MCP directory)
    'mcp/config',     // GetMCPConfigDir
    'mcp/servers',    // GetMCPServersDir
    'mcp/data',       // GetMCPDataDir
    'mcp/logs',       // GetMCPLogsDir
    'mcp/monitoring', // GetMCPMonitoringDir
    'plugins',        // GetPluginsDir
    'plugins/data',   // GetPluginDataDir
    'logs',           // GetLogsDir
    'quarantine',     // GetQuarantineDir
    'backups',        // GetBackupDir
    'security'        // GetSecurityDir
  ];

  subdirs.forEach(subdir => {
    const subdirPath = path.join(appDataPath, subdir);
    if (!fs.existsSync(subdirPath)) {
      fs.mkdirSync(subdirPath, { recursive: true });
    }
  });

  return appDataPath;
}



// Start the Go backend server
function startBackendServer() {
  return new Promise((resolve, reject) => {
    const appDataPath = getAppDataPath();
    
    // Determine the backend executable path
    let backendPath;
    if (app.isPackaged) {
      // In production, the backend should be bundled with the app
      backendPath = path.join(process.resourcesPath, 'backend', 'knirv-engine');
      if (process.platform === 'win32') {
        backendPath += '.exe';
      }
    } else {
      // In development, use the backend from the project root
      backendPath = path.join(__dirname, '..', 'knirv-engine');
      if (process.platform === 'win32') {
        backendPath += '.exe';
      }
    }
    
    // Check if backend executable exists
    if (!fs.existsSync(backendPath)) {
      console.error('Backend executable not found at:', backendPath);
      reject(new Error(`Backend executable not found at: ${backendPath}`));
      return;
    }
    
    // Set environment variables for the backend
    const env = {
      ...process.env,
      APP_DATA_PATH: appDataPath,
      GUI_PORT: '3001', // Use different port for Electron
      SERVER_PORT: '8081',
      ELECTRON_MODE: 'true' // Tell backend it's running in Electron
    };
    
    // Start the backend process
    console.log('Starting backend server:', backendPath);
    const args = app.isPackaged ? ['--production'] : [];
    backendProcess = spawn(backendPath, args, {
      env,
      stdio: ['pipe', 'pipe', 'pipe'],
      detached: false
    });
    
    backendProcess.stdout.on('data', (data) => {
      console.log('Backend stdout:', data.toString());
    });
    
    backendProcess.stderr.on('data', (data) => {
      console.error('Backend stderr:', data.toString());
    });
    
    backendProcess.on('error', (error) => {
      console.error('Backend process error:', error);
      reject(error);
    });
    
    backendProcess.on('exit', (code, signal) => {
      console.log(`Backend process exited with code ${code} and signal ${signal}`);
      if (!isQuitting) {
        // Backend crashed unexpectedly
        dialog.showErrorBox('Backend Error', 'The backend server has stopped unexpectedly. The application will close.');
        app.quit();
      }
    });
    
    // Wait a moment for the server to start
    setTimeout(() => {
      resolve();
    }, 2000);
  });
}

// Stop the backend server
function stopBackendServer() {
  if (backendProcess && !backendProcess.killed) {
    console.log('Stopping backend server...');
    backendProcess.kill('SIGTERM');
    
    // Force kill after 5 seconds if it doesn't stop gracefully
    setTimeout(() => {
      if (backendProcess && !backendProcess.killed) {
        console.log('Force killing backend server...');
        backendProcess.kill('SIGKILL');
      }
    }, 5000);
  }
}

// Create system tray
function createTray() {
  // Create tray icon
  const iconPath = path.join(__dirname, 'assets', 'icon.png');
  let trayIcon;

  try {
    trayIcon = nativeImage.createFromPath(iconPath);
    // Resize icon for tray (16x16 on most platforms)
    trayIcon = trayIcon.resize({ width: 16, height: 16 });
  } catch (error) {
    console.error('Failed to load tray icon:', error);
    // Create a simple fallback icon
    trayIcon = nativeImage.createEmpty();
  }

  tray = new Tray(trayIcon);

  // Set tooltip
  tray.setToolTip('KNIRVENGINE - NFT-Agent Platform');

  // Create context menu
  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'Show KNIRVENGINE',
      click: () => {
        showMainWindow();
      }
    },
    {
      label: 'Hide to Tray',
      click: () => {
        hideToTray();
      }
    },
    { type: 'separator' },
    {
      label: 'New Agent',
      click: () => {
        showMainWindow();
        mainWindow.webContents.send('menu-action', 'new-agent');
      }
    },
    {
      label: 'Import Plugin',
      click: () => {
        showMainWindow();
        mainWindow.webContents.send('menu-action', 'import-plugin');
      }
    },
    { type: 'separator' },
    {
      label: 'Settings',
      click: () => {
        showMainWindow();
        mainWindow.webContents.send('menu-action', 'settings');
      }
    },
    { type: 'separator' },
    {
      label: 'Quit KNIRVENGINE',
      click: () => {
        isQuitting = true;
        app.quit();
      }
    }
  ]);

  tray.setContextMenu(contextMenu);

  // Handle double-click to show/hide window
  tray.on('double-click', () => {
    if (mainWindow.isVisible()) {
      hideToTray();
    } else {
      showMainWindow();
    }
  });

  // Handle single click on Windows/Linux
  if (process.platform !== 'darwin') {
    tray.on('click', () => {
      if (mainWindow.isVisible()) {
        hideToTray();
      } else {
        showMainWindow();
      }
    });
  }
}

// Show main window
function showMainWindow() {
  if (mainWindow) {
    if (mainWindow.isMinimized()) {
      mainWindow.restore();
    }
    mainWindow.show();
    mainWindow.focus();
    isMinimizedToTray = false;

    // On macOS, show dock icon
    if (process.platform === 'darwin') {
      app.dock.show();
    }
  }
}

// Hide to tray
function hideToTray() {
  if (mainWindow) {
    mainWindow.hide();
    isMinimizedToTray = true;

    // On macOS, hide dock icon
    if (process.platform === 'darwin') {
      app.dock.hide();
    }

    // Show notification on first hide
    if (tray && !isMinimizedToTray) {
      tray.displayBalloon({
        iconType: 'info',
        title: 'KNIRVENGINE',
        content: 'Application was minimized to tray. Click the tray icon to restore.'
      });
    }
  }
}

// Show close confirmation dialog
async function showCloseDialog() {
  const choice = await dialog.showMessageBox(mainWindow, {
    type: 'question',
    buttons: ['Minimize to Tray', 'Quit Application', 'Cancel'],
    defaultId: 0,
    cancelId: 2,
    title: 'Close KNIRVENGINE',
    message: 'What would you like to do?',
    detail: 'You can minimize the application to the system tray to keep it running in the background, or quit the application completely.',
    icon: path.join(__dirname, 'assets', 'icon.png')
  });

  return choice.response;
}

function createWindow() {
  // Create the browser window
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1200,
    minHeight: 700,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      enableRemoteModule: false,
      webSecurity: false, // Allow file:// protocol access for assets
      preload: path.join(__dirname, 'preload.js')
    },
    icon: path.join(__dirname, 'assets', 'icon.png'), // Add app icon
    titleBarStyle: process.platform === 'darwin' ? 'hiddenInset' : 'default',
    show: false // Don't show until ready
  });

  // Load the app
  if (app.isPackaged) {
    // In production, load the built files from extraFiles (at app root level)
    // extraFiles are placed at the same level as the app executable
    const appDir = path.dirname(process.execPath);
    const indexPath = path.join(appDir, 'gui', 'dist', 'index.html');
    console.log('Loading frontend from:', indexPath);
    mainWindow.loadFile(indexPath);
  } else {
    // In development, load from Vite dev server
    mainWindow.loadURL('http://localhost:3001');

    // Open DevTools in development
    mainWindow.webContents.openDevTools();
  }

  // Show window when ready
  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
    
    // Focus on the window
    if (process.platform === 'darwin') {
      app.dock.show();
    }
  });

  // Handle window close event (X button)
  mainWindow.on('close', async (event) => {
    if (!isQuitting) {
      event.preventDefault();

      try {
        const choice = await showCloseDialog();

        switch (choice) {
          case 0: // Minimize to Tray
            hideToTray();
            break;
          case 1: // Quit Application
            isQuitting = true;
            app.quit();
            break;
          case 2: // Cancel
          default:
            // Do nothing, keep window open
            break;
        }
      } catch (error) {
        console.error('Error showing close dialog:', error);
        // Fallback: minimize to tray
        hideToTray();
      }
    }
  });

  // Handle window closed
  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // Handle external links
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: 'deny' };
  });
}

// Create application menu
function createMenu() {
  const template = [
    {
      label: 'File',
      submenu: [
        {
          label: 'New Agent',
          accelerator: 'CmdOrCtrl+N',
          click: () => {
            mainWindow.webContents.send('menu-action', 'new-agent');
          }
        },
        {
          label: 'Import Plugin',
          accelerator: 'CmdOrCtrl+I',
          click: () => {
            mainWindow.webContents.send('menu-action', 'import-plugin');
          }
        },
        { type: 'separator' },
        {
          label: 'Settings',
          accelerator: 'CmdOrCtrl+,',
          click: () => {
            mainWindow.webContents.send('menu-action', 'settings');
          }
        },
        { type: 'separator' },
        {
          label: 'Quit',
          accelerator: process.platform === 'darwin' ? 'Cmd+Q' : 'Ctrl+Q',
          click: () => {
            app.quit();
          }
        }
      ]
    },
    {
      label: 'View',
      submenu: [
        { role: 'reload' },
        { role: 'forceReload' },
        { role: 'toggleDevTools' },
        { type: 'separator' },
        { role: 'resetZoom' },
        { role: 'zoomIn' },
        { role: 'zoomOut' },
        { type: 'separator' },
        { role: 'togglefullscreen' }
      ]
    },
    {
      label: 'Window',
      submenu: [
        { role: 'minimize' },
        { role: 'close' }
      ]
    },
    {
      label: 'Help',
      submenu: [
        {
          label: 'About KNIRVENGINE',
          click: () => {
            dialog.showMessageBox(mainWindow, {
              type: 'info',
              title: 'About KNIRVENGINE',
              message: 'KNIRVENGINE',
              detail: 'NFT-Agent Platform for autonomous agent deployment and management.'
            });
          }
        },
        {
          label: 'Documentation',
          click: () => {
            shell.openExternal('https://github.com/your-repo/knirv-engine');
          }
        }
      ]
    }
  ];

  // macOS specific menu adjustments
  if (process.platform === 'darwin') {
    template.unshift({
      label: app.getName(),
      submenu: [
        { role: 'about' },
        { type: 'separator' },
        { role: 'services' },
        { type: 'separator' },
        { role: 'hide' },
        { role: 'hideOthers' },
        { role: 'unhide' },
        { type: 'separator' },
        { role: 'quit' }
      ]
    });

    // Window menu
    template[3].submenu = [
      { role: 'close' },
      { role: 'minimize' },
      { role: 'zoom' },
      { type: 'separator' },
      { role: 'front' }
    ];
  }

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

// Disable hardware acceleration to prevent Vulkan warnings
app.disableHardwareAcceleration();

// Alternative: Add command line switches to disable Vulkan (uncomment if you want to keep hardware acceleration)
// app.commandLine.appendSwitch('disable-vulkan');
// app.commandLine.appendSwitch('disable-gpu-sandbox');
// app.commandLine.appendSwitch('disable-software-rasterizer');

// Set custom userData path to match Go backend before app is ready
app.setPath('userData', getAppDataPath());

// Register custom protocol for serving assets
app.whenReady().then(async () => {
  // Register custom protocol for assets
  protocol.registerFileProtocol('app-asset', (request, callback) => {
    const url = request.url.substr(11); // Remove 'app-asset://' prefix
    let assetPath;

    if (url === 'Agentify_logo_2.png') {
      // Serve the logo from the assets directory
      assetPath = path.join(__dirname, 'assets', 'Agentify_logo_2.png');
    } else {
      // Serve other assets from gui/dist
      assetPath = path.join(__dirname, 'gui', 'dist', url);
    }

    callback({ path: assetPath });
  });

  try {
    // Ensure app data directory exists
    ensureAppDataDirectory();

    // Start backend server
    await startBackendServer();

    // Create window, menu, and tray
    createWindow();
    createMenu();
    createTray();

    console.log('KNIRVENGINE desktop app started successfully');
  } catch (error) {
    console.error('Failed to start application:', error);
    dialog.showErrorBox('Startup Error', `Failed to start the application: ${error.message}`);
    app.quit();
  }
});

app.on('window-all-closed', () => {
  // Don't quit when all windows are closed if we're using system tray
  if (process.platform !== 'darwin' && !isMinimizedToTray) {
    app.quit();
  }
});

app.on('activate', () => {
  // On macOS, re-create window when dock icon is clicked
  if (BrowserWindow.getAllWindows().length === 0) {
    createWindow();
  } else if (mainWindow) {
    showMainWindow();
  }
});

app.on('before-quit', () => {
  isQuitting = true;

  // Clean up tray
  if (tray) {
    tray.destroy();
    tray = null;
  }

  stopBackendServer();
});

// TEE Bridge IPC handlers
let teeSessionToken = null;

ipcMain.handle('tee-create-session', async (event, clientId, permissions) => {
  try {
    // In a real implementation, this would communicate with the Go backend
    // For now, we'll simulate a session token
    teeSessionToken = 'desktop-session-' + Math.random().toString(36).substring(7);
    return { success: true, token: teeSessionToken };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('tee-revoke-session', async (event, token) => {
  try {
    if (token === teeSessionToken) {
      teeSessionToken = null;
      return { success: true };
    }
    return { success: false, error: 'Invalid token' };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('tee-load-plugin', async (event, token, pluginId, securityContext) => {
  try {
    if (token !== teeSessionToken) {
      return { success: false, error: 'Invalid token' };
    }

    // In a real implementation, this would communicate with the Go backend
    // For now, we'll simulate plugin loading
    console.log('Loading plugin:', pluginId, 'with context:', securityContext);
    return { success: true, pluginId, status: 'loaded' };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('tee-unload-plugin', async (event, token, pluginId) => {
  try {
    if (token !== teeSessionToken) {
      return { success: false, error: 'Invalid token' };
    }

    console.log('Unloading plugin:', pluginId);
    return { success: true, pluginId, status: 'unloaded' };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('tee-execute-plugin', async (event, token, pluginId, command, args) => {
  try {
    if (token !== teeSessionToken) {
      return { success: false, error: 'Invalid token' };
    }

    console.log('Executing in plugin:', pluginId, 'command:', command, 'args:', args);
    return {
      success: true,
      stdout: 'Command executed successfully',
      stderr: '',
      exitCode: 0
    };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('tee-list-plugins', async (event, token) => {
  try {
    if (token !== teeSessionToken) {
      return { success: false, error: 'Invalid token' };
    }

    // Return mock plugin list
    return {
      success: true,
      plugins: [
        {
          id: 'plugin-1',
          name: 'Sample Plugin',
          version: '1.0.0',
          isVerified: true,
          teeType: 'process'
        }
      ]
    };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('tee-get-plugin-info', async (event, token, pluginId) => {
  try {
    if (token !== teeSessionToken) {
      return { success: false, error: 'Invalid token' };
    }

    return {
      success: true,
      plugin: {
        id: pluginId,
        name: 'Sample Plugin',
        version: '1.0.0',
        isVerified: true,
        teeType: 'process',
        loadedAt: new Date().toISOString()
      }
    };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('tee-get-status', async (event, token) => {
  try {
    if (token !== teeSessionToken) {
      return { success: false, error: 'Invalid token' };
    }

    return {
      success: true,
      status: {
        activePlugins: 1,
        totalPlugins: 1,
        memoryUsage: '256MB',
        cpuUsage: '15%'
      }
    };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

// Standard IPC handlers
ipcMain.handle('get-app-data-path', () => {
  return getAppDataPath();
});

ipcMain.handle('show-save-dialog', async (event, options) => {
  const result = await dialog.showSaveDialog(mainWindow, options);
  return result;
});

ipcMain.handle('show-open-dialog', async (event, options) => {
  const result = await dialog.showOpenDialog(mainWindow, options);
  return result;
});

ipcMain.handle('get-asset-path', (event, filename) => {
  const assetPath = path.join(__dirname, 'assets', filename);
  return `file://${assetPath}`;
});

// Auto-updater IPC handlers
ipcMain.handle('check-for-updates', async () => {
  try {
    const result = await autoUpdater.checkForUpdates();
    return { success: true, updateInfo: result };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('download-update', async () => {
  try {
    await autoUpdater.downloadUpdate();
    return { success: true };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('quit-and-install', () => {
  autoUpdater.quitAndInstall();
});

ipcMain.handle('get-app-version', () => {
  return app.getVersion();
});

// System tray IPC handlers
ipcMain.handle('minimize-to-tray', () => {
  hideToTray();
  return { success: true };
});

ipcMain.handle('show-from-tray', () => {
  showMainWindow();
  return { success: true };
});

ipcMain.handle('is-minimized-to-tray', () => {
  return isMinimizedToTray;
});

ipcMain.handle('show-close-dialog', async () => {
  try {
    const choice = await showCloseDialog();
    return { success: true, choice };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

// Handle app protocol for deep linking (optional)
app.setAsDefaultProtocolClient('knirv-engine');
