import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { blockchainApi, Account } from '../services/api';
import { ArrowLeft, Wallet, CreditCard, Hash, TrendingUp } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const AccountDetails: React.FC = () => {
  const { address } = useParams<{ address: string }>();
  const [account, setAccount] = useState<Account | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchAccount = async () => {
      if (!address) return;

      setLoading(true);
      setError(null);

      try {
        const accountData = await blockchainApi.getAccount(address);
        setAccount(accountData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch account');
      } finally {
        setLoading(false);
      }
    };

    fetchAccount();
  }, [address]);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <LoadingSpinner size="large" />
      </div>
    );
  }

  if (error || !account) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-6 text-center">
          <div className="text-red-400 text-lg font-medium mb-2">Account Not Found</div>
          <div className="text-gray-400 mb-4">{error || 'Account does not exist'}</div>
          <Link
            to="/accounts"
            className="inline-flex items-center space-x-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back to Accounts</span>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center space-x-4 mb-8">
        <Link
          to="/accounts"
          className="p-2 bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
        >
          <ArrowLeft className="w-5 h-5 text-white" />
        </Link>
        <div>
          <h1 className="text-3xl font-bold text-white">Account Details</h1>
          <p className="text-gray-400">Comprehensive account information</p>
        </div>
      </div>

      {/* Account Overview */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 mb-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="text-center">
            <div className="text-3xl font-bold text-blue-400">{account.balance.toLocaleString()}</div>
            <div className="text-gray-400">Balance (tokens)</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-green-400">{account.nonce}</div>
            <div className="text-gray-400">Transaction Nonce</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-purple-400">Regular</div>
            <div className="text-gray-400">Account Type</div>
          </div>
        </div>
      </div>

      {/* Account Information */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
        {/* Basic Information */}
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
          <h2 className="text-xl font-semibold text-white mb-6 flex items-center space-x-2">
            <Wallet className="w-5 h-5 text-blue-400" />
            <span>Account Information</span>
          </h2>
          
          <div className="space-y-4">
            <div>
              <label className="block text-gray-400 text-sm mb-1">Account Address</label>
              <div className="flex items-center space-x-2 bg-gray-700/50 rounded-lg p-3">
                <Hash className="w-4 h-4 text-gray-400" />
                <code 
                  className="text-gray-300 font-mono text-sm flex-1 cursor-pointer hover:text-white break-all"
                  onClick={() => copyToClipboard(account.address)}
                  title="Click to copy"
                >
                  {account.address}
                </code>
              </div>
            </div>

            <div>
              <label className="block text-gray-400 text-sm mb-1">Balance</label>
              <div className="flex items-center space-x-2 bg-gray-700/50 rounded-lg p-3">
                <Wallet className="w-4 h-4 text-gray-400" />
                <span className="text-gray-300 text-lg font-semibold">{account.balance.toLocaleString()} tokens</span>
              </div>
            </div>

            <div>
              <label className="block text-gray-400 text-sm mb-1">Transaction Nonce</label>
              <div className="flex items-center space-x-2 bg-gray-700/50 rounded-lg p-3">
                <CreditCard className="w-4 h-4 text-gray-400" />
                <span className="text-gray-300">{account.nonce}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Statistics */}
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
          <h2 className="text-xl font-semibold text-white mb-6 flex items-center space-x-2">
            <TrendingUp className="w-5 h-5 text-green-400" />
            <span>Account Statistics</span>
          </h2>
          
          <div className="space-y-6">
            <div className="grid grid-cols-2 gap-4">
              <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 text-center">
                <div className="text-2xl font-bold text-blue-400">${(account.balance * 0.85).toFixed(2)}</div>
                <div className="text-gray-400 text-sm">USD Value</div>
              </div>
              <div className="bg-green-500/10 border border-green-500/20 rounded-lg p-4 text-center">
                <div className="text-2xl font-bold text-green-400">{account.nonce}</div>
                <div className="text-gray-400 text-sm">Total Transactions</div>
              </div>
            </div>

            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-gray-400">Rank by Balance</span>
                <span className="text-gray-300">#247</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">Account Age</span>
                <span className="text-gray-300">156 days</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">Last Activity</span>
                <span className="text-gray-300">2 hours ago</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Transaction History */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
        <h2 className="text-xl font-semibold text-white mb-6 flex items-center space-x-2">
          <CreditCard className="w-5 h-5 text-green-400" />
          <span>Recent Transactions</span>
        </h2>

        {/* Placeholder for transaction history */}
        <div className="text-center py-12 text-gray-400">
          <CreditCard className="w-16 h-16 mx-auto mb-4 opacity-50" />
          <h3 className="text-lg font-medium mb-2">Transaction History</h3>
          <p className="text-gray-500 mb-4">
            Transaction history would be displayed here in a production environment.
          </p>
          <div className="bg-gray-700/30 rounded-lg p-4 text-left max-w-md mx-auto">
            <div className="text-sm text-gray-400 mb-2">Sample transaction format:</div>
            <div className="space-y-1 text-xs font-mono text-gray-300">
              <div>• TX Hash: 0xabc123...</div>
              <div>• Type: Transfer</div>
              <div>• Amount: 100 tokens</div>
              <div>• Time: 2 hours ago</div>
            </div>
          </div>
        </div>
      </div>

      {/* QR Code Section */}
      <div className="bg-gradient-to-br from-purple-500/10 to-purple-600/10 border border-purple-500/20 rounded-xl p-6 mt-8">
        <h3 className="text-lg font-semibold text-purple-400 mb-3">Share Account</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 items-center">
          <div>
            <p className="text-gray-300 mb-4">
              Share this account address with others or scan the QR code for easy access.
            </p>
            <button
              onClick={() => copyToClipboard(account.address)}
              className="px-4 py-2 bg-purple-500 hover:bg-purple-600 text-white rounded-lg transition-colors"
            >
              Copy Address
            </button>
          </div>
          <div className="flex justify-center">
            <div className="w-32 h-32 bg-gray-700/50 rounded-lg flex items-center justify-center">
              <div className="text-gray-400 text-center">
                <Hash className="w-8 h-8 mx-auto mb-2" />
                <div className="text-xs">QR Code</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AccountDetails;