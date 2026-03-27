import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  getServerUrl: () => {
    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get('serverUrl') || 'http://localhost:8090';
  },
  minimize: () => ipcRenderer.send('minimize-window'),
  close: () => ipcRenderer.send('close-window'),
});
