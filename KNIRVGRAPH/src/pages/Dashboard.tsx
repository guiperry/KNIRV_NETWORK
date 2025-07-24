import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useBlockchain } from '../context/BlockchainContext';
import { blockchainApi, Block, BlockchainStats } from '../services/api';
import { Activity, Blocks, CreditCard, Clock, TrendingUp, ArrowRight, RefreshCw } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';
import StatsCard from '../components/StatsCard';
import BlockCard from '../components/BlockCard';

const Dashboard: React.FC = () => {
  const { currentHeight, isLoading, error, refreshData } = useBlockchain();
  const [stats, setStats] = useState<BlockchainStats | null>(null);
  const [recentBlocks, setRecentBlocks] = useState<Block[]>([]);
  const [statsLoading, setStatsLoading] = useState(true);

  useEffect(() => {
    const fetchDashboardData = async () => {
      setStatsLoading(true);
      try {
        const [statsData, blocksData] = await Promise.all([
          blockchainApi.getBlockchainStats(),
          blockchainApi.getRecentBlocks(5),
        ]);
        setStats(statsData);
        setRecentBlocks(blocksData);
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error);
      } finally {
        setStatsLoading(false);
      }
    };

    if (!isLoading && currentHeight > 0) {
      fetchDashboardData();
    }
  }, [currentHeight, isLoading]);

  if (isLoading || statsLoading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <LoadingSpinner size="large" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-6 text-center">
          <div className="text-red-400 text-lg font-medium mb-2">Connection Error</div>
          <div className="text-gray-400 mb-4">{error}</div>
          <button
            onClick={refreshData}
            className="inline-flex items-center space-x-2 px-4 py-2 bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
            <span>Retry</span>
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 py-8 space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">
            Blockchain Explorer
          </h1>
          <p className="text-gray-400 mt-2">
            Real-time blockchain data and analytics
          </p>
        </div>
        <div className="flex items-center space-x-2 text-sm text-gray-400">
          <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
          <span>Live</span>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatsCard
          title="Current Height"
          value={currentHeight.toLocaleString()}
          icon={Activity}
          trend={+2.3}
          color="blue"
        />
        <StatsCard
          title="Total Transactions"
          value={stats?.totalTransactions.toLocaleString() || '0'}
          icon={CreditCard}
          trend={+5.1}
          color="green"
        />
        <StatsCard
          title="Avg Block Time"
          value={`${stats?.avgBlockTime.toFixed(1) || '0'}s`}
          icon={Clock}
          trend={-1.2}
          color="purple"
        />
        <StatsCard
          title="Network Activity"
          value="High"
          icon={TrendingUp}
          trend={+12.5}
          color="orange"
        />
      </div>

      {/* Recent Blocks */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center space-x-3">
            <Blocks className="w-6 h-6 text-blue-400" />
            <h2 className="text-xl font-semibold">Recent Blocks</h2>
          </div>
          <Link
            to="/blocks"
            className="inline-flex items-center space-x-2 text-blue-400 hover:text-blue-300 transition-colors"
          >
            <span>View All</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>

        {recentBlocks.length > 0 ? (
          <div className="space-y-4">
            {recentBlocks.map((block) => (
              <BlockCard key={block.hash} block={block} />
            ))}
          </div>
        ) : (
          <div className="text-center py-8 text-gray-400">
            <Blocks className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No blocks found</p>
          </div>
        )}
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Link
          to="/blocks"
          className="group bg-gradient-to-br from-blue-500/10 to-blue-600/10 border border-blue-500/20 rounded-xl p-6 hover:border-blue-500/40 transition-all duration-300 hover:scale-105"
        >
          <Blocks className="w-8 h-8 text-blue-400 mb-4" />
          <h3 className="text-lg font-semibold mb-2">Explore Blocks</h3>
          <p className="text-gray-400 text-sm">
            Browse through all blocks in the blockchain
          </p>
          <ArrowRight className="w-5 h-5 text-blue-400 mt-4 group-hover:translate-x-1 transition-transform" />
        </Link>

        <Link
          to="/transactions"
          className="group bg-gradient-to-br from-green-500/10 to-green-600/10 border border-green-500/20 rounded-xl p-6 hover:border-green-500/40 transition-all duration-300 hover:scale-105"
        >
          <CreditCard className="w-8 h-8 text-green-400 mb-4" />
          <h3 className="text-lg font-semibold mb-2">View Transactions</h3>
          <p className="text-gray-400 text-sm">
            Search and analyze blockchain transactions
          </p>
          <ArrowRight className="w-5 h-5 text-green-400 mt-4 group-hover:translate-x-1 transition-transform" />
        </Link>

        <Link
          to="/accounts"
          className="group bg-gradient-to-br from-purple-500/10 to-purple-600/10 border border-purple-500/20 rounded-xl p-6 hover:border-purple-500/40 transition-all duration-300 hover:scale-105"
        >
          <Activity className="w-8 h-8 text-purple-400 mb-4" />
          <h3 className="text-lg font-semibold mb-2">Account Lookup</h3>
          <p className="text-gray-400 text-sm">
            Check account balances and transaction history
          </p>
          <ArrowRight className="w-5 h-5 text-purple-400 mt-4 group-hover:translate-x-1 transition-transform" />
        </Link>
      </div>
    </div>
  );
};

export default Dashboard;