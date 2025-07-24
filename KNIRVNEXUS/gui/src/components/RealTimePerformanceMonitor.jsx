import React, { useState, useEffect, useRef } from 'react';
import { 
  Activity, 
  Cpu, 
  HardDrive, 
  Users, 
  Zap, 
  TrendingUp, 
  TrendingDown,
  Monitor,
  Server,
  Database,
  Network,
  AlertTriangle,
  CheckCircle,
  Clock,
  BarChart3,
  LineChart,
  PieChart,
  Maximize2,
  Minimize2
} from 'lucide-react';
import { subscribeToSystemMetrics, subscribeToAgentStatus } from '../utils/websocket';

const RealTimePerformanceMonitor = ({ isExpanded = false, onToggleExpand }) => {
  const [metrics, setMetrics] = useState({
    cpu: 0,
    memory: 0,
    disk: 0,
    network: 0,
    activeAgents: 0,
    totalInferences: 0,
    responseTime: 0,
    errorRate: 0,
    timestamp: new Date().toISOString(),
    history: {
      cpu: [],
      memory: [],
      disk: [],
      network: [],
      responseTime: [],
      errorRate: []
    }
  });

  const [agentMetrics, setAgentMetrics] = useState([]);
  const [alerts, setAlerts] = useState([]);
  const canvasRef = useRef(null);

  useEffect(() => {
    // Subscribe to real-time metrics
    const handleSystemMetrics = (update) => {
      setMetrics(prevMetrics => {
        const newHistory = {
          cpu: [...prevMetrics.history.cpu.slice(-29), { time: update.timestamp, value: update.cpu }],
          memory: [...prevMetrics.history.memory.slice(-29), { time: update.timestamp, value: update.memory }],
          disk: [...prevMetrics.history.disk.slice(-29), { time: update.timestamp, value: update.disk || Math.random() * 100 }],
          network: [...prevMetrics.history.network.slice(-29), { time: update.timestamp, value: update.network || Math.random() * 100 }],
          responseTime: [...prevMetrics.history.responseTime.slice(-29), { time: update.timestamp, value: update.responseTime || Math.random() * 1000 }],
          errorRate: [...prevMetrics.history.errorRate.slice(-29), { time: update.timestamp, value: update.errorRate || Math.random() * 5 }]
        };

        return {
          ...update,
          disk: update.disk || Math.random() * 100,
          network: update.network || Math.random() * 100,
          responseTime: update.responseTime || Math.random() * 1000,
          errorRate: update.errorRate || Math.random() * 5,
          history: newHistory
        };
      });

      // Check for alerts
      checkAlerts(update);
    };

    const handleAgentStatus = (update) => {
      setAgentMetrics(prev => {
        const existing = prev.find(a => a.id === update.agentId);
        if (existing) {
          return prev.map(a => a.id === update.agentId ? { ...a, ...update } : a);
        } else {
          return [...prev, { id: update.agentId, ...update }];
        }
      });
    };

    subscribeToSystemMetrics(handleSystemMetrics);
    subscribeToAgentStatus(handleAgentStatus);

    // Simulate real-time data for demo
    const interval = setInterval(() => {
      const mockUpdate = {
        cpu: Math.random() * 100,
        memory: Math.random() * 100,
        disk: Math.random() * 100,
        network: Math.random() * 100,
        activeAgents: Math.floor(Math.random() * 10) + 1,
        totalInferences: Math.floor(Math.random() * 1000) + 500,
        responseTime: Math.random() * 1000,
        errorRate: Math.random() * 5,
        timestamp: new Date().toISOString()
      };
      handleSystemMetrics(mockUpdate);
    }, 2000);

    return () => {
      clearInterval(interval);
    };
  }, []);

  const checkAlerts = (metrics) => {
    const newAlerts = [];

    if (metrics.cpu && metrics.cpu > 80) {
      newAlerts.push({
        id: 'cpu-high',
        type: 'warning',
        message: `High CPU usage: ${(metrics.cpu || 0).toFixed(1)}%`,
        timestamp: new Date().toISOString()
      });
    }

    if (metrics.memory && metrics.memory > 85) {
      newAlerts.push({
        id: 'memory-high',
        type: 'critical',
        message: `High memory usage: ${(metrics.memory || 0).toFixed(1)}%`,
        timestamp: new Date().toISOString()
      });
    }

    if (metrics.errorRate && metrics.errorRate > 3) {
      newAlerts.push({
        id: 'error-rate-high',
        type: 'critical',
        message: `High error rate: ${(metrics.errorRate || 0).toFixed(1)}%`,
        timestamp: new Date().toISOString()
      });
    }

    setAlerts(prev => {
      const filtered = prev.filter(alert => 
        !newAlerts.some(newAlert => newAlert.id === alert.id)
      );
      return [...filtered, ...newAlerts].slice(-5); // Keep last 5 alerts
    });
  };

  const renderMiniChart = (data, color) => {
    if (!canvasRef.current || data.length === 0) return null;

    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    const width = 100;
    const height = 30;
    
    canvas.width = width;
    canvas.height = height;
    
    ctx.clearRect(0, 0, width, height);
    
    const maxValue = Math.max(...data.map(d => d.value));
    const minValue = Math.min(...data.map(d => d.value));
    const range = maxValue - minValue || 1;
    
    ctx.strokeStyle = color;
    ctx.lineWidth = 2;
    ctx.beginPath();
    
    data.forEach((point, index) => {
      const x = (index / (data.length - 1)) * width;
      const y = height - ((point.value - minValue) / range) * height;
      
      if (index === 0) {
        ctx.moveTo(x, y);
      } else {
        ctx.lineTo(x, y);
      }
    });
    
    ctx.stroke();
  };

  const getMetricColor = (value, thresholds = { warning: 60, critical: 80 }) => {
    if (value > thresholds.critical) return 'text-red-400';
    if (value > thresholds.warning) return 'text-yellow-400';
    return 'text-green-400';
  };

  const formatTime = (timestamp) => {
    return new Date(timestamp).toLocaleTimeString();
  };

  const formatValue = (value, unit = '') => {
    if (typeof value !== 'number') return '0' + unit;
    return value.toFixed(1) + unit;
  };

  if (!isExpanded) {
    // Compact view
    return (
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-4 border border-slate-700/50">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-lg font-semibold text-white flex items-center space-x-2">
            <Activity className="w-5 h-5 text-blue-400" />
            <span>Performance</span>
          </h3>
          <button
            onClick={onToggleExpand}
            className="text-slate-400 hover:text-white transition-colors"
          >
            <Maximize2 className="w-4 h-4" />
          </button>
        </div>
        
        <div className="grid grid-cols-2 gap-3">
          <div className="text-center">
            <div className={`text-xl font-bold ${getMetricColor(metrics.cpu)}`}>
              {formatValue(metrics.cpu, '%')}
            </div>
            <div className="text-xs text-slate-400">CPU</div>
          </div>
          <div className="text-center">
            <div className={`text-xl font-bold ${getMetricColor(metrics.memory)}`}>
              {formatValue(metrics.memory, '%')}
            </div>
            <div className="text-xs text-slate-400">Memory</div>
          </div>
        </div>
        
        {alerts.length > 0 && (
          <div className="mt-3 flex items-center space-x-1">
            <AlertTriangle className="w-4 h-4 text-red-400" />
            <span className="text-xs text-red-400">{alerts.length} alert(s)</span>
          </div>
        )}
      </div>
    );
  }

  // Expanded view
  return (
    <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-xl font-semibold text-white flex items-center space-x-2">
          <Activity className="w-6 h-6 text-blue-400" />
          <span>Real-Time Performance Monitor</span>
        </h3>
        <button
          onClick={onToggleExpand}
          className="text-slate-400 hover:text-white transition-colors"
        >
          <Minimize2 className="w-5 h-5" />
        </button>
      </div>

      {/* Alerts */}
      {alerts.length > 0 && (
        <div className="mb-6">
          <h4 className="text-sm font-medium text-white mb-2">Active Alerts</h4>
          <div className="space-y-2">
            {alerts.map((alert) => (
              <div
                key={alert.id}
                className={`p-3 rounded-lg border flex items-center space-x-2 ${
                  alert.type === 'critical' 
                    ? 'bg-red-900/20 border-red-500/30 text-red-200'
                    : 'bg-yellow-900/20 border-yellow-500/30 text-yellow-200'
                }`}
              >
                <AlertTriangle className="w-4 h-4" />
                <span className="text-sm">{alert.message}</span>
                <span className="text-xs opacity-70 ml-auto">
                  {formatTime(alert.timestamp)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        {[
          { label: 'CPU Usage', value: metrics.cpu, unit: '%', icon: Cpu, history: metrics.history.cpu },
          { label: 'Memory', value: metrics.memory, unit: '%', icon: HardDrive, history: metrics.history.memory },
          { label: 'Response Time', value: metrics.responseTime, unit: 'ms', icon: Clock, history: metrics.history.responseTime },
          { label: 'Error Rate', value: metrics.errorRate, unit: '%', icon: AlertTriangle, history: metrics.history.errorRate }
        ].map((metric, index) => {
          const Icon = metric.icon;
          return (
            <div key={index} className="bg-slate-700/50 rounded-lg p-4">
              <div className="flex items-center justify-between mb-2">
                <Icon className="w-5 h-5 text-slate-400" />
                <span className="text-xs text-slate-500">
                  {formatTime(metrics.timestamp)}
                </span>
              </div>
              
              <div className="space-y-2">
                <div className={`text-2xl font-bold ${getMetricColor(metric.value)}`}>
                  {formatValue(metric.value, metric.unit)}
                </div>
                <div className="text-xs text-slate-400">{metric.label}</div>
                
                <div className="h-8">
                  <canvas
                    ref={canvasRef}
                    className="w-full h-full"
                    onLoad={() => renderMiniChart(metric.history, '#3b82f6')}
                  />
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Agent Performance */}
      <div>
        <h4 className="text-lg font-medium text-white mb-3">Agent Performance</h4>
        <div className="space-y-2">
          {agentMetrics.slice(0, 5).map((agent, index) => (
            <div key={agent.id} className="flex items-center justify-between p-3 bg-slate-700/30 rounded-lg">
              <div className="flex items-center space-x-3">
                <div className={`w-2 h-2 rounded-full ${
                  agent.status === 'active' ? 'bg-green-400' : 'bg-slate-400'
                }`} />
                <span className="text-white font-medium">{agent.name || `Agent ${agent.id}`}</span>
              </div>
              <div className="flex items-center space-x-4 text-sm">
                <span className="text-slate-300">{agent.inferences || 0} inferences</span>
                <span className={`${getMetricColor(agent.successRate || 95)}`}>
                  {formatValue(agent.successRate || 95, '%')} success
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default RealTimePerformanceMonitor;
