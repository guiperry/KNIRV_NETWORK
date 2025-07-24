import React from 'react';
import { DivideIcon as LucideIcon } from 'lucide-react';

interface StatsCardProps {
  title: string;
  value: string;
  icon: LucideIcon;
  trend?: number;
  color?: 'blue' | 'green' | 'purple' | 'orange' | 'red';
}

const StatsCard: React.FC<StatsCardProps> = ({ 
  title, 
  value, 
  icon: Icon, 
  trend, 
  color = 'blue' 
}) => {
  const colorClasses = {
    blue: 'from-blue-500/10 to-blue-600/10 border-blue-500/20 text-blue-400',
    green: 'from-green-500/10 to-green-600/10 border-green-500/20 text-green-400',
    purple: 'from-purple-500/10 to-purple-600/10 border-purple-500/20 text-purple-400',
    orange: 'from-orange-500/10 to-orange-600/10 border-orange-500/20 text-orange-400',
    red: 'from-red-500/10 to-red-600/10 border-red-500/20 text-red-400',
  };

  const trendColor = trend && trend > 0 ? 'text-green-400' : 'text-red-400';

  return (
    <div className={`bg-gradient-to-br ${colorClasses[color]} border rounded-xl p-6 backdrop-blur-xl transition-all duration-300 hover:scale-105`}>
      <div className="flex items-center justify-between mb-4">
        <Icon className={`w-8 h-8 ${colorClasses[color].split(' ')[2]}`} />
        {trend !== undefined && (
          <div className={`text-sm font-medium ${trendColor}`}>
            {trend > 0 ? '+' : ''}{trend.toFixed(1)}%
          </div>
        )}
      </div>
      <div className="space-y-1">
        <div className="text-2xl font-bold text-white">{value}</div>
        <div className="text-sm text-gray-400">{title}</div>
      </div>
    </div>
  );
};

export default StatsCard;