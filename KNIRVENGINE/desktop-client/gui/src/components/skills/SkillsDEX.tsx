import React, { useState, useEffect } from 'react';
import { Zap, Download, Upload, Star, Search, Filter, Plus, Code, TrendingUp, Package } from 'lucide-react';
import { useAuth } from '../AuthContext';

interface Skill {
  id: string;
  name: string;
  description: string;
  category: 'automation' | 'analysis' | 'communication' | 'development' | 'utility';
  version: string;
  author: string;
  rating: number;
  downloads: number;
  price: number; // 0 for free
  tags: string[];
  isInstalled: boolean;
  isOwned: boolean;
  createdAt: Date;
  updatedAt: Date;
  size: string;
  dependencies: string[];
  executionCost: number;
  complexity: 'low' | 'medium' | 'high';
  requirements: string[];
}

interface SkillStats {
  totalSkills: number;
  installedSkills: number;
  ownedSkills: number;
  avgRating: number;
}

const SkillsDEX: React.FC = () => {
  const { user } = useAuth();
  const [skills, setSkills] = useState<Skill[]>([]);
  const [stats, setStats] = useState<SkillStats>({
    totalSkills: 0,
    installedSkills: 0,
    ownedSkills: 0,
    avgRating: 0
  });
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [sortBy, setSortBy] = useState<'rating' | 'downloads' | 'recent'>('rating');

  useEffect(() => {
    // Load skills data from KNIRV Skills DEX
    // This would connect to the actual Skills DEX marketplace
    const loadSkills = async () => {
      try {
        // TODO: Replace with actual API call to Skills DEX
        // const response = await fetch('/api/skills-dex/skills');
        // const skillsData = await response.json();
        
        // Mock data for now - representing skills from the decentralized marketplace
        const mockSkills: Skill[] = [
          {
            id: 'skill-code-gen-pro',
            name: 'Code Generator Pro',
            description: 'Advanced code generation with multiple language support and optimization features',
            category: 'development',
            version: '2.1.0',
            author: 'DevTools Inc.',
            rating: 4.8,
            downloads: 15420,
            price: 0,
            tags: ['code', 'generation', 'optimization', 'multi-language'],
            isInstalled: true,
            isOwned: true,
            createdAt: new Date(Date.now() - 86400000 * 30),
            updatedAt: new Date(Date.now() - 86400000 * 5),
            size: '45 MB',
            dependencies: ['python-runtime', 'node-runtime'],
            executionCost: 0.02,
            complexity: 'medium',
            requirements: ['Python 3.8+', 'Node.js 16+']
          },
          {
            id: 'skill-data-analyzer',
            name: 'Data Analyzer Suite',
            description: 'Comprehensive data analysis toolkit with visualization and reporting capabilities',
            category: 'analysis',
            version: '1.8.3',
            author: 'DataCorp',
            rating: 4.6,
            downloads: 8930,
            price: 29.99,
            tags: ['data', 'analysis', 'visualization', 'reports'],
            isInstalled: false,
            isOwned: false,
            createdAt: new Date(Date.now() - 86400000 * 45),
            updatedAt: new Date(Date.now() - 86400000 * 12),
            size: '128 MB',
            dependencies: ['pandas', 'matplotlib'],
            executionCost: 0.05,
            complexity: 'high',
            requirements: ['Python 3.9+', 'NumPy', 'Pandas']
          },
          {
            id: 'skill-smart-automation',
            name: 'Smart Automation Engine',
            description: 'Intelligent workflow automation with machine learning optimization',
            category: 'automation',
            version: '3.0.1',
            author: 'AutoFlow Systems',
            rating: 4.9,
            downloads: 12750,
            price: 49.99,
            tags: ['automation', 'workflow', 'ml', 'optimization'],
            isInstalled: true,
            isOwned: true,
            createdAt: new Date(Date.now() - 86400000 * 60),
            updatedAt: new Date(Date.now() - 86400000 * 8),
            size: '89 MB',
            dependencies: ['tensorflow', 'scikit-learn'],
            executionCost: 0.08,
            complexity: 'high',
            requirements: ['TensorFlow 2.0+', 'Scikit-learn']
          },
          {
            id: 'skill-comm-hub',
            name: 'Communication Hub',
            description: 'Multi-platform communication integration with smart routing and filtering',
            category: 'communication',
            version: '1.5.2',
            author: 'CommTech',
            rating: 4.3,
            downloads: 6420,
            price: 19.99,
            tags: ['communication', 'integration', 'routing', 'filtering'],
            isInstalled: false,
            isOwned: false,
            createdAt: new Date(Date.now() - 86400000 * 20),
            updatedAt: new Date(Date.now() - 86400000 * 3),
            size: '67 MB',
            dependencies: ['requests', 'websockets'],
            executionCost: 0.03,
            complexity: 'medium',
            requirements: ['Python 3.7+', 'WebSocket support']
          },
          {
            id: 'skill-system-monitor',
            name: 'System Monitor Plus',
            description: 'Advanced system monitoring with predictive analytics and alerting',
            category: 'utility',
            version: '2.3.0',
            author: 'SysTools',
            rating: 4.7,
            downloads: 11200,
            price: 0,
            tags: ['monitoring', 'analytics', 'alerts', 'system'],
            isInstalled: true,
            isOwned: true,
            createdAt: new Date(Date.now() - 86400000 * 35),
            updatedAt: new Date(Date.now() - 86400000 * 7),
            size: '34 MB',
            dependencies: ['psutil', 'prometheus'],
            executionCost: 0.01,
            complexity: 'low',
            requirements: ['Python 3.6+', 'psutil']
          }
        ];

        setSkills(mockSkills);
        
        const installedSkills = mockSkills.filter(s => s.isInstalled).length;
        const ownedSkills = mockSkills.filter(s => s.isOwned).length;
        const avgRating = mockSkills.reduce((sum, s) => sum + s.rating, 0) / mockSkills.length;
        
        setStats({
          totalSkills: mockSkills.length,
          installedSkills,
          ownedSkills,
          avgRating: Math.round(avgRating * 10) / 10
        });
      } catch (error) {
        console.error('Failed to load skills from DEX:', error);
      }
    };

    loadSkills();
  }, []);

  const getCategoryColor = (category: string) => {
    switch (category) {
      case 'development':
        return 'text-blue-400 bg-blue-500/20';
      case 'analysis':
        return 'text-purple-400 bg-purple-500/20';
      case 'automation':
        return 'text-green-400 bg-green-500/20';
      case 'communication':
        return 'text-yellow-400 bg-yellow-500/20';
      case 'utility':
        return 'text-slate-400 bg-slate-500/20';
      default:
        return 'text-slate-400 bg-slate-500/20';
    }
  };

  const getComplexityColor = (complexity: string) => {
    switch (complexity) {
      case 'low':
        return 'text-green-400';
      case 'medium':
        return 'text-yellow-400';
      case 'high':
        return 'text-red-400';
      default:
        return 'text-slate-400';
    }
  };

  const renderStars = (rating: number) => {
    return Array.from({ length: 5 }, (_, i) => (
      <Star
        key={i}
        className={`w-3 h-3 ${
          i < Math.floor(rating) ? 'text-yellow-400 fill-current' : 'text-slate-600'
        }`}
      />
    ));
  };

  const handleInstall = async (skillId: string) => {
    try {
      // TODO: Implement actual skill installation via KNIRV network
      // await skillsService.installSkill(skillId);
      
      setSkills(prev => prev.map(skill => 
        skill.id === skillId ? { ...skill, isInstalled: true } : skill
      ));
    } catch (error) {
      console.error('Failed to install skill:', error);
    }
  };

  const handleUninstall = async (skillId: string) => {
    try {
      // TODO: Implement actual skill uninstallation
      // await skillsService.uninstallSkill(skillId);
      
      setSkills(prev => prev.map(skill => 
        skill.id === skillId ? { ...skill, isInstalled: false } : skill
      ));
    } catch (error) {
      console.error('Failed to uninstall skill:', error);
    }
  };

  const handlePurchase = async (skillId: string) => {
    try {
      // TODO: Implement actual skill purchase via KNIRV network
      // await skillsService.purchaseSkill(skillId);
      
      setSkills(prev => prev.map(skill => 
        skill.id === skillId ? { ...skill, isOwned: true, isInstalled: true } : skill
      ));
    } catch (error) {
      console.error('Failed to purchase skill:', error);
    }
  };

  const filteredSkills = skills
    .filter(skill => {
      const matchesCategory = selectedCategory === 'all' || skill.category === selectedCategory;
      const matchesSearch = skill.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                           skill.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
                           skill.tags.some(tag => tag.toLowerCase().includes(searchQuery.toLowerCase()));
      return matchesCategory && matchesSearch;
    })
    .sort((a, b) => {
      switch (sortBy) {
        case 'rating':
          return b.rating - a.rating;
        case 'downloads':
          return b.downloads - a.downloads;
        case 'recent':
          return b.updatedAt.getTime() - a.updatedAt.getTime();
        default:
          return 0;
      }
    });

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-yellow-500/20 rounded-lg">
            <Zap className="w-6 h-6 text-yellow-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">Skills DEX</h1>
            <p className="text-slate-400">Decentralized AI Skills Marketplace</p>
          </div>
        </div>
        
        <button className="flex items-center space-x-2 bg-yellow-600 text-white px-4 py-2 rounded-lg hover:bg-yellow-700 transition-colors">
          <Plus className="w-4 h-4" />
          <span>Publish Skill</span>
        </button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Package className="w-5 h-5 text-yellow-400" />
            <span className="text-sm text-slate-400">Total Skills</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.totalSkills}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Download className="w-5 h-5 text-green-400" />
            <span className="text-sm text-slate-400">Installed</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.installedSkills}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Code className="w-5 h-5 text-blue-400" />
            <span className="text-sm text-slate-400">Owned</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.ownedSkills}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Star className="w-5 h-5 text-yellow-400" />
            <span className="text-sm text-slate-400">Avg Rating</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.avgRating}</div>
        </div>
      </div>

      {/* Filters and Search */}
      <div className="flex flex-wrap items-center gap-4 mb-6">
        <div className="flex-1 min-w-64">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Search skills..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-slate-800/50 border border-slate-700/50 rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-yellow-500/50"
            />
          </div>
        </div>
        
        <select
          value={selectedCategory}
          onChange={(e) => setSelectedCategory(e.target.value)}
          className="bg-slate-800/50 border border-slate-700/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-yellow-500/50"
        >
          <option value="all">All Categories</option>
          <option value="development">Development</option>
          <option value="analysis">Analysis</option>
          <option value="automation">Automation</option>
          <option value="communication">Communication</option>
          <option value="utility">Utility</option>
        </select>
        
        <select
          value={sortBy}
          onChange={(e) => setSortBy(e.target.value as any)}
          className="bg-slate-800/50 border border-slate-700/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-yellow-500/50"
        >
          <option value="rating">Sort by Rating</option>
          <option value="downloads">Sort by Downloads</option>
          <option value="recent">Sort by Recent</option>
        </select>
      </div>

      {/* Skills Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
        {filteredSkills.map((skill) => (
          <div key={skill.id} className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors">
            <div className="flex items-start justify-between mb-4">
              <div className="flex-1">
                <div className="flex items-center space-x-2 mb-2">
                  <h3 className="text-lg font-semibold text-white">{skill.name}</h3>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${getCategoryColor(skill.category)}`}>
                    {skill.category}
                  </span>
                </div>
                <p className="text-slate-400 text-sm mb-3">{skill.description}</p>
                
                <div className="flex items-center space-x-4 text-sm mb-3">
                  <div className="flex items-center space-x-1">
                    {renderStars(skill.rating)}
                    <span className="text-slate-300 ml-1">{skill.rating}</span>
                  </div>
                  <div className="text-slate-400">
                    {skill.downloads.toLocaleString()} downloads
                  </div>
                  <div className={`${getComplexityColor(skill.complexity)}`}>
                    {skill.complexity} complexity
                  </div>
                </div>
                
                <div className="flex flex-wrap gap-1 mb-3">
                  {skill.tags.slice(0, 3).map((tag) => (
                    <span key={tag} className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded">
                      {tag}
                    </span>
                  ))}
                </div>
                
                <div className="text-xs text-slate-500 mb-2">
                  v{skill.version} • {skill.size} • by {skill.author}
                </div>
                
                <div className="text-xs text-slate-500">
                  Execution cost: {skill.executionCost} NRN per run
                </div>
              </div>
              
              <div className="text-right">
                {skill.price > 0 ? (
                  <div className="text-lg font-bold text-yellow-400">${skill.price}</div>
                ) : (
                  <div className="text-lg font-bold text-green-400">Free</div>
                )}
              </div>
            </div>
            
            <div className="flex space-x-2">
              {skill.isInstalled ? (
                <button
                  onClick={() => handleUninstall(skill.id)}
                  className="flex-1 bg-red-600/20 text-red-400 px-3 py-2 rounded text-sm font-medium hover:bg-red-600/30 transition-colors"
                >
                  Uninstall
                </button>
              ) : skill.isOwned ? (
                <button
                  onClick={() => handleInstall(skill.id)}
                  className="flex-1 bg-green-600/20 text-green-400 px-3 py-2 rounded text-sm font-medium hover:bg-green-600/30 transition-colors"
                >
                  Install
                </button>
              ) : (
                <button
                  onClick={() => handlePurchase(skill.id)}
                  className="flex-1 bg-yellow-600/20 text-yellow-400 px-3 py-2 rounded text-sm font-medium hover:bg-yellow-600/30 transition-colors"
                >
                  Buy & Install
                </button>
              )}
              <button className="bg-slate-600/50 text-slate-300 px-3 py-2 rounded text-sm font-medium hover:bg-slate-600 transition-colors">
                Details
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default SkillsDEX;
