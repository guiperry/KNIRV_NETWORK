import React, { useState, useEffect, useCallback } from 'react';
import { Brain, Download, CheckCircle, AlertCircle, Info, Zap, Settings } from 'lucide-react';

interface ModelDefinition {
  id: string;
  name: string;
  description: string;
  type: 'hrm' | 'cortex' | 'phi3' | 'gemma' | 'llama';
  size: string;
  parameters: string;
  capabilities: string[];
  license: string;
  source: 'builtin' | 'huggingface' | 'custom';
  architecture: string;
  contextLength: number;
  recommended: boolean;
}

interface ModelStatus {
  id: string;
  loaded: boolean;
  initialized: boolean;
  size: number;
  loadTime?: number;
  error?: string;
}

interface ModelSelectorProps {
  modelManager?: any;
  wasmOrchestrator?: any;
  onModelSelected?: (modelId: string) => void;
  onModelLoaded?: (modelId: string) => void;
  currentModelId?: string;
}

const ModelSelector: React.FC<ModelSelectorProps> = ({
  modelManager,
  wasmOrchestrator,
  onModelSelected,
  onModelLoaded,
  currentModelId
}) => {
  const [models, setModels] = useState<ModelDefinition[]>([]);
  const [modelStatuses, setModelStatuses] = useState<Map<string, ModelStatus>>(new Map());
  const [selectedModel, setSelectedModel] = useState<string | null>(currentModelId || null);
  const [isLoading, setIsLoading] = useState(false);
  const [showInstructions, setShowInstructions] = useState<string | null>(null);
  const [filterType, setFilterType] = useState<'all' | 'recommended' | 'builtin' | 'huggingface'>('all');

  useEffect(() => {
    if (modelManager) {
      loadModels();
      setupEventListeners();
    }
  }, [modelManager]);

  const loadModels = useCallback(() => {
    if (!modelManager) return;

    const availableModels = modelManager.getAvailableModels();
    const statuses = modelManager.getAllModelStatuses();
    
    setModels(availableModels);
    
    const statusMap = new Map();
    statuses.forEach((status: ModelStatus) => {
      statusMap.set(status.id, status);
    });
    setModelStatuses(statusMap);

    // Set default selection
    if (!selectedModel && availableModels.length > 0) {
      const defaultModel = modelManager.getDefaultModel();
      setSelectedModel(defaultModel.id);
    }
  }, [modelManager, selectedModel]);

  const setupEventListeners = useCallback(() => {
    if (!modelManager) return;

    const handleModelStatusUpdate = (data: { id: string; status: ModelStatus }) => {
      setModelStatuses(prev => new Map(prev.set(data.id, data.status)));
    };

    const handleCurrentModelChanged = (data: { currentModel: string }) => {
      setSelectedModel(data.currentModel);
      onModelSelected?.(data.currentModel);
    };

    modelManager.on('model_status_updated', handleModelStatusUpdate);
    modelManager.on('current_model_changed', handleCurrentModelChanged);

    return () => {
      modelManager.off('model_status_updated', handleModelStatusUpdate);
      modelManager.off('current_model_changed', handleCurrentModelChanged);
    };
  }, [modelManager, onModelSelected]);

  const handleModelSelect = useCallback(async (modelId: string) => {
    if (!modelManager || !wasmOrchestrator) return;

    setIsLoading(true);
    try {
      // Check if model is available
      const isAvailable = await modelManager.isModelAvailable(modelId);
      if (!isAvailable) {
        setShowInstructions(modelId);
        setIsLoading(false);
        return;
      }

      // Set as current model
      modelManager.setCurrentModel(modelId);
      
      // Load model in orchestrator
      const model = modelManager.getModel(modelId);
      if (model) {
        const modelConfig = {
          modelType: model.type,
          modelPath: model.wasmPath,
          weightsPath: model.weightsPath,
          maxTokens: 2048,
          temperature: 0.7,
          topP: 0.9,
          contextLength: model.contextLength
        };

        const success = await wasmOrchestrator.switchModel(modelConfig);
        if (success) {
          onModelLoaded?.(modelId);
        }
      }

    } catch (error) {
      console.error('Failed to select model:', error);
    } finally {
      setIsLoading(false);
    }
  }, [modelManager, wasmOrchestrator, onModelLoaded]);

  const getFilteredModels = useCallback(() => {
    switch (filterType) {
      case 'recommended':
        return models.filter(model => model.recommended);
      case 'builtin':
        return models.filter(model => model.source === 'builtin');
      case 'huggingface':
        return models.filter(model => model.source === 'huggingface');
      default:
        return models;
    }
  }, [models, filterType]);

  const getModelTypeIcon = (type: string) => {
    switch (type) {
      case 'hrm':
        return <Brain className="w-5 h-5 text-purple-600" />;
      case 'cortex':
        return <Zap className="w-5 h-5 text-blue-600" />;
      case 'phi3':
        return <Settings className="w-5 h-5 text-green-600" />;
      case 'gemma':
        return <Brain className="w-5 h-5 text-orange-600" />;
      case 'llama':
        return <Brain className="w-5 h-5 text-red-600" />;
      default:
        return <Brain className="w-5 h-5 text-gray-600" />;
    }
  };

  const getStatusIcon = (status: ModelStatus | undefined) => {
    if (!status) return <AlertCircle className="w-4 h-4 text-gray-400" />;
    
    if (status.error) return <AlertCircle className="w-4 h-4 text-red-500" />;
    if (status.initialized) return <CheckCircle className="w-4 h-4 text-green-500" />;
    if (status.loaded) return <Download className="w-4 h-4 text-blue-500" />;
    
    return <AlertCircle className="w-4 h-4 text-gray-400" />;
  };

  return (
    <div className="bg-white rounded-lg shadow-lg p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <Brain className="w-6 h-6 text-blue-600" />
          <h2 className="text-xl font-bold text-gray-900">Model Selection</h2>
        </div>
        
        <div className="flex items-center space-x-2">
          <select
            value={filterType}
            onChange={(e) => setFilterType(e.target.value as any)}
            className="px-3 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">All Models</option>
            <option value="recommended">Recommended</option>
            <option value="builtin">Built-in</option>
            <option value="huggingface">HuggingFace</option>
          </select>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {getFilteredModels().map((model) => {
          const status = modelStatuses.get(model.id);
          const isSelected = selectedModel === model.id;
          const isCurrentModel = currentModelId === model.id;

          return (
            <div
              key={model.id}
              className={`border rounded-lg p-4 cursor-pointer transition-all ${
                isSelected
                  ? 'border-blue-500 bg-blue-50'
                  : isCurrentModel
                  ? 'border-green-500 bg-green-50'
                  : 'border-gray-200 hover:border-gray-300'
              }`}
              onClick={() => setSelectedModel(model.id)}
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center space-x-2">
                  {getModelTypeIcon(model.type)}
                  <h3 className="font-semibold text-gray-900">{model.name}</h3>
                  {model.recommended && (
                    <span className="px-2 py-1 bg-yellow-100 text-yellow-800 text-xs rounded-full">
                      Recommended
                    </span>
                  )}
                </div>
                {getStatusIcon(status)}
              </div>

              <p className="text-sm text-gray-600 mb-3">{model.description}</p>

              <div className="grid grid-cols-2 gap-2 text-xs text-gray-500 mb-3">
                <div>Parameters: {model.parameters}</div>
                <div>Size: {model.size}</div>
                <div>Context: {model.contextLength.toLocaleString()}</div>
                <div>License: {model.license}</div>
              </div>

              <div className="flex flex-wrap gap-1 mb-3">
                {model.capabilities.slice(0, 3).map((capability) => (
                  <span
                    key={capability}
                    className="px-2 py-1 bg-gray-100 text-gray-700 text-xs rounded"
                  >
                    {capability}
                  </span>
                ))}
                {model.capabilities.length > 3 && (
                  <span className="px-2 py-1 bg-gray-100 text-gray-700 text-xs rounded">
                    +{model.capabilities.length - 3} more
                  </span>
                )}
              </div>

              <div className="flex items-center justify-between">
                <span className={`text-xs px-2 py-1 rounded ${
                  model.source === 'builtin'
                    ? 'bg-green-100 text-green-800'
                    : model.source === 'huggingface'
                    ? 'bg-blue-100 text-blue-800'
                    : 'bg-gray-100 text-gray-800'
                }`}>
                  {model.source}
                </span>

                {isCurrentModel && (
                  <span className="text-xs text-green-600 font-medium">
                    Current Model
                  </span>
                )}
              </div>

              {status?.error && (
                <div className="mt-2 text-xs text-red-600">
                  Error: {status.error}
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* Action Buttons */}
      <div className="flex items-center justify-between mt-6 pt-6 border-t">
        <div className="flex items-center space-x-2">
          {selectedModel && (
            <button
              onClick={() => setShowInstructions(selectedModel)}
              className="flex items-center space-x-2 px-3 py-2 text-blue-600 border border-blue-600 rounded-md hover:bg-blue-50 transition-colors"
            >
              <Info className="w-4 h-4" />
              <span>Instructions</span>
            </button>
          )}
        </div>

        <div className="flex items-center space-x-3">
          <button
            onClick={() => selectedModel && handleModelSelect(selectedModel)}
            disabled={!selectedModel || isLoading}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center space-x-2"
          >
            {isLoading ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white" />
                <span>Loading...</span>
              </>
            ) : (
              <>
                <Download className="w-4 h-4" />
                <span>Load Model</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Instructions Modal */}
      {showInstructions && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] overflow-y-auto">
            <div className="p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900">
                  Setup Instructions
                </h3>
                <button
                  onClick={() => setShowInstructions(null)}
                  className="text-gray-400 hover:text-gray-600"
                >
                  ×
                </button>
              </div>
              
              <div className="prose prose-sm max-w-none">
                <pre className="bg-gray-100 p-4 rounded-md text-sm overflow-x-auto whitespace-pre-wrap">
                  {modelManager?.getModelInstructions(showInstructions)}
                </pre>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ModelSelector;
