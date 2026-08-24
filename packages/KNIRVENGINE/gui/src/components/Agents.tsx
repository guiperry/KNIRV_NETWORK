import React, { useState, useEffect } from 'react';
import { Users, Bot, Target, Workflow, Settings, Play, Pause, TrendingUp } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import { AgentManager } from './AgentManager';
import { TargetManager } from './TargetManager';
import { WorkflowOrchestrator } from './WorkflowOrchestrator';

export const Agents: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  // Check if we're on a sub-route
  const isSubRoute = location.pathname !== '/agents';

  // If we're on a sub-route, render the sub-component
  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/my-agents" element={<AgentManager />} />
        <Route path="/my-targets" element={<TargetManager />} />
        <Route path="/my-workflows" element={<WorkflowOrchestrator />} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-blue-500/20 rounded-lg">
          <Users className="w-6 h-6 text-blue-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Agents</h1>
          <p className="text-slate-400">Manage your AI agents, targets, and workflows</p>
        </div>
      </div>

      {/* Agent Management Options */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {canAccessSubPage('agents', 'my-agents') && (
          <button
            onClick={() => navigate('/agents/my-agents')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-blue-500/20 rounded-lg group-hover:bg-blue-500/30 transition-colors">
                <Bot className="w-6 h-6 text-blue-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">My Agents</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Create, configure, and manage your AI agents. Monitor performance and control agent operations.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">3 active</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                <span className="text-slate-300">2.5K tasks</span>
              </div>
            </div>
          </button>
        )}

        {canAccessSubPage('agents', 'my-targets') && (
          <button
            onClick={() => navigate('/agents/my-targets')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-purple-500/20 rounded-lg group-hover:bg-purple-500/30 transition-colors">
                <Target className="w-6 h-6 text-purple-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">My Targets</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Define performance targets and goals for your agents. Track progress and measure success.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
                <span className="text-slate-300">2 active</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">1 completed</span>
              </div>
            </div>
          </button>
        )}

        {canAccessSubPage('agents', 'my-workflows') && (
          <button
            onClick={() => navigate('/agents/my-workflows')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-green-500/20 rounded-lg group-hover:bg-green-500/30 transition-colors">
                <Workflow className="w-6 h-6 text-green-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">My Workflows</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Create and manage automated workflows. Connect agents, targets, and tasks for seamless automation.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">2 active</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse"></div>
                <span className="text-slate-300">1 running</span>
              </div>
            </div>
          </button>
        )}
      </div>

      {/* Quick Overview */}
      <div className="mt-8 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Agent Overview</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-400">3</div>
            <div className="text-sm text-slate-400">Active Agents</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-400">3</div>
            <div className="text-sm text-slate-400">Active Targets</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-400">3</div>
            <div className="text-sm text-slate-400">Workflows</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-yellow-400">92%</div>
            <div className="text-sm text-slate-400">Success Rate</div>
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="mt-6 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Recent Activity</h2>
        <div className="space-y-3">
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Code Assistant completed 15 tasks</div>
              <div className="text-xs text-slate-400">2 minutes ago</div>
            </div>
            <div className="text-xs text-green-400">Success</div>
          </div>
          
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Daily Report Generation workflow started</div>
              <div className="text-xs text-slate-400">5 minutes ago</div>
            </div>
            <div className="text-xs text-blue-400">Running</div>
          </div>
          
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Response Accuracy target updated</div>
              <div className="text-xs text-slate-400">12 minutes ago</div>
            </div>
            <div className="text-xs text-purple-400">Updated</div>
          </div>
          
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Agent Health Check workflow completed</div>
              <div className="text-xs text-slate-400">18 minutes ago</div>
            </div>
            <div className="text-xs text-yellow-400">Completed</div>
          </div>
        </div>
      </div>
    </div>
  );
};
