import { app, BrowserWindow, screen, ipcMain, Tray, Menu, nativeImage, dialog } from 'electron';
import * as path from 'path';
import * as http from 'http';

let mainWindow: BrowserWindow | null = null;
let tray: Tray | null = null;

const FRONTEND_URL = process.env.KNIRV_SERVER_URL || 'http://localhost:8090';

function createTrayIcon(): Tray {
  // Build a simple 16x16 blue square icon from raw RGBA buffer
  const size = 16;
  const buf = Buffer.alloc(size * size * 4);
  for (let i = 0; i < size * size; i++) {
    buf[i * 4]     = 72;   // R
    buf[i * 4 + 1] = 136;  // G
    buf[i * 4 + 2] = 255;  // B
    buf[i * 4 + 3] = 255;  // A
  }
  const icon = nativeImage.createFromBuffer(buf, { width: size, height: size });

  const t = new Tray(icon);
  t.setToolTip('KNIRV Dashboard');
  t.setContextMenu(Menu.buildFromTemplate([
    {
      label: 'Open Dashboard',
      click: () => { mainWindow?.show(); tray?.destroy(); tray = null; },
    },
    { type: 'separator' },
    { label: 'Quit KNIRV', click: () => app.quit() },
  ]));
  t.on('click', () => { mainWindow?.show(); tray?.destroy(); tray = null; });
  return t;
}

function createWindow() {
  const { width, height } = screen.getPrimaryDisplay().workAreaSize;

  mainWindow = new BrowserWindow({
    width: width,
    height: height,
    transparent: true,
    frame: false,
    alwaysOnTop: false,
    skipTaskbar: false,
    resizable: true,
    webPreferences: {
      nodeIntegration: true,
      contextIsolation: false,
    },
  });

  mainWindow.setIgnoreMouseEvents(false);

  // index.html is copied into dist/ by the build script, so it lives alongside main.js
  const htmlPath = path.join(__dirname, 'index.html');
  mainWindow.loadFile(htmlPath);

  mainWindow.webContents.on('did-finish-load', () => {
    mainWindow?.webContents.send('set-frontend-url', FRONTEND_URL);
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

ipcMain.on('minimize-window', () => {
  mainWindow?.minimize();
});

function shutdownBackend(): Promise<void> {
  return new Promise((resolve) => {
    const token = process.env.KNIRV_SHUTDOWN_TOKEN;
    const port  = process.env.KNIRV_SERVER_PORT || '8090';
    if (!token) { resolve(); return; }
    const req = http.request(
      { hostname: 'localhost', port, path: '/shutdown', method: 'POST',
        headers: { 'X-Shutdown-Token': token } },
      () => resolve(),
    );
    req.on('error', () => resolve()); // server may already be down
    req.end();
  });
}

ipcMain.on('close-window', () => {
  if (!mainWindow) return;

  const choice = dialog.showMessageBoxSync(mainWindow, {
    type: 'question',
    buttons: ['Exit Application', 'Minimize to Tray', 'Cancel'],
    defaultId: 1,
    cancelId: 2,
    title: 'KNIRV Dashboard',
    message: 'Close KNIRV Dashboard?',
    detail: 'Exit the application fully, or keep it running in the system tray.',
  });

  if (choice === 0) {
    shutdownBackend().finally(() => app.quit());
  } else if (choice === 1) {
    tray = createTrayIcon();
    mainWindow.hide();
  }
  // choice === 2: cancel — do nothing
});

// Disable GPU/Vulkan acceleration — avoids driver warnings on headless/cloud systems.
// Both the API call and the command-line switch are needed: the switch suppresses
// GPU process initialisation (Vulkan ICDs) before the app 'ready' event fires.
app.disableHardwareAcceleration();
app.commandLine.appendSwitch('disable-gpu');
app.commandLine.appendSwitch('disable-software-rasterizer');
app.commandLine.appendSwitch('disable-vulkan');
app.commandLine.appendSwitch('disable-vulkan-surface');
app.commandLine.appendSwitch('disable-gpu-compositing');
app.commandLine.appendSwitch('use-gl=swiftshader');
app.commandLine.appendSwitch('enable-unsafe-swiftshader');

app.on('ready', createWindow);

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  if (mainWindow === null) {
    createWindow();
  }
});
