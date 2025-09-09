import React, { useState, useEffect } from 'react';
import { Briefcase, Search, Filter, Star, Download, Upload, Play, Code, Database, Globe, Zap } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import SkillsDEX from './skills/SkillsDEX';

export const Skills: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  // Check if we're on a sub-route
  const isSubRoute = location.pathname !== '/skills';

  // If we're on a sub-route, render the sub-component
  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/skills-dex" element={<SkillsDEX />} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-yellow-500/20 rounded-lg">
          <Zap className="w-6 h-6 text-yellow-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Skills</h1>
          <p className="text-slate-400">Decentralized AI Skills Marketplace</p>
        </div>
      </div>

      {/* Skills Options */}
      <div className="grid grid-cols-1 md:grid-cols-1 gap-6">
        {canAccessSubPage('skills', 'skills-dex') && (
          <button
            onClick={() => navigate('/skills/skills-dex')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-yellow-500/20 rounded-lg group-hover:bg-yellow-500/30 transition-colors">
                <Zap className="w-6 h-6 text-yellow-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Skills DEX</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Discover, purchase, and install AI skills from the decentralized skills marketplace. Browse skills created by the KNIRV community with execution costs, complexity ratings, and dependency management.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
                <span className="text-slate-300">5 skills available</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">3 installed</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                <span className="text-slate-300">2 owned</span>
              </div>
            </div>
          </button>
        )}
      </div>

      {/* Quick Overview */}
      <div className="mt-8 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Skills Marketplace Overview</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="text-2xl font-bold text-yellow-400">5</div>
            <div className="text-sm text-slate-400">Available Skills</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-400">3</div>
            <div className="text-sm text-slate-400">Installed</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-400">2</div>
            <div className="text-sm text-slate-400">Owned</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-400">4.7</div>
            <div className="text-sm text-slate-400">Avg Rating</div>
          </div>
        </div>
      </div>

      {/* Featured Skills */}
      <div className="mt-6 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Featured Skills</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div className="bg-slate-700/30 rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-2">
              <Code className="w-5 h-5 text-blue-400" />
              <h3 className="text-white font-medium">Code Generator Pro</h3>
            </div>
            <p className="text-slate-400 text-sm mb-2">Advanced code generation with multi-language support</p>
            <div className="flex items-center space-x-2 text-xs">
              <div className="flex items-center space-x-1">
                <Star className="w-3 h-3 text-yellow-400 fill-current" />
                <span className="text-slate-300">4.8</span>
              </div>
              <span className="text-slate-400">15.4K downloads</span>
              <span className="text-green-400 font-medium">Free</span>
            </div>
          </div>

          <div className="bg-slate-700/30 rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-2">
              <Database className="w-5 h-5 text-purple-400" />
              <h3 className="text-white font-medium">Data Analyzer Suite</h3>
            </div>
            <p className="text-slate-400 text-sm mb-2">Comprehensive data analysis with visualization</p>
            <div className="flex items-center space-x-2 text-xs">
              <div className="flex items-center space-x-1">
                <Star className="w-3 h-3 text-yellow-400 fill-current" />
                <span className="text-slate-300">4.6</span>
              </div>
              <span className="text-slate-400">8.9K downloads</span>
              <span className="text-yellow-400 font-medium">$29.99</span>
            </div>
          </div>

          <div className="bg-slate-700/30 rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-2">
              <Play className="w-5 h-5 text-green-400" />
              <h3 className="text-white font-medium">Smart Automation</h3>
            </div>
            <p className="text-slate-400 text-sm mb-2">Intelligent workflow automation engine</p>
            <div className="flex items-center space-x-2 text-xs">
              <div className="flex items-center space-x-1">
                <Star className="w-3 h-3 text-yellow-400 fill-current" />
                <span className="text-slate-300">4.9</span>
              </div>
              <span className="text-slate-400">12.7K downloads</span>
              <span className="text-yellow-400 font-medium">$49.99</span>
            </div>
          </div>
        </div>
      </div>

      {/* Skills vs Capabilities Info */}
      <div className="mt-6 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Skills vs Capabilities</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-slate-700/30 rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-3">
              <Zap className="w-5 h-5 text-yellow-400" />
              <h3 className="text-white font-medium">Skills (This Section)</h3>
            </div>
            <p className="text-slate-400 text-sm">
              AI Skills are specialized functions available on the decentralized marketplace. They can be purchased, installed, and executed with NRN tokens. Each skill has execution costs, complexity ratings, and specific requirements.
            </p>
          </div>

          <div className="bg-slate-700/30 rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-3">
              <Code className="w-5 h-5 text-blue-400" />
              <h3 className="text-white font-medium">Capabilities (MCP)</h3>
            </div>
            <p className="text-slate-400 text-sm">
              Capabilities refer to the Model Context Protocol (MCP) capabilities that provide foundational tools and interfaces for AI agents to interact with systems, APIs, and data sources.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};


