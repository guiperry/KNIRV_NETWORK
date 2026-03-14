import React from 'react';
import { Link } from 'react-router-dom';
import { SkillNode } from '../services/api';
import { Clock, Brain, CheckCircle, AlertCircle, TrendingUp } from 'lucide-react';

interface SkillNodeCardProps {
  skill: SkillNode;
}

const SkillNodeCard: React.FC<SkillNodeCardProps> = ({ skill }) => {
  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const getValidationColor = (validation?: SkillNode['validation']) => {
    if (!validation) return 'text-gray-400';
    if (validation.is_validated && validation.validation_score > 0.8) return 'text-green-400';
    if (validation.is_validated && validation.validation_score > 0.6) return 'text-yellow-400';
    return 'text-red-400';
  };

  const getValidationIcon = (validation?: SkillNode['validation']) => {
    if (!validation) return AlertCircle;
    if (validation.is_validated && validation.validation_score > 0.8) return CheckCircle;
    if (validation.is_validated && validation.validation_score > 0.6) return AlertCircle;
    return AlertCircle;
  };

  const ValidationIcon = getValidationIcon(skill.validation);

  return (
    <Link
      to={`/skill/${skill.id}`}
      className="block bg-gray-700/30 hover:bg-gray-700/50 border border-gray-600/50 hover:border-gray-500/50 rounded-lg p-4 transition-all duration-200 hover:scale-[1.02]"
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center">
            <Brain className="w-5 h-5 text-blue-400" />
          </div>
          <div>
            <div className="text-white font-medium">{skill.skill_type}</div>
            <div className="text-gray-400 text-sm flex items-center space-x-1">
              <Clock className="w-3 h-3" />
              <span>{formatTime(skill.timestamp)}</span>
            </div>
          </div>
        </div>
        <div className="text-right">
          <div className={`flex items-center space-x-1 text-sm ${getValidationColor(skill.validation)}`}>
            <ValidationIcon className="w-3 h-3" />
            <span>
              {skill.validation?.is_validated 
                ? `${(skill.validation.validation_score * 100).toFixed(0)}%` 
                : 'Unvalidated'
              }
            </span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
        <div className="space-y-2">
          <div>
            <div className="text-sm text-gray-400 mb-1">Capabilities</div>
            <div className="flex flex-wrap gap-1">
              {skill.capabilities.slice(0, 3).map((capability, index) => (
                <span 
                  key={index}
                  className="text-xs bg-blue-500/20 text-blue-300 px-2 py-1 rounded"
                >
                  {capability}
                </span>
              ))}
              {skill.capabilities.length > 3 && (
                <span className="text-xs text-gray-400">
                  +{skill.capabilities.length - 3} more
                </span>
              )}
            </div>
          </div>
        </div>
        <div className="space-y-2">
          {skill.performance && (
            <div>
              <div className="text-sm text-gray-400 mb-1">Performance</div>
              <div className="flex items-center space-x-2 text-sm">
                <TrendingUp className="w-3 h-3 text-green-400" />
                <span className="text-green-400">
                  {(skill.performance.success_rate * 100).toFixed(1)}% success
                </span>
              </div>
              <div className="text-xs text-gray-400">
                {skill.performance.total_resolutions} resolutions
              </div>
            </div>
          )}
        </div>
      </div>

      {skill.requirements && Object.keys(skill.requirements).length > 0 && (
        <div className="border-t border-gray-700/50 pt-3">
          <div className="text-sm text-gray-400 mb-1">Requirements</div>
          <div className="text-xs text-gray-300 bg-gray-700/30 rounded p-2 max-h-16 overflow-y-auto">
            {Object.entries(skill.requirements).map(([key, value]) => (
              <div key={key} className="flex justify-between">
                <span>{key}:</span>
                <span>{String(value)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </Link>
  );
};

export default SkillNodeCard;
