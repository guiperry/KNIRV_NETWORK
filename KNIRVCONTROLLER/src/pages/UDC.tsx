import { Shield, Key, CheckCircle, AlertTriangle, RefreshCw, Wallet, Clock, Zap} from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useState } from 'react';
import { SlidingPanel } from '../components/SlidingPanel';
import { NetworkStatus } from '../components/NetworkStatus';
import { AgentManager } from '../components/AgentManager';
import { CognitiveShellInterface } from '../components/CognitiveShellInterface';
import QRScanner from '../components/QRScanner';

export default function UDC() {
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const [activePanels, setActivePanels] = useState<string[]>([]);

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
        description: 'Code generation and optimization agent',
        author: 'KNIRV Labs',
        capabilities: ['code-generation', 'optimization'],
        requirements: {
          memory: 256,
          cpu: 2,
          storage: 1024
        },
        permissions: ['execute', 'read', 'write']
      },
      createdAt: '2024-01-01T00:00:00.000Z'
    },
    {
      agentId: 'agent-2',
      name: 'SEAL-Beta',
      version: '1.0.0',
      type: 'wasm' as const,
      status: 'Available' as const,
      nrnCost: 90,
      capabilities: ['learning', 'adaptation'],
      metadata: {
        name: 'SEAL-Beta',
        version: '1.0.0',
        description: 'Learning and adaptation agent',
        author: 'KNIRV Labs',
        capabilities: ['learning', 'adaptation'],
        requirements: {
          memory: 512,
          cpu: 4,
          storage: 2048
        },
        permissions: ['execute', 'read', 'write']
      },
      createdAt: '2024-01-02T00:00:00.000Z'
    }
  ]);

  const [currentNRVs] = useState([]);
  const [selectedNRV] = useState(null);
  const [nrnBalance] = useState(1250);

  const udc = {
    id: 'UDC-7A8B9C2D',
    status: 'valid' as const,
    issuedAt: '2024-08-01T10:30:00Z',
    expiresAt: '2024-08-08T10:30:00Z',
    permissions: [
      'agent.deploy',
      'skill.activate',
      'nrn.transfer',
      'dten.access',
      'wallet.connect'
    ]
  };

  const daysUntilExpiry = Math.ceil((new Date(udc.expiresAt).getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24));
  const isExpiringSoon = daysUntilExpiry <= 2;

  const statusConfig = {
    valid: { icon: CheckCircle, color: 'text-green-400', bg: 'bg-green-500/20', border: 'border-green-500/30' },
    expired: { icon: AlertTriangle, color: 'text-red-400', bg: 'bg-red-500/20', border: 'border-red-500/30' },
    revoked: { icon: AlertTriangle, color: 'text-red-400', bg: 'bg-red-500/20', border: 'border-red-500/30' },
    pending: { icon: Clock, color: 'text-yellow-400', bg: 'bg-yellow-500/20', border: 'border-yellow-500/30' }
  };

  const config = statusConfig[udc.status];
  const StatusIcon = config.icon;

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
    // This function is called by CognitiveShellInterface but we don't need to store the state
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
  interface BurgerMenuProps {
    isOpen: boolean;
    onToggle: () => void;
    children: React.ReactNode;
  }
  
  const BurgerMenu = ({ isOpen, onToggle, children }: BurgerMenuProps) => {
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
  interface MenuItemProps {
    onClick: () => void;
    icon: React.ReactNode;
    children: React.ReactNode;
  }
  
  const MenuItem = ({ onClick, icon, children }: MenuItemProps) => {
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
              User Delegation Certificate
            </h2>
            <p className="text-gray-400 text-sm">
              Your authorized access credentials for the D-TEN network
            </p>
          </div>

          {/* Certificate Status */}
          <div className={`bg-gray-800/80 border rounded-lg p-6 ${isExpiringSoon ? 'border-red-500/30' : 'border-green-500/30'}`}>
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center space-x-3">
                <div className={`w-12 h-12 ${config.bg} ${config.border} border rounded-xl flex items-center justify-center`}>
                  <StatusIcon className={`w-6 h-6 ${config.color}`} />
                </div>
                <div>
                  <h3 className="text-xl font-bold text-white">Certificate Active</h3>
                  <p className={`text-sm ${config.color} capitalize`}>{udc.status}</p>
                </div>
              </div>
              
              {isExpiringSoon && (
                <div className="px-3 py-1 rounded-full bg-red-500/20 border border-red-500/30">
                  <span className="text-xs text-red-400 font-medium">Expires Soon</span>
                </div>
              )}
            </div>

            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-gray-400">Certificate ID</p>
                  <p className="text-sm font-mono text-white">{udc.id}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Expires In</p>
                  <p className={`text-sm font-semibold ${isExpiringSoon ? 'text-red-400' : 'text-green-400'}`}>
                    {daysUntilExpiry} days
                  </p>
                </div>
              </div>

              <div>
                <p className="text-sm text-gray-400 mb-2">Valid Until</p>
                <p className="text-sm text-white">
                  {new Date(udc.expiresAt).toLocaleDateString()} at {new Date(udc.expiresAt).toLocaleTimeString()}
                </p>
              </div>
            </div>
          </div>

          {/* Permissions */}
          <div>
            <h3 className="text-lg font-semibold text-white mb-4">Granted Permissions</h3>
            <div className="space-y-3">
              {udc.permissions.map((permission, index) => (
                <div key={index} className="flex items-center justify-between p-3 bg-gray-800/80 border border-gray-600/50 rounded-lg">
                  <div className="flex items-center space-x-3">
                    <div className="w-8 h-8 bg-blue-500/20 rounded-lg flex items-center justify-center border border-blue-500/20">
                      <Key className="w-4 h-4 text-blue-400" />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-white">{permission}</p>
                      <p className="text-xs text-gray-400">Full access granted</p>
                    </div>
                  </div>
                  <CheckCircle className="w-5 h-5 text-green-400" />
                </div>
              ))}
            </div>
          </div>

          {/* Actions */}
          <div className="space-y-3">
            <button className="w-full flex items-center justify-center space-x-3 py-4 bg-blue-600 hover:bg-blue-700 rounded-lg text-white font-semibold transition-all">
              <RefreshCw className="w-5 h-5" />
              <span>Renew Certificate</span>
            </button>

            <div className="grid grid-cols-2 gap-3">
              <button className="flex items-center justify-center space-x-2 py-3 bg-gray-800/80 border border-gray-600/50 rounded-lg hover:border-blue-500/50 text-gray-300 hover:text-white transition-all">
                <Shield className="w-4 h-4" />
                <span className="text-sm">View Details</span>
              </button>
              <button className="flex items-center justify-center space-x-2 py-3 bg-gray-800/80 border border-gray-600/50 rounded-lg hover:border-blue-500/50 text-gray-300 hover:text-white transition-all">
                <Key className="w-4 h-4" />
                <span className="text-sm">Export Key</span>
              </button>
            </div>
          </div>

          {/* Certificate Chain */}
          <div>
            <h3 className="text-lg font-semibold text-white mb-4">Certificate Chain</h3>
            <div className="space-y-2">
              <div className="flex items-center space-x-3 p-3 bg-gray-800/30 border border-gray-600/30 rounded-lg">
                <div className="w-2 h-2 bg-green-400 rounded-full"></div>
                <span className="text-sm text-gray-300">KNIRV Root CA</span>
              </div>
              <div className="flex items-center space-x-3 p-3 bg-gray-800/30 border border-gray-600/30 rounded-lg ml-4">
                <div className="w-2 h-2 bg-green-400 rounded-full"></div>
                <span className="text-sm text-gray-300">D-TEN Intermediate CA</span>
              </div>
              <div className="flex items-center space-x-3 p-3 bg-gray-800/30 border border-gray-600/30 rounded-lg ml-8">
                <div className="w-2 h-2 bg-green-400 rounded-full"></div>
                <span className="text-sm text-gray-300">User Certificate</span>
              </div>
            </div>
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
