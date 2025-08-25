import { Search, Filter, Plus, Cpu, Zap, Shield, Wallet, QrCode, X } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useState } from 'react';
import SkillCard from '@components/SkillCard';
import { SlidingPanel } from '@components/SlidingPanel';
import { NetworkStatus } from '@components/NetworkStatus';
import { AgentManager } from '@components/AgentManager';
import { CognitiveShellInterface } from '@components/CognitiveShellInterface';
import QRScanner from '@components/QRScanner';

export default function Skills() {
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const [showQRScanner, setShowQRScanner] = useState(false);
  const [activePanels, setActivePanels] = useState<string[]>([]);
  const [cognitiveMode, setCognitiveMode] = useState(false);
  const [cognitiveState, setCognitiveState] = useState<any>(null);

  // Mock data for slideouts
  const [networkConnections] = useState<{
    [key: string]: 'connected' | 'disconnected' | 'connecting';
  }>({
    knirvChain: 'connected',
    knirvGraph: 'connected',
    knirvNexus: 'connecting',
    knirvGateway: 'disconnected'
  });

  const [availableAgents] = useState([
    {
      id: 'agent-1',
      name: 'CodeT5-Alpha',
      type: 'KNIRV-CORTEX',
      status: 'Available',
      specialization: ['code-generation', 'optimization'],
      nrnCost: 85
    },
    {
      id: 'agent-2',
      name: 'SEAL-Beta',
      type: 'KNIRVANA',
      status: 'Available',
      specialization: ['learning', 'adaptation'],
      nrnCost: 90
    }
  ]);

  const [currentNRVs] = useState([]);
  const [selectedNRV, setSelectedNRV] = useState(null);
  const [nrnBalance] = useState(1250);

  const skills = [
    {
      name: 'Code Analysis',
      description: 'Automated code review and optimization using advanced pattern recognition',
      category: 'analysis' as const,
      complexity: 8,
      nrnCost: 25,
      isActive: true
    },
    {
      name: 'Task Orchestration',
      description: 'Intelligent workflow automation across multiple systems and platforms',
      category: 'automation' as const,
      complexity: 7,
      nrnCost: 30,
      isActive: true
    },
    {
      name: 'Neural Synthesis',
      description: 'Advanced data processing and pattern synthesis for complex computations',
      category: 'computation' as const,
      complexity: 9,
      nrnCost: 45,
      isActive: false
    },
    {
      name: 'Agent Communication',
      description: 'Secure inter-agent messaging and coordination protocols',
      category: 'communication' as const,
      complexity: 6,
      nrnCost: 20,
      isActive: true
    },
    {
      name: 'Predictive Modeling',
      description: 'Real-time prediction and forecasting using machine learning algorithms',
      category: 'analysis' as const,
      complexity: 8,
      nrnCost: 35,
      isActive: false
    },
    {
      name: 'Resource Optimization',
      description: 'Dynamic resource allocation and performance tuning for optimal efficiency',
      category: 'automation' as const,
      complexity: 7,
      nrnCost: 28,
      isActive: false
    }
  ];

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

  const handleCognitiveStateChange = (state: any) => {
    setCognitiveState(state);
    setCognitiveMode(state.status === 'active' || state.status === 'learning');
  };

  const handleSkillInvoked = (skillId: string, result: any) => {
    console.log('Skill invoked:', skillId, result);
  };

  const handleAdaptationTriggered = (adaptationType: string) => {
    console.log('Adaptation triggered:', adaptationType);
  };

  const handleAgentAssignment = (nrv: any, agent: any) => {
    console.log('Agent assigned:', agent, 'to NRV:', nrv);
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
            <h2 className="text-2xl font-bold bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent mb-2">
              Agent Skills
            </h2>
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
          <div className="flex space-x-3">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search skills..."
                className="w-full pl-10 pr-4 py-3 bg-gray-800/80 border border-gray-600/50 rounded-lg focus:border-blue-500/50 focus:outline-none text-white placeholder-gray-400"
              />
            </div>
            <button className="px-4 py-3 bg-gray-800/80 border border-gray-600/50 rounded-lg hover:border-blue-500/50 text-gray-400 hover:text-blue-400 transition-all">
              <Filter className="w-4 h-4" />
            </button>
            <button className="px-4 py-3 bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 rounded-lg text-blue-400 hover:text-blue-300 transition-all">
              <Plus className="w-4 h-4" />
            </button>
          </div>

          {/* Skills Grid */}
          <div className="space-y-4">
            {skills.map((skill, index) => (
              <SkillCard key={index} {...skill} />
            ))}
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
