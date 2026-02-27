'use client';

import React, { useState, useEffect } from 'react';
import { Activity, X, TrendingUp, Cpu, HardDrive, Network, Play, AlertTriangle, CheckCircle, XCircle } from 'lucide-react';
import { 
  LineChart, 
  Line, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  ResponsiveContainer,
  AreaChart,
  Area
} from 'recharts';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useDemoMode } from '@/contexts/demo-mode-context';

interface MonitorPanelProps {
  isOpen: boolean;
  onClose: () => void;
  nodeId?: string;
  isSidebarOpen?: boolean;
  onOpenSolver?: () => void;
  onSelectTask?: (task: DVETask) => void;
}

export interface DVETask {
  id: string;
  name: string;
  type: string;
  status: 'completed' | 'failed' | 'running' | 'pending';
  timestamp: string;
  duration?: number;
  workerId?: string;
  errorDetails?: string;
  trace?: string[];
  documentsTouched?: string[];
  userMetadata?: Record<string, string>;
  policiesEnforced?: string[];
}

interface MetricData {
  time: string;
  cpu: number;
  memory: number;
  network: number;
}

const MonitorPanel: React.FC<MonitorPanelProps> = ({ isOpen, onClose, nodeId, isSidebarOpen, onOpenSolver, onSelectTask }) => {
  const [metrics, setMetrics] = useState<MetricData[]>([]);
  const [tasks, setTasks] = useState<DVETask[]>([]);
  const [selectedTask, setSelectedTask] = useState<string | null>(null);
  const { isDemoMode } = useDemoMode();

  useEffect(() => {
    if (isOpen) {
      loadTasks();
      loadMetrics();
    }
  }, [isOpen, isDemoMode, nodeId]);

  const loadTasks = async () => {
    if (isDemoMode) {
      setTasks(getDemoTasks());
    } else {
      try {
        const response = await fetch(`/api/dve/${nodeId}/tasks`);
        if (response.ok) {
          const data = await response.json();
          setTasks(data.tasks || []);
        } else {
          setTasks([]);
        }
      } catch (error) {
        console.error('Failed to load tasks:', error);
        setTasks([]);
      }
    }
  };

  const loadMetrics = async () => {
    if (isDemoMode) {
      const initialData = Array.from({ length: 20 }, (_, i) => ({
        time: `${10}:${45 + i}`,
        cpu: 40 + Math.random() * 20,
        memory: 60 + Math.random() * 10,
        network: 20 + Math.random() * 40,
      }));
      setMetrics(initialData);

      const interval = setInterval(() => {
        setMetrics(prev => {
          const newData = [...prev.slice(1)];
          const lastTime = prev[prev.length - 1].time;
          const [h, m] = lastTime.split(':').map(Number);
          const nextM = (m + 1) % 60;
          const nextH = nextM === 0 ? (h + 1) % 24 : h;
          
          newData.push({
            time: `${nextH}:${nextM < 10 ? '0' + nextM : nextM}`,
            cpu: 40 + Math.random() * 20,
            memory: 60 + Math.random() * 10,
            network: 20 + Math.random() * 40,
          });
          return newData;
        });
      }, 3000);

      return () => clearInterval(interval);
    } else {
      try {
        const response = await fetch(`/api/dve/${nodeId}/metrics`);
        if (response.ok) {
          const data = await response.json();
          setMetrics(data.metrics || []);
        }
      } catch (error) {
        console.error('Failed to load metrics:', error);
      }
    }
  };

  const getDemoTasks = (): DVETask[] => [
    { id: 'TASK-001', name: 'Data Validation Flow', type: 'validation', status: 'completed', timestamp: '2025-10-21T10:45:00Z', duration: 2340, workerId: 'WORKER-001', trace: ['init', 'validate', 'process', 'complete'], policiesEnforced: ['data_integrity', 'privacy'] },
    { id: 'TASK-002', name: 'Image Processing Pipeline', type: 'computation', status: 'failed', timestamp: '2025-10-21T10:44:00Z', duration: 890, workerId: 'WORKER-006', errorDetails: 'Memory allocation failed', trace: ['init', 'load', 'process', 'ERROR: out_of_memory'], documentsTouched: ['image_001.jpg', 'config.json'], userMetadata: { user_id: 'user_4821', session: 'sess_001' }, policiesEnforced: ['resource_limits'] },
    { id: 'TASK-003', name: 'API Request Handler', type: 'workflow', status: 'completed', timestamp: '2025-10-21T10:43:00Z', duration: 156, workerId: 'WORKER-004', trace: ['init', 'route', 'execute', 'respond'], policiesEnforced: ['rate_limit', 'auth'] },
    { id: 'TASK-004', name: 'Model Inference Task', type: 'inference', status: 'running', timestamp: '2025-10-21T10:42:30Z', workerId: 'WORKER-001', trace: ['init', 'load_model', 'tokenize', 'inference...'] },
    { id: 'TASK-005', name: 'Database Sync', type: 'sync', status: 'completed', timestamp: '2025-10-21T10:41:00Z', duration: 450, workerId: 'WORKER-008', trace: ['init', 'connect', 'sync', 'verify', 'complete'], policiesEnforced: ['data_consistency'] },
    { id: 'TASK-006', name: 'User Authentication', type: 'auth', status: 'failed', timestamp: '2025-10-21T10:40:00Z', duration: 45, workerId: 'WORKER-007', errorDetails: 'Invalid token signature', trace: ['init', 'validate_token', 'ERROR: invalid_signature'], userMetadata: { attempt: '3', ip: '192.168.1.1' }, policiesEnforced: ['authentication'] },
    { id: 'TASK-007', name: 'Cache Invalidation', type: 'maintenance', status: 'completed', timestamp: '2025-10-21T10:39:00Z', duration: 89, workerId: 'WORKER-004', trace: ['init', 'scan', 'invalidate', 'complete'], policiesEnforced: [] },
    { id: 'TASK-008', name: 'Report Generation', type: 'computation', status: 'pending', timestamp: '2025-10-21T10:38:00Z', workerId: 'WORKER-002', trace: ['init', 'gather_data', 'render...'] },
  ];

  const handleTaskClick = (task: DVETask) => {
    setSelectedTask(task.id);
    if (onSelectTask) {
      onSelectTask(task);
    }
    if (onOpenSolver) {
      onOpenSolver();
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed': return <CheckCircle className="w-3 h-3 text-green-400" />;
      case 'failed': return <XCircle className="w-3 h-3 text-red-400" />;
      case 'running': return <Activity className="w-3 h-3 text-blue-400 animate-pulse" />;
      case 'pending': return <Clock className="w-3 h-3 text-slate-400" />;
      default: return null;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'bg-green-500/20 text-green-400 border-green-500/30';
      case 'failed': return 'bg-red-500/20 text-red-400 border-red-500/30';
      case 'running': return 'bg-blue-500/20 text-blue-400 border-blue-500/30';
      case 'pending': return 'bg-slate-500/20 text-slate-400 border-slate-500/30';
      default: return 'bg-slate-500/20 text-slate-400 border-slate-500/30';
    }
  };

  const formatDuration = (ms?: number) => {
    if (!ms) return '-';
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${Math.floor(ms / 60000)}m ${((ms % 60000) / 1000).toFixed(0)}s`;
  };

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  };

  const failedTasks = tasks.filter(t => t.status === 'failed');
  const completedTasks = tasks.filter(t => t.status === 'completed');
  const runningTasks = tasks.filter(t => t.status === 'running');

  if (!isOpen) return null;

  return (
    <div
      className="absolute bottom-0 right-0 z-[90] transition-all duration-500 transform ease-in-out bg-gradient-to-t from-slate-950 via-blue-950/90 to-slate-900 border-t border-blue-600/50 shadow-[0_-10px_40px_rgba(0,0,0,0.5)] overflow-hidden"
      style={{
        height: '35vh',
        left: isSidebarOpen ? '300px' : '0',
      }}
    >
      <div className="h-full flex flex-col p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center space-x-4">
            <h2 className="text-sm font-bold flex items-center text-blue-300">
              <Activity className="w-4 h-4 mr-2 text-green-400 animate-pulse" />
              DVE Task Monitor {nodeId && `[${nodeId}]`}
            </h2>
            <div className="flex space-x-3">
              <div className="flex items-center text-[10px] text-green-400">
                <CheckCircle className="w-2.5 h-2.5 mr-1" /> {completedTasks.length}
              </div>
              <div className="flex items-center text-[10px] text-red-400">
                <XCircle className="w-2.5 h-2.5 mr-1" /> {failedTasks.length}
              </div>
              <div className="flex items-center text-[10px] text-blue-400">
                <Activity className="w-2.5 h-2.5 mr-1 animate-pulse" /> {runningTasks.length}
              </div>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            {!isDemoMode && (
              <Badge variant="outline" className="text-[9px] bg-green-500/10 text-green-400 border-green-500/20">
                LIVE
              </Badge>
            )}
            <button
              onClick={onClose}
              className="text-slate-400 hover:text-white hover:bg-slate-700 p-1 rounded transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        <div className="flex-1 grid grid-cols-1 lg:grid-cols-3 gap-4 overflow-hidden">
          <div className="lg:col-span-1 h-full bg-slate-950/50 rounded-lg border border-blue-600/20 p-2 flex flex-col">
            <div className="flex items-center justify-between mb-2">
              <span className="text-[10px] font-bold text-slate-400 uppercase">System Metrics</span>
              <div className="flex space-x-2">
                <div className="w-2 h-2 rounded-full bg-blue-500" />
                <div className="w-2 h-2 rounded-full bg-purple-500" />
                <div className="w-2 h-2 rounded-full bg-green-500" />
              </div>
            </div>
            <div className="flex-1 min-h-0">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={metrics}>
                  <defs>
                    <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
                  <XAxis 
                    dataKey="time" 
                    stroke="#475569" 
                    fontSize={9} 
                    tickLine={false}
                    axisLine={false}
                  />
                  <YAxis 
                    stroke="#475569" 
                    fontSize={9} 
                    tickLine={false}
                    axisLine={false}
                    domain={[0, 100]}
                  />
                  <Tooltip 
                    contentStyle={{ backgroundColor: '#0f172a', border: '1px solid #1e40af', fontSize: '10px' }}
                    itemStyle={{ padding: '0' }}
                  />
                  <Area type="monotone" dataKey="cpu" stroke="#3b82f6" fillOpacity={1} fill="url(#colorCpu)" isAnimationActive={false} />
                  <Area type="monotone" dataKey="memory" stroke="#a855f7" fillOpacity={0} isAnimationActive={false} />
                  <Area type="monotone" dataKey="network" stroke="#22c55e" fillOpacity={0} isAnimationActive={false} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>

          <div className="lg:col-span-2 overflow-y-auto pr-2 custom-scrollbar">
            <div className="text-[10px] font-bold text-slate-500 uppercase mb-2 px-1">Task History</div>
            <table className="w-full text-[11px] font-mono border-separate border-spacing-y-1">
              <thead className="sticky top-0 bg-slate-900/95 backdrop-blur-sm z-10">
                <tr>
                  <th className="text-left p-1.5 text-blue-400 font-semibold uppercase tracking-wider">Status</th>
                  <th className="text-left p-1.5 text-blue-400 font-semibold uppercase tracking-wider">Task</th>
                  <th className="text-left p-1.5 text-blue-400 font-semibold uppercase tracking-wider">Type</th>
                  <th className="text-left p-1.5 text-blue-400 font-semibold uppercase tracking-wider">Time</th>
                  <th className="text-left p-1.5 text-blue-400 font-semibold uppercase tracking-wider">Duration</th>
                  <th className="text-left p-1.5 text-blue-400 font-semibold uppercase tracking-wider">Worker</th>
                </tr>
              </thead>
              <tbody>
                {tasks.map((task) => (
                  <tr 
                    key={task.id} 
                    className={`bg-slate-800/30 hover:bg-blue-900/20 transition-all rounded-md cursor-pointer group ${selectedTask === task.id ? 'bg-blue-900/30' : ''}`}
                    onClick={() => handleTaskClick(task)}
                  >
                    <td className="p-1.5">
                      {getStatusIcon(task.status)}
                    </td>
                    <td className="p-1.5">
                      <div className="flex items-center space-x-2">
                        <span className="text-blue-300 font-bold">{task.id}</span>
                        <span className="text-slate-400 text-[9px] line-clamp-1 max-w-[150px]">{task.name}</span>
                      </div>
                    </td>
                    <td className="p-1.5 text-slate-400 capitalize">{task.type}</td>
                    <td className="p-1.5 text-slate-500">{formatTime(task.timestamp)}</td>
                    <td className="p-1.5 text-slate-400 font-mono">{formatDuration(task.duration)}</td>
                    <td className="p-1.5 text-slate-500 text-[9px]">{task.workerId || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

const Clock: React.FC<{ className?: string }> = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="10" />
    <path d="M12 6v6l4 2" />
  </svg>
);

export default MonitorPanel;
