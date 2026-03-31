import React from 'react';

import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, Bot, Upload, Coins } from 'lucide-react';
import { agentManagementService, AgentUploadRequest } from '../services/AgentManagementService';
import { walletIntegrationService } from '../services/WalletIntegrationService';

interface CortexModel {
  agentId: string;
  name: string;
  version: string;
  type: 'wasm';
  status: 'Available' | 'Deployed' | 'Error' | 'Compiling';
  nrnCost: number;
  capabilities: string[];
  metadata: {
    name: string;
    version: string;
    description: string;
    author: string;
    capabilities: string[];
    requirements: {
      memory: number;
      cpu: number;
      storage: number;
    };
  };
  createdAt: string;
  lastActivity?: string;
}

export default function CortexModels() {
  const navigate = useNavigate();
  const [models, setModels] = useState<CortexModel[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [uploadingModel, setUploadingModel] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [nrnBalance, setNrnBalance] = useState(1250);
  const [, setSelectedFile] = useState<File | null>(null);

  // Load cortex models
  useEffect(() => {
    const loadModels = async () => {
      try {
        setIsLoading(true);
        const availableModels = await agentManagementService.getAgents();
        const cortexModelsOnly = availableModels.filter(agent => agent.type === 'wasm') as CortexModel[];
        setModels(cortexModelsOnly);

        // Get wallet balance
        const currentAccount = walletIntegrationService.getCurrentAccount();
        if (currentAccount) {
          // In a real implementation, you'd get the balance from the wallet
          setNrnBalance(1250); // Mock balance
        }

        console.log('Cortex models loaded:', cortexModelsOnly.length);
      } catch (error) {
        console.error('Failed to load cortex models:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadModels();

    // Refresh every 30 seconds
    const interval = setInterval(loadModels, 30000);
    return () => clearInterval(interval);
  }, []);

  // Handle cortex model file upload
  const handleModelUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setSelectedFile(file);
    setUploadingModel(true);

    try {
      const uploadRequest: AgentUploadRequest = {
        file,
        metadata: {
          name: file.name.replace(/\.[^/.]+$/, ''), // Remove file extension
          description: `Uploaded cortex.wasm neural intelligence model from ${file.name}`,
          author: 'User'
        },
        type: 'wasm'
      };

      const newModel = await agentManagementService.uploadAgent(uploadRequest);

      // Refresh models list
      const updatedModels = await agentManagementService.getAgents();
      const cortexModelsOnly = updatedModels.filter(agent => agent.type === 'wasm') as CortexModel[];
      setModels(cortexModelsOnly);

      console.log('Cortex model uploaded successfully:', newModel);
    } catch (error) {
      console.error('Model upload failed:', error);
      alert(`Model upload failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    } finally {
      setUploadingModel(false);
      setSelectedFile(null);
      // Reset file input
      event.target.value = '';
    }
  };

  // Handle model click for navigation
  const handleModelClick = (model: CortexModel) => {
    navigate(`/cortex-models/${model.agentId}`);
  };

  // Filter models based on search
  const filteredModels = models.filter(model =>
    model.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    model.metadata.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
    model.capabilities.some(cap => cap.toLowerCase().includes(searchTerm.toLowerCase()))
  );

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Available': return 'text-green-400';
      case 'Deployed': return 'text-blue-400';
      case 'Compiling': return 'text-yellow-400';
      case 'Error': return 'text-red-500';
      default: return 'text-gray-400';
    }
  };

  const getStatusBg = (status: string) => {
    switch (status) {
      case 'Available': return 'bg-green-500/20 border-green-500/30';
      case 'Deployed': return 'bg-blue-500/20 border-blue-500/30';
      case 'Compiling': return 'bg-yellow-500/20 border-yellow-500/30';
      case 'Error': return 'bg-red-500/20 border-red-500/30';
      default: return 'bg-gray-500/20 border-gray-500/30';
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white relative overflow-hidden">
      <div className="max-w-6xl mx-auto p-4 pb-24 overflow-y-auto h-screen">
        <div className="space-y-6">
          {/* Header */}
          <div className="text-center py-4">
            <h1 className="text-2xl font-bold bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent mb-2">
              Cortex Models
            </h1>
            <p className="text-gray-400 text-sm">
              Manage and deploy neural intelligence models
            </p>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-2xl font-bold text-white">{models.length}</div>
              <div className="text-xs text-gray-400">Total Models</div>
            </div>
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-2xl font-bold text-green-400">
                {models.filter(m => m.status === 'Available').length}
              </div>
              <div className="text-xs text-gray-400">Available</div>
            </div>
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-2xl font-bold text-yellow-400">{nrnBalance.toLocaleString()}</div>
              <div className="text-xs text-gray-400">NRN Balance</div>
            </div>
          </div>

          {/* Search and Upload */}
          <div className="space-y-3">
            <div className="flex space-x-3">
              <div className="flex-1 relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
                <input
                  type="text"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  placeholder="Search cortex models..."
                  className="w-full pl-10 pr-4 py-3 bg-gray-800/80 border border-gray-600/50 rounded-lg focus:border-blue-500/50 focus:outline-none text-white placeholder-gray-400"
                />
              </div>
              <div className="flex items-center space-x-2 px-4 py-3 bg-gray-800/80 border border-gray-600/50 rounded-lg">
                <input
                  type="file"
                  accept=".wasm"
                  onChange={handleModelUpload}
                  disabled={uploadingModel}
                  className="hidden"
                  id="cortex-upload"
                />
                <label
                  htmlFor="cortex-upload"
                  className={`flex items-center space-x-2 cursor-pointer ${
                    uploadingModel ? 'text-gray-500' : 'text-blue-400 hover:text-blue-300'
                  }`}
                >
                  <Upload className="w-4 h-4" />
                  <span className="text-sm font-medium">
                    {uploadingModel ? 'Uploading...' : 'Upload Model'}
                  </span>
                </label>
              </div>
            </div>
          </div>

          {/* Models Grid */}
          <div className="space-y-4">
            {isLoading ? (
              <div className="text-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-400 mx-auto mb-4"></div>
                <p className="text-gray-400">Loading cortex models...</p>
              </div>
            ) : filteredModels.length === 0 ? (
              <div className="text-center py-8">
                <Bot className="w-12 h-12 mx-auto mb-4 text-gray-500 opacity-50" />
                <p className="text-gray-400 mb-2">
                  {searchTerm ? 'No models found matching your search' : 'No cortex models available'}
                </p>
                <p className="text-sm text-gray-500">
                  {searchTerm ? 'Try adjusting your search terms' : 'Upload a .wasm neural intelligence model to get started'}
                </p>
              </div>
            ) : (
              filteredModels.map((model) => (
                <div
                  key={model.agentId}
                  className={`relative group p-4 bg-gray-800/90 backdrop-blur-xl rounded-2xl border hover:border-purple-500/50 transition-all ${getStatusBg(model.status)}`}
                >
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center space-x-3">
                      <div className="relative">
                        <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-cyan-500 rounded-xl flex items-center justify-center">
                          <Bot className="w-5 h-5 text-white" />
                        </div>
                        <div className={`absolute -top-1 -right-1 w-4 h-4 rounded-full border-2 border-gray-800 ${getStatusBg(model.status)} flex items-center justify-center`}>
                          <div className={`w-2.5 h-2.5 rounded-full ${getStatusColor(model.status).replace('text-', 'bg-')}`}></div>
                        </div>
                      </div>
                      <div>
                        <button
                          onClick={() => handleModelClick(model)}
                          className="text-white font-semibold hover:text-blue-400 transition-colors text-left"
                        >
                          {model.name}
                        </button>
                        <p className="text-xs text-slate-400 capitalize">{model.status.toLowerCase()}</p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <div className="px-2 py-1 bg-blue-500/20 text-blue-400 text-xs rounded">
                        v{model.metadata.version}
                      </div>
                      <div className="flex items-center space-x-1 text-yellow-400">
                        <Coins className="w-3 h-3" />
                        <span className="text-xs font-medium">{model.nrnCost}</span>
                      </div>
                    </div>
                  </div>

                  <div className="space-y-3">
                    <p className="text-sm text-gray-300">{model.metadata.description}</p>

                    <div className="flex flex-wrap gap-1">
                      {model.capabilities.slice(0, 4).map((capability, index) => (
                        <span
                          key={index}
                          className="px-2 py-1 bg-purple-500/20 text-purple-400 text-xs rounded"
                        >
                          {capability}
                        </span>
                      ))}
                      {model.capabilities.length > 4 && (
                        <span className="px-2 py-1 bg-gray-500/20 text-gray-400 text-xs rounded">
                          +{model.capabilities.length - 4} more
                        </span>
                      )}
                    </div>

                    <div className="flex justify-between items-center text-xs">
                      <div className="flex items-center space-x-4 text-slate-500">
                        <span>Memory: {model.metadata.requirements.memory}MB</span>
                        <span>CPU: {model.metadata.requirements.cpu}</span>
                        <span>Storage: {model.metadata.requirements.storage}MB</span>
                      </div>
                      {model.lastActivity && (
                        <span className="text-slate-400">
                          Last active: {new Date(model.lastActivity).toLocaleDateString()}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>

          {/* Upload Instructions */}
          <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6 text-center">
            <h3 className="text-lg font-semibold text-white mb-2">Upload Neural Intelligence Models</h3>
            <p className="text-sm text-gray-400 mb-4">
              Deploy advanced AI capabilities with compiled WebAssembly neural networks
            </p>
            <div className="text-xs text-gray-500 space-y-1">
              <p>• Models must be compiled to WebAssembly (.wasm) format</p>
              <p>• Ensure models include proper metadata and capability definitions</p>
              <p>• Higher complexity models require more NRN for deployment</p>
            </div>
          </div>
        </div>
      </div>

      {/* Bottom Navigation */}
      <nav className="fixed bottom-0 left-0 right-0 z-20 border-t border-gray-600/50 backdrop-blur-xl bg-gray-900/80">
        <div className="grid grid-cols-3 px-2 py-2">
          <button
            onClick={() => navigate('/manager/skills')}
            className={`flex flex-col items-center py-2 px-1 rounded-lg transition-colors text-gray-400 hover:text-white`}
          >
            <Bot className="w-5 h-5 mb-1" />
            <span className="text-xs">Skills</span>
          </button>
          <button
            onClick={() => navigate('/manager/cortex-models')}
            className={`flex flex-col items-center py-2 px-1 rounded-lg transition-colors ${
              window.location.pathname === '/manager/cortex-models' ? 'text-blue-400 bg-blue-600/20' : 'text-gray-400 hover:text-white'
            }`}
          >
            <Bot className="w-5 h-5 mb-1" />
            <span className="text-xs">Cortex</span>
          </button>
          <button
            onClick={() => navigate('/manager/wallet')}
            className={`flex flex-col items-center py-2 px-1 rounded-lg transition-colors text-gray-400 hover:text-white`}
          >
            <Coins className="w-5 h-5 mb-1" />
            <span className="text-xs">Wallet</span>
          </button>
        </div>
      </nav>
    </div>
  );
}
