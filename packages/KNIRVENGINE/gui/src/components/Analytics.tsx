import React, { useState, useEffect } from 'react';
import { BarChart3, TrendingUp, Activity, Users, Zap, Clock } from 'lucide-react';
import { useAuth } from './AuthContext';
import { GlassCard } from './common';

interface AnalyticsData {
  totalAgents: number;
  activeSkills: number;
  networkRequests: number;
  uptime: string;
  performance: {
    cpu: number;
    memory: number;
    network: number;
  };
  recentActivity: Array<{
    id: string;
    type: string;
    description: string;
    timestamp: Date;
  }>;
}

const Analytics: React.FC = () => {
  const { user } = useAuth();
  const [analyticsData, setAnalyticsData] = useState<AnalyticsData>({
    totalAgents: 12,
    activeSkills: 8,
    networkRequests: 1847,
    uptime: '99.8%',
    performance: {
      cpu: 45,
      memory: 62,
      network: 78
    },
    recentActivity: [
      {
        id: '1',
        type: 'agent',
        description: 'Agent "DataProcessor" completed task successfully',
        timestamp: new Date(Date.now() - 5 * 60 * 1000)
      },
      {
        id: '2',
        type: 'skill',
        description: 'New skill "TextAnalyzer" installed',
        timestamp: new Date(Date.now() - 15 * 60 * 1000)
      },
      {
        id: '3',
        type: 'network',
        description: 'Connected to 3 new peer nodes',
        timestamp: new Date(Date.now() - 30 * 60 * 1000)
      }
    ]
  });

  const formatTimeAgo = (date: Date) => {
    const now = new Date();
    const diffInMinutes = Math.floor((now.getTime() - date.getTime()) / (1000 * 60));
    
    if (diffInMinutes < 1) return 'Just now';
    if (diffInMinutes < 60) return `${diffInMinutes}m ago`;
    if (diffInMinutes < 1440) return `${Math.floor(diffInMinutes / 60)}h ago`;
    return `${Math.floor(diffInMinutes / 1440)}d ago`;
  };

  const getActivityIcon = (type: string) => {
    switch (type) {
      case 'agent': return <Users className="w-4 h-4 text-blue-400" />;
      case 'skill': return <Zap className="w-4 h-4 text-purple-400" />;
      case 'network': return <Activity className="w-4 h-4 text-green-400" />;
      default: return <Activity className="w-4 h-4 text-slate-400" />;
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center space-x-3 mb-6">
        <BarChart3 className="w-8 h-8 text-blue-400" />
        <h1 className="text-2xl font-bold text-white">Analytics</h1>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <GlassCard>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-slate-400 text-sm">Total Agents</p>
              <p className="text-2xl font-bold text-white">{analyticsData.totalAgents}</p>
            </div>
            <Users className="w-8 h-8 text-blue-400" />
          </div>
        </GlassCard>

        <GlassCard>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-slate-400 text-sm">Active Skills</p>
              <p className="text-2xl font-bold text-white">{analyticsData.activeSkills}</p>
            </div>
            <Zap className="w-8 h-8 text-purple-400" />
          </div>
        </GlassCard>

        <GlassCard>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-slate-400 text-sm">Network Requests</p>
              <p className="text-2xl font-bold text-white">{analyticsData.networkRequests.toLocaleString()}</p>
            </div>
            <TrendingUp className="w-8 h-8 text-green-400" />
          </div>
        </GlassCard>

        <GlassCard>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-slate-400 text-sm">Uptime</p>
              <p className="text-2xl font-bold text-white">{analyticsData.uptime}</p>
            </div>
            <Clock className="w-8 h-8 text-yellow-400" />
          </div>
        </GlassCard>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Performance Metrics */}
        <GlassCard title="System Performance">
          <div className="space-y-4">
            <div>
              <div className="flex justify-between items-center mb-2">
                <span className="text-slate-300">CPU Usage</span>
                <span className="text-white">{analyticsData.performance.cpu}%</span>
              </div>
              <div className="w-full bg-slate-700/50 rounded-full h-2">
                <div 
                  className="bg-blue-500 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${analyticsData.performance.cpu}%` }}
                ></div>
              </div>
            </div>

            <div>
              <div className="flex justify-between items-center mb-2">
                <span className="text-slate-300">Memory Usage</span>
                <span className="text-white">{analyticsData.performance.memory}%</span>
              </div>
              <div className="w-full bg-slate-700/50 rounded-full h-2">
                <div 
                  className="bg-purple-500 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${analyticsData.performance.memory}%` }}
                ></div>
              </div>
            </div>

            <div>
              <div className="flex justify-between items-center mb-2">
                <span className="text-slate-300">Network Usage</span>
                <span className="text-white">{analyticsData.performance.network}%</span>
              </div>
              <div className="w-full bg-slate-700/50 rounded-full h-2">
                <div 
                  className="bg-green-500 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${analyticsData.performance.network}%` }}
                ></div>
              </div>
            </div>
          </div>
        </GlassCard>

        {/* Recent Activity */}
        <GlassCard title="Recent Activity">
          <div className="space-y-3">
            {analyticsData.recentActivity.map((activity) => (
              <div key={activity.id} className="flex items-start space-x-3 p-3 bg-slate-700/30 rounded-lg">
                <div className="mt-1">
                  {getActivityIcon(activity.type)}
                </div>
                <div className="flex-1">
                  <p className="text-white text-sm">{activity.description}</p>
                  <p className="text-slate-400 text-xs mt-1">{formatTimeAgo(activity.timestamp)}</p>
                </div>
              </div>
            ))}
          </div>
        </GlassCard>
      </div>

      {/* Usage Trends */}
      <div className="mt-8">
        <GlassCard title="Usage Trends">
          <div className="h-64 flex items-center justify-center">
            <div className="text-center">
              <BarChart3 className="w-16 h-16 text-slate-600 mx-auto mb-4" />
              <p className="text-slate-400">Chart visualization coming soon</p>
              <p className="text-slate-500 text-sm">Integration with analytics engine in progress</p>
            </div>
          </div>
        </GlassCard>
      </div>
    </div>
  );
};

export default Analytics;
