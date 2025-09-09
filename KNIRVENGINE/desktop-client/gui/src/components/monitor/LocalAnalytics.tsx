import React, { useState, useEffect } from 'react';
import { BarChart3, TrendingUp, TrendingDown, Activity, Cpu, HardDrive, Wifi } from 'lucide-react';
import { useAuth } from '../AuthContext';

interface SystemMetrics {
  cpu: number;
  memory: number;
  disk: number;
  network: {
    upload: number;
    download: number;
  };
  timestamp: Date;
}

interface PerformanceData {
  label: string;
  value: number;
  change: number;
  trend: 'up' | 'down' | 'stable';
}

const LocalAnalytics: React.FC = () => {
  const { user } = useAuth();
  const [metrics, setMetrics] = useState<SystemMetrics[]>([]);
  const [currentMetrics, setCurrentMetrics] = useState<SystemMetrics | null>(null);
  const [performanceData, setPerformanceData] = useState<PerformanceData[]>([]);

  useEffect(() => {
    const generateMetrics = (): SystemMetrics => ({
      cpu: Math.random() * 100,
      memory: Math.random() * 100,
      disk: Math.random() * 100,
      network: {
        upload: Math.random() * 1000,
        download: Math.random() * 5000
      },
      timestamp: new Date()
    });

    const updateMetrics = () => {
      const newMetrics = generateMetrics();
      setCurrentMetrics(newMetrics);
      setMetrics(prev => [...prev.slice(-19), newMetrics]); // Keep last 20 data points
      
      // Update performance data
      setPerformanceData([
        {
          label: 'Agent Responses',
          value: Math.floor(Math.random() * 1000) + 500,
          change: (Math.random() - 0.5) * 20,
          trend: Math.random() > 0.5 ? 'up' : 'down'
        },
        {
          label: 'Skills Executed',
          value: Math.floor(Math.random() * 100) + 50,
          change: (Math.random() - 0.5) * 10,
          trend: Math.random() > 0.5 ? 'up' : 'down'
        },
        {
          label: 'Network Requests',
          value: Math.floor(Math.random() * 5000) + 1000,
          change: (Math.random() - 0.5) * 50,
          trend: Math.random() > 0.5 ? 'up' : 'down'
        },
        {
          label: 'Data Processed',
          value: Math.floor(Math.random() * 1000) + 200,
          change: (Math.random() - 0.5) * 30,
          trend: Math.random() > 0.5 ? 'up' : 'down'
        }
      ]);
    };

    // Initial load
    updateMetrics();
    
    // Update every 5 seconds
    const interval = setInterval(updateMetrics, 5000);
    return () => clearInterval(interval);
  }, []);

  const getProgressBarColor = (value: number) => {
    if (value < 50) return 'bg-green-500';
    if (value < 80) return 'bg-yellow-500';
    return 'bg-red-500';
  };

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'up':
        return <TrendingUp className="w-4 h-4 text-green-500" />;
      case 'down':
        return <TrendingDown className="w-4 h-4 text-red-500" />;
      default:
        return <Activity className="w-4 h-4 text-slate-400" />;
    }
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-green-500/20 rounded-lg">
          <BarChart3 className="w-6 h-6 text-green-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Local Analytics</h1>
          <p className="text-slate-400">System performance and usage metrics</p>
        </div>
      </div>

      {/* System Metrics */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {/* Real-time System Stats */}
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-white mb-4">System Resources</h2>
          
          {currentMetrics && (
            <div className="space-y-4">
              <div>
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    <Cpu className="w-4 h-4 text-blue-400" />
                    <span className="text-sm text-slate-300">CPU Usage</span>
                  </div>
                  <span className="text-sm text-white">{currentMetrics.cpu.toFixed(1)}%</span>
                </div>
                <div className="w-full bg-slate-700 rounded-full h-2">
                  <div 
                    className={`h-2 rounded-full ${getProgressBarColor(currentMetrics.cpu)}`}
                    style={{ width: `${currentMetrics.cpu}%` }}
                  ></div>
                </div>
              </div>
              
              <div>
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    <Activity className="w-4 h-4 text-purple-400" />
                    <span className="text-sm text-slate-300">Memory Usage</span>
                  </div>
                  <span className="text-sm text-white">{currentMetrics.memory.toFixed(1)}%</span>
                </div>
                <div className="w-full bg-slate-700 rounded-full h-2">
                  <div 
                    className={`h-2 rounded-full ${getProgressBarColor(currentMetrics.memory)}`}
                    style={{ width: `${currentMetrics.memory}%` }}
                  ></div>
                </div>
              </div>
              
              <div>
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    <HardDrive className="w-4 h-4 text-yellow-400" />
                    <span className="text-sm text-slate-300">Disk Usage</span>
                  </div>
                  <span className="text-sm text-white">{currentMetrics.disk.toFixed(1)}%</span>
                </div>
                <div className="w-full bg-slate-700 rounded-full h-2">
                  <div 
                    className={`h-2 rounded-full ${getProgressBarColor(currentMetrics.disk)}`}
                    style={{ width: `${currentMetrics.disk}%` }}
                  ></div>
                </div>
              </div>
              
              <div className="grid grid-cols-2 gap-4 pt-2">
                <div>
                  <div className="flex items-center space-x-2 mb-1">
                    <Wifi className="w-4 h-4 text-green-400" />
                    <span className="text-xs text-slate-400">Upload</span>
                  </div>
                  <span className="text-sm text-white">{currentMetrics.network.upload.toFixed(1)} KB/s</span>
                </div>
                <div>
                  <div className="flex items-center space-x-2 mb-1">
                    <Wifi className="w-4 h-4 text-blue-400" />
                    <span className="text-xs text-slate-400">Download</span>
                  </div>
                  <span className="text-sm text-white">{currentMetrics.network.download.toFixed(1)} KB/s</span>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Performance Metrics */}
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-white mb-4">Performance Metrics</h2>
          
          <div className="space-y-4">
            {performanceData.map((item, index) => (
              <div key={index} className="flex items-center justify-between p-3 bg-slate-700/30 rounded-lg">
                <div>
                  <div className="text-sm font-medium text-white">{item.label}</div>
                  <div className="text-xs text-slate-400">Last 24 hours</div>
                </div>
                <div className="text-right">
                  <div className="flex items-center space-x-2">
                    <span className="text-lg font-bold text-white">{item.value.toLocaleString()}</span>
                    {getTrendIcon(item.trend)}
                  </div>
                  <div className={`text-xs ${item.change >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                    {item.change >= 0 ? '+' : ''}{item.change.toFixed(1)}%
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Activity Timeline */}
      <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Recent Activity</h2>
        
        <div className="space-y-3">
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Agent skill execution completed</div>
              <div className="text-xs text-slate-400">2 minutes ago</div>
            </div>
            <div className="text-xs text-green-400">Success</div>
          </div>
          
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Network connection established</div>
              <div className="text-xs text-slate-400">5 minutes ago</div>
            </div>
            <div className="text-xs text-blue-400">Info</div>
          </div>
          
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">High memory usage detected</div>
              <div className="text-xs text-slate-400">8 minutes ago</div>
            </div>
            <div className="text-xs text-yellow-400">Warning</div>
          </div>
          
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Data processing batch completed</div>
              <div className="text-xs text-slate-400">12 minutes ago</div>
            </div>
            <div className="text-xs text-purple-400">Process</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default LocalAnalytics;
