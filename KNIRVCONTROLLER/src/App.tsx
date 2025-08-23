import React, { useState, useEffect, useRef } from 'react';
import { BrowserRouter as Router, Routes, Route, useNavigate, useLocation, useParams } from 'react-router-dom';
import { QrCode, X } from 'lucide-react';

// Receiver components
import { KnirvShell } from './components/KnirvShell';
import { VoiceControl } from './components/VoiceControl';
import { NetworkStatus } from './components/NetworkStatus';
import { NRVVisualization } from './components/NRVVisualization';
import { SlidingPanel } from './components/SlidingPanel';
import { EdgeColoring } from './components/EdgeColoring';
import { AgentManager } from './components/AgentManager';
import { FabricAlgorithm } from './components/FabricAlgorithm';
import { CognitiveShellInterface } from './components/CognitiveShellInterface';
import { CognitiveState } from './sensory-shell/CognitiveEngine';

// Manager components
import UnifiedInterface from './manager/react-app/components/UnifiedInterface';
import Skills from './manager/react-app/pages/Skills';
import UDC from './manager/react-app/pages/UDC';
import WalletPage from './manager/react-app/pages/Wallet';

import { backendAPI } from './backend/api';
import { loraEngine } from './backend/loraEngine';
import { wasmCompiler } from './backend/wasmCompiler';
import { protobufHandler } from './backend/protobufHandler';
import { ComponentBridge, ComponentConfig } from './shared/ComponentBridge';

// Types from receiver
export interface Adaptation {
  id: string;
  type: string;
  description: string;
  timestamp: Date;
}

export interface SkillResult {
  success: boolean;
  data?: unknown;
  error?: string;
  executionTime?: number;
}

export interface NRV {
  id: string;
  problemDescription: string;
  sourceID: string;
  inputType: 'Voice' | 'Screenshot' | 'Log' | 'Camera';
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

export interface Agent {
  id: string;
  name: string;
  type: 'KNIRV-CORTEX' | 'KNIRVANA' | 'DVE';
  status: 'Available' | 'Busy' | 'Offline';
  specialization: string[];
  nrnCost: number;
}

// Navigation Button Component
const NavigationButton = ({ to, children, className = '' }) => {
  const navigate = useNavigate();

  return (
    <button
      onClick={() => navigate(to)}
      className={`bg-gray-800/80 hover:bg-gray-700/80 text-white px-4 py-2 rounded-lg shadow-lg transition-all duration-200 font-medium border border-gray-600/50 backdrop-blur-sm ${className}`}
    >
      {children}
    </button>
  );
};

// Burger Menu Component
const BurgerMenu = ({ isOpen, onToggle, children }) => {
  return (
    <div className="relative">
      {/* Burger Button */}
      <button
        onClick={onToggle}
        className="bg-gray-800/80 hover:bg-gray-700/80 text-white p-3 rounded-lg shadow-lg transition-all duration-200 border border-gray-600/50 backdrop-blur-sm"
        aria-label="Navigation menu"
      >
        <div className="w-5 h-5 flex flex-col justify-center items-center">
          <div className={`w-5 h-0.5 bg-white transition-all duration-300 ${isOpen ? 'rotate-45 translate-y-1' : ''}`}></div>
          <div className={`w-5 h-0.5 bg-white transition-all duration-300 mt-1 ${isOpen ? 'opacity-0' : ''}`}></div>
          <div className={`w-5 h-0.5 bg-white transition-all duration-300 mt-1 ${isOpen ? '-rotate-45 -translate-y-1' : ''}`}></div>
        </div>
      </button>

      {/* Menu Dropdown */}
      {isOpen && (
        <div className="absolute top-full right-0 mt-2 bg-gray-800/90 backdrop-blur-sm border border-gray-600/50 rounded-lg shadow-xl min-w-48 z-50">
          <div className="p-2 space-y-1">
            {children}
          </div>
        </div>
      )}
    </div>
  );
};

// Menu Item Component
const MenuItem = ({ onClick, children, icon, className = '' }) => {
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
  const navigate = useNavigate();
  const [shellStatus, setShellStatus] = useState<'idle' | 'processing' | 'listening' | 'error'>('idle');
  const [isVoiceActive, setIsVoiceActive] = useState(false);
  const [currentNRVs, setCurrentNRVs] = useState<NRV[]>([]);
  const [selectedNRV, setSelectedNRV] = useState<NRV | null>(null);
  const [availableAgents, setAvailableAgents] = useState<Agent[]>([]);
  const [activePanels, setActivePanels] = useState<string[]>([]);
  const [nrnBalance, setNrnBalance] = useState(1250);
  const [cognitiveMode, setCognitiveMode] = useState(false);
  const [cognitiveState, setCognitiveState] = useState<CognitiveState | null>(null);
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

  useEffect(() => {
    // Initialize mock agents
    const mockAgents: Agent[] = [
      {
        id: 'agent-1',
        name: 'System Diagnostics Agent',
        type: 'KNIRV-CORTEX',
        status: 'Available',
        specialization: ['error-detection', 'system-analysis'],
        nrnCost: 50
      },
      {
        id: 'agent-2', 
        name: 'UI/UX Optimization Agent',
        type: 'KNIRVANA',
        status: 'Available',
        specialization: ['interface-design', 'user-experience'],
        nrnCost: 75
      },
      {
        id: 'agent-3',
        name: 'Network Security Agent',
        type: 'DVE',
        status: 'Busy',
        specialization: ['security-analysis', 'threat-detection'],
        nrnCost: 100
      }
    ];
    setAvailableAgents(mockAgents);
  }, []);

  const handleVoiceCommand = (command: string) => {
    setShellStatus('processing');

    setTimeout(() => {
      const lowerCommand = command.toLowerCase();

      if (lowerCommand.includes('identify problems')) {
        const newNRV: NRV = {
          id: `nrv-${Date.now()}`,
          problemDescription: `User reported issue: ${command}`,
          sourceID: 'KNIRV-CORTEX-main',
          inputType: 'Voice',
          temporalContext: new Date(),
          severity: 'Medium',
          suggestedSolutionType: 'investigation',
          status: 'Identified'
        };
        setCurrentNRVs(prev => [...prev, newNRV]);
        setShellStatus('idle');
      } else if (lowerCommand.includes('show network')) {
        setActivePanels(['network-status']);
        setShellStatus('idle');
      } else if (lowerCommand.includes('assign agents')) {
        setActivePanels(['agent-manager']);
        setShellStatus('idle');
      } else if (lowerCommand.includes('cognitive mode') || lowerCommand.includes('enable cognitive')) {
        setCognitiveMode(true);
        setActivePanels(prev => [...prev, 'cognitive-shell']);
        setShellStatus('idle');
      } else if (lowerCommand.includes('start learning')) {
        setActivePanels(prev => [...prev, 'cognitive-shell']);
        setShellStatus('idle');
      } else if (lowerCommand.includes('capture screen')) {
        handleScreenshotCapture();
        return;
      } else if (lowerCommand.includes('toggle network')) {
        handleNetworkToggle();
        setShellStatus('idle');
      } else {
        setShellStatus('idle');
      }
    }, 1500);
  };

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
      prev.includes('agent-manager')
        ? prev
        : [...prev, 'agent-manager']
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

  const handleNetworkToggle = () => {
    setActivePanels(prev =>
      prev.includes('network-status')
        ? prev.filter(id => id !== 'network-status')
        : [...prev, 'network-status']
    );
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
    if (nrnBalance >= agent.nrnCost) {
      setNrnBalance(prev => prev - agent.nrnCost);
      setCurrentNRVs(prev => prev.map(n =>
        n.id === nrv.id ? { ...n, status: 'Assigned' } : n
      ));
      setAvailableAgents(prev => prev.map(a =>
        a.id === agent.id ? { ...a, status: 'Busy' } : a
      ));

      setTimeout(() => {
        setCurrentNRVs(prev => prev.map(n =>
          n.id === nrv.id ? { ...n, status: 'Resolved' } : n
        ));
        setAvailableAgents(prev => prev.map(a =>
          a.id === agent.id ? { ...a, status: 'Available' } : a
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
    setCognitiveState(state);
    // Set cognitive mode based on whether the cognitive shell is active
    setCognitiveMode(state.status === 'active' || state.status === 'learning');
  };

  const handleSkillInvoked = (skillId: string, result: SkillResult) => {
    console.log('Skill invoked:', skillId, result);

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

  const handleAdaptationTriggered = (adaptation: Adaptation) => {
    console.log('Adaptation triggered:', adaptation);
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

  const openCognitiveShell = () => {
    setActivePanels(prev =>
      prev.includes('cognitive-shell')
        ? prev
        : [...prev, 'cognitive-shell']
    );
    setMenuOpen(false);
  };

  const toggleNetworkPanel = () => {
    setActivePanels(prev =>
      prev.includes('network-status')
        ? prev.filter(id => id !== 'network-status')
        : [...prev, 'network-status']
    );
    setMenuOpen(false);
  };

  const toggleAgentPanel = () => {
    setActivePanels(prev =>
      prev.includes('agent-manager')
        ? prev.filter(id => id !== 'agent-manager')
        : [...prev, 'agent-manager']
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
          <MenuItem onClick={openCognitiveShell} icon="🧠">
            Cognitive Shell
          </MenuItem>
          <MenuItem onClick={toggleNetworkPanel} icon="🌐">
            Network Status
          </MenuItem>
          <MenuItem onClick={toggleAgentPanel} icon="🤖">
            Agent Management
          </MenuItem>
        </BurgerMenu>
      </div>
      
      <main ref={shellRef} className="relative w-full h-screen" role="main">
        <KnirvShell
          status={shellStatus}
          nrnBalance={nrnBalance}
          onScreenshotCapture={handleScreenshotCapture}
          cognitiveMode={cognitiveMode}
        />

        <VoiceControl
          isActive={isVoiceActive}
          onVoiceCommand={handleVoiceCommand}
          onToggle={setIsVoiceActive}
          cognitiveMode={cognitiveMode}
        />

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
          id="agent-manager"
          isOpen={activePanels.includes('agent-manager')}
          onClose={() => closePanel('agent-manager')}
          title="Agent Management"
          side="left"
        >
          <AgentManager
            agents={availableAgents}
            nrvs={currentNRVs}
            selectedNRV={selectedNRV}
            onAgentAssignment={handleAgentAssignment}
            nrnBalance={nrnBalance}
          />
        </SlidingPanel>

        <SlidingPanel
          id="cognitive-shell"
          isOpen={activePanels.includes('cognitive-shell')}
          onClose={() => closePanel('cognitive-shell')}
          title="Cognitive Shell"
          side="right"
        >
          <CognitiveShellInterface
            onStateChange={handleCognitiveStateChange}
            onSkillInvoked={handleSkillInvoked}
            onAdaptationTriggered={handleAdaptationTriggered}
          />
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

        {/* Voice Status Indicator */}
        {isVoiceActive && (
          <div className="absolute top-4 left-1/2 transform -translate-x-1/2 z-50">
            <div className="bg-teal-500 text-white px-4 py-2 rounded-full text-sm font-medium shadow-lg animate-pulse">
              <div className="flex items-center space-x-2">
                <div className="w-2 h-2 bg-white rounded-full animate-ping"></div>
                <span>Voice Active</span>
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

  // Burger Menu Component
  const BurgerMenu = ({ isOpen, onToggle, children }) => {
    return (
      <div className="relative">
        <button
          onClick={onToggle}
          className="bg-gray-800/80 hover:bg-gray-700/80 text-white p-3 rounded-lg shadow-lg transition-all duration-200 border border-gray-600/50 backdrop-blur-sm"
          aria-label="Navigation menu"
        >
          <div className="w-5 h-5 flex flex-col justify-center items-center">
            <div className={`w-5 h-0.5 bg-white transition-all duration-300 ${isOpen ? 'rotate-45 translate-y-1' : ''}`}></div>
            <div className={`w-5 h-0.5 bg-white transition-all duration-300 mt-1 ${isOpen ? 'opacity-0' : ''}`}></div>
            <div className={`w-5 h-0.5 bg-white transition-all duration-300 mt-1 ${isOpen ? '-rotate-45 -translate-y-1' : ''}`}></div>
          </div>
        </button>

        {isOpen && (
          <div className="absolute top-full right-0 mt-2 w-64 bg-gray-800/95 backdrop-blur-xl rounded-lg shadow-xl border border-gray-600/50 py-2 z-50">
            {children}
          </div>
        )}
      </div>
    );
  };

  const MenuItem = ({ onClick, icon, children }) => {
    return (
      <button
        onClick={onClick}
        className="w-full flex items-center space-x-3 px-4 py-3 text-left hover:bg-gray-700/50 transition-colors text-white"
      >
        <span className="text-lg">{icon}</span>
        <span className="font-medium">{children}</span>
      </button>
    );
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
              {agent.specialization.map((spec, index) => (
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
      <Routes>
        <Route path="/skills" element={<Skills />} />
        <Route path="/udc" element={<UDC />} />
        <Route path="/wallet" element={<WalletPage />} />
        <Route path="/agent/:agentId" element={<AgentProfile />} />
      </Routes>
    </div>
  );
};

// Main App Component
function App() {
  return (
    <Router>
      <Routes>
        <Route path="/*" element={<ReceiverInterface />} />
        <Route path="/manager/*" element={<ManagerInterface />} />
      </Routes>
    </Router>
  );
}

export default App;