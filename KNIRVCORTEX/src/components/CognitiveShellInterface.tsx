import React, { useState, useEffect, useRef } from 'react';
import { Brain, Activity, Zap, Eye, Mic, Settings, BarChart3 } from 'lucide-react';
import { CognitiveEngine, CognitiveConfig, CognitiveState } from '../cognitive-shell/CognitiveEngine';

interface CognitiveShellInterfaceProps {
  onStateChange?: (state: CognitiveState) => void;
  onSkillInvoked?: (skillId: string, result: any) => void;
  onAdaptationTriggered?: (adaptation: any) => void;
}

export const CognitiveShellInterface: React.FC<CognitiveShellInterfaceProps> = ({
  onStateChange,
  onSkillInvoked,
  onAdaptationTriggered,
}) => {
  const [cognitiveEngine, setCognitiveEngine] = useState<CognitiveEngine | null>(null);
  const [engineState, setEngineState] = useState<CognitiveState | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [metrics, setMetrics] = useState<any>(null);
  const [learningMode, setLearningMode] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [config, setConfig] = useState<CognitiveConfig>({
    maxContextSize: 100,
    learningRate: 0.01,
    adaptationThreshold: 0.3,
    skillTimeout: 30000,
    voiceEnabled: true,
    visualEnabled: true,
    loraEnabled: true,
    enhancedLoraEnabled: false,
    hrmEnabled: false,
    adaptiveLearningEnabled: true,
    walletIntegrationEnabled: true,
    chainIntegrationEnabled: true,
    ecosystemCommunicationEnabled: true,
  });

  const engineRef = useRef<CognitiveEngine | null>(null);

  useEffect(() => {
    initializeCognitiveEngine();
    return () => {
      if (engineRef.current) {
        engineRef.current.stop();
      }
    };
  }, []);

  const initializeCognitiveEngine = async () => {
    try {
      const engine = new CognitiveEngine(config);
      engineRef.current = engine;
      setCognitiveEngine(engine);

      // Set up event listeners
      engine.on('engineStarted', () => {
        setIsRunning(true);
        console.log('Cognitive Engine started');
      });

      engine.on('engineStopped', () => {
        setIsRunning(false);
        console.log('Cognitive Engine stopped');
      });

      engine.on('inputProcessed', (data) => {
        console.log('Input processed:', data);
        updateMetrics();
      });

      engine.on('skillInvoked', (data) => {
        console.log('Skill invoked:', data);
        if (onSkillInvoked) {
          onSkillInvoked(data.skillId, data.result);
        }
      });

      engine.on('adaptationTriggered', (data) => {
        console.log('Adaptation triggered:', data);
        if (onAdaptationTriggered) {
          onAdaptationTriggered(data);
        }
      });

      engine.on('learningModeStarted', () => {
        setLearningMode(true);
      });

      engine.on('cognitiveEvent', (event) => {
        console.log('Cognitive event:', event);
      });

      // Update state periodically
      const stateInterval = setInterval(() => {
        if (engine) {
          const state = engine.getState();
          setEngineState(state);
          if (onStateChange) {
            onStateChange(state);
          }
        }
      }, 1000);

      return () => clearInterval(stateInterval);

    } catch (error) {
      console.error('Failed to initialize Cognitive Engine:', error);
    }
  };

  const updateMetrics = () => {
    if (cognitiveEngine) {
      const newMetrics = cognitiveEngine.getMetrics();
      setMetrics(newMetrics);
    }
  };

  const handleStart = async () => {
    if (cognitiveEngine && !isRunning) {
      try {
        await cognitiveEngine.start();
      } catch (error) {
        console.error('Failed to start Cognitive Engine:', error);
      }
    }
  };

  const handleStop = async () => {
    if (cognitiveEngine && isRunning) {
      try {
        await cognitiveEngine.stop();
      } catch (error) {
        console.error('Failed to stop Cognitive Engine:', error);
      }
    }
  };

  const handleToggleLearning = async () => {
    if (cognitiveEngine) {
      try {
        if (learningMode) {
          // Learning mode is handled internally by the engine
          setLearningMode(false);
        } else {
          await cognitiveEngine.startLearningMode();
        }
      } catch (error) {
        console.error('Failed to toggle learning mode:', error);
      }
    }
  };

  const handleSaveAdaptation = async () => {
    if (cognitiveEngine) {
      try {
        await cognitiveEngine.saveCurrentAdaptation();
      } catch (error) {
        console.error('Failed to save adaptation:', error);
      }
    }
  };

  const handleConfigUpdate = (newConfig: Partial<CognitiveConfig>) => {
    setConfig(prev => ({ ...prev, ...newConfig }));
    // Note: Config changes would require engine restart in a real implementation
  };

  const getStatusColor = () => {
    if (!isRunning) return 'text-gray-400';
    if (learningMode) return 'text-blue-400';
    return 'text-green-400';
  };

  const getStatusText = () => {
    if (!isRunning) return 'Offline';
    if (learningMode) return 'Learning';
    return 'Active';
  };

  return (
    <div className="bg-gray-800/90 backdrop-blur-sm rounded-lg border border-gray-700/50 p-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-3">
          <div className="w-8 h-8 bg-gradient-to-r from-purple-500 to-pink-500 rounded-lg flex items-center justify-center">
            <Brain className="w-5 h-5 text-white" />
          </div>
          <div>
            <h3 className="text-lg font-semibold text-white">Cognitive Shell</h3>
            <p className={`text-sm ${getStatusColor()}`}>
              Status: {getStatusText()}
            </p>
          </div>
        </div>
        
        <div className="flex items-center space-x-2">
          <button
            onClick={() => setShowSettings(!showSettings)}
            className="p-2 text-gray-400 hover:text-white transition-colors"
          >
            <Settings className="w-4 h-4" />
          </button>
          
          {isRunning ? (
            <button
              onClick={handleStop}
              className="px-3 py-1 bg-red-500/20 text-red-400 rounded border border-red-500/30 hover:bg-red-500/30 transition-colors text-sm"
            >
              Stop
            </button>
          ) : (
            <button
              onClick={handleStart}
              className="px-3 py-1 bg-green-500/20 text-green-400 rounded border border-green-500/30 hover:bg-green-500/30 transition-colors text-sm"
            >
              Start
            </button>
          )}
        </div>
      </div>

      {/* Metrics Display */}
      {metrics && (
        <div className="grid grid-cols-2 gap-4 mb-4">
          <div className="bg-gray-700/50 rounded-lg p-3">
            <div className="flex items-center space-x-2 mb-1">
              <Activity className="w-4 h-4 text-blue-400" />
              <span className="text-sm text-gray-300">Confidence</span>
            </div>
            <div className="text-lg font-semibold text-white">
              {Math.round(metrics.confidenceLevel * 100)}%
            </div>
          </div>
          
          <div className="bg-gray-700/50 rounded-lg p-3">
            <div className="flex items-center space-x-2 mb-1">
              <Zap className="w-4 h-4 text-yellow-400" />
              <span className="text-sm text-gray-300">Adaptation</span>
            </div>
            <div className="text-lg font-semibold text-white">
              {Math.round(metrics.adaptationLevel * 100)}%
            </div>
          </div>
          
          <div className="bg-gray-700/50 rounded-lg p-3">
            <div className="flex items-center space-x-2 mb-1">
              <BarChart3 className="w-4 h-4 text-green-400" />
              <span className="text-sm text-gray-300">Active Skills</span>
            </div>
            <div className="text-lg font-semibold text-white">
              {metrics.activeSkills}
            </div>
          </div>
          
          <div className="bg-gray-700/50 rounded-lg p-3">
            <div className="flex items-center space-x-2 mb-1">
              <Brain className="w-4 h-4 text-purple-400" />
              <span className="text-sm text-gray-300">Learning Events</span>
            </div>
            <div className="text-lg font-semibold text-white">
              {metrics.learningEvents}
            </div>
          </div>
        </div>
      )}

      {/* Controls */}
      {isRunning && (
        <div className="flex flex-wrap gap-2 mb-4">
          <button
            onClick={handleToggleLearning}
            className={`px-3 py-1 rounded text-sm transition-colors ${
              learningMode
                ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                : 'bg-gray-600/20 text-gray-400 border border-gray-600/30 hover:bg-gray-600/30'
            }`}
          >
            {learningMode ? 'Stop Learning' : 'Start Learning'}
          </button>
          
          <button
            onClick={handleSaveAdaptation}
            className="px-3 py-1 bg-purple-500/20 text-purple-400 rounded border border-purple-500/30 hover:bg-purple-500/30 transition-colors text-sm"
          >
            Save Adaptation
          </button>
        </div>
      )}

      {/* Capabilities Status */}
      {isRunning && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium text-gray-300">Capabilities</h4>
          <div className="flex flex-wrap gap-2">
            <div className={`flex items-center space-x-1 px-2 py-1 rounded text-xs ${
              config.voiceEnabled ? 'bg-green-500/20 text-green-400' : 'bg-gray-600/20 text-gray-400'
            }`}>
              <Mic className="w-3 h-3" />
              <span>Voice</span>
            </div>
            
            <div className={`flex items-center space-x-1 px-2 py-1 rounded text-xs ${
              config.visualEnabled ? 'bg-green-500/20 text-green-400' : 'bg-gray-600/20 text-gray-400'
            }`}>
              <Eye className="w-3 h-3" />
              <span>Visual</span>
            </div>
            
            <div className={`flex items-center space-x-1 px-2 py-1 rounded text-xs ${
              config.loraEnabled ? 'bg-green-500/20 text-green-400' : 'bg-gray-600/20 text-gray-400'
            }`}>
              <Brain className="w-3 h-3" />
              <span>LoRA</span>
            </div>
          </div>
        </div>
      )}

      {/* Settings Panel */}
      {showSettings && (
        <div className="mt-4 pt-4 border-t border-gray-700/50">
          <h4 className="text-sm font-medium text-gray-300 mb-3">Configuration</h4>
          <div className="space-y-3">
            <div>
              <label className="block text-xs text-gray-400 mb-1">Learning Rate</label>
              <input
                type="range"
                min="0.001"
                max="0.1"
                step="0.001"
                value={config.learningRate}
                onChange={(e) => handleConfigUpdate({ learningRate: parseFloat(e.target.value) })}
                className="w-full"
              />
              <span className="text-xs text-gray-400">{config.learningRate}</span>
            </div>
            
            <div>
              <label className="block text-xs text-gray-400 mb-1">Context Size</label>
              <input
                type="range"
                min="10"
                max="500"
                step="10"
                value={config.maxContextSize}
                onChange={(e) => handleConfigUpdate({ maxContextSize: parseInt(e.target.value) })}
                className="w-full"
              />
              <span className="text-xs text-gray-400">{config.maxContextSize}</span>
            </div>
            
            <div className="flex items-center space-x-4">
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={config.voiceEnabled}
                  onChange={(e) => handleConfigUpdate({ voiceEnabled: e.target.checked })}
                  className="rounded"
                />
                <span className="text-xs text-gray-400">Voice Processing</span>
              </label>
              
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={config.visualEnabled}
                  onChange={(e) => handleConfigUpdate({ visualEnabled: e.target.checked })}
                  className="rounded"
                />
                <span className="text-xs text-gray-400">Visual Processing</span>
              </label>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
