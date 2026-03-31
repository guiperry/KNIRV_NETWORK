import React from 'react';

import { AlertTriangle, Zap, ShieldAlert, Eye, X, Clock } from 'lucide-react';
import { NRV } from '../App';

interface SystemNotification {
  id: string;
  type: 'Adversarial Drift' | 'Backprop Pulse';
  message: string;
  severity: 'Low' | 'Medium' | 'High' | 'Critical';
  timestamp: Date;
  metadata?: Record<string, unknown>;
}

interface NRVVisualizationProps {
  nrvs: NRV[];
  onNRVSelect: (nrv: NRV) => void;
  onNRVMapping: (nrv: NRV) => void;
  onNRVClose: (nrv: NRV) => void;
  onAnalyze: () => void;
}

export const NRVVisualization: React.FC<NRVVisualizationProps> = ({
  nrvs,
  onNRVSelect,
  onNRVMapping,
  onNRVClose,
  onAnalyze
}) => {
  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'Critical': return 'bg-red-500/20 border-red-500/50 text-red-400';
      case 'High': return 'bg-orange-500/20 border-orange-500/50 text-orange-400';
      case 'Medium': return 'bg-yellow-500/20 border-yellow-500/50 text-yellow-400';
      case 'Low': return 'bg-green-500/20 border-green-500/50 text-green-400';
      default: return 'bg-gray-500/20 border-gray-500/50 text-gray-400';
    }
  };

  const getNotificationIcon = (type: string) => {
    switch (type) {
      case 'Adversarial Drift': return <Eye className="w-4 h-4" />;
      case 'Backprop Pulse': return <Zap className="w-4 h-4" />;
      default: return <AlertTriangle className="w-4 h-4" />;
    }
  };

  // Transform NRVs into system notifications
  const systemNotifications: SystemNotification[] = nrvs.map(nrv => ({
    id: nrv.id,
    type: nrv.inputType === 'Error' ? 'Adversarial Drift' : 'Backprop Pulse',
    message: nrv.problemDescription,
    severity: nrv.severity,
    timestamp: nrv.temporalContext,
    metadata: {
      source: nrv.sourceID,
      suggestedSolution: nrv.suggestedSolutionType
    }
  }));

  return (
    <div className="absolute top-20 left-4 z-30 space-y-2 max-w-sm" data-testid="system-notifications">
      {systemNotifications.map((notification) => (
        <div
          key={notification.id}
          className={`p-3 rounded-lg border backdrop-blur-sm transition-all duration-300 cursor-pointer hover:scale-105 ${getSeverityColor(notification.severity)}`}
          onClick={() => {
            const nrv = nrvs.find(n => n.id === notification.id);
            if (nrv) onNRVSelect(nrv);
          }}
        >
          <div className="flex items-start justify-between mb-2">
            <div className="flex items-center space-x-2">
              <div className="text-blue-400">
                {getNotificationIcon(notification.type)}
              </div>
              <span className="text-xs font-medium">{notification.type}</span>
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-xs text-gray-400">{notification.severity}</span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  const nrv = nrvs.find(n => n.id === notification.id);
                  if (nrv) onNRVClose(nrv);
                }}
                className="w-5 h-5 flex items-center justify-center rounded bg-gray-700/50 hover:bg-gray-600/50 transition-colors"
              >
                <X className="w-3 h-3 text-gray-400 hover:text-white" />
              </button>
            </div>
          </div>
          
          <p className="text-sm text-white mb-2 line-clamp-2">{notification.message}</p>
          
          <div className="flex items-center justify-between">
            <span className="text-xs text-gray-400">
              {notification.timestamp.toLocaleTimeString()}
            </span>
            <div className="flex space-x-1">
              <span className="text-xs px-2 py-1 rounded bg-blue-500/20 text-blue-400">
                System Event
              </span>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
};
