import React, { useState, useEffect } from 'react';
import {
  Bell,
  X,
  CheckCircle,
  AlertTriangle,
  Info,
  Zap,
  Bot,
  Target,
  Activity,
  Clock,
  Settings,
  Download
} from 'lucide-react';
import { wsManager } from '../utils/websocket';

const RealTimeNotifications = () => {
  const [notifications, setNotifications] = useState([]);
  const [isOpen, setIsOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);

  useEffect(() => {
    // Subscribe to various WebSocket events for notifications
    const handleAgentStatus = (data) => {
      addNotification({
        id: Date.now() + Math.random(),
        type: data.status === 'error' ? 'error' : 'info',
        title: 'Agent Status Update',
        message: `Agent ${data.agentId} is now ${data.status}`,
        timestamp: new Date().toISOString(),
        icon: Bot,
        category: 'agent'
      });
    };

    const handleBuildProgress = (data) => {
      if (data.status === 'completed') {
        addNotification({
          id: Date.now() + Math.random(),
          type: 'success',
          title: 'Build Completed',
          message: `Agent ${data.agentId} build finished successfully`,
          timestamp: new Date().toISOString(),
          icon: CheckCircle,
          category: 'build'
        });
      } else if (data.status === 'failed') {
        addNotification({
          id: Date.now() + Math.random(),
          type: 'error',
          title: 'Build Failed',
          message: `Agent ${data.agentId} build failed: ${data.error}`,
          timestamp: new Date().toISOString(),
          icon: AlertTriangle,
          category: 'build'
        });
      }
    };

    const handleSystemMetrics = (data) => {
      // Only notify for critical system issues
      if (data.cpu > 90) {
        addNotification({
          id: Date.now() + Math.random(),
          type: 'warning',
          title: 'High CPU Usage',
          message: `System CPU usage is at ${data.cpu.toFixed(1)}%`,
          timestamp: new Date().toISOString(),
          icon: AlertTriangle,
          category: 'system'
        });
      }
      
      if (data.memory > 95) {
        addNotification({
          id: Date.now() + Math.random(),
          type: 'error',
          title: 'Critical Memory Usage',
          message: `System memory usage is at ${data.memory.toFixed(1)}%`,
          timestamp: new Date().toISOString(),
          icon: AlertTriangle,
          category: 'system'
        });
      }
    };

    const handleSubAgentUpdate = (data) => {
      addNotification({
        id: Date.now() + Math.random(),
        type: 'info',
        title: 'Sub-Agent Update',
        message: `Sub-agent ${data.subAgentId} status: ${data.status}`,
        timestamp: new Date().toISOString(),
        icon: Activity,
        category: 'sub-agent'
      });
    };

    // Subscribe to WebSocket events
    if (wsManager && wsManager.subscribe) {
      wsManager.subscribe('agent_status', handleAgentStatus);
      wsManager.subscribe('build_progress', handleBuildProgress);
      wsManager.subscribe('system_metrics', handleSystemMetrics);
      wsManager.subscribe('sub_agent_update', handleSubAgentUpdate);
    }

    // Simulate some initial notifications for demo
    setTimeout(() => {
      addNotification({
        id: Date.now() + Math.random(),
        type: 'success',
        title: 'System Ready',
        message: 'KNIRVENGINE is running and ready for deployment',
        timestamp: new Date().toISOString(),
        icon: CheckCircle,
        category: 'system'
      });
    }, 1000);

    return () => {
      // In a real implementation, unsubscribe from WebSocket events
      if (wsManager && wsManager.unsubscribe) {
        wsManager.unsubscribe('agent_status', handleAgentStatus);
        wsManager.unsubscribe('build_progress', handleBuildProgress);
        wsManager.unsubscribe('system_metrics', handleSystemMetrics);
        wsManager.unsubscribe('sub_agent_update', handleSubAgentUpdate);
      }
    };
  }, []);

  const addNotification = (notification) => {
    setNotifications(prev => [notification, ...prev.slice(0, 49)]); // Keep last 50 notifications
    setUnreadCount(prev => prev + 1);
  };

  const markAsRead = (notificationId) => {
    setNotifications(prev => 
      prev.map(notif => 
        notif.id === notificationId ? { ...notif, read: true } : notif
      )
    );
    setUnreadCount(prev => Math.max(0, prev - 1));
  };

  const markAllAsRead = () => {
    setNotifications(prev => prev.map(notif => ({ ...notif, read: true })));
    setUnreadCount(0);
  };

  const removeNotification = (notificationId) => {
    setNotifications(prev => prev.filter(notif => notif.id !== notificationId));
    const notification = notifications.find(n => n.id === notificationId);
    if (notification && !notification.read) {
      setUnreadCount(prev => Math.max(0, prev - 1));
    }
  };

  const clearAll = () => {
    setNotifications([]);
    setUnreadCount(0);
  };

  const getNotificationIcon = (type) => {
    switch (type) {
      case 'success': return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'error': return <AlertTriangle className="w-5 h-5 text-red-400" />;
      case 'warning': return <AlertTriangle className="w-5 h-5 text-yellow-400" />;
      case 'info': 
      default: return <Info className="w-5 h-5 text-blue-400" />;
    }
  };

  const getNotificationBg = (type) => {
    switch (type) {
      case 'success': return 'bg-green-900/20 border-green-500/30';
      case 'error': return 'bg-red-900/20 border-red-500/30';
      case 'warning': return 'bg-yellow-900/20 border-yellow-500/30';
      case 'info': 
      default: return 'bg-blue-900/20 border-blue-500/30';
    }
  };

  const formatTime = (timestamp) => {
    const now = new Date();
    const time = new Date(timestamp);
    const diffMs = now - time;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${diffDays}d ago`;
  };

  const groupedNotifications = notifications.reduce((groups, notification) => {
    const category = notification.category || 'other';
    if (!groups[category]) groups[category] = [];
    groups[category].push(notification);
    return groups;
  }, {});

  return (
    <div className="relative">
      {/* Notification Bell */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="relative p-2 text-slate-400 hover:text-white transition-colors duration-200"
      >
        <Bell className="w-6 h-6" />
        {unreadCount > 0 && (
          <span className="absolute -top-1 -right-1 bg-red-500 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {/* Notification Panel */}
      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-96 bg-slate-800 rounded-lg shadow-xl border border-slate-700 z-50 max-h-96 overflow-hidden">
          {/* Header */}
          <div className="flex items-center justify-between p-4 border-b border-slate-700">
            <h3 className="text-lg font-semibold text-white">Notifications</h3>
            <div className="flex items-center space-x-2">
              {unreadCount > 0 && (
                <button
                  onClick={markAllAsRead}
                  className="text-xs text-blue-400 hover:text-blue-300"
                >
                  Mark all read
                </button>
              )}
              <button
                onClick={clearAll}
                className="text-xs text-slate-400 hover:text-slate-300"
              >
                Clear all
              </button>
              <button
                onClick={() => setIsOpen(false)}
                className="text-slate-400 hover:text-white"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Notifications List */}
          <div className="max-h-80 overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="p-8 text-center text-slate-400">
                <Bell className="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>No notifications yet</p>
              </div>
            ) : (
              Object.entries(groupedNotifications).map(([category, categoryNotifications]) => (
                <div key={category}>
                  <div className="px-4 py-2 bg-slate-700/50 border-b border-slate-700">
                    <h4 className="text-sm font-medium text-slate-300 capitalize">
                      {category} ({categoryNotifications.length})
                    </h4>
                  </div>
                  {categoryNotifications.map((notification) => {
                    const Icon = notification.icon;
                    return (
                      <div
                        key={notification.id}
                        className={`p-4 border-b border-slate-700/50 hover:bg-slate-700/30 transition-colors cursor-pointer ${
                          !notification.read ? 'bg-slate-700/20' : ''
                        }`}
                        onClick={() => markAsRead(notification.id)}
                      >
                        <div className="flex items-start space-x-3">
                          <div className={`p-2 rounded-lg ${getNotificationBg(notification.type)}`}>
                            {Icon ? <Icon className="w-4 h-4" /> : getNotificationIcon(notification.type)}
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center justify-between">
                              <h4 className="text-sm font-medium text-white truncate">
                                {notification.title}
                              </h4>
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  removeNotification(notification.id);
                                }}
                                className="text-slate-400 hover:text-slate-300 ml-2"
                              >
                                <X className="w-3 h-3" />
                              </button>
                            </div>
                            <p className="text-sm text-slate-300 mt-1 line-clamp-2">
                              {notification.message}
                            </p>
                            <div className="flex items-center justify-between mt-2">
                              <span className="text-xs text-slate-500">
                                {formatTime(notification.timestamp)}
                              </span>
                              {!notification.read && (
                                <div className="w-2 h-2 bg-blue-500 rounded-full" />
                              )}
                            </div>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default RealTimeNotifications;
