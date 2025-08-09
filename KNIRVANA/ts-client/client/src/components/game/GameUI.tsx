import React from 'react';
import { useKnirvana } from '../../lib/stores/useKnirvana';
import GameHUD from '../ui/GameHUD';
import NodeInfo from '../ui/NodeInfo';
import AgentPanel from '../ui/AgentPanel';
import ResourceDisplay from '../ui/ResourceDisplay';
import CompetitiveOverlay from './CompetitiveOverlay';
import PretrainingInterface from './PretrainingInterface';
import { Button } from '../ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';

export default function GameUI() {
  const { 
    gamePhase, 
    startGame,
    selectedErrorNode,
    selectedAgent,
    errorNodes,
    agents
  } = useKnirvana();

  // Main menu screen
  if (gamePhase === 'menu') {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-90 flex items-center justify-center z-50">
        <Card className="w-[500px] bg-gray-900 border-cyan-500 glow-cyan">
          <CardHeader className="text-center">
            <CardTitle className="text-4xl text-cyan-400 font-bold mb-2">
              KNIRVANA
            </CardTitle>
            <p className="text-cyan-300 text-lg">
              The Experiential Gateway
            </p>
            <p className="text-cyan-200 text-sm mt-2">
              Enter the KNIRV D-TEN Network
            </p>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="text-cyan-200 text-center text-sm space-y-3 bg-gray-800 p-4 rounded border border-cyan-700">
              <p className="text-cyan-300 font-semibold">Mission Briefing:</p>
              <p>• Command AI agents to resolve ErrorNodes in the KNIRVGRAPH</p>
              <p>• Compete with other players in real-time strategy</p>
              <p>• Earn NRN tokens through collective intelligence</p>
              <p>• Transform failures into Skills for the D-TEN network</p>
            </div>
            
            <div className="text-cyan-400 text-xs text-center space-y-2 bg-gray-800 p-3 rounded border border-gray-600">
              <p className="text-cyan-300 font-semibold">Controls:</p>
              <div className="grid grid-cols-2 gap-2 text-left">
                <div>
                  <p><kbd className="bg-gray-700 px-1 rounded">WASD</kbd> - Move Camera</p>
                  <p><kbd className="bg-gray-700 px-1 rounded">Q/E</kbd> - Rotate View</p>
                </div>
                <div>
                  <p><kbd className="bg-gray-700 px-1 rounded">+/-</kbd> - Zoom</p>
                  <p><kbd className="bg-gray-700 px-1 rounded">Space</kbd> - Select</p>
                </div>
              </div>
            </div>
            
            <Button 
              onClick={startGame}
              className="w-full bg-cyan-600 hover:bg-cyan-700 text-white text-lg py-3 glow-cyan"
            >
              Initialize D-TEN Connection
            </Button>
            
            <div className="text-xs text-gray-400 text-center">
              KNIRV Decentralized Trusted Execution Network v2.0
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  // Pause screen
  if (gamePhase === 'paused') {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-80 flex items-center justify-center z-50">
        <Card className="w-80 bg-gray-900 border-yellow-500">
          <CardHeader>
            <CardTitle className="text-yellow-400 text-center">
              Network Paused
            </CardTitle>
          </CardHeader>
          <CardContent className="text-center space-y-4">
            <p className="text-yellow-300">
              D-TEN operations suspended
            </p>
            <Button 
              onClick={startGame}
              className="w-full bg-yellow-600 hover:bg-yellow-700 text-white"
            >
              Resume Operations
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  // Main game UI
  return (
    <div className="fixed inset-0 pointer-events-none z-10">
      {/* Main HUD */}
      <GameHUD />
      
      {/* Top Resource Bar */}
      <div className="absolute top-4 left-4 right-4 flex justify-between pointer-events-auto">
        <ResourceDisplay />
        
        <div className="text-cyan-400 font-mono text-sm bg-black bg-opacity-60 px-3 py-2 rounded border border-cyan-500">
          <span className="text-green-400">●</span> D-TEN Status: ACTIVE
        </div>
      </div>

      {/* Node Information Panel */}
      {(selectedErrorNode || selectedAgent) && (
        <NodeInfo />
      )}

      {/* Agent Management Panel */}
      <AgentPanel />
      
      {/* Competitive and Training Features */}
      <CompetitiveOverlay />
      <PretrainingInterface />

      {/* Network Statistics Footer */}
      <div className="absolute bottom-4 left-4 right-4 pointer-events-auto">
        <Card className="bg-black bg-opacity-80 border-cyan-500">
          <CardContent className="py-2">
            <div className="flex justify-between items-center text-xs">
              <div className="text-cyan-300 space-x-6 flex">
                <span>
                  <span className="text-red-400">●</span> ErrorNodes: {errorNodes.filter(n => !n.isBeingSolved).length}
                </span>
                <span>
                  <span className="text-yellow-400">●</span> Active Agents: {agents.filter(a => a.status === 'working').length}
                </span>
                <span>
                  <span className="text-green-400">●</span> Solutions: {errorNodes.filter(n => n.progress >= 1).length}
                </span>
              </div>
              
              <div className="text-cyan-400 space-x-4 flex">
                <span>Network Intelligence: <span className="text-green-400">Growing</span></span>
                <span>Collective Knowledge: <span className="text-blue-400">Expanding</span></span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
