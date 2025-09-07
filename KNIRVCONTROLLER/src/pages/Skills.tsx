import { Search, Zap, Shield, Wallet, Download, Activity } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import SkillCard from '../components/SkillCard';
import { SlidingPanel } from '../components/SlidingPanel';
import { NetworkStatus } from '../components/NetworkStatus';
import { AgentManager } from '../components/AgentManager';
import { CognitiveShellInterface } from '../components/CognitiveShellInterface';
import QRScanner from '../components/QRScanner';
import { Agent } from '../types/common';
// Temporarily comment out problematic imports for testing
// import { KNIRVRouterIntegration, LoRAAdapterData } from '../sensory-shell/KNIRVRouterIntegration';
// import { CognitiveEngine } from '../sensory-shell/CognitiveEngine';

// Temporary type definitions
interface LoRAAdapterData {
  id: string;
  name: string;
  networkScore: number;
  description: string;
  version: number;
  adapterId: string;
  adapterName: string;
  usageCount?: number;
}

interface LoRASkill {
  id: string;
  name: string;
  description: string;
  category: 'analysis' | 'automation' | 'computation' | 'communication';
  complexity: number;
  nrnCost: number;
  isActive: boolean;
  adapterId?: string;
  adapterData?: LoRAAdapterData;
  networkScore?: number;
  usageCount?: number;
}

export default function Skills() {
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const [activePanels, setActivePanels] = useState<string[]>([]);

  // Real LoRA adapter integration - temporarily disabled for testing
  // const [knirvRouter, setKnirvRouter] = useState<KNIRVRouterIntegration | null>(null);
  const [loraAdapters] = useState<LoRAAdapterData[]>([]);
  const [skills, setSkills] = useState<LoRASkill[]>([]);
  const [isLoading, setIsLoading] = useState(false); // Set to false to skip loading
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');

  // Mock data for slideouts
  const [networkConnections] = useState<{
    [key: string]: 'connected' | 'disconnected' | 'connecting';
  }>({
    knirvChain: 'connected',
    knirvGraph: 'connected',
    knirvNexus: 'connecting',
    knirvGateway: 'disconnected'
  });

  const [availableAgents] = useState<Agent[]>([
    {
      agentId: 'agent-1',
      name: 'CodeT5-Alpha',
      version: '1.0.0',
      type: 'wasm' as const,
      status: 'Available' as const,
      nrnCost: 85,
      capabilities: ['code-generation', 'optimization'],
      metadata: {
        name: 'CodeT5-Alpha',
        version: '1.0.0',
        description: 'Code generation agent',
        author: 'KNIRV Team',
        capabilities: ['code-generation', 'optimization'],
        requirements: {
          memory: 256,
          cpu: 2,
          storage: 50
        },
        permissions: ['code-execution', 'file-access']
      },
      createdAt: new Date().toISOString()
    },
    {
      agentId: 'agent-2',
      name: 'SEAL-Beta',
      version: '1.0.0',
      type: 'lora' as const,
      status: 'Available' as const,
      nrnCost: 90,
      capabilities: ['learning', 'adaptation'],
      metadata: {
        name: 'SEAL-Beta',
        version: '1.0.0',
        description: 'Learning and adaptation agent',
        author: 'KNIRV Team',
        capabilities: ['learning', 'adaptation'],
        requirements: {
          memory: 512,
          cpu: 4,
          storage: 100
        },
        permissions: ['learning', 'adaptation']
      },
      createdAt: new Date().toISOString()
    }
  ]);

  const [currentNRVs] = useState([]);
  const [selectedNRV] = useState(null);
  const [nrnBalance] = useState(1250);

  // Initialize KNIRVROUTER and Cognitive Engine - temporarily disabled for testing
  useEffect(() => {
    const initializeIntegrations = async () => {
      try {
        // Temporarily comment out problematic initialization
        /*
        // Initialize KNIRVROUTER
        const router = new KNIRVRouterIntegration({
          routerUrl: 'http://localhost:5000',
          graphUrl: 'http://localhost:5001',
          oracleUrl: 'http://localhost:5002',
          timeout: 30000,
          retryAttempts: 3,
          enableP2P: true,
          enableWASM: true
        });

        setKnirvRouter(router);

        // Initialize Cognitive Engine
        const engine = new CognitiveEngine({
          maxContextSize: 100,
          learningRate: 0.01,
          adaptationThreshold: 0.3,
          skillTimeout: 30000,
          voiceEnabled: true,
          visualEnabled: true,
          loraEnabled: true,
          enhancedLoraEnabled: true,
          hrmEnabled: true,
          adaptiveLearningEnabled: true,
          walletIntegrationEnabled: true,
          chainIntegrationEnabled: true,
          ecosystemCommunicationEnabled: true,
        });

        setCognitiveEngine(engine);

        // Load LoRA adapters from network
        await loadLoRAAdapters(router);
        */

        // Load default skills for testing
        setSkills(getDefaultSkills());

      } catch (error) {
        console.error('Failed to initialize integrations:', error);
      } finally {
        setIsLoading(false);
      }
    };

    initializeIntegrations();
  }, []); // Remove loadLoRAAdapters dependency to fix initialization error


  // Helper functions

  const getDefaultSkills = (): LoRASkill[] => [
    {
      id: 'default-code-analysis',
      name: 'Code Analysis',
      description: 'Automated code review and optimization using advanced pattern recognition',
      category: 'analysis' as const,
      complexity: 8,
      nrnCost: 25,
      isActive: true
    },
    {
      id: 'default-task-orchestration',
      name: 'Task Orchestration',
      description: 'Intelligent workflow automation across multiple systems and platforms',
      category: 'automation' as const,
      complexity: 7,
      nrnCost: 30,
      isActive: true
    },
    {
      id: 'default-neural-synthesis',
      name: 'Neural Synthesis',
      description: 'Advanced data processing and pattern synthesis for complex computations',
      category: 'computation' as const,
      complexity: 9,
      nrnCost: 45,
      isActive: false
    },
    {
      id: 'default-agent-communication',
      name: 'Agent Communication',
      description: 'Secure inter-agent messaging and coordination protocols',
      category: 'communication' as const,
      complexity: 6,
      nrnCost: 20,
      isActive: true
    },
    {
      id: 'default-predictive-modeling',
      name: 'Predictive Modeling',
      description: 'Real-time prediction and forecasting using machine learning algorithms',
      category: 'analysis' as const,
      complexity: 8,
      nrnCost: 35,
      isActive: false
    },
    {
      id: 'default-resource-optimization',
      name: 'Resource Optimization',
      description: 'Dynamic resource allocation and performance tuning for optimal efficiency',
      category: 'automation' as const,
      complexity: 7,
      nrnCost: 28,
      isActive: false
    }
  ];


  // Filter skills based on search and category
  const filteredSkills = skills.filter(skill => {
    const matchesSearch = skill.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         skill.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || skill.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const activeSkills = skills.filter(skill => skill.isActive).length;
  const totalNrnCost = skills.filter(skill => skill.isActive).reduce((sum, skill) => sum + skill.nrnCost, 0);

  // Panel management functions
  const closePanel = (panelId: string) => {
    setActivePanels(prev => prev.filter(id => id !== panelId));
  };

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
      prev.includes('agent-management')
        ? prev.filter(id => id !== 'agent-management')
        : [...prev, 'agent-management']
    );
    setMenuOpen(false);
  };

  const handleQRScan = () => {
    setActivePanels(prev =>
      prev.includes('qr-scanner')
        ? prev.filter(id => id !== 'qr-scanner')
        : [...prev, 'qr-scanner']
    );
    setMenuOpen(false);
  };

  const handleCognitiveStateChange = (state: unknown) => {
    console.log('Cognitive state changed:', state);
  };

  const handleSkillInvoked = (skillId: string, result: unknown) => {
    console.log('Skill invoked:', skillId, result);
  };

  const handleAdaptationTriggered = (adaptation: unknown) => {
    console.log('Adaptation triggered:', adaptation);
  };

  const handleAgentAssignment = (nrv: unknown, agent: unknown) => {
    console.log('Agent assigned:', agent, 'to NRV:', nrv);
  };

  // Burger Menu Component
  const BurgerMenu = ({ isOpen, onToggle, children }: {
    isOpen: boolean;
    onToggle: () => void;
    children: React.ReactNode
  }) => {
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

        {/* Menu Items */}
        {isOpen && (
          <div className="absolute top-full right-0 mt-2 w-64 bg-gray-800/95 backdrop-blur-xl rounded-lg shadow-xl border border-gray-600/50 py-2 z-50">
            {children}
          </div>
        )}
      </div>
    );
  };

  // Menu Item Component
  const MenuItem = ({ onClick, icon, children }: {
    onClick: () => void;
    icon: React.ReactNode;
    children: React.ReactNode
  }) => {
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
          <MenuItem onClick={() => { navigate('/'); setMenuOpen(false); }} icon="🏠">
            Input Interface
          </MenuItem>
        </BurgerMenu>
      </div>

      <div className="max-w-6xl mx-auto p-4 pb-24 overflow-y-auto h-screen">
        <div className="space-y-6">
          {/* Header */}
          <div className="text-center py-4">
            <h1 className="text-2xl font-bold bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent mb-2">
              Skills
            </h1>
            <p className="text-gray-400 text-sm">
              Manage and configure your AI agent capabilities
            </p>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-2xl font-bold text-white">{activeSkills}</div>
              <div className="text-xs text-gray-400">Active Skills</div>
            </div>
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-2xl font-bold text-blue-400">{totalNrnCost}</div>
              <div className="text-xs text-gray-400">NRN/Hour</div>
            </div>
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-2xl font-bold text-cyan-400">{skills.length}</div>
              <div className="text-xs text-gray-400">Available</div>
            </div>
          </div>

          {/* Search and Filter */}
          <div className="space-y-3">
            <div className="flex space-x-3">
              <div className="flex-1 relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
                <input
                  type="text"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  placeholder="Search LoRA adapters and skills..."
                  className="w-full pl-10 pr-4 py-3 bg-gray-800/80 border border-gray-600/50 rounded-lg focus:border-blue-500/50 focus:outline-none text-white placeholder-gray-400"
                />
              </div>
              <select
                value={selectedCategory}
                onChange={(e) => setSelectedCategory(e.target.value)}
                className="px-4 py-3 bg-gray-800/80 border border-gray-600/50 rounded-lg focus:border-blue-500/50 focus:outline-none text-white"
              >
                <option value="all">All Categories</option>
                <option value="analysis">Analysis</option>
                <option value="automation">Automation</option>
                <option value="computation">Computation</option>
                <option value="communication">Communication</option>
              </select>
              <button
                onClick={() => {/* loadLoRAAdapters(knirvRouter!) */}}
                disabled={true /* !knirvRouter */}
                className="px-4 py-3 bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 rounded-lg text-blue-400 hover:text-blue-300 transition-all disabled:opacity-50"
              >
                <Download className="w-4 h-4" />
              </button>
            </div>

            {/* LoRA Adapter Status */}
            <div className="flex items-center justify-between p-3 bg-gray-800/50 rounded-lg border border-gray-600/30">
              <div className="flex items-center space-x-3">
                <Activity className="w-4 h-4 text-purple-400" />
                <span className="text-sm text-white">LoRA Network Status</span>

              </div>
              <div className="flex items-center space-x-4 text-xs text-gray-400">
                <span>Adapters: {loraAdapters.length}</span>
                <span>Active: {activeSkills}</span>
                <span>Network Score: {loraAdapters.length > 0 ? (loraAdapters.reduce((sum, a) => sum + a.networkScore, 0) / loraAdapters.length).toFixed(2) : '0.00'}</span>
              </div>
            </div>
          </div>

          {/* Skills Grid */}
          <div className="space-y-4">
            {isLoading ? (
              <div className="text-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-400 mx-auto mb-4"></div>
                <p className="text-gray-400">Loading LoRA adapters from KNIRVROUTER network...</p>
              </div>
            ) : filteredSkills.length === 0 ? (
              <div className="text-center py-8">
                <Zap className="w-12 h-12 mx-auto mb-4 text-gray-500 opacity-50" />
                <p className="text-gray-400 mb-2">No skills found</p>
                <p className="text-sm text-gray-500">
                  {searchTerm || selectedCategory !== 'all'
                    ? 'Try adjusting your search or filter criteria'
                    : 'Connect to KNIRVROUTER network to load LoRA adapters'
                  }
                </p>
              </div>
            ) : (
              filteredSkills.map((skill) => (
                <div key={skill.id} className="relative">
                  <SkillCard
                    name={skill.name}
                    description={skill.description}
                    category={skill.category}
                    complexity={skill.complexity}
                    nrnCost={skill.nrnCost}
                    isActive={skill.isActive}
                  />
                  {skill.adapterData && (
                    <div className="absolute top-2 right-2 flex space-x-1">
                      {skill.adapterData.networkScore > 0.8 && (
                        <div className="px-2 py-1 bg-green-500/20 text-green-400 text-xs rounded">
                          High Score
                        </div>
                      )}
                      <div className="px-2 py-1 bg-purple-500/20 text-purple-400 text-xs rounded">
                        LoRA
                      </div>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>

          {/* Install New Skills */}
          <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6 text-center">
            <h3 className="text-lg font-semibold text-white mb-2">Discover New Skills</h3>
            <p className="text-sm text-gray-400 mb-4">
              Browse the KNIRV marketplace for cutting-edge AI capabilities
            </p>
            <button className="px-6 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-white font-medium transition-all">
              Browse Marketplace
            </button>
          </div>
        </div>
      </div>

      {/* Bottom Navigation */}
      <nav className="fixed bottom-0 left-0 right-0 z-20 border-t border-gray-600/50 backdrop-blur-xl bg-gray-900/80">
        <div className="grid grid-cols-3 px-2 py-2">
          <button
            onClick={() => navigate('/manager/skills')}
            className={`flex flex-col items-center py-2 px-1 rounded-lg transition-colors ${
              window.location.pathname === '/manager/skills' ? 'text-blue-400 bg-blue-600/20' : 'text-gray-400 hover:text-white'
            }`}
          >
            <Zap className="w-5 h-5 mb-1" />
            <span className="text-xs">Skills</span>
          </button>
          <button
            onClick={() => navigate('/manager/udc')}
            className={`flex flex-col items-center py-2 px-1 rounded-lg transition-colors ${
              window.location.pathname === '/manager/udc' ? 'text-blue-400 bg-blue-600/20' : 'text-gray-400 hover:text-white'
            }`}
          >
            <Shield className="w-5 h-5 mb-1" />
            <span className="text-xs">UDC</span>
          </button>
          <button
            onClick={() => navigate('/manager/wallet')}
            className={`flex flex-col items-center py-2 px-1 rounded-lg transition-colors ${
              window.location.pathname === '/manager/wallet' ? 'text-blue-400 bg-blue-600/20' : 'text-gray-400 hover:text-white'
            }`}
          >
            <Wallet className="w-5 h-5 mb-1" />
            <span className="text-xs">Wallet</span>
          </button>
        </div>
      </nav>

      {/* Sliding Panels */}
      <SlidingPanel
        id="qr-scanner"
        isOpen={activePanels.includes('qr-scanner')}
        onClose={() => closePanel('qr-scanner')}
        title="QR Scanner"
        side="right"
      >
        <QRScanner
          onScan={(result) => console.log('QR Result:', result)}
          onClose={() => closePanel('qr-scanner')}
          isOpen={activePanels.includes('qr-scanner')}
        />
      </SlidingPanel>

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
        id="agent-management"
        isOpen={activePanels.includes('agent-management')}
        onClose={() => closePanel('agent-management')}
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
    </div>
  );
}
