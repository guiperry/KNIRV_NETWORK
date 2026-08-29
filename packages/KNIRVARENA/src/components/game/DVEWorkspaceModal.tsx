import React, { useState, useEffect, useMemo, useRef } from 'react';
import {
  Box, X, FileText, Folder, FolderOpen, Shield, Brain, Terminal,
  CircleDot, ChevronRight, Anchor, Database, Play,
} from 'lucide-react';
import {
  useKnirvana,
  getRingAnchors,
  ringCount,
  isRingSet,
  isRingCommitted,
  RING_SIZE,
  type RewardAnchor,
} from './stores/useKnirvana';
import { useAudio } from './stores/useAudio';
import { actuarialSyndicateService, type BountyPosting } from '../../services/ActuarialSyndicateService';
import { selectCuratedChallenge } from '../../services/actuarialChallenge';
import { DATASET_TEMPLATES, DEFAULT_TEMPLATE } from './VerifierOverlay';

// Sovereign DVE Workspace — the arena counterpart of KNIRVSERVER's
// dve-workspace-panel. Every error node is assigned its own unique
// workspace; validation nodes (the reward-anchor ring) appear as pages in
// the explorer tree and on the VALIDATION tab.

type WorkspaceTab = 'overview' | 'validation' | 'dataset' | 'console';

type AnchorStage = 'PLACED' | 'SET' | 'COMMITTED' | 'STRAIGHTENED';

function anchorStage(a: RewardAnchor): AnchorStage {
  if (a.isCommitted && a.isHorizontal === false) return 'STRAIGHTENED';
  if (a.isCommitted) return 'COMMITTED';
  if (a.isSet) return 'SET';
  return 'PLACED';
}

const STAGE_COLOR: Record<AnchorStage, string> = {
  PLACED: 'text-amber-400 border-amber-500/40 bg-amber-500/10',
  SET: 'text-blue-400 border-blue-500/40 bg-blue-500/10',
  COMMITTED: 'text-emerald-400 border-emerald-500/40 bg-emerald-500/10',
  STRAIGHTENED: 'text-green-300 border-green-400/50 bg-green-400/10',
};

const STAGE_DOT: Record<AnchorStage, string> = {
  PLACED: 'bg-amber-400',
  SET: 'bg-blue-400',
  COMMITTED: 'bg-emerald-400',
  STRAIGHTENED: 'bg-green-300',
};

// ── Validation node detail page ───────────────────────────────────────────

function ValidationNodePage({ anchor, index }: { anchor: RewardAnchor; index: number }) {
  const updateRewardAnchor = useKnirvana(s => s.updateRewardAnchor);
  const removeRewardAnchor = useKnirvana(s => s.removeRewardAnchor);
  const [weights, setWeights] = useState(anchor.weights);
  const [constraints, setConstraints] = useState(anchor.constraints);

  useEffect(() => {
    setWeights(anchor.weights);
    setConstraints(anchor.constraints);
  }, [anchor.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const stage = anchorStage(anchor);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <div className="text-sm font-bold text-slate-200 font-mono flex items-center gap-2">
            <Anchor className="w-4 h-4 text-cyan-400" />
            validation-node-{String(index + 1).padStart(2, '0')}
            {anchor.anchorType === 'noise' && (
              <span className="text-[10px] px-1.5 py-0.5 rounded border border-purple-500/40 bg-purple-500/10 text-purple-300">NOISE</span>
            )}
          </div>
          <div className="text-[11px] text-slate-500 font-mono mt-0.5">{anchor.id}</div>
        </div>
        <span className={`text-[10px] font-mono px-2 py-1 rounded border ${STAGE_COLOR[stage]}`}>● {stage}</span>
      </div>

      <div className="grid grid-cols-2 gap-3 text-xs">
        <div className="p-3 rounded-lg border border-slate-800 bg-slate-900/60">
          <div className="text-slate-500 mb-1">Position</div>
          <div className="font-mono text-slate-300">
            ({anchor.position.x.toFixed(2)}, {anchor.position.z.toFixed(2)})
          </div>
        </div>
        <div className="p-3 rounded-lg border border-slate-800 bg-slate-900/60">
          <div className="text-slate-500 mb-1">Orientation</div>
          <div className="font-mono text-slate-300">
            {anchor.isHorizontal === false ? 'VERTICAL (straightened)' : 'HORIZONTAL'}
          </div>
        </div>
      </div>

      {/* Reward weights */}
      <div className="p-3 rounded-lg border border-slate-800 bg-slate-900/60 space-y-2">
        <div className="text-[11px] text-cyan-400 font-mono tracking-wider">REWARD_WEIGHTS</div>
        {([['w_c', 'Correctness'], ['w_l', 'Latency'], ['w_s', 'Simplicity']] as const).map(([key, label]) => (
          <div key={key} className="flex items-center gap-2 text-xs">
            <span className="text-slate-400 w-24">{label}</span>
            <input
              type="range" min="0" max="1" step="0.05"
              value={weights[key]}
              disabled={anchor.isCommitted}
              onChange={e => setWeights(w => ({ ...w, [key]: parseFloat(e.target.value) }))}
              className="flex-1"
            />
            <span className="font-mono text-slate-300 w-10 text-right">{weights[key].toFixed(2)}</span>
          </div>
        ))}
      </div>

      {/* Constraints */}
      <div className="p-3 rounded-lg border border-slate-800 bg-slate-900/60">
        <div className="text-[11px] text-cyan-400 font-mono tracking-wider mb-2">UNIT_TEST_INJECTION</div>
        <textarea
          value={constraints}
          disabled={anchor.isCommitted}
          onChange={e => setConstraints(e.target.value)}
          spellCheck={false}
          className="w-full h-28 bg-black/60 border border-slate-800 rounded p-2 text-[11px] font-mono text-emerald-300 resize-none disabled:opacity-60"
        />
      </div>

      <div className="flex gap-2">
        {!anchor.isCommitted && (
          <>
            <button
              onClick={() => updateRewardAnchor(anchor.id, { weights, constraints, isSet: true })}
              className="flex-1 px-3 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold transition-colors"
            >
              {anchor.isSet ? 'UPDATE NODE' : 'SET VALIDATION NODE'}
            </button>
            <button
              onClick={() => removeRewardAnchor(anchor.id)}
              className="px-3 py-2 rounded-lg bg-red-900/40 hover:bg-red-700 border border-red-500/30 text-red-300 text-xs transition-colors"
            >
              REMOVE
            </button>
          </>
        )}
        {anchor.isCommitted && (
          <div className="flex-1 px-3 py-2 rounded-lg border border-emerald-500/30 bg-emerald-500/10 text-emerald-300 text-xs text-center font-mono">
            Committed beneath the grid — locked
          </div>
        )}
      </div>
    </div>
  );
}

// ── Main modal ────────────────────────────────────────────────────────────

export default function DVEWorkspaceModal() {
  const nodeId = useKnirvana(s => s.dveWorkspaceNodeId);
  const errorNodes = useKnirvana(s => s.errorNodes);
  const rewardAnchors = useKnirvana(s => s.rewardAnchors);
  const agents = useKnirvana(s => s.agents);
  const dveWorkspaces = useKnirvana(s => s.dveWorkspaces);
  const closeDVEWorkspace = useKnirvana(s => s.closeDVEWorkspace);
  const updateDVEWorkspace = useKnirvana(s => s.updateDVEWorkspace);
  const setAllStraightenedAnchors = useKnirvana(s => s.setAllStraightenedAnchors);
  const commitSetAnchors = useKnirvana(s => s.commitSetAnchors);
  const deployAgent = useKnirvana(s => s.deployAgent);
  const setAnalyzing = useKnirvana(s => s.setAnalyzing);
  const playSfx = useAudio(s => s.playSfx);

  const node = errorNodes.find(n => n.id === nodeId);
  const meta = nodeId ? dveWorkspaces[nodeId] : undefined;

  const [tab, setTab] = useState<WorkspaceTab>(meta?.lastTab ?? 'overview');
  const [page, setPage] = useState<string | null>(meta?.lastPage ?? 'overview');
  const [backendPostings, setBackendPostings] = useState<BountyPosting[]>([]);
  const consoleEndRef = useRef<HTMLDivElement>(null);

  // Backend postings supersede the local catalog after the one-time seed. The
  // fallback keeps old saved games usable until that admin migration is run.
  useEffect(() => { let active = true; actuarialSyndicateService.listPostings('code_error').then(items => { if (active) setBackendPostings(items); }).catch(() => {}); return () => { active = false; }; }, []);

  // Restore per-workspace tab/page when switching nodes
  useEffect(() => {
    if (meta) {
      setTab(meta.lastTab);
      setPage(meta.lastPage);
    }
  }, [nodeId]); // eslint-disable-line react-hooks/exhaustive-deps

  // Persist navigation on the workspace assigned to this node
  useEffect(() => {
    if (nodeId && meta) updateDVEWorkspace(nodeId, { lastTab: tab, lastPage: page });
  }, [tab, page]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    consoleEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [meta?.log.length, tab]);

  const ringAnchors = useMemo(
    () => (node ? getRingAnchors(node.id, rewardAnchors) : []),
    [node, rewardAnchors]
  );
  const noiseAnchors = useMemo(
    () => (node ? rewardAnchors.filter(a => a.linkedErrorNode === node.id && a.anchorType === 'noise') : []),
    [node, rewardAnchors]
  );
  const allAnchors = useMemo(() => [...ringAnchors, ...noiseAnchors], [ringAnchors, noiseAnchors]);

  if (!nodeId || !node || !meta) return null;

  const placed = ringCount(node.id, rewardAnchors);
  const ringSet = isRingSet(node.id, rewardAnchors);
  const ringCommitted = isRingCommitted(node.id, rewardAnchors);
  const ringStraightened = ringCommitted && ringAnchors.length > 0 && ringAnchors.every(a => a.isHorizontal === false);
  const challenge = selectCuratedChallenge(backendPostings, node.challengeId);
  const template = DATASET_TEMPLATES[node.type] ?? DEFAULT_TEMPLATE;
  const idleFieldAgent = agents.find(a => !a.staged && a.status === 'idle');

  const selectedAnchor = page && page !== 'overview' ? allAnchors.find(a => a.id === page) : undefined;
  const selectedAnchorIndex = selectedAnchor ? allAnchors.indexOf(selectedAnchor) : -1;

  const openPage = (tabName: WorkspaceTab, pageName: string | null) => {
    playSfx('click');
    setTab(tabName);
    setPage(pageName);
  };

  const nextHint = !placed
    ? 'Enter Analyze mode and click the white spikes to place 8 validation nodes.'
    : placed < RING_SIZE
      ? `Place ${RING_SIZE - placed} more validation node${RING_SIZE - placed > 1 ? 's' : ''} to complete the ring.`
      : !ringSet
        ? 'Configure and SET each validation node (or use SET ALL).'
        : !ringCommitted
          ? 'Sculpt the ring — it will sink beneath the grid.'
          : !ringStraightened
            ? 'Deploy staged agents to straighten the sunken ring.'
            : !node.isBeingSolved
              ? 'Assign an agent to begin resolving this error.'
              : 'Resolution in progress — run epochs to accelerate.';

  return (
    <div className="fixed inset-0 z-[100001] flex items-center justify-center p-4 md:p-10 pointer-events-auto">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/75 backdrop-blur-sm" onClick={closeDVEWorkspace} />

      <div className="relative w-full max-w-5xl h-[82vh] flex flex-col bg-[#03050a] border border-blue-600/50 rounded-xl overflow-hidden shadow-[0_0_60px_rgba(37,99,235,0.25)]">
        {/* ── Header ─────────────────────────────────────────────────── */}
        <div className="h-14 border-b border-blue-600/30 bg-slate-900/80 backdrop-blur-sm px-4 flex items-center justify-between flex-shrink-0">
          <div className="flex items-center gap-3 min-w-0">
            <Box className="w-5 h-5 text-blue-500 animate-pulse flex-shrink-0" />
            <div className="min-w-0">
              <h1 className="text-xs font-black tracking-tighter text-blue-100 uppercase truncate">
                Sovereign DVE Workspace — {node.type}
              </h1>
              <p className="text-[10px] font-mono text-slate-500 truncate">
                Secure Enclave Context: {meta.workspaceId}
              </p>
            </div>
            <div className="hidden md:flex gap-1.5 ml-2">
              <span className="text-[9px] font-mono px-1.5 py-0.5 rounded border border-green-500/20 bg-green-500/10 text-green-400">TEE: VERIFIED</span>
              <span className="text-[9px] font-mono px-1.5 py-0.5 rounded border border-blue-500/20 bg-blue-500/10 text-blue-400">FABRIC: SYNCED</span>
              <span className={`text-[9px] font-mono px-1.5 py-0.5 rounded border flex items-center gap-1 ${
                ringCommitted
                  ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-400'
                  : 'border-purple-500/20 bg-purple-500/10 text-purple-400'
              }`}>
                <Brain className="w-2.5 h-2.5" />
                RING: {placed}/{RING_SIZE}{ringCommitted ? ' COMMITTED' : ringSet ? ' SET' : ''}
              </span>
            </div>
          </div>
          <button
            onClick={closeDVEWorkspace}
            className="bg-red-900/20 hover:bg-red-600 text-red-500 hover:text-white p-1.5 rounded-lg transition-colors border border-red-500/20 flex-shrink-0"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="flex-1 flex min-h-0">
          {/* ── Explorer ─────────────────────────────────────────────── */}
          <div className="w-56 border-r border-slate-800 bg-slate-950/60 flex flex-col flex-shrink-0 overflow-y-auto">
            <div className="px-3 py-2 text-[10px] font-mono text-slate-500 tracking-widest border-b border-slate-800/60">
              DVE EXPLORER
            </div>
            <div className="p-2 text-[11px] font-mono space-y-0.5">
              <div className="flex items-center gap-1.5 text-slate-300 px-1 py-0.5">
                <FolderOpen className="w-3.5 h-3.5 text-blue-400" />
                dve-{node.id}/
              </div>

              <button
                onClick={() => openPage('overview', 'overview')}
                className={`w-full flex items-center gap-1.5 px-1 py-0.5 pl-5 rounded text-left transition-colors ${
                  tab === 'overview' ? 'bg-blue-600/20 text-blue-200' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                }`}
              >
                <FileText className="w-3.5 h-3.5 text-slate-500" />
                overview.md
              </button>

              {/* validation/ — one page per validation node */}
              <div className="flex items-center gap-1.5 text-slate-300 px-1 py-0.5 pl-5">
                <Folder className="w-3.5 h-3.5 text-amber-400" />
                validation/
                <span className="text-slate-600 ml-auto">{allAnchors.length}</span>
              </div>
              {allAnchors.length === 0 && (
                <div className="pl-10 text-slate-600 italic py-0.5">empty — place anchors</div>
              )}
              {allAnchors.map((anchor, i) => {
                const stage = anchorStage(anchor);
                const isNoise = anchor.anchorType === 'noise';
                return (
                  <button
                    key={anchor.id}
                    onClick={() => openPage('validation', anchor.id)}
                    className={`w-full flex items-center gap-1.5 px-1 py-0.5 pl-10 rounded text-left transition-colors ${
                      page === anchor.id ? 'bg-blue-600/20 text-blue-200' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                    }`}
                    title={`${anchor.id} — ${stage}`}
                  >
                    <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${STAGE_DOT[stage]}`} />
                    {isNoise ? `noise-${String(i + 1).padStart(2, '0')}.val` : `vnode-${String(i + 1).padStart(2, '0')}.val`}
                  </button>
                );
              })}

              <button
                onClick={() => openPage('dataset', 'dataset')}
                className={`w-full flex items-center gap-1.5 px-1 py-0.5 pl-5 rounded text-left transition-colors ${
                  tab === 'dataset' ? 'bg-blue-600/20 text-blue-200' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                }`}
              >
                <Database className="w-3.5 h-3.5 text-emerald-500" />
                dataset/template.json
              </button>

              <button
                onClick={() => openPage('console', 'console')}
                className={`w-full flex items-center gap-1.5 px-1 py-0.5 pl-5 rounded text-left transition-colors ${
                  tab === 'console' ? 'bg-blue-600/20 text-blue-200' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                }`}
              >
                <Terminal className="w-3.5 h-3.5 text-purple-400" />
                console.log
              </button>
            </div>

            {/* Next step hint */}
            <div className="mt-auto p-3 border-t border-slate-800/60">
              <div className="text-[9px] font-mono text-amber-500 tracking-widest mb-1 flex items-center gap-1">
                <ChevronRight className="w-3 h-3" /> NEXT STEP
              </div>
              <p className="text-[10px] text-slate-400 leading-relaxed">{nextHint}</p>
            </div>
          </div>

          {/* ── Content ──────────────────────────────────────────────── */}
          <div className="flex-1 flex flex-col min-w-0">
            {/* Tab bar */}
            <div className="flex border-b border-slate-800 bg-slate-950/40 flex-shrink-0">
              {(['overview', 'validation', 'dataset', 'console'] as WorkspaceTab[]).map(t => (
                <button
                  key={t}
                  onClick={() => openPage(t, t === 'validation' ? (selectedAnchor ? page : null) : t)}
                  className={`px-4 py-2 text-[11px] font-mono tracking-wider uppercase transition-colors border-b-2 ${
                    tab === t
                      ? 'text-cyan-300 border-cyan-400 bg-slate-900/60'
                      : 'text-slate-500 border-transparent hover:text-slate-300'
                  }`}
                >
                  {t}
                  {t === 'validation' && (
                    <span className="ml-1.5 text-[9px] text-slate-500">{placed}/{RING_SIZE}</span>
                  )}
                </button>
              ))}
            </div>

            <div className="flex-1 overflow-y-auto p-4">
              {/* ── OVERVIEW ─────────────────────────────────────────── */}
              {tab === 'overview' && (
                <div className="space-y-4 max-w-2xl">
                  <div className="grid grid-cols-3 gap-3">
                    <div className="p-3 rounded-lg border border-slate-800 bg-slate-900/60">
                      <div className="text-[10px] text-slate-500 uppercase mb-1">Workspace</div>
                      <div className="text-xs font-mono text-blue-300 break-all">{meta.workspaceId}</div>
                    </div>
                    <div className="p-3 rounded-lg border border-slate-800 bg-slate-900/60">
                      <div className="text-[10px] text-slate-500 uppercase mb-1">Difficulty</div>
                      <div className="text-xs font-mono text-red-400">{Math.round(node.difficulty * 100)}%</div>
                    </div>
                    <div className="p-3 rounded-lg border border-slate-800 bg-slate-900/60">
                      <div className="text-[10px] text-slate-500 uppercase mb-1">Bounty</div>
                      <div className="text-xs font-mono text-yellow-400">{node.bounty} NRN</div>
                    </div>
                  </div>

                  {node.isBeingSolved && (
                    <div className="p-3 rounded-lg border border-orange-500/30 bg-orange-500/5">
                      <div className="flex justify-between text-[10px] text-orange-300 font-mono mb-1">
                        <span>RESOLUTION IN PROGRESS</span>
                        <span>{Math.round(node.progress * 100)}%</span>
                      </div>
                      <div className="w-full bg-slate-800 rounded-full h-1.5">
                        <div
                          className="bg-orange-400 h-1.5 rounded-full transition-all"
                          style={{ width: `${node.progress * 100}%` }}
                        />
                      </div>
                    </div>
                  )}

                  {challenge && (
                    <div className="p-3 rounded-lg border border-slate-800 bg-slate-900/60 space-y-2">
                      <div className="text-[11px] font-mono text-cyan-400 tracking-wider">CHALLENGE // {challenge.title}</div>
                      <p className="text-xs text-slate-400 leading-relaxed">{challenge.description}</p>
                      <pre className="text-[10px] font-mono text-rose-300 bg-black/60 border border-slate-800 rounded p-2 overflow-x-auto max-h-40">
                        {challenge.buggyCode}
                      </pre>
                      <p className="text-[10px] text-slate-500 italic">{challenge.context}</p>
                    </div>
                  )}

                  {!node.isBeingSolved && ringStraightened && idleFieldAgent && (
                    <button
                      onClick={() => deployAgent(idleFieldAgent.id, node.id)}
                      className="w-full px-3 py-2.5 rounded-lg bg-cyan-700 hover:bg-cyan-600 text-white text-xs font-bold transition-colors flex items-center justify-center gap-2"
                    >
                      <Play className="w-3.5 h-3.5" />
                      ASSIGN {idleFieldAgent.name.toUpperCase()} (10 NRN)
                    </button>
                  )}
                  {!placed && (
                    <button
                      onClick={() => { setAnalyzing(true); closeDVEWorkspace(); }}
                      className="w-full px-3 py-2.5 rounded-lg bg-red-700 hover:bg-red-600 text-white text-xs font-bold transition-colors"
                    >
                      ENTER ANALYZE MODE — PLACE VALIDATION RING
                    </button>
                  )}
                </div>
              )}

              {/* ── VALIDATION ───────────────────────────────────────── */}
              {tab === 'validation' && (
                selectedAnchor ? (
                  <div className="max-w-2xl">
                    <button
                      onClick={() => openPage('validation', null)}
                      className="text-[10px] font-mono text-slate-500 hover:text-slate-300 mb-3 flex items-center gap-1"
                    >
                      ← all validation nodes
                    </button>
                    <ValidationNodePage anchor={selectedAnchor} index={selectedAnchorIndex} />
                  </div>
                ) : (
                  <div className="space-y-4 max-w-2xl">
                    {/* Ring progress */}
                    <div className="p-3 rounded-lg border border-slate-800 bg-slate-900/60">
                      <div className="flex justify-between text-[10px] font-mono text-slate-400 mb-2">
                        <span>VALIDATION RING</span>
                        <span className={ringCommitted ? 'text-emerald-400' : ringSet ? 'text-blue-400' : 'text-amber-400'}>
                          {placed}/{RING_SIZE} {ringCommitted ? 'COMMITTED' : ringSet ? 'SET' : 'PLACED'}
                        </span>
                      </div>
                      <div className="flex gap-1">
                        {Array.from({ length: RING_SIZE }).map((_, i) => {
                          const a = ringAnchors[i];
                          const cls = !a
                            ? 'bg-slate-800'
                            : STAGE_DOT[anchorStage(a)];
                          return <div key={i} className={`flex-1 h-2 rounded-full ${cls}`} />;
                        })}
                      </div>
                    </div>

                    {/* Validation node pages */}
                    {allAnchors.length === 0 ? (
                      <div className="text-center py-10">
                        <CircleDot className="w-8 h-8 mx-auto mb-2 text-slate-700" />
                        <p className="text-xs text-slate-500">
                          No validation nodes yet. Enter Analyze mode and click the white spikes around this node.
                        </p>
                      </div>
                    ) : (
                      <div className="grid grid-cols-2 gap-2">
                        {allAnchors.map((anchor, i) => {
                          const stage = anchorStage(anchor);
                          const isNoise = anchor.anchorType === 'noise';
                          return (
                            <button
                              key={anchor.id}
                              onClick={() => openPage('validation', anchor.id)}
                              className="p-3 rounded-lg border border-slate-800 bg-slate-900/60 hover:border-slate-600 text-left transition-colors"
                            >
                              <div className="flex items-center justify-between mb-1">
                                <span className="text-[11px] font-mono text-slate-200">
                                  {isNoise ? `noise-${String(i + 1).padStart(2, '0')}` : `vnode-${String(i + 1).padStart(2, '0')}`}
                                </span>
                                <span className={`w-2 h-2 rounded-full ${STAGE_DOT[stage]}`} />
                              </div>
                              <div className="text-[9px] font-mono text-slate-500">
                                w_c {anchor.weights.w_c.toFixed(2)} · w_l {anchor.weights.w_l.toFixed(2)} · w_s {anchor.weights.w_s.toFixed(2)}
                              </div>
                              <div className={`text-[9px] font-mono mt-1 ${STAGE_COLOR[stage].split(' ')[0]}`}>{stage}</div>
                            </button>
                          );
                        })}
                      </div>
                    )}

                    {/* Ring actions */}
                    <div className="flex gap-2">
                      <button
                        onClick={setAllStraightenedAnchors}
                        disabled={!allAnchors.some(a => !a.isSet)}
                        className="flex-1 px-3 py-2 rounded-lg bg-blue-700 hover:bg-blue-600 disabled:bg-slate-800 disabled:text-slate-600 text-white text-xs font-bold transition-colors"
                      >
                        SET ALL
                      </button>
                      <button
                        onClick={commitSetAnchors}
                        disabled={!ringSet || ringCommitted}
                        className="flex-1 px-3 py-2 rounded-lg bg-emerald-700 hover:bg-emerald-600 disabled:bg-slate-800 disabled:text-slate-600 text-white text-xs font-bold transition-colors flex items-center justify-center gap-1.5"
                      >
                        <Shield className="w-3.5 h-3.5" />
                        SCULPT RING
                      </button>
                    </div>
                  </div>
                )
              )}

              {/* ── DATASET ──────────────────────────────────────────── */}
              {tab === 'dataset' && (
                <div className="space-y-3 max-w-2xl font-mono">
                  <div>
                    <div className="text-emerald-400 text-[11px] mb-0.5">● TRL FORMAT — {template.format.toUpperCase()}</div>
                    <div className="text-slate-500 text-[10px]">
                      Trainer: <span className="text-emerald-200">{template.trlTrainer}</span>
                    </div>
                    <p className="text-slate-500 text-[10px] mt-1">{template.description}</p>
                  </div>
                  {!ringStraightened && (
                    <div className="p-2 rounded border border-amber-500/30 bg-amber-500/5 text-amber-300 text-[10px]">
                      Template locked for submission until the validation ring is straightened by agents.
                    </div>
                  )}
                  <pre className="text-[10px] text-emerald-200 bg-black/60 border border-emerald-900/40 rounded p-3 overflow-x-auto whitespace-pre-wrap">
                    {JSON.stringify(template.example, null, 2)}
                  </pre>
                </div>
              )}

              {/* ── CONSOLE ──────────────────────────────────────────── */}
              {tab === 'console' && (
                <div className="font-mono text-[11px] space-y-1">
                  {meta.log.map((line, i) => (
                    <div key={i} className="text-slate-400">
                      <span className="text-emerald-600 mr-1.5">›</span>
                      {line}
                    </div>
                  ))}
                  <div ref={consoleEndRef} />
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
