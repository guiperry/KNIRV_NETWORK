import React, { useState, useEffect } from 'react';
import { Monitor as MonitorIcon, Activity, Globe, BarChart3, Wifi, Server, Database, Cpu, Search } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import NetworkMonitor from './monitor/NetworkMonitor';
import LocalAnalytics from './monitor/LocalAnalytics';
import NetworkExplorers from './monitor/NetworkExplorers';

export const Monitor: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  // Check if we're on a sub-route
  const isSubRoute = location.pathname !== '/monitor';

  // If we're on a sub-route, render the sub-component
  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/network-monitor" element={<NetworkMonitor />} />
        <Route path="/local-analytics" element={<LocalAnalytics />} />
        <Route path="/network-explorers/*" element={<NetworkExplorers />} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-blue-500/20 rounded-lg">
          <MonitorIcon className="w-6 h-6 text-blue-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Monitor</h1>
          <p className="text-slate-400">Network monitoring and analytics dashboard</p>
        </div>
      </div>

      {/* Monitor Options */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {canAccessSubPage('monitor', 'network-monitor') && (
          <button
            onClick={() => navigate('/monitor/network-monitor')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-blue-500/20 rounded-lg group-hover:bg-blue-500/30 transition-colors">
                <MonitorIcon className="w-6 h-6 text-blue-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Network Monitor</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Real-time monitoring of KNIRV network nodes, connections, and health status.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">12 nodes online</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                <span className="text-slate-300">98% uptime</span>
              </div>
            </div>
          </button>
        )}

        {canAccessSubPage('monitor', 'local-analytics') && (
          <button
            onClick={() => navigate('/monitor/local-analytics')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-green-500/20 rounded-lg group-hover:bg-green-500/30 transition-colors">
                <BarChart3 className="w-6 h-6 text-green-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Local Analytics</h3>
            </div>
            <p className="text-slate-400 mb-4">
              System performance metrics, resource usage, and local activity analytics.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <Cpu className="w-3 h-3 text-blue-400" />
                <span className="text-slate-300">CPU: 45%</span>
              </div>
              <div className="flex items-center space-x-1">
                <Activity className="w-3 h-3 text-purple-400" />
                <span className="text-slate-300">Memory: 62%</span>
              </div>
            </div>
          </button>
        )}

        {canAccessSubPage('monitor', 'network-explorers') && (
          <button
            onClick={() => navigate('/monitor/network-explorers')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-purple-500/20 rounded-lg group-hover:bg-purple-500/30 transition-colors">
                <Search className="w-6 h-6 text-purple-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Network Explorers</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Explore Graph, Chain, Oracle, Router, and Nexus components in detail.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <Globe className="w-3 h-3 text-purple-400" />
                <span className="text-slate-300">5 explorers</span>
              </div>
              <div className="flex items-center space-x-1">
                <Database className="w-3 h-3 text-yellow-400" />
                <span className="text-slate-300">Live data</span>
              </div>
            </div>
          </button>
        )}
      </div>

      {/* Quick Stats */}
      <div className="mt-8 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Quick Overview</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-400">12</div>
            <div className="text-sm text-slate-400">Active Nodes</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-400">98.7%</div>
            <div className="text-sm text-slate-400">Network Health</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-400">1.2K</div>
            <div className="text-sm text-slate-400">Requests/min</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-yellow-400">45ms</div>
            <div className="text-sm text-slate-400">Avg Latency</div>
          </div>
        </div>
      </div>
    </div>
  );
};

