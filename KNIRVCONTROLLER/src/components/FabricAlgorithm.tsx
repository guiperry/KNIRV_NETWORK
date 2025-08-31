import * as React from 'react';
import { Brain, Cpu, Network, Eye } from 'lucide-react';

interface FabricAlgorithmProps {
  status: 'idle' | 'processing' | 'listening' | 'error';
  nrvCount: number;
}

export const FabricAlgorithm: React.FC<FabricAlgorithmProps> = ({
  status,
  nrvCount
}) => {
  const getAlgorithmSteps = () => {
    return [
      { name: 'Input Ingestion', icon: Eye, active: status === 'listening' },
      { name: 'Perception & Pre-processing', icon: Brain, active: status === 'processing' },
      { name: 'Contextualization', icon: Cpu, active: status === 'processing' },
      { name: 'NRV Structuring', icon: Network, active: status === 'processing' },
    ];
  };

  if (status === 'idle' && nrvCount === 0) {
    return null;
  }

  return (
    <div className="absolute top-20 right-4 z-30 w-64">
      <div className="bg-gray-800/90 backdrop-blur-sm rounded-lg p-4 border border-gray-700/50">
        <div className="flex items-center space-x-2 mb-3">
          <div className="w-6 h-6 bg-gradient-to-r from-blue-500 to-teal-500 rounded flex items-center justify-center">
            <Brain className="w-3 h-3 text-white" />
          </div>
          <h3 className="text-sm font-medium text-white">The Fabric Algorithm</h3>
        </div>
        
        <div className="space-y-2">
          {getAlgorithmSteps().map((step, _index) => (
            <div
              key={step.name}
              className={`flex items-center space-x-2 p-2 rounded transition-all duration-300 ${
                step.active 
                  ? 'bg-blue-500/20 text-blue-400' 
                  : 'bg-gray-700/20 text-gray-400'
              }`}
            >
              <step.icon className="w-4 h-4" />
              <span className="text-xs">{step.name}</span>
              {step.active && (
                <div className="ml-auto w-2 h-2 bg-blue-400 rounded-full animate-pulse"></div>
              )}
            </div>
          ))}
        </div>
        
        <div className="mt-3 pt-3 border-t border-gray-700/50">
          <div className="flex items-center justify-between">
            <span className="text-xs text-gray-400">Active NRVs</span>
            <span className="text-sm font-medium text-white">{nrvCount}</span>
          </div>
        </div>
      </div>
    </div>
  );
};