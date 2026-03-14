import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Users, Search, TrendingUp, Wallet } from 'lucide-react';

const Accounts: React.FC = () => {
  const [searchAddress, setSearchAddress] = useState('');
  const navigate = useNavigate();

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchAddress.trim()) {
      navigate(`/account/${searchAddress.trim()}`);
    }
  };

  // Sample accounts for demonstration
  const sampleAccounts = [
    {
      address: '0x1234567890123456789012345678901234567890',
      balance: 1000000,
      transactions: 156,
      type: 'Validator'
    },
    {
      address: '0x0987654321098765432109876543210987654321',
      balance: 250000,
      transactions: 89,
      type: 'Regular'
    },
    {
      address: '0xabcdefabcdefabcdefabcdefabcdefabcdefabcd',
      balance: 500000,
      transactions: 234,
      type: 'Contract'
    }
  ];

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-8">
        <Users className="w-8 h-8 text-purple-400" />
        <div>
          <h1 className="text-3xl font-bold text-white">Accounts</h1>
          <p className="text-gray-400">Search and explore blockchain accounts</p>
        </div>
      </div>

      {/* Search Section */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 mb-8">
        <h2 className="text-xl font-semibold text-white mb-4">Account Lookup</h2>
        <form onSubmit={handleSearch} className="flex space-x-4">
          <div className="flex-1">
            <input
              type="text"
              value={searchAddress}
              onChange={(e) => setSearchAddress(e.target.value)}
              placeholder="Enter account address (0x...)"
              className="w-full px-4 py-3 bg-gray-700/50 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
            />
          </div>
          <button
            type="submit"
            className="px-6 py-3 bg-purple-500 hover:bg-purple-600 text-white rounded-lg font-medium transition-colors flex items-center space-x-2"
          >
            <Search className="w-4 h-4" />
            <span>Search</span>
          </button>
        </form>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-gradient-to-br from-purple-500/10 to-purple-600/10 border border-purple-500/20 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <Users className="w-8 h-8 text-purple-400" />
            <div className="text-sm font-medium text-green-400">+2.3%</div>
          </div>
          <div className="text-2xl font-bold text-white">2,547</div>
          <div className="text-gray-400 text-sm">Total Accounts</div>
        </div>

        <div className="bg-gradient-to-br from-blue-500/10 to-blue-600/10 border border-blue-500/20 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <Wallet className="w-8 h-8 text-blue-400" />
            <div className="text-sm font-medium text-green-400">+5.1%</div>
          </div>
          <div className="text-2xl font-bold text-white">15.2M</div>
          <div className="text-gray-400 text-sm">Total Balance</div>
        </div>

        <div className="bg-gradient-to-br from-green-500/10 to-green-600/10 border border-green-500/20 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <TrendingUp className="w-8 h-8 text-green-400" />
            <div className="text-sm font-medium text-green-400">+12.5%</div>
          </div>
          <div className="text-2xl font-bold text-white">1,089</div>
          <div className="text-gray-400 text-sm">Active Today</div>
        </div>
      </div>

      {/* Sample Accounts */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
        <h2 className="text-xl font-semibold text-white mb-6">Sample Accounts</h2>
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">
          {sampleAccounts.map((account) => (
            <div
              key={account.address}
              onClick={() => navigate(`/account/${account.address}`)}
              className="bg-gray-700/30 hover:bg-gray-700/50 border border-gray-600/50 hover:border-gray-500/50 rounded-lg p-4 cursor-pointer transition-all duration-200 hover:scale-105"
            >
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center space-x-2">
                  <div className="w-8 h-8 bg-purple-500/20 rounded-lg flex items-center justify-center">
                    <Users className="w-4 h-4 text-purple-400" />
                  </div>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${
                    account.type === 'Validator' ? 'bg-blue-500/20 text-blue-400' :
                    account.type === 'Contract' ? 'bg-orange-500/20 text-orange-400' :
                    'bg-gray-500/20 text-gray-400'
                  }`}>
                    {account.type}
                  </span>
                </div>
              </div>
              
              <div className="space-y-2 mb-4">
                <div className="text-gray-400 text-xs">Address</div>
                <div className="text-gray-300 font-mono text-sm break-all">
                  {account.address}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-gray-400 text-xs">Balance</div>
                  <div className="text-white font-semibold">{account.balance.toLocaleString()}</div>
                </div>
                <div>
                  <div className="text-gray-400 text-xs">Transactions</div>
                  <div className="text-white font-semibold">{account.transactions}</div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Usage Instructions */}
      <div className="bg-blue-500/10 border border-blue-500/20 rounded-xl p-6 mt-8">
        <h3 className="text-lg font-semibold text-blue-400 mb-3">How to Use</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-gray-300">
          <div>
            <h4 className="font-medium text-white mb-2">Search by Address</h4>
            <p className="text-sm">Enter a complete account address (starting with 0x) to view detailed information including balance, transaction history, and more.</p>
          </div>
          <div>
            <h4 className="font-medium text-white mb-2">Browse Sample Accounts</h4>
            <p className="text-sm">Click on any of the sample accounts above to explore their details and understand the account structure.</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Accounts;