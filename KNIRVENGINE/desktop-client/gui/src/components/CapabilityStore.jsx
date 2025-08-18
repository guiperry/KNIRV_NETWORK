import React, { useState } from 'react';
import {
  Search,
  Filter,
  Star,
  Download,
  Settings,
  Zap,
  Brain,
  Image,
  FileText,
  Code,
  Shield,
  TrendingUp,
  Globe,
  Database,
  Terminal,
  Server,
  Package,
  Layers
} from 'lucide-react';
import MCPServerBrowser from './MCPServerBrowser';

export const CapabilityStore = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [activeTab, setActiveTab] = useState('capabilities');
  const [installedCapabilities, setInstalledCapabilities] = useState([]); // 'capabilities' | 'mcp-servers'

  const categories = [
    { id: 'all', name: 'All Capabilities', icon: Zap },
    { id: 'vision', name: 'Computer Vision', icon: Image },
    { id: 'nlp', name: 'Natural Language', icon: FileText },
    { id: 'code', name: 'Code Analysis', icon: Code },
    { id: 'web', name: 'Web Interaction', icon: Globe },
    { id: 'system', name: 'System Tools', icon: Terminal },
    { id: 'security', name: 'Security', icon: Shield },
    { id: 'data', name: 'Data Processing', icon: Database },
  ];

  const capabilities = [
    {
      id: 1,
      name: 'GPT-4 Vision Analysis',
      description: 'Advanced image understanding and visual content analysis for target systems.',
      category: 'vision',
      provider: 'OpenAI MCP Server',
      rating: 4.9,
      downloads: 15420,
      price: 'Free',
      tags: ['Image Analysis', 'OCR', 'Scene Understanding', 'Visual AI'],
      status: 'installed',
      mcpServer: 'openai-vision',
      targetCompatibility: ['Browser', 'File System', 'Applications'],
      processingTime: '2-5 seconds',
      useCase: 'Analyze screenshots, images, and visual content from target systems'
    },
    {
      id: 2,
      name: 'Claude Text Analysis',
      description: 'Advanced natural language processing and understanding for text-based targets.',
      category: 'nlp',
      provider: 'Anthropic MCP Server',
      rating: 4.8,
      downloads: 12300,
      price: 'Free',
      tags: ['NLP', 'Text Analysis', 'Sentiment', 'Language Understanding'],
      status: 'available',
      mcpServer: 'anthropic-claude',
      targetCompatibility: ['Browser', 'File System', 'Applications', 'Terminal'],
      processingTime: '1-3 seconds',
      useCase: 'Process and understand text content from documents, web pages, and applications'
    },
    {
      id: 3,
      name: 'Web Scraping Toolkit',
      description: 'Comprehensive web data extraction and browser automation capabilities.',
      category: 'web',
      provider: 'Browser MCP Server',
      rating: 4.7,
      downloads: 9800,
      price: '$0.01/request',
      tags: ['Web Scraping', 'DOM Analysis', 'Data Extraction', 'Browser Automation'],
      status: 'installed',
      mcpServer: 'browser-automation',
      targetCompatibility: ['Browser'],
      processingTime: '3-10 seconds',
      useCase: 'Extract structured data from websites and web applications'
    },
    {
      id: 4,
      name: 'File System Analyzer',
      description: 'Deep file and directory analysis with metadata extraction and content processing.',
      category: 'system',
      provider: 'FileSystem MCP Server',
      rating: 4.6,
      downloads: 8900,
      price: 'Free',
      tags: ['File Analysis', 'Metadata', 'Content Extraction', 'Directory Scanning'],
      status: 'available',
      mcpServer: 'filesystem-tools',
      targetCompatibility: ['File System'],
      processingTime: '1-5 seconds',
      useCase: 'Analyze files, extract metadata, and process directory structures'
    },
    {
      id: 5,
      name: 'Code Quality Inspector',
      description: 'Comprehensive code analysis, quality assessment, and vulnerability detection.',
      category: 'code',
      provider: 'CodeAnalysis MCP Server',
      rating: 4.8,
      downloads: 11200,
      price: 'Free',
      tags: ['Code Analysis', 'Quality Assessment', 'Bug Detection', 'Security Scan'],
      status: 'installed',
      mcpServer: 'code-inspector',
      targetCompatibility: ['Applications', 'File System'],
      processingTime: '5-15 seconds',
      useCase: 'Analyze code quality, detect bugs, and identify security vulnerabilities'
    },
    {
      id: 6,
      name: 'System Monitor Pro',
      description: 'Real-time system monitoring, process analysis, and performance tracking.',
      category: 'system',
      provider: 'SystemTools MCP Server',
      rating: 4.9,
      downloads: 7600,
      price: '$0.02/minute',
      tags: ['System Monitoring', 'Process Analysis', 'Performance', 'Resource Tracking'],
      status: 'available',
      mcpServer: 'system-monitor',
      targetCompatibility: ['System', 'Terminal'],
      processingTime: 'Real-time',
      useCase: 'Monitor system performance, track processes, and analyze resource usage'
    },
    {
      id: 7,
      name: 'Security Threat Scanner',
      description: 'Advanced security analysis and threat detection for target systems.',
      category: 'security',
      provider: 'SecurityTools MCP Server',
      rating: 4.9,
      downloads: 6800,
      price: '$0.05/scan',
      tags: ['Security Analysis', 'Threat Detection', 'Vulnerability Scan', 'Risk Assessment'],
      status: 'available',
      mcpServer: 'security-scanner',
      targetCompatibility: ['System', 'Network', 'Applications'],
      processingTime: '10-30 seconds',
      useCase: 'Scan for security threats, vulnerabilities, and potential risks'
    },
    {
      id: 8,
      name: 'Database Query Engine',
      description: 'Intelligent database interaction and query optimization for data analysis.',
      category: 'data',
      provider: 'Database MCP Server',
      rating: 4.7,
      downloads: 5400,
      price: 'Free',
      tags: ['Database', 'SQL', 'Query Optimization', 'Data Analysis'],
      status: 'installed',
      mcpServer: 'database-tools',
      targetCompatibility: ['Database', 'Applications'],
      processingTime: '2-8 seconds',
      useCase: 'Execute optimized database queries and analyze data structures'
    }
  ];

  const getCategoryIcon = (categoryId) => {
    const category = categories.find(cat => cat.id === categoryId);
    return category ? category.icon : Zap;
  };

  // Handle server installation and transformation to capability
  const handleServerInstalled = (capability) => {
    console.log('Server installed and transformed to capability:', capability);
    setInstalledCapabilities(prev => [...prev, capability]);
  };

  // Navigate to capabilities tab
  const handleNavigateToCapabilities = () => {
    console.log('Navigating to capabilities tab...');
    setActiveTab('capabilities');
  };

  const filteredCapabilities = capabilities.filter(capability => {
    const matchesSearch = capability.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         capability.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         capability.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase()));
    const matchesCategory = selectedCategory === 'all' || capability.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">MCP Capability Store</h1>
          <p className="text-slate-400">Discover and install Model Context Protocol capabilities and servers for your agents.</p>
        </div>
        <div className="mt-4 lg:mt-0 flex items-center space-x-3">
          <button
            onClick={() => setActiveTab('mcp-servers')}
            className="bg-slate-800/50 border border-slate-700/50 text-slate-300 px-4 py-2 rounded-lg hover:text-white hover:border-slate-600/50 transition-all duration-200 flex items-center space-x-2"
          >
            <Settings className="w-4 h-4" />
            <span>Browse Servers</span>
          </button>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="flex space-x-1 bg-slate-800/50 p-1 rounded-lg border border-slate-700/50">
        <button
          onClick={() => setActiveTab('capabilities')}
          className={`flex-1 flex items-center justify-center space-x-2 px-4 py-2 rounded-md transition-all duration-200 ${
            activeTab === 'capabilities'
              ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 text-white border border-purple-500/30'
              : 'text-slate-400 hover:text-white'
          }`}
        >
          <Layers className="w-4 h-4" />
          <span>Capabilities</span>
        </button>
        <button
          onClick={() => setActiveTab('mcp-servers')}
          className={`flex-1 flex items-center justify-center space-x-2 px-4 py-2 rounded-md transition-all duration-200 ${
            activeTab === 'mcp-servers'
              ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 text-white border border-purple-500/30'
              : 'text-slate-400 hover:text-white'
          }`}
        >
          <Server className="w-4 h-4" />
          <span>MCP Servers</span>
        </button>
      </div>

      {/* Conditional Content */}
      {activeTab === 'mcp-servers' ? (
        <MCPServerBrowser
          onServerInstalled={handleServerInstalled}
          onNavigateToCapabilities={handleNavigateToCapabilities}
        />
      ) : (
        <>
          {/* Search and Filter */}
          <div className="flex flex-col lg:flex-row lg:items-center space-y-4 lg:space-y-0 lg:space-x-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
              <input
                type="text"
                placeholder="Search MCP capabilities, providers, or use cases..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full bg-slate-800/50 border border-slate-700/50 rounded-lg pl-10 pr-4 py-3 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
              />
            </div>
            <button className="bg-slate-800/50 border border-slate-700/50 rounded-lg px-4 py-3 text-slate-300 hover:text-white hover:border-slate-600/50 transition-all duration-200 flex items-center space-x-2">
              <Filter className="w-5 h-5" />
              <span>Filter</span>
            </button>
          </div>

          {/* Categories */}
          <div className="flex flex-wrap gap-3">
            {categories.map((category) => {
              const Icon = category.icon;
              return (
                <button
                  key={category.id}
                  onClick={() => setSelectedCategory(category.id)}
                  className={`flex items-center space-x-2 px-4 py-2 rounded-lg transition-all duration-200 ${
                    selectedCategory === category.id
                      ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 text-white border border-purple-500/30'
                      : 'bg-slate-800/50 text-slate-300 border border-slate-700/50 hover:text-white hover:border-slate-600/50'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                  <span className="font-medium">{category.name}</span>
                </button>
              );
            })}
          </div>

          {/* Capabilities Grid */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {filteredCapabilities.map((capability) => {
              const CategoryIcon = getCategoryIcon(capability.category);
              return (
                <div key={capability.id} className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50 hover:border-slate-600/50 transition-all duration-300 group">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center space-x-3">
                      <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
                        <CategoryIcon className="w-5 h-5 text-purple-400" />
                      </div>
                      <div>
                        <h3 className="text-lg font-bold text-white">{capability.name}</h3>
                        <p className="text-slate-400 text-sm">{capability.provider}</p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-1">
                      <Star className="w-4 h-4 text-yellow-400 fill-current" />
                      <span className="text-white text-sm font-medium">{capability.rating}</span>
                    </div>
                  </div>

                  <p className="text-slate-300 text-sm mb-4 line-clamp-2">{capability.description}</p>

                  <div className="space-y-3 mb-4">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">Downloads:</span>
                      <span className="text-white">{capability.downloads.toLocaleString()}</span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">Price:</span>
                      <span className="text-white font-medium">{capability.price}</span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">Processing Time:</span>
                      <span className="text-white">{capability.processingTime}</span>
                    </div>
                  </div>

                  <div className="space-y-3 mb-6">
                    <div>
                      <p className="text-slate-400 text-sm mb-2">Use Case:</p>
                      <p className="text-slate-300 text-xs">{capability.useCase}</p>
                    </div>

                    <div>
                      <p className="text-slate-400 text-sm mb-2">Target Compatibility:</p>
                      <div className="flex flex-wrap gap-1">
                        {capability.targetCompatibility.map((target, index) => (
                          <span key={index} className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30">
                            {target}
                          </span>
                        ))}
                      </div>
                    </div>

                    <div>
                      <p className="text-slate-400 text-sm mb-2">Tags:</p>
                      <div className="flex flex-wrap gap-1">
                        {capability.tags.slice(0, 3).map((tag, index) => (
                          <span key={index} className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full">
                            {tag}
                          </span>
                        ))}
                        {capability.tags.length > 3 && (
                          <span className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full">
                            +{capability.tags.length - 3} more
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center space-x-3">
                    {capability.status === 'installed' ? (
                      <>
                        <div className="flex-1 bg-green-500/20 border border-green-500/30 text-green-400 py-2 px-4 rounded-lg text-center font-medium">
                          Installed
                        </div>
                        <button className="bg-slate-700/50 border border-slate-600/50 text-slate-300 hover:text-white hover:border-slate-500/50 p-2 rounded-lg transition-all duration-200">
                          <Settings className="w-4 h-4" />
                        </button>
                      </>
                    ) : (
                      <button className="flex-1 bg-gradient-to-r from-purple-500/20 to-blue-500/20 hover:from-purple-500/30 hover:to-blue-500/30 border border-purple-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2">
                        <Download className="w-4 h-4" />
                        <span>Install</span>
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </>
      )}
    </div>
  );
};