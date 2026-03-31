import React, { useState, useEffect } from 'react';
import { useKnirvana, RewardAnchor } from './stores/useKnirvana';
import { useAudio } from './stores/useAudio';
import { SabotageType } from '../../engine/Sabotage';
import VerifierOverlay from './VerifierOverlay';
import EpochResultsPanel from './EpochResultsPanel';
import { CognitiveShellInterface } from '../CognitiveShellInterface';
import { ChatBrainProvider } from '../../contexts/ChatBrainContext';
import { initializeTournamentController } from '../../engine/TournamentControllerIntegration';

export default function GameUI() {
  const {
    gamePhase,
    nrnBalance,
    errorsResolved,
    skillsLearned,
    epochNumber,
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
    agents,
    usingMockLLM,
    lastEpochResult,
    saveProgress,
  } = useKnirvana();

  const errorNodes = useKnirvana(s => s.errorNodes);
  const isAnalyzing = useKnirvana(s => s.isAnalyzing);
  const setAnalyzing = useKnirvana(s => s.setAnalyzing);
  const isSculpting = useKnirvana(s => s.isSculpting);
  const setSculpting = useKnirvana(s => s.setSculpting);
  const rewardAnchors = useKnirvana(s => s.rewardAnchors);
  const selectedRewardAnchor = useKnirvana(s => s.selectedRewardAnchor);
  const updateRewardAnchor = useKnirvana(s => s.updateRewardAnchor);

  const { isMuted, toggleMute } = useAudio();
  const [isEpochRunning, setIsEpochRunning] = useState(false);
  const [showVerifier, setShowVerifier] = useState(false);
  const [showCognitiveShell, setShowCognitiveShell] = useState(false);
  const [showEpochResults, setShowEpochResults] = useState(false);
  const [preloadedPrompt] = useState<string>('');
  const [sabotageType, setSabotageType] = useState<SabotageType>(SabotageType.NOISE_INJECTION);
  const [sabotageMagnitude, setSabotageMagnitude] = useState<number>(1);

  // Initialize Tournament Controller on mount
  useEffect(() => {
    initializeTournamentController();
  }, []);

  // Auto-show results panel when a new epoch completes
  useEffect(() => {
    if (lastEpochResult) {
      setShowEpochResults(true);
    }
  }, [lastEpochResult]);

  // ── Menu screen ──────────────────────────────────────────────────────────

  if (gamePhase === 'menu') {
    return (
      <div className="fixed inset-0 flex items-center justify-center z-50 bg-black/80">
        <div className="bg-gray-900/90 border border-cyan-500/50 rounded-lg p-8 max-w-md w-full mx-4">
          <h1 className="text-3xl font-bold text-cyan-400 mb-2 text-center">KNIRVANA</h1>
          <p className="text-gray-300 mb-2 text-center text-sm">
            Transform AI errors into collective knowledge
          </p>
          <p className="text-gray-500 mb-6 text-center text-xs">
            LLM agents compete to fix real bugs — you sculpt the arena and watch them evolve.
          </p>

          {usingMockLLM && (
            <div className="bg-yellow-900/30 border border-yellow-500/30 rounded-lg p-3 mb-4 text-xs text-yellow-200/80 text-center">
              Mock mode — no API keys set. Configure <code className="bg-black/30 px-1 rounded">VITE_OPENAI_API_KEY</code> for live LLM gameplay.
            </div>
          )}

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

  // ── Handlers ─────────────────────────────────────────────────────────────

  const handleRunEpoch = async () => {
    setIsEpochRunning(true);
    await runEpoch();
    setIsEpochRunning(false);
  };

  const handleAnalyze = () => {
    const next = !isAnalyzing;
    setAnalyzing(next);
  };

  const handleSculpt = () => {
    if (!isAnalyzing) return;
    setSculpting(!isSculpting);
  };

  const handleCreateAgent = () => {
    const agentType = ['Analyzer', 'Optimizer', 'Solver'][Math.floor(Math.random() * 3)];
    createAgent(agentType);
  };

  const handleApplySabotage = () => {
    if (!selectedAgent) return;
    applySabotage(selectedAgent, sabotageType, sabotageMagnitude);
  };

  const handleStartTraining = () => {
    if (!selectedAgent) return;
    startTraining(selectedAgent);
  };

  const handleDistillTrajectory = () => {
    if (!selectedAgent) return;
    distillTrajectory(selectedAgent);
  };

  const handleHardenAgent = () => {
    if (!selectedAgent) return;
    hardenAgent(selectedAgent);
  };

  const handlePauseGame = () => {
    pauseGame();
    saveProgress();
  };

  // ── HUD ──────────────────────────────────────────────────────────────────

  return (
    <div className="fixed inset-0 pointer-events-none z-40">

      {/* ── Top bar ──────────────────────────────────────────────────────── */}
      <div className="absolute top-4 left-4 right-4 flex gap-3 z-50 flex-wrap">

        {/* Game title / phase */}
        <div className="bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-3 flex-shrink-0">
          <div className="text-cyan-400 font-bold">KNIRVANA</div>
          <div className="text-gray-400 text-xs">
            Phase: <span className="text-cyan-300">{gamePhase}</span>
            {usingMockLLM && (
              <span className="ml-2 text-yellow-500 text-xs">[mock LLM]</span>
            )}
          </div>
        </div>

        {/* Selected agent info */}
        {selectedAgent && (
          <div className="bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-3 max-w-[220px] flex-shrink-0">
            <h3 className="text-cyan-400 font-semibold text-sm mb-1">AI Agent</h3>
            <div className="text-xs space-y-0.5">
              <div className="text-gray-300">
                <span className="text-gray-500">Type:</span>{' '}
                {agents.find(a => a.id === selectedAgent)?.type}
              </div>
              <div className="text-gray-300">
                <span className="text-gray-500">Status:</span>{' '}
                <span className="text-green-400">
                  {agents.find(a => a.id === selectedAgent)?.status}
                </span>
              </div>
              <div className="text-gray-300">
                <span className="text-gray-500">Policy:</span>{' '}
                <span className="text-purple-400">
                  {agents.find(a => a.id === selectedAgent)?.policy}
                </span>
              </div>
              <div className="text-gray-300">
                <span className="text-gray-500">Compute:</span>{' '}
                <span className="text-blue-400">
                  {agents.find(a => a.id === selectedAgent)?.resources.compute.toFixed(0)}
                </span>
              </div>
              <div className="text-gray-300">
                <span className="text-gray-500">Parity:</span>{' '}
                <span className="text-green-400">
                  {agents.find(a => a.id === selectedAgent)?.resources.parity.toFixed(0)}
                </span>
              </div>
            </div>
          </div>
        )}

        {/* Tournament panel */}
        <div className="bg-gray-900/80 backdrop-blur-sm border border-purple-500/30 rounded-lg p-3 flex-shrink-0 ml-auto">
          <h3 className="text-purple-400 font-semibold text-sm mb-1">Tournament</h3>
          <div className="space-y-1 text-xs">
            <div className="text-gray-300">
              Epoch: <span className="text-cyan-300">#{epochNumber}</span>
            </div>
            <div className="text-gray-300">
              Skill Slot: <span className="text-purple-300">{skillSlotOwner || 'Uncontested'}</span>
            </div>
            <div className="text-gray-300">
              Incumbent: <span className="text-yellow-400">{(incumbentScore * 100).toFixed(0)}%</span>
            </div>

            {/* Red Queen meter */}
            <div>
              <div className="text-xs text-gray-500 mb-0.5">Red Queen Meter</div>
              <div className="w-full bg-gray-700 rounded-full h-1.5">
                <div
                  className="bg-purple-500 h-1.5 rounded-full transition-all"
                  style={{ width: `${Math.min(100, incumbentScore * 100)}%` }}
                />
              </div>
            </div>

            <div className="flex gap-1 pt-1">
              <button
                onClick={handleRunEpoch}
                disabled={isEpochRunning}
                className={`flex-1 px-2 py-1.5 rounded text-xs pointer-events-auto transition-colors font-medium ${
                  isEpochRunning
                    ? 'bg-yellow-700 text-white cursor-not-allowed'
                    : 'bg-green-600 hover:bg-green-500 text-white'
                }`}
              >
                {isEpochRunning ? 'Running...' : 'Run Epoch'}
              </button>

              {lastEpochResult && (
                <button
                  onClick={() => setShowEpochResults(true)}
                  className="px-2 py-1.5 rounded text-xs pointer-events-auto bg-purple-700 hover:bg-purple-600 text-white transition-colors"
                >
                  Results
                </button>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* ── Stats panel (left) ────────────────────────────────────────────── */}
      <div className="absolute top-28 left-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-3 z-50">
        <h3 className="text-cyan-400 font-semibold text-sm mb-1">Statistics</h3>
        <div className="space-y-0.5 text-xs">
          <div className="text-gray-300">
            NRN: <span className="text-yellow-400 font-bold">{nrnBalance}</span>
          </div>
          <div className="text-gray-300">
            Errors Fixed: <span className="text-green-400">{errorsResolved}</span>
          </div>
          <div className="text-gray-300">
            Skills: <span className="text-cyan-400">{skillsLearned}</span>
          </div>
        </div>
      </div>

      {/* ── Agent management panel (left) ─────────────────────────────────── */}
      <div className="absolute top-52 left-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-3 z-50 w-52">
        <h3 className="text-cyan-400 font-semibold text-sm mb-2">Agent Management</h3>
        <div className="space-y-2 text-sm">
          <button
            onClick={handleCreateAgent}
            className="w-full bg-blue-600 hover:bg-blue-500 text-white px-3 py-1.5 rounded text-xs pointer-events-auto transition-colors"
          >
            Create Agent (50 NRN)
          </button>

          {selectedAgent && (
            <>
              <div className="pt-1 border-t border-gray-700">
                <h4 className="text-cyan-300 font-medium text-xs mb-1">Training</h4>
                <div className="grid grid-cols-2 gap-1">
                  <button
                    onClick={handleStartTraining}
                    className="bg-purple-700 hover:bg-purple-600 text-white px-2 py-1 rounded text-xs pointer-events-auto transition-colors"
                  >
                    Train
                  </button>
                  <button
                    onClick={handleDistillTrajectory}
                    className="bg-indigo-700 hover:bg-indigo-600 text-white px-2 py-1 rounded text-xs pointer-events-auto transition-colors"
                  >
                    Distill
                  </button>
                  <button
                    onClick={handleHardenAgent}
                    className="bg-red-700 hover:bg-red-600 text-white px-2 py-1 rounded text-xs pointer-events-auto col-span-2 transition-colors"
                  >
                    Harden
                  </button>
                </div>
              </div>

              <div className="pt-1 border-t border-gray-700">
                <h4 className="text-red-300 font-medium text-xs mb-1">Sabotage</h4>
                <div className="space-y-1">
                  <select
                    value={sabotageType}
                    onChange={e => setSabotageType(e.target.value as SabotageType)}
                    className="w-full bg-gray-800 text-white text-xs p-1 rounded pointer-events-auto border border-gray-600"
                  >
                    <option value={SabotageType.NOISE_INJECTION}>Noise Injection</option>
                    <option value={SabotageType.BACKPROP_PULSE}>Backprop Pulse</option>
                    <option value={SabotageType.GRADIENT_GHOSTING}>Gradient Ghosting</option>
                  </select>
                  <div className="flex items-center gap-2">
                    <span className="text-gray-400 text-xs">Mag:</span>
                    <input
                      type="range" min="0.1" max="5" step="0.1"
                      value={sabotageMagnitude}
                      onChange={e => setSabotageMagnitude(parseFloat(e.target.value))}
                      className="flex-1 pointer-events-auto"
                    />
                    <span className="text-gray-300 text-xs w-6">{sabotageMagnitude.toFixed(1)}</span>
                  </div>
                  <button
                    onClick={handleApplySabotage}
                    className="w-full bg-red-800 hover:bg-red-700 text-white px-2 py-1 rounded text-xs pointer-events-auto transition-colors"
                  >
                    Apply
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* ── Error node info (right) ───────────────────────────────────────── */}
      {selectedErrorNode && (
        <div className="absolute top-28 right-4 bg-gray-900/80 backdrop-blur-sm border border-orange-500/30 rounded-lg p-3 w-56 z-50">
          <h3 className="text-orange-400 font-semibold text-sm mb-1">Error Node</h3>
          <div className="text-xs space-y-0.5">
            <div className="text-gray-300">
              <span className="text-gray-500">Type:</span>{' '}
              {errorNodes.find(n => n.id === selectedErrorNode)?.type}
            </div>
            <div className="text-gray-300">
              <span className="text-gray-500">Status:</span>{' '}
              <span className="text-orange-400">
                {errorNodes.find(n => n.id === selectedErrorNode)?.isBeingSolved ? 'In Progress' : 'Active'}
              </span>
            </div>
            <div className="text-gray-300">
              <span className="text-gray-500">Difficulty:</span>{' '}
              <span className="text-red-400">
                {Math.round((errorNodes.find(n => n.id === selectedErrorNode)?.difficulty ?? 0) * 100)}%
              </span>
            </div>
            <div className="text-gray-300">
              <span className="text-gray-500">Bounty:</span>{' '}
              <span className="text-yellow-400">
                {errorNodes.find(n => n.id === selectedErrorNode)?.bounty} NRN
              </span>
            </div>
            {selectedAgent && !errorNodes.find(n => n.id === selectedErrorNode)?.isBeingSolved && (
              <button
                onClick={() => deployAgent(selectedAgent, selectedErrorNode)}
                className="mt-2 w-full bg-cyan-700 hover:bg-cyan-600 text-white px-2 py-1.5 rounded text-xs pointer-events-auto transition-colors"
              >
                Deploy Agent (10 NRN)
              </button>
            )}
          </div>
        </div>
      )}

      {/* ── Sculpt mode indicator ─────────────────────────────────────────── */}
      {isSculpting && (
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 pointer-events-none">
          <div className="bg-yellow-500/20 border-2 border-yellow-500 rounded-lg px-6 py-3 text-yellow-300 text-sm text-center">
            Click on the arena floor to place a reward anchor
          </div>
        </div>
      )}

      {/* ── Bottom controls ───────────────────────────────────────────────── */}
      <div className="absolute bottom-4 left-1/2 -translate-x-1/2 flex gap-2 z-50 flex-wrap justify-center">
        <button
          onClick={handleAnalyze}
          className={`px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors ${
            isAnalyzing
              ? 'bg-red-500 ring-2 ring-red-300 text-white'
              : 'bg-red-700 hover:bg-red-600 text-white'
          }`}
        >
          {isAnalyzing ? 'Analyzing...' : 'Analyze'}
        </button>

        <button
          onClick={handleSculpt}
          disabled={!isAnalyzing}
          className={`px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors ${
            isSculpting
              ? 'bg-yellow-500 ring-2 ring-yellow-300 text-white'
              : isAnalyzing
                ? 'bg-yellow-700 hover:bg-yellow-600 text-white'
                : 'bg-gray-700 text-gray-500 cursor-not-allowed'
          }`}
        >
          {isSculpting ? 'Sculpting...' : 'Sculpt'}
        </button>

        <button
          onClick={() => setShowVerifier(true)}
          disabled={!selectedRewardAnchor}
          className={`px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors ${
            selectedRewardAnchor
              ? 'bg-green-700 hover:bg-green-600 text-white'
              : 'bg-gray-700 text-gray-500 cursor-not-allowed'
          }`}
        >
          Verify{!selectedRewardAnchor ? ' (Select Anchor)' : ''}
        </button>

        <button
          onClick={() => setShowCognitiveShell(true)}
          className="bg-cyan-700 hover:bg-cyan-600 text-white px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors"
        >
          Cognitive Shell
        </button>

        <button
          onClick={handlePauseGame}
          className="bg-gray-700 hover:bg-gray-600 text-white px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors"
        >
          Pause
        </button>

        <button
          onClick={toggleMute}
          className="bg-gray-700 hover:bg-gray-600 text-white px-3 py-2 rounded-lg pointer-events-auto text-sm transition-colors"
          aria-label={isMuted ? 'Unmute' : 'Mute'}
        >
          {isMuted ? '🔇' : '🔊'}
        </button>
      </div>

      {/* ── Epoch results panel ───────────────────────────────────────────── */}
      {showEpochResults && lastEpochResult && (
        <EpochResultsPanel
          result={lastEpochResult}
          epochNumber={epochNumber}
          onClose={() => setShowEpochResults(false)}
        />
      )}

      {/* ── Cognitive Shell modal ─────────────────────────────────────────── */}
      {showCognitiveShell && (
        <ChatBrainProvider>
          <CognitiveShellInterface preloadedPrompt={preloadedPrompt} />
        </ChatBrainProvider>
      )}

      {/* ── Verifier overlay ──────────────────────────────────────────────── */}
      {showVerifier && selectedRewardAnchor && (
        <VerifierOverlay
          onClose={() => setShowVerifier(false)}
          rewardAnchors={rewardAnchors}
          updateRewardAnchor={updateRewardAnchor}
          initialAnchor={rewardAnchors.find(a => a.id === selectedRewardAnchor) ?? null}
        />
      )}
    </div>
  );
}
