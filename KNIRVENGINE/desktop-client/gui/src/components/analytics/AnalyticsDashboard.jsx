import React, { useState, useEffect } from 'react';
import {
  TrendingUp,
  TrendingDown,
  BarChart3,
  PieChart,
  Activity,
  Clock,
  Zap,
  Target,
  ArrowUpRight,
  ArrowDownRight,
  Bot,
  GitBranch,
  RefreshCw,
  Calendar,
  CheckCircle,
  AlertCircle
} from 'lucide-react';

export const AnalyticsDashboard = () => {
  const [isLoading, setIsLoading] = useState(true);
  const [timeRange, setTimeRange] = useState('30d');
  const [analyticsData, setAnalyticsData] = useState(null);

  useEffect(() => {
    fetchAnalyticsData();
  }, [timeRange]);

  const fetchAnalyticsData = async () => {
    setIsLoading(true);
    try {
      // In a real implementation, this would be an API call with the timeRange parameter
      // For now, we'll simulate it with a delay
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      // Sample data
      setAnalyticsData({
        summary: {
          totalWorkflows: 2847,
          completedWorkflows: 2765,
          failedWorkflows: 82,
          successRate: 97.2,
          averageResponseTime: 4.2,
          todayWorkflows: 124,
          agentCount: 47,
          targetCount: 23
        },
        chartData: [
          { name: 'Jan', orchestrations: 420, success: 96.2 },
          { name: 'Feb', orchestrations: 680, success: 95.8 },
          { name: 'Mar', orchestrations: 890, success: 97.1 },
          { name: 'Apr', orchestrations: 1240, success: 97.8 },
          { name: 'May', orchestrations: 1680, success: 97.2 },
          { name: 'Jun', orchestrations: 2100, success: 98.1 }
        ],
        targetSystemUsage: [
          { name: 'Chrome Browser', usage: 1520, percentage: 34.2, color: 'from-blue-500 to-cyan-500' },
          { name: 'File System', usage: 1240, percentage: 27.9, color: 'from-purple-500 to-pink-500' },
          { name: 'VS Code Editor', usage: 890, percentage: 20.0, color: 'from-green-500 to-emerald-500' },
          { name: 'Terminal/Shell', usage: 650, percentage: 14.6, color: 'from-orange-500 to-red-500' },
          { name: 'Network Interface', usage: 320, percentage: 7.2, color: 'from-yellow-500 to-amber-500' },
          { name: 'Others', usage: 180, percentage: 4.1, color: 'from-slate-500 to-gray-500' }
        ],
        topCapabilities: [
          { id: 'web_scraping', name: 'Web Scraping', growth: '+34%', volume: '1.2K', trend: 'up' },
          { id: 'file_analysis', name: 'File Analysis', growth: '+28%', volume: '890', trend: 'up' },
          { id: 'code_analysis', name: 'Code Analysis', growth: '+15%', volume: '650', trend: 'up' },
          { id: 'system_monitoring', name: 'System Monitoring', growth: '-8%', volume: '420', trend: 'down' }
        ],
        agentPerformance: [
          {
            agent: 'CyberPunk Agent #7804',
            orchestrations: 247,
            successRate: 98.7,
            avgTime: '3.2s',
            trend: 'up'
          },
          {
            agent: 'Data Miner #3749',
            successRate: 94.2,
            orchestrations: 189,
            avgTime: '5.8s',
            trend: 'up'
          },
          {
            agent: 'Security Guardian #182',
            orchestrations: 156,
            successRate: 99.1,
            avgTime: '2.9s',
            trend: 'up'
          },
          {
            agent: 'Code Reviewer #7894',
            orchestrations: 98,
            successRate: 87.3,
            avgTime: '8.1s',
            trend: 'down'
          }
        ]
      });
    } catch (error) {
      console.error('Error fetching analytics data:', error);
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading || !analyticsData) {
    return (
      <div className="p-6 flex items-center justify-center h-[80vh]">
        <div className="text-center">
          <RefreshCw className="w-12 h-12 text-purple-500 animate-spin mx-auto mb-4" />
          <h2 className="text-xl font-bold text-white">Loading Analytics Data...</h2>
          <p className="text-slate-400 mt-2">Please wait while we gather your insights.</p>
        </div>
      </div>
    );
  }

  const { summary, chartData, targetSystemUsage, topCapabilities, agentPerformance } = analyticsData;

  return (
    <div className="p-6 space-y-8">
      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">Analytics Dashboard</h1>
          <p className="text-slate-400">Monitor agent performance, orchestration patterns, and system insights.</p>
        </div>
        <div className="mt-4 lg:mt-0 flex items-center space-x-3">
          <select 
            className="bg-slate-800/50 border border-slate-700/50 text-white px-4 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-purple-500"
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value)}
          >
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
            <option value="90d">Last 90 days</option>
            <option value="1y">Last year</option>
          </select>
          <button 
            onClick={fetchAnalyticsData}
            className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2"
          >
            <RefreshCw className="w-5 h-5" />
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center justify-between mb-2">
            <p className="text-slate-400 text-sm font-medium">Total Orchestrations</p>
            <ArrowUpRight className="w-4 h-4 text-green-400" />
          </div>
          <p className="text-3xl font-bold text-white mb-2">{summary.totalWorkflows.toLocaleString()}</p>
          <div className="flex items-center space-x-2">
            <span className="text-sm font-medium text-green-400">+18.5%</span>
            <span className="text-slate-400 text-sm">vs last month</span>
          </div>
        </div>

        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center justify-between mb-2">
            <p className="text-slate-400 text-sm font-medium">Success Rate</p>
            <ArrowUpRight className="w-4 h-4 text-green-400" />
          </div>
          <p className="text-3xl font-bold text-white mb-2">{summary.successRate}%</p>
          <div className="flex items-center space-x-2">
            <span className="text-sm font-medium text-green-400">+1.8%</span>
            <span className="text-slate-400 text-sm">vs last month</span>
          </div>
        </div>

        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center justify-between mb-2">
            <p className="text-slate-400 text-sm font-medium">Avg Execution Time</p>
            <ArrowUpRight className="w-4 h-4 text-green-400" />
          </div>
          <p className="text-3xl font-bold text-white mb-2">{summary.averageResponseTime}s</p>
          <div className="flex items-center space-x-2">
            <span className="text-sm font-medium text-green-400">-1.3s</span>
            <span className="text-slate-400 text-sm">vs last month</span>
          </div>
        </div>

        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center justify-between mb-2">
            <p className="text-slate-400 text-sm font-medium">Active Agents</p>
            <ArrowUpRight className="w-4 h-4 text-green-400" />
          </div>
          <p className="text-3xl font-bold text-white mb-2">{summary.agentCount}</p>
          <div className="flex items-center space-x-2">
            <span className="text-sm font-medium text-green-400">+12</span>
            <span className="text-slate-400 text-sm">vs last month</span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Orchestration Volume Chart */}
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-bold text-white">Orchestration Volume</h2>
            <div className="flex items-center space-x-2">
              <div className="flex items-center space-x-2">
                <div className="w-3 h-3 bg-gradient-to-r from-purple-500 to-blue-500 rounded-full"></div>
                <span className="text-slate-400 text-sm">Orchestrations</span>
              </div>
            </div>
          </div>
          
          <div className="space-y-4">
            {chartData.map((data, index) => (
              <div key={index} className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <span className="text-slate-300 text-sm font-medium w-8">{data.name}</span>
                  <div className="w-32 bg-slate-700 rounded-full h-2">
                    <div 
                      className="bg-gradient-to-r from-purple-500 to-blue-500 h-2 rounded-full transition-all duration-1000"
                      style={{ width: `${(data.orchestrations / 2100) * 100}%` }}
                    ></div>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-white font-medium">{data.orchestrations.toLocaleString()}</p>
                  <p className="text-slate-400 text-sm">{data.success}% success</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Target System Usage */}
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <h2 className="text-xl font-bold text-white mb-6">Target System Usage</h2>
          <div className="space-y-4">
            {targetSystemUsage.map((system, index) => (
              <div key={index} className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-slate-300 text-sm font-medium">{system.name}</span>
                  <div className="text-right">
                    <span className="text-white font-medium">{system.usage.toLocaleString()}</span>
                    <span className="text-slate-400 text-sm ml-2">({system.percentage}%)</span>
                  </div>
                </div>
                <div className="w-full bg-slate-700 rounded-full h-2">
                  <div 
                    className={`bg-gradient-to-r ${system.color} h-2 rounded-full transition-all duration-1000`}
                    style={{ width: `${system.percentage}%` }}
                  ></div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Agent Performance */}
        <div className="lg:col-span-2">
          <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold text-white">Agent Performance</h2>
              <button className="text-purple-400 hover:text-purple-300 text-sm font-medium">View Details</button>
            </div>
            
            <div className="space-y-4">
              {agentPerformance.map((agent, index) => (
                <div key={index} className="flex items-center justify-between p-4 rounded-lg bg-slate-700/30 border border-slate-600/30">
                  <div className="flex items-center space-x-4">
                    <div className={`p-2 rounded-lg ${
                      agent.trend === 'up' ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
                    }`}>
                      <Bot className="w-5 h-5" />
                    </div>
                    <div>
                      <p className="text-white font-medium">{agent.agent}</p>
                      <p className="text-slate-400 text-sm">{agent.orchestrations} orchestrations</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="flex items-center space-x-4">
                      <div>
                        <p className="text-white font-medium">{agent.successRate}%</p>
                        <p className="text-slate-400 text-sm">success rate</p>
                      </div>
                      <div>
                        <p className="text-white font-medium">{agent.avgTime}</p>
                        <p className="text-slate-400 text-sm">avg time</p>
                      </div>
                      <div className={`flex items-center space-x-1 ${
                        agent.trend === 'up' ? 'text-green-400' : 'text-red-400'
                      }`}>
                        {agent.trend === 'up' ? <ArrowUpRight className="w-4 h-4" /> : <ArrowDownRight className="w-4 h-4" />}
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Capability Trends & System Health */}
        <div className="space-y-6">
          <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-gradient-to-r from-green-500/20 to-emerald-500/20 rounded-lg border border-green-500/30">
                <Target className="w-5 h-5 text-green-400" />
              </div>
              <h3 className="text-lg font-bold text-white">System Health</h3>
            </div>
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-slate-400 text-sm">Agent Uptime</span>
                <span className="text-white font-medium">99.2%</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-slate-400 text-sm">Target Connectivity</span>
                <span className="text-white font-medium">97.8%</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-slate-400 text-sm">MCP Server Health</span>
                <span className="text-white font-medium">98.5%</span>
              </div>
            </div>
          </div>

          <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-gradient-to-r from-blue-500/20 to-cyan-500/20 rounded-lg border border-blue-500/30">
                <GitBranch className="w-5 h-5 text-blue-400" />
              </div>
              <h3 className="text-lg font-bold text-white">Capability Trends</h3>
            </div>
            <div className="space-y-3">
              {topCapabilities.map((trend, index) => (
                <div key={index} className="flex items-center justify-between">
                  <div>
                    <p className="text-white text-sm font-medium">{trend.name}</p>
                    <p className="text-slate-400 text-xs">{trend.volume} uses</p>
                  </div>
                  <div className={`flex items-center space-x-1 ${
                    trend.trend === 'up' ? 'text-green-400' : 'text-red-400'
                  }`}>
                    <span className="font-medium text-sm">{trend.growth}</span>
                    {trend.trend === 'up' ? <ArrowUpRight className="w-3 h-3" /> : <ArrowDownRight className="w-3 h-3" />}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-bold text-white">Recent Activity</h2>
          <button className="text-purple-400 hover:text-purple-300 text-sm font-medium">View All</button>
        </div>
        
        <div className="space-y-4">
          <div className="flex items-start space-x-4 p-3 rounded-lg hover:bg-slate-700/30 transition-colors duration-200">
            <div className="flex-shrink-0 mt-1">
              <div className="p-2 bg-green-500/20 rounded-full">
                <CheckCircle className="w-4 h-4 text-green-400" />
              </div>
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center space-x-2 mb-1">
                <p className="text-white font-medium truncate">CyberPunk Agent #7804</p>
                <span className="text-slate-400">→</span>
                <p className="text-slate-300 text-sm truncate">Chrome Browser</p>
              </div>
              <p className="text-slate-400 text-sm mb-1">Web Scraping Analysis</p>
              <p className="text-slate-500 text-xs">Extracted 247 data points from target website</p>
              <div className="flex items-center justify-between mt-2">
                <span className="px-2 py-1 rounded-full text-xs font-medium bg-green-500/20 text-green-400">
                  completed
                </span>
                <span className="text-slate-500 text-xs">2 minutes ago</span>
              </div>
            </div>
          </div>
          
          <div className="flex items-start space-x-4 p-3 rounded-lg hover:bg-slate-700/30 transition-colors duration-200">
            <div className="flex-shrink-0 mt-1">
              <div className="p-2 bg-blue-500/20 rounded-full">
                <Clock className="w-4 h-4 text-blue-400 animate-spin" />
              </div>
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center space-x-2 mb-1">
                <p className="text-white font-medium truncate">Data Miner #3749</p>
                <span className="text-slate-400">→</span>
                <p className="text-slate-300 text-sm truncate">Local File System</p>
              </div>
              <p className="text-slate-400 text-sm mb-1">Document Classification</p>
              <div className="flex items-center justify-between mt-2">
                <span className="px-2 py-1 rounded-full text-xs font-medium bg-blue-500/20 text-blue-400">
                  running
                </span>
                <span className="text-slate-500 text-xs">5 minutes ago</span>
              </div>
            </div>
          </div>
          
          <div className="flex items-start space-x-4 p-3 rounded-lg hover:bg-slate-700/30 transition-colors duration-200">
            <div className="flex-shrink-0 mt-1">
              <div className="p-2 bg-red-500/20 rounded-full">
                <AlertCircle className="w-4 h-4 text-red-400" />
              </div>
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center space-x-2 mb-1">
                <p className="text-white font-medium truncate">Code Reviewer #7894</p>
                <span className="text-slate-400">→</span>
                <p className="text-slate-300 text-sm truncate">VS Code Editor</p>
              </div>
              <p className="text-slate-400 text-sm mb-1">Code Quality Analysis</p>
              <p className="text-slate-500 text-xs">Error: Permission denied accessing editor context</p>
              <div className="flex items-center justify-between mt-2">
                <span className="px-2 py-1 rounded-full text-xs font-medium bg-red-500/20 text-red-400">
                  failed
                </span>
                <span className="text-slate-500 text-xs">18 minutes ago</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AnalyticsDashboard;