import React, { useState, useEffect } from 'react';
import { Layers, Image, FileText, Code, Music, Video, Lock, Unlock, Eye, Download, Upload, Plus, Package, Shield } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import { NFTManager } from './NFTManager';

export const Properties: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  // Check if we're on a sub-route
  const isSubRoute = location.pathname !== '/properties';

  // If we're on a sub-route, render the sub-component
  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/nft-ip-vault" element={<NFTManager />} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-purple-500/20 rounded-lg">
          <Package className="w-6 h-6 text-purple-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Properties</h1>
          <p className="text-slate-400">NFT IP Vault and Intellectual Property Management</p>
        </div>
      </div>

      {/* Properties Options */}
      <div className="grid grid-cols-1 md:grid-cols-1 gap-6">
        {canAccessSubPage('properties', 'nft-ip-vault') && (
          <button
            onClick={() => navigate('/properties/nft-ip-vault')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-purple-500/20 rounded-lg group-hover:bg-purple-500/30 transition-colors">
                <Shield className="w-6 h-6 text-purple-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">NFT IP Vault</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Manage your NFT intellectual property vault. Store, organize, and control access to digital assets, smart contracts, documents, and other IP-related content with blockchain-based ownership verification.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
                <span className="text-slate-300">6 properties</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">4 public</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
                <span className="text-slate-300">2 private</span>
              </div>
            </div>
          </button>
        )}
      </div>

      {/* Quick Overview */}
      <div className="mt-8 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">IP Vault Overview</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-400">6</div>
            <div className="text-sm text-slate-400">Total Properties</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-400">4</div>
            <div className="text-sm text-slate-400">Public Access</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-yellow-400">2</div>
            <div className="text-sm text-slate-400">Private Access</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-400">12.8</div>
            <div className="text-sm text-slate-400">Total Size (MB)</div>
          </div>
        </div>
      </div>

      {/* Property Types */}
      <div className="mt-6 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Property Types</h2>
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Image className="w-8 h-8 text-blue-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Images</h3>
            <p className="text-slate-400 text-sm">2 items</p>
          </div>

          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <FileText className="w-8 h-8 text-green-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Documents</h3>
            <p className="text-slate-400 text-sm">1 item</p>
          </div>

          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Code className="w-8 h-8 text-purple-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Code</h3>
            <p className="text-slate-400 text-sm">2 items</p>
          </div>

          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Music className="w-8 h-8 text-yellow-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Audio</h3>
            <p className="text-slate-400 text-sm">1 item</p>
          </div>

          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Video className="w-8 h-8 text-red-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Video</h3>
            <p className="text-slate-400 text-sm">0 items</p>
          </div>

          <div className="bg-slate-700/30 rounded-lg p-4 text-center">
            <Layers className="w-8 h-8 text-cyan-400 mx-auto mb-2" />
            <h3 className="text-white font-medium mb-1">Data</h3>
            <p className="text-slate-400 text-sm">0 items</p>
          </div>
        </div>
      </div>

      {/* Recent Properties */}
      <div className="mt-6 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Recent Properties</h2>
        <div className="space-y-3">
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <Image className="w-5 h-5 text-blue-400" />
            <div className="flex-1">
              <div className="text-sm text-white">KNIRV Network Logo</div>
              <div className="text-xs text-slate-400">Official brand assets • 2.4 MB • Public</div>
            </div>
            <div className="text-xs text-green-400">Active</div>
          </div>

          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <Code className="w-5 h-5 text-purple-400" />
            <div className="flex-1">
              <div className="text-sm text-white">Smart Contract Template</div>
              <div className="text-xs text-slate-400">Solidity contract • 45 KB • Public</div>
            </div>
            <div className="text-xs text-green-400">Active</div>
          </div>

          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <FileText className="w-5 h-5 text-green-400" />
            <div className="flex-1">
              <div className="text-sm text-white">Technical Whitepaper</div>
              <div className="text-xs text-slate-400">PDF document • 1.8 MB • Private</div>
            </div>
            <div className="text-xs text-yellow-400">Restricted</div>
          </div>

          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <Music className="w-5 h-5 text-yellow-400" />
            <div className="flex-1">
              <div className="text-sm text-white">KNIRV Theme Audio</div>
              <div className="text-xs text-slate-400">MP3 audio • 3.2 MB • Public</div>
            </div>
            <div className="text-xs text-green-400">Active</div>
          </div>
        </div>
      </div>

      {/* Access Control Info */}
      <div className="mt-6 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Access Control</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="bg-slate-700/30 rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-3">
              <Unlock className="w-5 h-5 text-green-400" />
              <h3 className="text-white font-medium">Public</h3>
            </div>
            <p className="text-slate-400 text-sm">
              Publicly accessible properties that can be viewed and used by anyone in the KNIRV network.
            </p>
          </div>

          <div className="bg-slate-700/30 rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-3">
              <Lock className="w-5 h-5 text-yellow-400" />
              <h3 className="text-white font-medium">Private</h3>
            </div>
            <p className="text-slate-400 text-sm">
              Private properties accessible only to the owner and explicitly granted users.
            </p>
          </div>

          <div className="bg-slate-700/30 rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-3">
              <Shield className="w-5 h-5 text-red-400" />
              <h3 className="text-white font-medium">Restricted</h3>
            </div>
            <p className="text-slate-400 text-sm">
              Highly restricted properties with advanced access controls and usage limitations.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

