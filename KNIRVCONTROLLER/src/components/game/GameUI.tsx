import React, { useState, useRef, useEffect } from 'react';
import { useKnirvana } from './stores/useKnirvana';
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

  const { isMuted, toggleMute } = useAudio();
  const [isEpochRunning, setIsEpochRunning] = useState(false);
  const [isAnalyzed, setIsAnalyzed] = useState(false);
  const [showHeatmap, setShowHeatmap] = useState(false);
  const [showVerifier, setShowVerifier] = useState(false);
  const [showCognitiveShell, setShowCognitiveShell] = useState(false);
  const [selectedAnchor, setSelectedAnchor] = useState(null);
  const [rewardAnchors, setRewardAnchors] = useState([]);
  const canvasRef = useRef(null);

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
    setIsAnalyzed(true);
    setShowHeatmap(true);
    // Generate heatmap overlay with red fabric
    console.log('Generating heatmap overlay...');
  };

  const handleSculpt = () => {
    if (!isAnalyzed) return;
    console.log('Sculpt mode enabled - click on grid to place reward anchors');
  };

  const handleGridClick = (event) => {
    if (!isAnalyzed) return;
    
    const rect = event.currentTarget.getBoundingClientRect();
    const x = ((event.clientX - rect.left) / rect.width) * 100;
    const y = ((event.clientY - rect.top) / rect.height) * 100;
    
    const errorNodeData = selectedErrorNode ? errorNodes.find(n => n.id === selectedErrorNode) : null;
    const metadata = errorNodeData ? {
      logs: errorNodeData.logs || ['Error: Memory allocation failed', 'Stack trace at line 42'],
      traces: errorNodeData.traces || ['Component: DataProcessor', 'Method: processBatch'],
      severity: errorNodeData.severity || 'high',
      description: errorNodeData.description || 'Memory leak detected in processing pipeline'
    } : null;
    
    const newAnchor = {
      id: Date.now(),
      x,
      y,
      weights: { w_c: 0.6, w_l: 0.3, w_s: 0.1 },
      constraints: '// Define constraints here',
      metadata
    };
    
    setRewardAnchors([...rewardAnchors, newAnchor]);
    
    // Auto-load metadata into Cognitive Shell with preloaded prompt
    setTimeout(() => {
      const preloadedPrompt = `Based on the following error node metadata, please generate a comprehensive test case for the reward anchor:\n\nError Metadata:\n${JSON.stringify(metadata, null, 2)}\n\nPlease provide:\n1. Specific test conditions\n2. Expected outcomes\n3. Edge cases to cover\n4. Performance constraints\n\nThis test case will be used to create reward shaping constraints for agent training.`;
      
      // Trigger cognitive shell with preloaded prompt
      setShowCognitiveShell(true);
    }, 100);
  };

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
      </div>

      {/* Tournament Info */}
      <div className="absolute top-4 right-4 bg-gray-900/80 backdrop-blur-sm border border-purple-500/30 rounded-lg p-4">
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

      {/* Agent Info Panel */}
      {selectedAgent && (
        <div className="absolute top-36 right-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4 max-w-xs">
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

      {/* Error Node Info Panel */}
      {selectedErrorNode && (
        <div className="absolute top-80 right-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-4 max-w-xs">
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

      {/* Heatmap Overlay */}
      {showHeatmap && (
        <div 
          className="absolute inset-0 pointer-events-none z-30"
          style={{
            background: 'radial-gradient(circle at var(--mouse-x, 50%) var(--mouse-y, 50%), rgba(255, 0, 0, 0.3) 0%, rgba(255, 0, 0, 0.1) 50%, transparent 70%)',
            mixBlendMode: 'multiply'
          }}
        />
      )}

      {/* Reward Anchors */}
      {rewardAnchors.map(anchor => (
        <div
          key={anchor.id}
          className="absolute w-4 h-4 bg-yellow-400 rounded-full cursor-pointer pointer-events-auto z-35"
          style={{
            left: `${anchor.x}%`,
            top: `${anchor.y}%`,
            transform: 'translate(-50%, -50%)',
            boxShadow: '0 0 20px rgba(255, 255, 0, 0.8)'
          }}
          onClick={() => {
            setSelectedAnchor(anchor);
            setShowVerifier(true);
          }}
          title="Reward Anchor - Click to edit"
        />
      ))}

      {/* Bottom Controls */}
      <div className="absolute bottom-4 left-4 right-4 flex justify-between items-end">
        <div className="space-x-2">
          <button
            onClick={handleAnalyze}
            className="bg-red-600 hover:bg-red-500 text-white px-4 py-2 rounded text-sm pointer-events-auto transition-colors"
          >
            Analyze
          </button>
          <button
            onClick={handleSculpt}
            disabled={!isAnalyzed}
            className={`px-4 py-2 rounded text-sm pointer-events-auto transition-colors ${
              !isAnalyzed
                ? 'bg-gray-600 text-white cursor-not-allowed'
                : 'bg-purple-600 hover:bg-purple-500 text-white'
            }`}
          >
            Sculpt
          </button>
          <div className="bg-gray-700 text-white px-4 py-2 rounded text-sm pointer-events-none">
            <span className="text-gray-300">Status: </span>
            <span className="text-yellow-400 font-medium">
              {selectedErrorNode && errorNodes.find(n => n.id === selectedErrorNode)?.isBeingSolved ? 'Solving' : 'Active'}
            </span>
          </div>
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

      {/* Invisible click surface for placing reward anchors */}
      {isAnalyzed && (
        <div 
          className="absolute inset-0 z-20 pointer-events-auto"
          onClick={handleGridClick}
          style={{ cursor: 'crosshair' }}
        />
      )}

      {/* Cognitive Shell Modal */}
      {showCognitiveShell && (
        <div className="fixed inset-0 z-50 flex items-start justify-center pt-24 bg-black/80">
          <div className="bg-gray-900 border border-cyan-500/50 rounded-lg w-full h-full max-w-6xl max-h-[80vh] m-4 flex flex-col">
            <div className="flex items-center justify-between p-4 border-b border-gray-700">
              <h2 className="text-xl font-bold text-cyan-400">Cognitive Shell - Reward Anchor Configuration</h2>
              <button
                onClick={() => setShowCognitiveShell(false)}
                className="text-gray-400 hover:text-white transition-colors"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
             <div className="flex-1 overflow-hidden">
              <ChatBrainProvider>
                <CognitiveShellInterface 
                  onOpenCortexBuilder={() => console.log('Cortex builder opened')}
                  onSkillInvoked={(skillId, result) => console.log('Skill invoked:', skillId, result)}
                  onAdaptationTriggered={(adaptation) => console.log('Adaptation triggered:', adaptation)}
                />
              </ChatBrainProvider>
            </div>
          </div>
        </div>
      )}

      {/* Verifier Overlay Modal */}
      {showVerifier && (
        <VerifierOverlay
          onClose={() => {
            setShowVerifier(false);
            setSelectedAnchor(null);
          }}
          rewardAnchors={rewardAnchors}
          setRewardAnchors={setRewardAnchors}
          initialAnchor={selectedAnchor}
        />
      )}
    </div>
  );
}
