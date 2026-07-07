import React, { useState } from 'react';
import { useLogStream, ModuleLog } from '@/hooks/use-log-stream';

interface ModuleLogViewerProps {
  module: string;
  maxLines?: number;
  maxHeight?: string;
  className?: string;
}

export function ModuleLogViewer({ 
  module, 
  maxLines = 3, 
  maxHeight = 'h-16',
  className = '' 
}: ModuleLogViewerProps) {
  const { logs, isConnected } = useLogStream({ 
    module, 
    autoConnect: true 
  });
  
  const recentLogs = logs.slice(-maxLines);
  
  const formatTime = (timestamp: string) => {
    try {
      const date = new Date(timestamp);
      return date.toLocaleTimeString('en-US', { 
        hour12: false, 
        hour: '2-digit', 
        minute: '2-digit', 
        second: '2-digit' 
      });
    } catch {
      return '--:--:--';
    }
  };
  
  const getLevelColor = (level: string) => {
    switch (level) {
      case 'error': return 'text-red-400';
      case 'warn': return 'text-amber-400';
      case 'debug': return 'text-gray-500';
      default: return 'text-indigo-400';
    }
  };

  if (!isConnected || recentLogs.length === 0) {
    return (
      <div className={`bg-black/40 rounded-lg p-2 font-mono text-[9px] text-gray-500 ${maxHeight} overflow-hidden`}>
        <div className="line-clamp-1">Waiting for {module} logs...</div>
      </div>
    );
  }

  return (
    <div className={`bg-black/40 rounded-lg p-2 font-mono text-[9px] ${maxHeight} overflow-hidden ${className}`}>
      {recentLogs.map((log, idx) => (
        <div 
          key={idx} 
          className={`line-clamp-1 ${getLevelColor(log.level)}`}
          title={log.message}
        >
          [{formatTime(log.timestamp)}] {log.message}
        </div>
      ))}
    </div>
  );
}

export default ModuleLogViewer;