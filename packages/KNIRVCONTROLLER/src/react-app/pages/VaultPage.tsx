import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router';
import { Wallet, ArrowUpRight, ArrowDownLeft, Zap, TrendingUp, Copy, ExternalLink, Lock, Loader, RefreshCw } from 'lucide-react';
import Layout from '@/react-app/components/Layout';
import { useVault } from '@/react-app/hooks/useVault';
import { useBackend } from '@/react-app/hooks/useBackend';

export default function WalletPage() {
  const navigate = useNavigate();
  const { status, currentAccount, unlockVault, lockVault } = useVault();
  const { walletData, transactions, isLoading, refresh, sendNRN } = useBackend(currentAccount ? currentAccount.getAddress('knirv') : null);
  
  const [password, setPassword] = useState('');
  const [showUnlockModal, setShowUnlockModal] = useState(false);
  const [showSendModal, setShowSendModal] = useState(false);
  const [sendAddress, setSendAddress] = useState('');
  const [sendAmount, setSendAmount] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const [txError, setTxError] = useState<string | null>(null);

  useEffect(() => {
    if (status === 'no_vault') {
      navigate('/onboarding');
    } else if (status === 'locked') {
      setShowUnlockModal(true);
    } else {
      setShowUnlockModal(false);
    }
  }, [status, navigate]);

  const handleUnlock = async () => {
    try {
      await unlockVault(password);
      setPassword('');
    } catch (err) {
      // Error handled in hook
    }
  };

  const handleSend = async () => {
    if (!sendAddress || !sendAmount) return;
    
    setIsProcessing(true);
    setTxError(null);
    try {
      const result = await sendNRN(sendAddress, parseFloat(sendAmount));
      if (result.success) {
        setShowSendModal(false);
        setSendAddress('');
        setSendAmount('');
        refresh();
      } else {
        setTxError(result.error || 'Transaction failed');
      }
    } catch (err) {
      setTxError('Transaction failed');
    } finally {
      setIsProcessing(false);
    }
  };

  if (status === 'initializing') {
    return (
      <Layout>
        <div className="flex items-center justify-center min-h-[60vh]">
          <Loader className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="p-4 pb-24 space-y-6 relative">
        {/* Header - Gradient Text */}
        <div className="text-center py-4 relative">
          <h2 className="text-2xl font-bold gradient-text mb-2">
            Vault
          </h2>
          <p className="text-slate-400 text-sm font-mono uppercase tracking-wider">
            Manage your Vault and DVE wallet assets
          </p>
          <button 
            onClick={lockVault}
            className="absolute right-0 top-4 p-2 text-slate-500 hover:text-white transition-colors"
          >
            <Lock className="w-5 h-5" />
          </button>
        </div>

        {/* Balance Card - Glass Panel */}
        <div className="relative group">
          <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/50 to-blue-800/50 rounded-2xl blur opacity-30 group-hover:opacity-50 transition duration-300"></div>
          
          <div className="relative glass-panel p-6 glow-hover">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center space-x-3">
                <div className="w-12 h-12 bg-blue-600 rounded-xl flex items-center justify-center">
                  <Zap className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-white">NRN Balance</h3>
                  <p className="text-sm text-slate-400 font-mono">Neural Resource Network</p>
                </div>
              </div>
              
              {walletData && (
                <div className={`px-3 py-1 rounded-full ${walletData.change24h >= 0 ? 'bg-green-500/20 border border-green-500/30' : 'bg-red-500/20 border border-red-500/30'}`}>
                  <div className="flex items-center space-x-1">
                    <TrendingUp className={`w-3 h-3 ${walletData.change24h >= 0 ? 'text-green-400' : 'text-red-400 rotate-180'}`} />
                    <span className={`text-xs font-medium font-mono ${walletData.change24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                      {walletData.change24h >= 0 ? '+' : ''}{walletData.change24h}%
                    </span>
                  </div>
                </div>
              )}
            </div>

            <div className="space-y-2">
              {isLoading ? (
                <div className="h-16 animate-pulse bg-slate-800 rounded-lg"></div>
              ) : (
                <>
                  <div className="text-3xl font-bold text-white">
                    {walletData?.nrnBalance.toLocaleString() ?? '0'} NRN
                  </div>
                  <div className="text-lg text-slate-300 font-mono">
                    ≈ ${walletData?.usdValue.toFixed(2) ?? '0.00'} USD
                  </div>
                </>
              )}
            </div>
          </div>
        </div>

        {/* Wallet Address */}
        <div className="space-y-3">
          <h3 className="text-lg font-semibold text-white">Wallet Address</h3>
          <div className="flex items-center space-x-3 p-4 glass-panel glow-hover">
            <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center border border-blue-500/20">
              <Wallet className="w-5 h-5 text-blue-400" />
            </div>
            <div className="flex-1 overflow-hidden">
              <p className="font-mono text-sm text-white truncate">
                {currentAccount ? currentAccount.getAddress('knirv') : 'Loading...'}
              </p>
              <p className="text-xs text-slate-500 font-mono">KNIRV Network</p>
            </div>
            <div className="flex space-x-2">
              <button 
                onClick={() => {
                  if (currentAccount) {
                    navigator.clipboard.writeText(currentAccount.getAddress('knirv'));
                  }
                }}
                className="p-2 hover:bg-white/5 rounded-lg text-slate-400 hover:text-white transition-all"
              >
                <Copy className="w-4 h-4" />
              </button>
              <button className="p-2 hover:bg-white/5 rounded-lg text-slate-400 hover:text-white transition-all">
                <ExternalLink className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        {/* Quick Actions */}
        <div className="grid grid-cols-2 gap-3">
          <button className="flex items-center justify-center space-x-3 py-4 bg-green-600/20 hover:bg-green-600/30 rounded-xl border border-green-500/30 text-green-400 hover:text-green-300 transition-all glow-hover">
            <ArrowDownLeft className="w-5 h-5" />
            <span className="font-medium font-mono">Add Funds</span>
          </button>
          <button 
            onClick={() => setShowSendModal(true)}
            className="flex items-center justify-center space-x-3 py-4 bg-blue-600/20 hover:bg-blue-600/30 rounded-xl border border-blue-500/30 text-blue-400 hover:text-blue-300 transition-all glow-hover"
          >
            <ArrowUpRight className="w-5 h-5" />
            <span className="font-medium font-mono">Send NRN</span>
          </button>
        </div>

        {/* Transaction History */}
        <div>
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-white">Recent Transactions</h3>
            <button 
              onClick={refresh}
              className="text-sm text-blue-400 hover:text-blue-300 transition-colors font-mono uppercase flex items-center space-x-1"
            >
              <RefreshCw className={`w-3 h-3 ${isLoading ? 'animate-spin' : ''}`} />
              <span>Refresh</span>
            </button>
          </div>
          
          <div className="space-y-3">
            {transactions.map((tx) => (
              <TransactionItem key={tx.id} {...tx} />
            ))}
          </div>
        </div>

        {/* Unlock Modal */}
        {showUnlockModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4">
            <div className="w-full max-w-sm bg-slate-900 border border-slate-700 rounded-2xl p-6 space-y-4">
              <div className="text-center">
                <div className="w-12 h-12 bg-blue-600/20 rounded-full flex items-center justify-center mx-auto mb-3">
                  <Lock className="w-6 h-6 text-blue-400" />
                </div>
                <h3 className="text-xl font-bold text-white">Unlock Vault</h3>
                <p className="text-sm text-slate-400">Enter your password to access your vault</p>
              </div>
              
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-slate-950 border border-slate-700 rounded-lg p-3 text-white focus:border-blue-500 focus:outline-none"
                placeholder="Password"
              />
              
              <button
                onClick={handleUnlock}
                className="w-full py-3 bg-blue-600 hover:bg-blue-500 text-white rounded-xl font-bold text-sm uppercase tracking-wide transition-all"
              >
                Unlock
              </button>
            </div>
          </div>
        )}

        {/* Send Modal */}
        {showSendModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4">
            <div className="w-full max-w-sm bg-slate-900 border border-slate-700 rounded-2xl p-6 space-y-4">
              <div className="text-center">
                <h3 className="text-xl font-bold text-white">Send NRN</h3>
                <p className="text-sm text-slate-400">Transfer tokens to another address</p>
              </div>
              
              <div className="space-y-3">
                <div>
                  <label className="text-xs font-mono text-slate-500 uppercase">Recipient Address</label>
                  <input
                    type="text"
                    value={sendAddress}
                    onChange={(e) => setSendAddress(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-700 rounded-lg p-3 text-white focus:border-blue-500 focus:outline-none text-sm font-mono"
                    placeholder="knirv1..."
                  />
                </div>
                <div>
                  <label className="text-xs font-mono text-slate-500 uppercase">Amount (NRN)</label>
                  <input
                    type="number"
                    value={sendAmount}
                    onChange={(e) => setSendAmount(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-700 rounded-lg p-3 text-white focus:border-blue-500 focus:outline-none"
                    placeholder="0.00"
                  />
                </div>
              </div>
              
              {txError && (
                <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-xs text-center">
                  {txError}
                </div>
              )}

              <div className="flex space-x-3">
                <button
                  onClick={() => setShowSendModal(false)}
                  className="flex-1 py-3 bg-slate-800 hover:bg-slate-700 text-white rounded-xl font-bold text-sm uppercase tracking-wide transition-all"
                >
                  Cancel
                </button>
                <button
                  onClick={handleSend}
                  disabled={isProcessing || !sendAddress || !sendAmount}
                  className="flex-1 py-3 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-xl font-bold text-sm uppercase tracking-wide transition-all flex items-center justify-center"
                >
                  {isProcessing ? <Loader className="w-4 h-4 animate-spin" /> : 'Send'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}

interface TransactionItemProps {
  type: 'consumption' | 'reward' | 'transfer';
  amount: number;
  description: string;
  timestamp: string;
  workflowName?: string;
}

function TransactionItem({ type, amount, description, timestamp, workflowName }: TransactionItemProps) {
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
      prefix: amount > 0 ? '+' : ''
    }
  };

  const config = typeConfig[type];
  const Icon = config.icon;

  return (
    <div className="flex items-center justify-between p-4 glass-panel glow-hover">
      <div className="flex items-center space-x-3">
        <div className={`w-10 h-10 ${config.bg} ${config.border} border rounded-xl flex items-center justify-center`}>
          <Icon className={`w-5 h-5 ${config.color}`} />
        </div>
        <div>
          <p className="font-medium text-white">{description}</p>
          <div className="flex items-center space-x-2 text-xs text-slate-400 font-mono">
            {workflowName && <span>{workflowName}</span>}
            <span>•</span>
            <span>{new Date(timestamp).toLocaleTimeString()}</span>
          </div>
        </div>
      </div>
      <div className="text-right">
        <p className={`font-semibold ${config.color} font-mono`}>
          {config.prefix}{Math.abs(amount)} NRN
        </p>
      </div>
    </div>
  );
}
