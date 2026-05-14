import React, { useState } from 'react';
import { Wallet, TrendingUp, ArrowDownLeft, ArrowUpRight, Copy, CheckCircle, QrCode } from 'lucide-react';

interface Transaction {
  id: string;
  type: 'receive' | 'send';
  amount: number;
  description: string;
  timestamp: Date;
  status: 'completed' | 'pending' | 'failed';
}

interface WalletModalContentProps {
  nrnBalance?: number;
  cognitiveMode?: boolean;
  cognitiveState?: {
    mode: string;
    isActive: boolean;
  };
}

export const WalletModalContent: React.FC<WalletModalContentProps> = ({ 
  nrnBalance = 1250, 
  cognitiveMode = false,
  cognitiveState 
}) => {
  const [copySuccess, setCopySuccess] = useState(false);
  const [walletData] = useState({
    nrnBalance: nrnBalance,
    usdValue: nrnBalance * 0.85,
    walletAddress: 'knirv1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t',
    change24h: 2.4
  });

  const [transactions] = useState<Transaction[]>([
    {
      id: 'tx-001',
      type: 'receive',
      amount: 50,
      description: 'Skill execution reward',
      timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000),
      status: 'completed'
    },
    {
      id: 'tx-002',
      type: 'send',
      amount: 25,
      description: 'Agent deployment cost',
      timestamp: new Date(Date.now() - 4 * 60 * 60 * 1000),
      status: 'completed'
    },
    {
      id: 'tx-003',
      type: 'receive',
      amount: 100,
      description: 'Staking reward',
      timestamp: new Date(Date.now() - 6 * 60 * 60 * 1000),
      status: 'completed'
    }
  ]);

  const handleCopyAddress = () => {
    navigator.clipboard.writeText(walletData.walletAddress);
    setCopySuccess(true);
    setTimeout(() => setCopySuccess(false), 2000);
  };

  const handleShowQRCode = () => {
    console.log('Show QR Code for address');
  };

  const handleAddFunds = () => {
    console.log('Add funds functionality');
  };

  const handleSendNRN = () => {
    console.log('Send NRN functionality');
  };

  const TransactionItem = ({ transaction }: { transaction: Transaction }) => {
    const isReceive = transaction.type === 'receive';
    
    return (
      <div className="flex items-center justify-between p-3 bg-gray-800/80 border border-gray-600/50 rounded-lg">
        <div className="flex items-center space-x-3">
          <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${
            isReceive ? 'bg-green-500/20 border border-green-500/20' : 'bg-red-500/20 border border-red-500/20'
          }`}>
            {isReceive ? (
              <ArrowDownLeft className="w-4 h-4 text-green-400" />
            ) : (
              <ArrowUpRight className="w-4 h-4 text-red-400" />
            )}
          </div>
          <div>
            <p className="text-sm font-medium text-white">{transaction.description}</p>
            <p className="text-xs text-gray-400">
              {transaction.timestamp.toLocaleString()}
            </p>
          </div>
        </div>
        <div className="text-right">
          <p className={`text-sm font-semibold ${
            isReceive ? 'text-green-400' : 'text-red-400'
          }`}>
            {isReceive ? '+' : '-'}{transaction.amount} NRN
          </p>
          <p className="text-xs text-gray-400 capitalize">{transaction.status}</p>
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-4">
      {/* Balance Card */}
      <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center space-x-2">
            <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-cyan-500 rounded-xl flex items-center justify-center">
              <Wallet className="w-5 h-5 text-white" />
            </div>
            <div>
              <h3 className="text-base font-semibold text-white">NRN Balance</h3>
              <p className="text-xs text-gray-400">Neural Resource Network</p>
            </div>
          </div>
          
          <div className={walletData.change24h >= 0 ? 'px-2 py-1 rounded-full bg-green-500/20 border border-green-500/30' : 'px-2 py-1 rounded-full bg-red-500/20 border border-red-500/30'}>
            <div className="flex items-center space-x-1">
              <TrendingUp className={walletData.change24h >= 0 ? 'w-2.5 h-2.5 text-green-400' : 'w-2.5 h-2.5 text-red-400 rotate-180'} />
              <span className={walletData.change24h >= 0 ? 'text-xs font-medium text-green-400' : 'text-xs font-medium text-red-400'}>
                {walletData.change24h >= 0 ? '+' : ''}{walletData.change24h}%
              </span>
            </div>
          </div>
        </div>

        <div className="space-y-2">
          <div className="text-2xl font-bold text-white">
            {walletData.nrnBalance.toLocaleString()} NRN
          </div>
          <div className="text-base text-gray-300">
            ≈ ${walletData.usdValue.toFixed(2)} USD
          </div>

          {/* Cognitive Status Indicator */}
          {cognitiveMode && (
            <div className="flex items-center space-x-1.5 mt-1.5 px-2 py-1 bg-purple-600/20 border border-purple-500/30 rounded-lg">
              <div className="w-1.5 h-1.5 bg-purple-400 rounded-full animate-pulse"></div>
              <span className="text-xs text-purple-400">
                Cognitive Mode: {cognitiveState?.mode || 'active'} |
                Status: {cognitiveState?.isActive ? 'online' : 'offline'}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Wallet Address */}
      <div className="space-y-2">
        <h3 className="text-base font-semibold text-white">Wallet Address</h3>
        <div className="flex items-center space-x-2 p-3 bg-gray-800/80 border border-gray-600/50 rounded-lg">
          <div className="w-8 h-8 bg-blue-500/20 rounded-lg flex items-center justify-center border border-blue-500/20">
            <Wallet className="w-4 h-4 text-blue-400" />
          </div>
          <div className="flex-1">
            <p className="font-mono text-xs text-white truncate">{walletData.walletAddress}</p>
            <p className="text-xs text-gray-400">KNIRV Network</p>
          </div>
          <div className="flex space-x-1">
            <button
              onClick={handleCopyAddress}
              className="p-1.5 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
              title="Copy Address"
            >
              {copySuccess ? <CheckCircle className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
            </button>
            <button
              onClick={handleShowQRCode}
              className="p-1.5 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
              title="Show QR Code"
            >
              <QrCode className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-2 gap-2">
        <button
          onClick={handleAddFunds}
          className="flex items-center justify-center space-x-2 py-3 bg-green-600/20 hover:bg-green-600/30 border border-green-500/30 rounded-lg text-green-400 hover:text-green-300 transition-all text-sm"
        >
          <ArrowDownLeft className="w-4 h-4" />
          <span className="font-medium">Add Funds</span>
        </button>
        <button
          onClick={handleSendNRN}
          className="flex items-center justify-center space-x-2 py-3 bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 rounded-lg text-blue-400 hover:text-blue-300 transition-all text-sm"
        >
          <ArrowUpRight className="w-4 h-4" />
          <span className="font-medium">Send NRN</span>
        </button>
      </div>

      {/* Transaction History */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-base font-semibold text-white">Recent Transactions</h3>
          <button className="text-sm text-blue-400 hover:text-blue-300 transition-colors">
            View All
          </button>
        </div>

        <div className="space-y-2">
          {transactions.map((tx) => (
            <TransactionItem key={tx.id} transaction={tx} />
          ))}
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-3">
        <div className="text-center p-3 bg-gray-800/80 border border-gray-600/50 rounded-lg">
          <div className="text-lg font-bold text-green-400">+127</div>
          <div className="text-xs text-gray-400">Earned Today</div>
        </div>
        <div className="text-center p-3 bg-gray-800/80 border border-gray-600/50 rounded-lg">
          <div className="text-lg font-bold text-red-400">-89</div>
          <div className="text-xs text-gray-400">Spent Today</div>
        </div>
        <div className="text-center p-3 bg-gray-800/80 border border-gray-600/50 rounded-lg">
          <div className="text-lg font-bold text-blue-400">15</div>
          <div className="text-xs text-gray-400">Transactions</div>
        </div>
      </div>
    </div>
  );
};