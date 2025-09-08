import React, { useState, useEffect } from 'react';
import {
  TrendingUp,
  Zap,
  Bot,
  Activity,
  ArrowUpRight,
  Clock,
  Target,
  AlertTriangle,
  CheckCircle,
  Loader,
  Monitor,
  Upload,
  Wallet,
  Eye,
  EyeOff
} from 'lucide-react';
import AgentCreationModal from './modals/AgentCreationModal';
import RealTimePerformanceMonitor from './RealTimePerformanceMonitor';
import DragDropDeployment from './DragDropDeployment';
import { useNavigate } from 'react-router-dom';
import OnboardingTrigger from './onboarding/OnboardingTrigger';
import { api } from '../services/api';
import { walletService } from '../services/walletService';

export const Dashboard: React.FC = () => {
  const [isAgentCreationModalOpen, setIsAgentCreationModalOpen] = useState(false);
  const [showPerformanceMonitor, setShowPerformanceMonitor] = useState(true);
  const [performanceExpanded, setPerformanceExpanded] = useState(false);
  const [showDeploymentArea, setShowDeploymentArea] = useState(false);
  const [dashboardStats, setDashboardStats] = useState(null);
  const [recentActivity, setRecentActivity] = useState([]);
  const [targetSystems, setTargetSystems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [showWalletBalance, setShowWalletBalance] = useState(true);
  const [walletData, setWalletData] = useState({
    nrnBalance: 1247,
    usdValue: 312.75,
    change24h: 5.2
  });
  const navigate = useNavigate();

  // Convert dashboard stats to display format
  const stats = dashboardStats ? [
    {
      label: 'Active Agents',
      value: dashboardStats.active_agents.toString(),
      change: dashboardStats.changes?.active_agents || '+0%',
      icon: Bot,
      color: 'from-blue-500 to-cyan-500'
    },
    {
      label: 'Target Systems',
      value: dashboardStats.target_systems.toString(),
      change: dashboardStats.changes?.target_systems || '+0%',
      icon: Target,
      color: 'from-purple-500 to-pink-500'
    },
    {
      label: 'Inferences Today',
      value: dashboardStats.inferences_today.toLocaleString(),
      change: dashboardStats.changes?.inferences_today || '+0%',
      icon: Activity,
      color: 'from-green-500 to-emerald-500'
    },
    {
      label: 'Success Rate',
      value: `${dashboardStats.success_rate}%`,
      change: dashboardStats.changes?.success_rate || '+0%',
      icon: TrendingUp,
      color: 'from-orange-500 to-red-500'
    },
  ] : [];

  // Fetch dashboard data from API
  useEffect(() => {
    const fetchDashboardData = async () => {
      try {
        setLoading(true);
        setError(null);

        // Fetch dashboard stats
        const statsResponse = await api.get('/api/v1/analytics/dashboard-stats');
        if (statsResponse.data.success) {
          setDashboardStats(statsResponse.data.data);
        }

        // Fetch recent activity
        const activityResponse = await api.get('/api/v1/analytics/recent-activity?limit=6');
        if (activityResponse.data.success) {
          setRecentActivity(activityResponse.data.data);
        }

        // Fetch system status
        const statusResponse = await api.get('/api/v1/system/status');
        if (statusResponse.data.success) {
          setTargetSystems(statusResponse.data.data.target_systems);
        }

        // Load wallet data (only if controller is connected)
        try {
          const controllerStatus = await walletService.checkControllerConnection();
          if (controllerStatus.connected && controllerStatus.walletLinked) {
            const balanceData = await walletService.getBalance();
            setWalletData({
              nrnBalance: balanceData.nrnBalance,
              usdValue: balanceData.usdValue,
              change24h: balanceData.change24h
            });
          } else {
            // Show disconnected state
            setWalletData({
              nrnBalance: 0,
              usdValue: 0,
              change24h: 0
            });
          }
        } catch (walletError) {
          console.warn('Failed to load wallet data:', walletError);
          // Keep default wallet data
        }

      } catch (err) {
        console.error('Error fetching dashboard data:', err);
        setError(err.message || 'Failed to load dashboard data');
      } finally {
        setLoading(false);
      }
    };

    fetchDashboardData();

    // Refresh data every 30 seconds
    const interval = setInterval(fetchDashboardData, 30000);
    return () => clearInterval(interval);
  }, []);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'running':
        return <Loader className="w-5 h-5 text-blue-400 animate-spin" />;
      case 'failed':
        return <AlertTriangle className="w-5 h-5 text-red-400" />;
      default:
        return <Clock className="w-5 h-5 text-slate-400" />;
    }
  };

  const getTargetStatusColor = (status: string) => {
    switch (status) {
      case 'connected':
        return 'text-green-400 bg-green-500/20';
      case 'monitoring':
        return 'text-blue-400 bg-blue-500/20';
      case 'limited':
        return 'text-yellow-400 bg-yellow-500/20';
      case 'disconnected':
        return 'text-red-400 bg-red-500/20';
      default:
        return 'text-slate-400 bg-slate-500/20';
    }
  };

  const handleAgentCreated = () => {
    setIsAgentCreationModalOpen(false);
    // In a real implementation, you would update the state with the new agent
    // and potentially navigate to the agent manager
    navigate('/agents');
  };

  const handleNavigateToTargets = () => {
    navigate('/targets');
  };

  const handleNavigateToCapabilities = () => {
    navigate('/capabilities');
  };

  const handleNavigateToOrchestrator = () => {
    navigate('/orchestrator');
  };

  const handleDeploymentComplete = (results) => {
    console.log('Deployment completed:', results);
    // Handle deployment results - could show notifications, update agent list, etc.
  };

  // Mock target systems for demo
  const mockTargetSystems = [
    { id: 1, name: 'Production Server', type: 'Linux Server' },
    { id: 2, name: 'Staging Environment', type: 'Docker Container' },
    { id: 3, name: 'Development VM', type: 'Virtual Machine' },
    { id: 4, name: 'Cloud Instance', type: 'AWS EC2' }
  ];

  return (
    <div className="p-6 space-y-8">
      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">Agent Command Center</h1>
          <p className="text-slate-400">Monitor and orchestrate your NFT-Agents across target systems and environments.</p>
        </div>
        <div className="mt-4 lg:mt-0 flex items-center space-x-3">
          {/* Wallet Balance Display */}
          <div
            onClick={() => navigate('/wallet')}
            className="bg-gradient-to-r from-slate-800/50 to-slate-700/50 backdrop-blur-xl border border-slate-600/50 rounded-lg p-3 hover:from-slate-700/50 hover:to-slate-600/50 transition-all duration-200 cursor-pointer"
          >
            <div className="flex items-center space-x-3">
              <div className="flex items-center space-x-2">
                <Wallet className="w-4 h-4 text-yellow-400" />
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    setShowWalletBalance(!showWalletBalance);
                  }}
                  className="text-slate-400 hover:text-white transition-colors"
                >
                  {showWalletBalance ? <Eye className="w-3 h-3" /> : <EyeOff className="w-3 h-3" />}
                </button>
              </div>
              <div className="text-right">
                <div className="text-xs text-slate-400">NRN Balance</div>
                <div className="text-sm font-bold text-yellow-400">
                  {showWalletBalance ? `${walletData.nrnBalance} NRN` : '••••••'}
                </div>
              </div>
              <div className="text-right">
                <div className="text-xs text-slate-400">USD Value</div>
                <div className="text-sm font-bold text-white">
                  {showWalletBalance ? `$${walletData.usdValue.toFixed(2)}` : '••••••'}
                </div>
              </div>
              <div className={`text-xs flex items-center ${walletData.change24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                {walletData.change24h >= 0 ? (
                  <TrendingUp className="w-3 h-3 mr-1" />
                ) : (
                  <ArrowUpRight className="w-3 h-3 mr-1 rotate-180" />
                )}
                {walletData.change24h >= 0 ? '+' : ''}{walletData.change24h}%
              </div>
            </div>
          </div>

          <OnboardingTrigger />
          <button
            onClick={() => setShowPerformanceMonitor(!showPerformanceMonitor)}
            className="bg-gradient-to-r from-green-500 to-emerald-500 text-white px-4 py-2 rounded-lg font-medium hover:from-green-600 hover:to-emerald-600 transition-all duration-200 flex items-center space-x-2"
          >
            <Monitor className="w-4 h-4" />
            <span>{showPerformanceMonitor ? 'Hide' : 'Show'} Monitor</span>
          </button>
          <button
            onClick={() => setShowDeploymentArea(!showDeploymentArea)}
            className="bg-gradient-to-r from-orange-500 to-red-500 text-white px-4 py-2 rounded-lg font-medium hover:from-orange-600 hover:to-red-600 transition-all duration-200 flex items-center space-x-2"
          >
            <Upload className="w-4 h-4" />
            <span>{showDeploymentArea ? 'Hide' : 'Show'} Deploy</span>
          </button>
          <button
            onClick={() => setIsAgentCreationModalOpen(true)}
            className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2"
          >
            <Zap className="w-5 h-5" />
            <span>Create Agent</span>
          </button>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {loading ? (
          // Loading skeleton
          Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="h-4 bg-slate-700 rounded animate-pulse mb-2"></div>
                  <div className="h-8 bg-slate-700 rounded animate-pulse mb-2"></div>
                  <div className="h-4 bg-slate-700 rounded animate-pulse w-16"></div>
                </div>
                <div className="w-12 h-12 bg-slate-700 rounded-lg animate-pulse"></div>
              </div>
            </div>
          ))
        ) : error ? (
          // Error state
          <div className="col-span-full bg-red-900/20 border border-red-500/50 rounded-xl p-6">
            <div className="flex items-center space-x-3">
              <AlertTriangle className="w-6 h-6 text-red-400" />
              <div>
                <h3 className="text-red-400 font-medium">Failed to load dashboard data</h3>
                <p className="text-red-300/70 text-sm mt-1">{error}</p>
              </div>
            </div>
          </div>
        ) : (
          // Actual stats
          stats.map((stat, index) => {
            const Icon = stat.icon;
            return (
              <div key={index} className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50 hover:border-slate-600/50 transition-all duration-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-slate-400 text-sm font-medium">{stat.label}</p>
                    <p className="text-2xl font-bold text-white mt-1">{stat.value}</p>
                    <div className="flex items-center mt-2">
                      <ArrowUpRight className="w-4 h-4 text-green-400 mr-1" />
                      <span className="text-green-400 text-sm font-medium">{stat.change}</span>
                    </div>
                  </div>
                  <div className={`p-3 rounded-lg bg-gradient-to-r ${stat.color}`}>
                    <Icon className="w-6 h-6 text-white" />
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Real-Time Performance Monitor */}
      {showPerformanceMonitor && (
        <RealTimePerformanceMonitor
          isExpanded={performanceExpanded}
          onToggleExpand={() => setPerformanceExpanded(!performanceExpanded)}
        />
      )}

      {/* Drag & Drop Deployment Area */}
      {showDeploymentArea && (
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <h2 className="text-xl font-bold text-white mb-6 flex items-center space-x-2">
            <Upload className="w-6 h-6 text-orange-400" />
            <span>Quick Deployment</span>
          </h2>
          <DragDropDeployment
            onDeploymentComplete={handleDeploymentComplete}
            targetSystems={mockTargetSystems}
          />
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Recent Agent Activity */}
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-bold text-white">Recent Agent Activity</h2>
            <button 
              onClick={handleNavigateToOrchestrator}
              className="text-purple-400 hover:text-purple-300 text-sm font-medium"
            >
              View All
            </button>
          </div>
          <div className="space-y-4">
            {recentActivity.map((activity) => (
              <div key={activity.id} className="flex items-start space-x-4 p-3 rounded-lg hover:bg-slate-700/30 transition-colors duration-200">
                <div className="flex-shrink-0 mt-1">
                  {getStatusIcon(activity.status)}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-2 mb-1">
                    <p className="text-white font-medium truncate">{activity.agent}</p>
                    <span className="text-slate-400">→</span>
                    <p className="text-slate-300 text-sm truncate">{activity.target}</p>
                  </div>
                  <p className="text-slate-400 text-sm mb-1">{activity.capability}</p>
                  {activity.result && (
                    <p className="text-slate-500 text-xs">{activity.result}</p>
                  )}
                  <div className="flex items-center justify-between mt-2">
                    <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                      activity.status === 'completed' ? 'bg-green-500/20 text-green-400' :
                      activity.status === 'running' ? 'bg-blue-500/20 text-blue-400' :
                      activity.status === 'failed' ? 'bg-red-500/20 text-red-400' :
                      'bg-yellow-500/20 text-yellow-400'
                    }`}>
                      {activity.status}
                    </span>
                    <span className="text-slate-500 text-xs">{activity.time}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Target System Status */}
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-bold text-white">Target System Status</h2>
            <div className="flex items-center space-x-2">
              <div className="w-2 h-2 bg-green-400 rounded-full"></div>
              <span className="text-green-400 text-sm font-medium">
                {targetSystems.length} System{targetSystems.length !== 1 ? 's' : ''} Online
              </span>
            </div>
          </div>
          <div className="space-y-4">
            {targetSystems.map((system, index) => (
              <div key={index} className="flex items-center justify-between p-4 rounded-lg bg-slate-700/30 border border-slate-600/30">
                <div className="flex items-center space-x-3">
                  <div className={`w-3 h-3 rounded-full ${
                    system.status === 'connected' ? 'bg-green-400' :
                    system.status === 'monitoring' ? 'bg-blue-400' :
                    system.status === 'limited' ? 'bg-yellow-400' :
                    'bg-red-400'
                  }`}></div>
                  <div>
                    <p className="text-white font-medium">{system.name}</p>
                    <p className="text-slate-400 text-sm">{system.agents} active agents</p>
                  </div>
                </div>
                <div className="text-right">
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${getTargetStatusColor(system.status)}`}>
                    {system.status}
                  </span>
                  <p className="text-slate-500 text-xs mt-1">{system.last_activity || system.lastActivity}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
        <h2 className="text-xl font-bold text-white mb-6">Quick Actions</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <button 
            onClick={() => setIsAgentCreationModalOpen(true)}
            className="p-4 bg-gradient-to-r from-purple-500/20 to-blue-500/20 hover:from-purple-500/30 hover:to-blue-500/30 border border-purple-500/30 rounded-lg transition-all duration-200 text-left"
          >
            <Bot className="w-8 h-8 text-purple-400 mb-2" />
            <p className="text-white font-medium">Deploy New Agent</p>
            <p className="text-slate-400 text-sm">Launch agent to target system</p>
          </button>
          <button 
            onClick={handleNavigateToTargets}
            className="p-4 bg-gradient-to-r from-green-500/20 to-emerald-500/20 hover:from-green-500/30 hover:to-emerald-500/30 border border-green-500/30 rounded-lg transition-all duration-200 text-left"
          >
            <Target className="w-8 h-8 text-green-400 mb-2" />
            <p className="text-white font-medium">Add Target System</p>
            <p className="text-slate-400 text-sm">Connect new environment</p>
          </button>
          <button 
            onClick={handleNavigateToCapabilities}
            className="p-4 bg-gradient-to-r from-blue-500/20 to-cyan-500/20 hover:from-blue-500/30 hover:to-cyan-500/30 border border-blue-500/30 rounded-lg transition-all duration-200 text-left"
          >
            <Zap className="w-8 h-8 text-blue-400 mb-2" />
            <p className="text-white font-medium">Install Capability</p>
            <p className="text-slate-400 text-sm">Add MCP server capability</p>
          </button>
          <button 
            onClick={handleNavigateToOrchestrator}
            className="p-4 bg-gradient-to-r from-orange-500/20 to-red-500/20 hover:from-orange-500/30 hover:to-red-500/30 border border-orange-500/30 rounded-lg transition-all duration-200 text-left"
          >
            <Activity className="w-8 h-8 text-orange-400 mb-2" />
            <p className="text-white font-medium">Monitor Activity</p>
            <p className="text-slate-400 text-sm">View real-time logs</p>
          </button>
        </div>
      </div>

      {/* Agent Creation Modal */}
      <AgentCreationModal 
        isOpen={isAgentCreationModalOpen} 
        onClose={() => setIsAgentCreationModalOpen(false)} 
        onAgentCreated={handleAgentCreated} 
      />
    </div>
  );
};