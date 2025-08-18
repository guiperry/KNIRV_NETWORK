import React, { useState, useEffect } from 'react';
import {
  X,
  Bot,
  Zap,
  Target,
  Activity,
  Clock,
  BarChart3,
  ArrowUpRight,
  ArrowDownRight,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Calendar,
  User,
  Play,
  Square,
  Settings,
  ExternalLink,
  Download,
  Trash2
} from 'lucide-react';
import { fetchAgentActivity, fetchAgentPerformance, deleteAgent } from '../../utils/api';
import { handleApiError } from '../../utils/errorHandler';

const AgentDetailModal = ({ isOpen, onClose, agent, onDeploy, onStop, onConfigure, onDelete }) => {
  const [activeTab, setActiveTab] = useState('overview');
  const [activityData, setActivityData] = useState([]);
  const [performanceData, setPerformanceData] = useState({
    successRate: [],
    responseTime: [],
    usage: []
  });
  const [isLoading, setIsLoading] = useState(true);
  const [isDeleting, setIsDeleting] = useState(false);

  // Fetch agent details when modal opens
  useEffect(() => {
    if (isOpen && agent) {
      setIsLoading(true);

      // Fetch real data from API
      const fetchData = async () => {
        try {
          // Fetch activity data
          const activityData = await fetchAgentActivity(agent.id);
          setActivityData(activityData);

          // Fetch performance data
          const performanceData = await fetchAgentPerformance(agent.id);
          setPerformanceData(performanceData);

        } catch (error) {
          console.error('Failed to fetch agent data:', error);
          handleApiError(error, {
            operation: 'fetch_agent_data',
            component: 'AgentDetailModal',
            agentId: agent.id,
            agentName: agent.name,
            timestamp: new Date().toISOString(),
            context: 'Loading agent activity and performance data'
          });
          // Keep existing mock data as fallback - the API functions already handle this
        } finally {
          setIsLoading(false);
        }
      };

      fetchData();
    }
  }, [isOpen, agent]);

  // Reset state when modal closes
  useEffect(() => {
    if (!isOpen) {
      setActiveTab('overview');
      setActivityData([]);
      setPerformanceData({
        successRate: [],
        responseTime: [],
        usage: []
      });
    }
  }, [isOpen]);

  const handleDeleteAgent = async () => {
    if (!agent) return;

    const confirmDelete = window.confirm(
      `Are you sure you want to delete "${agent.name}"? This action cannot be undone.`
    );

    if (!confirmDelete) return;

    setIsDeleting(true);
    try {
      await deleteAgent(agent.id);

      // Notify parent component of successful deletion
      if (onDelete) {
        onDelete(agent);
      }

      // Close the modal
      onClose();
    } catch (error) {
      console.error('Error deleting agent:', error);
      handleApiError(error, {
        operation: 'delete_agent',
        component: 'AgentDetailModal',
        agentId: agent.id,
        agentName: agent.name,
        timestamp: new Date().toISOString(),
        context: 'User attempted to delete an agent'
      });
      alert(`Failed to delete agent: ${error.message}`);
    } finally {
      setIsDeleting(false);
    }
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'running':
        return <Activity className="w-5 h-5 text-blue-400 animate-pulse" />;
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-400" />;
      case 'warning':
        return <AlertTriangle className="w-5 h-5 text-yellow-400" />;
      default:
        return <Clock className="w-5 h-5 text-slate-400" />;
    }
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed':
        return 'text-green-400 bg-green-500/20 border-green-500/30';
      case 'running':
        return 'text-blue-400 bg-blue-500/20 border-blue-500/30';
      case 'failed':
        return 'text-red-400 bg-red-500/20 border-red-500/30';
      case 'warning':
        return 'text-yellow-400 bg-yellow-500/20 border-yellow-500/30';
      default:
        return 'text-slate-400 bg-slate-500/20 border-slate-500/30';
    }
  };

  const getActivityTypeIcon = (type) => {
    switch (type) {
      case 'deployment':
        return <Target className="w-4 h-4" />;
      case 'inference':
        return <Zap className="w-4 h-4" />;
      case 'configuration':
        return <Settings className="w-4 h-4" />;
      default:
        return <Activity className="w-4 h-4" />;
    }
  };

  // Calculate performance metrics
  const calculateMetrics = () => {
    // Default values to prevent undefined errors
    const defaultMetrics = {
      successRate: 0,
      responseTime: 0,
      usage: 0,
      successRateTrend: 0,
      responseTimeTrend: 0,
      usageTrend: 0
    };

    if (!agent) return defaultMetrics;

    // Calculate success rate trend
    const successRateTrend = performanceData.successRate.length >= 2
      ? performanceData.successRate[performanceData.successRate.length - 1].rate -
        performanceData.successRate[performanceData.successRate.length - 2].rate
      : 0;

    // Calculate response time trend (negative is better)
    const responseTimeTrend = performanceData.responseTime.length >= 2
      ? performanceData.responseTime[performanceData.responseTime.length - 2].time -
        performanceData.responseTime[performanceData.responseTime.length - 1].time
      : 0;

    // Calculate usage trend
    const usageTrend = performanceData.usage.length >= 2
      ? performanceData.usage[performanceData.usage.length - 1].count -
        performanceData.usage[performanceData.usage.length - 2].count
      : 0;

    return {
      successRate: performanceData.successRate.length > 0
        ? performanceData.successRate[performanceData.successRate.length - 1].rate || 0
        : agent.successRate || 0,
      responseTime: performanceData.responseTime.length > 0
        ? performanceData.responseTime[performanceData.responseTime.length - 1].time || 0
        : 0,
      usage: performanceData.usage.length > 0
        ? performanceData.usage[performanceData.usage.length - 1].count || 0
        : agent.totalInferences || 0,
      successRateTrend: successRateTrend || 0,
      responseTimeTrend: responseTimeTrend || 0,
      usageTrend: usageTrend || 0
    };
  };

  const metrics = calculateMetrics();

  if (!isOpen || !agent) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
        onClick={onClose}
      ></div>
      
      {/* Modal */}
      <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-5xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <div className="flex items-center space-x-4">
            <img
              src={agent.image}
              alt={agent.name}
              className="w-12 h-12 rounded-lg object-contain p-1"
            />
            <div>
              <h2 className="text-xl font-bold text-white">{agent.name}</h2>
              <div className="flex items-center space-x-2">
                <span className="text-slate-400">{agent.collection}</span>
                <span className="w-1.5 h-1.5 rounded-full bg-slate-500"></span>
                <span className={`px-2 py-0.5 rounded-full text-xs font-medium capitalize ${
                  agent.status === 'active' ? 'bg-green-500/20 text-green-400' :
                  agent.status === 'deployed' ? 'bg-blue-500/20 text-blue-400' :
                  agent.status === 'idle' ? 'bg-slate-500/20 text-slate-400' :
                  'bg-yellow-500/20 text-yellow-400'
                }`}>
                  {agent.status}
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            {agent.status === 'idle' ? (
              <button 
                onClick={() => onDeploy(agent)}
                className="px-3 py-1.5 bg-gradient-to-r from-green-500/20 to-emerald-500/20 hover:from-green-500/30 hover:to-emerald-500/30 border border-green-500/30 text-white rounded-lg transition-all duration-200 flex items-center space-x-1"
              >
                <Play className="w-4 h-4" />
                <span>Deploy</span>
              </button>
            ) : (
              <button 
                onClick={() => onStop(agent)}
                className="px-3 py-1.5 bg-gradient-to-r from-red-500/20 to-orange-500/20 hover:from-red-500/30 hover:to-orange-500/30 border border-red-500/30 text-white rounded-lg transition-all duration-200 flex items-center space-x-1"
              >
                <Square className="w-4 h-4" />
                <span>Stop</span>
              </button>
            )}
            <button
              onClick={handleDeleteAgent}
              disabled={isDeleting}
              className="px-3 py-1.5 bg-red-500/20 border border-red-500/30 text-red-400 hover:text-red-300 hover:border-red-500/50 rounded-lg transition-all duration-200 flex items-center space-x-1 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Trash2 className="w-4 h-4" />
              <span>{isDeleting ? 'Deleting...' : 'Delete'}</span>
            </button>
            <button
              onClick={onClose}
              aria-label="Close"
              className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>
        
        {/* Tabs */}
        <div className="flex border-b border-slate-700">
          <button
            onClick={() => setActiveTab('overview')}
            className={`flex items-center space-x-2 px-4 py-3 transition-all duration-200 ${
              activeTab === 'overview'
                ? 'bg-slate-700/50 text-white border-b-2 border-purple-500'
                : 'text-slate-400 hover:text-white hover:bg-slate-700/30'
            }`}
          >
            <Bot className="w-4 h-4" />
            <span>Overview</span>
          </button>
          <button
            onClick={() => setActiveTab('activity')}
            className={`flex items-center space-x-2 px-4 py-3 transition-all duration-200 ${
              activeTab === 'activity'
                ? 'bg-slate-700/50 text-white border-b-2 border-purple-500'
                : 'text-slate-400 hover:text-white hover:bg-slate-700/30'
            }`}
          >
            <Activity className="w-4 h-4" />
            <span>Activity</span>
          </button>
          <button
            onClick={() => setActiveTab('performance')}
            className={`flex items-center space-x-2 px-4 py-3 transition-all duration-200 ${
              activeTab === 'performance'
                ? 'bg-slate-700/50 text-white border-b-2 border-purple-500'
                : 'text-slate-400 hover:text-white hover:bg-slate-700/30'
            }`}
          >
            <BarChart3 className="w-4 h-4" />
            <span>Performance</span>
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-220px)]">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="w-8 h-8 border-4 border-purple-500/30 border-t-purple-500 rounded-full animate-spin"></div>
              <span className="ml-3 text-slate-400">Loading agent details...</span>
            </div>
          ) : (
            <>
              {/* Overview Tab */}
              {activeTab === 'overview' && (
                <div className="space-y-6">
                  {/* Agent Details */}
                  <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                    {/* Agent Information */}
                    <div className="lg:col-span-2 bg-slate-700/30 rounded-xl p-6 border border-slate-600/30">
                      <h3 className="text-lg font-semibold text-white mb-4">Agent Information</h3>
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <p className="text-slate-400 text-sm">Status</p>
                          <p className="text-white font-medium capitalize">{agent.status}</p>
                        </div>
                        <div>
                          <p className="text-slate-400 text-sm">Collection</p>
                          <p className="text-white font-medium">{agent.collection}</p>
                        </div>
                        <div>
                          <p className="text-slate-400 text-sm">Total Inferences</p>
                          <p className="text-white font-medium">{agent.totalInferences?.toLocaleString() || '0'}</p>
                        </div>
                        <div>
                          <p className="text-slate-400 text-sm">Success Rate</p>
                          <p className="text-white font-medium">{agent.successRate || '0'}%</p>
                        </div>
                        <div>
                          <p className="text-slate-400 text-sm">Last Activity</p>
                          <p className="text-white font-medium">{agent.lastActivity || 'Never'}</p>
                        </div>
                        <div>
                          <p className="text-slate-400 text-sm">Created</p>
                          <p className="text-white font-medium">{agent.createdAt ? formatDate(agent.createdAt) : 'Unknown'}</p>
                        </div>
                        <div>
                          <p className="text-slate-400 text-sm">Owner</p>
                          <p className="text-white font-medium">{agent.owner || 'Unknown'}</p>
                        </div>
                        <div>
                          <p className="text-slate-400 text-sm">Token ID</p>
                          <p className="text-white font-medium">{agent.tokenID || 'N/A'}</p>
                        </div>
                      </div>
                    </div>
                    
                    {/* Current Deployment */}
                    <div className="bg-slate-700/30 rounded-xl p-6 border border-slate-600/30">
                      <h3 className="text-lg font-semibold text-white mb-4">Current Deployment</h3>
                      {agent.status === 'active' || agent.status === 'deployed' ? (
                        <div className="space-y-4">
                          <div>
                            <p className="text-slate-400 text-sm">Target System</p>
                            <p className="text-white font-medium">{agent.currentTarget || 'None'}</p>
                          </div>
                          <div>
                            <p className="text-slate-400 text-sm">Active Capability</p>
                            <p className="text-white font-medium">{agent.capability || 'None'}</p>
                          </div>
                          <div>
                            <p className="text-slate-400 text-sm">Deployed Since</p>
                            <p className="text-white font-medium">{agent.deployedSince || 'Unknown'}</p>
                          </div>
                          <div className="pt-2">
                            <button 
                              onClick={() => onStop(agent)}
                              className="w-full bg-gradient-to-r from-red-500/20 to-orange-500/20 hover:from-red-500/30 hover:to-orange-500/30 border border-red-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2"
                            >
                              <Square className="w-4 h-4" />
                              <span>Stop Agent</span>
                            </button>
                          </div>
                        </div>
                      ) : (
                        <div className="space-y-4">
                          <div className="text-center py-4">
                            <p className="text-slate-400 mb-2">Agent is not currently deployed</p>
                            <button 
                              onClick={() => onDeploy(agent)}
                              className="bg-gradient-to-r from-green-500/20 to-emerald-500/20 hover:from-green-500/30 hover:to-emerald-500/30 border border-green-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2 mx-auto"
                            >
                              <Play className="w-4 h-4" />
                              <span>Deploy Agent</span>
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                  
                  {/* Capabilities and Target Types */}
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {/* Capabilities */}
                    <div className="bg-slate-700/30 rounded-xl p-6 border border-slate-600/30">
                      <h3 className="text-lg font-semibold text-white mb-4">Capabilities</h3>
                      <div className="space-y-2">
                        {agent.capabilities && agent.capabilities.length > 0 ? (
                          agent.capabilities.map((capability, index) => (
                            <div 
                              key={index}
                              className="p-3 bg-slate-800/50 rounded-lg border border-slate-700/50 flex items-center space-x-3"
                            >
                              <div className="p-2 bg-purple-500/20 rounded-lg border border-purple-500/30">
                                <Zap className="w-4 h-4 text-purple-400" />
                              </div>
                              <span className="text-white">{capability}</span>
                            </div>
                          ))
                        ) : (
                          <p className="text-slate-400 text-center py-2">No capabilities configured</p>
                        )}
                      </div>
                    </div>
                    
                    {/* Target Types */}
                    <div className="bg-slate-700/30 rounded-xl p-6 border border-slate-600/30">
                      <h3 className="text-lg font-semibold text-white mb-4">Target Types</h3>
                      <div className="space-y-2">
                        {agent.targetTypes && agent.targetTypes.length > 0 ? (
                          agent.targetTypes.map((targetType, index) => (
                            <div 
                              key={index}
                              className="p-3 bg-slate-800/50 rounded-lg border border-slate-700/50 flex items-center space-x-3"
                            >
                              <div className="p-2 bg-blue-500/20 rounded-lg border border-blue-500/30">
                                <Target className="w-4 h-4 text-blue-400" />
                              </div>
                              <span className="text-white">{targetType}</span>
                            </div>
                          ))
                        ) : (
                          <p className="text-slate-400 text-center py-2">No target types configured</p>
                        )}
                      </div>
                    </div>
                  </div>
                  
                  {/* Performance Metrics */}
                  <div className="bg-slate-700/30 rounded-xl p-6 border border-slate-600/30">
                    <h3 className="text-lg font-semibold text-white mb-4">Performance Metrics</h3>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                      {/* Success Rate */}
                      <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/50">
                        <div className="flex items-center justify-between mb-2">
                          <p className="text-slate-400 text-sm">Success Rate</p>
                          {metrics.successRateTrend > 0 ? (
                            <ArrowUpRight className="w-4 h-4 text-green-400" />
                          ) : metrics.successRateTrend < 0 ? (
                            <ArrowDownRight className="w-4 h-4 text-red-400" />
                          ) : (
                            <div className="w-4 h-4" />
                          )}
                        </div>
                        <p className="text-2xl font-bold text-white mb-1">{(metrics.successRate || 0).toFixed(1)}%</p>
                        <div className="flex items-center">
                          <span className={`text-sm font-medium ${metrics.successRateTrend > 0 ? 'text-green-400' : metrics.successRateTrend < 0 ? 'text-red-400' : 'text-slate-400'}`}>
                            {metrics.successRateTrend > 0 ? '+' : ''}{(metrics.successRateTrend || 0).toFixed(1)}%
                          </span>
                          <span className="text-slate-500 text-xs ml-1">vs previous</span>
                        </div>
                      </div>
                      
                      {/* Response Time */}
                      <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/50">
                        <div className="flex items-center justify-between mb-2">
                          <p className="text-slate-400 text-sm">Avg Response Time</p>
                          {metrics.responseTimeTrend > 0 ? (
                            <ArrowUpRight className="w-4 h-4 text-green-400" />
                          ) : metrics.responseTimeTrend < 0 ? (
                            <ArrowDownRight className="w-4 h-4 text-red-400" />
                          ) : (
                            <div className="w-4 h-4" />
                          )}
                        </div>
                        <p className="text-2xl font-bold text-white mb-1">{(metrics.responseTime || 0).toFixed(1)}s</p>
                        <div className="flex items-center">
                          <span className={`text-sm font-medium ${metrics.responseTimeTrend > 0 ? 'text-green-400' : metrics.responseTimeTrend < 0 ? 'text-red-400' : 'text-slate-400'}`}>
                            {metrics.responseTimeTrend > 0 ? '+' : ''}{(metrics.responseTimeTrend || 0).toFixed(1)}s
                          </span>
                          <span className="text-slate-500 text-xs ml-1">vs previous</span>
                        </div>
                      </div>
                      
                      {/* Usage */}
                      <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/50">
                        <div className="flex items-center justify-between mb-2">
                          <p className="text-slate-400 text-sm">Daily Inferences</p>
                          {metrics.usageTrend > 0 ? (
                            <ArrowUpRight className="w-4 h-4 text-green-400" />
                          ) : metrics.usageTrend < 0 ? (
                            <ArrowDownRight className="w-4 h-4 text-red-400" />
                          ) : (
                            <div className="w-4 h-4" />
                          )}
                        </div>
                        <p className="text-2xl font-bold text-white mb-1">{metrics.usage}</p>
                        <div className="flex items-center">
                          <span className={`text-sm font-medium ${metrics.usageTrend > 0 ? 'text-green-400' : metrics.usageTrend < 0 ? 'text-red-400' : 'text-slate-400'}`}>
                            {metrics.usageTrend > 0 ? '+' : ''}{metrics.usageTrend}
                          </span>
                          <span className="text-slate-500 text-xs ml-1">vs previous</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}
              
              {/* Activity Tab */}
              {activeTab === 'activity' && (
                <div className="space-y-6">
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-lg font-semibold text-white">Recent Activity</h3>
                    <div className="flex items-center space-x-2">
                      <button className="text-purple-400 hover:text-purple-300 text-sm font-medium flex items-center space-x-1">
                        <Download className="w-4 h-4" />
                        <span>Export</span>
                      </button>
                      <button className="text-purple-400 hover:text-purple-300 text-sm font-medium flex items-center space-x-1">
                        <ExternalLink className="w-4 h-4" />
                        <span>View All</span>
                      </button>
                    </div>
                  </div>
                  
                  <div className="bg-slate-700/30 rounded-xl border border-slate-600/30 overflow-hidden">
                    <div className="overflow-x-auto">
                      <table className="w-full">
                        <thead className="bg-slate-800/50 border-b border-slate-700/50">
                          <tr>
                            <th className="text-left py-3 px-4 text-slate-300 font-medium">Type</th>
                            <th className="text-left py-3 px-4 text-slate-300 font-medium">Status</th>
                            <th className="text-left py-3 px-4 text-slate-300 font-medium">Target</th>
                            <th className="text-left py-3 px-4 text-slate-300 font-medium">Capability</th>
                            <th className="text-left py-3 px-4 text-slate-300 font-medium">Time</th>
                            <th className="text-left py-3 px-4 text-slate-300 font-medium">Duration</th>
                            <th className="text-left py-3 px-4 text-slate-300 font-medium">Result</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-700/50">
                          {activityData.map((activity) => (
                            <tr key={activity.id} className="hover:bg-slate-700/20 transition-colors duration-200">
                              <td className="py-3 px-4">
                                <div className="flex items-center space-x-2">
                                  <div className={`p-1 rounded-lg ${
                                    activity.type === 'deployment' ? 'bg-blue-500/20 text-blue-400' :
                                    activity.type === 'inference' ? 'bg-purple-500/20 text-purple-400' :
                                    'bg-slate-500/20 text-slate-400'
                                  }`}>
                                    {getActivityTypeIcon(activity.type)}
                                  </div>
                                  <span className="text-white capitalize">{activity.type}</span>
                                </div>
                              </td>
                              <td className="py-3 px-4">
                                <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(activity.status)}`}>
                                  {activity.status}
                                </span>
                              </td>
                              <td className="py-3 px-4 text-white">{activity.target}</td>
                              <td className="py-3 px-4 text-white">{activity.capability}</td>
                              <td className="py-3 px-4 text-slate-300">{formatDate(activity.timestamp)}</td>
                              <td className="py-3 px-4 text-slate-300">{activity.duration}</td>
                              <td className="py-3 px-4">
                                <div className="max-w-xs truncate text-slate-300" title={activity.result}>
                                  {activity.result}
                                </div>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                    
                    {activityData.length === 0 && (
                      <div className="text-center py-8 text-slate-400">
                        No activity data available for this agent.
                      </div>
                    )}
                  </div>
                  
                  <div className="flex items-center justify-between">
                    <div className="text-sm text-slate-400">
                      Showing {activityData.length} of {activityData.length} activities
                    </div>
                    <div className="flex items-center space-x-2">
                      <button className="px-3 py-1 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed">
                        Previous
                      </button>
                      <button className="px-3 py-1 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed">
                        Next
                      </button>
                    </div>
                  </div>
                </div>
              )}
              
              {/* Performance Tab */}
              {activeTab === 'performance' && (
                <div className="space-y-6">
                  {/* Performance Metrics */}
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    {/* Success Rate */}
                    <div className="bg-slate-700/30 rounded-xl p-6 border border-slate-600/30">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-semibold text-white">Success Rate</h3>
                        <div className={`flex items-center space-x-1 ${metrics.successRateTrend > 0 ? 'text-green-400' : metrics.successRateTrend < 0 ? 'text-red-400' : 'text-slate-400'}`}>
                          {metrics.successRateTrend > 0 ? (
                            <ArrowUpRight className="w-4 h-4" />
                          ) : metrics.successRateTrend < 0 ? (
                            <ArrowDownRight className="w-4 h-4" />
                          ) : null}
                          <span>{metrics.successRateTrend > 0 ? '+' : ''}{(metrics.successRateTrend || 0).toFixed(1)}%</span>
                        </div>
                      </div>
                      
                      <div className="h-40 flex items-end space-x-2">
                        {performanceData.successRate.map((data, index) => (
                          <div key={index} className="flex-1 flex flex-col items-center">
                            <div 
                              className="w-full bg-gradient-to-t from-green-500/50 to-green-500/10 rounded-t-sm"
                              style={{ height: `${data.rate}%` }}
                            ></div>
                            <div className="text-xs text-slate-500 mt-1 rotate-45 origin-left translate-y-2">
                              {data.date.split('-')[2]}
                            </div>
                          </div>
                        ))}
                      </div>
                      
                      <div className="mt-6 text-center">
                        <div className="text-2xl font-bold text-white">{(metrics.successRate || 0).toFixed(1)}%</div>
                        <div className="text-sm text-slate-400">Current success rate</div>
                      </div>
                    </div>
                    
                    {/* Response Time */}
                    <div className="bg-slate-700/30 rounded-xl p-6 border border-slate-600/30">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-semibold text-white">Response Time</h3>
                        <div className={`flex items-center space-x-1 ${metrics.responseTimeTrend > 0 ? 'text-green-400' : metrics.responseTimeTrend < 0 ? 'text-red-400' : 'text-slate-400'}`}>
                          {metrics.responseTimeTrend > 0 ? (
                            <ArrowUpRight className="w-4 h-4" />
                          ) : metrics.responseTimeTrend < 0 ? (
                            <ArrowDownRight className="w-4 h-4" />
                          ) : null}
                          <span>{metrics.responseTimeTrend > 0 ? '+' : ''}{(metrics.responseTimeTrend || 0).toFixed(1)}s</span>
                        </div>
                      </div>
                      
                      <div className="h-40 flex items-end space-x-2">
                        {performanceData.responseTime.map((data, index) => (
                          <div key={index} className="flex-1 flex flex-col items-center">
                            <div 
                              className="w-full bg-gradient-to-t from-blue-500/50 to-blue-500/10 rounded-t-sm"
                              style={{ height: `${(data.time / 6) * 100}%` }}
                            ></div>
                            <div className="text-xs text-slate-500 mt-1 rotate-45 origin-left translate-y-2">
                              {data.date.split('-')[2]}
                            </div>
                          </div>
                        ))}
                      </div>
                      
                      <div className="mt-6 text-center">
                        <div className="text-2xl font-bold text-white">{(metrics.responseTime || 0).toFixed(1)}s</div>
                        <div className="text-sm text-slate-400">Average response time</div>
                      </div>
                    </div>
                    
                    {/* Usage */}
                    <div className="bg-slate-700/30 rounded-xl p-6 border border-slate-600/30">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-semibold text-white">Daily Usage</h3>
                        <div className={`flex items-center space-x-1 ${metrics.usageTrend > 0 ? 'text-green-400' : metrics.usageTrend < 0 ? 'text-red-400' : 'text-slate-400'}`}>
                          {metrics.usageTrend > 0 ? (
                            <ArrowUpRight className="w-4 h-4" />
                          ) : metrics.usageTrend < 0 ? (
                            <ArrowDownRight className="w-4 h-4" />
                          ) : null}
                          <span>{metrics.usageTrend > 0 ? '+' : ''}{metrics.usageTrend}</span>
                        </div>
                      </div>
                      
                      <div className="h-40 flex items-end space-x-2">
                        {performanceData.usage.map((data, index) => (
                          <div key={index} className="flex-1 flex flex-col items-center">
                            <div 
                              className="w-full bg-gradient-to-t from-purple-500/50 to-purple-500/10 rounded-t-sm"
                              style={{ height: `${(data.count / 80) * 100}%` }}
                            ></div>
                            <div className="text-xs text-slate-500 mt-1 rotate-45 origin-left translate-y-2">
                              {data.date.split('-')[2]}
                            </div>
                          </div>
                        ))}
                      </div>
                      
                      <div className="mt-6 text-center">
                        <div className="text-2xl font-bold text-white">{metrics.usage}</div>
                        <div className="text-sm text-slate-400">Inferences today</div>
                      </div>
                    </div>
                  </div>
                  
                  {/* Additional Metrics */}
                  <div className="bg-slate-700/30 rounded-xl p-6 border border-slate-600/30">
                    <h3 className="text-lg font-semibold text-white mb-4">Detailed Metrics</h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                      <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/50">
                        <p className="text-slate-400 text-sm mb-1">Total Inferences</p>
                        <p className="text-xl font-bold text-white">{agent.totalInferences?.toLocaleString() || '0'}</p>
                      </div>
                      <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/50">
                        <p className="text-slate-400 text-sm mb-1">Avg. Duration</p>
                        <p className="text-xl font-bold text-white">{(metrics.responseTime || 0).toFixed(1)}s</p>
                      </div>
                      <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/50">
                        <p className="text-slate-400 text-sm mb-1">Error Rate</p>
                        <p className="text-xl font-bold text-white">{(100 - (metrics.successRate || 0)).toFixed(1)}%</p>
                      </div>
                      <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/50">
                        <p className="text-slate-400 text-sm mb-1">Uptime</p>
                        <p className="text-xl font-bold text-white">99.7%</p>
                      </div>
                    </div>
                  </div>
                  
                  {/* Usage by Target */}
                  <div className="bg-slate-700/30 rounded-xl p-6 border border-slate-600/30">
                    <h3 className="text-lg font-semibold text-white mb-4">Usage by Target</h3>
                    <div className="space-y-4">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-3">
                          <div className="p-2 bg-blue-500/20 rounded-lg border border-blue-500/30">
                            <Target className="w-4 h-4 text-blue-400" />
                          </div>
                          <span className="text-white">Chrome Browser</span>
                        </div>
                        <span className="text-slate-300">78%</span>
                      </div>
                      <div className="w-full bg-slate-800 rounded-full h-2">
                        <div className="bg-gradient-to-r from-blue-500 to-cyan-500 h-2 rounded-full" style={{ width: '78%' }}></div>
                      </div>
                      
                      <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-3">
                          <div className="p-2 bg-green-500/20 rounded-lg border border-green-500/30">
                            <Target className="w-4 h-4 text-green-400" />
                          </div>
                          <span className="text-white">File System</span>
                        </div>
                        <span className="text-slate-300">15%</span>
                      </div>
                      <div className="w-full bg-slate-800 rounded-full h-2">
                        <div className="bg-gradient-to-r from-green-500 to-emerald-500 h-2 rounded-full" style={{ width: '15%' }}></div>
                      </div>
                      
                      <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-3">
                          <div className="p-2 bg-purple-500/20 rounded-lg border border-purple-500/30">
                            <Target className="w-4 h-4 text-purple-400" />
                          </div>
                          <span className="text-white">Other Targets</span>
                        </div>
                        <span className="text-slate-300">7%</span>
                      </div>
                      <div className="w-full bg-slate-800 rounded-full h-2">
                        <div className="bg-gradient-to-r from-purple-500 to-pink-500 h-2 rounded-full" style={{ width: '7%' }}></div>
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

export default AgentDetailModal;