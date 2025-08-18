import React, { useState, useRef } from 'react';
import {
  Upload,
  Cloud,
  FileText,
  CheckCircle,
  AlertCircle,
  X,
  Zap,
  Target,
  Settings,
  Play,
  Loader2,
  Download
} from 'lucide-react';
import { deployAgent, uploadAgentFile } from '../utils/api';

const DragDropDeployment = ({ onDeploymentComplete, targetSystems = [] }) => {
  const [dragActive, setDragActive] = useState(false);
  const [uploadedFiles, setUploadedFiles] = useState([]);
  const [deploymentQueue, setDeploymentQueue] = useState([]);
  const [deploying, setDeploying] = useState(false);
  const [deploymentResults, setDeploymentResults] = useState([]);
  const [selectedTargets, setSelectedTargets] = useState([]);
  const fileInputRef = useRef(null);

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
      file.name.endsWith('.so') || 
      file.name.endsWith('.dll') || 
      file.name.endsWith('.dylib') ||
      file.name.endsWith('.zip') ||
      file.name.endsWith('.tar.gz')
    );
    
    if (validFiles.length > 0) {
      const newFiles = validFiles.map(file => ({
        id: Date.now() + Math.random(),
        file: file,
        name: file.name,
        size: file.size,
        type: getFileType(file.name),
        status: 'pending',
        uploadProgress: 0
      }));
      
      setUploadedFiles(prev => [...prev, ...newFiles]);
      
      // Auto-upload files
      newFiles.forEach(fileInfo => uploadFile(fileInfo));
    }
  };

  const getFileType = (filename) => {
    if (filename.endsWith('.so')) return 'Linux Plugin';
    if (filename.endsWith('.dll')) return 'Windows Plugin';
    if (filename.endsWith('.dylib')) return 'macOS Plugin';
    if (filename.endsWith('.zip') || filename.endsWith('.tar.gz')) return 'Archive';
    return 'Unknown';
  };

  const uploadFile = async (fileInfo) => {
    try {
      setUploadedFiles(prev => prev.map(f => 
        f.id === fileInfo.id ? { ...f, status: 'uploading' } : f
      ));

      // Simulate upload progress
      for (let progress = 0; progress <= 100; progress += 10) {
        await new Promise(resolve => setTimeout(resolve, 100));
        setUploadedFiles(prev => prev.map(f => 
          f.id === fileInfo.id ? { ...f, uploadProgress: progress } : f
        ));
      }

      // In a real implementation, you would upload to the server here
      // const result = await uploadAgentFile(fileInfo.file);
      
      setUploadedFiles(prev => prev.map(f => 
        f.id === fileInfo.id ? { 
          ...f, 
          status: 'uploaded', 
          uploadProgress: 100,
          agentId: `agent_${Date.now()}`
        } : f
      ));

    } catch (error) {
      setUploadedFiles(prev => prev.map(f => 
        f.id === fileInfo.id ? { ...f, status: 'error', error: error.message } : f
      ));
    }
  };

  const addToDeploymentQueue = (fileInfo) => {
    if (selectedTargets.length === 0) {
      alert('Please select at least one target system');
      return;
    }

    const deploymentItem = {
      id: Date.now() + Math.random(),
      fileInfo,
      targets: [...selectedTargets],
      status: 'queued',
      timestamp: new Date().toISOString()
    };

    setDeploymentQueue(prev => [...prev, deploymentItem]);
  };

  const deployAll = async () => {
    if (deploymentQueue.length === 0) return;

    setDeploying(true);
    const results = [];

    for (const item of deploymentQueue) {
      try {
        setDeploymentQueue(prev => prev.map(q => 
          q.id === item.id ? { ...q, status: 'deploying' } : q
        ));

        // Simulate deployment to each target
        for (const target of item.targets) {
          await new Promise(resolve => setTimeout(resolve, 1000)); // Simulate deployment time
          
          // In a real implementation:
          // await deployAgent(item.fileInfo.agentId, target.id);
        }

        setDeploymentQueue(prev => prev.map(q => 
          q.id === item.id ? { ...q, status: 'deployed' } : q
        ));

        results.push({
          ...item,
          status: 'success',
          deployedAt: new Date().toISOString()
        });

      } catch (error) {
        setDeploymentQueue(prev => prev.map(q => 
          q.id === item.id ? { ...q, status: 'error', error: error.message } : q
        ));

        results.push({
          ...item,
          status: 'error',
          error: error.message
        });
      }
    }

    setDeploymentResults(results);
    setDeploying(false);
    
    if (onDeploymentComplete) {
      onDeploymentComplete(results);
    }
  };

  const removeFile = (fileId) => {
    setUploadedFiles(prev => prev.filter(f => f.id !== fileId));
    setDeploymentQueue(prev => prev.filter(q => q.fileInfo.id !== fileId));
  };

  const removeFromQueue = (queueId) => {
    setDeploymentQueue(prev => prev.filter(q => q.id !== queueId));
  };

  const onButtonClick = () => {
    fileInputRef.current?.click();
  };

  const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <div className="space-y-6">
      {/* Drag and Drop Upload Area */}
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
          accept=".so,.dll,.dylib,.zip,.tar.gz"
          onChange={(e) => handleFiles(e.target.files)}
          className="hidden"
        />
        <Cloud className={`w-16 h-16 mx-auto mb-4 ${dragActive ? 'text-blue-400' : 'text-slate-400'}`} />
        <h3 className={`text-xl font-medium mb-2 ${dragActive ? 'text-blue-300' : 'text-white'}`}>
          {dragActive ? 'Drop agent files here' : 'Drag & drop agent files for deployment'}
        </h3>
        <p className="text-slate-400 mb-4">
          Supports .so, .dll, .dylib, .zip, .tar.gz files
        </p>
        <button
          onClick={onButtonClick}
          className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors duration-200 flex items-center space-x-2 mx-auto"
        >
          <Upload className="w-5 h-5" />
          <span>Browse Files</span>
        </button>
      </div>

      {/* Target System Selection */}
      {targetSystems.length > 0 && (
        <div className="bg-slate-800/50 rounded-lg p-4">
          <h4 className="text-lg font-medium text-white mb-3 flex items-center space-x-2">
            <Target className="w-5 h-5 text-green-400" />
            <span>Select Target Systems</span>
          </h4>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {targetSystems.map((target) => (
              <label
                key={target.id}
                className="flex items-center space-x-3 p-3 bg-slate-700/50 rounded-lg cursor-pointer hover:bg-slate-700/70 transition-colors"
              >
                <input
                  type="checkbox"
                  checked={selectedTargets.some(t => t.id === target.id)}
                  onChange={(e) => {
                    if (e.target.checked) {
                      setSelectedTargets(prev => [...prev, target]);
                    } else {
                      setSelectedTargets(prev => prev.filter(t => t.id !== target.id));
                    }
                  }}
                  className="rounded border-slate-600 text-blue-600 focus:ring-blue-500"
                />
                <div>
                  <div className="text-white font-medium">{target.name}</div>
                  <div className="text-slate-400 text-sm">{target.type}</div>
                </div>
              </label>
            ))}
          </div>
        </div>
      )}

      {/* Uploaded Files */}
      {uploadedFiles.length > 0 && (
        <div className="bg-slate-800/50 rounded-lg p-4">
          <h4 className="text-lg font-medium text-white mb-3 flex items-center space-x-2">
            <FileText className="w-5 h-5 text-blue-400" />
            <span>Uploaded Files</span>
          </h4>
          <div className="space-y-3">
            {uploadedFiles.map((file) => (
              <div key={file.id} className="flex items-center justify-between p-3 bg-slate-700/50 rounded-lg">
                <div className="flex items-center space-x-3">
                  <div className={`p-2 rounded-lg ${
                    file.status === 'uploaded' ? 'bg-green-500/20 text-green-400' :
                    file.status === 'uploading' ? 'bg-blue-500/20 text-blue-400' :
                    file.status === 'error' ? 'bg-red-500/20 text-red-400' :
                    'bg-slate-500/20 text-slate-400'
                  }`}>
                    {file.status === 'uploading' ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : file.status === 'uploaded' ? (
                      <CheckCircle className="w-4 h-4" />
                    ) : file.status === 'error' ? (
                      <AlertCircle className="w-4 h-4" />
                    ) : (
                      <FileText className="w-4 h-4" />
                    )}
                  </div>
                  <div>
                    <div className="text-white font-medium">{file.name}</div>
                    <div className="text-slate-400 text-sm">
                      {file.type} • {formatFileSize(file.size)}
                    </div>
                    {file.status === 'uploading' && (
                      <div className="w-32 bg-slate-600 rounded-full h-1 mt-1">
                        <div 
                          className="bg-blue-500 h-1 rounded-full transition-all duration-300"
                          style={{ width: `${file.uploadProgress}%` }}
                        />
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  {file.status === 'uploaded' && (
                    <button
                      onClick={() => addToDeploymentQueue(file)}
                      className="px-3 py-1 bg-green-600 text-white rounded text-sm hover:bg-green-700 transition-colors flex items-center space-x-1"
                    >
                      <Zap className="w-3 h-3" />
                      <span>Deploy</span>
                    </button>
                  )}
                  <button
                    onClick={() => removeFile(file.id)}
                    className="text-slate-400 hover:text-red-400 transition-colors"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Deployment Queue */}
      {deploymentQueue.length > 0 && (
        <div className="bg-slate-800/50 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h4 className="text-lg font-medium text-white flex items-center space-x-2">
              <Play className="w-5 h-5 text-orange-400" />
              <span>Deployment Queue ({deploymentQueue.length})</span>
            </h4>
            <button
              onClick={deployAll}
              disabled={deploying}
              className="px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors disabled:opacity-50 flex items-center space-x-2"
            >
              {deploying && <Loader2 className="w-4 h-4 animate-spin" />}
              <span>{deploying ? 'Deploying...' : 'Deploy All'}</span>
            </button>
          </div>
          <div className="space-y-2">
            {deploymentQueue.map((item) => (
              <div key={item.id} className="flex items-center justify-between p-3 bg-slate-700/30 rounded-lg">
                <div className="flex items-center space-x-3">
                  <div className={`w-2 h-2 rounded-full ${
                    item.status === 'deployed' ? 'bg-green-400' :
                    item.status === 'deploying' ? 'bg-blue-400 animate-pulse' :
                    item.status === 'error' ? 'bg-red-400' :
                    'bg-slate-400'
                  }`} />
                  <div>
                    <div className="text-white font-medium">{item.fileInfo.name}</div>
                    <div className="text-slate-400 text-sm">
                      {item.targets.length} target(s) • {item.status}
                    </div>
                  </div>
                </div>
                <button
                  onClick={() => removeFromQueue(item.id)}
                  disabled={item.status === 'deploying'}
                  className="text-slate-400 hover:text-red-400 transition-colors disabled:opacity-50"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default DragDropDeployment;
