import React, { useState, useEffect } from 'react';
import { Search, Zap, Download, Activity } from 'lucide-react';
import SkillCard from '../SkillCard';

// Temporary type definitions
interface LoRAAdapterData {
  id: string;
  name: string;
  networkScore: number;
  description: string;
  version: number;
  adapterId: string;
  adapterName: string;
  usageCount?: number;
}

interface LoRASkill {
  id: string;
  name: string;
  description: string;
  category: 'analysis' | 'automation' | 'computation' | 'communication';
  complexity: number;
  nrnCost: number;
  isActive: boolean;
  adapterId?: string;
  adapterData?: LoRAAdapterData;
  networkScore?: number;
  usageCount?: number;
}

interface SkillsModalContentProps {
  nrnBalance?: number;
}

export const SkillsModalContent: React.FC<SkillsModalContentProps> = ({ nrnBalance = 1250 }) => {
  const [loraAdapters] = useState<LoRAAdapterData[]>([]);
  const [skills, setSkills] = useState<LoRASkill[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');

  // Initialize with default skills
  useEffect(() => {
    const getDefaultSkills = (): LoRASkill[] => [
      {
        id: 'default-code-analysis',
        name: 'Code Analysis',
        description: 'Automated code review and optimization using advanced pattern recognition',
        category: 'analysis' as const,
        complexity: 8,
        nrnCost: 25,
        isActive: true
      },
      {
        id: 'default-task-orchestration',
        name: 'Task Orchestration',
        description: 'Intelligent workflow automation across multiple systems and platforms',
        category: 'automation' as const,
        complexity: 7,
        nrnCost: 30,
        isActive: true
      },
      {
        id: 'default-neural-synthesis',
        name: 'Neural Synthesis',
        description: 'Advanced data processing and pattern synthesis for complex computations',
        category: 'computation' as const,
        complexity: 9,
        nrnCost: 45,
        isActive: false
      },
      {
        id: 'default-agent-communication',
        name: 'Agent Communication',
        description: 'Secure inter-agent messaging and coordination protocols',
        category: 'communication' as const,
        complexity: 6,
        nrnCost: 20,
        isActive: true
      },
      {
        id: 'default-predictive-modeling',
        name: 'Predictive Modeling',
        description: 'Real-time prediction and forecasting using machine learning algorithms',
        category: 'analysis' as const,
        complexity: 8,
        nrnCost: 35,
        isActive: false
      },
      {
        id: 'default-resource-optimization',
        name: 'Resource Optimization',
        description: 'Dynamic resource allocation and performance tuning for optimal efficiency',
        category: 'automation' as const,
        complexity: 7,
        nrnCost: 28,
        isActive: false
      }
    ];

    setSkills(getDefaultSkills());
    setIsLoading(false);
  }, []);

  // Filter skills based on search and category
  const filteredSkills = skills.filter(skill => {
    const matchesSearch = skill.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         skill.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || skill.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const activeSkills = skills.filter(skill => skill.isActive).length;
  const totalNrnCost = skills.filter(skill => skill.isActive).reduce((sum, skill) => sum + skill.nrnCost, 0);

  return (
    <div className="space-y-4">
      {/* Stats */}
      <div className="grid grid-cols-3 gap-3">
        <div className="text-center p-3 bg-gray-800/80 border border-gray-600/50 rounded-lg">
          <div className="text-xl font-bold text-white">{activeSkills}</div>
          <div className="text-xs text-gray-400">Active Skills</div>
        </div>
        <div className="text-center p-3 bg-gray-800/80 border border-gray-600/50 rounded-lg">
          <div className="text-xl font-bold text-blue-400">{totalNrnCost}</div>
          <div className="text-xs text-gray-400">NRN/Hour</div>
        </div>
        <div className="text-center p-3 bg-gray-800/80 border border-gray-600/50 rounded-lg">
          <div className="text-xl font-bold text-cyan-400">{skills.length}</div>
          <div className="text-xs text-gray-400">Available</div>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="space-y-2">
        <div className="flex space-x-2">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Search LoRA adapters and skills..."
              className="w-full pl-10 pr-3 py-2.5 bg-gray-800/80 border border-gray-600/50 rounded-lg focus:border-blue-500/50 focus:outline-none text-white placeholder-gray-400 text-sm"
            />
          </div>
          <select
            value={selectedCategory}
            onChange={(e) => setSelectedCategory(e.target.value)}
            className="px-3 py-2.5 bg-gray-800/80 border border-gray-600/50 rounded-lg focus:border-blue-500/50 focus:outline-none text-white text-sm"
          >
            <option value="all">All Categories</option>
            <option value="analysis">Analysis</option>
            <option value="automation">Automation</option>
            <option value="computation">Computation</option>
            <option value="communication">Communication</option>
          </select>
          <button
            disabled
            className="px-3 py-2.5 bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 rounded-lg text-blue-400 hover:text-blue-300 transition-all disabled:opacity-50"
            title="Refresh from network"
          >
            <Download className="w-4 h-4" />
          </button>
        </div>

        {/* LoRA Adapter Status */}
        <div className="flex items-center justify-between p-2.5 bg-gray-800/50 rounded-lg border border-gray-600/30">
          <div className="flex items-center space-x-2">
            <Activity className="w-3.5 h-3.5 text-purple-400" />
            <span className="text-xs text-white">LoRA Network Status</span>
          </div>
          <div className="flex items-center space-x-3 text-xs text-gray-400">
            <span>Adapters: {loraAdapters.length}</span>
            <span>Active: {activeSkills}</span>
            <span>Network Score: {loraAdapters.length > 0 ? (loraAdapters.reduce((sum, a) => sum + a.networkScore, 0) / loraAdapters.length).toFixed(2) : '0.00'}</span>
          </div>
        </div>
      </div>

      {/* Skills Grid */}
      <div className="space-y-3">
        {isLoading ? (
          <div className="text-center py-6">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-400 mx-auto mb-3"></div>
            <p className="text-sm text-gray-400">Loading available skills...</p>
          </div>
        ) : filteredSkills.length === 0 ? (
          <div className="text-center py-6">
            <Zap className="w-10 h-10 mx-auto mb-3 text-gray-500 opacity-50" />
            <p className="text-gray-400 mb-2 text-sm">No skills found</p>
            <p className="text-xs text-gray-500">
              {searchTerm || selectedCategory !== 'all'
                ? 'Try adjusting your search or filter criteria'
                : 'No skills available. Check back later for updates.'
              }
            </p>
          </div>
        ) : (
          filteredSkills.map((skill) => (
            <div key={skill.id} className="relative">
              <SkillCard
                name={skill.name}
                description={skill.description}
                category={skill.category}
                complexity={skill.complexity}
                nrnCost={skill.nrnCost}
                isActive={skill.isActive}
              />
              {skill.adapterData && (
                <div className="absolute top-2 right-2 flex space-x-1">
                  {skill.adapterData.networkScore > 0.8 && (
                    <div className="px-1.5 py-0.5 bg-green-500/20 text-green-400 text-xs rounded">
                      High Score
                    </div>
                  )}
                  <div className="px-1.5 py-0.5 bg-purple-500/20 text-purple-400 text-xs rounded">
                    LoRA
                  </div>
                </div>
              )}
            </div>
          ))
        )}
      </div>

      {/* Install New Skills */}
      <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-4 text-center">
        <h3 className="text-base font-semibold text-white mb-2">Discover New Skills</h3>
        <p className="text-xs text-gray-400 mb-3">
          Browse KNIRV marketplace for cutting-edge AI capabilities
        </p>
        <button className="px-5 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-white font-medium transition-all text-sm">
          Browse Marketplace
        </button>
      </div>
    </div>
  );
};