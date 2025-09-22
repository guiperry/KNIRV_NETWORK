import * as React from 'react';
import { useState, useEffect } from 'react';
import { Brain, Settings, Download, Play, Save, Trash2, BarChart3, Layers, Cpu } from 'lucide-react';
import { personalKNIRVGRAPHService, GraphNode } from '../services/PersonalKNIRVGRAPHService';
import { cortexTrainingService, TrainingConfig, CortexModel, TrainingProgress } from '../services/CortexTrainingService';

interface CortexBuilderProps {
  isOpen: boolean;
  onClose: () => void;
}

// Use imported types from service
type ModelVersion = CortexModel;

export const CortexBuilder: React.FC<CortexBuilderProps> = ({ isOpen, onClose }) => {
  const [activeTab, setActiveTab] = useState<'overview' | 'training' | 'data' | 'versions'>('overview');
  const [trainingConfig, setTrainingConfig] = useState<TrainingConfig>({
    learningRate: 0.001,
    batchSize: 32,
    epochs: 100,
    validationSplit: 0.2,
    optimizerType: 'adam',
    lossFunction: 'mse'
  });
  const [modelVersions, setModelVersions] = useState<ModelVersion[]>([]);
  const [isTraining, setIsTraining] = useState(false);
  const [trainingProgress, setTrainingProgress] = useState(0);
  const [graphData, setGraphData] = useState<GraphNode[]>([]);

  useEffect(() => {
    if (isOpen) {
      loadGraphData();
      loadModelVersions();
    }
  }, [isOpen]);

  const loadGraphData = async () => {
    try {
      const graph = personalKNIRVGRAPHService.getCurrentGraph();
      if (graph) {
        setGraphData(graph.nodes);
      }
    } catch (error) {
      console.error('Failed to load graph data:', error);
    }
  };

  const loadModelVersions = async () => {
    try {
      const models = await cortexTrainingService.getSavedModels();
      setModelVersions(models);
    } catch (error) {
      console.error('Failed to load model versions:', error);
      setModelVersions([]);
    }
  };

  const handleStartTraining = async () => {
    setIsTraining(true);
    setTrainingProgress(0);

    try {
      const newModel = await cortexTrainingService.trainModel(
        trainingConfig,
        (progress: TrainingProgress) => {
          setTrainingProgress((progress.epoch / progress.totalEpochs) * 100);
        }
      );

      // Store the new model and refresh model versions
      console.log('Training completed successfully:', newModel.id);
      await loadModelVersions();

      setIsTraining(false);
      setTrainingProgress(100);
    } catch (error) {
      console.error('Training failed:', error);
      setIsTraining(false);
      setTrainingProgress(0);
      alert(`Training failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const getDataStats = () => {
    return {
      errorNodes: graphData.filter(n => n.type === 'error').length,
      skillNodes: graphData.filter(n => n.type === 'skill').length,
      capabilityNodes: graphData.filter(n => n.type === 'capability').length,
      propertyNodes: graphData.filter(n => n.type === 'property').length,
      totalNodes: graphData.length
    };
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-gray-900 rounded-lg border border-gray-700 w-full max-w-6xl h-5/6 flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-700">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-gradient-to-r from-purple-500 to-pink-500 rounded-lg flex items-center justify-center">
              <Brain className="w-6 h-6 text-white" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">CORTEX Builder</h2>
              <p className="text-sm text-gray-400">Train Your Own KNIRVCORTEX</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors"
          >
            ✕
          </button>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-gray-700">
          {[
            { id: 'overview', label: 'Overview', icon: BarChart3 },
            { id: 'training', label: 'Training', icon: Settings },
            { id: 'data', label: 'Data', icon: Layers },
            { id: 'versions', label: 'Versions', icon: Cpu }
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as 'overview' | 'training' | 'data' | 'versions')}
              className={`flex items-center space-x-2 px-6 py-3 border-b-2 transition-colors ${
                activeTab === tab.id
                  ? 'border-purple-500 text-purple-400'
                  : 'border-transparent text-gray-400 hover:text-white'
              }`}
            >
              <tab.icon className="w-4 h-4" />
              <span>{tab.label}</span>
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 p-6 overflow-auto">
          {activeTab === 'overview' && (
            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="bg-gray-800 rounded-lg p-4">
                  <div className="flex items-center space-x-3 mb-3">
                    <Brain className="w-5 h-5 text-purple-400" />
                    <h3 className="font-medium text-white">Current Model</h3>
                  </div>
                  <p className="text-2xl font-bold text-white">
                    {modelVersions.length > 0 ? modelVersions[modelVersions.length - 1].name : 'No Model'}
                  </p>
                  <p className="text-sm text-gray-400">
                    {modelVersions.length > 0 ? `Accuracy: ${(modelVersions[modelVersions.length - 1].metrics.accuracy * 100).toFixed(1)}%` : 'Train your first model'}
                  </p>
                </div>

                <div className="bg-gray-800 rounded-lg p-4">
                  <div className="flex items-center space-x-3 mb-3">
                    <Layers className="w-5 h-5 text-blue-400" />
                    <h3 className="font-medium text-white">Training Data</h3>
                  </div>
                  <p className="text-2xl font-bold text-white">{getDataStats().totalNodes}</p>
                  <p className="text-sm text-gray-400">Total nodes in personal graph</p>
                </div>

                <div className="bg-gray-800 rounded-lg p-4">
                  <div className="flex items-center space-x-3 mb-3">
                    <Cpu className="w-5 h-5 text-green-400" />
                    <h3 className="font-medium text-white">Model Versions</h3>
                  </div>
                  <p className="text-2xl font-bold text-white">{modelVersions.length}</p>
                  <p className="text-sm text-gray-400">Trained iterations</p>
                </div>
              </div>

              <div className="bg-gray-800 rounded-lg p-6">
                <h3 className="text-lg font-medium text-white mb-4">Personal KNIRVGRAPH Integration</h3>
                <p className="text-gray-300 mb-4">
                  Your CORTEX model is trained on your personal KNIRVGRAPH data, learning from your unique patterns of errors, skills, capabilities, and ideas.
                </p>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  {Object.entries(getDataStats()).filter(([key]) => key !== 'totalNodes').map(([type, count]) => (
                    <div key={type} className="text-center">
                      <div className="text-2xl font-bold text-white">{count}</div>
                      <div className="text-sm text-gray-400 capitalize">{type.replace('Nodes', ' Nodes')}</div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {activeTab === 'training' && (
            <div className="space-y-6">
              <div className="bg-gray-800 rounded-lg p-6">
                <h3 className="text-lg font-medium text-white mb-4">Training Configuration</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Learning Rate</label>
                    <input
                      type="number"
                      step="0.0001"
                      value={trainingConfig.learningRate}
                      onChange={(e) => setTrainingConfig(prev => ({ ...prev, learningRate: parseFloat(e.target.value) }))}
                      className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Batch Size</label>
                    <input
                      type="number"
                      value={trainingConfig.batchSize}
                      onChange={(e) => setTrainingConfig(prev => ({ ...prev, batchSize: parseInt(e.target.value) }))}
                      className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Epochs</label>
                    <input
                      type="number"
                      value={trainingConfig.epochs}
                      onChange={(e) => setTrainingConfig(prev => ({ ...prev, epochs: parseInt(e.target.value) }))}
                      className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Validation Split</label>
                    <input
                      type="number"
                      step="0.1"
                      min="0"
                      max="1"
                      value={trainingConfig.validationSplit}
                      onChange={(e) => setTrainingConfig(prev => ({ ...prev, validationSplit: parseFloat(e.target.value) }))}
                      className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white"
                    />
                  </div>
                </div>

                <div className="mt-6 flex items-center justify-between">
                  <div className="flex space-x-4">
                    <button
                      onClick={handleStartTraining}
                      disabled={isTraining || getDataStats().totalNodes < 5}
                      className="flex items-center space-x-2 px-4 py-2 bg-purple-600 hover:bg-purple-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-md transition-colors"
                    >
                      <Play className="w-4 h-4" />
                      <span>{isTraining ? 'Training...' : 'Start Training'}</span>
                    </button>
                    <button className="flex items-center space-x-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-md transition-colors">
                      <Save className="w-4 h-4" />
                      <span>Save Config</span>
                    </button>
                  </div>
                  {getDataStats().totalNodes < 5 && (
                    <p className="text-sm text-yellow-400">Need at least 5 nodes to start training</p>
                  )}
                </div>

                {isTraining && (
                  <div className="mt-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-sm text-gray-300">Training Progress</span>
                      <span className="text-sm text-gray-300">{trainingProgress.toFixed(0)}%</span>
                    </div>
                    <div className="w-full bg-gray-700 rounded-full h-2">
                      <div
                        className="bg-purple-600 h-2 rounded-full transition-all duration-300"
                        style={{ width: `${trainingProgress}%` }}
                      />
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTab === 'data' && (
            <div className="space-y-6">
              <div className="bg-gray-800 rounded-lg p-6">
                <h3 className="text-lg font-medium text-white mb-4">Personal KNIRVGRAPH Data</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
                  {Object.entries(getDataStats()).filter(([key]) => key !== 'totalNodes').map(([type, count]) => (
                    <div key={type} className="bg-gray-700 rounded-lg p-4">
                      <div className="text-2xl font-bold text-white">{count}</div>
                      <div className="text-sm text-gray-400 capitalize">{type.replace('Nodes', ' Nodes')}</div>
                    </div>
                  ))}
                </div>
                
                <div className="space-y-4">
                  <h4 className="font-medium text-white">Recent Nodes</h4>
                  <div className="space-y-2 max-h-64 overflow-auto">
                    {graphData.slice(0, 10).map(node => (
                      <div key={node.id} className="flex items-center justify-between p-3 bg-gray-700 rounded-md">
                        <div>
                          <div className="font-medium text-white">{node.label}</div>
                          <div className="text-sm text-gray-400 capitalize">{node.type} node</div>
                        </div>
                        <div className={`px-2 py-1 rounded text-xs ${
                          node.type === 'error' ? 'bg-red-500/20 text-red-400' :
                          node.type === 'skill' ? 'bg-green-500/20 text-green-400' :
                          node.type === 'capability' ? 'bg-blue-500/20 text-blue-400' :
                          node.type === 'property' ? 'bg-yellow-500/20 text-yellow-400' :
                          'bg-gray-500/20 text-gray-400'
                        }`}>
                          {node.type}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'versions' && (
            <div className="space-y-6">
              <div className="bg-gray-800 rounded-lg p-6">
                <h3 className="text-lg font-medium text-white mb-4">Model Versions</h3>
                <div className="space-y-4">
                  {modelVersions.map(version => (
                    <div key={version.id} className="flex items-center justify-between p-4 bg-gray-700 rounded-lg">
                      <div className="flex-1">
                        <div className="flex items-center space-x-3 mb-2">
                          <h4 className="font-medium text-white">{version.name}</h4>
                          <span className="px-2 py-1 bg-purple-500/20 text-purple-400 rounded text-xs">
                            v{version.version}
                          </span>
                        </div>
                        <p className="text-sm text-gray-400 mb-2">{version.description}</p>
                        <div className="flex items-center space-x-4 text-xs text-gray-500">
                          <span>Accuracy: {(version.metrics.accuracy * 100).toFixed(1)}%</span>
                          <span>Nodes: {version.graphSnapshot.nodeCount}</span>
                          <span>Created: {version.createdAt.toLocaleDateString()}</span>
                        </div>
                      </div>
                      <div className="flex items-center space-x-2">
                        <button className="p-2 text-gray-400 hover:text-white transition-colors">
                          <Download className="w-4 h-4" />
                        </button>
                        <button className="p-2 text-gray-400 hover:text-red-400 transition-colors">
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
