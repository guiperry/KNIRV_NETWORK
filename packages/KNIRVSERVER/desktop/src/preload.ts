import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  send: (channel: string, payload?: unknown) => ipcRenderer.send(channel, payload),
  onInitServerUrl: (callback: (url: string) => void) => {
    ipcRenderer.on('init-server-url', (_event, url: string) => callback(url));
  },
  onShowMenu: (callback: (payload: { menuUrl: string }) => void) => {
    ipcRenderer.on('show-menu', (_event, payload: { menuUrl: string }) => callback(payload));
  },
  onShowDesktop: (callback: (payload: { frontendUrl: string }) => void) => {
    ipcRenderer.on('show-desktop', (_event, payload: { frontendUrl: string }) => callback(payload));
  },
  onFrontendStatus: (callback: (status: string) => void) => {
    ipcRenderer.on('frontend-status', (_event, status: string) => callback(status));
  },
  getSystemInfo: () => ({
    type: process.platform,
    release: typeof process.getSystemVersion === 'function' ? process.getSystemVersion() : '',
    arch: process.arch,
  }),
  getSystemMetrics: () => ({
    uptime: 0,
    totalMem: 0,
    freeMem: 0,
    cpus: [],
  }),
});
