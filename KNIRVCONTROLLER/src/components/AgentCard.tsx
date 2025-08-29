import { Bot} from 'lucide-react';

interface AgentCardProps {
  name: string;
  status: 'active' | 'idle' | 'error';
  tasks: number;
  performance: number;
  lastActive: string;
}

export default function AgentCard({ name, status, tasks, performance, lastActive }: AgentCardProps) {
  const statusConfig = {
    active: { color: 'text-green-400', bg: 'bg-green-500/20', border: 'border-green-500/30' },
    idle: { color: 'text-yellow-400', bg: 'bg-yellow-500/20', border: 'border-yellow-500/30' },
    error: { color: 'text-red-400', bg: 'bg-red-500/20', border: 'border-red-500/30' }
  };

  const config = statusConfig[status];

  return (
    <div className="relative group">
      {/* Card Glow Effect */}
      <div className="absolute -inset-0.5 bg-gradient-to-r from-purple-600 to-cyan-600 rounded-2xl blur opacity-25 group-hover:opacity-75 transition duration-300"></div>
      
      <div className="relative bg-slate-800/90 backdrop-blur-xl rounded-2xl p-4 border border-slate-700/50 hover:border-purple-500/50 transition-all">
        <div className="flex items-start justify-between mb-3">
          <div className="flex items-center space-x-3">
            <div className="relative">
              <div className="w-10 h-10 bg-gradient-to-br from-purple-500 to-cyan-500 rounded-xl flex items-center justify-center">
                <Bot className="w-5 h-5 text-white" />
              </div>
              <div className={`absolute -top-1 -right-1 w-4 h-4 rounded-full ${config.bg} ${config.border} border flex items-center justify-center`}>
                <div className={`w-2.5 h-2.5 rounded-full ${config.color.replace('text-', 'bg-')}`}></div>
              </div>
            </div>
            <div>
              <h3 className="font-semibold text-white">{name}</h3>
              <p className="text-xs text-slate-400 capitalize">{status}</p>
            </div>
          </div>
        </div>

        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-sm text-slate-400">Active Tasks</span>
            <span className="text-sm font-medium text-white">{tasks}</span>
          </div>
          
          <div className="space-y-1">
            <div className="flex justify-between items-center">
              <span className="text-sm text-slate-400">Performance</span>
              <span className="text-sm font-medium text-white">{performance}%</span>
            </div>
            <div className="w-full bg-slate-700 rounded-full h-2">
              <div 
                className="bg-gradient-to-r from-purple-500 to-cyan-500 h-2 rounded-full transition-all duration-300"
                style={{ width: `${performance}%` }}
              ></div>
            </div>
          </div>

          <div className="flex justify-between items-center text-xs">
            <span className="text-slate-500">Last Active</span>
            <span className="text-slate-400">{lastActive}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
