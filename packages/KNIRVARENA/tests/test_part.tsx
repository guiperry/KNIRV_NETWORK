import React, { useState } from 'react';
import { CognitiveConfig, CognitiveState } from '../src/sensory-shell/CognitiveEngine';

interface CognitiveShellInterfaceProps {
  onStateChange?: (state: CognitiveState) => void;
}

export const CognitiveShellInterface: React.FC<CognitiveShellInterfaceProps> = ({
  onStateChange,
}) => {
  const [isEngineRunning] = useState(false);
  const [learningMode] = useState(false);
  const [config] = useState<CognitiveConfig>({
    maxContextSize: 100,
    learningRate: 0.01,
    adaptationThreshold: 0.5,
    skillTimeout: 5000,
    voiceEnabled: false,
    visualEnabled: false,
    loraEnabled: false,
    enhancedLoraEnabled: false,
    hrmEnabled: false,
    wasmAgentsEnabled: false,
    typeScriptCompilerEnabled: false,
    adaptiveLearningEnabled: false,
    walletIntegrationEnabled: false,
    chainIntegrationEnabled: false,
    ecosystemCommunicationEnabled: false,
    errorContextEnabled: false,
  });

  React.useEffect(() => {
    // Simulate state change on component mount
    if (onStateChange) {
      onStateChange({
        status: 'initialized',
        timestamp: Date.now(),
        config,
        isRunning: isEngineRunning,
        learningMode
      } as any);
    }
  }, [onStateChange, config, isEngineRunning, learningMode]);

  return (
    <div>
      <h1>Cognitive Shell Interface</h1>
      <p>Engine Running: {isEngineRunning ? 'Yes' : 'No'}</p>
      <p>Learning Mode: {learningMode ? 'On' : 'Off'}</p>
    </div>
  );
};