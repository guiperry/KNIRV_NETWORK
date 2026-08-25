const { contextBridge, ipcRenderer, webUtils } = require('electron');

// Expose the smallest possible native surface to the renderer. The sandbox
// picker returns only the path the user deliberately selected.
contextBridge.exposeInMainWorld('electronAPI', {
  isElectron: true,
  // Static GUI assets are served by the engine's GUI host. Keep these legacy
  // helpers so existing renderer code can use the secure bridge unchanged.
  getAssetPath: (filename) => filename === 'knirv-logo.png' || filename === 'agent.png' ? `/${filename}` : '',
  selectSandboxBinary: (defaultPath) => ipcRenderer.invoke('knirv:select-sandbox-binary', defaultPath),
  selectSandboxProject: () => ipcRenderer.invoke('knirv:select-sandbox-project'),
  listSandboxProjectFiles: () => ipcRenderer.invoke('knirv:list-sandbox-project-files'),
  readSandboxProjectFile: (relativePath) => ipcRenderer.invoke('knirv:read-sandbox-project-file', relativePath),
  getPathForFile: (file) => webUtils.getPathForFile(file),
});
