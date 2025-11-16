import React, { useState } from 'react';
import { CognitiveConfig, CognitiveState } from '../sensory-shell/CognitiveEngine';

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
      } as CognitiveState);
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