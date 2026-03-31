/**
 * Enhanced Loading States Components
 * Provides various loading indicators for different contexts
 */

import React from 'react';
import { Loader2, Wifi, Database, Cpu, Zap, Brain, Network } from 'lucide-react';

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg' | 'xl';
  color?: 'blue' | 'green' | 'purple' | 'orange' | 'red';
  className?: string;
}

export const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({ 
  size = 'md', 
  color = 'blue',
  className = '' 
}) => {
  const sizeClasses = {
    sm: 'w-4 h-4',
    md: 'w-6 h-6',
    lg: 'w-8 h-8',
    xl: 'w-12 h-12'
  };

  const colorClasses = {
    blue: 'text-blue-500',
    green: 'text-green-500',
    purple: 'text-purple-500',
    orange: 'text-orange-500',
    red: 'text-red-500'
  };

  return (
    <Loader2 
      className={`animate-spin ${sizeClasses[size]} ${colorClasses[color]} ${className}`} 
    />
  );
};

interface ProgressBarProps {
  progress: number;
  label?: string;
  showPercentage?: boolean;
  color?: 'blue' | 'green' | 'purple' | 'orange';
  className?: string;
}

export const ProgressBar: React.FC<ProgressBarProps> = ({
  progress,
  label,
  showPercentage = true,
  color = 'blue',
  className = ''
}) => {
  const colorClasses = {
    blue: 'bg-blue-500',
    green: 'bg-green-500',
    purple: 'bg-purple-500',
    orange: 'bg-orange-500'
  };

  return (
    <div className={`w-full ${className}`}>
      {label && (
        <div className="flex justify-between items-center mb-2">
          <span className="text-sm font-medium text-gray-300">{label}</span>
          {showPercentage && (
            <span className="text-sm text-gray-400">{Math.round(progress)}%</span>
          )}
        </div>
      )}
      <div className="w-full bg-gray-700 rounded-full h-2">
        <div 
          className={`h-2 rounded-full transition-all duration-300 ${colorClasses[color]}`}
          style={{ width: `${Math.min(100, Math.max(0, progress))}%` }}
        />
      </div>
    </div>
  );
};

interface SkeletonProps {
  className?: string;
  variant?: 'text' | 'rectangular' | 'circular';
  width?: string | number;
  height?: string | number;
  animation?: 'pulse' | 'wave';
}

export const Skeleton: React.FC<SkeletonProps> = ({
  className = '',
  variant = 'text',
  width,
  height,
  animation = 'pulse'
}) => {
  const baseClasses = 'bg-gray-700 rounded';
  
  const variantClasses = {
    text: 'h-4',
    rectangular: 'h-12',
    circular: 'rounded-full'
  };

  const animationClasses = {
    pulse: 'animate-pulse',
    wave: 'animate-pulse' // Could implement wave animation with CSS
  };

  const style: React.CSSProperties = {};
  if (width) style.width = typeof width === 'number' ? `${width}px` : width;
  if (height) style.height = typeof height === 'number' ? `${height}px` : height;

  return (
    <div 
      className={`${baseClasses} ${variantClasses[variant]} ${animationClasses[animation]} ${className}`}
      style={style}
    />
  );
};

interface LoadingCardProps {
  title?: string;
  description?: string;
  icon?: React.ReactNode;
  progress?: number;
  steps?: string[];
  currentStep?: number;
  className?: string;
}

export const LoadingCard: React.FC<LoadingCardProps> = ({
  title = 'Loading...',
  description,
  icon,
  progress,
  steps,
  currentStep,
  className = ''
}) => {
  return (
    <div className={`bg-gray-800/50 border border-gray-700 rounded-lg p-6 ${className}`}>
      <div className="flex items-center space-x-3 mb-4">
        {icon || <LoadingSpinner size="md" />}
        <div>
          <h3 className="text-lg font-semibold text-white">{title}</h3>
          {description && (
            <p className="text-sm text-gray-400">{description}</p>
          )}
        </div>
      </div>

      {progress !== undefined && (
        <ProgressBar progress={progress} className="mb-4" />
      )}

      {steps && (
        <div className="space-y-2">
          {steps.map((step, index) => (
            <div 
              key={index}
              className={`flex items-center space-x-2 text-sm ${
                currentStep !== undefined && index <= currentStep 
                  ? 'text-green-400' 
                  : index === (currentStep ?? -1) + 1
                  ? 'text-blue-400'
                  : 'text-gray-500'
              }`}
            >
              <div className={`w-2 h-2 rounded-full ${
                currentStep !== undefined && index <= currentStep 
                  ? 'bg-green-400' 
                  : index === (currentStep ?? -1) + 1
                  ? 'bg-blue-400 animate-pulse'
                  : 'bg-gray-600'
              }`} />
              <span>{step}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

interface NetworkLoadingProps {
  status: 'connecting' | 'syncing' | 'processing' | 'complete' | 'error';
  message?: string;
  className?: string;
}

export const NetworkLoading: React.FC<NetworkLoadingProps> = ({
  status,
  message,
  className = ''
}) => {
  const statusConfig = {
    connecting: {
      icon: <Wifi className="w-6 h-6 text-blue-500 animate-pulse" />,
      color: 'blue',
      defaultMessage: 'Connecting to KNIRV Network...'
    },
    syncing: {
      icon: <Database className="w-6 h-6 text-purple-500 animate-pulse" />,
      color: 'purple',
      defaultMessage: 'Syncing data...'
    },
    processing: {
      icon: <Cpu className="w-6 h-6 text-orange-500 animate-pulse" />,
      color: 'orange',
      defaultMessage: 'Processing...'
    },
    complete: {
      icon: <Zap className="w-6 h-6 text-green-500" />,
      color: 'green',
      defaultMessage: 'Complete!'
    },
    error: {
      icon: <Network className="w-6 h-6 text-red-500" />,
      color: 'red',
      defaultMessage: 'Connection failed'
    }
  };

  const config = statusConfig[status];

  return (
    <div className={`flex items-center space-x-3 ${className}`}>
      {config.icon}
      <span className={`text-${config.color}-400 font-medium`}>
        {message || config.defaultMessage}
      </span>
    </div>
  );
};

interface AILoadingProps {
  stage: 'initializing' | 'thinking' | 'processing' | 'generating' | 'complete';
  message?: string;
  progress?: number;
  className?: string;
}

export const AILoading: React.FC<AILoadingProps> = ({
  stage,
  message,
  progress,
  className = ''
}) => {
  const stageConfig = {
    initializing: {
      icon: <Brain className="w-6 h-6 text-blue-500 animate-pulse" />,
      defaultMessage: 'Initializing AI...',
      color: 'blue'
    },
    thinking: {
      icon: <Brain className="w-6 h-6 text-purple-500 animate-pulse" />,
      defaultMessage: 'AI is thinking...',
      color: 'purple'
    },
    processing: {
      icon: <Cpu className="w-6 h-6 text-orange-500 animate-pulse" />,
      defaultMessage: 'Processing request...',
      color: 'orange'
    },
    generating: {
      icon: <Zap className="w-6 h-6 text-green-500 animate-pulse" />,
      defaultMessage: 'Generating response...',
      color: 'green'
    },
    complete: {
      icon: <Zap className="w-6 h-6 text-green-500" />,
      defaultMessage: 'Complete!',
      color: 'green'
    }
  };

  const config = stageConfig[stage];

  return (
    <div className={`space-y-3 ${className}`}>
      <div className="flex items-center space-x-3">
        {config.icon}
        <span className={`text-${config.color}-400 font-medium`}>
          {message || config.defaultMessage}
        </span>
      </div>
      {progress !== undefined && (
        <ProgressBar progress={progress} color={config.color === 'orange' ? 'purple' : config.color as 'blue' | 'green' | 'purple'} />
      )}
    </div>
  );
};

interface FullScreenLoadingProps {
  title?: string;
  description?: string;
  progress?: number;
  steps?: string[];
  currentStep?: number;
  onCancel?: () => void;
}

export const FullScreenLoading: React.FC<FullScreenLoadingProps> = ({
  title = 'Loading KNIRV Controller',
  description = 'Please wait while we initialize the application...',
  progress,
  steps,
  currentStep,
  onCancel
}) => {
  return (
    <div className="fixed inset-0 bg-gray-900 z-50 flex items-center justify-center">
      <div className="max-w-md w-full mx-4">
        <div className="text-center mb-8">
          <div className="w-16 h-16 mx-auto mb-4 bg-blue-500/20 rounded-full flex items-center justify-center">
            <LoadingSpinner size="lg" />
          </div>
          <h2 className="text-2xl font-bold text-white mb-2">{title}</h2>
          <p className="text-gray-400">{description}</p>
        </div>

        {progress !== undefined && (
          <ProgressBar 
            progress={progress} 
            showPercentage={true}
            className="mb-6"
          />
        )}

        {steps && (
          <div className="space-y-3 mb-6">
            {steps.map((step, index) => (
              <div 
                key={index}
                className={`flex items-center space-x-3 text-sm ${
                  currentStep !== undefined && index <= currentStep 
                    ? 'text-green-400' 
                    : index === (currentStep ?? -1) + 1
                    ? 'text-blue-400'
                    : 'text-gray-500'
                }`}
              >
                <div className={`w-3 h-3 rounded-full ${
                  currentStep !== undefined && index <= currentStep 
                    ? 'bg-green-400' 
                    : index === (currentStep ?? -1) + 1
                    ? 'bg-blue-400 animate-pulse'
                    : 'bg-gray-600'
                }`} />
                <span>{step}</span>
              </div>
            ))}
          </div>
        )}

        {onCancel && (
          <div className="text-center">
            <button
              onClick={onCancel}
              className="px-4 py-2 text-gray-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
