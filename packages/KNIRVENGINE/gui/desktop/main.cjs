const { app, BrowserWindow, dialog, ipcMain } = require('electron');
const { spawn } = require('child_process');
const fs = require('fs');
const http = require('http');
const path = require('path');

const engineRoot = path.resolve(__dirname, '..', '..');
let backend;
let mainWindow;
let guiPortFile;
let shuttingDown = false;
let selectedProjectPath = '';

const resolveSelectedProjectFile = (relativePath) => {
  if (!selectedProjectPath || typeof relativePath !== 'string') throw new Error('No project directory is selected.');
  const root = path.resolve(selectedProjectPath);
  const candidate = path.resolve(root, relativePath);
  if (candidate !== root && !candidate.startsWith(`${root}${path.sep}`)) throw new Error('File is outside the selected project.');
  return candidate;
};

const stopBackend = (signal = 'SIGTERM') => {
  if (!backend || backend.exitCode !== null || backend.signalCode !== null) return;
  try {
    // The engine can spawn a GUI child in development mode. It owns a separate
    // process group so one desktop shutdown signal reaches the whole tree.
    if (process.platform !== 'win32') process.kill(-backend.pid, signal);
    else backend.kill(signal);
  } catch (error) {
    if (error.code !== 'ESRCH') console.error(`Failed to send ${signal} to backend:`, error);
  }
};

const requestShutdown = () => {
  if (shuttingDown) return;
  shuttingDown = true;
  stopBackend();
  if (mainWindow && !mainWindow.isDestroyed()) mainWindow.close();
  app.quit();
  const forceTimer = setTimeout(() => {
    stopBackend('SIGKILL');
    app.exit(0);
  }, 5000);
  forceTimer.unref();
};

ipcMain.handle('knirv:select-sandbox-binary', async (_event, defaultPath) => {
  const result = await dialog.showOpenDialog(mainWindow, {
    title: 'Select sandbox target binary',
    buttonLabel: 'Use selected binary',
    defaultPath: typeof defaultPath === 'string' && defaultPath ? defaultPath : undefined,
    properties: ['openFile'],
    filters: [{ name: 'All files', extensions: ['*'] }],
  });
  return result.canceled ? null : result.filePaths[0] || null;
});

ipcMain.handle('knirv:select-sandbox-project', async () => {
  const result = await dialog.showOpenDialog(mainWindow, {
    title: 'Select target project folder',
    buttonLabel: 'Use selected project',
    properties: ['openDirectory'],
  });
  selectedProjectPath = result.canceled ? '' : result.filePaths[0] || '';
  return selectedProjectPath || null;
});

ipcMain.handle('knirv:list-sandbox-project-files', async () => {
  if (!selectedProjectPath) return [];
  const files = [];
  const visit = async (relativeDir = '') => {
    const directory = resolveSelectedProjectFile(relativeDir || '.');
    const entries = await fs.promises.readdir(directory, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.name === 'node_modules' || entry.name === '.git' || entry.name === 'dist') continue;
      const relativePath = relativeDir ? path.join(relativeDir, entry.name) : entry.name;
      if (entry.isDirectory()) await visit(relativePath);
      else if (entry.isFile()) {
        const info = await fs.promises.stat(resolveSelectedProjectFile(relativePath));
        files.push({ name: entry.name, path: relativePath, size: info.size });
      }
    }
  };
  await visit();
  return files;
});

ipcMain.handle('knirv:read-sandbox-project-file', async (_event, relativePath) => {
  const filePath = resolveSelectedProjectFile(relativePath);
  return fs.promises.readFile(filePath, 'utf8');
});

const waitForGuiPort = (attempts = 60) => new Promise((resolve, reject) => {
  const check = (remaining) => {
    fs.readFile(guiPortFile, 'utf8', (error, value) => {
      const port = Number(value?.trim());
      if (!error && Number.isInteger(port) && port > 0 && port < 65536) resolve(port);
      else if (remaining <= 0) reject(new Error('KNIRVENGINE did not report its selected GUI port.'));
      else setTimeout(() => check(remaining - 1), 500);
    });
  };
  check(attempts);
});

const waitForGui = (guiPort, attempts = 60) => new Promise((resolve, reject) => {
  const check = (remaining) => {
    const request = http.get(`http://127.0.0.1:${guiPort}`, (response) => {
      response.resume();
      if (response.statusCode && response.statusCode < 500) resolve();
      else retry(remaining);
    });
    request.on('error', () => retry(remaining));
    request.setTimeout(1000, () => request.destroy());
  };
  const retry = (remaining) => {
    if (remaining <= 0) reject(new Error(`KNIRVENGINE did not start on port ${guiPort}.`));
    else setTimeout(() => check(remaining - 1), 500);
  };
  check(attempts);
});

const waitForApiThroughGui = (guiPort, attempts = 60) => new Promise((resolve, reject) => {
  const check = (remaining) => {
    const request = http.get(`http://127.0.0.1:${guiPort}/api/v1/health`, (response) => {
      response.resume();
      if (response.statusCode === 200) resolve();
      else retry(remaining);
    });
    request.on('error', () => retry(remaining));
    request.setTimeout(1000, () => request.destroy());
  };
  const retry = (remaining) => {
    if (remaining <= 0) reject(new Error('KNIRVENGINE API did not become healthy.'));
    else setTimeout(() => check(remaining - 1), 500);
  };
  check(attempts);
});

const startBackend = () => {
  const environment = { ...process.env, ELECTRON_MODE: 'true' };
  const configuredBinary = process.env.KNIRVENGINE_BACKEND;
  const packagedBinary = path.join(process.resourcesPath, process.platform === 'win32' ? 'knirv-engine.exe' : 'knirv-engine');
  const localBinary = path.join(engineRoot, process.platform === 'win32' ? 'knirv-engine.exe' : 'knirv-engine');
  const binary = configuredBinary || (app.isPackaged ? packagedBinary : localBinary);
  const backendArgs = ['--production', '--gui-port-file', guiPortFile];

  if (configuredBinary || app.isPackaged || require('fs').existsSync(localBinary)) {
    backend = spawn(binary, backendArgs, { cwd: engineRoot, env: environment, stdio: 'inherit', detached: process.platform !== 'win32' });
  } else {
    // Development fallback: build and run the Go host directly, while Electron
    // owns the user-facing window.
    backend = spawn('go', ['run', '.', ...backendArgs], { cwd: engineRoot, env: environment, stdio: 'inherit', detached: process.platform !== 'win32' });
  }

  backend.on('error', (error) => dialog.showErrorBox('KNIRVENGINE failed to start', error.message));
  backend.on('exit', () => { backend = null; });
};

const createWindow = async () => {
  let guiPort;
  guiPortFile = path.join(app.getPath('temp'), `knirvengine-gui-port-${process.pid}`);
  fs.rmSync(guiPortFile, { force: true });
  startBackend();
  try {
    guiPort = await waitForGuiPort();
    await waitForGui(guiPort);
    await waitForApiThroughGui(guiPort);
  } catch (error) {
    dialog.showErrorBox('KNIRVENGINE failed to start', error.message);
    app.quit();
    return;
  }

  mainWindow = new BrowserWindow({
    width: 1440,
    height: 920,
    minWidth: 960,
    minHeight: 640,
    title: 'KNIRVENGINE',
    backgroundColor: '#020617',
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      preload: path.join(__dirname, 'preload.cjs'),
    },
  });
  await mainWindow.loadURL(`http://127.0.0.1:${guiPort}`);
};

app.whenReady().then(createWindow);
app.on('window-all-closed', () => app.quit());
app.on('before-quit', () => {
  stopBackend();
  if (guiPortFile) fs.rmSync(guiPortFile, { force: true });
});
process.on('SIGINT', requestShutdown);
process.on('SIGTERM', requestShutdown);
