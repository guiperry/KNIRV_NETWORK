import React, { useState, useEffect, useRef } from 'react';
import {
  X,
  Upload,
  FileText,
  AlertCircle,
  CheckCircle,
  Loader2,
  Download,
  Info,
  Cloud
} from 'lucide-react';
import { discoverAllPlugins, importPlugin } from '../../utils/api';
import { handleApiError } from '../../utils/errorHandler';

const PluginImportModal = ({ isOpen, onClose, onPluginImported }) => {
  const [plugins, setPlugins] = useState([]);
  const [loading, setLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [selectedPlugin, setSelectedPlugin] = useState(null);
  const [importForm, setImportForm] = useState({
    agentId: '',
    version: '1.0'
  });
  const [dragActive, setDragActive] = useState(false);
  const [uploadedFiles, setUploadedFiles] = useState([]);
  const fileInputRef = useRef(null);

  useEffect(() => {
    if (isOpen) {
      loadPlugins();
    }
  }, [isOpen]);

  const loadPlugins = async () => {
    setLoading(true);
    setError('');
    try {
      const discoveredPlugins = await discoverAllPlugins();
      setPlugins(discoveredPlugins);
    } catch (err) {
      console.error('Error discovering plugins:', err);
      handleApiError(err, {
        operation: 'discover_plugins',
        component: 'PluginImportModal',
        timestamp: new Date().toISOString(),
        context: 'Loading available plugins for import'
      });
      setError(`Failed to discover plugins: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handlePluginSelect = (plugin) => {
    setSelectedPlugin(plugin);
    // Pre-fill form if plugin has metadata
    if (plugin.agentId) {
      setImportForm(prev => ({
        ...prev,
        agentId: plugin.agentId
      }));
    }
    if (plugin.version) {
      setImportForm(prev => ({
        ...prev,
        version: plugin.version
      }));
    }
  };

  const handleImport = async () => {
    if (!selectedPlugin || !importForm.agentId || !importForm.version) {
      setError('Please select a plugin and provide agent ID and version');
      return;
    }

    setImporting(true);
    setError('');
    setSuccess('');
    try {
      await importPlugin({
        filePath: selectedPlugin.filePath,
        agentId: importForm.agentId,
        version: importForm.version
      });

      setSuccess(`Plugin imported successfully! Agent "${importForm.agentId}" has been created and will appear in your agent list.`);

      // Notify parent component to refresh agent list
      if (onPluginImported) {
        onPluginImported();
      }

      // Reset form but keep modal open to show success message
      setSelectedPlugin(null);
      setImportForm({ agentId: '', version: '1.0' });

      // Auto-close after 3 seconds
      setTimeout(() => {
        onClose();
        setSuccess('');
      }, 3000);

    } catch (err) {
      console.error('Error importing plugin:', err);
      handleApiError(err, {
        operation: 'import_plugin',
        component: 'PluginImportModal',
        agentId: importForm.agentId,
        version: importForm.version,
        pluginPath: selectedPlugin?.filePath,
        timestamp: new Date().toISOString(),
        context: 'User attempted to import a plugin agent'
      });

      // Handle specific error cases
      if (err.message.includes('already exists')) {
        setError(`Agent "${importForm.agentId}" version ${importForm.version} already exists. Please use a different agent ID or version.`);
      } else {
        setError(`Failed to import plugin: ${err.message}`);
      }
    } finally {
      setImporting(false);
    }
  };

  const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  // Drag and drop handlers
  const handleDrag = (e) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  };

  const handleDrop = (e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFiles(e.dataTransfer.files);
    }
  };

  const handleFiles = (files) => {
    const fileArray = Array.from(files);
    const validFiles = fileArray.filter(file =>
      file.name.endsWith('.so') || file.name.endsWith('.dll') || file.name.endsWith('.dylib')
    );

    if (validFiles.length > 0) {
      setUploadedFiles(prev => [...prev, ...validFiles]);
      // For demo purposes, we'll simulate adding these to the plugins list
      const newPlugins = validFiles.map(file => ({
        fileName: file.name,
        filePath: `uploaded/${file.name}`,
        size: file.size,
        modTime: new Date().toISOString(),
        isUploaded: true,
        file: file
      }));
      setPlugins(prev => [...newPlugins, ...prev]);
    } else {
      setError('Please upload valid plugin files (.so, .dll, or .dylib)');
    }
  };

  const onButtonClick = () => {
    fileInputRef.current?.click();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-slate-800 rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <div className="flex items-center space-x-3">
            <Download className="w-6 h-6 text-blue-400" />
            <h2 className="text-xl font-semibold text-white">Import Plugin Agent</h2>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors duration-200"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-200px)]">
          {error && (
            <div className="mb-4 p-4 bg-red-900/50 border border-red-500 rounded-lg flex items-center space-x-2">
              <AlertCircle className="w-5 h-5 text-red-400" />
              <span className="text-red-200">{error}</span>
            </div>
          )}

          {success && (
            <div className="mb-4 p-4 bg-green-900/50 border border-green-500 rounded-lg flex items-center space-x-2">
              <CheckCircle className="w-5 h-5 text-green-400" />
              <span className="text-green-200">{success}</span>
            </div>
          )}

          {/* Drag and Drop Upload Area */}
          <div className="mb-6">
            <div
              className={`relative border-2 border-dashed rounded-lg p-8 text-center transition-all duration-200 ${
                dragActive
                  ? 'border-blue-400 bg-blue-900/20'
                  : 'border-slate-600 bg-slate-700/30 hover:border-slate-500'
              }`}
              onDragEnter={handleDrag}
              onDragLeave={handleDrag}
              onDragOver={handleDrag}
              onDrop={handleDrop}
            >
              <input
                ref={fileInputRef}
                type="file"
                multiple
                accept=".so,.dll,.dylib"
                onChange={(e) => handleFiles(e.target.files)}
                className="hidden"
              />
              <Cloud className={`w-12 h-12 mx-auto mb-4 ${dragActive ? 'text-blue-400' : 'text-slate-400'}`} />
              <p className={`text-lg font-medium mb-2 ${dragActive ? 'text-blue-300' : 'text-white'}`}>
                {dragActive ? 'Drop plugin files here' : 'Drag & drop plugin files'}
              </p>
              <p className="text-slate-400 mb-4">
                Supports .so (Linux), .dll (Windows), .dylib (macOS) files
              </p>
              <button
                onClick={onButtonClick}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors duration-200"
              >
                Browse Files
              </button>
            </div>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 text-blue-400 animate-spin" />
              <span className="ml-3 text-slate-300">Discovering plugins...</span>
            </div>
          ) : (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Plugin List */}
              <div>
                <h3 className="text-lg font-medium text-white mb-4">Available Plugins</h3>
                <div className="space-y-3 max-h-96 overflow-y-auto">
                  {plugins.length === 0 ? (
                    <div className="text-center py-8 text-slate-400">
                      <FileText className="w-12 h-12 mx-auto mb-3 opacity-50" />
                      <p>No plugins found in the plugins directory</p>
                    </div>
                  ) : (
                    plugins.map((plugin, index) => (
                      <div
                        key={index}
                        className={`p-4 rounded-lg border cursor-pointer transition-all duration-200 ${
                          selectedPlugin === plugin
                            ? 'border-blue-500 bg-blue-900/20'
                            : 'border-slate-600 bg-slate-700/50 hover:border-slate-500'
                        }`}
                        onClick={() => handlePluginSelect(plugin)}
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <div className="flex items-center space-x-2">
                              <FileText className="w-4 h-4 text-slate-400" />
                              <span className="font-medium text-white">{plugin.fileName}</span>
                              {plugin.isRegistered && (
                                <CheckCircle className="w-4 h-4 text-green-400" />
                              )}
                            </div>
                            <div className="mt-2 space-y-1 text-sm text-slate-300">
                              <p>Size: {formatFileSize(plugin.size)}</p>
                              <p>Modified: {formatDate(plugin.modTime)}</p>
                              {plugin.agentId && (
                                <p>Agent ID: <span className="text-blue-300">{plugin.agentId}</span></p>
                              )}
                              {plugin.version && (
                                <p>Version: <span className="text-green-300">{plugin.version}</span></p>
                              )}
                            </div>
                            {plugin.error && (
                              <div className="mt-2 flex items-center space-x-1 text-red-400 text-sm">
                                <AlertCircle className="w-3 h-3" />
                                <span>{plugin.error}</span>
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* Import Form */}
              <div>
                <h3 className="text-lg font-medium text-white mb-4">Import Configuration</h3>
                {selectedPlugin ? (
                  <div className="space-y-4">
                    <div className="p-4 bg-slate-700/50 rounded-lg">
                      <h4 className="font-medium text-white mb-2">Selected Plugin</h4>
                      <p className="text-slate-300 text-sm">{selectedPlugin.fileName}</p>
                      <p className="text-slate-400 text-xs mt-1">{selectedPlugin.filePath}</p>
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-slate-300 mb-2">
                        Agent ID *
                      </label>
                      <input
                        type="text"
                        value={importForm.agentId}
                        onChange={(e) => setImportForm(prev => ({ ...prev, agentId: e.target.value }))}
                        className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        placeholder="e.g., shopify_assistant"
                      />
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-slate-300 mb-2">
                        Version *
                      </label>
                      <input
                        type="text"
                        value={importForm.version}
                        onChange={(e) => setImportForm(prev => ({ ...prev, version: e.target.value }))}
                        className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        placeholder="e.g., 1.0"
                      />
                    </div>

                    {selectedPlugin.metadata && (
                      <div>
                        <h4 className="font-medium text-slate-300 mb-2">Plugin Metadata</h4>
                        <div className="p-3 bg-slate-700/50 rounded-lg text-sm">
                          <pre className="text-slate-300 whitespace-pre-wrap">
                            {JSON.stringify(selectedPlugin.metadata, null, 2)}
                          </pre>
                        </div>
                      </div>
                    )}

                    <div className="flex items-start space-x-2 p-3 bg-blue-900/20 border border-blue-500/30 rounded-lg">
                      <Info className="w-4 h-4 text-blue-400 mt-0.5" />
                      <div className="text-sm text-blue-200">
                        <p className="mb-2">When imported, this plugin will:</p>
                        <ul className="space-y-1 text-xs">
                          <li>• Be renamed to: <code className="text-blue-300">agent_{importForm.agentId}_{importForm.version}.so</code></li>
                          <li>• Create a new agent record in the database</li>
                          <li>• Appear in your agent list as "{importForm.agentId}"</li>
                          <li>• Be available for deployment and execution</li>
                        </ul>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="text-center py-8 text-slate-400">
                    <Upload className="w-12 h-12 mx-auto mb-3 opacity-50" />
                    <p>Select a plugin from the list to configure import settings</p>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between p-6 border-t border-slate-700">
          <button
            onClick={loadPlugins}
            disabled={loading}
            className="px-4 py-2 text-slate-300 hover:text-white transition-colors duration-200 disabled:opacity-50"
          >
            Refresh
          </button>
          <div className="flex items-center space-x-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-slate-300 hover:text-white transition-colors duration-200"
            >
              Cancel
            </button>
            <button
              onClick={handleImport}
              disabled={!selectedPlugin || !importForm.agentId || !importForm.version || importing}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
            >
              {importing && <Loader2 className="w-4 h-4 animate-spin" />}
              <span>{importing ? 'Importing...' : 'Import Plugin'}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default PluginImportModal;
