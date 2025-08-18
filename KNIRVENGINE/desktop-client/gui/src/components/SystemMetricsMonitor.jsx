import React, { useState, useEffect } from 'react';
import { Activity, Cpu, HardDrive, Users, Zap, TrendingUp, TrendingDown } from 'lucide-react';
import { subscribeToSystemMetrics, SystemMetricsUpdate } from '../utils/websocket';

const SystemMetricsMonitor = () => {
  const [metrics, setMetrics] = useState({
    cpu: 0,
    memory: 0,
    activeAgents: 0,
    totalInferences: 0,
    timestamp: new Date().toISOString(),
    history: {
      cpu: [],
      memory: [],
      activeAgents: [],
      totalInferences: []
    }
  });

  useEffect(() => {
    // Subscribe to system metrics updates
    const handleSystemMetrics = (update) => {
      setMetrics(prevMetrics => {
        const newHistory = {
          cpu: [...prevMetrics.history.cpu.slice(-19), { time: update.timestamp, value: update.cpu }],
          memory: [...prevMetrics.history.memory.slice(-19), { time: update.timestamp, value: update.memory }],
          activeAgents: [...prevMetrics.history.activeAgents.slice(-19), { time: update.timestamp, value: update.activeAgents }],
          totalInferences: [...prevMetrics.history.totalInferences.slice(-19), { time: update.timestamp, value: update.totalInferences }]
        };

        return {
          cpu: update.cpu,
          memory: update.memory,
          activeAgents: update.activeAgents,
          totalInferences: update.totalInferences,
          timestamp: update.timestamp,
          history: newHistory
        };
      });
    };

    subscribeToSystemMetrics(handleSystemMetrics);

    // Simulate initial data and periodic updates for demo
    const interval = setInterval(() => {
      const mockUpdate = {
        cpu: Math.random() * 100,
        memory: Math.random() * 100,
        activeAgents: Math.floor(Math.random() * 10) + 1,
        totalInferences: Math.floor(Math.random() * 1000) + 500,
        timestamp: new Date().toISOString()
      };
      handleSystemMetrics(mockUpdate);
    }, 2000);

    return () => {
      clearInterval(interval);
      // In a real implementation, unsubscribe from WebSocket events here
    };
  }, []);

  const getMetricColor = (value, threshold = 80) => {
    if (value > threshold) return 'text-red-400';
    if (value > threshold * 0.7) return 'text-yellow-400';
    return 'text-green-400';
  };

  const getTrend = (history) => {
    if (history.length < 2) return 0;
    const recent = history.slice(-3);
    const avg = recent.reduce((sum, item) => sum + item.value, 0) / recent.length;
    const previous = history.slice(-6, -3);
    if (previous.length === 0) return 0;
    const prevAvg = previous.reduce((sum, item) => sum + item.value, 0) / previous.length;
    return avg - prevAvg;
  };

  const formatTime = (timestamp) => {
    return new Date(timestamp).toLocaleTimeString();
  };

  const renderMiniChart = (data, color = 'blue') => {
    if (data.length < 2) return null;

    const max = Math.max(...data.map(d => d.value));
    const min = Math.min(...data.map(d => d.value));
    const range = max - min || 1;

    const points = data.map((point, index) => {
      const x = (index / (data.length - 1)) * 100;
      const y = 100 - ((point.value - min) / range) * 100;
      return `${x},${y}`;
    }).join(' ');

    return (
      <svg className="w-full h-8" viewBox="0 0 100 100" preserveAspectRatio="none">
        <polyline
          fill="none"
          stroke={`rgb(var(--color-${color}-400))`}
          strokeWidth="2"
          points={points}
        />
      </svg>
    );
  };

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {/* CPU Usage */}
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-4 border border-slate-700/50">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center space-x-2">
            <Cpu className="w-5 h-5 text-blue-400" />
            <span className="text-slate-300 font-medium">CPU</span>
          </div>
          {getTrend(metrics.history.cpu) > 0 ? (
            <TrendingUp className="w-4 h-4 text-red-400" />
          ) : (
            <TrendingDown className="w-4 h-4 text-green-400" />
          )}
        </div>
        
        <div className="space-y-2">
          <div className="flex items-end justify-between">
            <span className={`text-2xl font-bold ${getMetricColor(metrics.cpu)}`}>
              {(metrics.cpu || 0).toFixed(1)}%
            </span>
            <span className="text-xs text-slate-500">
              {formatTime(metrics.timestamp)}
            </span>
          </div>
          
          <div className="w-full bg-slate-700 rounded-full h-2">
            <div 
              className={`h-2 rounded-full transition-all duration-300 ${
                metrics.cpu > 80 ? 'bg-red-500' :
                metrics.cpu > 60 ? 'bg-yellow-500' :
                'bg-green-500'
              }`}
              style={{ width: `${metrics.cpu}%` }}
            ></div>
          </div>
          
          <div className="h-8">
            {renderMiniChart(metrics.history.cpu, 'blue')}
          </div>
        </div>
      </div>

      {/* Memory Usage */}
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-4 border border-slate-700/50">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center space-x-2">
            <HardDrive className="w-5 h-5 text-purple-400" />
            <span className="text-slate-300 font-medium">Memory</span>
          </div>
          {getTrend(metrics.history.memory) > 0 ? (
            <TrendingUp className="w-4 h-4 text-red-400" />
          ) : (
            <TrendingDown className="w-4 h-4 text-green-400" />
          )}
        </div>
        
        <div className="space-y-2">
          <div className="flex items-end justify-between">
            <span className={`text-2xl font-bold ${getMetricColor(metrics.memory)}`}>
              {(metrics.memory || 0).toFixed(1)}%
            </span>
            <span className="text-xs text-slate-500">
              {formatTime(metrics.timestamp)}
            </span>
          </div>
          
          <div className="w-full bg-slate-700 rounded-full h-2">
            <div 
              className={`h-2 rounded-full transition-all duration-300 ${
                metrics.memory > 80 ? 'bg-red-500' :
                metrics.memory > 60 ? 'bg-yellow-500' :
                'bg-green-500'
              }`}
              style={{ width: `${metrics.memory}%` }}
            ></div>
          </div>
          
          <div className="h-8">
            {renderMiniChart(metrics.history.memory, 'purple')}
          </div>
        </div>
      </div>

      {/* Active Agents */}
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-4 border border-slate-700/50">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center space-x-2">
            <Users className="w-5 h-5 text-green-400" />
            <span className="text-slate-300 font-medium">Active Agents</span>
          </div>
          {getTrend(metrics.history.activeAgents) > 0 ? (
            <TrendingUp className="w-4 h-4 text-green-400" />
          ) : (
            <TrendingDown className="w-4 h-4 text-red-400" />
          )}
        </div>
        
        <div className="space-y-2">
          <div className="flex items-end justify-between">
            <span className="text-2xl font-bold text-green-400">
              {metrics.activeAgents}
            </span>
            <span className="text-xs text-slate-500">
              {formatTime(metrics.timestamp)}
            </span>
          </div>
          
          <div className="h-8">
            {renderMiniChart(metrics.history.activeAgents, 'green')}
          </div>
        </div>
      </div>

      {/* Total Inferences */}
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-4 border border-slate-700/50">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center space-x-2">
            <Zap className="w-5 h-5 text-yellow-400" />
            <span className="text-slate-300 font-medium">Inferences</span>
          </div>
          {getTrend(metrics.history.totalInferences) > 0 ? (
            <TrendingUp className="w-4 h-4 text-green-400" />
          ) : (
            <TrendingDown className="w-4 h-4 text-red-400" />
          )}
        </div>
        
        <div className="space-y-2">
          <div className="flex items-end justify-between">
            <span className="text-2xl font-bold text-yellow-400">
              {metrics.totalInferences.toLocaleString()}
            </span>
            <span className="text-xs text-slate-500">
              {formatTime(metrics.timestamp)}
            </span>
          </div>
          
          <div className="h-8">
            {renderMiniChart(metrics.history.totalInferences, 'yellow')}
          </div>
        </div>
      </div>
    </div>
  );
};

export default SystemMetricsMonitor;
