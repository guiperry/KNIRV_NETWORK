import { Zap, Star, ArrowRight } from 'lucide-react';

interface BadgeCardProps {
  name: string;
  description: string;
  category: 'automation' | 'analysis' | 'communication' | 'computation';
  complexity: number;
  nrnCost: number;
  isActive: boolean;
}

export default function BadgeCard({ name, description, category, complexity, nrnCost, isActive }: BadgeCardProps) {
  const categoryColors = {
    automation: 'from-blue-500 to-cyan-500',
    analysis: 'from-green-500 to-emerald-500',
    communication: 'from-blue-400 to-blue-600',
    computation: 'from-orange-500 to-red-500'
  };

  const categoryIcons = {
    automation: '⚙️',
    analysis: '📊',
    communication: '💬',
    computation: '🧠'
  };

  return (
    <div className="relative group">
      <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/20 to-blue-800/20 rounded-2xl blur opacity-20 group-hover:opacity-40 transition duration-300"></div>
      
      <div className="relative glass-panel p-4 glow-hover">
        <div className="flex items-start justify-between mb-3">
          <div className="flex items-center space-x-3">
            <div className={`w-10 h-10 bg-gradient-to-br ${categoryColors[category]} rounded-xl flex items-center justify-center text-white font-bold text-lg`}>
              {categoryIcons[category]}
            </div>
            <div className="flex-1">
              <h3 className="font-semibold text-white">{name}</h3>
              <p className="text-xs text-slate-400 capitalize font-mono">{category}</p>
            </div>
          </div>
          
          {isActive && (
            <div className="px-2 py-1 rounded-full bg-green-500/20 border border-green-500/30">
              <span className="text-xs text-green-400 font-medium font-mono">Active</span>
            </div>
          )}
        </div>

        <p className="text-sm text-slate-300 mb-4 line-clamp-2">{description}</p>

        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-1">
              <Star className="w-4 h-4 text-yellow-400" />
              <span className="text-sm text-slate-400 font-mono">{complexity}/10</span>
            </div>
            <div className="flex items-center space-x-1">
              <Zap className="w-4 h-4 text-blue-400" />
              <span className="text-sm text-slate-400 font-mono">{nrnCost} NRN</span>
            </div>
          </div>
        </div>

        <button className="w-full flex items-center justify-center space-x-2 py-2 bg-blue-600/20 hover:bg-blue-600/30 rounded-xl border border-blue-500/30 text-blue-400 hover:text-blue-300 transition-all group/button">
          <span className="text-sm font-medium font-mono">{isActive ? 'Configure' : 'Activate'}</span>
          <ArrowRight className="w-4 h-4 group-hover/button:translate-x-0.5 transition-transform" />
        </button>
      </div>
    </div>
  );
}
