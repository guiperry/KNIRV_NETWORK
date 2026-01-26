import React from 'react';
import { useKnirvana } from './stores/useKnirvana';
import { useAudio } from './stores/useAudio';

export default function GameUI() {
  const {
    gamePhase,
    nrnBalance,
    errorsResolved,
    skillsLearned,
    selectedErrorNode,
    selectedAgent,
    startGame,
    pauseGame,
    createAgent,
    deployAgent
  } = useKnirvana();
  const errorNodes = useKnirvana(state => state.errorNodes);
  const agents = useKnirvana(state => state.agents);

  const { isMuted, toggleMute } = useAudio();

  if (gamePhase === "menu") {
    return (
      <div className="fixed inset-0 flex items-center justify-center z-50 bg-black/80">
        <div className="bg-gray-900/90 border border-cyan-500/50 rounded-lg p-8 max-w-md">
          <h1 className="text-3xl font-bold text-cyan-400 mb-4 text-center">KNIRVANA</h1>
          <p className="text-gray-300 mb-6 text-center">
            Transform AI errors into collective knowledge
          </p>
          <div className="text-center">
            <button
              onClick={startGame}
              className="bg-cyan-600 hover:bg-cyan-500 text-white px-8 py-3 rounded-lg font-medium transition-colors"
            >
              Start Game
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 pointer-events-none z-40">
      {/* Top HUD */}
      <div className="absolute top-4 left-4 right-4 flex justify-between items-start">
        <div className="bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4">
          <div className="text-cyan-400 font-bold text-lg">KNIRVANA</div>
          <div className="text-gray-300 text-sm">
            Phase: <span className="text-cyan-300">{gamePhase}</span>
          </div>
        </div>
        
        <div className="bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4">
          <div className="text-yellow-400 font-bold text-lg">{nrnBalance} NRN</div>
          <div className="text-gray-300 text-sm">Balance</div>
        </div>
      </div>

      {/* Stats Panel */}
      <div className="absolute top-24 left-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4">
        <h3 className="text-cyan-400 font-semibold mb-2">Statistics</h3>
        <div className="space-y-1 text-sm">
          <div className="text-gray-300">
            Errors Resolved: <span className="text-green-400">{errorsResolved}</span>
          </div>
          <div className="text-gray-300">
            Skills Learned: <span className="text-cyan-400">{skillsLearned}</span>
          </div>
        </div>
      </div>

      {/* Selection Info */}
      {(selectedErrorNode || selectedAgent) && (
        <div className="absolute top-24 right-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4 max-w-xs">
          <h3 className="text-cyan-400 font-semibold mb-2">
            {selectedErrorNode ? 'Error Node' : 'AI Agent'}
          </h3>
          
          {selectedErrorNode && (
            <div className="text-sm space-y-1">
              <div className="text-gray-300">ID: {selectedErrorNode}</div>
              <div className="text-gray-300">
                Status: <span className="text-orange-400">
                  {errorNodes.find(n => n.id === selectedErrorNode)?.isBeingSolved ? 'In Progress' : 'Active'}
                </span>
              </div>
              {selectedAgent && (
                <button
                  onClick={() => deployAgent(selectedAgent, selectedErrorNode)}
                  className="mt-2 bg-cyan-600 hover:bg-cyan-500 text-white px-3 py-1 rounded text-sm pointer-events-auto transition-colors"
                >
                  Deploy Agent (10 NRN)
                </button>
              )}
            </div>
          )}
          
          {selectedAgent && (
            <div className="text-sm space-y-1">
              <div className="text-gray-300">ID: {selectedAgent}</div>
              <div className="text-gray-300">
                Type: {agents.find(a => a.id === selectedAgent)?.type}
              </div>
              <div className="text-gray-300">
                Status: <span className="text-green-400">
                  {agents.find(a => a.id === selectedAgent)?.status}
                </span>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Bottom Controls */}
      <div className="absolute bottom-4 left-4 right-4 flex justify-between items-end">
        <div className="space-x-2">
          <button
            onClick={() => createAgent('Analyzer')}
            className="bg-blue-600 hover:bg-blue-500 text-white px-4 py-2 rounded text-sm pointer-events-auto transition-colors"
          >
            Create Analyzer (50 NRN)
          </button>
          <button
            onClick={() => createAgent('Optimizer')}
            className="bg-purple-600 hover:bg-purple-500 text-white px-4 py-2 rounded text-sm pointer-events-auto transition-colors"
          >
            Create Optimizer (50 NRN)
          </button>
        </div>
        
        <div className="space-x-2">
          {gamePhase === "playing" && (
            <button
              onClick={pauseGame}
              className="bg-yellow-600 hover:bg-yellow-500 text-white px-4 py-2 rounded text-sm pointer-events-auto transition-colors"
            >
              Pause
            </button>
          )}
          {gamePhase === "paused" && (
            <button
              onClick={startGame}
              className="bg-green-600 hover:bg-green-500 text-white px-4 py-2 rounded text-sm pointer-events-auto transition-colors"
            >
              Resume
            </button>
          )}
          <button
            onClick={toggleMute}
            className="bg-gray-600 hover:bg-gray-500 text-white px-4 py-2 rounded text-sm pointer-events-auto transition-colors"
          >
            {isMuted ? '🔇' : '🔊'} Sound
          </button>
        </div>
      </div>
    </div>
  );
}