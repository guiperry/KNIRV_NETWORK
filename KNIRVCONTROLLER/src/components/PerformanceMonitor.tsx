/**
 * Performance Monitor Component
 * Comprehensive performance monitoring dashboard with real-time metrics
 */

import * as React from 'react';
import { useState, useEffect, useCallback } from 'react';
import { performanceOptimizer } from '../utils/PerformanceOptimizer';
import { errorHandler } from '../utils/ErrorHandler';
import { memoryManager } from '../utils/MemoryManager';
import { networkOptimizer } from '../utils/NetworkOptimizer';

interface PerformanceMonitorProps {
  isOpen: boolean;
  onClose: () => void;
}

interface PerformanceData {
  performance: ReturnType<typeof performanceOptimizer.getMetrics>;
  memory: ReturnType<typeof memoryManager.getCurrentMetrics>;
  network: ReturnType<typeof networkOptimizer.getMetrics>;
  errors: ReturnType<typeof errorHandler.getErrorStats>;
  memoryTrend: ReturnType<typeof memoryManager.getUsageTrend>;
  memoryLeaks: ReturnType<typeof memoryManager.detectMemoryLeaks>;
}

const PerformanceMonitor: React.FC<PerformanceMonitorProps> = ({ isOpen, onClose }) => {
  const [activeTab, setActiveTab] = useState<'overview' | 'memory' | 'network' | 'errors' | 'optimization'>('overview');
  const [performanceData, setPerformanceData] = useState<PerformanceData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);

  /**
   * Collect all performance data
   */
  const collectPerformanceData = useCallback(async () => {
    setIsLoading(true);
    try {
      const data: PerformanceData = {
        performance: performanceOptimizer.getMetrics(),
        memory: memoryManager.getCurrentMetrics(),
        network: networkOptimizer.getMetrics(),
        errors: errorHandler.getErrorStats(),
        memoryTrend: memoryManager.getUsageTrend(),
        memoryLeaks: memoryManager.detectMemoryLeaks()
      };
      setPerformanceData(data);
    } catch (error) {
      console.error('Failed to collect performance data:', error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Auto-refresh effect
   */
  useEffect(() => {
    if (!isOpen) return;

    collectPerformanceData();

    if (autoRefresh) {
      const interval = setInterval(collectPerformanceData, 5000); // Every 5 seconds
      return () => clearInterval(interval);
    }
  }, [isOpen, autoRefresh, collectPerformanceData]);

  /**
   * Trigger memory cleanup
   */
  const handleMemoryCleanup = () => {
    memoryManager.triggerCleanup();
    performanceOptimizer.clearCache();
    setTimeout(collectPerformanceData, 1000); // Refresh after cleanup
  };

  /**
   * Clear error history
   */
  const handleClearErrors = () => {
    errorHandler.clearHistory();
    collectPerformanceData();
  };

  /**
   * Generate performance report
   */
  const handleGenerateReport = () => {
    const perfReport = performanceOptimizer.generateReport();
    const memoryReport = memoryManager.generateReport();
    const networkReport = networkOptimizer.generateReport();

    const fullReport = {
      timestamp: new Date().toISOString(),
      performance: perfReport,
      memory: memoryReport,
      network: networkReport,
      errors: errorHandler.getErrorStats()
    };

    const blob = new Blob([JSON.stringify(fullReport, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `knirv-performance-report-${Date.now()}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-gray-900/95 backdrop-blur-md rounded-2xl border border-gray-700/50 w-full max-w-6xl h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-700/50">
          <div>
            <h2 className="text-2xl font-bold text-white">Performance Monitor</h2>
            <p className="text-gray-400 mt-1">Real-time system performance and optimization</p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={() => setAutoRefresh(!autoRefresh)}
              className={`px-3 py-1 rounded-lg text-sm font-medium transition-colors ${
                autoRefresh 
                  ? 'bg-green-500/20 text-green-400 border border-green-500/30' 
                  : 'bg-gray-700/50 text-gray-400 border border-gray-600/50'
              }`}
            >
              Auto Refresh {autoRefresh ? 'ON' : 'OFF'}
            </button>
            <button
              onClick={collectPerformanceData}
              disabled={isLoading}
              className="p-2 rounded-lg bg-blue-500/20 text-blue-400 border border-blue-500/30 hover:bg-blue-500/30 transition-colors disabled:opacity-50"
              title="Refresh Data"
            >
              <svg className={`w-5 h-5 ${isLoading ? 'animate-spin' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </button>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-white text-2xl font-bold transition-colors"
            >
              ×
            </button>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-gray-700/50">
          {([
            { id: 'overview' as const, label: 'Overview', icon: '📊' },
            { id: 'memory' as const, label: 'Memory', icon: '🧠' },
            { id: 'network' as const, label: 'Network', icon: '🌐' },
            { id: 'errors' as const, label: 'Errors', icon: '⚠️' },
            { id: 'optimization' as const, label: 'Optimization', icon: '⚡' }
          ] as const).map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-6 py-3 text-sm font-medium transition-colors border-b-2 ${
                activeTab === tab.id
                  ? 'text-cyan-400 border-cyan-400'
                  : 'text-gray-400 border-transparent hover:text-gray-300'
              }`}
            >
              <span className="mr-2">{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-6">
          {!performanceData ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-center">
                <div className="animate-spin w-8 h-8 border-2 border-cyan-400 border-t-transparent rounded-full mx-auto mb-4"></div>
                <p className="text-gray-400">Loading performance data...</p>
              </div>
            </div>
          ) : (
            <>
              {/* Overview Tab */}
              {activeTab === 'overview' && (
                <div className="space-y-6">
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                    <MetricCard
                      title="Memory Usage"
                      value={`${performanceData.memory?.usagePercentage.toFixed(1) || 0}%`}
                      status={getMemoryStatus(performanceData.memory?.usagePercentage || 0)}
                      icon="🧠"
                    />
                    <MetricCard
                      title="Network Latency"
                      value={`${performanceData.network.averageLatency.toFixed(0)}ms`}
                      status={getLatencyStatus(performanceData.network.averageLatency)}
                      icon="🌐"
                    />
                    <MetricCard
                      title="Cache Hit Rate"
                      value={`${performanceData.performance.cacheHitRate.toFixed(1)}%`}
                      status={getCacheStatus(performanceData.performance.cacheHitRate)}
                      icon="💾"
                    />
                    <MetricCard
                      title="Error Rate"
                      value={`${((performanceData.errors.total > 0 ? performanceData.errors.bySeverity.high / performanceData.errors.total : 0) * 100).toFixed(1)}%`}
                      status={getErrorStatus(performanceData.errors.total)}
                      icon="⚠️"
                    />
                  </div>

                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div className="bg-gray-800/50 rounded-xl p-6 border border-gray-700/50">
                      <h3 className="text-lg font-semibold text-white mb-4">Memory Trend</h3>
                      <div className="space-y-3">
                        <div className="flex justify-between">
                          <span className="text-gray-400">Trend:</span>
                          <span className={`font-medium ${
                            performanceData.memoryTrend.trend === 'increasing' ? 'text-red-400' :
                            performanceData.memoryTrend.trend === 'decreasing' ? 'text-green-400' :
                            'text-yellow-400'
                          }`}>
                            {performanceData.memoryTrend.trend}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-gray-400">Average Change:</span>
                          <span className="text-white">{performanceData.memoryTrend.averageChange.toFixed(2)}%</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-gray-400">Current Usage:</span>
                          <span className="text-white">{performanceData.memoryTrend.currentUsage.toFixed(1)}%</span>
                        </div>
                      </div>
                    </div>

                    <div className="bg-gray-800/50 rounded-xl p-6 border border-gray-700/50">
                      <h3 className="text-lg font-semibold text-white mb-4">Memory Leak Detection</h3>
                      <div className="space-y-3">
                        <div className="flex justify-between">
                          <span className="text-gray-400">Status:</span>
                          <span className={`font-medium ${
                            performanceData.memoryLeaks.hasLeak ? 'text-red-400' : 'text-green-400'
                          }`}>
                            {performanceData.memoryLeaks.hasLeak ? 'Potential Leak' : 'No Leaks'}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-gray-400">Confidence:</span>
                          <span className="text-white">{performanceData.memoryLeaks.confidence}%</span>
                        </div>
                        <div className="text-sm text-gray-400 mt-2">
                          {performanceData.memoryLeaks.details}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* Memory Tab */}
              {activeTab === 'memory' && (
                <div className="space-y-6">
                  <div className="flex justify-between items-center">
                    <h3 className="text-xl font-semibold text-white">Memory Management</h3>
                    <button
                      onClick={handleMemoryCleanup}
                      className="px-4 py-2 bg-red-500/20 text-red-400 border border-red-500/30 rounded-lg hover:bg-red-500/30 transition-colors"
                    >
                      Trigger Cleanup
                    </button>
                  </div>

                  {performanceData.memory && (
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <MetricCard
                        title="Used Heap Size"
                        value={formatBytes(performanceData.memory.usedJSHeapSize)}
                        status="info"
                        icon="📊"
                      />
                      <MetricCard
                        title="Total Heap Size"
                        value={formatBytes(performanceData.memory.totalJSHeapSize)}
                        status="info"
                        icon="📈"
                      />
                      <MetricCard
                        title="Heap Size Limit"
                        value={formatBytes(performanceData.memory.jsHeapSizeLimit)}
                        status="info"
                        icon="🔒"
                      />
                    </div>
                  )}

                  <div className="bg-gray-800/50 rounded-xl p-6 border border-gray-700/50">
                    <h4 className="text-lg font-semibold text-white mb-4">Memory Usage History</h4>
                    <div className="h-64 flex items-center justify-center text-gray-400">
                      Memory usage chart would be rendered here
                    </div>
                  </div>
                </div>
              )}

              {/* Network Tab */}
              {activeTab === 'network' && (
                <div className="space-y-6">
                  <h3 className="text-xl font-semibold text-white">Network Performance</h3>

                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    <MetricCard
                      title="Total Requests"
                      value={performanceData.network.totalRequests.toString()}
                      status="info"
                      icon="📡"
                    />
                    <MetricCard
                      title="Success Rate"
                      value={`${((performanceData.network.successfulRequests / Math.max(performanceData.network.totalRequests, 1)) * 100).toFixed(1)}%`}
                      status={getSuccessRateStatus((performanceData.network.successfulRequests / Math.max(performanceData.network.totalRequests, 1)) * 100)}
                      icon="✅"
                    />
                    <MetricCard
                      title="Cache Hit Rate"
                      value={`${performanceData.network.cacheHitRate.toFixed(1)}%`}
                      status={getCacheStatus(performanceData.network.cacheHitRate)}
                      icon="💾"
                    />
                  </div>

                  <div className="bg-gray-800/50 rounded-xl p-6 border border-gray-700/50">
                    <h4 className="text-lg font-semibold text-white mb-4">Connection Pool Status</h4>
                    <div className="space-y-3">
                      <div className="flex justify-between">
                        <span className="text-gray-400">Active Connections:</span>
                        <span className="text-white">{networkOptimizer.getConnectionPoolStatus().activeConnections}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Max Connections:</span>
                        <span className="text-white">{networkOptimizer.getConnectionPoolStatus().maxConnections}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Queued Requests:</span>
                        <span className="text-white">{networkOptimizer.getConnectionPoolStatus().queuedRequests.length}</span>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* Errors Tab */}
              {activeTab === 'errors' && (
                <div className="space-y-6">
                  <div className="flex justify-between items-center">
                    <h3 className="text-xl font-semibold text-white">Error Monitoring</h3>
                    <button
                      onClick={handleClearErrors}
                      className="px-4 py-2 bg-red-500/20 text-red-400 border border-red-500/30 rounded-lg hover:bg-red-500/30 transition-colors"
                    >
                      Clear History
                    </button>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                    <MetricCard
                      title="Total Errors"
                      value={performanceData.errors.total.toString()}
                      status={getErrorStatus(performanceData.errors.total)}
                      icon="⚠️"
                    />
                    <MetricCard
                      title="Critical Errors"
                      value={performanceData.errors.bySeverity.critical.toString()}
                      status={performanceData.errors.bySeverity.critical > 0 ? 'error' : 'success'}
                      icon="🚨"
                    />
                    <MetricCard
                      title="High Severity"
                      value={performanceData.errors.bySeverity.high.toString()}
                      status={performanceData.errors.bySeverity.high > 0 ? 'warning' : 'success'}
                      icon="⚠️"
                    />
                    <MetricCard
                      title="Network Errors"
                      value={performanceData.errors.byCategory.network.toString()}
                      status={performanceData.errors.byCategory.network > 0 ? 'warning' : 'success'}
                      icon="🌐"
                    />
                  </div>

                  <div className="bg-gray-800/50 rounded-xl p-6 border border-gray-700/50">
                    <h4 className="text-lg font-semibold text-white mb-4">Recent Errors</h4>
                    <div className="space-y-3 max-h-64 overflow-y-auto">
                      {performanceData.errors.recentErrors.length === 0 ? (
                        <p className="text-gray-400 text-center py-4">No recent errors</p>
                      ) : (
                        performanceData.errors.recentErrors.map((error, index) => (
                          <div key={index} className="bg-gray-700/50 rounded-lg p-3 border border-gray-600/50">
                            <div className="flex justify-between items-start">
                              <div>
                                <p className="text-white font-medium">{error.message}</p>
                                <p className="text-gray-400 text-sm">{error.category} • {error.severity}</p>
                              </div>
                              <span className="text-gray-400 text-xs">
                                {error.context.timestamp.toLocaleTimeString()}
                              </span>
                            </div>
                          </div>
                        ))
                      )}
                    </div>
                  </div>
                </div>
              )}

              {/* Optimization Tab */}
              {activeTab === 'optimization' && (
                <div className="space-y-6">
                  <div className="flex justify-between items-center">
                    <h3 className="text-xl font-semibold text-white">Performance Optimization</h3>
                    <button
                      onClick={handleGenerateReport}
                      className="px-4 py-2 bg-blue-500/20 text-blue-400 border border-blue-500/30 rounded-lg hover:bg-blue-500/30 transition-colors"
                    >
                      Generate Report
                    </button>
                  </div>

                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div className="bg-gray-800/50 rounded-xl p-6 border border-gray-700/50">
                      <h4 className="text-lg font-semibold text-white mb-4">Performance Recommendations</h4>
                      <div className="space-y-3">
                        {getPerformanceRecommendations(performanceData).map((recommendation, index) => (
                          <div key={index} className="flex items-start gap-3">
                            <span className="text-yellow-400 mt-1">💡</span>
                            <p className="text-gray-300 text-sm">{recommendation}</p>
                          </div>
                        ))}
                      </div>
                    </div>

                    <div className="bg-gray-800/50 rounded-xl p-6 border border-gray-700/50">
                      <h4 className="text-lg font-semibold text-white mb-4">Optimization Actions</h4>
                      <div className="space-y-3">
                        <button
                          onClick={handleMemoryCleanup}
                          className="w-full p-3 bg-red-500/20 text-red-400 border border-red-500/30 rounded-lg hover:bg-red-500/30 transition-colors text-left"
                        >
                          🧹 Trigger Memory Cleanup
                        </button>
                        <button
                          onClick={() => performanceOptimizer.clearCache()}
                          className="w-full p-3 bg-yellow-500/20 text-yellow-400 border border-yellow-500/30 rounded-lg hover:bg-yellow-500/30 transition-colors text-left"
                        >
                          🗑️ Clear Performance Cache
                        </button>
                        <button
                          onClick={() => networkOptimizer.resetMetrics()}
                          className="w-full p-3 bg-blue-500/20 text-blue-400 border border-blue-500/30 rounded-lg hover:bg-blue-500/30 transition-colors text-left"
                        >
                          🔄 Reset Network Metrics
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};

// Helper Components and Functions
interface MetricCardProps {
  title: string;
  value: string;
  status: 'success' | 'warning' | 'error' | 'info';
  icon: string;
}

const MetricCard: React.FC<MetricCardProps> = ({ title, value, status, icon }) => {
  const statusColors = {
    success: 'bg-green-500/20 border-green-500/30 text-green-400',
    warning: 'bg-yellow-500/20 border-yellow-500/30 text-yellow-400',
    error: 'bg-red-500/20 border-red-500/30 text-red-400',
    info: 'bg-blue-500/20 border-blue-500/30 text-blue-400'
  };

  return (
    <div className={`rounded-xl p-4 border ${statusColors[status]}`}>
      <div className="flex items-center justify-between mb-2">
        <span className="text-2xl">{icon}</span>
        <span className="text-2xl font-bold">{value}</span>
      </div>
      <p className="text-sm opacity-80">{title}</p>
    </div>
  );
};

// Helper Functions
const getMemoryStatus = (usage: number): 'success' | 'warning' | 'error' => {
  if (usage < 70) return 'success';
  if (usage < 85) return 'warning';
  return 'error';
};

const getLatencyStatus = (latency: number): 'success' | 'warning' | 'error' => {
  if (latency < 100) return 'success';
  if (latency < 500) return 'warning';
  return 'error';
};

const getCacheStatus = (hitRate: number): 'success' | 'warning' | 'error' => {
  if (hitRate > 70) return 'success';
  if (hitRate > 40) return 'warning';
  return 'error';
};

const getErrorStatus = (errorCount: number): 'success' | 'warning' | 'error' => {
  if (errorCount === 0) return 'success';
  if (errorCount < 10) return 'warning';
  return 'error';
};

const getSuccessRateStatus = (rate: number): 'success' | 'warning' | 'error' => {
  if (rate > 95) return 'success';
  if (rate > 85) return 'warning';
  return 'error';
};

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const getPerformanceRecommendations = (data: PerformanceData): string[] => {
  const recommendations: string[] = [];

  if (data.memory && data.memory.usagePercentage > 80) {
    recommendations.push('High memory usage detected. Consider implementing memory cleanup strategies.');
  }

  if (data.network.averageLatency > 1000) {
    recommendations.push('High network latency. Consider implementing request caching and optimization.');
  }

  if (data.performance.cacheHitRate < 50) {
    recommendations.push('Low cache hit rate. Review and optimize caching strategy.');
  }

  if (data.errors.total > 10) {
    recommendations.push('High error count. Review error handling and implement better error recovery.');
  }

  if (data.memoryLeaks.hasLeak) {
    recommendations.push(`Potential memory leak detected (${data.memoryLeaks.confidence}% confidence). ${data.memoryLeaks.details}`);
  }

  if (recommendations.length === 0) {
    recommendations.push('System performance is optimal. No immediate optimizations needed.');
  }

  return recommendations;
};

export default PerformanceMonitor;
