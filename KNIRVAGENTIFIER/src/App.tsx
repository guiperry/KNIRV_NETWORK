import React, { useState, useEffect, useRef } from 'react';
import { KnirvShell } from './components/KnirvShell';
import { VoiceControl } from './components/VoiceControl';
import { NetworkStatus } from './components/NetworkStatus';
import { NRVVisualization } from './components/NRVVisualization';
import { SlidingPanel } from './components/SlidingPanel';
import { EdgeColoring } from './components/EdgeColoring';
import { AgentManager } from './components/AgentManager';
import { FabricAlgorithm } from './components/FabricAlgorithm';
import { CognitiveShellInterface } from './components/CognitiveShellInterface';
import { CognitiveState } from './cognitive-shell/CognitiveEngine';

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
  type: 'KNIRV-AGENTIFIER' | 'KNIRVANA' | 'DVE';
  status: 'Available' | 'Busy' | 'Offline';
  specialization: string[];
  nrnCost: number;
}

function App() {
  const [shellStatus, setShellStatus] = useState<'idle' | 'processing' | 'listening' | 'error'>('idle');
  const [isVoiceActive, setIsVoiceActive] = useState(false);
  const [currentNRVs, setCurrentNRVs] = useState<NRV[]>([]);
  const [selectedNRV, setSelectedNRV] = useState<NRV | null>(null);
  const [availableAgents, setAvailableAgents] = useState<Agent[]>([]);
  const [activePanels, setActivePanels] = useState<string[]>([]);
  const [nrnBalance, setNrnBalance] = useState(1250);
  const [cognitiveMode, setCognitiveMode] = useState(false);
  const [cognitiveState, setCognitiveState] = useState<CognitiveState | null>(null);
  const [networkConnections, setNetworkConnections] = useState({
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
        type: 'KNIRV-AGENTIFIER',
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

    // Enhanced voice command processing with cognitive mode support
    setTimeout(() => {
      const lowerCommand = command.toLowerCase();

      if (lowerCommand.includes('identify problems')) {
        const newNRV: NRV = {
          id: `nrv-${Date.now()}`,
          problemDescription: `User reported issue: ${command}`,
          sourceID: 'KNIRV-AGENTIFIER-main',
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
    
    // Simulate screenshot analysis
    setTimeout(() => {
      const newNRV: NRV = {
        id: `nrv-${Date.now()}`,
        problemDescription: 'Visual anomaly detected in interface',
        sourceID: 'KNIRV-AGENTIFIER-main',
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
    
    // Open the agent manager panel
    setActivePanels(prev => 
      prev.includes('agent-manager') 
        ? prev 
        : [...prev, 'agent-manager']
    );
    
    // Simulate system analysis
    setTimeout(() => {
      const newNRV: NRV = {
        id: `nrv-${Date.now()}`,
        problemDescription: 'System performance degradation detected',
        sourceID: 'KNIRV-AGENTIFIER-main',
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

  const handleAnalyzeToggle = () => {
    setActivePanels(prev => 
      prev.includes('agent-manager') 
        ? prev.filter(id => id !== 'agent-manager')
        : [...prev, 'agent-manager']
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
      
      // Simulate resolution
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

  const handleSkillInvoked = (skillId: string, result: any) => {
    console.log('Skill invoked:', skillId, result);

    // Create NRV for skill invocation
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

  const handleAdaptationTriggered = (adaptation: any) => {
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

  return (
    <div className="min-h-screen bg-gray-900 text-white relative overflow-hidden">
      <EdgeColoring color={getEdgeColor()} intensity={shellStatus !== 'idle' ? 0.8 : 0.3} />
      
      <div ref={shellRef} className="relative w-full h-screen">
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
          <div className="absolute top-4 right-4 z-50">
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
      </div>
    </div>
  );
}

export default App;