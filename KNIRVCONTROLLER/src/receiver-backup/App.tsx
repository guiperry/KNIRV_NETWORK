import React, { useState, useEffect, useRef } from 'react';
import { BrowserRouter as Router, Routes, Route, useNavigate, useLocation } from 'react-router-dom';

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
import { CognitiveState } from '../sensory-shell/CognitiveEngine';

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

  const openCognitiveShell = () => {
    setCognitiveMode(true);
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

  const navigateToManager = () => {
    setMenuOpen(false);
    // Small delay to allow menu to close before navigation
    setTimeout(() => {
      window.location.href = '/manager';
    }, 100);
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white relative overflow-hidden">
      <EdgeColoring color={getEdgeColor()} intensity={shellStatus !== 'idle' ? 0.8 : 0.3} />

      {/* Burger Menu Navigation - positioned to avoid time metrics */}
      <div className="absolute top-20 right-4 z-50">
        <BurgerMenu isOpen={menuOpen} onToggle={() => setMenuOpen(!menuOpen)}>
          <MenuItem onClick={navigateToManager} icon="🔧">
            Manager Interface
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
          onAnalyze={handleAnalyze}
          onNetworkToggle={handleNetworkToggle}
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
        />

        <FabricAlgorithm
          status={shellStatus}
          nrvCount={currentNRVs.length}
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

        {/* Cognitive Mode Indicator */}
        {cognitiveMode && (
          <div className="absolute top-4 right-20 z-50">
            <div className="bg-purple-500 text-white px-3 py-1 rounded-full text-xs font-medium shadow-lg">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-white rounded-full animate-pulse"></div>
                <span>Cognitive Mode</span>
              </div>
            </div>
          </div>
        )}

        {/* Status Indicator */}
        <div className="absolute bottom-4 left-4 z-40">
          <div className={`px-3 py-1 rounded-full text-xs font-medium transition-all duration-300 ${
            shellStatus === 'idle' ? 'bg-green-500/20 text-green-400 border border-green-500/30' :
            shellStatus === 'processing' ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' :
            shellStatus === 'listening' ? 'bg-teal-500/20 text-teal-400 border border-teal-500/30' :
            'bg-red-500/20 text-red-400 border border-red-500/30'
          }`}>
            {shellStatus.charAt(0).toUpperCase() + shellStatus.slice(1)}
          </div>
        </div>
      </main>
    </div>
  );
};

// Simple Manager Interface Fallback
const SimpleManagerInterface = () => {
  const navigate = useNavigate();



  return (
    <div className="min-h-screen bg-gray-900 text-white p-8">
      <div className="max-w-6xl mx-auto">
        <div className="mb-8">
          <h1 className="text-4xl font-bold mb-4">KNIRV Manager Interface</h1>
          <p className="text-gray-400">Unified management interface for KNIRV Controller</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div
            onClick={() => navigate('/manager/skills')}
            className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6 hover:bg-gray-700/80 transition-all duration-200 cursor-pointer"
          >
            <div className="flex items-center mb-4">
              <div className="w-12 h-12 bg-blue-600 rounded-lg flex items-center justify-center mr-4">
                <span className="text-2xl">🧠</span>
              </div>
              <h3 className="text-xl font-semibold">Skills Management</h3>
            </div>
            <p className="text-gray-400">Manage and configure AI skills and capabilities</p>
          </div>

          <div
            onClick={() => navigate('/manager/udc')}
            className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6 hover:bg-gray-700/80 transition-all duration-200 cursor-pointer"
          >
            <div className="flex items-center mb-4">
              <div className="w-12 h-12 bg-green-600 rounded-lg flex items-center justify-center mr-4">
                <span className="text-2xl">🔐</span>
              </div>
              <h3 className="text-xl font-semibold">UDC Management</h3>
            </div>
            <p className="text-gray-400">Universal Data Connector configuration and monitoring</p>
          </div>

          <div
            onClick={() => navigate('/manager/wallet')}
            className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6 hover:bg-gray-700/80 transition-all duration-200 cursor-pointer"
          >
            <div className="flex items-center mb-4">
              <div className="w-12 h-12 bg-purple-600 rounded-lg flex items-center justify-center mr-4">
                <span className="text-2xl">💰</span>
              </div>
              <h3 className="text-xl font-semibold">Wallet Operations</h3>
            </div>
            <p className="text-gray-400">Manage NRN tokens and wallet operations</p>
          </div>
        </div>

        <div className="mt-8 bg-gray-800/80 border border-gray-600/50 rounded-lg p-6">
          <h2 className="text-2xl font-semibold mb-4">System Status</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center">
              <div className="text-3xl font-bold text-green-400">Online</div>
              <div className="text-gray-400">System Status</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-blue-400">1,250</div>
              <div className="text-gray-400">NRN Balance</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-purple-400">3</div>
              <div className="text-gray-400">Active Agents</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Manager Interface Wrapper
const ManagerInterface = () => {


  return (
    <div className="min-h-screen bg-gray-900 text-white relative">
      <div className="absolute top-4 right-4 z-50">
        <NavigationButton to="/" className="bg-gray-800/80 hover:bg-gray-700/80 border border-gray-600/50">
          ← Receiver Interface
        </NavigationButton>
      </div>

      <Routes>
        <Route path="/" element={<SimpleManagerInterface />} />
        <Route path="/skills" element={<Skills />} />
        <Route path="/udc" element={<UDC />} />
        <Route path="/wallet" element={<WalletPage />} />
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