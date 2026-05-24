import React from 'react';
import { useState, useEffect, useRef, Suspense, lazy } from 'react';
import { BrowserRouter as Router, Routes, Route, useNavigate, useParams } from 'react-router-dom';
import { QrCode, X } from 'lucide-react';
import { initSentry } from './utils/sentry';

// Receiver components
import { KnirvShell } from './components/KnirvShell';
import { GameMenu } from './components/game/GameMenu';

import { NetworkStatus } from './components/NetworkStatus';
import { NRVVisualization } from './components/NRVVisualization';
import { SlidingPanel } from './components/SlidingPanel';
import { EdgeColoring } from './components/EdgeColoring';
import { FabricAlgorithm } from './components/FabricAlgorithm';
import { CognitiveShellInterface } from './components/CognitiveShellInterface';
import { ChatBrainProvider } from './contexts/ChatBrainContext';
import { CortexBuilder } from './components/CortexBuilder';
import { ApiKeyManager } from './components/ApiKeyManager';
import USDCToNRNPurchase from './components/USDCToNRNPurchase';
import { SlideDownModal } from './components/SlideDownModal';
import { SkillsModalContent } from './components/modals/SkillsModalContent';
import { UDCModalContent } from './components/modals/UDCModalContent';
import { WalletModalContent } from './components/modals/WalletModalContent';
import { AgentManagementModal } from './components/modals/AgentManagementModal';
import { ErrorNodeModal } from './components/modals/ErrorNodeModal';
import { TraverseClustersModal } from './components/modals/TraverseClustersModal';
interface CognitiveState {
  activeSkills: string[];
  confidenceLevel: number;
}
import NetworkSelector, { NetworkType } from './components/NetworkSelector';

// Manager components - lazy loaded
const Skills = lazy(() => import('./pages/Skills'));
const UDC = lazy(() => import('./pages/UDC'));
const WalletPage = lazy(() => import('./pages/Wallet'));
const ChatBrain = lazy(() => import('./pages/ChatBrain'));


// Types
import { Agent } from './types/common';
import { useKnirvana } from './components/game/stores/useKnirvana';







// Window interface for global modal functions
declare global {
  interface Window {
    openErrorNodeModal?: () => void;
  }
}

// Sub-Agent interface for new Agent Management modal
export interface SubAgent {
  id: string;
  name: string;
  status: 'idle' | 'training' | 'ready' | 'deployed';
  capabilities: string[];
  expertise: string[];
  trainingProgress: number;
  nrnCost: number;
  description: string;
}

// Types from receiver
export interface Adaptation {
  id: string;
  type: string;
  description: string;
  parameters: Record<string, unknown>;
  timestamp: Date;
  confidence: number;
}

export interface SkillResult {
  success: boolean;
  data?: unknown;
  output?: unknown;
  error?: string;
  executionTime?: number;
}

export interface NRV {
  id: string;
  problemDescription: string;
  sourceID: string;
  inputType: 'Voice' | 'Screenshot' | 'Log' | 'Camera' | 'Error' | 'Context' | 'Idea';
  visualContext?: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  temporalContext: Date;
  severity: 'Low' | 'Medium' | 'High' | 'Critical';
  suggestedSolutionType: string;
  status: 'Identified' | 'Mapped' | 'Assigned' | 'Resolved';
}

// Note: convertAgentToLegacy function removed - not currently needed but can be added back if LegacyAgent compatibility is required


// Burger Menu Component
interface BurgerMenuProps {
  isOpen: boolean;
  onToggle: () => void;
  children: React.ReactNode;
  showNetworkSelector?: boolean;
  setShowNetworkSelector?: (show: boolean) => void;
  currentNetwork?: NetworkType | null;
  setCurrentNetwork?: (network: NetworkType | null) => void;
}

const BurgerMenu: React.FC<BurgerMenuProps> = ({ isOpen, onToggle, children, showNetworkSelector, setShowNetworkSelector, currentNetwork, setCurrentNetwork }) => {
  return (
    <div className="relative">
      {/* Burger Button */}
      <button
        onClick={onToggle}
        className="bg-gray-800/80 hover:bg-gray-700/80 text-white p-3 rounded-lg shadow-lg transition-all duration-200 border border-gray-600/50 backdrop-blur-sm"
        aria-label="Navigation menu"
        data-testid="burger-menu"
      >
        <div className="w-5 h-5 flex flex-col justify-center items-center">
          <div className={`w-5 h-0.5 bg-white transition-all duration-300 ${isOpen ? 'rotate-45 translate-y-1' : ''}`}></div>
          <div className={`w-5 h-0.5 bg-white transition-all duration-300 mt-1 ${isOpen ? 'opacity-0' : ''}`}></div>
          <div className={`w-5 h-0.5 bg-white transition-all duration-300 mt-1 ${isOpen ? '-rotate-45 -translate-y-1' : ''}`}></div>
        </div>
      </button>

      {/* Menu Dropdown */}
      {isOpen && (
        <div className="absolute top-full right-0 mt-2 bg-gray-800/90 backdrop-blur-sm border border-gray-600/50 rounded-lg shadow-xl min-w-48 z-50" data-testid="burger-menu-content">
          <div className="p-2 space-y-1">
            {children}
          </div>
        </div>
      )}

      {/* Network Selector Modal */}
      {showNetworkSelector && (
        <NetworkSelector
          isOpen={showNetworkSelector}
          onClose={() => setShowNetworkSelector?.(false)}
          onNetworkChange={(network) => {
            setCurrentNetwork?.(network);
            console.log('Network changed to:', network.name);
          }}
          currentNetwork={currentNetwork ?? undefined}
        />
      )}
    </div>
  );
};

// Menu Item Component
interface MenuItemProps {
  onClick: () => void;
  children: React.ReactNode;
  icon: string;
  className?: string;
}

const MenuItem: React.FC<MenuItemProps> = ({ onClick, children, icon, className = '' }) => {
  return (
    <button
      onClick={onClick}
      className={`w-full text-left px-3 py-2 rounded-md text-white hover:bg-gray-700/80 transition-all duration-200 flex items-center space-x-2 ${className}`}
    >
      {icon && <span className="text-lg">{icon}</span>}
      <span className="font-medium">{children}</span>
    </button>
  );
};

// Receiver Interface Component
const ReceiverInterface = () => {
  const gameStore = useKnirvana();
  const nrnBalance = gameStore.nrnBalance;
  const gamePhase = gameStore.gamePhase;
  const [shellStatus, setShellStatus] = useState<'idle' | 'processing' | 'listening' | 'error'>('idle');
  const [isVoiceActive, setIsVoiceActive] = useState(false);
  const [hasSeenIntro, setHasSeenIntro] = useState(false);
  const [currentNRVs, setCurrentNRVs] = useState<NRV[]>([]);
  const [selectedNRV, setSelectedNRV] = useState<NRV | null>(null);
  const [availableAgents, setAvailableAgents] = useState<Agent[]>([]);
  const [activePanels, setActivePanels] = useState<string[]>([]);
  const [cognitiveMode, setCognitiveMode] = useState(false);
  const [isCortexBuilderOpen, setIsCortexBuilderOpen] = useState(false);
  const [isApiKeyManagerOpen, setIsApiKeyManagerOpen] = useState(false);
  const [isUSDCPurchaseOpen, setIsUSDCPurchaseOpen] = useState(false);
  const [activeModal, setActiveModal] = useState<'skills' | 'udc' | 'wallet' | null>(null);
  const [isAgentManagementOpen, setIsAgentManagementOpen] = useState(false);
  const [isErrorNodeOpen, setIsErrorNodeOpen] = useState(false);
  const [isTraverseClustersOpen, setIsTraverseClustersOpen] = useState(false);
  const [networkConnections] = useState<{
    [key: string]: 'connected' | 'disconnected' | 'connecting';
  }>({
    knirvChain: 'connected',
    knirvGraph: 'connected',
    knirvWallet: 'connected',
    knirvRouters: 'connected',
    knirvana: 'connected',
    knirvNexus: 'connected'
  });

  const shellRef = useRef<HTMLDivElement>(null);

  // Mock sub-agent data
  const [mockSubAgents] = useState<SubAgent[]>([
    {
      id: 'sub-agent-1',
      name: 'React Debugger',
      status: 'ready',
      capabilities: ['Debugging', 'React', 'Error Detection'],
      expertise: ['React Components', 'State Management', 'Hooks'],
      trainingProgress: 100,
      nrnCost: 75,
      description: 'Specialized in debugging React applications and component issues'
    },
    {
      id: 'sub-agent-2',
      name: 'API Integration Expert',
      status: 'training',
      capabilities: ['API Development', 'REST', 'GraphQL'],
      expertise: ['Microservices', 'API Design', 'Data Integration'],
      trainingProgress: 65,
      nrnCost: 90,
      description: 'Expert in API design, integration patterns and microservices architecture'
    },
    {
      id: 'sub-agent-3',
      name: 'Security Analyst',
      status: 'idle',
      capabilities: ['Security', 'Vulnerability Assessment', 'Code Review'],
      expertise: ['OWASP', 'Security Testing', 'Penetration Testing'],
      trainingProgress: 0,
      nrnCost: 120,
      description: 'Security-focused agent for vulnerability detection and security best practices'
    },
    {
      id: 'sub-agent-4',
      name: 'Performance Optimizer',
      status: 'deployed',
      capabilities: ['Performance', 'Optimization', 'Profiling'],
      expertise: ['Performance Tuning', 'Memory Management', 'CPU Optimization'],
      trainingProgress: 100,
      nrnCost: 85,
      description: 'Specializes in application performance optimization and resource management'
    }
  ]);

  // Sync shell status based on game phase
  useEffect(() => {
    if (gameStore.gamePhase === 'playing') {
      setShellStatus('idle');
    }
  }, [gameStore.gamePhase, setShellStatus]);

  // Sync agent management modal state with store
  useEffect(() => {
    if (gameStore.showAgentManagementModal && !isAgentManagementOpen) {
      setIsAgentManagementOpen(true);
      gameStore.setShowAgentManagementModal(false);
    }
  }, [gameStore.showAgentManagementModal, isAgentManagementOpen]);

  useEffect(() => {
    // Initialize mock agents using the new Agent interface
    const mockAgents: Agent[] = [
      {
        agentId: 'agent-1',
        name: 'System Diagnostics Agent',
        version: '1.0.0',
        type: 'wasm',
        status: 'Available',
        nrnCost: 50,
        capabilities: ['error-detection', 'system-analysis'],
        metadata: {
          name: 'System Diagnostics Agent',
          version: '1.0.0',
          description: 'Advanced system diagnostics and error detection',
          author: 'KNIRV Network',
          capabilities: ['error-detection', 'system-analysis'],
          requirements: { memory: 256, cpu: 1, storage: 50 },
          permissions: ['read', 'analyze']
        },
        createdAt: new Date().toISOString()
      },
      {
        agentId: 'agent-2',
        name: 'UI/UX Optimization Agent',
        version: '1.0.0',
        type: 'lora',
        status: 'Available',
        nrnCost: 75,
        capabilities: ['interface-design', 'user-experience'],
        metadata: {
          name: 'UI/UX Optimization Agent',
          version: '1.0.0',
          description: 'User interface and experience optimization specialist',
          author: 'KNIRV Network',
          capabilities: ['interface-design', 'user-experience'],
          requirements: { memory: 512, cpu: 2, storage: 100 },
          permissions: ['read', 'write', 'design']
        },
        createdAt: new Date().toISOString()
      },
      {
        agentId: 'agent-3',
        name: 'Network Security Agent',
        version: '1.0.0',
        type: 'hybrid',
        status: 'Deployed',
        nrnCost: 100,
        capabilities: ['security-analysis', 'threat-detection'],
        metadata: {
          name: 'Network Security Agent',
          version: '1.0.0',
          description: 'Network security analysis and threat detection system',
          author: 'KNIRV Network',
          capabilities: ['security-analysis', 'threat-detection'],
          requirements: { memory: 1024, cpu: 4, storage: 200 },
          permissions: ['admin', 'security', 'monitor']
        },
        createdAt: new Date().toISOString()
      }
    ];
    setAvailableAgents(mockAgents);
  }, []);

  const handleScreenshotCapture = () => {
    setShellStatus('processing');

    setTimeout(() => {
      const newNRV: NRV = {
        id: `nrv-${Date.now()}`,
        problemDescription: 'Visual anomaly detected in interface',
        sourceID: 'KNIRV-CORTEX-main',
        inputType: 'Screenshot',
        visualContext: {
          x: Math.random() * 800,
          y: Math.random() * 600,
          width: 200,
          height: 150
        },
        temporalContext: new Date(),
        severity: 'Low',
        suggestedSolutionType: 'ui-adjustment',
        status: 'Identified'
      };
      setCurrentNRVs(prev => [...prev, newNRV]);
      setShellStatus('idle');
    }, 2000);
  };

  const handleAnalyze = () => {
    setShellStatus('processing');

    setActivePanels(prev =>
      prev.includes('key-agent-display')
        ? prev
        : [...prev, 'key-agent-display']
    );

    setTimeout(() => {
      const newNRV: NRV = {
        id: `nrv-${Date.now()}`,
        problemDescription: 'System performance degradation detected',
        sourceID: 'KNIRV-CORTEX-main',
        inputType: 'Log',
        temporalContext: new Date(),
        severity: 'Medium',
        suggestedSolutionType: 'optimization',
        status: 'Identified'
      };
      setCurrentNRVs(prev => [...prev, newNRV]);
      setShellStatus('idle');
    }, 1500);
  };

  const handleSubmitError = async () => {
    setShellStatus('processing');

    try {
      // Generate factuality slice and POST to server endpoint
      const { createFactualitySlice } = await import('./slices/factualitySlice');
      const errorId = `error-${Date.now()}`;
      const factuality = createFactualitySlice('User-submitted error for SkillNode training', { source: 'KNIRV-CONTROLLER-user' });

      await fetch('/api/graph/error', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          userId: 'local-user',
          errorId,
          errorType: 'user-submitted',
          description: 'User-submitted error for SkillNode training',
          context: { source: 'KNIRV-CONTROLLER-user', submissionType: 'manual' },
          timestamp: Date.now(),
          factualitySlice: factuality
        })
      });

      const newNRV: NRV = {
        id: `nrv-${Date.now()}`,
        problemDescription: 'User-submitted error for SkillNode training',
        sourceID: 'KNIRV-CONTROLLER-user',
        inputType: 'Error',
        temporalContext: new Date(),
        severity: 'High',
        suggestedSolutionType: 'skill-training',
        status: 'Identified'
      };
      setCurrentNRVs(prev => [...prev, newNRV]);
      setShellStatus('idle');
    } catch (error) {
      console.error('Failed to submit error:', error);
      setShellStatus('error');
      setTimeout(() => setShellStatus('idle'), 2000);
    }
  };

  const handleSubmitContext = async () => {
    setShellStatus('processing');

    try {
      // Generate a simple capability slice and POST to server
      const { createFactualitySlice } = await import('./slices/factualitySlice');
      const contextId = `context-${Date.now()}`;
      const capabilitySlice = createFactualitySlice('MCP server context for CapabilityNode creation', { serverType: 'user-submitted' });

      await fetch('/api/graph/context', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          userId: 'local-user',
          contextId,
          contextName: 'MCP Server Context',
          description: 'MCP server context for CapabilityNode creation',
          mcpServerInfo: {
            serverType: 'user-submitted',
            capabilities: ['data-processing', 'api-integration'],
            version: '1.0.0'
          },
          category: 'integration',
          timestamp: Date.now(),
          capabilitySlice
        })
      });

      const newNRV: NRV = {
        id: `nrv-${Date.now()}`,
        problemDescription: 'MCP server context for CapabilityNode creation',
        sourceID: 'KNIRV-CONTROLLER-user',
        inputType: 'Context',
        temporalContext: new Date(),
        severity: 'Medium',
        suggestedSolutionType: 'capability-mapping',
        status: 'Identified'
      };
      setCurrentNRVs(prev => [...prev, newNRV]);
      setShellStatus('idle');
    } catch (error) {
      console.error('Failed to submit context:', error);
      setShellStatus('error');
      setTimeout(() => setShellStatus('idle'), 2000);
    }
  };

  const handleSubmitIdea = async () => {
    setShellStatus('processing');

    try {
      // Generate feasibility slice and POST to server
      const { createFeasibilitySlice } = await import('./slices/feasibilitySlice');
      const ideaId = `idea-${Date.now()}`;
      const feasibility = createFeasibilitySlice('User Innovation Concept', 'User idea for PropertyNode development', []);

      await fetch('/api/graph/idea', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          userId: 'local-user',
          ideaId,
          ideaName: 'User Innovation Concept',
          description: 'User idea for PropertyNode development',
          timestamp: Date.now(),
          feasibilitySlice: feasibility
        })
      });

      const newNRV: NRV = {
        id: `nrv-${Date.now()}`,
        problemDescription: 'User idea for PropertyNode development',
        sourceID: 'KNIRV-CONTROLLER-user',
        inputType: 'Idea',
        temporalContext: new Date(),
        severity: 'Low',
        suggestedSolutionType: 'property-development',
        status: 'Identified'
      };
      setCurrentNRVs(prev => [...prev, newNRV]);
      setShellStatus('idle');
    } catch (error) {
      console.error('Failed to submit idea:', error);
      setShellStatus('error');
      setTimeout(() => setShellStatus('idle'), 2000);
    }
  };

  const handleSubmitDemo = () => {
    // Toggle the voice demo functionality
    setIsVoiceActive(prev => !prev);
    
    // Create Error Node demo NRVs
    const demoNRVs: NRV[] = [
      {
        id: `demo-error-${Date.now()}-1`,
        problemDescription: 'React component failed to render due to missing props',
        sourceID: 'KNIRV-DEMO',
        inputType: 'Error',
        temporalContext: new Date(),
        severity: 'High',
        suggestedSolutionType: 'component-fix',
        status: 'Identified'
      },
      {
        id: `demo-error-${Date.now()}-2`,
        problemDescription: 'API endpoint timeout detected in user authentication',
        sourceID: 'KNIRV-DEMO',
        inputType: 'Error',
        temporalContext: new Date(Date.now() - 5000),
        severity: 'Medium',
        suggestedSolutionType: 'api-optimization',
        status: 'Identified'
      },
      {
        id: `demo-error-${Date.now()}-3`,
        problemDescription: 'Memory leak detected in background service worker',
        sourceID: 'KNIRV-DEMO',
        inputType: 'Error',
        temporalContext: new Date(Date.now() - 10000),
        severity: 'Critical',
        suggestedSolutionType: 'memory-management',
        status: 'Identified'
      }
    ];
    
    // Add demo NRVs to current list
    setCurrentNRVs(prev => [...demoNRVs, ...prev]);
    
    // Set shell status to processing briefly to show activity
    setShellStatus('processing');
    setTimeout(() => setShellStatus('idle'), 1500);
  };

  const handleNRVMapping = (nrv: NRV) => {
    setCurrentNRVs(prev => prev.map(n =>
      n.id === nrv.id ? { ...n, status: 'Mapped' } : n
    ));
    setShellStatus('processing');

    // Open Fabric Algorithm slideout
    setActivePanels(prev =>
      prev.includes('fabric-algorithm')
        ? prev
        : [...prev, 'fabric-algorithm']
    );

    setTimeout(() => {
      setShellStatus('idle');
    }, 1000);
  };

  const handleAgentAssignment = (nrv: NRV, agent: Agent) => {
    if (gameStore.spendNRN(agent.nrnCost)) {
      setCurrentNRVs(prev => prev.map(n =>
        n.id === nrv.id ? { ...n, status: 'Assigned' } : n
      ));
      setAvailableAgents(prev => prev.map(a =>
        a.agentId === agent.agentId ? { ...a, status: 'Deployed' } : a
      ));

      setTimeout(() => {
        setCurrentNRVs(prev => prev.map(n =>
          n.id === nrv.id ? { ...n, status: 'Resolved' } : n
        ));
        setAvailableAgents(prev => prev.map(a =>
          a.agentId === agent.agentId ? { ...a, status: 'Available' } : a
        ));
      }, 5000);
    }
  };

  const handleNRVClose = (nrv: NRV) => {
    setCurrentNRVs(prev => prev.filter(n => n.id !== nrv.id));
    if (selectedNRV?.id === nrv.id) {
      setSelectedNRV(null);
    }
  };

  const closePanel = (panelId: string) => {
    setActivePanels(prev => prev.filter(id => id !== panelId));
  };

  const handleCognitiveStateChange = (state: CognitiveState) => {
    // Set cognitive mode based on whether the cognitive shell has active skills or confidence
    setCognitiveMode(state.activeSkills.length > 0 || state.confidenceLevel > 0.5);
  };

  const handleSkillInvoked = (skillId: string, result: unknown) => {
    const skillResult = result as SkillResult;
    console.log('Skill invoked:', skillId, skillResult);

    const newNRV: NRV = {
      id: `nrv-skill-${Date.now()}`,
      problemDescription: `Skill invoked: ${skillId}`,
      sourceID: 'cognitive-shell',
      inputType: 'Voice',
      temporalContext: new Date(),
      severity: 'Low',
      suggestedSolutionType: 'skill-execution',
      status: 'Resolved'
    };
    setCurrentNRVs(prev => [...prev, newNRV]);
  };

  const handleAdaptationTriggered = (adaptation: unknown) => {
    const adaptationData = adaptation as Adaptation;
    console.log('Adaptation triggered:', adaptationData);
    setShellStatus('processing');

    setTimeout(() => {
      setShellStatus('idle');
    }, 2000);
  };

  const getEdgeColor = () => {
    switch (shellStatus) {
      case 'processing': return '#3B82F6';
      case 'listening': return '#14B8A6';
      case 'error': return '#EF4444';
      default: return '#10B981';
    }
  };

  const [menuOpen, setMenuOpen] = useState(false);
  const [showQRScanner, setShowQRScanner] = useState(false);
  const [showNetworkSelector, setShowNetworkSelector] = useState(false);
  const [currentNetwork, setCurrentNetwork] = useState<NetworkType | null>(null);

  const openCognitiveShell = () => {
    setActivePanels(prev =>
      prev.includes('cognitive-shell')
        ? prev
        : [...prev, 'cognitive-shell']
    );
    setMenuOpen(false);
  };

  const openCortexBuilder = () => {
    setIsCortexBuilderOpen(true);
  };

  const openApiKeyManager = () => {
    setIsApiKeyManagerOpen(true);
  };

  const openUSDCPurchase = () => {
    setIsUSDCPurchaseOpen(true);
  };

  const handleSkillsOpen = () => {
    setActiveModal('skills');
  };

  const handleUDCOpen = () => {
    setActiveModal('udc');
  };

  const handleWalletOpen = () => {
    setActiveModal('wallet');
  };

  const handleModalClose = () => {
    setActiveModal(null);
  };

  const handleTrainSubAgent = (subAgentId: string, userInstructions: string) => {
    console.log('Training sub-agent:', subAgentId, 'with instructions:', userInstructions);
    // In a real implementation, this would trigger the training process
  };

  const handleDeployAgent = (agentId: string, position: { x: number; y: number; z: number }) => {
    console.log('Deploying agent:', agentId, 'to position:', position);
    
    // Find the sub-agent name for animation
    const subAgent = mockSubAgents.find(sa => sa.id === agentId);
    const agentName = subAgent?.name || `Agent ${agentId}`;
    
    // Start deploy animation
    gameStore.startDeployAnimation(agentId, agentName, position);
    
    // Close the modal
    setIsAgentManagementOpen(false);
  };

  // Global function to open Error Node modal from any component
  useEffect(() => {
    if (typeof window !== 'undefined') {
      window.openErrorNodeModal = () => {
        setIsErrorNodeOpen(true);
      };
    }
  }, []);

  const toggleNetworkPanel = () => {
    setActivePanels(prev =>
      prev.includes('network-status')
        ? prev.filter(id => id !== 'network-status')
        : [...prev, 'network-status']
    );
    setMenuOpen(false);
  };

  const toggleKeyAgentPanel = () => {
    setActivePanels(prev =>
      prev.includes('key-agent-display')
        ? prev.filter(id => id !== 'key-agent-display')
        : [...prev, 'key-agent-display']
    );
    setMenuOpen(false);
  };

  const handleQRScan = () => {
    setShowQRScanner(true);
    setMenuOpen(false);
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white relative overflow-hidden">
      <EdgeColoring color={getEdgeColor()} intensity={shellStatus !== 'idle' ? 0.8 : 0.3} />

      {/* Burger Menu Navigation - positioned to avoid time metrics */}
      <div className="absolute top-20 right-4 z-50">
        <BurgerMenu
          isOpen={menuOpen}
          onToggle={() => setMenuOpen(!menuOpen)}
          showNetworkSelector={showNetworkSelector}
          setShowNetworkSelector={setShowNetworkSelector}
          currentNetwork={currentNetwork}
          setCurrentNetwork={setCurrentNetwork}
        >
          <MenuItem onClick={handleQRScan} icon="📱">
            QR Scanner
          </MenuItem>
          <MenuItem onClick={openApiKeyManager} icon="🔑">
            API Keys
          </MenuItem>
          <MenuItem onClick={openUSDCPurchase} icon="💰">
            Buy NRN Tokens
          </MenuItem>
          <MenuItem onClick={() => { setShowNetworkSelector?.(true); setMenuOpen(false); }} icon="🔗">
            Network
          </MenuItem>
          <MenuItem onClick={() => setIsAgentManagementOpen(true)} icon="🤖">
            Agent Management
          </MenuItem>
          <MenuItem onClick={() => setIsTraverseClustersOpen(true)} icon="🔀">
            Traverse Clusters
          </MenuItem>
          <MenuItem onClick={() => toggleKeyAgentPanel()} icon="🤖">
            Key Agent Status
          </MenuItem>
        </BurgerMenu>
      </div>
      
      <main ref={shellRef} className="relative w-full h-screen" role="main">
        <KnirvShell
          status={shellStatus}
          nrnBalance={nrnBalance}
          onScreenshotCapture={handleScreenshotCapture}
          cognitiveMode={cognitiveMode}
          isVoiceActive={isVoiceActive}
          onSubmitError={handleSubmitError}
          onSubmitContext={handleSubmitContext}
          onSubmitIdea={handleSubmitIdea}
          onSubmitDemo={handleSubmitDemo}
          onSkillsOpen={handleSkillsOpen}
          onUDCOpen={handleUDCOpen}
          onWalletOpen={handleWalletOpen}
        />

        {gamePhase === 'menu' && !hasSeenIntro && (
          <div className="absolute inset-0 z-[999999] pointer-events-auto">
            <GameMenu 
              onStart={() => {
                gameStore.startGame();
                setHasSeenIntro(true);
              }} 
              usingMockLLM={gameStore.usingMockLLM}
            />
          </div>
        )}

        <NRVVisualization
          nrvs={currentNRVs}
          onNRVSelect={setSelectedNRV}
          onNRVMapping={handleNRVMapping}
          onNRVClose={handleNRVClose}
          onAnalyze={handleAnalyze}
        />

        {/* Sliding Panels */}
        <SlidingPanel
          id="network-status"
          isOpen={activePanels.includes('network-status')}
          onClose={() => closePanel('network-status')}
          title="Network Status"
          side="right"
        >
          <NetworkStatus connections={networkConnections} />
        </SlidingPanel>

        <SlidingPanel
          id="key-agent-display"
          isOpen={activePanels.includes('key-agent-display')}
          onClose={() => closePanel('key-agent-display')}
          title="Key Agent Status"
          side="left"
        >
          <div className="p-4 space-y-4">
            {/* Large Robot Image */}
            <div className="flex justify-center">
              <img
                src={`${import.meta.env.BASE_URL}assets/avatar/bot_default.png`}
                alt="Key Agent"
                className="w-64 h-64 rounded-full border-4 border-green-500/50 shadow-2xl"
              />
            </div>
            
            {/* Key Agent Info */}
            <div className="text-center space-y-2">
              <h3 className="text-xl font-bold text-white">Key Agent</h3>
              <div className="flex items-center justify-center gap-2">
                <div className="w-3 h-3 bg-green-400 rounded-full animate-pulse"></div>
                <span className="text-green-400 font-medium">Online & Active</span>
              </div>
              <p className="text-sm text-gray-400">
                Your personal AI assistant for training and managing sub-agents
              </p>
            </div>

            {/* Status Message */}
            <div className="bg-gray-800 rounded-lg p-4">
              <div className="text-sm font-medium text-gray-300 mb-1">Current Status</div>
              <div className="text-base text-white">
                {availableAgents.length > 0 
                  ? `Monitoring ${availableAgents.length} active agents`
                  : 'Awaiting agent deployment'
                }
              </div>
              <div className="text-xs text-gray-400 mt-2">
                Last activity: {new Date().toLocaleTimeString()}
              </div>
            </div>

            {/* Action Buttons */}
            <div className="space-y-2">
              <button
                onClick={() => {
                  openCortexBuilder();
                  closePanel('key-agent-display');
                }}
                className="w-full px-4 py-2 bg-purple-600 hover:bg-purple-700 text-white rounded-lg transition-colors"
              >
                Open CORTEX Builder
              </button>
              <button
                onClick={() => {
                  setIsAgentManagementOpen(true);
                  closePanel('key-agent-display');
                }}
                className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
              >
                Open Agent Management
              </button>
            </div>
            
            {/* Quick Stats */}
            <div className="grid grid-cols-2 gap-3 mt-4">
              <div className="bg-gray-800 rounded-lg p-3 text-center">
                <div className="text-lg font-bold text-white">{mockSubAgents.filter(sa => sa.status === 'ready').length}</div>
                <div className="text-xs text-gray-400">Ready Agents</div>
              </div>
              <div className="bg-gray-800 rounded-lg p-3 text-center">
                <div className="text-lg font-bold text-blue-400">{mockSubAgents.filter(sa => sa.status === 'training').length}</div>
                <div className="text-xs text-gray-400">In Training</div>
              </div>
            </div>
          </div>
        </SlidingPanel>

        <SlidingPanel
          id="cognitive-shell"
          isOpen={activePanels.includes('cognitive-shell')}
          onClose={() => closePanel('cognitive-shell')}
          title="Cognitive Shell"
          side="right"
        >
          <ChatBrainProvider>
            <CognitiveShellInterface
              onStateChange={handleCognitiveStateChange}
              onSkillInvoked={handleSkillInvoked}
              onAdaptationTriggered={handleAdaptationTriggered}
              onOpenCortexBuilder={openCortexBuilder}
            />
          </ChatBrainProvider>
        </SlidingPanel>

        <SlidingPanel
          id="fabric-algorithm"
          isOpen={activePanels.includes('fabric-algorithm')}
          onClose={() => closePanel('fabric-algorithm')}
          title="Fabric Algorithm"
          side="right"
        >
          <FabricAlgorithm
            status={shellStatus}
            nrvCount={currentNRVs.length}
          />
        </SlidingPanel>

        {/* CORTEX Builder Modal */}
        <CortexBuilder
          isOpen={isCortexBuilderOpen}
          onClose={() => setIsCortexBuilderOpen(false)}
        />

        {/* API Key Manager Modal */}
        <ApiKeyManager
          isOpen={isApiKeyManagerOpen}
          onClose={() => setIsApiKeyManagerOpen(false)}
        />

        {/* USDC to NRN Purchase Modal */}
        {isUSDCPurchaseOpen && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="relative">
              <button
                onClick={() => setIsUSDCPurchaseOpen(false)}
                className="absolute -top-2 -right-2 bg-red-500 text-white rounded-full w-6 h-6 flex items-center justify-center text-sm hover:bg-red-600 z-10"
              >
                ×
              </button>
              <USDCToNRNPurchase
                onPurchaseComplete={(result) => {
                  console.log('Purchase completed:', result);
                  // Update NRN balance if needed
                  gameStore.addNRN(parseFloat(result.nrnAmount));
                }}
                onError={(error) => {
                  console.error('Purchase error:', error);
                  // Could show a toast notification here
                }}
              />
            </div>
          </div>
        )}

        {/* Auto Demo Status Indicator */}
        {isVoiceActive && (
          <div className="absolute top-4 left-1/2 transform -translate-x-1/2 z-50">
            <div className="bg-teal-500 text-white px-4 py-2 rounded-full text-sm font-medium shadow-lg animate-pulse">
              <div className="flex items-center space-x-2">
                <div className="w-2 h-2 bg-white rounded-full animate-ping"></div>
                <span>Auto Demo</span>
                {cognitiveMode && <span className="text-xs opacity-75">(Cognitive)</span>}
              </div>
            </div>
          </div>
        )}

        {/* Status Indicator - Shows Cognitive Mode when active, otherwise shows shell status */}
        <div className="absolute bottom-4 left-4 z-40">
          {cognitiveMode ? (
            <div className="bg-purple-500/20 text-purple-400 border border-purple-500/30 px-3 py-1 rounded-full text-xs font-medium shadow-lg">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-purple-400 rounded-full animate-pulse"></div>
                <span>Cognitive Mode</span>
              </div>
            </div>
          ) : (
            <div className={`px-3 py-1 rounded-full text-xs font-medium transition-all duration-300 ${
              shellStatus === 'idle' ? 'bg-green-500/20 text-green-400 border border-green-500/30' :
              shellStatus === 'processing' ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' :
              shellStatus === 'listening' ? 'bg-teal-500/20 text-teal-400 border border-teal-500/30' :
              'bg-red-500/20 text-red-400 border border-red-500/30'
            }`}>
              {shellStatus.charAt(0).toUpperCase() + shellStatus.slice(1)}
            </div>
            )}
        </div>
        </main>

      {/* Slide-Down Modals */}
      <SlideDownModal
        isOpen={activeModal === 'skills'}
        onClose={handleModalClose}
        title="Skills"
      >
        <SkillsModalContent nrnBalance={nrnBalance} />
      </SlideDownModal>

      <SlideDownModal
        isOpen={activeModal === 'udc'}
        onClose={handleModalClose}
        title="User Delegation Certificate"
      >
        <UDCModalContent />
      </SlideDownModal>

      <SlideDownModal
        isOpen={activeModal === 'wallet'}
        onClose={handleModalClose}
        title="KNIRV Wallet"
      >
        <WalletModalContent 
          nrnBalance={nrnBalance} 
          cognitiveMode={cognitiveMode}
          cognitiveState={{ mode: 'active', isActive: cognitiveMode }}
        />
      </SlideDownModal>

      {/* New Agent Management Modal */}
      <AgentManagementModal
        isOpen={isAgentManagementOpen}
        onClose={() => setIsAgentManagementOpen(false)}
        keyAgent={availableAgents[0] || null}
        subAgents={mockSubAgents}
        nrnBalance={nrnBalance}
        onTrainSubAgent={handleTrainSubAgent}
        onDeployAgent={handleDeployAgent}
      />

      {/* Error Node Management Modal */}
      <ErrorNodeModal
        isOpen={isErrorNodeOpen}
        onClose={() => setIsErrorNodeOpen(false)}
        nrvs={currentNRVs}
        selectedNRV={selectedNRV}
        nrnBalance={nrnBalance}
        onDeployAgent={(nrv: NRV) => {
          // Find the first available agent to deploy
          const availableAgent = availableAgents.find(agent =>
            agent.status === 'Available' && nrnBalance >= agent.nrnCost
          );
          if (availableAgent) {
            handleAgentAssignment(nrv, availableAgent);
          } else {
            console.warn('No available agent found for deployment');
          }
        }}
      />

      {/* Traverse Clusters Modal - Flythrough Animation */}
      <TraverseClustersModal
        isOpen={isTraverseClustersOpen}
        onClose={() => setIsTraverseClustersOpen(false)}
      />

      {/* QR Scanner Modal */}
      {showQRScanner && (
        <div className="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center z-50">
          <div className="bg-gray-800 p-6 rounded-lg max-w-md w-full mx-4 border border-gray-600">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-semibold text-white">QR Code Scanner</h3>
              <button
                onClick={() => setShowQRScanner(false)}
                className="text-gray-400 hover:text-white transition-colors"
              >
                <X size={20} />
              </button>
            </div>
            <div className="aspect-square bg-gray-700 rounded-lg flex items-center justify-center">
              <div className="text-center">
                <QrCode size={48} className="text-gray-400 mx-auto mb-2" />
                <p className="text-gray-400">QR Scanner Component</p>
                <p className="text-gray-500 text-sm mt-1">Camera access required</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// Agent Profile Component
const AgentProfile = () => {
  const navigate = useNavigate();
  const { agentId } = useParams();
  const [menuOpen, setMenuOpen] = useState(false);
  const [showQRScanner, setShowQRScanner] = useState(false);

  // Mock agent data - in real app this would come from API
  const agent = {
    id: agentId,
    name: agentId === 'codet5-alpha' ? 'CodeT5-Alpha' :
          agentId === 'seal-beta' ? 'SEAL-Beta' :
          agentId === 'lora-gamma' ? 'LoRA-Gamma' : 'Unknown Agent',
    type: agentId === 'codet5-alpha' ? 'KNIRV-CORTEX' :
          agentId === 'seal-beta' ? 'KNIRVANA' :
          agentId === 'lora-gamma' ? 'DVE' : 'Unknown',
    status: agentId === 'codet5-alpha' ? 'active' :
            agentId === 'seal-beta' ? 'active' :
            agentId === 'lora-gamma' ? 'idle' : 'offline',
    performance: agentId === 'codet5-alpha' ? 94 :
                 agentId === 'seal-beta' ? 87 :
                 agentId === 'lora-gamma' ? 91 : 78,
    tasks: agentId === 'codet5-alpha' ? 12 :
           agentId === 'seal-beta' ? 8 :
           agentId === 'lora-gamma' ? 0 : 0,
    lastActive: agentId === 'codet5-alpha' ? '2 min ago' :
                agentId === 'seal-beta' ? '5 min ago' :
                agentId === 'lora-gamma' ? '1 hour ago' : '3 hours ago',
    specialization: agentId === 'codet5-alpha' ? ['code-generation', 'optimization'] :
                    agentId === 'seal-beta' ? ['learning', 'adaptation'] :
                    agentId === 'lora-gamma' ? ['fine-tuning', 'model-adaptation'] : ['unknown'],
    nrnCost: agentId === 'codet5-alpha' ? 85 :
             agentId === 'seal-beta' ? 90 :
             agentId === 'lora-gamma' ? 120 : 100,
    description: agentId === 'codet5-alpha' ? 'Advanced code generation and optimization agent powered by CodeT5 architecture.' :
                 agentId === 'seal-beta' ? 'Self-evolving adaptive learning agent with continuous improvement capabilities.' :
                 agentId === 'lora-gamma' ? 'Low-rank adaptation specialist for fine-tuning large language models.' : 'Unknown agent type.',
    capabilities: agentId === 'codet5-alpha' ? ['Code Generation', 'Bug Detection', 'Performance Optimization', 'Documentation'] :
                  agentId === 'seal-beta' ? ['Adaptive Learning', 'Pattern Recognition', 'Behavior Modeling', 'Prediction'] :
                  agentId === 'lora-gamma' ? ['Model Fine-tuning', 'Parameter Optimization', 'Transfer Learning', 'Efficiency'] : ['Unknown'],
    metrics: {
      uptime: agentId === 'codet5-alpha' ? '99.2%' :
              agentId === 'seal-beta' ? '98.7%' :
              agentId === 'lora-gamma' ? '95.1%' : '89.3%',
      accuracy: agentId === 'codet5-alpha' ? '94.8%' :
                agentId === 'seal-beta' ? '92.3%' :
                agentId === 'lora-gamma' ? '96.7%' : '78.2%',
      responseTime: agentId === 'codet5-alpha' ? '1.2s' :
                    agentId === 'seal-beta' ? '0.8s' :
                    agentId === 'lora-gamma' ? '2.1s' : '3.4s',
      totalTasks: agentId === 'codet5-alpha' ? 1247 :
                  agentId === 'seal-beta' ? 892 :
                  agentId === 'lora-gamma' ? 634 : 234
    }
  };

  const handleQRScan = () => {
    setShowQRScanner(true);
    setMenuOpen(false);
  };



  return (
    <div className="min-h-screen bg-gray-900 text-white relative overflow-hidden">
      {/* Burger Menu Navigation */}
      <div className="absolute top-4 right-4 z-50">
        <BurgerMenu isOpen={menuOpen} onToggle={() => setMenuOpen(!menuOpen)}>
          <MenuItem onClick={() => { navigate('/manager/skills'); setMenuOpen(false); }} icon="⚡">
            Skills
          </MenuItem>
          <MenuItem onClick={() => { navigate('/manager/udc'); setMenuOpen(false); }} icon="🔐">
            UDC
          </MenuItem>
          <MenuItem onClick={() => { navigate('/manager/wallet'); setMenuOpen(false); }} icon="💰">
            Wallet
          </MenuItem>
          <MenuItem onClick={handleQRScan} icon="📱">
            QR Scanner
          </MenuItem>
          <MenuItem onClick={() => { navigate('/'); setMenuOpen(false); }} icon="🏠">
            Input Interface
          </MenuItem>
        </BurgerMenu>
      </div>

      <div className="max-w-6xl mx-auto p-4 pb-24 overflow-y-auto h-screen">
        <div className="space-y-6">
          {/* Header */}
          <div className="flex items-center space-x-4 mb-6">
            <button
              onClick={() => navigate('/')}
              className="bg-gray-800/80 hover:bg-gray-700/80 text-white px-4 py-2 rounded-lg shadow-lg transition-all duration-200 font-medium border border-gray-600/50"
            >
              ← Back
            </button>
            <div>
              <h1 className="text-3xl font-bold text-white">{agent.name}</h1>
              <p className="text-gray-400">{agent.type} Agent Profile</p>
            </div>
          </div>

          {/* Status Card */}
          <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center space-x-3">
                <div className={`w-4 h-4 rounded-full ${
                  agent.status === 'active' ? 'bg-green-400 animate-pulse' :
                  agent.status === 'idle' ? 'bg-yellow-400' : 'bg-red-400'
                }`}></div>
                <h2 className="text-xl font-semibold text-white">Status: {agent.status}</h2>
              </div>
              <div className="text-right">
                <p className="text-lg font-semibold text-white">{agent.nrnCost} NRN/hour</p>
                <p className="text-gray-400 text-sm">Operating cost</p>
              </div>
            </div>
            <p className="text-gray-300 mb-4">{agent.description}</p>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="text-center">
                <div className="text-2xl font-bold text-blue-400">{agent.tasks}</div>
                <div className="text-gray-400 text-sm">Active Tasks</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-green-400">{agent.performance}%</div>
                <div className="text-gray-400 text-sm">Performance</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-purple-400">{agent.metrics.uptime}</div>
                <div className="text-gray-400 text-sm">Uptime</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-cyan-400">{agent.metrics.totalTasks}</div>
                <div className="text-gray-400 text-sm">Total Tasks</div>
              </div>
            </div>
          </div>

          {/* Capabilities */}
          <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-white mb-4">Capabilities</h3>
            <div className="grid grid-cols-2 gap-3">
              {agent.capabilities.map((capability, index) => (
                <div key={index} className="bg-gray-700/50 border border-gray-600/30 rounded-lg p-3">
                  <p className="text-white font-medium">{capability}</p>
                </div>
              ))}
            </div>
          </div>

          {/* Performance Metrics */}
          <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-white mb-4">Performance Metrics</h3>
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <span className="text-gray-400">Accuracy</span>
                <span className="text-white font-semibold">{agent.metrics.accuracy}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-400">Response Time</span>
                <span className="text-white font-semibold">{agent.metrics.responseTime}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-400">Last Active</span>
                <span className="text-white font-semibold">{agent.lastActive}</span>
              </div>
            </div>
          </div>

          {/* Specializations */}
          <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-white mb-4">Specializations</h3>
            <div className="flex flex-wrap gap-2">
              {agent.specialization?.map((spec, index) => (
                <span key={index} className="bg-blue-600/20 text-blue-400 px-3 py-1 rounded-full text-sm">
                  {spec}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* QR Scanner Modal */}
      {showQRScanner && (
        <div className="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center z-50">
          <div className="bg-gray-800 p-6 rounded-lg max-w-md w-full mx-4 border border-gray-600">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-semibold text-white">QR Code Scanner</h3>
              <button
                onClick={() => setShowQRScanner(false)}
                className="text-gray-400 hover:text-white transition-colors"
              >
                <X size={20} />
              </button>
            </div>
            <div className="aspect-square bg-gray-700 rounded-lg flex items-center justify-center">
              <div className="text-center">
                <QrCode size={48} className="text-gray-400 mx-auto mb-2" />
                <p className="text-gray-400">QR Scanner Component</p>
                <p className="text-gray-500 text-sm mt-1">Camera access required</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// Manager Interface Wrapper
const ManagerInterface = () => {
  const navigate = useNavigate();

  // Redirect to skills page by default
  React.useEffect(() => {
    if (window.location.pathname === '/manager' || window.location.pathname === '/manager/') {
      navigate('/manager/skills', { replace: true });
    }
  }, [navigate]);

  return (
    <div className="min-h-screen bg-gray-900 text-white relative">
      <Suspense fallback={<div className="flex items-center justify-center h-screen"><div className="text-white">Loading...</div></div>}>
        <Routes>
          <Route path="/skills" element={<Skills />} />
          <Route path="/udc" element={<UDC />} />
          <Route path="/wallet" element={<WalletPage />} />
          <Route path="/chat-brain" element={<ChatBrain />} />
          <Route path="/agent/:agentId" element={<AgentProfile />} />
        </Routes>
      </Suspense>
    </div>
  );
};

// Main App Component
function App() {
  useEffect(() => {
    // Initialize Sentry for error monitoring
    initSentry();
  }, []);

  return (
    <Router basename="/arena">
      <Routes>
        <Route path="/*" element={<ReceiverInterface />} />
        <Route path="/manager/*" element={<ManagerInterface />} />
      </Routes>
    </Router>
  );
}

export default App;
