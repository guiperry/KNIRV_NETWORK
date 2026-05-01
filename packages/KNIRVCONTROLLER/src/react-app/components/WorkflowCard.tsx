import { Bot, Zap, AlertCircle, CheckCircle } from 'lucide-react';

interface WorkflowCardProps {
  name: string;
  status: 'active' | 'idle' | 'error';
  tasks: number;
  performance: number;
  lastActive: string;
}

export default function WorkflowCard({ name, status, tasks, performance, lastActive }: WorkflowCardProps) {
  const statusConfig = {
    active: { icon: CheckCircle, color: 'text-green-400', bg: 'bg-green-500/20', border: 'border-green-500/30' },
    idle: { icon: Zap, color: 'text-yellow-400', bg: 'bg-yellow-500/20', border: 'border-yellow-500/30' },
    error: { icon: AlertCircle, color: 'text-red-400', bg: 'bg-red-500/20', border: 'border-red-500/30' }
  };

  const config = statusConfig[status];
  const StatusIcon = config.icon;

  return (
    <div className="relative group">
      <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/30 to-blue-800/30 rounded-2xl blur opacity-20 group-hover:opacity-50 transition duration-300"></div>
      
      <div className="relative glass-panel p-4 glow-hover">
        <div className="flex items-start justify-between mb-3">
          <div className="flex items-center space-x-3">
            <div className="relative">
              <div className="w-10 h-10 bg-blue-600 rounded-xl flex items-center justify-center">
                <Bot className="w-5 h-5 text-white" />
              </div>
              <div className={`absolute -top-1 -right-1 w-4 h-4 rounded-full ${config.bg} ${config.border} border flex items-center justify-center`}>
                <StatusIcon className={`w-2.5 h-2.5 ${config.color}`} />
              </div>
            </div>
            <div>
              <h3 className="font-semibold text-white">{name}</h3>
              <p className="text-xs text-slate-400 capitalize font-mono">{status}</p>
            </div>
          </div>
        </div>

        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-sm text-slate-400 font-mono uppercase text-[10px] tracking-wider">Active Tasks</span>
            <span className="text-sm font-medium text-white">{tasks}</span>
          </div>
          
          <div className="space-y-1">
            <div className="flex justify-between items-center">
              <span className="text-sm text-slate-400 font-mono uppercase text-[10px] tracking-wider">Performance</span>
              <span className="text-sm font-medium text-white">{performance}%</span>
            </div>
            <div className="w-full bg-slate-800 rounded-full h-2">
              <div 
                className="bg-gradient-to-r from-blue-500 to-blue-700 h-2 rounded-full transition-all duration-300"
                style={{ width: `${performance}%` }}
              ></div>
            </div>
          </div>

          <div className="flex justify-between items-center text-xs">
            <span className="text-slate-500 font-mono uppercase text-[10px] tracking-wider">Last Active</span>
            <span className="text-slate-400 font-mono">{lastActive}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
