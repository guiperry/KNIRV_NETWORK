import * as React from 'react';
import { MapPin, Eye, X, Clock, CheckCircle, AlertTriangle } from 'lucide-react';
import { NRV } from '../App';

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

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'Identified': return <Eye className="w-4 h-4" />;
      case 'Mapped': return <MapPin className="w-4 h-4" />;
      case 'Assigned': return <Clock className="w-4 h-4" />;
      case 'Resolved': return <CheckCircle className="w-4 h-4" />;
      default: return <AlertTriangle className="w-4 h-4" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Identified': return 'text-blue-400';
      case 'Mapped': return 'text-teal-400';
      case 'Assigned': return 'text-yellow-400';
      case 'Resolved': return 'text-green-400';
      default: return 'text-gray-400';
    }
  };

  return (
    <div className="absolute top-20 left-4 z-30 space-y-2 max-w-sm" data-testid="nrv-visualization">
      {nrvs.map((nrv) => (
        <div
          key={nrv.id}
          className={`p-3 rounded-lg border backdrop-blur-sm transition-all duration-300 cursor-pointer hover:scale-105 ${getSeverityColor(nrv.severity)}`}
          onClick={() => onNRVSelect(nrv)}
        >
          <div className="flex items-start justify-between mb-2">
            <div className="flex items-center space-x-2">
              <div className={`${getStatusColor(nrv.status)}`}>
                {getStatusIcon(nrv.status)}
              </div>
              <span className="text-xs font-medium">{nrv.severity}</span>
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-xs text-gray-400">{nrv.inputType}</span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onNRVClose(nrv);
                }}
                className="w-5 h-5 flex items-center justify-center rounded bg-gray-700/50 hover:bg-gray-600/50 transition-colors"
              >
                <X className="w-3 h-3 text-gray-400 hover:text-white" />
              </button>
            </div>
          </div>
          
          <p className="text-sm text-white mb-2 line-clamp-2">{nrv.problemDescription}</p>
          
          <div className="flex items-center justify-between">
            <span className="text-xs text-gray-400">
              {nrv.temporalContext.toLocaleTimeString()}
            </span>
            <div className="flex space-x-1">
              {nrv.status === 'Identified' && (
                <>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onNRVMapping(nrv);
                    }}
                    className="text-xs px-2 py-1 bg-teal-500/20 text-teal-400 rounded hover:bg-teal-500/30 transition-colors"
                  >
                    Map
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onAnalyze();
                    }}
                    className="text-xs px-2 py-1 bg-blue-500/20 text-blue-400 rounded hover:bg-blue-500/30 transition-colors"
                  >
                    Analyze
                  </button>
                </>
              )}
              <span className={`text-xs px-2 py-1 rounded ${getStatusColor(nrv.status)} bg-current bg-opacity-20`}>
                {nrv.status}
              </span>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
};