import React, { useState, useEffect } from 'react';
import { Brain, Code, Settings, Vote, Play, Pause, Download, Upload, Package } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import CodexBuilder from './models/CodexBuilder';
import FallbackConfig from './models/FallbackConfig';
import DAOVoting from './models/DAOVoting';

export const Models: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  // Check if we're on a sub-route
  const isSubRoute = location.pathname !== '/models';

  // If we're on a sub-route, render the sub-component
  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/codex-builder" element={<CodexBuilder />} />
        <Route path="/fallback-config" element={<FallbackConfig />} />
        <Route path="/dao-voting" element={<DAOVoting />} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-purple-500/20 rounded-lg">
          <Brain className="w-6 h-6 text-purple-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Models</h1>
          <p className="text-slate-400">AI model management and governance dashboard</p>
        </div>
      </div>

      {/* Model Management Options */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {canAccessSubPage('models', 'codex-builder') && (
          <button
            onClick={() => navigate('/models/codex-builder')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-blue-500/20 rounded-lg group-hover:bg-blue-500/30 transition-colors">
                <Package className="w-6 h-6 text-blue-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Codex Builder</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Manage your AI models, skills, datasets, and capabilities inventory. Build and organize your AI codex.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                <span className="text-slate-300">15 models</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">8 skills</span>
              </div>
            </div>
          </button>
        )}

        {canAccessSubPage('models', 'fallback-config') && (
          <button
            onClick={() => navigate('/models/fallback-config')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-orange-500/20 rounded-lg group-hover:bg-orange-500/30 transition-colors">
                <Settings className="w-6 h-6 text-orange-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Fallback API & HOM Config</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Configure optional fallback API providers and Health-Oriented Monitoring settings for robust AI operations.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">3 providers</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
                <span className="text-slate-300">HOM enabled</span>
              </div>
            </div>
          </button>
        )}

        {canAccessSubPage('models', 'dao-voting') && (
          <button
            onClick={() => navigate('/models/dao-voting')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-purple-500/20 rounded-lg group-hover:bg-purple-500/30 transition-colors">
                <Vote className="w-6 h-6 text-purple-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">DAO KNIRVCORTEX Voting</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Participate in shared model governance through decentralized voting on model additions, removals, and parameters.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
                <span className="text-slate-300">4 proposals</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
                <span className="text-slate-300">2 active</span>
              </div>
            </div>
          </button>
        )}
      </div>

      {/* Quick Overview */}
      <div className="mt-8 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Model Overview</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-400">15</div>
            <div className="text-sm text-slate-400">Total Models</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-400">12</div>
            <div className="text-sm text-slate-400">Active Models</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-400">3</div>
            <div className="text-sm text-slate-400">API Providers</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-yellow-400">78%</div>
            <div className="text-sm text-slate-400">DAO Participation</div>
          </div>
        </div>
      </div>
    </div>
  );
};

