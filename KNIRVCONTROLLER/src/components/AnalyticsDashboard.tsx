import * as React from 'react';
import { useState, useEffect } from 'react';
import {
  TrendingUp,
  Activity,
  Users,
  Target,
  Clock,
  CheckCircle,
  AlertTriangle,
  BarChart3,
  Download,
  RefreshCw
} from 'lucide-react';
import { analyticsService, DashboardStats, PerformanceMetrics, UsageAnalytics, AgentAnalytics } from '../services/AnalyticsService';

interface AnalyticsDashboardProps {
  isOpen: boolean;
  onClose: () => void;
}

const AnalyticsDashboard: React.FC<AnalyticsDashboardProps> = ({ isOpen, onClose }) => {
  const [dashboardStats, setDashboardStats] = useState<DashboardStats | null>(null);
  const [performanceMetrics, setPerformanceMetrics] = useState<PerformanceMetrics | null>(null);
  const [usageAnalytics, setUsageAnalytics] = useState<UsageAnalytics | null>(null);
  const [agentAnalytics, setAgentAnalytics] = useState<AgentAnalytics | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'performance' | 'usage' | 'agents'>('overview');

  useEffect(() => {
    if (isOpen) {
      loadAnalytics();
      // Auto-refresh every 30 seconds
      const interval = setInterval(loadAnalytics, 30000);
      return () => clearInterval(interval);
    }
  }, [isOpen]);

  const loadAnalytics = async () => {
    setIsLoading(true);
    try {
      const [dashboard, performance, usage, agents] = await Promise.all([
        analyticsService.getDashboardStats(),
        analyticsService.getPerformanceMetrics(),
        analyticsService.getUsageAnalytics(),
        analyticsService.getAgentAnalytics()
      ]);

      setDashboardStats(dashboard);
      setPerformanceMetrics(performance);
      setUsageAnalytics(usage);
      setAgentAnalytics(agents);
    } catch (error) {
      console.error('Failed to load analytics:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleExport = async (format: 'json' | 'csv') => {
    try {
      const data = await analyticsService.exportData(format);
      const blob = new Blob([data], { 
        type: format === 'json' ? 'application/json' : 'text/csv' 
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `knirv-analytics.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Failed to export analytics:', error);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-gray-900/95 border border-gray-700/50 rounded-xl w-full max-w-6xl h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-700/50">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center border border-blue-500/20">
              <BarChart3 className="w-5 h-5 text-blue-400" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">Analytics Dashboard</h2>
              <p className="text-sm text-gray-400">Real-time system metrics and insights</p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={loadAnalytics}
              disabled={isLoading}
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={() => handleExport('json')}
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
            >
              <Download className="w-4 h-4" />
            </button>
            <button
              onClick={onClose}
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
            >
              ×
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-gray-700/50">
          {([
            { id: 'overview' as const, label: 'Overview', icon: TrendingUp },
            { id: 'performance' as const, label: 'Performance', icon: Activity },
            { id: 'usage' as const, label: 'Usage', icon: Users },
            { id: 'agents' as const, label: 'Agents', icon: Target }
          ] as const).map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center space-x-2 px-6 py-3 text-sm font-medium transition-all ${
                activeTab === tab.id
                  ? 'text-blue-400 border-b-2 border-blue-400 bg-blue-500/10'
                  : 'text-gray-400 hover:text-white hover:bg-gray-700/30'
              }`}
            >
              <tab.icon className="w-4 h-4" />
              <span>{tab.label}</span>
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto h-full">
          {activeTab === 'overview' && dashboardStats && (
            <div className="space-y-6">
              {/* Key Metrics */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <MetricCard
                  title="Active Agents"
                  value={dashboardStats.activeAgents}
                  change={dashboardStats.changes.active_agents}
                  icon={Users}
                  color="blue"
                />
                <MetricCard
                  title="Target Systems"
                  value={dashboardStats.targetSystems}
                  change={dashboardStats.changes.target_systems}
                  icon={Target}
                  color="green"
                />
                <MetricCard
                  title="Inferences Today"
                  value={dashboardStats.inferencesToday}
                  change={dashboardStats.changes.inferences_today}
                  icon={Activity}
                  color="purple"
                />
                <MetricCard
                  title="Success Rate"
                  value={`${dashboardStats.successRate}%`}
                  change={dashboardStats.changes.success_rate}
                  icon={CheckCircle}
                  color="emerald"
                />
              </div>

              {/* Last Updated */}
              <div className="text-center text-sm text-gray-400">
                Last updated: {dashboardStats.lastUpdated.toLocaleString()}
              </div>
            </div>
          )}

          {activeTab === 'performance' && performanceMetrics && (
            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <MetricCard
                  title="CPU Usage"
                  value={`${performanceMetrics.cpuUsage}%`}
                  icon={Activity}
                  color={performanceMetrics.cpuUsage > 80 ? "red" : "blue"}
                />
                <MetricCard
                  title="Memory Usage"
                  value={`${performanceMetrics.memoryUsage}%`}
                  icon={Activity}
                  color={performanceMetrics.memoryUsage > 85 ? "red" : "blue"}
                />
                <MetricCard
                  title="Network Latency"
                  value={`${performanceMetrics.networkLatency}ms`}
                  icon={Activity}
                  color="blue"
                />
                <MetricCard
                  title="Response Time"
                  value={`${performanceMetrics.responseTime}ms`}
                  icon={Clock}
                  color="blue"
                />
                <MetricCard
                  title="Throughput"
                  value={`${performanceMetrics.throughput}/s`}
                  icon={TrendingUp}
                  color="green"
                />
                <MetricCard
                  title="Error Rate"
                  value={`${performanceMetrics.errorRate}%`}
                  icon={AlertTriangle}
                  color={performanceMetrics.errorRate > 5 ? "red" : "green"}
                />
              </div>
            </div>
          )}

          {activeTab === 'usage' && usageAnalytics && (
            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <MetricCard
                  title="Total Sessions"
                  value={usageAnalytics.totalSessions}
                  icon={Users}
                  color="blue"
                />
                <MetricCard
                  title="Avg Session Duration"
                  value={`${usageAnalytics.averageSessionDuration}m`}
                  icon={Clock}
                  color="blue"
                />
                <MetricCard
                  title="User Engagement"
                  value={`${usageAnalytics.userEngagement}%`}
                  icon={TrendingUp}
                  color="green"
                />
              </div>

              {/* Most Used Features */}
              <div className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
                <h3 className="text-lg font-semibold text-white mb-4">Most Used Features</h3>
                <div className="space-y-2">
                  {usageAnalytics.mostUsedFeatures.map((feature, index) => (
                    <div key={index} className="flex items-center justify-between">
                      <span className="text-gray-300">{feature.feature}</span>
                      <span className="text-blue-400 font-medium">{feature.usage}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {activeTab === 'agents' && agentAnalytics && (
            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <MetricCard
                  title="Total Agents"
                  value={agentAnalytics.totalAgents}
                  icon={Target}
                  color="blue"
                />
                <MetricCard
                  title="Active Agents"
                  value={agentAnalytics.activeAgents}
                  icon={CheckCircle}
                  color="green"
                />
                <MetricCard
                  title="Deployment Success"
                  value={`${agentAnalytics.deploymentSuccess}%`}
                  icon={CheckCircle}
                  color="green"
                />
                <MetricCard
                  title="Avg Execution Time"
                  value={`${agentAnalytics.averageExecutionTime}ms`}
                  icon={Clock}
                  color="blue"
                />
                <MetricCard
                  title="Skill Invocations"
                  value={agentAnalytics.skillInvocations}
                  icon={Activity}
                  color="purple"
                />
                <MetricCard
                  title="Error Count"
                  value={agentAnalytics.errorCount}
                  icon={AlertTriangle}
                  color={agentAnalytics.errorCount > 10 ? "red" : "green"}
                />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

interface MetricCardProps {
  title: string;
  value: string | number;
  change?: string;
  icon: React.ComponentType<{ className?: string }>;
  color: 'blue' | 'green' | 'purple' | 'emerald' | 'red';
}

const MetricCard: React.FC<MetricCardProps> = ({ title, value, change, icon: Icon, color }) => {
  const colorClasses = {
    blue: 'bg-blue-500/20 border-blue-500/20 text-blue-400',
    green: 'bg-green-500/20 border-green-500/20 text-green-400',
    purple: 'bg-purple-500/20 border-purple-500/20 text-purple-400',
    emerald: 'bg-emerald-500/20 border-emerald-500/20 text-emerald-400',
    red: 'bg-red-500/20 border-red-500/20 text-red-400'
  };

  return (
    <div className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
      <div className="flex items-center justify-between mb-2">
        <div className={`w-8 h-8 rounded-lg flex items-center justify-center border ${colorClasses[color]}`}>
          <Icon className="w-4 h-4" />
        </div>
        {change && (
          <span className={`text-xs font-medium ${
            change.startsWith('+') ? 'text-green-400' : 'text-red-400'
          }`}>
            {change}
          </span>
        )}
      </div>
      <div className="text-2xl font-bold text-white mb-1">{value}</div>
      <div className="text-sm text-gray-400">{title}</div>
    </div>
  );
};

export default AnalyticsDashboard;
