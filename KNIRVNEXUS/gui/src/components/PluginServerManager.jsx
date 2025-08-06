import React, { useState, useEffect } from 'react';
import { 
  Upload, 
  Download, 
  Trash2, 
  File, 
  Server, 
  AlertCircle, 
  CheckCircle,
  RefreshCw,
  HardDrive
} from 'lucide-react';
import { pluginServerService } from '../services/pluginServerService';

const PluginServerManager = ({ onAgentSelect, selectedAgent = null }) => {
  const [agents, setAgents] = useState([]);
  const [serverInfo, setServerInfo] = useState(null);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [error, setError] = useState(null);
  const [serverAvailable, setServerAvailable] = useState(false);

  // Load data on component mount
  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    setError(null);
    
    try {
      // Check if server is available
      const available = await pluginServerService.isServerAvailable();
      setServerAvailable(available);
      
      if (available) {
        // Load server info and agents list
        const [info, agentsList] = await Promise.all([
          pluginServerService.getServerInfo(),
          pluginServerService.listAgents()
        ]);
        
        setServerInfo(info);
        setAgents(agentsList.agents || []);
      }
    } catch (err) {
      setError(err.message);
      setServerAvailable(false);
    } finally {
      setLoading(false);
    }
  };

  const handleFileUpload = async (event) => {
    const file = event.target.files[0];
    if (!file) return;

    // Validate file type
    if (!pluginServerService.isValidAgentFile(file)) {
      setError('Invalid file type. Please upload .wasm, .so, .dll, or .dylib files.');
      return;
    }

    setUploading(true);
    setUploadProgress(0);
    setError(null);

    try {
      await pluginServerService.uploadAgent(file, (progress) => {
        setUploadProgress(progress);
      });
      
      // Reload agents list
      await loadData();
      setUploadProgress(0);
    } catch (err) {
      setError(`Upload failed: ${err.message}`);
    } finally {
      setUploading(false);
      // Reset file input
      event.target.value = '';
    }
  };

  const handleDownload = async (agent) => {
    try {
      const { blob, filename } = await pluginServerService.downloadAgent(agent.name);
      
      // Create download link
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      setError(`Download failed: ${err.message}`);
    }
  };

  const handleDelete = async (agent) => {
    if (!window.confirm(`Are you sure you want to delete ${agent.name}?`)) {
      return;
    }

    try {
      await pluginServerService.deleteAgent(agent.name);
      await loadData(); // Reload list
    } catch (err) {
      setError(`Delete failed: ${err.message}`);
    }
  };

  const handleAgentSelect = (agent) => {
    if (onAgentSelect) {
      onAgentSelect({
        name: agent.name,
        url: pluginServerService.getAgentUrl(agent.name),
        size: agent.size,
        lastModified: agent.last_modified,
        type: pluginServerService.getFileTypeDescription(agent.name)
      });
    }
  };

  if (!serverAvailable && !loading) {
    return (
      <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
        <div className="flex items-center space-x-3 mb-4">
          <Server className="w-6 h-6 text-red-400" />
          <h3 className="text-lg font-semibold text-white">Plugin Server</h3>
        </div>
        <div className="flex items-center space-x-2 text-red-400">
          <AlertCircle className="w-5 h-5" />
          <span>Plugin server is not available</span>
        </div>
        <p className="text-slate-400 mt-2 text-sm">
          The plugin server is required for managing WASM and binary agent files.
        </p>
        <button
          onClick={loadData}
          className="mt-4 px-4 py-2 bg-knirv-primary text-white rounded-lg hover:bg-knirv-secondary transition-colors"
        >
          <RefreshCw className="w-4 h-4 inline mr-2" />
          Retry Connection
        </button>
      </div>
    );
  }

  return (
    <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-3">
          <HardDrive className="w-6 h-6 text-knirv-primary" />
          <h3 className="text-lg font-semibold text-white">Plugin Server Storage</h3>
          {serverAvailable && (
            <div className="flex items-center space-x-1 text-green-400">
              <CheckCircle className="w-4 h-4" />
              <span className="text-sm">Connected</span>
            </div>
          )}
        </div>
        <button
          onClick={loadData}
          disabled={loading}
          className="p-2 text-slate-400 hover:text-white transition-colors"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Server Info */}
      {serverInfo && (
        <div className="bg-slate-700 rounded-lg p-3 mb-4 text-sm">
          <div className="grid grid-cols-2 gap-2 text-slate-300">
            <div>Server: {serverInfo.name}</div>
            <div>Port: {serverInfo.port}</div>
            <div>Version: {serverInfo.version}</div>
            <div>Agents: {agents.length}</div>
          </div>
        </div>
      )}

      {/* Upload Section */}
      <div className="mb-6">
        <label className="block text-sm font-medium text-slate-300 mb-2">
          Upload Agent File
        </label>
        <div className="relative">
          <input
            type="file"
            accept=".wasm,.so,.dll,.dylib"
            onChange={handleFileUpload}
            disabled={uploading || !serverAvailable}
            className="hidden"
            id="agent-file-upload"
          />
          <label
            htmlFor="agent-file-upload"
            className={`flex items-center justify-center w-full p-4 border-2 border-dashed rounded-lg cursor-pointer transition-colors ${
              uploading || !serverAvailable
                ? 'border-slate-600 bg-slate-700 cursor-not-allowed'
                : 'border-knirv-primary bg-slate-700 hover:bg-slate-600'
            }`}
          >
            <div className="text-center">
              <Upload className="w-8 h-8 text-knirv-primary mx-auto mb-2" />
              <p className="text-sm text-slate-300">
                {uploading ? 'Uploading...' : 'Click to upload agent file'}
              </p>
              <p className="text-xs text-slate-400 mt-1">
                Supports .wasm, .so, .dll, .dylib files
              </p>
            </div>
          </label>
        </div>
        
        {uploading && (
          <div className="mt-2">
            <div className="bg-slate-700 rounded-full h-2">
              <div 
                className="bg-knirv-primary h-2 rounded-full transition-all duration-300"
                style={{ width: `${uploadProgress}%` }}
              />
            </div>
            <p className="text-xs text-slate-400 mt-1">{Math.round(uploadProgress)}% uploaded</p>
          </div>
        )}
      </div>

      {/* Error Display */}
      {error && (
        <div className="bg-red-900/20 border border-red-500 rounded-lg p-3 mb-4">
          <div className="flex items-center space-x-2 text-red-400">
            <AlertCircle className="w-4 h-4" />
            <span className="text-sm">{error}</span>
          </div>
        </div>
      )}

      {/* Agents List */}
      <div>
        <h4 className="text-sm font-medium text-slate-300 mb-3">Available Agents</h4>
        {loading ? (
          <div className="text-center py-8">
            <RefreshCw className="w-6 h-6 text-knirv-primary animate-spin mx-auto mb-2" />
            <p className="text-slate-400">Loading agents...</p>
          </div>
        ) : agents.length === 0 ? (
          <div className="text-center py-8 text-slate-400">
            <File className="w-8 h-8 mx-auto mb-2" />
            <p>No agents available</p>
          </div>
        ) : (
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {agents.map((agent, index) => (
              <div
                key={index}
                className={`flex items-center justify-between p-3 rounded-lg border transition-colors cursor-pointer ${
                  selectedAgent?.name === agent.name
                    ? 'bg-knirv-primary/20 border-knirv-primary'
                    : 'bg-slate-700 border-slate-600 hover:bg-slate-600'
                }`}
                onClick={() => handleAgentSelect(agent)}
              >
                <div className="flex items-center space-x-3">
                  <File className="w-4 h-4 text-knirv-primary" />
                  <div>
                    <p className="text-sm font-medium text-white">{agent.name}</p>
                    <p className="text-xs text-slate-400">
                      {pluginServerService.formatFileSize(agent.size)} • 
                      {pluginServerService.getFileTypeDescription(agent.name)}
                    </p>
                  </div>
                </div>
                <div className="flex items-center space-x-1">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDownload(agent);
                    }}
                    className="p-1 text-slate-400 hover:text-knirv-primary transition-colors"
                    title="Download"
                  >
                    <Download className="w-4 h-4" />
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDelete(agent);
                    }}
                    className="p-1 text-slate-400 hover:text-red-400 transition-colors"
                    title="Delete"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default PluginServerManager;
