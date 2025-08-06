import React, { useState, useEffect } from 'react';
import {
  Activity,
  BarChart3,
  TrendingUp,
  Clock,
  Cpu,
  Database,
  HardDrive,
  Zap,
  Filter,
  Download,
  RefreshCw,
  AlertTriangle,
  CheckCircle
} from 'lucide-react';

const PerformanceObservability = () => {
  const [selectedTimeRange, setSelectedTimeRange] = useState('1h');
  const [selectedMetric, setSelectedMetric] = useState('latency');
  const [loading, setLoading] = useState(false);
  const [performanceData, setPerformanceData] = useState({});

  // Mock performance data
  useEffect(() => {
    const mockData = {
      latency: {
        current: '45ms',
        trend: '+2.3%',
        data: [42, 45, 43, 47, 44, 46, 45, 48, 45, 44]
      },
      throughput: {
        current: '1,247 req/s',
        trend: '+15.2%',
        data: [1100, 1150, 1200, 1180, 1220, 1247, 1230, 1260, 1247, 1255]
      },
      resourceUtilization: {
        cpu: 68,
        memory: 72,
        storage: 45,
        teeUsage: 82
      },
      tasks: [
        {
          id: 'task-001',
          name: 'ML Inference Pipeline',
          status: 'running',
          latency: '42ms',
          throughput: '850 req/s',
          cpuUsage: 75,
          memoryUsage: 68,
          teeType: 'Intel SGX'
        },
        {
          id: 'task-002',
          name: 'Data Processing Task',
          status: 'running',
          latency: '38ms',
          throughput: '397 req/s',
          cpuUsage: 45,
          memoryUsage: 52,
          teeType: 'AMD SEV-SNP'
        },
        {
          id: 'task-003',
          name: 'Security Analysis',
          status: 'completed',
          latency: '125ms',
          throughput: '156 req/s',
          cpuUsage: 32,
          memoryUsage: 28,
          teeType: 'Intel TDX'
        }
      ]
    };
    setPerformanceData(mockData);
  }, [selectedTimeRange]);

  const getStatusColor = (status) => {
    switch (status) {
      case 'running':
        return 'text-green-400';
      case 'completed':
        return 'text-blue-400';
      case 'failed':
        return 'text-red-400';
      default:
        return 'text-gray-400';
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'running':
        return <Activity className="w-4 h-4 text-green-400" />;
      case 'completed':
        return <CheckCircle className="w-4 h-4 text-blue-400" />;
      case 'failed':
        return <AlertTriangle className="w-4 h-4 text-red-400" />;
      default:
        return <Clock className="w-4 h-4 text-gray-400" />;
    }
  };

  const MetricCard = ({ title, value, trend, icon: Icon, color }) => (
    <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-3">
          <div className={`p-2 rounded-lg bg-gradient-to-r ${color}`}>
            <Icon className="w-6 h-6 text-white" />
          </div>
          <h3 className="text-lg font-semibold text-white">{title}</h3>
        </div>
        <span className="text-sm text-green-400">{trend}</span>
      </div>
      <p className="text-3xl font-bold text-white">{value}</p>
    </div>
  );

  const ResourceGauge = ({ label, value, max = 100, color }) => (
    <div className="bg-slate-700 rounded-lg p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-slate-300">{label}</span>
        <span className="text-sm text-white font-medium">{value}%</span>
      </div>
      <div className="w-full bg-slate-600 rounded-full h-2">
        <div
          className={`h-2 rounded-full bg-gradient-to-r ${color}`}
          style={{ width: `${(value / max) * 100}%` }}
        />
      </div>
    </div>
  );

  return (
    <div className="min-h-screen bg-knirv-gradient p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div className="flex items-center space-x-3">
            <BarChart3 className="w-8 h-8 text-knirv-primary" />
            <div>
              <h1 className="text-3xl font-bold text-white">Performance & Observability</h1>
              <p className="text-slate-300">Deep insights into task performance and resource utilization</p>
            </div>
          </div>
          <div className="flex items-center space-x-4">
            <select
              value={selectedTimeRange}
              onChange={(e) => setSelectedTimeRange(e.target.value)}
              className="bg-slate-800 text-white rounded-lg px-3 py-2 border border-slate-600"
            >
              <option value="1h">Last Hour</option>
              <option value="24h">Last 24 Hours</option>
              <option value="7d">Last 7 Days</option>
              <option value="30d">Last 30 Days</option>
            </select>
            <button
              onClick={() => setLoading(true)}
              className="flex items-center space-x-2 px-4 py-2 bg-knirv-primary text-white rounded-lg hover:bg-knirv-secondary transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              <span>Refresh</span>
            </button>
          </div>
        </div>

        {/* Key Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <MetricCard
            title="Avg Latency"
            value={performanceData.latency?.current || '0ms'}
            trend={performanceData.latency?.trend || '0%'}
            icon={Clock}
            color="from-knirv-primary to-knirv-secondary"
          />
          <MetricCard
            title="Throughput"
            value={performanceData.throughput?.current || '0 req/s'}
            trend={performanceData.throughput?.trend || '0%'}
            icon={TrendingUp}
            color="from-green-500 to-emerald-500"
          />
          <MetricCard
            title="Active Tasks"
            value={performanceData.tasks?.filter(t => t.status === 'running').length || 0}
            trend="+8.5%"
            icon={Activity}
            color="from-blue-500 to-cyan-500"
          />
          <MetricCard
            title="TEE Efficiency"
            value="94.2%"
            trend="+3.1%"
            icon={Zap}
            color="from-purple-500 to-pink-500"
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Resource Utilization */}
          <div className="bg-slate-800 rounded-lg border border-slate-700">
            <div className="p-6 border-b border-slate-700">
              <div className="flex items-center space-x-3">
                <Cpu className="w-6 h-6 text-knirv-primary" />
                <h2 className="text-xl font-semibold text-white">Resource Utilization</h2>
              </div>
            </div>
            <div className="p-6 space-y-4">
              <ResourceGauge
                label="CPU Usage"
                value={performanceData.resourceUtilization?.cpu || 0}
                color="from-blue-500 to-cyan-500"
              />
              <ResourceGauge
                label="Memory Usage"
                value={performanceData.resourceUtilization?.memory || 0}
                color="from-green-500 to-emerald-500"
              />
              <ResourceGauge
                label="Storage Usage"
                value={performanceData.resourceUtilization?.storage || 0}
                color="from-yellow-500 to-orange-500"
              />
              <ResourceGauge
                label="TEE Usage"
                value={performanceData.resourceUtilization?.teeUsage || 0}
                color="from-knirv-primary to-knirv-secondary"
              />
            </div>
          </div>

          {/* Task Performance */}
          <div className="lg:col-span-2 bg-slate-800 rounded-lg border border-slate-700">
            <div className="p-6 border-b border-slate-700">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <Activity className="w-6 h-6 text-knirv-primary" />
                  <h2 className="text-xl font-semibold text-white">Task Performance</h2>
                </div>
                <button className="flex items-center space-x-2 px-3 py-1 bg-slate-700 text-white rounded-lg hover:bg-slate-600 transition-colors">
                  <Download className="w-4 h-4" />
                  <span>Export</span>
                </button>
              </div>
            </div>
            <div className="p-6">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-slate-700">
                      <th className="text-left py-3 px-4 text-slate-300 font-medium">Task</th>
                      <th className="text-left py-3 px-4 text-slate-300 font-medium">Status</th>
                      <th className="text-left py-3 px-4 text-slate-300 font-medium">Latency</th>
                      <th className="text-left py-3 px-4 text-slate-300 font-medium">Throughput</th>
                      <th className="text-left py-3 px-4 text-slate-300 font-medium">CPU</th>
                      <th className="text-left py-3 px-4 text-slate-300 font-medium">Memory</th>
                      <th className="text-left py-3 px-4 text-slate-300 font-medium">TEE Type</th>
                    </tr>
                  </thead>
                  <tbody>
                    {performanceData.tasks?.map((task) => (
                      <tr key={task.id} className="border-b border-slate-700/50 hover:bg-slate-700/30">
                        <td className="py-3 px-4">
                          <div>
                            <p className="text-white font-medium">{task.name}</p>
                            <p className="text-xs text-slate-400">{task.id}</p>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center space-x-2">
                            {getStatusIcon(task.status)}
                            <span className={`capitalize ${getStatusColor(task.status)}`}>
                              {task.status}
                            </span>
                          </div>
                        </td>
                        <td className="py-3 px-4 text-white">{task.latency}</td>
                        <td className="py-3 px-4 text-white">{task.throughput}</td>
                        <td className="py-3 px-4">
                          <div className="flex items-center space-x-2">
                            <div className="w-12 bg-slate-600 rounded-full h-2">
                              <div
                                className="h-2 rounded-full bg-blue-500"
                                style={{ width: `${task.cpuUsage}%` }}
                              />
                            </div>
                            <span className="text-white text-sm">{task.cpuUsage}%</span>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex items-center space-x-2">
                            <div className="w-12 bg-slate-600 rounded-full h-2">
                              <div
                                className="h-2 rounded-full bg-green-500"
                                style={{ width: `${task.memoryUsage}%` }}
                              />
                            </div>
                            <span className="text-white text-sm">{task.memoryUsage}%</span>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <span className="px-2 py-1 bg-knirv-primary/20 text-knirv-primary rounded text-xs">
                            {task.teeType}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default PerformanceObservability;
