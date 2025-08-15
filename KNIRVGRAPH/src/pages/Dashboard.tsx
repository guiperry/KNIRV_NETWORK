import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useGraphChain } from '../context/GraphChainContext';
import { graphChainApi, SkillNode, ErrorNode, GraphChainStats } from '../services/api';
import { Brain, AlertTriangle, Clock, ArrowRight, RefreshCw, Network } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';
import StatsCard from '../components/StatsCard';
import SkillNodeCard from '../components/SkillNodeCard';

const Dashboard: React.FC = () => {
  const { currentHeight, isLoading, error, refreshData } = useGraphChain();
  const [stats, setStats] = useState<GraphChainStats | null>(null);
  const [recentSkills, setRecentSkills] = useState<SkillNode[]>([]);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [recentErrors, setRecentErrors] = useState<ErrorNode[]>([]); // TODO: Display recent errors in dashboard
  const [statsLoading, setStatsLoading] = useState(true);

  useEffect(() => {
    const fetchDashboardData = async () => {
      console.log('Fetching GraphChain dashboard data...');
      setStatsLoading(true);
      try {
        const [statsData, skillsData, errorsData] = await Promise.all([
          graphChainApi.getGraphChainStats(),
          graphChainApi.getRecentSkills(5),
          graphChainApi.getRecentErrors(5),
        ]);
        console.log('Dashboard data fetched successfully:', { statsData, skillsData, errorsData });
        setStats(statsData);
        setRecentSkills(skillsData);
        setRecentErrors(errorsData);
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error);
      } finally {
        setStatsLoading(false);
      }
    };

    console.log('Dashboard useEffect triggered:', { isLoading, currentHeight });
    if (!isLoading && currentHeight >= 0) {
      fetchDashboardData();
    } else if (!isLoading) {
      // If not loading but currentHeight is still negative, stop loading anyway
      setStatsLoading(false);
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
            GraphChain Explorer
          </h1>
          <p className="text-gray-400 mt-2">
            Real-time GraphChain data, SkillNodes, and ErrorNode analytics
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
          title="Graph Height"
          value={currentHeight.toLocaleString()}
          icon={Network}
          trend={+2.3}
          color="blue"
        />
        <StatsCard
          title="SkillNodes"
          value={stats?.totalSkillNodes.toLocaleString() || '0'}
          icon={Brain}
          trend={+5.1}
          color="green"
        />
        <StatsCard
          title="ErrorNodes"
          value={stats?.totalErrorNodes.toLocaleString() || '0'}
          icon={AlertTriangle}
          trend={+3.2}
          color="orange"
        />
        <StatsCard
          title="Avg Resolution Time"
          value={`${stats?.avgResolutionTime.toFixed(1) || '0'}s`}
          icon={Clock}
          trend={-1.2}
          color="purple"
        />
      </div>

      {/* Recent SkillNodes */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center space-x-3">
            <Brain className="w-6 h-6 text-blue-400" />
            <h2 className="text-xl font-semibold">Recent SkillNodes</h2>
          </div>
          <Link
            to="/skills"
            className="inline-flex items-center space-x-2 text-blue-400 hover:text-blue-300 transition-colors"
          >
            <span>View All</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>

        {recentSkills.length > 0 ? (
          <div className="space-y-4">
            {recentSkills.map((skill) => (
              <SkillNodeCard key={skill.id} skill={skill} />
            ))}
          </div>
        ) : (
          <div className="text-center py-8 text-gray-400">
            <Brain className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No SkillNodes found</p>
          </div>
        )}
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Link
          to="/skills"
          className="group bg-gradient-to-br from-blue-500/10 to-blue-600/10 border border-blue-500/20 rounded-xl p-6 hover:border-blue-500/40 transition-all duration-300 hover:scale-105"
        >
          <Brain className="w-8 h-8 text-blue-400 mb-4" />
          <h3 className="text-lg font-semibold mb-2">Explore SkillNodes</h3>
          <p className="text-gray-400 text-sm">
            Browse through all SkillNodes in the GraphChain
          </p>
          <ArrowRight className="w-5 h-5 text-blue-400 mt-4 group-hover:translate-x-1 transition-transform" />
        </Link>

        <Link
          to="/errors"
          className="group bg-gradient-to-br from-orange-500/10 to-orange-600/10 border border-orange-500/20 rounded-xl p-6 hover:border-orange-500/40 transition-all duration-300 hover:scale-105"
        >
          <AlertTriangle className="w-8 h-8 text-orange-400 mb-4" />
          <h3 className="text-lg font-semibold mb-2">View ErrorNodes</h3>
          <p className="text-gray-400 text-sm">
            Explore ErrorNodes and their resolution paths
          </p>
          <ArrowRight className="w-5 h-5 text-orange-400 mt-4 group-hover:translate-x-1 transition-transform" />
        </Link>

        <Link
          to="/graph"
          className="group bg-gradient-to-br from-purple-500/10 to-purple-600/10 border border-purple-500/20 rounded-xl p-6 hover:border-purple-500/40 transition-all duration-300 hover:scale-105"
        >
          <Network className="w-8 h-8 text-purple-400 mb-4" />
          <h3 className="text-lg font-semibold mb-2">Graph Visualization</h3>
          <p className="text-gray-400 text-sm">
            Visualize SkillNode-ErrorNode relationships
          </p>
          <ArrowRight className="w-5 h-5 text-purple-400 mt-4 group-hover:translate-x-1 transition-transform" />
        </Link>
      </div>
    </div>
  );
};

export default Dashboard;