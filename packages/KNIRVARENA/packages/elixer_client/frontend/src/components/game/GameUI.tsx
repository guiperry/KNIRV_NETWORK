import React, { useState, useEffect } from 'react';
import { useKnirvana, RewardAnchor } from './stores/useKnirvana';
import { useAudio } from './stores/useAudio';
import { SabotageType } from '../../engine/Sabotage';
import VerifierOverlay from './VerifierOverlay';
import { CognitiveShellInterface } from '../CognitiveShellInterface';
import { ChatBrainProvider } from '../../contexts/ChatBrainContext';
import { initializeTournamentController } from '../../engine/TournamentControllerIntegration';

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
    deployAgent,
    runEpoch,
    applySabotage,
    startTraining,
    distillTrajectory,
    hardenAgent,
    skillSlotOwner,
    incumbentScore,
    agents
  } = useKnirvana();
  const errorNodes = useKnirvana(state => state.errorNodes);
  const isAnalyzing = useKnirvana(state => state.isAnalyzing);
  const setAnalyzing = useKnirvana(state => state.setAnalyzing);
  const isSculpting = useKnirvana(state => state.isSculpting);
  const setSculpting = useKnirvana(state => state.setSculpting);
  const rewardAnchors = useKnirvana(state => state.rewardAnchors);
  const selectedRewardAnchor = useKnirvana(state => state.selectedRewardAnchor);
  const updateRewardAnchor = useKnirvana(state => state.updateRewardAnchor);

  const { isMuted, toggleMute } = useAudio();
  const [isEpochRunning, setIsEpochRunning] = useState(false);
  const [showVerifier, setShowVerifier] = useState(false);
  const [showCognitiveShell, setShowCognitiveShell] = useState(false);
  const [preloadedPrompt, setPreloadedPrompt] = useState<string>('');
  const [sabotageType, setSabotageType] = useState<SabotageType>(SabotageType.NOISE_INJECTION);
  const [sabotageMagnitude, setSabotageMagnitude] = useState<number>(1);


  // Initialize Tournament Controller on component mount
  useEffect(() => {
    initializeTournamentController();
  }, []);

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

  const handleRunEpoch = async () => {
    setIsEpochRunning(true);
    await runEpoch();
    setIsEpochRunning(false);
  };

  const handleAnalyze = () => {
    // Toggle analyze mode - highlights error nodes in the 3D scene with red rings
    const newAnalyzing = !isAnalyzing;
    setAnalyzing(newAnalyzing);
    console.log(`Analyze mode: ${newAnalyzing ? 'ON' : 'OFF'}`);
  };

  const handleSculpt = () => {
    if (!isAnalyzing) return;
    // Toggle sculpt mode
    const newSculpting = !isSculpting;
    setSculpting(newSculpting);
    console.log(`Sculpt mode: ${newSculpting ? 'ON - click on arena floor to place reward anchors' : 'OFF'}`);
  };

  const handleCreateAgent = () => {
    const agentType = ['Analyzer', 'Optimizer', 'Solver'][Math.floor(Math.random() * 3)];
    createAgent(agentType);
  };

  const handleApplySabotage = () => {
    if (!selectedAgent) {
      alert('Please select an agent first');
      return;
    }
    applySabotage(selectedAgent, sabotageType, sabotageMagnitude);
  };

  const handleStartTraining = () => {
    if (!selectedAgent) {
      alert('Please select an agent first');
      return;
    }
    startTraining(selectedAgent);
  };

  const handleDistillTrajectory = () => {
    if (!selectedAgent) {
      alert('Please select an agent first');
      return;
    }
    distillTrajectory(selectedAgent);
  };

  const handleHardenAgent = () => {
    if (!selectedAgent) {
      alert('Please select an agent first');
      return;
    }
    hardenAgent(selectedAgent);
  };

  const handlePauseGame = () => {
    pauseGame();
    alert('Game paused. Training and epoch progression halted.');
  };


  return (
    <div className="fixed inset-0 pointer-events-none z-40">
      {/* Top Bar Grid Layout */}
      <div className="absolute top-4 left-4 right-4 flex gap-4 z-50">
        {/* KNIRVANA */}
        <div className="bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4 flex-shrink-0">
          <div className="text-cyan-400 font-bold text-lg">KNIRVANA</div>
          <div className="text-gray-300 text-sm">
            Phase: <span className="text-cyan-300">{gamePhase}</span>
          </div>
        </div>

        {/* Agent Info Panel */}
        {selectedAgent && (
          <div className="bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4 max-w-xs flex-shrink-0">
            <h3 className="text-cyan-400 font-semibold mb-2">AI Agent</h3>
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
              <div className="text-gray-300">
                Policy: <span className="text-purple-400">
                  {agents.find(a => a.id === selectedAgent)?.policy}
                </span>
              </div>
              <div className="text-gray-300">
                Compute: <span className="text-blue-400">
                  {agents.find(a => a.id === selectedAgent)?.resources.compute.toFixed(0)}
                </span>
              </div>
              <div className="text-gray-300">
                Parity: <span className="text-green-400">
                  {agents.find(a => a.id === selectedAgent)?.resources.parity.toFixed(0)}
                </span>
              </div>
              <div className="text-gray-300">
                Generation: <span className="text-yellow-400">
                  {agents.find(a => a.id === selectedAgent)?.resources.generation}
                </span>
              </div>
            </div>
          </div>
        )}

        {/* Tournament Info */}
        <div className="bg-gray-900/80 backdrop-blur-sm border border-purple-500/30 rounded-lg p-4 flex-shrink-0 ml-auto">
          <h3 className="text-purple-400 font-semibold mb-2">Tournament</h3>
          <div className="space-y-1 text-sm">
            <div className="text-gray-300">
              Skill Slot Owner: <span className="text-purple-300">{skillSlotOwner || 'None'}</span>
            </div>
            <div className="text-gray-300">
              Incumbent Score: <span className="text-yellow-400">{(incumbentScore * 100).toFixed(0)}%</span>
            </div>
            <div className="mt-2">
              <div className="text-xs text-gray-400 mb-1">Red Queen Meter</div>
              <div className="w-full bg-gray-700 rounded-full h-2">
                <div 
                  className="bg-purple-500 h-2 rounded-full transition-all"
                  style={{ width: `${Math.random() * 100}%` }}
                />
              </div>
            </div>

            <button
              onClick={handleRunEpoch}
              disabled={isEpochRunning}
              className={`mt-2 w-full px-3 py-1 rounded text-sm pointer-events-auto transition-colors ${
                isEpochRunning
                  ? 'bg-yellow-600 text-white cursor-not-allowed'
                  : 'bg-green-600 hover:bg-green-500 text-white'
              }`}
            >
              {isEpochRunning ? 'Epoch Running' : 'Run Epoch'}
            </button>
          </div>
        </div>
      </div>

      {/* Stats Panel */}
      <div className="absolute top-24 left-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4 z-50">
        <h3 className="text-cyan-400 font-semibold mb-2">Statistics</h3>
        <div className="space-y-1 text-sm">
          <div className="text-gray-300">
            NRN Balance: <span className="text-yellow-400">{nrnBalance}</span>
          </div>
          <div className="text-gray-300">
            Errors Resolved: <span className="text-green-400">{errorsResolved}</span>
          </div>
          <div className="text-gray-300">
            Skills Learned: <span className="text-cyan-400">{skillsLearned}</span>
          </div>
        </div>
      </div>

      {/* Agent Creation & Training Panel */}
      <div className="absolute top-64 left-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4 z-50 max-w-xs">
        <h3 className="text-cyan-400 font-semibold mb-2">Agent Management</h3>
        <div className="space-y-2 text-sm">
          <button
            onClick={handleCreateAgent}
            className="w-full bg-blue-600 hover:bg-blue-500 text-white px-3 py-2 rounded text-sm pointer-events-auto transition-colors"
          >
            Create New Agent (50 NRN)
          </button>
          
          {selectedAgent && (
            <>
              <div className="pt-2 border-t border-gray-700">
                <h4 className="text-cyan-300 font-medium mb-1">Training</h4>
                <div className="grid grid-cols-2 gap-1">
                  <button
                    onClick={handleStartTraining}
                    className="bg-purple-600 hover:bg-purple-500 text-white px-2 py-1 rounded text-xs pointer-events-auto transition-colors"
                  >
                    Start Training
                  </button>
                  <button
                    onClick={handleDistillTrajectory}
                    className="bg-indigo-600 hover:bg-indigo-500 text-white px-2 py-1 rounded text-xs pointer-events-auto transition-colors"
                  >
                    Distill Trajectory
                  </button>
                  <button
                    onClick={handleHardenAgent}
                    className="bg-red-600 hover:bg-red-500 text-white px-2 py-1 rounded text-xs pointer-events-auto transition-colors col-span-2"
                  >
                    Harden Agent
                  </button>
                </div>
              </div>
              
              <div className="pt-2 border-t border-gray-700">
                <h4 className="text-red-300 font-medium mb-1">Sabotage</h4>
                <div className="space-y-1">
                  <select
                    value={sabotageType}
                    onChange={(e) => setSabotageType(e.target.value as SabotageType)}
                    className="w-full bg-gray-800 text-white text-xs p-1 rounded pointer-events-auto"
                  >
                    <option value={SabotageType.NOISE_INJECTION}>Noise Injection</option>
                    <option value={SabotageType.BACKPROP_PULSE}>Backprop Pulse</option>
                    <option value={SabotageType.GRADIENT_GHOSTING}>Gradient Ghosting</option>
                  </select>
                  <div className="flex items-center space-x-2">
                    <span className="text-gray-300 text-xs">Magnitude:</span>
                    <input
                      type="range"
                      min="0.1"
                      max="5"
                      step="0.1"
                      value={sabotageMagnitude}
                      onChange={(e) => setSabotageMagnitude(parseFloat(e.target.value))}
                      className="flex-1 pointer-events-auto"
                    />
                    <span className="text-gray-300 text-xs w-8">{sabotageMagnitude.toFixed(1)}</span>
                  </div>
                  <button
                    onClick={handleApplySabotage}
                    className="w-full bg-red-700 hover:bg-red-600 text-white px-3 py-1 rounded text-sm pointer-events-auto transition-colors"
                  >
                    Apply Sabotage
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Error Node Info Panel */}
      {selectedErrorNode && (
        <div className="absolute top-80 right-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4 max-w-xs z-50">
          <h3 className="text-cyan-400 font-semibold mb-2">Error Node</h3>
          <div className="text-sm space-y-1">
            <div className="text-gray-300">ID: {selectedErrorNode}</div>
            <div className="text-gray-300">
              Type: {errorNodes.find(n => n.id === selectedErrorNode)?.type}
            </div>
            <div className="text-gray-300">
              Status: <span className="text-orange-400">
                {errorNodes.find(n => n.id === selectedErrorNode)?.isBeingSolved ? 'In Progress' : 'Active'}
              </span>
            </div>
            <div className="text-gray-300">
              Severity: <span className="text-red-400">High</span>
            </div>
            <div className="text-gray-300">
              Description: Memory leak detected in processing pipeline
            </div>
            <div className="text-gray-300">
              NRN Bounty: <span className="text-yellow-400">{errorNodes.find(n => n.id === selectedErrorNode)?.bounty} NRN</span>
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
        </div>
      )}

      {/* Sculpt mode indicator */}
      {isSculpting && (
        <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 pointer-events-none">
          <div className="bg-yellow-500/20 border-2 border-yellow-500 rounded-lg px-6 py-3 text-yellow-300 text-center">
            Click on the arena floor to place a reward anchor
          </div>
        </div>
      )}

      {/* Bottom Controls */}
      <div className="absolute bottom-4 left-1/2 transform -translate-x-1/2 flex gap-4 z-50">
        <button
          onClick={handleAnalyze}
          className={`px-4 py-2 rounded-lg pointer-events-auto transition-colors ${
            isAnalyzing
              ? 'bg-red-500 ring-2 ring-red-300 text-white'
              : 'bg-red-600 hover:bg-red-500 text-white'
          }`}
        >
          {isAnalyzing ? 'Analyzing...' : 'Analyze'}
        </button>
        <button
          onClick={handleSculpt}
          disabled={!isAnalyzing}
          className={`px-4 py-2 rounded-lg pointer-events-auto transition-colors ${
            isSculpting
              ? 'bg-yellow-500 ring-2 ring-yellow-300 text-white'
              : isAnalyzing
                ? 'bg-yellow-600 hover:bg-yellow-500 text-white'
                : 'bg-gray-600 text-gray-400 cursor-not-allowed'
          }`}
        >
          {isSculpting ? 'Sculpting...' : 'Sculpt'}
        </button>
        <button
          onClick={() => setShowVerifier(true)}
          disabled={!selectedRewardAnchor}
          className={`px-4 py-2 rounded-lg pointer-events-auto transition-colors ${
            selectedRewardAnchor
              ? 'bg-green-600 hover:bg-green-500 text-white'
              : 'bg-gray-600 text-gray-400 cursor-not-allowed'
          }`}
        >
          Verify{selectedRewardAnchor ? '' : ' (Select Anchor)'}
        </button>
        <button
          onClick={() => setShowCognitiveShell(true)}
          className="bg-cyan-600 hover:bg-cyan-500 text-white px-4 py-2 rounded-lg pointer-events-auto transition-colors"
        >
          Cognitive Shell
        </button>
        <button
          onClick={handlePauseGame}
          className="bg-red-600 hover:bg-red-500 text-white px-4 py-2 rounded-lg pointer-events-auto transition-colors"
        >
          Pause Game
        </button>
        <button
          onClick={toggleMute}
          className="bg-gray-700 hover:bg-gray-600 text-white px-4 py-2 rounded-lg pointer-events-auto transition-colors"
        >
          {isMuted ? '🔇' : '🔊'}
        </button>
      </div>

      {/* Cognitive Shell Modal */}
      {showCognitiveShell && (
        <ChatBrainProvider>
          <CognitiveShellInterface
            preloadedPrompt={preloadedPrompt}
          />
        </ChatBrainProvider>
      )}

      {/* Verifier Overlay */}
      {showVerifier && selectedRewardAnchor && (
        <VerifierOverlay
          onClose={() => setShowVerifier(false)}
          rewardAnchors={rewardAnchors}
          updateRewardAnchor={updateRewardAnchor}
          initialAnchor={rewardAnchors.find(a => a.id === selectedRewardAnchor) || null}
        />
      )}
    </div>
  );
}
