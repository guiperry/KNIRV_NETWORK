import { app, BrowserWindow, screen, ipcMain, Tray, Menu, nativeImage, dialog } from 'electron';
import * as path from 'path';
import * as http from 'http';
import * as fs from 'fs';
import * as urlMod from 'url';

let mainWindow: BrowserWindow | null = null;
let tray: Tray | null = null;
let menuServer: http.Server | null = null;

const FRONTEND_URL = process.env.KNIRV_SERVER_URL || 'http://localhost:8090';

// ─── Static MIME types for the menu file server ───────────────────────────────

function getMimeType(filePath: string): string {
  const ext = path.extname(filePath).toLowerCase();
  const types: Record<string, string> = {
    '.html': 'text/html; charset=utf-8',
    '.js':   'application/javascript',
    '.css':  'text/css',
    '.json': 'application/json',
    '.png':  'image/png',
    '.jpg':  'image/jpeg',
    '.jpeg': 'image/jpeg',
    '.gif':  'image/gif',
    '.svg':  'image/svg+xml',
    '.ico':  'image/x-icon',
    '.woff': 'font/woff',
    '.woff2':'font/woff2',
    '.ttf':  'font/ttf',
    '.otf':  'font/otf',
    '.map':  'application/json',
  };
  return types[ext] || 'application/octet-stream';
}

// ─── Local HTTP server for the built Next.js menu static files ────────────────

function startMenuServer(): Promise<string> {
  return new Promise((resolve, reject) => {
    const menuDir = path.join(__dirname, 'menu');

    if (!fs.existsSync(menuDir)) {
      reject(new Error(`Menu build not found: ${menuDir}\nRun "npm run build" first.`));
      return;
    }

    menuServer = http.createServer((req, res) => {
      const parsed = urlMod.parse(req.url || '/');
      let pathname  = parsed.pathname || '/';

      // Normalise to prevent directory traversal
      pathname = path.posix.normalize(pathname);
      if (pathname.startsWith('..')) pathname = '/';

      // Map root → index.html
      if (pathname === '/') pathname = '/index.html';

      let filePath = path.join(menuDir, pathname);

      // Next.js trailingSlash: try directory/index.html
      if (!fs.existsSync(filePath)) {
        const withIndex = path.join(filePath, 'index.html');
        if (fs.existsSync(withIndex)) {
          filePath = withIndex;
        } else {
          // SPA fallback
          filePath = path.join(menuDir, 'index.html');
        }
      }

      try {
        const content = fs.readFileSync(filePath);
        res.writeHead(200, {
          'Content-Type': getMimeType(filePath),
          'Cache-Control': 'no-cache',
          'Access-Control-Allow-Origin': '*',
        });
        res.end(content);
      } catch {
        res.writeHead(404);
        res.end('Not found');
      }
    });

    menuServer.listen(0, '127.0.0.1', () => {
      const addr = menuServer!.address() as { port: number };
      resolve(`http://127.0.0.1:${addr.port}`);
    });

    menuServer.on('error', reject);
  });
}

// ─── Tray ─────────────────────────────────────────────────────────────────────

function createTrayIcon(): Tray {
  const size = 16;
  const buf = Buffer.alloc(size * size * 4);
  for (let i = 0; i < size * size; i++) {
    buf[i * 4]     = 72;
    buf[i * 4 + 1] = 136;
    buf[i * 4 + 2] = 255;
    buf[i * 4 + 3] = 255;
  }
  const icon = nativeImage.createFromBuffer(buf, { width: size, height: size });
  const t = new Tray(icon);
  t.setToolTip('KNIRV Dashboard');
  t.setContextMenu(Menu.buildFromTemplate([
    { label: 'Open Dashboard', click: () => { mainWindow?.show(); tray?.destroy(); tray = null; } },
    { type: 'separator' },
    { label: 'Quit KNIRV', click: () => app.quit() },
  ]));
  t.on('click', () => { mainWindow?.show(); tray?.destroy(); tray = null; });
  return t;
}

// ─── Main window ──────────────────────────────────────────────────────────────

function createWindow() {
  const { width, height } = screen.getPrimaryDisplay().workAreaSize;

  mainWindow = new BrowserWindow({
    width,
    height,
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

  const htmlPath = path.join(__dirname, 'index.html');
  mainWindow.loadFile(htmlPath);

  // Send the backend/gateway URL so the renderer can use it for the login call
  mainWindow.webContents.on('did-finish-load', () => {
    mainWindow?.webContents.send('init-server-url', FRONTEND_URL);
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
    menuServer?.close();
    menuServer = null;
  });
}

// ─── IPC handlers ─────────────────────────────────────────────────────────────

ipcMain.on('minimize-window', () => {
  mainWindow?.minimize();
});

// Phase 1 → 2: user authenticated — start menu server and tell renderer to show menu
ipcMain.on('login-success', () => {
  startMenuServer()
    .then((menuUrl) => {
      mainWindow?.webContents.send('show-menu', { menuUrl });
    })
    .catch((err) => {
      console.error('Menu server failed, skipping menu phase:', err.message);
      // Fallback: skip menu and jump straight to the desktop frame
      mainWindow?.webContents.send('show-desktop', { frontendUrl: FRONTEND_URL });
    });
});

// Phase 2 → 3: menu animation done — tell renderer to show the HUD frame
ipcMain.on('menu-complete', () => {
  mainWindow?.webContents.send('show-desktop', { frontendUrl: FRONTEND_URL });
});

// ─── Shutdown ─────────────────────────────────────────────────────────────────

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
    req.on('error', () => resolve());
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
    shutdownBackend().finally(() => {
      menuServer?.close();
      app.quit();
    });
  } else if (choice === 1) {
    tray = createTrayIcon();
    mainWindow.hide();
  }
});

// ─── App lifecycle ────────────────────────────────────────────────────────────

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
    menuServer?.close();
    app.quit();
  }
});

app.on('activate', () => {
  if (mainWindow === null) createWindow();
});
