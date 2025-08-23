import { Search, Filter, Plus } from 'lucide-react';
import Layout from './components/Layout';
import SkillCard from './components/SkillCard';

export default function Skills() {
  const skills = [
    {
      name: 'Code Analysis',
      description: 'Automated code review and optimization using advanced pattern recognition',
      category: 'analysis' as const,
      complexity: 8,
      nrnCost: 25,
      isActive: true
    },
    {
      name: 'Task Orchestration',
      description: 'Intelligent workflow automation across multiple systems and platforms',
      category: 'automation' as const,
      complexity: 7,
      nrnCost: 30,
      isActive: true
    },
    {
      name: 'Neural Synthesis',
      description: 'Advanced data processing and pattern synthesis for complex computations',
      category: 'computation' as const,
      complexity: 9,
      nrnCost: 45,
      isActive: false
    },
    {
      name: 'Agent Communication',
      description: 'Secure inter-agent messaging and coordination protocols',
      category: 'communication' as const,
      complexity: 6,
      nrnCost: 20,
      isActive: true
    },
    {
      name: 'Predictive Modeling',
      description: 'Real-time prediction and forecasting using machine learning algorithms',
      category: 'analysis' as const,
      complexity: 8,
      nrnCost: 35,
      isActive: false
    },
    {
      name: 'Resource Optimization',
      description: 'Dynamic resource allocation and performance tuning for optimal efficiency',
      category: 'automation' as const,
      complexity: 7,
      nrnCost: 28,
      isActive: false
    }
  ];

  const activeSkills = skills.filter(skill => skill.isActive).length;
  const totalNrnCost = skills.filter(skill => skill.isActive).reduce((sum, skill) => sum + skill.nrnCost, 0);

  return (
    <Layout>
      <div className="p-4 pb-24 space-y-6">
        {/* Header */}
        <div className="text-center py-4">
          <h2 className="text-2xl font-bold bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent mb-2">
            Agent Skills
          </h2>
          <p className="text-slate-400 text-sm">
            Manage and configure your AI agent capabilities
          </p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4">
          <div className="text-center p-4 bg-slate-800/50 backdrop-blur-xl rounded-xl border border-slate-700/50">
            <div className="text-2xl font-bold text-white">{activeSkills}</div>
            <div className="text-xs text-slate-400">Active Skills</div>
          </div>
          <div className="text-center p-4 bg-slate-800/50 backdrop-blur-xl rounded-xl border border-slate-700/50">
            <div className="text-2xl font-bold text-blue-400">{totalNrnCost}</div>
            <div className="text-xs text-slate-400">NRN/Hour</div>
          </div>
          <div className="text-center p-4 bg-slate-800/50 backdrop-blur-xl rounded-xl border border-slate-700/50">
            <div className="text-2xl font-bold text-cyan-400">{skills.length}</div>
            <div className="text-xs text-slate-400">Available</div>
          </div>
        </div>

        {/* Search and Filter */}
        <div className="flex space-x-3">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Search skills..."
              className="w-full pl-10 pr-4 py-3 bg-slate-800/80 backdrop-blur-xl rounded-xl border border-slate-700/50 focus:border-blue-500/50 focus:outline-none text-white placeholder-slate-400"
            />
          </div>
          <button className="px-4 py-3 bg-slate-800/80 backdrop-blur-xl rounded-xl border border-slate-700/50 hover:border-blue-500/50 text-slate-400 hover:text-blue-400 transition-all">
            <Filter className="w-4 h-4" />
          </button>
          <button className="px-4 py-3 bg-gradient-to-r from-blue-600/20 to-cyan-600/20 hover:from-blue-600/30 hover:to-cyan-600/30 rounded-xl border border-blue-500/30 text-blue-400 hover:text-blue-300 transition-all">
            <Plus className="w-4 h-4" />
          </button>
        </div>

        {/* Skills Grid */}
        <div className="space-y-4">
          {skills.map((skill, index) => (
            <SkillCard key={index} {...skill} />
          ))}
        </div>

        {/* Install New Skills */}
        <div className="relative group">
          <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/30 to-cyan-600/30 rounded-xl blur opacity-20 group-hover:opacity-40 transition duration-300"></div>
          <div className="relative bg-slate-800/60 backdrop-blur-xl rounded-xl p-6 border border-slate-700/50 hover:border-blue-500/50 transition-all text-center">
            <h3 className="text-lg font-semibold text-white mb-2">Discover New Skills</h3>
            <p className="text-sm text-slate-400 mb-4">
              Browse the KNIRV marketplace for cutting-edge AI capabilities
            </p>
            <button className="px-6 py-2 bg-gradient-to-r from-blue-600 to-cyan-600 hover:from-blue-700 hover:to-cyan-700 rounded-lg text-white font-medium transition-all transform hover:scale-105">
              Browse Marketplace
            </button>
          </div>
        </div>
      </div>
    </Layout>
  );
}
