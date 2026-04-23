import { create } from "zustand";
import { subscribeWithSelector } from "zustand/middleware";
import { Tournament } from "../../../engine/Tournament";
import { Verifier } from "../../../engine/Verifier";
import { LoraxClient } from "../../../networking/LoraxClient";
import { TrainingManager } from "../../../engine/TrainingManager";
import { SabotageEngine, SabotageType } from "../../../engine/Sabotage";
import { getGameLLMService, DEFAULT_PERSONAS, type AgentPersona, type SolutionProposal } from "../../../services/gameLLMService";
import { CHALLENGES, getChallengeForType, type Challenge } from "../../../data/challenges";

// ── Types ──────────────────────────────────────────────────────────────────

export interface DeployAnimation {
  id: string;
  agentId: string;
  agentName: string;
  startPosition: { x: number; y: number; z: number };
  endPosition: { x: number; y: number; z: number };
  startTime: number;
  duration: number;
}

export type GamePhase = "menu" | "playing" | "paused" | "training";

export interface AgentResources {
  compute: number;
  parity: number;
  generation: number;
}

export interface ErrorNode {
  id: string;
  position: { x: number; y: number; z: number };
  type: string;
  difficulty: number;
  bounty: number;
  isBeingSolved: boolean;
  progress: number;
  solverAgent?: string;
  challengeId?: string; // links to challenges.ts dataset
}

export interface SkillNode {
  id: string;
  position: { x: number; y: number; z: number };
  name: string;
  creator: string;
  usageCount: number;
}

export interface IdeaNode {
  id: string;
  position: { x: number; y: number; z: number };
  name: string;
  ideaType: 'asset' | 'characteristic' | 'attribute' | 'innovation' | 'improvement' | 'feature';
  description: string;
  feasibilityScore: number;
  existenceCheck: boolean;
  collaborators: string[];
  stakes: Record<string, number>;
  collaborationValue: number;
  status: 'pending' | 'collaborative' | 'property_created' | 'abandoned';
}

export interface PropertyNode {
  id: string;
  position: { x: number; y: number; z: number };
  name: string;
  propertyType: 'asset' | 'characteristic' | 'attribute' | 'license' | 'patent' | 'trademark';
  sourceIdea: string;
  valueType: string;
  immutable: boolean;
  category: string;
  owners: Record<string, number>;
  marketValue: number;
  usageCount: number;
}

export interface RewardAnchor {
  id: string;
  position: { x: number; y: number; z: number };
  weights: { w_c: number; w_l: number; w_s: number };
  constraints: string;
  linkedErrorNode?: string;
  isSet?: boolean;        // true once the user has filled in data and clicked "Set Anchor"
  isCommitted?: boolean;  // true once Sculpt commits the ring to the arena
  isCommitting?: boolean; // true while the sink+rotate animation is in progress
  isHorizontal?: boolean; // true when placed (horizontal), false after agents straighten it vertical
  anchorType?: 'standard' | 'noise'; // noise anchors are purple, placed via Noise Injection mode
  metadata?: {
    logs: string[];
    traces: string[];
    severity: string;
    description: string;
  };
}

export interface Agent {
  id: string;
  name: string;
  position: { x: number; y: number; z: number };
  target: string | null;
  status: 'idle' | 'moving' | 'working' | 'upgrading';
  type: string;
  staged: boolean;
  efficiency: number;
  accuracy?: number;
  experience: number;
  resources: AgentResources;
  policy: 'greedy' | 'bayesian' | 'stochastic';
  personaId: string; // links to AgentPersona
  stats?: {
    errorsResolved: number;
    skillsLearned: number;
    ideasDeveloped: number;
    propertiesCreated: number;
    sabotageApplied: number;
    trainingTime: number;
  };
  proposeSolution: (challenge: Challenge, persona: AgentPersona) => Promise<SolutionProposal>;
}

export interface EpochResult {
  epochNumber: number;
  challenge: Challenge;
  agentResults: {
    agentId: string;
    agentName: string;
    personaName: string;
    proposal: SolutionProposal;
    score: number;
    reasoning?: string;
  }[];
  winnerId: string | null;
  winnerScore: number;
  timestamp: number;
}

export interface KnirvanaState {
  // Game state
  gamePhase: GamePhase;
  gameTime: number;
  epochNumber: number;
  usingMockLLM: boolean;

  // Resources
  nrnBalance: number;
  skillsLearned: number;
  errorsResolved: number;
  ideasDeveloped: number;
  propertiesCreated: number;

  // Game objects
  errorNodes: ErrorNode[];
  skillNodes: SkillNode[];
  ideaNodes: IdeaNode[];
  propertyNodes: PropertyNode[];
  agents: Agent[];
  rewardAnchors: RewardAnchor[];

  // Agent personas
  personas: AgentPersona[];

  // Epoch results
  lastEpochResult: EpochResult | null;
  epochHistory: EpochResult[];

  // Deploy animations
  deployAnimations: DeployAnimation[];

  // Selection
  selectedErrorNode: string | null;
  selectedIdeaNode: string | null;
  selectedPropertyNode: string | null;
  selectedAgent: string | null;
  selectedRewardAnchor: string | null;
  showAnchorConfigModal: boolean;

  // Analyze / sculpt modes
  isAnalyzing: boolean;
  isSculpting: boolean;
  isNoiseInjecting: boolean;

  // Anchor straightening sequence
  isStraighteningAnchors: boolean;

  // Tournament specific
  skillSlotOwner: string | null;
  incumbentScore: number;

  // Engine instances
  tournament: Tournament;
  verifier: Verifier;
  loraxClient: LoraxClient;
  trainingManager: TrainingManager;

  // Actions
  startGame: () => void;
  pauseGame: () => void;
  updateGameTime: (delta: number) => void;

  selectErrorNode: (id: string | null) => void;
  selectAgent: (id: string | null) => void;

  setAnalyzing: (analyzing: boolean) => void;
  setSculpting: (sculpting: boolean) => void;
  setNoiseInjecting: (v: boolean) => void;

  addRewardAnchor: (anchor: RewardAnchor) => void;
  selectRewardAnchor: (id: string | null) => void;
  setShowAnchorConfigModal: (show: boolean) => void;
  updateRewardAnchor: (id: string, updates: Partial<RewardAnchor>) => void;
  removeRewardAnchor: (id: string) => void;
  commitSetAnchors: () => void;
  setAllStraightenedAnchors: () => void;
  completeStraightening: () => void;
  startStraighteningSequence: () => void;

  createAgent: (type: string) => void;
  deployAgent: (agentId: string, nodeId: string) => void;
  deployAllStaged: () => void;
  deployOne: (agentId: string) => void;
  stageAgent: (agentId: string) => void;
  moveAgent: (agentId: string, position: { x: number; y: number; z: number }) => void;

  addNRN: (amount: number) => void;
  spendNRN: (amount: number) => boolean;

  runEpoch: () => Promise<void>;

  applySabotage: (targetAgentId: string, type: SabotageType, magnitude: number) => void;

  startTraining: (agentId: string) => void;
  distillTrajectory: (agentId: string) => void;
  hardenAgent: (agentId: string) => void;

  startDeployAnimation: (agentId: string, agentName: string, endPosition: { x: number; y: number; z: number }) => void;
  updateDeployAnimations: () => void;

  updateVerifierWeights: (weights: { correctness: number; latency: number; simplicity: number }) => void;
  addVerifierConstraint: (id: string, validator: (res: unknown) => boolean) => void;

  // Persona management
  updatePersona: (id: string, updates: Partial<AgentPersona>) => void;
  addPersona: (persona: AgentPersona) => void;
  removePersona: (id: string) => void;

  // Persistence
  saveProgress: () => void;
  loadProgress: () => void;

  // UI State
  showAgentManagementModal: boolean;
  setShowAgentManagementModal: (show: boolean) => void;
  requestAgentManagementOpen: boolean;
  clearRequestAgentManagementOpen: () => void;
}

// ── Persistence helpers ───────────────────────────────────────────────────

const STORAGE_KEY = 'knirvana_progress_v1';

interface PersistedState {
  nrnBalance: number;
  errorsResolved: number;
  skillsLearned: number;
  epochNumber: number;
  personas: AgentPersona[];
  epochHistory: EpochResult[];
  incumbentScore: number;
  skillSlotOwner: string | null;
}

function saveToStorage(state: Partial<PersistedState>) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch (e) {
    console.warn('[Knirvana] Could not persist state:', e);
  }
}

function loadFromStorage(): Partial<PersistedState> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return {};
    return JSON.parse(raw) as Partial<PersistedState>;
  } catch {
    return {};
  }
}

// ── Initial data generators ───────────────────────────────────────────────

const generateErrorNodes = (): ErrorNode[] => {
  const types = ['Memory Leak', 'Logic Error', 'Race Condition', 'Buffer Overflow', 'API Timeout'] as const;
  const nodes: ErrorNode[] = [];

  for (let i = 0; i < 15; i++) {
    const type = types[i % types.length];
    const challenge = getChallengeForType(type);
    nodes.push({
      id: `error-${i}`,
      position: {
        x: (Math.random() - 0.5) * 40,
        y: 0,
        z: (Math.random() - 0.5) * 40,
      },
      type,
      difficulty: challenge.difficulty,
      bounty: challenge.bounty,
      isBeingSolved: false,
      progress: 0,
      challengeId: challenge.id,
    });
  }

  return nodes;
};

const generateSkillNodes = (): SkillNode[] => {
  const skills = ['Memory Optimization', 'Logic Verification', 'Concurrency Control', 'Security Hardening', 'Performance Tuning'];
  const nodes: SkillNode[] = [];

  for (let i = 0; i < 8; i++) {
    nodes.push({
      id: `skill-${i}`,
      position: {
        x: (Math.random() - 0.5) * 35,
        y: 0,
        z: (Math.random() - 0.5) * 35,
      },
      name: skills[Math.floor(Math.random() * skills.length)],
      creator: `Agent-${Math.floor(Math.random() * 100)}`,
      usageCount: Math.floor(Math.random() * 50),
    });
  }

  return nodes;
};

// ── Staging positions ─────────────────────────────────────────────────────

const STAGING_Z = -22;
const STAGING_SPACING = 5;

export function getStagingPosition(index: number): { x: number; y: number; z: number } {
  return { x: (index - 1) * STAGING_SPACING, y: -2, z: STAGING_Z };
}

/** Build a proposeSolution function that uses the GameLLMService */
function makeLLMProposer(): Agent['proposeSolution'] {
  return async (challenge: Challenge, persona: AgentPersona): Promise<SolutionProposal> => {
    const llm = getGameLLMService();
    return llm.propose(challenge, persona);
  };
}

const generateInitialAgents = (personas: AgentPersona[]): Agent[] => {
  const types = ['Analyzer', 'Optimizer', 'Solver'];
  const policies: Array<'greedy' | 'bayesian' | 'stochastic'> = ['greedy', 'bayesian', 'stochastic'];

  return personas.slice(0, 3).map((persona, i) => ({
    id: `agent-${i}`,
    name: `${types[i]} (${persona.name})`,
    position: getStagingPosition(i),
    target: null,
    status: 'idle' as const,
    type: types[i],
    staged: true,
    efficiency: 0.6 + Math.random() * 0.4,
    experience: Math.floor(Math.random() * 100),
    resources: { compute: 100, parity: 100, generation: 1 },
    policy: policies[i % policies.length],
    personaId: persona.id,
    proposeSolution: makeLLMProposer(),
  }));
};

// ── Engine initialization ─────────────────────────────────────────────────

const verifier = new Verifier();
const loraxClient = new LoraxClient('http://localhost:8080');
const tournament = new Tournament(verifier, loraxClient);
const trainingManager = new TrainingManager();

// Wire LLM service into verifier immediately
const llmSvc = getGameLLMService();
verifier.setLLMService(llmSvc);

// Load persisted state
const persisted = loadFromStorage();
const initialPersonas: AgentPersona[] = persisted.personas?.length
  ? persisted.personas
  : DEFAULT_PERSONAS;

// ── Store ─────────────────────────────────────────────────────────────────

export const useKnirvana = create<KnirvanaState>()(
  subscribeWithSelector((set, get) => ({
    // Initial state
    gamePhase: "menu",
    gameTime: 0,
    epochNumber: persisted.epochNumber ?? 0,
    usingMockLLM: llmSvc.isUsingMock(),

    nrnBalance: persisted.nrnBalance ?? 500,
    skillsLearned: persisted.skillsLearned ?? 0,
    errorsResolved: persisted.errorsResolved ?? 0,
    ideasDeveloped: 0,
    propertiesCreated: 0,

    errorNodes: generateErrorNodes(),
    skillNodes: generateSkillNodes(),
    ideaNodes: [],
    propertyNodes: [],
    agents: generateInitialAgents(initialPersonas),
    rewardAnchors: [],
    deployAnimations: [],

    personas: initialPersonas,

    lastEpochResult: null,
    epochHistory: persisted.epochHistory ?? [],

    selectedErrorNode: null,
    selectedIdeaNode: null,
    selectedPropertyNode: null,
    selectedAgent: null,
    selectedRewardAnchor: null,
    showAnchorConfigModal: false,

    isAnalyzing: false,
    isSculpting: false,
    isNoiseInjecting: false,
    isStraighteningAnchors: false,

    skillSlotOwner: persisted.skillSlotOwner ?? null,
    incumbentScore: persisted.incumbentScore ?? 0.8,

    tournament,
    verifier,
    loraxClient,
    trainingManager,

    // ── Game control ──────────────────────────────────────────────────────

    startGame: () => {
      set({ gamePhase: "playing" });
    },

    pauseGame: () => {
      set({ gamePhase: "paused" });
      get().saveProgress();
    },

    updateGameTime: (delta) => {
      set((state) => ({ gameTime: state.gameTime + delta }));
      get().updateDeployAnimations();

      const state = get();
      const updatedErrorNodes = state.errorNodes.map(node => {
        if (node.isBeingSolved && node.progress < 1) {
          const agent = state.agents.find(a => a.id === node.solverAgent);
          const progressRate = agent ? agent.efficiency * 0.1 : 0.05;
          const newProgress = Math.min(1, node.progress + progressRate * delta);

          if (newProgress >= 1 && node.progress < 1) {
            get().addNRN(node.bounty);
            set((s) => ({ errorsResolved: s.errorsResolved + 1 }));
          }

          return { ...node, progress: newProgress };
        }
        return node;
      });

      set({ errorNodes: updatedErrorNodes });
    },

    // ── Selection ─────────────────────────────────────────────────────────

    selectErrorNode: (id) => set({ selectedErrorNode: id }),
    selectAgent: (id) => set({ selectedAgent: id }),

    // ── Analyze / sculpt ──────────────────────────────────────────────────

    setAnalyzing: (analyzing) => {
      set({ isAnalyzing: analyzing });
      if (analyzing) {
        set({ isSculpting: true });
      } else {
        set({ isSculpting: false });
      }
    },

    setSculpting: (sculpting) => set({ isSculpting: sculpting }),

    setNoiseInjecting: (v) => set({ isNoiseInjecting: v }),

    // ── Reward anchors ────────────────────────────────────────────────────

    addRewardAnchor: (anchor) => {
      set(state => ({
        rewardAnchors: [...state.rewardAnchors, { ...anchor, isHorizontal: true }],
      }));
    },

    selectRewardAnchor: (id) => set({ selectedRewardAnchor: id, showAnchorConfigModal: false }),
    setShowAnchorConfigModal: (show) => set({ showAnchorConfigModal: show }),

    updateRewardAnchor: (id, updates) => {
      set(state => ({
        rewardAnchors: state.rewardAnchors.map(anchor =>
          anchor.id === id ? { ...anchor, ...updates } : anchor
        ),
      }));
    },

    removeRewardAnchor: (id) => {
      set(state => ({
        rewardAnchors: state.rewardAnchors.filter(anchor => anchor.id !== id),
        selectedRewardAnchor: state.selectedRewardAnchor === id ? null : state.selectedRewardAnchor,
      }));
    },

    setAllStraightenedAnchors: () => {
      set(state => ({
        rewardAnchors: state.rewardAnchors.map(anchor =>
          !anchor.isHorizontal && !anchor.isSet ? { ...anchor, isSet: true } : anchor
        ),
      }));
    },

    commitSetAnchors: () => {
      set(state => ({
        rewardAnchors: state.rewardAnchors.map(anchor =>
          anchor.isSet && !anchor.isCommitted
            ? { ...anchor, isCommitted: true, isCommitting: true }
            : anchor
        ),
        // Straightening is triggered by Deploy, not here
      }));
    },

    startStraighteningSequence: () => {
      console.log('Starting straightening sequence');
      set({ isStraighteningAnchors: true });
    },

    completeStraightening: () => {
      set(state => ({
        isStraighteningAnchors: false,
        rewardAnchors: state.rewardAnchors.map(anchor =>
          anchor.isHorizontal ? { ...anchor, isHorizontal: false } : anchor
        ),
      }));
    },

    // ── Deploy animations ─────────────────────────────────────────────────

    startDeployAnimation: (agentId, agentName, endPosition) => {
      const animation: DeployAnimation = {
        id: `deploy-${Date.now()}`,
        agentId,
        agentName,
        startPosition: { x: 0, y: 20, z: 0 },
        endPosition,
        startTime: Date.now(),
        duration: 1500,
      };
      set(state => ({ deployAnimations: [...state.deployAnimations, animation] }));
    },

    updateDeployAnimations: () => {
      const now = Date.now();
      set(state => {
        const active = state.deployAnimations.filter(a => now - a.startTime < a.duration);
        const completed = state.deployAnimations.filter(a => now - a.startTime >= a.duration);

        if (completed.length === 0) return { deployAnimations: active };

        const newAgents = [...state.agents];
        const personas = state.personas;

        completed.forEach(anim => {
          const persona = personas[newAgents.length % personas.length] ?? DEFAULT_PERSONAS[0];
          const newAgent: Agent = {
            id: anim.agentId,
            name: anim.agentName,
            position: anim.endPosition,
            target: null,
            status: 'idle',
            type: 'deployed',
            staged: false,
            policy: 'greedy',
            efficiency: Math.random() * 0.5 + 0.5,
            accuracy: Math.random() * 0.3 + 0.7,
            experience: 0,
            resources: { compute: 100, parity: 100, generation: 1 },
            personaId: persona.id,
            stats: {
              errorsResolved: 0,
              skillsLearned: 0,
              ideasDeveloped: 0,
              propertiesCreated: 0,
              sabotageApplied: 0,
              trainingTime: 0,
            },
            proposeSolution: makeLLMProposer(),
          };
          newAgents.push(newAgent);
        });

        return { deployAnimations: active, agents: newAgents };
      });
    },

    // ── Agent management ──────────────────────────────────────────────────

    createAgent: (type) => {
      const cost = 50;
      if (get().spendNRN(cost)) {
        const state = get();
        const personas = state.personas;
        const persona = personas[state.agents.length % personas.length] ?? DEFAULT_PERSONAS[0];
        const policies: Array<'greedy' | 'bayesian' | 'stochastic'> = ['greedy', 'bayesian', 'stochastic'];

        const stagedCount = state.agents.filter(a => a.staged).length;
        const newAgent: Agent = {
          id: `agent-${Date.now()}`,
          name: `${type} (${persona.name})`,
          position: getStagingPosition(stagedCount),
          target: null,
          status: 'idle',
          type,
          staged: true,
          efficiency: 0.5 + Math.random() * 0.3,
          experience: 0,
          resources: { compute: 100, parity: 100, generation: 1 },
          policy: policies[Math.floor(Math.random() * policies.length)],
          personaId: persona.id,
          proposeSolution: makeLLMProposer(),
        };

        set((s) => ({ agents: [...s.agents, newAgent] }));
      }
    },

    deployAgent: (agentId, nodeId) => {
      const cost = 10;
      if (get().spendNRN(cost)) {
        set((state) => ({
          agents: state.agents.map(agent =>
            agent.id === agentId
              ? { ...agent, target: nodeId, status: 'working' as const }
              : agent
          ),
          errorNodes: state.errorNodes.map(node =>
            node.id === nodeId
              ? { ...node, isBeingSolved: true, solverAgent: agentId }
              : node
          ),
        }));
      }
    },

    moveAgent: (agentId, position) => {
      set((state) => ({
        agents: state.agents.map(agent =>
          agent.id === agentId
            ? { ...agent, position, status: 'moving' as const }
            : agent
        ),
      }));
    },

    deployAllStaged: () => {
      set((state) => ({
        agents: state.agents.map(agent =>
          agent.staged
            ? {
                ...agent,
                staged: false,
                position: {
                  x: (Math.random() - 0.5) * 10,
                  y: 1,
                  z: (Math.random() - 0.5) * 10,
                },
              }
            : agent
        ),
      }));
    },

    deployOne: (agentId) => {
      set((state) => ({
        agents: state.agents.map(agent =>
          agent.id === agentId && agent.staged
            ? {
                ...agent,
                staged: false,
                position: {
                  x: (Math.random() - 0.5) * 10,
                  y: 1,
                  z: (Math.random() - 0.5) * 10,
                },
              }
            : agent
        ),
      }));
    },

    stageAgent: (agentId) => {
      const state = get();
      const agent = state.agents.find(a => a.id === agentId);
      if (!agent || agent.staged) return;

      const stagedCount = state.agents.filter(a => a.staged).length;
      const stagingPos = getStagingPosition(stagedCount);
      const startPos = { ...agent.position };
      const startTime = Date.now();
      const duration = 1800;

      // Mark as moving immediately
      set(s => ({
        agents: s.agents.map(a =>
          a.id === agentId ? { ...a, status: 'moving' as const } : a
        ),
      }));

      const interval = setInterval(() => {
        const elapsed = Date.now() - startTime;
        const t = Math.min(elapsed / duration, 1);
        const eased = 1 - Math.pow(1 - t, 2);

        if (t >= 1) {
          clearInterval(interval);
          set(s => ({
            agents: s.agents.map(a =>
              a.id === agentId
                ? { ...a, staged: true, status: 'idle' as const, target: null, position: stagingPos }
                : a
            ),
          }));
        } else {
          set(s => ({
            agents: s.agents.map(a =>
              a.id === agentId
                ? {
                    ...a,
                    position: {
                      x: startPos.x + (stagingPos.x - startPos.x) * eased,
                      y: 1,
                      z: startPos.z + (stagingPos.z - startPos.z) * eased,
                    },
                  }
                : a
            ),
          }));
        }
      }, 50);
    },

    // ── NRN economy ───────────────────────────────────────────────────────

    addNRN: (amount) => {
      set((state) => ({ nrnBalance: state.nrnBalance + amount }));
    },

    spendNRN: (amount) => {
      const state = get();
      if (state.nrnBalance >= amount) {
        set({ nrnBalance: state.nrnBalance - amount });
        return true;
      }
      return false;
    },

    // ── Tournament epoch ──────────────────────────────────────────────────

    runEpoch: async () => {
      const state = get();
      if (state.gamePhase !== 'playing') return;

      const epochNumber = state.epochNumber + 1;
      set({ epochNumber });

      // Pick the active error node that has a challenge
      const activeErrorNode = state.errorNodes.find(node => node.isBeingSolved && node.challengeId)
        ?? state.errorNodes.find(node => node.challengeId);

      if (!activeErrorNode) {
        console.warn('[Tournament] No error nodes with challenges available');
        return;
      }

      const challenge = CHALLENGES.find(c => c.id === activeErrorNode.challengeId)
        ?? CHALLENGES[epochNumber % CHALLENGES.length];

      const llm = getGameLLMService();
      const personas = state.personas;

      // Run all agents in parallel
      const agentResults = await Promise.all(
        state.agents.map(async (agent) => {
          const persona = personas.find(p => p.id === agent.personaId) ?? DEFAULT_PERSONAS[0];

          const proposal = await agent.proposeSolution(challenge, persona);
          const evalResult = await llm.evaluate(challenge, proposal, persona);

          return {
            agentId: agent.id,
            agentName: agent.name,
            personaId: persona.id,
            personaName: persona.name,
            proposal,
            score: evalResult.score,
            reasoning: evalResult.reasoning,
          };
        })
      );

      // Rank by score
      const ranked = [...agentResults].sort((a, b) => b.score - a.score);
      const winner = ranked[0];

      const epochResult: EpochResult = {
        epochNumber,
        challenge,
        agentResults,
        winnerId: winner?.agentId ?? null,
        winnerScore: winner?.score ?? 0,
        timestamp: Date.now(),
      };

      // Update skill slot if winner beats incumbent
      let newSkillSlotOwner = state.skillSlotOwner;
      let newIncumbentScore = state.incumbentScore;

      if (winner && winner.score > state.incumbentScore) {
        newSkillSlotOwner = winner.agentId;
        newIncumbentScore = winner.score;
        // Award extra NRN for hijacking the skill slot
        get().addNRN(Math.round(challenge.bounty * 1.5));
      }

      // Update persona win rates
      const updatedPersonas = personas.map(persona => {
        const result = agentResults.find(r => r.personaId === persona.id);
        if (!result) return persona;
        const isWinner = result.agentId === winner?.agentId;
        const newTotal = persona.totalEpochs + 1;
        const newWins = persona.wins + (isWinner ? 1 : 0);
        return {
          ...persona,
          totalEpochs: newTotal,
          wins: newWins,
          winRate: newWins / newTotal,
        };
      });

      set({
        lastEpochResult: epochResult,
        epochHistory: [...state.epochHistory.slice(-49), epochResult], // keep last 50
        skillSlotOwner: newSkillSlotOwner,
        incumbentScore: newIncumbentScore,
        personas: updatedPersonas,
      });

      // Auto-save after each epoch
      get().saveProgress();
    },

    // ── Sabotage ──────────────────────────────────────────────────────────

    applySabotage: (targetAgentId, type, magnitude) => {
      const state = get();
      const targetAgent = state.agents.find(a => a.id === targetAgentId);

      if (targetAgent) {
        const cost = magnitude * 10;
        if (state.spendNRN(cost)) {
          SabotageEngine.applyEffect(type, targetAgent, magnitude);
          set((s) => ({
            agents: s.agents.map(a =>
              a.id === targetAgentId ? { ...targetAgent } : a
            ),
          }));
        }
      }
    },

    // ── Training ──────────────────────────────────────────────────────────

    startTraining: (agentId) => {
      set({ gamePhase: 'training' });
      console.log(`[Training] Starting for agent ${agentId}`);
    },

    distillTrajectory: (agentId) => {
      console.log(`[Training] Distilling trajectory for agent ${agentId}`);
    },

    hardenAgent: (agentId) => {
      const state = get();
      const agent = state.agents.find(a => a.id === agentId);
      if (agent) {
        state.trainingManager.harden(agent);
        set((s) => ({
          agents: s.agents.map(a => a.id === agentId ? { ...agent } : a),
        }));
      }
    },

    // ── Verifier controls ─────────────────────────────────────────────────

    updateVerifierWeights: (weights) => {
      const state = get();
      state.verifier.updateWeights(weights);
    },

    addVerifierConstraint: (id, validator) => {
      const state = get();
      state.verifier.addConstraint(id, validator);
    },

    // ── Persona management ────────────────────────────────────────────────

    updatePersona: (id, updates) => {
      set(state => ({
        personas: state.personas.map(p => p.id === id ? { ...p, ...updates } : p),
      }));
      get().saveProgress();
    },

    addPersona: (persona) => {
      set(state => ({ personas: [...state.personas, persona] }));
      get().saveProgress();
    },

    removePersona: (id) => {
      set(state => ({
        personas: state.personas.filter(p => p.id !== id),
      }));
      get().saveProgress();
    },

    // ── Persistence ───────────────────────────────────────────────────────

    saveProgress: () => {
      const state = get();
      saveToStorage({
        nrnBalance: state.nrnBalance,
        errorsResolved: state.errorsResolved,
        skillsLearned: state.skillsLearned,
        epochNumber: state.epochNumber,
        personas: state.personas,
        epochHistory: state.epochHistory.slice(-20), // save last 20 epochs
        incumbentScore: state.incumbentScore,
        skillSlotOwner: state.skillSlotOwner,
      });
    },

    loadProgress: () => {
      const saved = loadFromStorage();
      if (Object.keys(saved).length > 0) {
        set({
          nrnBalance: saved.nrnBalance ?? 500,
          errorsResolved: saved.errorsResolved ?? 0,
          skillsLearned: saved.skillsLearned ?? 0,
          epochNumber: saved.epochNumber ?? 0,
          personas: saved.personas?.length ? saved.personas : DEFAULT_PERSONAS,
          epochHistory: saved.epochHistory ?? [],
          incumbentScore: saved.incumbentScore ?? 0.8,
          skillSlotOwner: saved.skillSlotOwner ?? null,
        });
      }
    },

    // ── UI State ────────────────────────────────────────────────────────────

    showAgentManagementModal: false,
    setShowAgentManagementModal: (show) => set({ showAgentManagementModal: show }),
    requestAgentManagementOpen: false,
    clearRequestAgentManagementOpen: () => set({ requestAgentManagementOpen: false }),
  }))
);
