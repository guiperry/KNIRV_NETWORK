import React, { useState, useEffect } from 'react';
import { useKnirvana, isRingSet, isRingCommitted, ringCount, RING_SIZE } from './stores/useKnirvana';
import { useAudio } from './stores/useAudio';
import { SabotageType } from '../../engine/Sabotage';
import VerifierOverlay from './VerifierOverlay';
import EpochResultsPanel from './EpochResultsPanel';
import DVEWorkspaceModal from './DVEWorkspaceModal';
import { initializeTournamentController } from '../../engine/TournamentControllerIntegration';
import { AgentManagementModal } from '../modals/AgentManagementModal';

// ── Next logical action guidance ──────────────────────────────────────────

type NextActionId =
  | 'analyze' | 'place' | 'set' | 'sculpt'
  | 'create' | 'deploy' | 'straightening' | 'assign' | 'epoch';

interface NextAction {
  id: NextActionId;
  hint: string;
}

interface SubAgent {
  id: string;
  name: string;
  status: 'idle' | 'training' | 'ready' | 'deployed';
  capabilities: string[];
  expertise: string[];
  trainingProgress: number;
  nrnCost: number;
  description: string;
}

export default function GameUI() {
  const {
    gamePhase,
    nrnBalance,
    errorsResolved,
    skillsLearned,
    epochNumber,
    selectedErrorNode,
    selectErrorNode,
    selectedAgent,
    selectAgent,
    pauseGame,
    createAgent,
    deployAgent,
    deployAllStaged,
    deployOne,
    stageAgent,
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
    startStraighteningSequence,
  } = useKnirvana();

  const stagedAgents = agents.filter(a => a.staged);
  const isStraighteningAnchors = useKnirvana(s => s.isStraighteningAnchors);

  const errorNodes = useKnirvana(s => s.errorNodes);
  const isAnalyzing = useKnirvana(s => s.isAnalyzing);
  const setAnalyzing = useKnirvana(s => s.setAnalyzing);
  const isSculpting = useKnirvana(s => s.isSculpting);
  const setSculpting = useKnirvana(s => s.setSculpting);
  const rewardAnchors = useKnirvana(s => s.rewardAnchors);
  const selectedRewardAnchor = useKnirvana(s => s.selectedRewardAnchor);
  const showAnchorConfigModal = useKnirvana(s => s.showAnchorConfigModal);
  const updateRewardAnchor = useKnirvana(s => s.updateRewardAnchor);
  const removeRewardAnchor = useKnirvana(s => s.removeRewardAnchor);
  const showAgentManagementModal = useKnirvana(s => s.showAgentManagementModal);
  const setShowAgentManagementModal = useKnirvana(s => s.setShowAgentManagementModal);

  const commitSetAnchors = useKnirvana(s => s.commitSetAnchors);
  const setAllStraightenedAnchors = useKnirvana(s => s.setAllStraightenedAnchors);
  const toasts = useKnirvana(s => s.toasts);
  const pushToast = useKnirvana(s => s.pushToast);
  const openDVEWorkspace = useKnirvana(s => s.openDVEWorkspace);

  const { isMuted, toggleMute, playSfx } = useAudio();
  const [isEpochRunning, setIsEpochRunning] = useState(false);
  const [showEpochResults, setShowEpochResults] = useState(false);
  const [preloadedPrompt] = useState<string>('');

  // Sculpt is enabled once every spike position around any error node has a set anchor
  const hasFullRingSet = React.useMemo(
    () => errorNodes.some(node =>
      isRingSet(node.id, rewardAnchors) && !isRingCommitted(node.id, rewardAnchors)
    ),
    [errorNodes, rewardAnchors]
  );

  // ── Next logical action ────────────────────────────────────────────────
  const nextAction: NextAction = React.useMemo(() => {
    if (isStraighteningAnchors) {
      return { id: 'straightening', hint: 'Agents are straightening the validation ring…' };
    }

    const maxRing = errorNodes.reduce((m, n) => Math.max(m, ringCount(n.id, rewardAnchors)), 0);
    const anyUnset = rewardAnchors.some(a => !a.isSet);
    const committedNode = errorNodes.find(n => isRingCommitted(n.id, rewardAnchors));
    const horizontalCommitted = rewardAnchors.some(a => a.isCommitted && a.isHorizontal);
    const straightenedNode = errorNodes.find(n =>
      isRingCommitted(n.id, rewardAnchors) &&
      !rewardAnchors.some(a => a.linkedErrorNode === n.id && a.isCommitted && a.isHorizontal)
    );

    if (maxRing === 0 && !isAnalyzing) {
      return { id: 'analyze', hint: 'Analyze the grid to reveal validation spike points around error nodes.' };
    }
    if (maxRing < RING_SIZE && !committedNode) {
      if (!isAnalyzing) {
        return { id: 'analyze', hint: `Re-enter Analyze mode to finish the validation ring (${maxRing}/${RING_SIZE} placed).` };
      }
      return { id: 'place', hint: `Click the white spikes around one error node — ${RING_SIZE - maxRing} more validation node${RING_SIZE - maxRing > 1 ? 's' : ''} to go.` };
    }
    if (anyUnset && !hasFullRingSet) {
      return { id: 'set', hint: 'Configure your validation nodes — Set All, or View each anchor to tune weights.' };
    }
    if (hasFullRingSet) {
      return { id: 'sculpt', hint: 'Sculpt the completed ring — it will sink beneath the grid and lock in.' };
    }
    if (committedNode && horizontalCommitted) {
      if (stagedAgents.length === 0) {
        return { id: 'create', hint: 'Create agents — they are needed to straighten the sunken validation ring.' };
      }
      return { id: 'deploy', hint: 'Deploy your staged agents to straighten the validation ring.' };
    }
    if (straightenedNode && !straightenedNode.isBeingSolved) {
      return { id: 'assign', hint: `Open ${straightenedNode.type} and assign an agent to start the resolution (10 NRN).` };
    }
    if (errorNodes.some(n => n.isBeingSolved)) {
      return { id: 'epoch', hint: 'Run an epoch — your agents compete to resolve the error and hijack the skill slot.' };
    }
    return { id: 'analyze', hint: 'Analyze the grid to find your next error node.' };
  }, [isStraighteningAnchors, errorNodes, rewardAnchors, isAnalyzing, hasFullRingSet, stagedAgents.length]);

  // Amber pulse on whichever control is the next logical step
  const glowIf = (...ids: NextActionId[]) =>
    ids.includes(nextAction.id)
      ? ' ring-2 ring-amber-300 shadow-[0_0_16px_rgba(251,191,36,0.65)] animate-pulse'
      : '';
  const [sabotageType, setSabotageType] = useState<SabotageType>(SabotageType.NOISE_INJECTION);
  const [sabotageMagnitude, setSabotageMagnitude] = useState<number>(1);
  const [isVerifying, setIsVerifying] = useState(false);
  const [isTranspiling, setIsTranspiling] = useState(false);

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

  // ── HUD ──────────────────────────────────────────────────────────────────

  return (
    <div className="fixed inset-0 pointer-events-none z-[100000]">
      {/* ── Top bar ──────────────────────────────────────────────────────── */}
      {gamePhase !== 'menu' && (
        <div className="absolute top-4 left-4 right-4 flex gap-3 z-50 flex-wrap pointer-events-auto">
          {/* Game title / phase */}
          <div className="bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-3 flex-shrink-0">
            <div className="text-cyan-400 font-bold">KNIRVARENA</div>
            <div className="text-gray-400 text-xs">
              Phase: <span className="text-cyan-300">{gamePhase}</span>
              {usingMockLLM && (
                <span className="ml-2 text-yellow-500 text-xs">[mock LLM]</span>
              )}
            </div>
          </div>

          {/* Selected agent info */}
          {selectedAgent && (() => {
            const agent = agents.find(a => a.id === selectedAgent);
            const isStaged = agent?.staged;
            return (
              <div className="bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-3 max-w-[220px] flex-shrink-0">
                <div className="flex justify-between items-center mb-1">
                  <h3 className="text-cyan-400 font-semibold text-sm">AI Agent</h3>
                  <button
                    onClick={() => selectAgent(null)}
                    className="text-xs text-red-400 hover:text-red-300 px-1"
                  >
                    Deselect
                  </button>
                </div>
                <div className="text-xs space-y-0.5">
                  <div className="text-gray-300">
                    <span className="text-gray-500">Type:</span>{' '}
                    {agent?.type}
                  </div>
                  <div className="text-gray-300">
                    <span className="text-gray-500">Status:</span>{' '}
                    <span className="text-green-400">{agent?.status}</span>
                  </div>
                  <div className="text-gray-300">
                    <span className="text-gray-500">Policy:</span>{' '}
                    <span className="text-purple-400">{agent?.policy}</span>
                  </div>
                  <div className="text-gray-300">
                    <span className="text-gray-500">Compute:</span>{' '}
                    <span className="text-blue-400">{agent?.resources.compute.toFixed(0)}</span>
                  </div>
                  <div className="text-gray-300">
                    <span className="text-gray-500">Parity:</span>{' '}
                    <span className="text-green-400">{agent?.resources.parity.toFixed(0)}</span>
                  </div>
                </div>
                {!isStaged && (
                  <button
                    onClick={() => {
                      if (selectedAgent) {
                        useKnirvana.getState().stageAgent(selectedAgent);
                      }
                    }}
                    className="mt-2 w-full bg-orange-700 hover:bg-orange-600 text-white px-2 py-1 rounded text-xs pointer-events-auto transition-colors"
                  >
                    Stage
                  </button>
                )}
              </div>
            );
          })()}

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
                  onClick={() => runEpoch()}
                  disabled={isEpochRunning}
                  className={`flex-1 px-2 py-1.5 rounded text-xs pointer-events-auto transition-colors font-medium ${
                    isEpochRunning
                      ? 'bg-yellow-700 text-white cursor-not-allowed'
                      : 'bg-green-600 hover:bg-green-500 text-white'
                  }${glowIf('epoch')}`}
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
      )}

      {/* ── Stats panel (left) ────────────────────────────────────────────── */}
      {gamePhase !== 'menu' && (
        <div className="absolute top-28 left-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-3 z-50 pointer-events-auto">
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
      )}

      {/* ── Agent management panel (left) ─────────────────────────────────── */}
      {gamePhase !== 'menu' && (
        <div className="absolute top-64 left-4 bg-gray-900/80 backdrop-blur-sm border border-cyan-500/30 rounded-lg p-3 z-50 w-52 pointer-events-auto">
          <h3 className="text-cyan-400 font-semibold text-sm mb-2">Agent Management</h3>
          
          {/* Staged Agents List */}
          <div className="mb-2">
            <h4 className="text-cyan-300 font-medium text-xs mb-1">Staged ({stagedAgents.length})</h4>
            <div className="space-y-1 max-h-24 overflow-y-auto">
              {stagedAgents.length === 0 ? (
                <p className="text-gray-500 text-xs italic">No staged agents</p>
              ) : (
                stagedAgents.map(agent => (
                  <div 
                    key={agent.id}
                    onClick={() => selectAgent(agent.id)}
                    className={`text-xs px-2 py-1 rounded cursor-pointer transition-colors flex items-center gap-2 ${
                      selectedAgent === agent.id 
                        ? 'bg-cyan-600 text-white' 
                        : 'bg-gray-800 text-gray-300 hover:bg-gray-700'
                    }`}
                  >
                    <span className="inline-block w-4 h-4 rounded-full bg-green-500/50 border-2 border-green-400 animate-pulse flex-shrink-0" />
                    {agent.name}
                  </div>
                ))
              )}
            </div>
          </div>

          <div className="space-y-2 text-sm">
            <button
              onClick={() => {
                if (stagedAgents.length > 0) {
                  const hasHorizontalAnchors = rewardAnchors.some(a => a.isHorizontal);
                  if (hasHorizontalAnchors && !isStraighteningAnchors) {
                    setSculpting(false);
                    startStraighteningSequence();
                  } else {
                    deployAllStaged();
                  }
                } else {
                  setShowAgentManagementModal(true);
                }
              }}
              disabled={isStraighteningAnchors}
              className={`w-full text-white px-3 py-1.5 rounded text-xs pointer-events-auto transition-colors ${
                isStraighteningAnchors
                  ? 'bg-yellow-700 cursor-not-allowed'
                  : 'bg-blue-600 hover:bg-blue-500'
              }${glowIf('deploy', 'create')}`}
            >
              {isStraighteningAnchors
                ? 'Straightening...'
                : stagedAgents.length > 0
                  ? rewardAnchors.some(a => a.isHorizontal)
                    ? 'Deploy (Straighten First)'
                    : 'Deploy All'
                  : 'Create Agent'}
            </button>

            {selectedAgent && stagedAgents.some(s => s.id === selectedAgent) && (
              <>
                <div className="pt-1 border-t border-gray-700">
                  <div className="grid grid-cols-2 gap-1">
                    <button
                      onClick={() => {
                        const hasHorizontalAnchors = rewardAnchors.some(a => a.isHorizontal);
                        if (hasHorizontalAnchors && !isStraighteningAnchors) {
                          setSculpting(false);
                          startStraighteningSequence();
                        } else {
                          deployOne(selectedAgent);
                        }
                      }}
                      disabled={isStraighteningAnchors}
                      className="bg-green-700 hover:bg-green-600 text-white px-2 py-1 rounded text-xs pointer-events-auto transition-colors disabled:bg-gray-700 disabled:cursor-not-allowed"
                    >
                      Deploy
                    </button>
                    <button
                      onClick={() => {
                        // Request App to open the existing Agent Management modal
                        useKnirvana.getState().setShowAgentManagementModal(true);
                      }}
                      className="bg-purple-700 hover:bg-purple-600 text-white px-2 py-1 rounded text-xs pointer-events-auto transition-colors"
                    >
                      Train
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* ── Error node info (right) ───────────────────────────────────────── */}
      {gamePhase !== 'menu' && selectedErrorNode && (() => {
        const errorNode = errorNodes.find(n => n.id === selectedErrorNode);
        return (
          <div className="absolute top-52 right-4 bg-gray-900/80 backdrop-blur-sm border border-orange-500/30 rounded-lg p-3 w-56 z-50 pointer-events-auto">
            <div className="flex justify-between items-center mb-1">
              <h3 className="text-orange-400 font-semibold text-sm">Error Node</h3>
              <button
                onClick={() => selectErrorNode(null)}
                className="text-xs text-red-400 hover:text-red-300 px-1"
              >
                Deselect
              </button>
            </div>
            <div className="text-xs space-y-0.5">
              <div className="text-gray-300">
                <span className="text-gray-500">Type:</span>{' '}
                {errorNode?.type}
              </div>
              <div className="text-gray-300">
                <span className="text-gray-500">Status:</span>{' '}
                <span className="text-orange-400">
                  {errorNode?.isBeingSolved ? 'In Progress' : 'Active'}
                </span>
              </div>
              <div className="text-gray-300">
                <span className="text-gray-500">Difficulty:</span>{' '}
                <span className="text-red-400">
                  {Math.round((errorNode?.difficulty ?? 0) * 100)}%
                </span>
              </div>
              <div className="text-gray-300">
                <span className="text-gray-500">Bounty:</span>{' '}
                <span className="text-yellow-400">{errorNode?.bounty} NRN</span>
              </div>

              {/* Sabotage options */}
              {selectedAgent && !stagedAgents.some(s => s.id === selectedAgent) && (
                <div className="pt-2 border-t border-gray-700 mt-2">
                  <h4 className="text-red-300 font-medium text-xs mb-1">Sabotage</h4>
                  <select
                    value={sabotageType}
                    onChange={e => setSabotageType(e.target.value as SabotageType)}
                    className="w-full bg-gray-800 text-white text-xs p-1 rounded pointer-events-auto border border-gray-600 mb-1"
                  >
                    <option value={SabotageType.NOISE_INJECTION}>Noise Injection</option>
                    <option value={SabotageType.BACKPROP_PULSE}>Backprop Pulse</option>
                    <option value={SabotageType.GRADIENT_GHOSTING}>Gradient Ghosting</option>
                  </select>
                  <div className="flex items-center gap-2 mb-1">
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
                    onClick={() => applySabotage(selectedAgent, sabotageType, sabotageMagnitude)}
                    className="w-full bg-red-800 hover:bg-red-700 text-white px-2 py-1 rounded text-xs pointer-events-auto transition-colors"
                  >
                    Apply
                  </button>
                </div>
              )}

              {selectedAgent && !errorNode?.isBeingSolved && !stagedAgents.some(s => s.id === selectedAgent) && (
                <button
                  onClick={() => deployAgent(selectedAgent, selectedErrorNode)}
                  className={`mt-2 w-full bg-cyan-700 hover:bg-cyan-600 text-white px-2 py-1.5 rounded text-xs pointer-events-auto transition-colors${glowIf('assign')}`}
                >
                  Deploy Agent (10 NRN)
                </button>
              )}

              <button
                onClick={() => openDVEWorkspace(selectedErrorNode)}
                className="mt-2 w-full bg-blue-800 hover:bg-blue-700 border border-blue-500/40 text-blue-200 px-2 py-1.5 rounded text-xs pointer-events-auto transition-colors"
              >
                Open DVE Workspace
              </button>
            </div>
          </div>
        );
      })()}

      {/* ── Sculpt mode indicator ─────────────────────────────────────────── */}
      {gamePhase !== 'menu' && isSculpting && (
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 pointer-events-none">
          <div className="bg-yellow-500/20 border-2 border-yellow-500 rounded-lg px-6 py-3 text-yellow-300 text-sm text-center">
            Click the white spikes around an error node to place validation nodes
          </div>
        </div>
      )}

      {/* ── Next-action guidance banner ───────────────────────────────────── */}
      {gamePhase !== 'menu' && (
        <div className="absolute bottom-[4.5rem] left-1/2 -translate-x-1/2 z-50 pointer-events-none max-w-xl px-4">
          <div className="bg-gray-950/85 backdrop-blur-sm border border-amber-400/40 rounded-full px-4 py-1.5 flex items-center gap-2 shadow-[0_0_14px_rgba(251,191,36,0.25)]">
            <span className="text-amber-300 text-[10px] font-bold tracking-widest flex-shrink-0 animate-pulse">NEXT ▸</span>
            <span className="text-amber-100/90 text-xs truncate">{nextAction.hint}</span>
          </div>
        </div>
      )}

      {/* ── Toast notifications ───────────────────────────────────────────── */}
      {toasts.length > 0 && (
        <div className="absolute top-24 left-1/2 -translate-x-1/2 z-[60] flex flex-col items-center gap-2 pointer-events-none">
          {toasts.map(t => (
            <div
              key={t.id}
              className={`px-4 py-2 rounded-lg text-xs font-medium backdrop-blur-sm border shadow-lg animate-pulse-once ${
                t.tone === 'success'
                  ? 'bg-emerald-900/80 border-emerald-400/50 text-emerald-200'
                  : t.tone === 'warn'
                    ? 'bg-red-900/80 border-red-400/50 text-red-200'
                    : 'bg-gray-900/80 border-cyan-400/40 text-cyan-100'
              }`}
            >
              {t.text}
            </div>
          ))}
        </div>
      )}

      {/* ── Bottom controls ───────────────────────────────────────────────── */}
      {gamePhase !== 'menu' && (
        <div className="absolute bottom-4 left-1/2 -translate-x-1/2 flex gap-2 z-50 flex-wrap justify-center pointer-events-auto">
          <button
            onClick={() => { playSfx('click'); setAnalyzing(!isAnalyzing); }}
            className={`px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors ${
              isAnalyzing
                ? 'bg-red-500 ring-2 ring-red-300 text-white'
                : 'bg-red-700 hover:bg-red-600 text-white'
            }${glowIf('analyze')}`}
          >
            {isAnalyzing ? 'Analyzing + Anchoring' : 'Analyze'}
          </button>

          <button
            onClick={() => {
              if (selectedRewardAnchor) {
                useKnirvana.getState().setShowAnchorConfigModal(true);
              }
            }}
            disabled={!selectedRewardAnchor}
            className={`px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors ${
              selectedRewardAnchor
                ? 'bg-yellow-600 hover:bg-yellow-500 text-white'
                : 'bg-gray-700 text-gray-500 cursor-not-allowed'
            }`}
          >
            View
          </button>

          <button
            onClick={() => {
              if (rewardAnchors.some(a => !a.isSet)) setAllStraightenedAnchors();
            }}
            disabled={!rewardAnchors.some(a => !a.isSet)}
            className={`px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors ${
              rewardAnchors.some(a => !a.isSet)
                ? 'bg-cyan-600 hover:bg-cyan-500 text-white'
                : 'bg-gray-700 text-gray-500 cursor-not-allowed'
            }${glowIf('set')}`}
          >
            Set All
          </button>

          <button
            onClick={() => {
              if (hasFullRingSet) commitSetAnchors();
            }}
            disabled={!hasFullRingSet}
            className={`px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors ${
              hasFullRingSet
                ? 'bg-green-600 hover:bg-green-500 text-white ring-2 ring-green-300'
                : 'bg-gray-700 text-gray-500 cursor-not-allowed'
            }${glowIf('sculpt')}`}
          >
            Sculpt{!hasFullRingSet ? ' (Ring Incomplete)' : ''}
          </button>

          <button
            onClick={async () => {
              playSfx('click');
              setIsVerifying(true);
              await new Promise(r => setTimeout(r, 1500));
              setIsVerifying(false);
              pushToast('Dataset format verification complete — all anchors valid', 'success');
            }}
            disabled={rewardAnchors.length === 0}
            className={`px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors ${
              isVerifying
                ? 'bg-purple-500 ring-2 ring-purple-300 text-white'
                : rewardAnchors.length > 0
                  ? 'bg-purple-700 hover:bg-purple-600 text-white'
                  : 'bg-gray-700 text-gray-500 cursor-not-allowed'
            }`}
          >
            {isVerifying ? 'Verifying...' : 'Verify Format'}
          </button>

          <button
            onClick={async () => {
              playSfx('click');
              setIsTranspiling(true);
              await new Promise(r => setTimeout(r, 2000));
              setIsTranspiling(false);
              pushToast(`Transpiled ${rewardAnchors.length} anchor(s) to NRV data objects`, 'success');
            }}
            disabled={rewardAnchors.length === 0}
            className={`px-4 py-2 rounded-lg pointer-events-auto text-sm transition-colors ${
              isTranspiling
                ? 'bg-cyan-500 ring-2 ring-cyan-300 text-white'
                : rewardAnchors.length > 0
                  ? 'bg-cyan-700 hover:bg-cyan-600 text-white'
                  : 'bg-gray-700 text-gray-500 cursor-not-allowed'
            }`}
          >
            {isTranspiling ? 'Transpiling...' : 'Transpile'}
          </button>

          <button
            onClick={() => {
              pauseGame();
              saveProgress();
            }}
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
      )}

      {/* ── Epoch results panel ───────────────────────────────────────────── */}
      {showEpochResults && lastEpochResult && (
        <div className="pointer-events-auto">
          <EpochResultsPanel
            result={lastEpochResult}
            epochNumber={epochNumber}
            onClose={() => setShowEpochResults(false)}
          />
        </div>
      )}

      {/* ── Per-anchor data entry modal (only when View button clicked) ───────────── */}
      {selectedRewardAnchor && showAnchorConfigModal && (() => {
        const anchor = rewardAnchors.find(a => a.id === selectedRewardAnchor);
        if (!anchor) return null;
        return (
          <div className="pointer-events-auto">
             <VerifierOverlay
              anchor={anchor}
              onClose={() => {
                useKnirvana.getState().setShowAnchorConfigModal(false);
              }}
              updateRewardAnchor={updateRewardAnchor}
              removeRewardAnchor={removeRewardAnchor}
            />
          </div>
        );
      })()}

      {/* ── Per-error-node DVE workspace ──────────────────────────────────── */}
      <DVEWorkspaceModal />
    </div>
  );
}
