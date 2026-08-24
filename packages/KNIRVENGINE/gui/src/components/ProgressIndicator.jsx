// Progress Indicator Components for Long-Running Operations
import React from 'react';
import { useProgress, useProgressFormatter } from '../hooks/useProgress';

// Individual progress bar component
export const ProgressBar = ({ 
  progress, 
  status, 
  showPercentage = true, 
  height = 'h-2',
  className = '' 
}) => {
  const getProgressColor = (status) => {
    switch (status) {
      case 'running': return 'bg-blue-500';
      case 'completed': return 'bg-green-500';
      case 'failed': return 'bg-red-500';
      case 'cancelled': return 'bg-gray-500';
      default: return 'bg-gray-300';
    }
  };

  return (
    <div className={`w-full ${className}`}>
      <div className={`w-full ${height} bg-gray-200 rounded-full overflow-hidden`}>
        <div
          className={`${height} ${getProgressColor(status)} transition-all duration-300 ease-out`}
          style={{ width: `${Math.max(0, Math.min(100, progress))}%` }}
        />
      </div>
      {showPercentage && (
        <div className="text-xs text-gray-600 mt-1">
          {Math.round(progress)}%
        </div>
      )}
    </div>
  );
};

// Detailed progress card component
export const ProgressCard = ({ operationId, onClose, compact = false }) => {
  const { state, cancel } = useProgress(operationId);
  const { formatDuration, formatTimeRemaining, getStatusColor, getStatusIcon } = useProgressFormatter();

  if (!state) {
    return null;
  }

  const handleCancel = () => {
    if (state.status === 'running' || state.status === 'pending') {
      cancel('Cancelled by user');
    }
  };

  if (compact) {
    return (
      <div className="flex items-center space-x-3 p-2 bg-white border rounded-lg shadow-sm">
        <div className="text-lg">{getStatusIcon(state.status)}</div>
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium text-gray-900 truncate">
            {state.message}
          </div>
          <ProgressBar 
            progress={state.progress} 
            status={state.status} 
            showPercentage={false}
            height="h-1"
          />
        </div>
        <div className="text-xs text-gray-500">
          {Math.round(state.progress)}%
        </div>
        {onClose && (
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            ×
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="bg-white border border-gray-200 rounded-lg shadow-sm p-4">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center space-x-2">
          <span className="text-xl">{getStatusIcon(state.status)}</span>
          <div>
            <h3 className="text-sm font-medium text-gray-900">{state.message}</h3>
            <p className="text-xs text-gray-500 capitalize">{state.type.replace('_', ' ')}</p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          {(state.status === 'running' || state.status === 'pending') && (
            <button
              onClick={handleCancel}
              className="text-xs text-red-600 hover:text-red-800"
            >
              Cancel
            </button>
          )}
          {onClose && (
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600"
            >
              ×
            </button>
          )}
        </div>
      </div>

      <ProgressBar 
        progress={state.progress} 
        status={state.status} 
        showPercentage={true}
        className="mb-3"
      />

      <div className="space-y-2 text-xs text-gray-600">
        <div className="flex justify-between">
          <span>Status:</span>
          <span className={`font-medium ${getStatusColor(state.status)}`}>
            {state.status.charAt(0).toUpperCase() + state.status.slice(1)}
          </span>
        </div>
        
        {state.details && (
          <div className="flex justify-between">
            <span>Details:</span>
            <span className="text-right max-w-48 truncate">{state.details}</span>
          </div>
        )}

        <div className="flex justify-between">
          <span>Duration:</span>
          <span>{formatDuration(new Date().getTime() - state.startTime.getTime())}</span>
        </div>

        {state.status === 'running' && (
          <div className="flex justify-between">
            <span>Est. remaining:</span>
            <span>{formatTimeRemaining(state.estimatedTimeRemaining)}</span>
          </div>
        )}

        {state.endTime && (
          <div className="flex justify-between">
            <span>Completed:</span>
            <span>{state.endTime.toLocaleTimeString()}</span>
          </div>
        )}
      </div>
    </div>
  );
};

// Floating progress notifications
export const ProgressNotifications = ({ maxVisible = 3 }) => {
  const { activeOperations } = useAllProgress();
  const visibleOperations = activeOperations.slice(0, maxVisible);

  if (visibleOperations.length === 0) {
    return null;
  }

  return (
    <div className="fixed bottom-4 right-4 space-y-2 z-50">
      {visibleOperations.map((operation) => (
        <ProgressCard
          key={operation.id}
          operationId={operation.id}
          compact={true}
        />
      ))}
      {activeOperations.length > maxVisible && (
        <div className="text-xs text-gray-500 text-center p-2 bg-white border rounded-lg shadow-sm">
          +{activeOperations.length - maxVisible} more operations
        </div>
      )}
    </div>
  );
};

// Progress summary for dashboard
export const ProgressSummary = () => {
  const { activeOperations, completedOperations, failedOperations } = useAllProgress();

  if (activeOperations.length === 0 && completedOperations.length === 0 && failedOperations.length === 0) {
    return (
      <div className="text-sm text-gray-500">
        No recent operations
      </div>
    );
  }

  return (
    <div className="flex items-center space-x-4 text-sm">
      {activeOperations.length > 0 && (
        <div className="flex items-center space-x-1 text-blue-600">
          <span className="animate-spin">🔄</span>
          <span>{activeOperations.length} running</span>
        </div>
      )}
      {completedOperations.length > 0 && (
        <div className="flex items-center space-x-1 text-green-600">
          <span>✅</span>
          <span>{completedOperations.length} completed</span>
        </div>
      )}
      {failedOperations.length > 0 && (
        <div className="flex items-center space-x-1 text-red-600">
          <span>❌</span>
          <span>{failedOperations.length} failed</span>
        </div>
      )}
    </div>
  );
};

// Inline progress indicator for buttons
export const ButtonProgress = ({ isLoading, progress, children, ...props }) => {
  if (!isLoading) {
    return (
      <button {...props}>
        {children}
      </button>
    );
  }

  return (
    <button {...props} disabled className={`${props.className} relative overflow-hidden`}>
      <div className="flex items-center justify-center space-x-2">
        <div className="animate-spin w-4 h-4 border-2 border-white border-t-transparent rounded-full"></div>
        <span>{children}</span>
        {progress !== undefined && (
          <span className="text-xs">({Math.round(progress)}%)</span>
        )}
      </div>
      {progress !== undefined && (
        <div 
          className="absolute bottom-0 left-0 h-1 bg-white bg-opacity-30 transition-all duration-300"
          style={{ width: `${progress}%` }}
        />
      )}
    </button>
  );
};
