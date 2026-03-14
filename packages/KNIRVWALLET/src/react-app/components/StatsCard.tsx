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
      <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/30 to-blue-800/30 rounded-xl blur opacity-20 group-hover:opacity-40 transition duration-300"></div>
      
      <div className="relative glass-panel p-4 glow-hover">
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <p className="text-xs text-slate-500 mb-1 font-mono uppercase tracking-wider">{title}</p>
            <p className="text-2xl font-bold text-white">{value}</p>
            {change && (
              <p className={`text-xs ${trendColors[trend]} mt-1 font-mono`}>
                {change}
              </p>
            )}
          </div>
          <div className="ml-4">
            <div className="w-12 h-12 bg-blue-500/20 rounded-xl flex items-center justify-center border border-blue-500/20">
              <Icon className="w-6 h-6 text-blue-400" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
