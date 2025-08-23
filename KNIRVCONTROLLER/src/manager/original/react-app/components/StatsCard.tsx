import { LucideIcon } from 'lucide-react';

interface StatsCardProps {
  title: string;
  value: string | number;
  change?: string;
  icon: LucideIcon;
  trend?: 'up' | 'down' | 'neutral';
}

export default function StatsCard({ title, value, change, icon: Icon, trend = 'neutral' }: StatsCardProps) {
  const trendColors = {
    up: 'text-green-400',
    down: 'text-red-400',
    neutral: 'text-slate-400'
  };

  return (
    <div className="relative group">
      {/* Card Glow Effect */}
      <div className="absolute -inset-0.5 bg-gradient-to-r from-purple-600/50 to-cyan-600/50 rounded-xl blur opacity-20 group-hover:opacity-50 transition duration-300"></div>
      
      <div className="relative bg-slate-800/80 backdrop-blur-xl rounded-xl p-4 border border-slate-700/50 hover:border-purple-500/30 transition-all">
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <p className="text-sm text-slate-400 mb-1">{title}</p>
            <p className="text-2xl font-bold text-white">{value}</p>
            {change && (
              <p className={`text-xs ${trendColors[trend]} mt-1`}>
                {change}
              </p>
            )}
          </div>
          <div className="ml-4">
            <div className="w-12 h-12 bg-gradient-to-br from-purple-500/20 to-cyan-500/20 rounded-xl flex items-center justify-center border border-purple-500/20">
              <Icon className="w-6 h-6 text-purple-400" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
