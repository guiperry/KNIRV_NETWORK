import React from 'react';
import { Card, CardContent } from './card';

interface AgentProfileCardProps {
  agentId: string;
  type: string;
  status: 'idle' | 'moving' | 'working' | 'upgrading';
  efficiency: number;
  experience: number;
  isSelected?: boolean;
  onClick?: () => void;
}

export default function AgentProfileCard({
  agentId,
  type,
  status,
  efficiency,
  experience,
  isSelected = false,
  onClick
}: AgentProfileCardProps) {
  const getTypeColor = (agentType: string) => {
    switch (agentType) {
      case 'Analyzer': return 'border-red-500 bg-red-900';
      case 'Optimizer': return 'border-teal-500 bg-teal-900';
      case 'Synthesizer': return 'border-blue-500 bg-blue-900';
      case 'Debugger': return 'border-yellow-500 bg-yellow-900';
      default: return 'border-gray-500 bg-gray-900';
    }
  };

  const getStatusIcon = (agentStatus: string) => {
    switch (agentStatus) {
      case 'working': return '⚡';
      case 'moving': return '🚀';
      case 'upgrading': return '🔧';
      default: return '💤';
    }
  };

  const getExperienceLevel = () => {
    return Math.floor(experience / 25) + 1;
  };

  return (
    <Card 
      className={`w-48 cursor-pointer transition-all duration-200 bg-opacity-90 ${
        isSelected 
          ? `${getTypeColor(type)} ring-2 ring-cyan-400` 
          : `${getTypeColor(type)} bg-opacity-60 hover:bg-opacity-80`
      }`}
      onClick={onClick}
    >
      <CardContent className="p-3">
        <div className="flex items-center space-x-3">
          {/* Agent Avatar */}
          <div className="relative">
            <div className="w-12 h-12 rounded-lg bg-gray-800 border border-cyan-500 flex items-center justify-center overflow-hidden">
              <img 
                src="/textures/agent_avatar.png" 
                alt="Agent Avatar"
                className="w-10 h-10 object-cover"
                style={{
                  filter: `hue-rotate(${
                    type === 'Analyzer' ? '0deg' :
                    type === 'Optimizer' ? '180deg' :
                    type === 'Synthesizer' ? '240deg' :
                    type === 'Debugger' ? '60deg' : '0deg'
                  })`
                }}
              />
            </div>
            
            {/* Status indicator */}
            <div className="absolute -top-1 -right-1 w-4 h-4 rounded-full bg-black border border-gray-600 flex items-center justify-center text-xs">
              {getStatusIcon(status)}
            </div>
          </div>

          {/* Agent Info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between mb-1">
              <h4 className="text-sm font-semibold text-white truncate">
                {type} Agent
              </h4>
              <span className="text-xs text-gray-400">
                L{getExperienceLevel()}
              </span>
            </div>
            
            <div className="text-xs text-gray-400 mb-1">
              ID: {agentId.slice(-6)}
            </div>
            
            <div className="flex items-center justify-between text-xs">
              <span className="capitalize text-gray-300">
                {status}
              </span>
              <span className={`font-semibold ${
                efficiency > 0.8 ? 'text-green-400' :
                efficiency > 0.5 ? 'text-yellow-400' :
                'text-red-400'
              }`}>
                {Math.round(efficiency * 100)}%
              </span>
            </div>

            {/* Efficiency bar */}
            <div className="mt-2 w-full bg-gray-700 rounded-full h-1">
              <div 
                className={`h-1 rounded-full transition-all duration-300 ${
                  efficiency > 0.8 ? 'bg-green-400' :
                  efficiency > 0.5 ? 'bg-yellow-400' :
                  'bg-red-400'
                }`}
                style={{ width: `${efficiency * 100}%` }}
              />
            </div>
          </div>
        </div>

        {/* Experience points */}
        <div className="mt-2 flex justify-center space-x-1">
          {Array.from({ length: 5 }, (_, i) => (
            <div
              key={i}
              className={`w-2 h-1 rounded-full ${
                i < getExperienceLevel() ? 'bg-purple-400' : 'bg-gray-600'
              }`}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}