import { Wallet, ArrowUpRight, ArrowDownLeft, Zap, TrendingUp, Copy, Shield, QrCode, X, CheckCircle } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { SlidingPanel } from '@components/SlidingPanel';
import { NetworkStatus } from '@components/NetworkStatus';
import { AgentManager } from '@components/AgentManager';
import { CognitiveShellInterface } from '@components/CognitiveShellInterface';
import QRScanner from '@components/QRScanner';
import { apiService, isAPIError, getErrorMessage } from '../services/APIService';
import { webSocketService, subscribeToWallet } from '../services/WebSocketService';

export default function WalletPage() {
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const [activePanels, setActivePanels] = useState<string[]>([]);
  const [showQRCode, setShowQRCode] = useState(false);
  const [showAddFunds, setShowAddFunds] = useState(false);
  const [showSendNRN, setShowSendNRN] = useState(false);
  const [copySuccess, setCopySuccess] = useState(false);
  const [cognitiveState, setCognitiveState] = useState<{ mode?: string; isActive?: boolean; timestamp?: number } | null>(null);
  const [cognitiveMode, setCognitiveMode] = useState(false);

  // Wallet operation states
  const [addFundsLoading, setAddFundsLoading] = useState(false);
  const [sendNRNLoading, setSendNRNLoading] = useState(false);
  const [addFundsError, setAddFundsError] = useState<string | null>(null);
  const [sendNRNError, setSendNRNError] = useState<string | null>(null);
  const [addFundsSuccess, setAddFundsSuccess] = useState(false);
  const [sendNRNSuccess, setSendNRNSuccess] = useState(false);

  // Form states
  const [addFundsAmount, setAddFundsAmount] = useState('');
  const [addFundsMethod, setAddFundsMethod] = useState('Credit Card');
  const [sendNRNAddress, setSendNRNAddress] = useState('');
  const [sendNRNAmount, setSendNRNAmount] = useState('');
  const [sendNRNNote, setSendNRNNote] = useState('');

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
      capabilities: ['code-generation', 'optimization'],
      nrnCost: 85,
      metadata: {
        name: 'CodeT5-Alpha',
        version: '1.0.0',
        description: 'Code generation and optimization agent',
        author: 'KNIRV Network',
        capabilities: ['code-generation', 'optimization'],
        requirements: { memory: 512, cpu: 2, storage: 100 },
        permissions: ['read', 'write']
      },
      createdAt: new Date().toISOString()
    },
    {
      agentId: 'agent-2',
      name: 'SEAL-Beta',
      version: '1.0.0',
      type: 'lora' as const,
      status: 'Available' as const,
      capabilities: ['learning', 'adaptation'],
      nrnCost: 90,
      metadata: {
        name: 'SEAL-Beta',
        version: '1.0.0',
        description: 'Learning and adaptation agent',
        author: 'KNIRV Network',
        capabilities: ['learning', 'adaptation'],
        requirements: { memory: 256, cpu: 1, storage: 50 },
        permissions: ['read']
      },
      createdAt: new Date().toISOString()
    }
  ]);

  const [currentNRVs] = useState([]);
  const [selectedNRV] = useState(null);
  const [nrnBalance] = useState(1250);

  const walletData = {
    nrnBalance: 1247,
    usdValue: 312.75,
    change24h: 5.2,
    walletAddress: '0x742d35Cc6aa34567...8B9fA2e1C4D'
  };

  // WebSocket subscription
  useEffect(() => {
    const connectWebSocket = async () => {
      const connected = await webSocketService.connect();
      if (connected) {
        // Subscribe to wallet updates
        subscribeToWallet(walletData.walletAddress);
      }
    };

    connectWebSocket();

    return () => {
      webSocketService.disconnect();
    };
  }, [walletData.walletAddress]);

  const transactions = [
    {
      id: '1',
      type: 'consumption' as const,
      amount: -25,
      description: 'Code Analysis Skill',
      timestamp: '2024-08-06T01:15:00Z',
      agentName: 'CodeT5-Alpha'
    },
    {
      id: '2',
      type: 'reward' as const,
      amount: 50,
      description: 'Task completion bonus',
      timestamp: '2024-08-06T00:45:00Z',
      agentName: 'SEAL-Beta'
    },
    {
      id: '3',
      type: 'consumption' as const,
      amount: -30,
      description: 'Task Orchestration',
      timestamp: '2024-08-05T23:20:00Z',
      agentName: 'CodeT5-Alpha'
    },
    {
      id: '4',
      type: 'transfer' as const,
      amount: 100,
      description: 'Wallet funding',
      timestamp: '2024-08-05T22:10:00Z',
      agentName: null
    }
  ];

  // Panel management functions
  const closePanel = (panelId: string) => {
    setActivePanels(prev => prev.filter(id => id !== panelId));
  };

  const openCognitiveShell = () => {
    // Toggle cognitive mode when opening shell
    setCognitiveMode(true);
    setCognitiveState({
      isActive: true,
      timestamp: Date.now(),
      mode: 'interactive'
    });

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

  // Wallet functionality handlers
  const handleCopyAddress = async () => {
    const walletAddress = walletData.walletAddress;
    try {
      await navigator.clipboard.writeText(walletAddress);
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    } catch (err) {
      console.error('Failed to copy address:', err);
    }
  };

  const handleShowQRCode = () => {
    setShowQRCode(true);
  };

  const handleAddFunds = () => {
    setShowAddFunds(true);
  };

  const handleSendNRN = () => {
    setShowSendNRN(true);
  };

  // Add funds functionality
  const executeAddFunds = async () => {
    if (!addFundsAmount || parseFloat(addFundsAmount) <= 0) {
      setAddFundsError('Please enter a valid amount');
      return;
    }

    setAddFundsLoading(true);
    setAddFundsError(null);

    try {
      const response = await apiService.post('/wallet/add-funds', {
        amount: addFundsAmount,
        paymentMethod: addFundsMethod,
        currency: 'USD'
      });

      if (isAPIError(response)) {
        setAddFundsError(getErrorMessage(response));
        return;
      }

      setAddFundsSuccess(true);
      setTimeout(() => {
        setShowAddFunds(false);
        setAddFundsSuccess(false);
        setAddFundsAmount('');
      }, 2000);

    } catch {
      setAddFundsError('Failed to add funds. Please try again.');
    } finally {
      setAddFundsLoading(false);
    }
  };

  // Send NRN functionality
  const executeSendNRN = async () => {
    if (!sendNRNAddress || !sendNRNAmount || parseFloat(sendNRNAmount) <= 0) {
      setSendNRNError('Please provide recipient address and valid amount');
      return;
    }

    if (parseFloat(sendNRNAmount) > walletData.nrnBalance) {
      setSendNRNError('Insufficient balance');
      return;
    }

    setSendNRNLoading(true);
    setSendNRNError(null);

    try {
      const response = await apiService.post('/wallet/send-transaction', {
        recipient: sendNRNAddress,
        amount: sendNRNAmount,
        note: sendNRNNote || undefined
      });

      if (isAPIError(response)) {
        setSendNRNError(getErrorMessage(response));
        return;
      }

      setSendNRNSuccess(true);
      setTimeout(() => {
        setShowSendNRN(false);
        setSendNRNSuccess(false);
        setSendNRNAddress('');
        setSendNRNAmount('');
        setSendNRNNote('');
      }, 2000);

    } catch {
      setSendNRNError('Failed to send NRN. Please try again.');
    } finally {
      setSendNRNLoading(false);
    }
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
    setCognitiveState(state as { mode?: string; isActive?: boolean; timestamp?: number } | null);
    if (state && typeof state === 'object' && 'status' in state) {
      const stateObj = state as { status: string };
      setCognitiveMode(stateObj.status === 'active' || stateObj.status === 'learning');
    }
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
  const BurgerMenu = ({ isOpen, onToggle, children }: { isOpen: boolean; onToggle: () => void; children: React.ReactNode }) => {
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
  const MenuItem = ({ onClick, icon, children }: { onClick: () => void; icon: React.ReactNode; children: React.ReactNode }) => {
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
              KNIRV Wallet
            </h2>
            <p className="text-gray-400 text-sm">
              Manage your NRN tokens and transaction history
            </p>
          </div>

          {/* Balance Card */}
          <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center space-x-3">
                <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-cyan-500 rounded-xl flex items-center justify-center">
                  <Zap className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-white">NRN Balance</h3>
                  <p className="text-sm text-gray-400">Neural Resource Network</p>
                </div>
              </div>
              
              <div className={walletData.change24h >= 0 ? 'px-3 py-1 rounded-full bg-green-500/20 border border-green-500/30' : 'px-3 py-1 rounded-full bg-red-500/20 border border-red-500/30'}>
                <div className="flex items-center space-x-1">
                  <TrendingUp className={walletData.change24h >= 0 ? 'w-3 h-3 text-green-400' : 'w-3 h-3 text-red-400 rotate-180'} />
                  <span className={walletData.change24h >= 0 ? 'text-xs font-medium text-green-400' : 'text-xs font-medium text-red-400'}>
                    {walletData.change24h >= 0 ? '+' : ''}{walletData.change24h}%
                  </span>
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <div className="text-3xl font-bold text-white">
                {walletData.nrnBalance.toLocaleString()} NRN
              </div>
              <div className="text-lg text-gray-300">
                ≈ ${walletData.usdValue.toFixed(2)} USD
              </div>

              {/* Cognitive Status Indicator */}
              {cognitiveMode && (
                <div className="flex items-center space-x-2 mt-2 px-2 py-1 bg-purple-600/20 border border-purple-500/30 rounded-lg">
                  <div className="w-2 h-2 bg-purple-400 rounded-full animate-pulse"></div>
                  <span className="text-xs text-purple-400">
                    Cognitive Mode: {cognitiveState?.mode || 'active'} |
                    Status: {cognitiveState?.isActive ? 'online' : 'offline'}
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Wallet Address */}
          <div className="space-y-3">
            <h3 className="text-lg font-semibold text-white">Wallet Address</h3>
            <div className="flex items-center space-x-3 p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center border border-blue-500/20">
                <Wallet className="w-5 h-5 text-blue-400" />
              </div>
              <div className="flex-1">
                <p className="font-mono text-sm text-white">{walletData.walletAddress}</p>
                <p className="text-xs text-gray-400">KNIRV Network</p>
              </div>
              <div className="flex space-x-2">
                <button
                  onClick={handleCopyAddress}
                  className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
                  title="Copy Address"
                >
                  {copySuccess ? <CheckCircle className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4" />}
                </button>
                <button
                  onClick={handleShowQRCode}
                  className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
                  title="Show QR Code"
                >
                  <QrCode className="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>

          {/* Quick Actions */}
          <div className="grid grid-cols-2 gap-3">
            <button
              onClick={handleAddFunds}
              className="flex items-center justify-center space-x-3 py-4 bg-green-600/20 hover:bg-green-600/30 border border-green-500/30 rounded-lg text-green-400 hover:text-green-300 transition-all"
            >
              <ArrowDownLeft className="w-5 h-5" />
              <span className="font-medium">Add Funds</span>
            </button>
            <button
              onClick={handleSendNRN}
              className="flex items-center justify-center space-x-3 py-4 bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 rounded-lg text-blue-400 hover:text-blue-300 transition-all"
            >
              <ArrowUpRight className="w-5 h-5" />
              <span className="font-medium">Send NRN</span>
            </button>
          </div>

          {/* Transaction History */}
          <div>
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Recent Transactions</h3>
              <button className="text-sm text-blue-400 hover:text-blue-300 transition-colors">
                View All
              </button>
            </div>

            <div className="space-y-3">
              {transactions.map((tx) => (
                <TransactionItem key={tx.id} {...tx} />
              ))}
            </div>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-xl font-bold text-green-400">+127</div>
              <div className="text-xs text-gray-400">Earned Today</div>
            </div>
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-xl font-bold text-red-400">-89</div>
              <div className="text-xs text-gray-400">Spent Today</div>
            </div>
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-xl font-bold text-blue-400">15</div>
              <div className="text-xs text-gray-400">Transactions</div>
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

      {/* QR Code Modal */}
      {showQRCode && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 max-w-sm w-full">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Wallet QR Code</h3>
              <button
                onClick={() => setShowQRCode(false)}
                className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            <div className="bg-white p-4 rounded-lg mb-4">
              <div className="w-48 h-48 mx-auto bg-gray-200 rounded-lg flex items-center justify-center">
                <QrCode className="w-24 h-24 text-gray-600" />
              </div>
            </div>
            <p className="text-sm text-gray-400 text-center">
              Scan this QR code to send funds to your wallet
            </p>
            <p className="text-xs text-gray-500 text-center mt-2 font-mono break-all">
              {walletData.walletAddress}
            </p>
          </div>
        </div>
      )}

      {/* Add Funds Modal */}
      {showAddFunds && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 max-w-md w-full">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Add Funds</h3>
              <button
                onClick={() => setShowAddFunds(false)}
                className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
                disabled={addFundsLoading}
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            <div className="space-y-4">
              {addFundsError && (
                <div className="p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
                  <p className="text-sm text-red-400">{addFundsError}</p>
                </div>
              )}

              {addFundsSuccess && (
                <div className="p-3 bg-green-500/20 border border-green-500/30 rounded-lg">
                  <p className="text-sm text-green-400">Funds added successfully!</p>
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">Amount (NRN)</label>
                <input
                  type="number"
                  placeholder="0.00"
                  value={addFundsAmount}
                  onChange={(e) => setAddFundsAmount(e.target.value)}
                  className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                  disabled={addFundsLoading}
                  step="0.01"
                  min="0"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">Payment Method</label>
                <select
                  value={addFundsMethod}
                  onChange={(e) => setAddFundsMethod(e.target.value)}
                  className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                  disabled={addFundsLoading}
                >
                  <option>Credit Card</option>
                  <option>Bank Transfer</option>
                  <option>Crypto Transfer</option>
                </select>
              </div>
              <button
                onClick={executeAddFunds}
                disabled={addFundsLoading || addFundsSuccess}
                className="w-full py-3 bg-green-600 hover:bg-green-700 disabled:bg-green-600/50 disabled:cursor-not-allowed text-white rounded-lg font-medium transition-all"
              >
                {addFundsLoading ? 'Processing...' : addFundsSuccess ? 'Success!' : 'Add Funds'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Send NRN Modal */}
      {showSendNRN && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 max-w-md w-full">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Send NRN</h3>
              <button
                onClick={() => setShowSendNRN(false)}
                className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
                disabled={sendNRNLoading}
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            <div className="space-y-4">
              {sendNRNError && (
                <div className="p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
                  <p className="text-sm text-red-400">{sendNRNError}</p>
                </div>
              )}

              {sendNRNSuccess && (
                <div className="p-3 bg-green-500/20 border border-green-500/30 rounded-lg">
                  <p className="text-sm text-green-400">NRN sent successfully!</p>
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">Recipient Address</label>
                <input
                  type="text"
                  placeholder="0x..."
                  value={sendNRNAddress}
                  onChange={(e) => setSendNRNAddress(e.target.value)}
                  className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                  disabled={sendNRNLoading}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">Amount (NRN)</label>
                <input
                  type="number"
                  placeholder="0.00"
                  value={sendNRNAmount}
                  onChange={(e) => setSendNRNAmount(e.target.value)}
                  className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                  disabled={sendNRNLoading}
                  step="0.01"
                  min="0"
                  max={walletData.nrnBalance}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">Note (Optional)</label>
                <input
                  type="text"
                  placeholder="Payment for..."
                  value={sendNRNNote}
                  onChange={(e) => setSendNRNNote(e.target.value)}
                  className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                  disabled={sendNRNLoading}
                />
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-3">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-400">Network Fee:</span>
                  <span className="text-white">0.001 NRN</span>
                </div>
                <div className="flex justify-between text-sm mt-1">
                  <span className="text-gray-400">Available Balance:</span>
                  <span className="text-white">{walletData.nrnBalance.toLocaleString()} NRN</span>
                </div>
              </div>
              <button
                onClick={executeSendNRN}
                disabled={sendNRNLoading || sendNRNSuccess}
                className="w-full py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-600/50 disabled:cursor-not-allowed text-white rounded-lg font-medium transition-all"
              >
                {sendNRNLoading ? 'Sending...' : sendNRNSuccess ? 'Success!' : 'Send NRN'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface TransactionItemProps {
  type: 'consumption' | 'reward' | 'transfer';
  amount: number;
  description: string;
  timestamp: string;
  agentName: string | null;
}

function TransactionItem({ type, amount, description, timestamp, agentName }: TransactionItemProps) {
  const typeConfig = {
    consumption: { 
      icon: ArrowUpRight, 
      color: 'text-red-400', 
      bg: 'bg-red-500/20', 
      border: 'border-red-500/30',
      prefix: '-'
    },
    reward: { 
      icon: ArrowDownLeft, 
      color: 'text-green-400', 
      bg: 'bg-green-500/20', 
      border: 'border-green-500/30',
      prefix: '+'
    },
    transfer: { 
      icon: ArrowUpRight, 
      color: 'text-blue-400', 
      bg: 'bg-blue-500/20', 
      border: 'border-blue-500/30',
      prefix: amount > 0 ? '+' : '-'
    }
  };

  const config = typeConfig[type];
  const Icon = config.icon;

  return (
    <div className="flex items-center justify-between p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg hover:border-purple-500/30 transition-all">
      <div className="flex items-center space-x-3">
        <div className={`w-10 h-10 ${config.bg} ${config.border} border rounded-lg flex items-center justify-center`}>
          <Icon className={`w-5 h-5 ${config.color}`} />
        </div>
        <div>
          <p className="font-medium text-white">{description}</p>
          <div className="flex items-center space-x-2 text-xs text-gray-400">
            {agentName && <span>{agentName}</span>}
            <span>•</span>
            <span>{new Date(timestamp).toLocaleTimeString()}</span>
          </div>
        </div>
      </div>
      <div className="text-right">
        <p className={`font-semibold ${config.color}`}>
          {config.prefix}{Math.abs(amount)} NRN
        </p>
      </div>
    </div>
  );
}
