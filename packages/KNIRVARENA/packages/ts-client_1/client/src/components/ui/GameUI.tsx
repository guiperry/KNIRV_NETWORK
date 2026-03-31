import { useKnirvana } from "../../lib/stores/useKnirvana";
import AgentPanel from "./AgentPanel";
import ResourceDisplay from "./ResourceDisplay";
import { Button } from "./button";
import { Card, CardContent, CardHeader, CardTitle } from "./card";
import EcosystemGameSlideout from "./EcosystemGameSlideout";
import { useState } from "react";

export default function GameUI() {
  const {
    selectedErrorNode,
    selectedAgent,
    errorNodes,
    agents,
    nrnBalance,
    deployAgent,
    gamePhase,
    startGame
  } = useKnirvana();

  const [isEcosystemGameOpen, setIsEcosystemGameOpen] = useState(false);

  const selectedErrorNodeData = errorNodes.find(node => node.id === selectedErrorNode);
  const selectedAgentData = agents.find(agent => agent.id === selectedAgent);

  if (gamePhase === 'menu') {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-80 flex items-center justify-center z-50">
        <Card className="w-96 bg-gray-900 border-cyan-500">
          <CardHeader>
            <CardTitle className="text-cyan-400 text-center text-2xl">
              KNIRVANA
            </CardTitle>
            <p className="text-cyan-300 text-center text-sm">
              The Experiential Gateway to the KNIRV D-TEN
            </p>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="text-cyan-200 text-center text-xs space-y-2">
              <p>Command AI agents to resolve ErrorNodes</p>
              <p>Compete in the living KNIRVGRAPH</p>
              <p>Earn NRN tokens through collective intelligence</p>
            </div>
            <Button 
              onClick={startGame}
              className="w-full bg-cyan-600 hover:bg-cyan-700 text-white"
            >
              Enter the D-TEN
            </Button>
            <div className="text-cyan-400 text-xs text-center space-y-1">
              <p><strong>Controls:</strong></p>
              <p>🖱️ Click & Drag - Navigate | Scroll - Zoom</p>
              <p>WASD - Move | Q/E - Rotate | +/- - Zoom</p>
              <p>Double-Click or R - Reset View | Space - Select</p>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 pointer-events-none z-10">
      {/* Top HUD */}
      <div className="absolute top-4 left-4 right-4 flex justify-between pointer-events-auto">
        <ResourceDisplay />

        <div className="flex items-center gap-4">
          <Button
            onClick={() => setIsEcosystemGameOpen(true)}
            className="bg-purple-600 hover:bg-purple-700 text-white px-4 py-2 text-sm"
          >
            🎮 Ecosystem Menu
          </Button>

          <div className="text-cyan-400 font-mono text-sm bg-black bg-opacity-60 px-3 py-2 rounded border border-cyan-500">
            KNIRV D-TEN Network Status: <span className="text-green-400">ACTIVE</span>
          </div>
        </div>
      </div>

      {/* Left Panel - Selected ErrorNode Info */}
      {selectedErrorNodeData && (
        <div className="absolute top-20 left-4 w-80 pointer-events-auto">
          <Card className="bg-black bg-opacity-80 border-red-500">
            <CardHeader>
              <CardTitle className="text-red-400 text-lg">
                ErrorNode: {selectedErrorNodeData.type}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div className="text-red-300">
                <p><strong>Difficulty:</strong> {Math.round(selectedErrorNodeData.difficulty * 100)}%</p>
                <p><strong>Bounty:</strong> {selectedErrorNodeData.bounty} NRN</p>
                <p><strong>Status:</strong> {selectedErrorNodeData.isBeingSolved ? 'Being Solved' : 'Available'}</p>
                {selectedErrorNodeData.isBeingSolved && (
                  <p><strong>Progress:</strong> {Math.round(selectedErrorNodeData.progress * 100)}%</p>
                )}
              </div>
              
              {selectedAgent && !selectedErrorNodeData.isBeingSolved && (
                <Button
                  onClick={() => deployAgent(selectedAgent, selectedErrorNode!)}
                  className="w-full bg-red-600 hover:bg-red-700 text-white mt-2"
                  disabled={nrnBalance < 10}
                >
                  Deploy Agent (10 NRN)
                </Button>
              )}
            </CardContent>
          </Card>
        </div>
      )}

      {/* Right Panel - Agent Management */}
      <AgentPanel />

      {/* Bottom Info Panel */}
      <div className="absolute bottom-4 left-4 right-4 pointer-events-auto">
        <Card className="bg-black bg-opacity-80 border-cyan-500">
          <CardContent className="py-3">
            <div className="flex justify-between items-center text-sm">
              <div className="text-cyan-300 space-x-6">
                <span>ErrorNodes: {errorNodes.filter(n => !n.isBeingSolved).length} available</span>
                <span>Active Agents: {agents.filter(a => a.status === 'working').length}</span>
                <span>Solutions Found: {errorNodes.filter(n => n.progress >= 1).length}</span>
              </div>
              <div className="text-cyan-400">
                Network Intelligence Level: <span className="text-green-400">Growing</span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Selected Agent Info Overlay */}
      {selectedAgentData && (
        <div className="absolute top-1/2 right-4 transform -translate-y-1/2 w-72 pointer-events-auto">
          <Card className="bg-black bg-opacity-90 border-cyan-500">
            <CardHeader>
              <CardTitle className="text-cyan-400">
                {selectedAgentData.type} Agent
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <div className="text-cyan-300">
                <p><strong>Status:</strong> {selectedAgentData.status}</p>
                <p><strong>Efficiency:</strong> {Math.round(selectedAgentData.efficiency * 100)}%</p>
                <p><strong>Target:</strong> {selectedAgentData.target || 'None'}</p>
              </div>
              
              {selectedAgentData.status === 'working' && (
                <div className="bg-gray-800 p-2 rounded border border-cyan-600">
                  <p className="text-cyan-200 text-xs mb-1">Agent Thought Process:</p>
                  <div className="text-green-400 text-xs font-mono">
                    <p>→ Analyzing error patterns...</p>
                    <p>→ Searching skill registry...</p>
                    <p>→ Optimizing solution path...</p>
                    <p>→ Applying learned heuristics...</p>
                  </div>
                </div>
              )}
              
              <div className="flex space-x-2">
                <Button size="sm" className="flex-1 bg-cyan-600 hover:bg-cyan-700">
                  Upgrade
                </Button>
                <Button size="sm" className="flex-1 bg-yellow-600 hover:bg-yellow-700">
                  Retrain
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Ecosystem Game Slideout */}
      <EcosystemGameSlideout
        isOpen={isEcosystemGameOpen}
        onClose={() => setIsEcosystemGameOpen(false)}
      />
    </div>
  );
}
