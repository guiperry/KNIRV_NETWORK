import React from 'react';
import { useKnirvana } from '../../lib/stores/useKnirvana';
import { Card, CardContent, CardHeader, CardTitle } from './card';
import { Button } from './button';
import { Progress } from './progress';

export default function NodeInfo() {
  const {
    selectedErrorNode,
    selectedAgent,
    errorNodes,
    agents,
    nrnBalance,
    deployAgent
  } = useKnirvana();

  const selectedErrorNodeData = errorNodes.find(node => node.id === selectedErrorNode);
  const selectedAgentData = agents.find(agent => agent.id === selectedAgent);

  if (!selectedErrorNodeData && !selectedAgentData) {
    return null;
  }

  return (
    <div className="absolute top-20 left-4 w-80 space-y-4 pointer-events-auto">
      {/* ErrorNode Information */}
      {selectedErrorNodeData && (
        <Card className="bg-black bg-opacity-90 border-red-500 glow-red">
          <CardHeader>
            <CardTitle className="text-red-400 text-lg flex items-center">
              <span className="w-3 h-3 bg-red-500 rounded-full mr-2 animate-pulse" />
              ErrorNode Analysis
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Basic Info */}
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-red-300">Type:</span>
                <span className="text-red-100 font-mono">{selectedErrorNodeData.type}</span>
              </div>
              
              <div className="flex justify-between">
                <span className="text-red-300">Node ID:</span>
                <span className="text-red-100 font-mono text-xs">
                  {selectedErrorNodeData.id.slice(-8)}
                </span>
              </div>
              
              <div className="flex justify-between">
                <span className="text-red-300">Difficulty:</span>
                <span className={`font-bold ${
                  selectedErrorNodeData.difficulty > 0.7 ? 'text-red-400' :
                  selectedErrorNodeData.difficulty > 0.4 ? 'text-yellow-400' :
                  'text-green-400'
                }`}>
                  {Math.round(selectedErrorNodeData.difficulty * 100)}%
                </span>
              </div>
              
              <div className="flex justify-between">
                <span className="text-red-300">Bounty:</span>
                <span className="text-green-400 font-bold">
                  {selectedErrorNodeData.bounty} NRN
                </span>
              </div>
            </div>

            {/* Status and Progress */}
            <div className="space-y-2">
              <div className="flex justify-between items-center">
                <span className="text-red-300 text-sm">Status:</span>
                <span className={`text-sm font-semibold ${
                  selectedErrorNodeData.isBeingSolved ? 'text-yellow-400' : 'text-red-400'
                }`}>
                  {selectedErrorNodeData.isBeingSolved ? 'Being Resolved' : 'Available'}
                </span>
              </div>
              
              {selectedErrorNodeData.isBeingSolved && (
                <div className="space-y-1">
                  <div className="flex justify-between text-xs">
                    <span className="text-gray-400">Progress:</span>
                    <span className="text-yellow-400">
                      {Math.round(selectedErrorNodeData.progress * 100)}%
                    </span>
                  </div>
                  <Progress 
                    value={selectedErrorNodeData.progress * 100} 
                    className="h-2 bg-gray-700"
                  />
                  {selectedErrorNodeData.solverAgent && (
                    <div className="text-xs text-gray-400">
                      Solver: Agent {selectedErrorNodeData.solverAgent.slice(-4)}
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Action Buttons */}
            <div className="space-y-2 pt-2 border-t border-red-700">
              {selectedAgent && !selectedErrorNodeData.isBeingSolved && (
                <Button
                  onClick={() => deployAgent(selectedAgent, selectedErrorNode!)}
                  disabled={nrnBalance < 10}
                  className="w-full bg-red-600 hover:bg-red-700 text-white"
                >
                  Deploy Selected Agent
                  <span className="ml-2 text-xs opacity-80">(10 NRN)</span>
                </Button>
              )}
              
              {!selectedAgent && !selectedErrorNodeData.isBeingSolved && (
                <div className="text-center text-yellow-400 text-sm">
                  Select an agent to deploy
                </div>
              )}
              
              {selectedErrorNodeData.isBeingSolved && (
                <div className="text-center text-gray-400 text-sm">
                  Resolution in progress...
                </div>
              )}
            </div>

            {/* Estimated Solution Time */}
            {selectedErrorNodeData.isBeingSolved && selectedAgentData && (
              <div className="bg-gray-800 p-2 rounded border border-gray-600">
                <div className="text-xs text-gray-300 mb-1">Estimated Completion:</div>
                <div className="text-sm text-cyan-400">
                  {Math.max(1, Math.round((1 - selectedErrorNodeData.progress) / selectedAgentData.efficiency * 30))} seconds
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Agent Information */}
      {selectedAgentData && !selectedErrorNodeData && (
        <Card className="bg-black bg-opacity-90 border-cyan-500 glow-cyan">
          <CardHeader>
            <CardTitle className="text-cyan-400 text-lg flex items-center">
              <span className={`w-3 h-3 rounded-full mr-2 ${
                selectedAgentData.status === 'working' ? 'bg-yellow-500 animate-pulse' :
                selectedAgentData.status === 'moving' ? 'bg-blue-500' :
                'bg-gray-500'
              }`} />
              Agent Profile
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Agent Details */}
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-cyan-300">Type:</span>
                <span className="text-cyan-100 font-semibold">{selectedAgentData.type}</span>
              </div>
              
              <div className="flex justify-between">
                <span className="text-cyan-300">Agent ID:</span>
                <span className="text-cyan-100 font-mono text-xs">
                  {selectedAgentData.id.slice(-8)}
                </span>
              </div>
              
              <div className="flex justify-between">
                <span className="text-cyan-300">Status:</span>
                <span className={`font-semibold capitalize ${
                  selectedAgentData.status === 'working' ? 'text-yellow-400' :
                  selectedAgentData.status === 'moving' ? 'text-blue-400' :
                  selectedAgentData.status === 'upgrading' ? 'text-purple-400' :
                  'text-gray-400'
                }`}>
                  {selectedAgentData.status}
                </span>
              </div>
              
              <div className="flex justify-between">
                <span className="text-cyan-300">Efficiency:</span>
                <span className={`font-bold ${
                  selectedAgentData.efficiency > 0.8 ? 'text-green-400' :
                  selectedAgentData.efficiency > 0.5 ? 'text-yellow-400' :
                  'text-red-400'
                }`}>
                  {Math.round(selectedAgentData.efficiency * 100)}%
                </span>
              </div>
              
              <div className="flex justify-between">
                <span className="text-cyan-300">Experience:</span>
                <span className="text-purple-400 font-bold">
                  Level {Math.floor(selectedAgentData.experience / 25) + 1}
                </span>
              </div>
            </div>

            {/* Current Task */}
            {selectedAgentData.target && (
              <div className="bg-gray-800 p-2 rounded border border-cyan-700">
                <div className="text-xs text-cyan-300 mb-1">Current Task:</div>
                <div className="text-sm text-cyan-100">
                  Resolving {selectedAgentData.target}
                </div>
              </div>
            )}

            {/* Agent Thoughts (for working agents) */}
            {selectedAgentData.status === 'working' && (
              <div className="bg-gray-800 p-3 rounded border border-yellow-600">
                <div className="text-xs text-yellow-300 mb-2">Agent Thought Process:</div>
                <div className="space-y-1 text-xs font-mono">
                  <div className="text-green-400">→ Analyzing error patterns...</div>
                  <div className="text-green-400">→ Accessing KNIRVGRAPH database...</div>
                  <div className="text-green-400">→ Computing optimal solution path...</div>
                  <div className="text-green-400">→ Applying learned heuristics...</div>
                </div>
              </div>
            )}

            {/* Agent Actions */}
            <div className="space-y-2 pt-2 border-t border-cyan-700">
              <div className="grid grid-cols-2 gap-2">
                <Button
                  size="sm"
                  className="bg-cyan-600 hover:bg-cyan-700 text-xs"
                  disabled={nrnBalance < 25}
                >
                  Upgrade
                  <span className="block text-xs opacity-80">25 NRN</span>
                </Button>
                
                <Button
                  size="sm"
                  className="bg-purple-600 hover:bg-purple-700 text-xs"
                  disabled={nrnBalance < 15}
                >
                  Retrain
                  <span className="block text-xs opacity-80">15 NRN</span>
                </Button>
              </div>
              
              {selectedAgentData.status === 'idle' && (
                <Button
                  size="sm"
                  className="w-full bg-green-600 hover:bg-green-700 text-xs"
                >
                  Auto-Deploy to Nearest ErrorNode
                </Button>
              )}
            </div>

            {/* Performance Metrics */}
            <div className="bg-gray-800 p-2 rounded border border-gray-600">
              <div className="text-xs text-gray-300 mb-2">Performance Metrics:</div>
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div>
                  <span className="text-gray-400">Success Rate:</span>
                  <span className="text-green-400 ml-1">
                    {Math.round(selectedAgentData.efficiency * 100)}%
                  </span>
                </div>
                <div>
                  <span className="text-gray-400">Tasks Completed:</span>
                  <span className="text-blue-400 ml-1">
                    {Math.floor(selectedAgentData.experience / 10)}
                  </span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
