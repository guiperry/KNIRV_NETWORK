import React, { useState, useEffect } from 'react';
import { Package, Upload, Download, Settings, CheckCircle, AlertTriangle, Clock, Plus } from 'lucide-react';
import { useAuth } from '../AuthContext';

interface CodexItem {
  id: string;
  name: string;
  type: 'model' | 'dataset' | 'skill' | 'capability';
  version: string;
  status: 'active' | 'pending' | 'error' | 'archived';
  size: string;
  lastUpdated: Date;
  description: string;
}

interface InventoryStats {
  totalItems: number;
  activeItems: number;
  pendingItems: number;
  totalSize: string;
}

const CodexBuilder: React.FC = () => {
  const { user } = useAuth();
  const [items, setItems] = useState<CodexItem[]>([]);
  const [stats, setStats] = useState<InventoryStats>({
    totalItems: 0,
    activeItems: 0,
    pendingItems: 0,
    totalSize: '0 MB'
  });
  const [selectedType, setSelectedType] = useState<string>('all');
  const [isUploading, setIsUploading] = useState(false);

  useEffect(() => {
    // Simulate loading inventory data
    const mockItems: CodexItem[] = [
      {
        id: 'model-1',
        name: 'GPT-4 Turbo',
        type: 'model',
        version: '1.0.0',
        status: 'active',
        size: '1.2 GB',
        lastUpdated: new Date(Date.now() - 86400000),
        description: 'Advanced language model for general tasks'
      },
      {
        id: 'skill-1',
        name: 'Code Generator',
        type: 'skill',
        version: '2.1.0',
        status: 'active',
        size: '45 MB',
        lastUpdated: new Date(Date.now() - 3600000),
        description: 'Automated code generation and optimization'
      },
      {
        id: 'dataset-1',
        name: 'Training Dataset Alpha',
        type: 'dataset',
        version: '1.5.2',
        status: 'pending',
        size: '850 MB',
        lastUpdated: new Date(Date.now() - 7200000),
        description: 'Curated training data for model fine-tuning'
      },
      {
        id: 'capability-1',
        name: 'Image Recognition',
        type: 'capability',
        version: '3.0.1',
        status: 'active',
        size: '320 MB',
        lastUpdated: new Date(Date.now() - 1800000),
        description: 'Advanced computer vision capabilities'
      },
      {
        id: 'model-2',
        name: 'Claude Sonnet',
        type: 'model',
        version: '2.0.0',
        status: 'error',
        size: '980 MB',
        lastUpdated: new Date(Date.now() - 10800000),
        description: 'Anthropic Claude model integration'
      }
    ];

    setItems(mockItems);
    
    const activeItems = mockItems.filter(item => item.status === 'active').length;
    const pendingItems = mockItems.filter(item => item.status === 'pending').length;
    
    setStats({
      totalItems: mockItems.length,
      activeItems,
      pendingItems,
      totalSize: '3.4 GB'
    });
  }, []);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'pending':
        return <Clock className="w-4 h-4 text-yellow-500" />;
      case 'error':
        return <AlertTriangle className="w-4 h-4 text-red-500" />;
      default:
        return <div className="w-4 h-4 bg-slate-500 rounded-full" />;
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'model':
        return 'text-blue-400 bg-blue-500/20';
      case 'skill':
        return 'text-green-400 bg-green-500/20';
      case 'dataset':
        return 'text-purple-400 bg-purple-500/20';
      case 'capability':
        return 'text-yellow-400 bg-yellow-500/20';
      default:
        return 'text-slate-400 bg-slate-500/20';
    }
  };

  const filteredItems = selectedType === 'all' 
    ? items 
    : items.filter(item => item.type === selectedType);

  const handleUpload = () => {
    setIsUploading(true);
    // Simulate upload process
    setTimeout(() => {
      setIsUploading(false);
      // Add new item to inventory
      const newItem: CodexItem = {
        id: `item-${Date.now()}`,
        name: 'New Upload',
        type: 'model',
        version: '1.0.0',
        status: 'pending',
        size: '125 MB',
        lastUpdated: new Date(),
        description: 'Recently uploaded item'
      };
      setItems(prev => [newItem, ...prev]);
    }, 2000);
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-blue-500/20 rounded-lg">
            <Package className="w-6 h-6 text-blue-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">Codex Builder</h1>
            <p className="text-slate-400">Manage your AI models, skills, and capabilities inventory</p>
          </div>
        </div>
        
        <button
          onClick={handleUpload}
          disabled={isUploading}
          className="flex items-center space-x-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          {isUploading ? (
            <>
              <div className="animate-spin w-4 h-4 border-2 border-white border-t-transparent rounded-full"></div>
              <span>Uploading...</span>
            </>
          ) : (
            <>
              <Plus className="w-4 h-4" />
              <span>Add Item</span>
            </>
          )}
        </button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Package className="w-5 h-5 text-blue-400" />
            <span className="text-sm text-slate-400">Total Items</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.totalItems}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <CheckCircle className="w-5 h-5 text-green-400" />
            <span className="text-sm text-slate-400">Active</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.activeItems}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Clock className="w-5 h-5 text-yellow-400" />
            <span className="text-sm text-slate-400">Pending</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.pendingItems}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Download className="w-5 h-5 text-purple-400" />
            <span className="text-sm text-slate-400">Total Size</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.totalSize}</div>
        </div>
      </div>

      {/* Filter Tabs */}
      <div className="flex space-x-2 mb-6">
        {['all', 'model', 'skill', 'dataset', 'capability'].map((type) => (
          <button
            key={type}
            onClick={() => setSelectedType(type)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              selectedType === type
                ? 'bg-blue-600/30 text-blue-300 border border-blue-500/50'
                : 'bg-slate-700/30 text-slate-400 hover:bg-slate-600/30 hover:text-slate-300'
            }`}
          >
            {type.charAt(0).toUpperCase() + type.slice(1)}
          </button>
        ))}
      </div>

      {/* Inventory Table */}
      <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg overflow-hidden">
        <div className="p-4 border-b border-slate-700/50">
          <h2 className="text-lg font-semibold text-white">Inventory Items</h2>
        </div>
        
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-slate-700/30">
              <tr>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Item</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Type</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Version</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Status</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Size</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Last Updated</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredItems.map((item) => (
                <tr key={item.id} className="border-t border-slate-700/30 hover:bg-slate-700/20">
                  <td className="p-4">
                    <div>
                      <div className="text-white font-medium">{item.name}</div>
                      <div className="text-xs text-slate-400">{item.description}</div>
                    </div>
                  </td>
                  <td className="p-4">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${getTypeColor(item.type)}`}>
                      {item.type}
                    </span>
                  </td>
                  <td className="p-4 text-white text-sm">{item.version}</td>
                  <td className="p-4">
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(item.status)}
                      <span className="text-sm text-white capitalize">{item.status}</span>
                    </div>
                  </td>
                  <td className="p-4 text-white text-sm">{item.size}</td>
                  <td className="p-4 text-slate-400 text-sm">
                    {item.lastUpdated.toLocaleDateString()}
                  </td>
                  <td className="p-4">
                    <div className="flex items-center space-x-2">
                      <button className="p-1 text-slate-400 hover:text-blue-400 transition-colors">
                        <Settings className="w-4 h-4" />
                      </button>
                      <button className="p-1 text-slate-400 hover:text-green-400 transition-colors">
                        <Download className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default CodexBuilder;
