import React, { useState, useEffect } from 'react';
import { X, CheckCircle, XCircle, Clock, AlertTriangle, Download } from 'lucide-react';
import { subscribeToBuildProgress, BuildProgressUpdate } from '../utils/websocket';

const BuildProgressMonitor = ({ isOpen, onClose, buildId, agentName }) => {
  const [buildStatus, setBuildStatus] = useState({
    status: 'started',
    progress: 0,
    message: 'Initializing build...',
    timestamp: new Date().toISOString(),
    logs: []
  });

  useEffect(() => {
    if (!isOpen || !buildId) return;

    // Subscribe to build progress updates
    const handleBuildProgress = (update) => {
      if (update.buildId === buildId) {
        setBuildStatus(prevStatus => ({
          ...prevStatus,
          status: update.status,
          progress: update.progress,
          message: update.message,
          timestamp: update.timestamp,
          logs: [...prevStatus.logs, {
            timestamp: update.timestamp,
            message: update.message,
            level: update.status === 'failed' ? 'error' : 'info'
          }]
        }));

        // Auto-close on completion or failure after 3 seconds
        if (update.status === 'completed' || update.status === 'failed') {
          setTimeout(() => {
            onClose();
          }, 3000);
        }
      }
    };

    subscribeToBuildProgress(handleBuildProgress);

    // Cleanup function would be needed here in a real implementation
    // to unsubscribe from the WebSocket events
  }, [isOpen, buildId, onClose]);

  const getStatusIcon = () => {
    switch (buildStatus.status) {
      case 'completed':
        return <CheckCircle className="w-6 h-6 text-green-400" />;
      case 'failed':
        return <XCircle className="w-6 h-6 text-red-400" />;
      case 'in_progress':
        return <Clock className="w-6 h-6 text-blue-400 animate-pulse" />;
      default:
        return <AlertTriangle className="w-6 h-6 text-yellow-400" />;
    }
  };

  const getStatusColor = () => {
    switch (buildStatus.status) {
      case 'completed':
        return 'text-green-400 bg-green-500/20 border-green-500/30';
      case 'failed':
        return 'text-red-400 bg-red-500/20 border-red-500/30';
      case 'in_progress':
        return 'text-blue-400 bg-blue-500/20 border-blue-500/30';
      default:
        return 'text-yellow-400 bg-yellow-500/20 border-yellow-500/30';
    }
  };

  const formatTime = (timestamp) => {
    return new Date(timestamp).toLocaleTimeString();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
        onClick={onClose}
      ></div>
      
      {/* Modal */}
      <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-2xl max-h-[80vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <div className="flex items-center space-x-3">
            {getStatusIcon()}
            <div>
              <h2 className="text-xl font-bold text-white">Building Agent Plugin</h2>
              <p className="text-slate-400">{agentName}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6 space-y-6">
          {/* Status */}
          <div className="flex items-center justify-between">
            <div>
              <span className={`px-3 py-1 rounded-full text-sm font-medium capitalize ${getStatusColor()}`}>
                {buildStatus.status.replace('_', ' ')}
              </span>
            </div>
            <div className="text-slate-400 text-sm">
              {formatTime(buildStatus.timestamp)}
            </div>
          </div>

          {/* Progress Bar */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-slate-300">Progress</span>
              <span className="text-slate-400">{buildStatus.progress}%</span>
            </div>
            <div className="w-full bg-slate-700 rounded-full h-2">
              <div 
                className={`h-2 rounded-full transition-all duration-300 ${
                  buildStatus.status === 'completed' ? 'bg-green-500' :
                  buildStatus.status === 'failed' ? 'bg-red-500' :
                  'bg-blue-500'
                }`}
                style={{ width: `${buildStatus.progress}%` }}
              ></div>
            </div>
          </div>

          {/* Current Message */}
          <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
            <h3 className="text-white font-medium mb-2">Current Status</h3>
            <p className="text-slate-300 text-sm">{buildStatus.message}</p>
          </div>

          {/* Build Logs */}
          <div className="bg-slate-900/50 rounded-lg border border-slate-700/50">
            <div className="p-4 border-b border-slate-700/50">
              <h3 className="text-white font-medium">Build Logs</h3>
            </div>
            <div className="p-4 max-h-64 overflow-y-auto">
              {buildStatus.logs.length > 0 ? (
                <div className="space-y-2">
                  {buildStatus.logs.map((log, index) => (
                    <div key={index} className="flex items-start space-x-3 text-sm">
                      <span className="text-slate-500 font-mono text-xs">
                        {formatTime(log.timestamp)}
                      </span>
                      <span className={`flex-1 ${
                        log.level === 'error' ? 'text-red-400' : 'text-slate-300'
                      }`}>
                        {log.message}
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-slate-500 text-sm">No logs available yet...</p>
              )}
            </div>
          </div>

          {/* Actions */}
          {buildStatus.status === 'completed' && (
            <div className="flex items-center space-x-3">
              <button 
                onClick={() => {
                  // In a real implementation, this would download the built plugin
                  console.log('Downloading plugin for build:', buildId);
                }}
                className="flex-1 bg-gradient-to-r from-green-500/20 to-emerald-500/20 hover:from-green-500/30 hover:to-emerald-500/30 border border-green-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2"
              >
                <Download className="w-4 h-4" />
                <span>Download Plugin</span>
              </button>
              <button 
                onClick={onClose}
                className="px-6 py-2 bg-slate-700/50 border border-slate-600/50 text-slate-300 hover:text-white hover:border-slate-500/50 rounded-lg transition-all duration-200"
              >
                Close
              </button>
            </div>
          )}

          {buildStatus.status === 'failed' && (
            <div className="flex items-center space-x-3">
              <button 
                onClick={() => {
                  // In a real implementation, this would retry the build
                  console.log('Retrying build for:', buildId);
                }}
                className="flex-1 bg-gradient-to-r from-blue-500/20 to-cyan-500/20 hover:from-blue-500/30 hover:to-cyan-500/30 border border-blue-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200"
              >
                Retry Build
              </button>
              <button 
                onClick={onClose}
                className="px-6 py-2 bg-slate-700/50 border border-slate-600/50 text-slate-300 hover:text-white hover:border-slate-500/50 rounded-lg transition-all duration-200"
              >
                Close
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default BuildProgressMonitor;
