const { app, BrowserWindow, dialog } = require('electron');
const { spawn } = require('child_process');
const fs = require('fs');
const http = require('http');
const path = require('path');

const engineRoot = path.resolve(__dirname, '..', '..');
let backend;
let mainWindow;
let guiPortFile;

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

const startBackend = () => {
  const environment = { ...process.env, ELECTRON_MODE: 'true' };
  const configuredBinary = process.env.KNIRVENGINE_BACKEND;
  const packagedBinary = path.join(process.resourcesPath, process.platform === 'win32' ? 'knirv-engine.exe' : 'knirv-engine');
  const localBinary = path.join(engineRoot, process.platform === 'win32' ? 'knirv-engine.exe' : 'knirv-engine');
  const binary = configuredBinary || (app.isPackaged ? packagedBinary : localBinary);
  const backendArgs = ['--production', '--gui-port-file', guiPortFile];

  if (configuredBinary || app.isPackaged || require('fs').existsSync(localBinary)) {
    backend = spawn(binary, backendArgs, { cwd: engineRoot, env: environment, stdio: 'inherit' });
  } else {
    // Development fallback: build and run the Go host directly, while Electron
    // owns the user-facing window.
    backend = spawn('go', ['run', '.', ...backendArgs], { cwd: engineRoot, env: environment, stdio: 'inherit' });
  }

  backend.on('error', (error) => dialog.showErrorBox('KNIRVENGINE failed to start', error.message));
};

const createWindow = async () => {
  let guiPort;
  guiPortFile = path.join(app.getPath('temp'), `knirvengine-gui-port-${process.pid}`);
  fs.rmSync(guiPortFile, { force: true });
  startBackend();
  try {
    guiPort = await waitForGuiPort();
    await waitForGui(guiPort);
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
    webPreferences: { contextIsolation: true, nodeIntegration: false, sandbox: true },
  });
  await mainWindow.loadURL(`http://127.0.0.1:${guiPort}`);
};

app.whenReady().then(createWindow);
app.on('window-all-closed', () => app.quit());
app.on('before-quit', () => {
  if (backend && !backend.killed) backend.kill('SIGTERM');
  if (guiPortFile) fs.rmSync(guiPortFile, { force: true });
});
