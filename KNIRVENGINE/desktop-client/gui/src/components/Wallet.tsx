import React, { useState, useEffect } from 'react';
import {
  Wallet as WalletIcon,
  Send,
  Download,
  ArrowUpDown,
  Filter,
  Search,
  TrendingUp,
  TrendingDown,
  Copy,
  ExternalLink,
  Eye,
  EyeOff,
  Zap,
  Activity,
  CheckCircle,
  AlertTriangle,
  Loader
} from 'lucide-react';
import { api } from '../services/api';
import { walletService, WalletBalance, WalletTransaction, ControllerConnectionStatus } from '../services/walletService';

interface CryptoAsset {
  symbol: string;
  name: string;
  price: string;
  change: number;
  amount: string;
  value: string;
  icon: string;
}

interface Transaction {
  id: string;
  type: 'send' | 'receive' | 'swap';
  asset: string;
  amount: string;
  value: string;
  time: string;
  status: 'pending' | 'completed' | 'failed';
  hash?: string;
}

interface WalletData {
  nrnBalance: number;
  usdValue: number;
  change24h: number;
  walletAddress: string;
}

export const Wallet: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'all' | 'crypto' | 'nft' | 'defi' | 'nrn'>('all');
  const [showBalance, setShowBalance] = useState(true);
  const [loading, setLoading] = useState(true);
  const [controllerStatus, setControllerStatus] = useState<ControllerConnectionStatus>({
    connected: false,
    walletLinked: false
  });
  const [walletData, setWalletData] = useState<WalletData>({
    nrnBalance: 1247,
    usdValue: 312.75,
    change24h: 5.2,
    walletAddress: '0x742d35Cc6aa34567...8B9fA2e1C4D'
  });
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [cryptoAssets, setCryptoAssets] = useState<CryptoAsset[]>([]);
  const [copySuccess, setCopySuccess] = useState(false);

  // Mock data for initial display
  const mockCryptoAssets: CryptoAsset[] = [
    {
      symbol: 'NRN',
      name: 'Neural Reasoning Network',
      price: '$0.251',
      change: 5.2,
      amount: '1247',
      value: '$312.75',
      icon: '/assets/nrn-icon.png',
    },
    {
      symbol: 'BTC',
      name: 'Bitcoin',
      price: '$47,842.50',
      change: 3.24,
      amount: '0.2845',
      value: '$13,613.25',
      icon: '/assets/btc-icon.png',
    },
    {
      symbol: 'ETH',
      name: 'Ethereum',
      price: '$2,845.32',
      change: -1.86,
      amount: '4.2567',
      value: '$12,115.89',
      icon: '/assets/eth-icon.png',
    },
  ];

  const mockTransactions: Transaction[] = [
    {
      id: '1',
      type: 'receive',
      asset: 'NRN',
      amount: '+125 NRN',
      value: '+$31.25',
      time: '2 hours ago',
      status: 'completed',
      hash: '0x1234...5678'
    },
    {
      id: '2',
      type: 'send',
      asset: 'ETH',
      amount: '-0.5 ETH',
      value: '-$1,422.66',
      time: '1 day ago',
      status: 'completed',
      hash: '0x5678...9012'
    },
    {
      id: '3',
      type: 'swap',
      asset: 'BTC → NRN',
      amount: '0.01 BTC',
      value: '$478.42',
      time: '3 days ago',
      status: 'completed',
      hash: '0x9012...3456'
    },
  ];

  useEffect(() => {
    const loadWalletData = async () => {
      try {
        setLoading(true);

        // Check controller connection first
        const status = await walletService.checkControllerConnection();
        setControllerStatus(status);

        if (status.connected && status.walletLinked) {
          // Load wallet balance and data
          const balanceData = await walletService.getBalance();
          setWalletData({
            nrnBalance: balanceData.nrnBalance,
            usdValue: balanceData.usdValue,
            change24h: balanceData.change24h,
            walletAddress: balanceData.walletAddress
          });

          // Load crypto assets
          const assets = await walletService.getAssets();
          setCryptoAssets(assets);

          // Load transactions
          const txData = await walletService.getTransactions(10, 0);
          setTransactions(txData);
        } else {
          // Fall back to mock data when controller not connected
          setCryptoAssets(mockCryptoAssets);
          setTransactions(mockTransactions);
        }

        setLoading(false);
      } catch (error) {
        console.error('Failed to load wallet data:', error);
        // Fall back to mock data
        setCryptoAssets(mockCryptoAssets);
        setTransactions(mockTransactions);
        setLoading(false);
      }
    };

    loadWalletData();
  }, []);

  const handleCopyAddress = async () => {
    try {
      await navigator.clipboard.writeText(walletData.walletAddress);
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    } catch (err) {
      console.error('Failed to copy address:', err);
    }
  };

  const tabs = [
    { key: 'all', label: 'All Assets' },
    { key: 'crypto', label: 'Crypto' },
    { key: 'nft', label: 'NFTs' },
    { key: 'defi', label: 'DeFi' },
    { key: 'nrn', label: 'NRN' },
  ];

  if (loading) {
    return (
      <div className="p-6 space-y-8">
        <div className="flex items-center justify-center h-64">
          <Loader className="w-8 h-8 animate-spin text-blue-500" />
          <span className="ml-2 text-slate-400">Loading wallet...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-8">
      {/* Controller Connection Status */}
      {!controllerStatus.connected || !controllerStatus.walletLinked ? (
        <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4 mb-6">
          <div className="flex items-center space-x-3">
            <AlertTriangle className="w-5 h-5 text-yellow-400" />
            <div>
              <h3 className="text-yellow-400 font-medium">KNIRVCONTROLLER Connection Required</h3>
              <p className="text-yellow-300/80 text-sm mt-1">
                Wallet functionality requires an active connection to KNIRVCONTROLLER.
                Please connect your KNIRVCONTROLLER to enable full wallet features.
              </p>
            </div>
          </div>
        </div>
      ) : (
        <div className="bg-green-500/10 border border-green-500/20 rounded-lg p-4 mb-6">
          <div className="flex items-center space-x-3">
            <CheckCircle className="w-5 h-5 text-green-400" />
            <div>
              <h3 className="text-green-400 font-medium">KNIRVCONTROLLER Connected</h3>
              <p className="text-green-300/80 text-sm mt-1">
                Wallet is successfully linked with KNIRVCONTROLLER at {controllerStatus.controllerEndpoint}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">Wallet</h1>
          <p className="text-slate-400">Manage your digital assets and NRN tokens</p>
        </div>
        <div className="mt-4 lg:mt-0 flex items-center space-x-3">
          <button className="bg-slate-700/50 text-white px-4 py-2 rounded-lg font-medium hover:bg-slate-600/50 transition-all duration-200 flex items-center space-x-2">
            <Search className="w-4 h-4" />
          </button>
          <button className="bg-slate-700/50 text-white px-4 py-2 rounded-lg font-medium hover:bg-slate-600/50 transition-all duration-200 flex items-center space-x-2">
            <Filter className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Balance Card */}
      <div className="bg-gradient-to-r from-slate-800/50 to-slate-700/50 backdrop-blur-xl border border-slate-600/50 rounded-xl p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <div className="flex items-center space-x-2 mb-2">
              <span className="text-slate-400 text-sm">Total Balance</span>
              <button
                onClick={() => setShowBalance(!showBalance)}
                className="text-slate-400 hover:text-white transition-colors"
              >
                {showBalance ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
              </button>
            </div>
            <div className="text-3xl font-bold text-white mb-2">
              {showBalance ? `$${walletData.usdValue.toFixed(2)}` : '••••••'}
            </div>
            <div className="flex items-center space-x-2">
              {walletData.change24h >= 0 ? (
                <TrendingUp className="w-4 h-4 text-green-400" />
              ) : (
                <TrendingDown className="w-4 h-4 text-red-400" />
              )}
              <span className={`text-sm font-medium ${walletData.change24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                {walletData.change24h >= 0 ? '+' : ''}{walletData.change24h}% today
              </span>
            </div>
          </div>
          <div className="text-right">
            <div className="text-slate-400 text-sm mb-2">NRN Balance</div>
            <div className="text-xl font-bold text-yellow-400 mb-2">
              {showBalance ? `${walletData.nrnBalance} NRN` : '••••••'}
            </div>
            <div className="text-xs text-slate-500">Neural Reasoning Network</div>
          </div>
        </div>

        {/* Wallet Address */}
        <div className="bg-slate-700/30 rounded-lg p-4 mb-6">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-xs text-slate-400 mb-1">Wallet Address</div>
              <div className="font-mono text-sm text-white">{walletData.walletAddress}</div>
            </div>
            <div className="flex space-x-2">
              <button
                onClick={handleCopyAddress}
                className="p-2 hover:bg-slate-600/50 rounded-lg text-slate-400 hover:text-white transition-all"
                title="Copy Address"
              >
                {copySuccess ? <CheckCircle className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4" />}
              </button>
              <button
                className="p-2 hover:bg-slate-600/50 rounded-lg text-slate-400 hover:text-white transition-all"
                title="View on Explorer"
              >
                <ExternalLink className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="grid grid-cols-3 gap-4">
          <button className="bg-gradient-to-r from-blue-500 to-blue-600 text-white px-4 py-3 rounded-lg font-medium hover:from-blue-600 hover:to-blue-700 transition-all duration-200 flex items-center justify-center space-x-2">
            <Send className="w-4 h-4" />
            <span>Send</span>
          </button>
          <button className="bg-slate-700/50 border border-slate-600/50 text-white px-4 py-3 rounded-lg font-medium hover:bg-slate-600/50 transition-all duration-200 flex items-center justify-center space-x-2">
            <Download className="w-4 h-4" />
            <span>Receive</span>
          </button>
          <button className="bg-slate-700/50 border border-slate-600/50 text-white px-4 py-3 rounded-lg font-medium hover:bg-slate-600/50 transition-all duration-200 flex items-center justify-center space-x-2">
            <ArrowUpDown className="w-4 h-4" />
            <span>Swap</span>
          </button>
        </div>
      </div>

      {/* Asset Tabs */}
      <div className="flex space-x-1 bg-slate-800/50 p-1 rounded-lg">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key as any)}
            className={`flex-1 px-4 py-2 rounded-md text-sm font-medium transition-all duration-200 ${
              activeTab === tab.key
                ? 'bg-blue-500 text-white'
                : 'text-slate-400 hover:text-white hover:bg-slate-700/50'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Assets List */}
      <div className="space-y-4">
        <h2 className="text-xl font-semibold text-white">Your Assets</h2>
        <div className="space-y-3">
          {cryptoAssets.map((asset, index) => (
            <div
              key={index}
              className="bg-slate-800/50 backdrop-blur-xl border border-slate-600/50 rounded-lg p-4 hover:bg-slate-700/50 transition-all duration-200"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-gradient-to-r from-blue-500 to-purple-500 rounded-full flex items-center justify-center">
                    <span className="text-white font-bold text-sm">{asset.symbol.charAt(0)}</span>
                  </div>
                  <div>
                    <div className="font-semibold text-white">{asset.symbol}</div>
                    <div className="text-sm text-slate-400">{asset.name}</div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="font-semibold text-white">{asset.price}</div>
                  <div className={`text-sm flex items-center ${asset.change >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                    {asset.change >= 0 ? (
                      <TrendingUp className="w-3 h-3 mr-1" />
                    ) : (
                      <TrendingDown className="w-3 h-3 mr-1" />
                    )}
                    {asset.change >= 0 ? '+' : ''}{asset.change.toFixed(2)}%
                  </div>
                </div>
                <div className="text-right">
                  <div className="font-semibold text-white">{asset.amount} {asset.symbol}</div>
                  <div className="text-sm text-slate-400">{asset.value}</div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Recent Transactions */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold text-white">Recent Transactions</h2>
          <button className="text-blue-400 hover:text-blue-300 text-sm font-medium">View All</button>
        </div>
        <div className="space-y-3">
          {transactions.map((tx) => (
            <div
              key={tx.id}
              className="bg-slate-800/50 backdrop-blur-xl border border-slate-600/50 rounded-lg p-4 hover:bg-slate-700/50 transition-all duration-200"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                    tx.type === 'receive' ? 'bg-green-500/20 text-green-400' :
                    tx.type === 'send' ? 'bg-red-500/20 text-red-400' :
                    'bg-blue-500/20 text-blue-400'
                  }`}>
                    {tx.type === 'receive' ? <Download className="w-5 h-5" /> :
                     tx.type === 'send' ? <Send className="w-5 h-5" /> :
                     <ArrowUpDown className="w-5 h-5" />}
                  </div>
                  <div>
                    <div className="font-semibold text-white capitalize">{tx.type} {tx.asset}</div>
                    <div className="text-sm text-slate-400">{tx.time}</div>
                  </div>
                </div>
                <div className="text-right">
                  <div className={`font-semibold ${
                    tx.type === 'receive' ? 'text-green-400' : 'text-white'
                  }`}>
                    {tx.amount}
                  </div>
                  <div className="text-sm text-slate-400">{tx.value}</div>
                </div>
                <div className="flex items-center space-x-2">
                  <div className={`px-2 py-1 rounded-full text-xs font-medium ${
                    tx.status === 'completed' ? 'bg-green-500/20 text-green-400' :
                    tx.status === 'pending' ? 'bg-yellow-500/20 text-yellow-400' :
                    'bg-red-500/20 text-red-400'
                  }`}>
                    {tx.status === 'completed' ? <CheckCircle className="w-3 h-3 inline mr-1" /> :
                     tx.status === 'pending' ? <Loader className="w-3 h-3 inline mr-1 animate-spin" /> :
                     <AlertTriangle className="w-3 h-3 inline mr-1" />}
                    {tx.status}
                  </div>
                  {tx.hash && (
                    <button
                      className="p-1 hover:bg-slate-600/50 rounded text-slate-400 hover:text-white transition-all"
                      title="View Transaction"
                    >
                      <ExternalLink className="w-3 h-3" />
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
