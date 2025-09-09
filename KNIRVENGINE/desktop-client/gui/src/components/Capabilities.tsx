import React, { useState, useEffect } from 'react';
import { Zap, Settings, Download, Eye, Code, Shield, Globe, Database, Terminal, Package } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import { CapabilityStore } from './CapabilityStore';
import { MCPCapabilityManager } from './MCPCapabilityManager';
import { MCPServerBrowser } from './MCPServerBrowser';

export const Capabilities: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  // Check if we're on a sub-route
  const isSubRoute = location.pathname !== '/capabilities';

  // If we're on a sub-route, render the sub-component
  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/capability-store" element={<CapabilityStore />} />
        <Route path="/mcp-manager" element={<MCPCapabilityManager />} />
        <Route path="/mcp-servers" element={<MCPServerBrowser />} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-blue-500/20 rounded-lg">
          <Code className="w-6 h-6 text-blue-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Capabilities</h1>
          <p className="text-slate-400">Model Context Protocol (MCP) Capabilities Management</p>
        </div>
      </div>

      {/* Capabilities Options */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {canAccessSubPage('capabilities', 'capability-store') && (
          <button
            onClick={() => navigate('/capabilities/capability-store')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-blue-500/20 rounded-lg group-hover:bg-blue-500/30 transition-colors">
                <Package className="w-6 h-6 text-blue-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Capability Store</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Browse and install MCP capabilities including computer vision, natural language processing, code analysis, and system tools.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                <span className="text-slate-300">8 categories</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">15 installed</span>
              </div>
            </div>
          </button>
        )}

        {canAccessSubPage('capabilities', 'mcp-manager') && (
          <button
            onClick={() => navigate('/capabilities/mcp-manager')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-purple-500/20 rounded-lg group-hover:bg-purple-500/30 transition-colors">
                <Settings className="w-6 h-6 text-purple-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">MCP Manager</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Manage your installed MCP capabilities, configure settings, and monitor capability performance and usage.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">12 active</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
                <span className="text-slate-300">3 pending</span>
              </div>
            </div>
          </button>
        )}

        {canAccessSubPage('capabilities', 'mcp-servers') && (
          <button
            onClick={() => navigate('/capabilities/mcp-servers')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-green-500/20 rounded-lg group-hover:bg-green-500/30 transition-colors">
                <Terminal className="w-6 h-6 text-green-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">MCP Servers</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Browse and connect to MCP servers that provide various capabilities and tools for AI agents and applications.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">5 connected</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                <span className="text-slate-300">20 available</span>
              </div>
            </div>
          </button>
        )}
      </div>

      {/* Quick Overview */}
      <div className="mt-8 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">MCP Capabilities Overview</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-400">15</div>
            <div className="text-sm text-slate-400">Installed Capabilities</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-400">5</div>
            <div className="text-sm text-slate-400">Connected Servers</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-400">8</div>
            <div className="text-sm text-slate-400">Categories</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-yellow-400">98%</div>
            <div className="text-sm text-slate-400">Uptime</div>
          </div>
        </div>
      </div>

      {/* Capability Categories */}
      <div className="mt-6 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Capability Categories</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Eye className="w-8 h-8 text-blue-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Computer Vision</h3>
            <p className="text-slate-400 text-sm">Image analysis and visual understanding</p>
          </div>
          
          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Code className="w-8 h-8 text-green-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Code Analysis</h3>
            <p className="text-slate-400 text-sm">Code understanding and generation</p>
          </div>
          
          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Globe className="w-8 h-8 text-purple-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Web Interaction</h3>
            <p className="text-slate-400 text-sm">Browser automation and web APIs</p>
          </div>
          
          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Database className="w-8 h-8 text-yellow-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Data Processing</h3>
            <p className="text-slate-400 text-sm">Data analysis and transformation</p>
          </div>
          
          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Terminal className="w-8 h-8 text-red-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">System Tools</h3>
            <p className="text-slate-400 text-sm">System interaction and automation</p>
          </div>
          
          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Shield className="w-8 h-8 text-orange-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Security</h3>
            <p className="text-slate-400 text-sm">Security analysis and protection</p>
          </div>
          
          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Package className="w-8 h-8 text-cyan-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Natural Language</h3>
            <p className="text-slate-400 text-sm">Text processing and understanding</p>
          </div>
          
          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Zap className="w-8 h-8 text-pink-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">AI/ML</h3>
            <p className="text-slate-400 text-sm">Machine learning and AI tools</p>
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="mt-6 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Recent Capability Activity</h2>
        <div className="space-y-3">
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">GPT-4 Vision Analysis capability installed</div>
              <div className="text-xs text-slate-400">5 minutes ago</div>
            </div>
            <div className="text-xs text-green-400">Installed</div>
          </div>
          
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Web Scraping Tools capability updated</div>
              <div className="text-xs text-slate-400">12 minutes ago</div>
            </div>
            <div className="text-xs text-blue-400">Updated</div>
          </div>
          
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Connected to new MCP server: DataAnalytics Pro</div>
              <div className="text-xs text-slate-400">25 minutes ago</div>
            </div>
            <div className="text-xs text-purple-400">Connected</div>
          </div>
          
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Code Analysis capability configuration updated</div>
              <div className="text-xs text-slate-400">1 hour ago</div>
            </div>
            <div className="text-xs text-yellow-400">Configured</div>
          </div>
        </div>
      </div>

      {/* Capabilities vs Skills Info */}
      <div className="mt-6 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Capabilities vs Skills</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-slate-700/30 rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-3">
              <Code className="w-5 h-5 text-blue-400" />
              <h3 className="text-white font-medium">Capabilities (This Section)</h3>
            </div>
            <p className="text-slate-400 text-sm">
              MCP (Model Context Protocol) capabilities provide foundational tools and interfaces for AI agents to interact with systems, APIs, and data sources. These are typically free and provided by the community.
            </p>
          </div>
          
          <div className="bg-slate-700/30 rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-3">
              <Zap className="w-5 h-5 text-yellow-400" />
              <h3 className="text-white font-medium">Skills (Marketplace)</h3>
            </div>
            <p className="text-slate-400 text-sm">
              Skills are specialized AI functions available on the decentralized marketplace. They can be purchased, installed, and executed with NRN tokens, with execution costs and complexity ratings.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};
