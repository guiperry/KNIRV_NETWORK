import { Wallet, ArrowUpRight, ArrowDownLeft, Zap, TrendingUp, Copy, ExternalLink } from 'lucide-react';
import Layout from './components/Layout';

export default function WalletPage() {
  const walletData = {
    nrnBalance: 1247,
    usdValue: 312.75,
    change24h: 5.2,
    walletAddress: '0x742d35Cc6aa34567...8B9fA2e1C4D'
  };

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

  return (
    <Layout>
      <div className="p-4 pb-24 space-y-6">
        {/* Header */}
        <div className="text-center py-4">
          <h2 className="text-2xl font-bold bg-gradient-to-r from-purple-400 to-cyan-400 bg-clip-text text-transparent mb-2">
            KNIRV Wallet
          </h2>
          <p className="text-slate-400 text-sm">
            Manage your NRN tokens and transaction history
          </p>
        </div>

        {/* Balance Card */}
        <div className="relative group">
          <div className="absolute -inset-0.5 bg-gradient-to-r from-purple-600/50 to-cyan-600/50 rounded-2xl blur opacity-30 group-hover:opacity-50 transition duration-300"></div>
          
          <div className="relative bg-slate-800/90 backdrop-blur-xl rounded-2xl p-6 border border-purple-500/30">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center space-x-3">
                <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-cyan-500 rounded-xl flex items-center justify-center">
                  <Zap className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-white">NRN Balance</h3>
                  <p className="text-sm text-slate-400">Neural Resource Network</p>
                </div>
              </div>
              
              <div className={`px-3 py-1 rounded-full ${walletData.change24h >= 0 ? 'bg-green-500/20 border border-green-500/30' : 'bg-red-500/20 border border-red-500/30'}`}>
                <div className="flex items-center space-x-1">
                  <TrendingUp className={`w-3 h-3 ${walletData.change24h >= 0 ? 'text-green-400' : 'text-red-400 rotate-180'}`} />
                  <span className={`text-xs font-medium ${walletData.change24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                    {walletData.change24h >= 0 ? '+' : ''}{walletData.change24h}%
                  </span>
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <div className="text-3xl font-bold text-white">
                {walletData.nrnBalance.toLocaleString()} NRN
              </div>
              <div className="text-lg text-slate-300">
                ≈ ${walletData.usdValue.toFixed(2)} USD
              </div>
            </div>
          </div>
        </div>

        {/* Wallet Address */}
        <div className="space-y-3">
          <h3 className="text-lg font-semibold text-white">Wallet Address</h3>
          <div className="flex items-center space-x-3 p-4 bg-slate-800/60 backdrop-blur-xl rounded-xl border border-slate-700/50">
            <div className="w-10 h-10 bg-gradient-to-br from-purple-500/20 to-cyan-500/20 rounded-lg flex items-center justify-center border border-purple-500/20">
              <Wallet className="w-5 h-5 text-purple-400" />
            </div>
            <div className="flex-1">
              <p className="font-mono text-sm text-white">{walletData.walletAddress}</p>
              <p className="text-xs text-slate-400">KNIRV Network</p>
            </div>
            <div className="flex space-x-2">
              <button className="p-2 hover:bg-slate-700/50 rounded-lg text-slate-400 hover:text-white transition-all">
                <Copy className="w-4 h-4" />
              </button>
              <button className="p-2 hover:bg-slate-700/50 rounded-lg text-slate-400 hover:text-white transition-all">
                <ExternalLink className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        {/* Quick Actions */}
        <div className="grid grid-cols-2 gap-3">
          <button className="flex items-center justify-center space-x-3 py-4 bg-gradient-to-r from-green-600/20 to-emerald-600/20 hover:from-green-600/30 hover:to-emerald-600/30 rounded-xl border border-green-500/30 text-green-400 hover:text-green-300 transition-all">
            <ArrowDownLeft className="w-5 h-5" />
            <span className="font-medium">Add Funds</span>
          </button>
          <button className="flex items-center justify-center space-x-3 py-4 bg-gradient-to-r from-blue-600/20 to-cyan-600/20 hover:from-blue-600/30 hover:to-cyan-600/30 rounded-xl border border-blue-500/30 text-blue-400 hover:text-blue-300 transition-all">
            <ArrowUpRight className="w-5 h-5" />
            <span className="font-medium">Send NRN</span>
          </button>
        </div>

        {/* Transaction History */}
        <div>
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-white">Recent Transactions</h3>
            <button className="text-sm text-purple-400 hover:text-purple-300 transition-colors">
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
          <div className="text-center p-4 bg-slate-800/50 backdrop-blur-xl rounded-xl border border-slate-700/50">
            <div className="text-xl font-bold text-green-400">+127</div>
            <div className="text-xs text-slate-400">Earned Today</div>
          </div>
          <div className="text-center p-4 bg-slate-800/50 backdrop-blur-xl rounded-xl border border-slate-700/50">
            <div className="text-xl font-bold text-red-400">-89</div>
            <div className="text-xs text-slate-400">Spent Today</div>
          </div>
          <div className="text-center p-4 bg-slate-800/50 backdrop-blur-xl rounded-xl border border-slate-700/50">
            <div className="text-xl font-bold text-purple-400">15</div>
            <div className="text-xs text-slate-400">Transactions</div>
          </div>
        </div>
      </div>
    </Layout>
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
    <div className="flex items-center justify-between p-4 bg-slate-800/60 backdrop-blur-xl rounded-xl border border-slate-700/50 hover:border-purple-500/30 transition-all">
      <div className="flex items-center space-x-3">
        <div className={`w-10 h-10 ${config.bg} ${config.border} border rounded-xl flex items-center justify-center`}>
          <Icon className={`w-5 h-5 ${config.color}`} />
        </div>
        <div>
          <p className="font-medium text-white">{description}</p>
          <div className="flex items-center space-x-2 text-xs text-slate-400">
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
