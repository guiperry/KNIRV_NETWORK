import React, { useState, useRef, useCallback } from 'react';
import { Upload, FileText, Cpu, Zap, CheckCircle, AlertCircle, X } from 'lucide-react';

interface AgentMetadata {
  name: string;
  version: string;
  description: string;
  capabilities: string[];
  author: string;
}

interface WASMAgentUploaderProps {
  onAgentUploaded?: (metadata: AgentMetadata) => void;
  onUploadError?: (error: string) => void;
  cognitiveEngine?: any; // CognitiveEngine instance
  isOpen: boolean;
  onClose: () => void;
}

const WASMAgentUploader: React.FC<WASMAgentUploaderProps> = ({
  onAgentUploaded,
  onUploadError,
  cognitiveEngine,
  isOpen,
  onClose
}) => {
  const [isDragging, setIsDragging] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadStatus, setUploadStatus] = useState<'idle' | 'uploading' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState('');
  const [metadata, setMetadata] = useState<AgentMetadata>({
    name: '',
    version: '1.0.0',
    description: '',
    capabilities: [],
    author: ''
  });
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    
    const files = Array.from(e.dataTransfer.files);
    const wasmFile = files.find(file => file.name.endsWith('.wasm'));
    
    if (wasmFile) {
      setSelectedFile(wasmFile);
      setMetadata(prev => ({
        ...prev,
        name: prev.name || wasmFile.name.replace('.wasm', '')
      }));
    } else {
      setErrorMessage('Please select a .wasm file');
      setUploadStatus('error');
    }
  }, []);

  const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file && file.name.endsWith('.wasm')) {
      setSelectedFile(file);
      setMetadata(prev => ({
        ...prev,
        name: prev.name || file.name.replace('.wasm', '')
      }));
      setUploadStatus('idle');
      setErrorMessage('');
    } else {
      setErrorMessage('Please select a .wasm file');
      setUploadStatus('error');
    }
  }, []);

  const handleMetadataChange = useCallback((field: keyof AgentMetadata, value: string | string[]) => {
    setMetadata(prev => ({
      ...prev,
      [field]: value
    }));
  }, []);

  const handleCapabilitiesChange = useCallback((value: string) => {
    const capabilities = value.split(',').map(cap => cap.trim()).filter(cap => cap.length > 0);
    handleMetadataChange('capabilities', capabilities);
  }, [handleMetadataChange]);

  const uploadAgent = useCallback(async () => {
    if (!selectedFile || !cognitiveEngine) {
      setErrorMessage('No file selected or cognitive engine not available');
      setUploadStatus('error');
      return;
    }

    setIsUploading(true);
    setUploadStatus('uploading');
    setUploadProgress(0);

    try {
      // Read file as ArrayBuffer
      const arrayBuffer = await selectedFile.arrayBuffer();
      const wasmBytes = new Uint8Array(arrayBuffer);

      // Simulate upload progress
      const progressInterval = setInterval(() => {
        setUploadProgress(prev => {
          if (prev >= 90) {
            clearInterval(progressInterval);
            return 90;
          }
          return prev + 10;
        });
      }, 100);

      // Upload to cognitive engine
      const success = await cognitiveEngine.uploadWASMAgent(wasmBytes, metadata);

      clearInterval(progressInterval);
      setUploadProgress(100);

      if (success) {
        setUploadStatus('success');
        onAgentUploaded?.(metadata);
        
        // Auto-close after success
        setTimeout(() => {
          onClose();
          resetForm();
        }, 2000);
      } else {
        throw new Error('Upload failed');
      }

    } catch (error) {
      setUploadStatus('error');
      const errorMsg = error instanceof Error ? error.message : 'Upload failed';
      setErrorMessage(errorMsg);
      onUploadError?.(errorMsg);
    } finally {
      setIsUploading(false);
    }
  }, [selectedFile, cognitiveEngine, metadata, onAgentUploaded, onUploadError, onClose]);

  const resetForm = useCallback(() => {
    setSelectedFile(null);
    setMetadata({
      name: '',
      version: '1.0.0',
      description: '',
      capabilities: [],
      author: ''
    });
    setUploadStatus('idle');
    setUploadProgress(0);
    setErrorMessage('');
    setIsUploading(false);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, []);

  const handleClose = useCallback(() => {
    resetForm();
    onClose();
  }, [resetForm, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <div className="flex items-center space-x-3">
            <Cpu className="w-6 h-6 text-blue-600" />
            <h2 className="text-xl font-semibold text-gray-900">Upload WASM Agent</h2>
          </div>
          <button
            onClick={handleClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        <div className="p-6 space-y-6">
          {/* File Upload Area */}
          <div
            className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
              isDragging
                ? 'border-blue-500 bg-blue-50'
                : selectedFile
                ? 'border-green-500 bg-green-50'
                : 'border-gray-300 hover:border-gray-400'
            }`}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
          >
            {selectedFile ? (
              <div className="space-y-2">
                <CheckCircle className="w-12 h-12 text-green-500 mx-auto" />
                <p className="text-lg font-medium text-green-700">{selectedFile.name}</p>
                <p className="text-sm text-green-600">
                  {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                </p>
              </div>
            ) : (
              <div className="space-y-4">
                <Upload className="w-12 h-12 text-gray-400 mx-auto" />
                <div>
                  <p className="text-lg font-medium text-gray-900">
                    Drop your agent.wasm file here
                  </p>
                  <p className="text-sm text-gray-500">or click to browse</p>
                </div>
                <button
                  onClick={() => fileInputRef.current?.click()}
                  className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
                >
                  Select File
                </button>
              </div>
            )}
            
            <input
              ref={fileInputRef}
              type="file"
              accept=".wasm"
              onChange={handleFileSelect}
              className="hidden"
            />
          </div>

          {/* Metadata Form */}
          {selectedFile && (
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900">Agent Metadata</h3>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Agent Name *
                  </label>
                  <input
                    type="text"
                    value={metadata.name}
                    onChange={(e) => handleMetadataChange('name', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="My Custom Agent"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Version
                  </label>
                  <input
                    type="text"
                    value={metadata.version}
                    onChange={(e) => handleMetadataChange('version', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="1.0.0"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Author
                  </label>
                  <input
                    type="text"
                    value={metadata.author}
                    onChange={(e) => handleMetadataChange('author', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Your Name"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Capabilities
                  </label>
                  <input
                    type="text"
                    value={metadata.capabilities.join(', ')}
                    onChange={(e) => handleCapabilitiesChange(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="coding, analysis, reasoning"
                  />
                  <p className="text-xs text-gray-500 mt-1">Comma-separated list</p>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Description
                </label>
                <textarea
                  value={metadata.description}
                  onChange={(e) => handleMetadataChange('description', e.target.value)}
                  rows={3}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Describe what this agent does..."
                />
              </div>
            </div>
          )}

          {/* Upload Progress */}
          {uploadStatus === 'uploading' && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-gray-700">Uploading...</span>
                <span className="text-sm text-gray-500">{uploadProgress}%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${uploadProgress}%` }}
                />
              </div>
            </div>
          )}

          {/* Status Messages */}
          {uploadStatus === 'success' && (
            <div className="flex items-center space-x-2 text-green-600">
              <CheckCircle className="w-5 h-5" />
              <span>Agent uploaded successfully!</span>
            </div>
          )}

          {uploadStatus === 'error' && (
            <div className="flex items-center space-x-2 text-red-600">
              <AlertCircle className="w-5 h-5" />
              <span>{errorMessage}</span>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end space-x-3 p-6 border-t bg-gray-50">
          <button
            onClick={handleClose}
            className="px-4 py-2 text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={uploadAgent}
            disabled={!selectedFile || !metadata.name || isUploading || uploadStatus === 'success'}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center space-x-2"
          >
            {isUploading ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white" />
                <span>Uploading...</span>
              </>
            ) : (
              <>
                <Zap className="w-4 h-4" />
                <span>Upload Agent</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};

export default WASMAgentUploader;
