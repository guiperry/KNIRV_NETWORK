import React, { useEffect, useState } from 'react';
import { 
  Minimize2, 
  X, 
  Settings, 
  Info,
  Download,
  RefreshCw,
  CheckCircle,
  AlertTriangle
} from 'lucide-react';

const SystemTrayManager = () => {
  const [isMinimizedToTray, setIsMinimizedToTray] = useState(false);
  const [updateInfo, setUpdateInfo] = useState(null);
  const [downloadProgress, setDownloadProgress] = useState(null);
  const [updateError, setUpdateError] = useState(null);
  const [appVersion, setAppVersion] = useState('');

  useEffect(() => {
    // Check if running in Electron
    if (!window.electronAPI) return;

    // Get app version
    window.electronAPI.getAppVersion().then(version => {
      setAppVersion(version);
    });

    // Check tray status
    window.electronAPI.isMinimizedToTray().then(minimized => {
      setIsMinimizedToTray(minimized);
    });

    // Set up auto-updater event listeners
    window.electronAPI.onUpdateAvailable((info) => {
      setUpdateInfo(info);
      console.log('Update available:', info);
    });

    window.electronAPI.onUpdateDownloaded((info) => {
      setUpdateInfo({ ...info, downloaded: true });
      setDownloadProgress(null);
      console.log('Update downloaded:', info);
    });

    window.electronAPI.onDownloadProgress((progress) => {
      setDownloadProgress(progress);
      console.log('Download progress:', progress);
    });

    window.electronAPI.onUpdateError((error) => {
      setUpdateError(error);
      console.error('Update error:', error);
    });

    // Cleanup function
    return () => {
      // Remove listeners if needed
    };
  }, []);

  const handleMinimizeToTray = async () => {
    if (window.electronAPI) {
      try {
        await window.electronAPI.minimizeToTray();
        setIsMinimizedToTray(true);
      } catch (error) {
        console.error('Failed to minimize to tray:', error);
      }
    }
  };

  const handleShowFromTray = async () => {
    if (window.electronAPI) {
      try {
        await window.electronAPI.showFromTray();
        setIsMinimizedToTray(false);
      } catch (error) {
        console.error('Failed to show from tray:', error);
      }
    }
  };

  const handleCheckForUpdates = async () => {
    if (window.electronAPI) {
      try {
        const result = await window.electronAPI.checkForUpdates();
        if (result.success) {
          console.log('Update check completed:', result.updateInfo);
        } else {
          setUpdateError(result.error);
        }
      } catch (error) {
        console.error('Failed to check for updates:', error);
        setUpdateError(error.message);
      }
    }
  };

  const handleDownloadUpdate = async () => {
    if (window.electronAPI && updateInfo && !updateInfo.downloaded) {
      try {
        const result = await window.electronAPI.downloadUpdate();
        if (!result.success) {
          setUpdateError(result.error);
        }
      } catch (error) {
        console.error('Failed to download update:', error);
        setUpdateError(error.message);
      }
    }
  };

  const handleInstallUpdate = async () => {
    if (window.electronAPI && updateInfo && updateInfo.downloaded) {
      try {
        await window.electronAPI.quitAndInstall();
      } catch (error) {
        console.error('Failed to install update:', error);
        setUpdateError(error.message);
      }
    }
  };

  const handleShowCloseDialog = async () => {
    if (window.electronAPI) {
      try {
        const result = await window.electronAPI.showCloseDialog();
        if (result.success) {
          console.log('Close dialog choice:', result.choice);
          // The main process will handle the action based on the choice
        }
      } catch (error) {
        console.error('Failed to show close dialog:', error);
      }
    }
  };

  // Don't render if not in Electron
  if (!window.electronAPI) {
    return null;
  }

  return (
    <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-xl font-semibold text-white flex items-center space-x-2">
          <Settings className="w-6 h-6 text-blue-400" />
          <span>System Tray & Updates</span>
        </h3>
        <div className="text-sm text-slate-400">
          v{appVersion}
        </div>
      </div>

      {/* System Tray Controls */}
      <div className="mb-6">
        <h4 className="text-lg font-medium text-white mb-3">System Tray</h4>
        <div className="space-y-3">
          <div className="flex items-center justify-between p-3 bg-slate-700/30 rounded-lg">
            <div>
              <p className="text-white font-medium">Minimize to System Tray</p>
              <p className="text-slate-400 text-sm">Hide the application to the system tray</p>
            </div>
            <button
              onClick={handleMinimizeToTray}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors duration-200 flex items-center space-x-2"
            >
              <Minimize2 className="w-4 h-4" />
              <span>Minimize</span>
            </button>
          </div>

          {isMinimizedToTray && (
            <div className="flex items-center justify-between p-3 bg-green-900/20 border border-green-500/30 rounded-lg">
              <div className="flex items-center space-x-2">
                <CheckCircle className="w-5 h-5 text-green-400" />
                <div>
                  <p className="text-green-200 font-medium">Minimized to Tray</p>
                  <p className="text-green-300 text-sm">Application is running in the background</p>
                </div>
              </div>
              <button
                onClick={handleShowFromTray}
                className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors duration-200"
              >
                Show
              </button>
            </div>
          )}

          <div className="flex items-center justify-between p-3 bg-slate-700/30 rounded-lg">
            <div>
              <p className="text-white font-medium">Close Dialog</p>
              <p className="text-slate-400 text-sm">Test the close confirmation dialog</p>
            </div>
            <button
              onClick={handleShowCloseDialog}
              className="px-4 py-2 bg-slate-600 text-white rounded-lg hover:bg-slate-700 transition-colors duration-200 flex items-center space-x-2"
            >
              <X className="w-4 h-4" />
              <span>Test Close</span>
            </button>
          </div>
        </div>
      </div>

      {/* Auto-Updater Controls */}
      <div>
        <h4 className="text-lg font-medium text-white mb-3">Auto-Updater</h4>
        <div className="space-y-3">
          <div className="flex items-center justify-between p-3 bg-slate-700/30 rounded-lg">
            <div>
              <p className="text-white font-medium">Check for Updates</p>
              <p className="text-slate-400 text-sm">Check if a new version is available</p>
            </div>
            <button
              onClick={handleCheckForUpdates}
              className="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors duration-200 flex items-center space-x-2"
            >
              <RefreshCw className="w-4 h-4" />
              <span>Check</span>
            </button>
          </div>

          {updateError && (
            <div className="p-3 bg-red-900/20 border border-red-500/30 rounded-lg">
              <div className="flex items-center space-x-2">
                <AlertTriangle className="w-5 h-5 text-red-400" />
                <div>
                  <p className="text-red-200 font-medium">Update Error</p>
                  <p className="text-red-300 text-sm">{updateError}</p>
                </div>
              </div>
            </div>
          )}

          {updateInfo && (
            <div className="p-3 bg-blue-900/20 border border-blue-500/30 rounded-lg">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <Info className="w-5 h-5 text-blue-400" />
                  <div>
                    <p className="text-blue-200 font-medium">
                      Update Available: v{updateInfo.version}
                    </p>
                    <p className="text-blue-300 text-sm">
                      {updateInfo.downloaded ? 'Ready to install' : 'Ready to download'}
                    </p>
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  {!updateInfo.downloaded ? (
                    <button
                      onClick={handleDownloadUpdate}
                      className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors duration-200 flex items-center space-x-2"
                    >
                      <Download className="w-4 h-4" />
                      <span>Download</span>
                    </button>
                  ) : (
                    <button
                      onClick={handleInstallUpdate}
                      className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors duration-200 flex items-center space-x-2"
                    >
                      <CheckCircle className="w-4 h-4" />
                      <span>Install</span>
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}

          {downloadProgress && (
            <div className="p-3 bg-blue-900/20 border border-blue-500/30 rounded-lg">
              <div className="flex items-center justify-between mb-2">
                <p className="text-blue-200 font-medium">Downloading Update</p>
                <p className="text-blue-300 text-sm">{downloadProgress.percent.toFixed(1)}%</p>
              </div>
              <div className="w-full bg-slate-700 rounded-full h-2">
                <div 
                  className="bg-blue-500 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${downloadProgress.percent}%` }}
                ></div>
              </div>
              <div className="flex justify-between text-xs text-blue-300 mt-1">
                <span>{(downloadProgress.transferred / 1024 / 1024).toFixed(1)} MB</span>
                <span>{(downloadProgress.total / 1024 / 1024).toFixed(1)} MB</span>
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="mt-6 p-3 bg-slate-700/20 rounded-lg">
        <p className="text-slate-400 text-sm">
          <strong>System Tray:</strong> When minimized to tray, the application continues running in the background. 
          Right-click the tray icon to access quick actions or double-click to restore the window.
        </p>
      </div>
    </div>
  );
};

export default SystemTrayManager;
