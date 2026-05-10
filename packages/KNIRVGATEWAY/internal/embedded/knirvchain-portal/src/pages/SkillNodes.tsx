import React, { useEffect, useState } from 'react';
import { graphChainApi, SkillNode } from '../services/api';
import { useGraphChain } from '../context/GraphChainContext';
import { Brain, ChevronLeft, ChevronRight } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';
import SkillNodeCard from '../components/SkillNodeCard';

const SkillNodes: React.FC = () => {
  const { currentDensity, isLoading } = useGraphChain();
  const [skills, setSkills] = useState<SkillNode[]>([]);
  const [filteredSkills, setFilteredSkills] = useState<SkillNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<string>('all');
  const [searchTerm, setSearchTerm] = useState<string>('');

  const skillsPerPage = 10;

  useEffect(() => {
    const fetchSkills = async () => {
      setLoading(true);
      setError(null);

      try {
        const fetchedSkills = await graphChainApi.getAllSkills();
        setSkills(fetchedSkills);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch SkillNodes');
      } finally {
        setLoading(false);
      }
    };

    fetchSkills();
  }, [currentDensity]);

  // Filter and search skills
  useEffect(() => {
    let filtered = skills;

    // Apply filter
    if (filter !== 'all') {
      filtered = filtered.filter(skill => {
        switch (filter) {
          case 'validated':
            return skill.validation?.is_validated === true;
          case 'unvalidated':
            return !skill.validation?.is_validated;
          case 'high-performance':
            return skill.performance && skill.performance.success_rate > 0.8;
          default:
            return true;
        }
      });
    }

    // Apply search
    if (searchTerm) {
      filtered = filtered.filter(skill =>
        skill.skill_type.toLowerCase().includes(searchTerm.toLowerCase()) ||
        skill.capabilities.some(cap => cap.toLowerCase().includes(searchTerm.toLowerCase()))
      );
    }

    setFilteredSkills(filtered);
  }, [skills, filter, searchTerm]);

  const totalPages = Math.ceil(filteredSkills.length / skillsPerPage);
  const paginatedSkills = filteredSkills.slice(
    (page - 1) * skillsPerPage,
    page * skillsPerPage
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <LoadingSpinner size="large" />
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-8">
        <Brain className="w-8 h-8 text-blue-400" />
        <div>
          <h1 className="text-3xl font-bold text-white">SkillNodes</h1>
          <p className="text-gray-400">Browse all SkillNodes in the GraphChain</p>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 mb-8">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">Search SkillNodes</label>
            <input
              type="text"
              placeholder="Search by skill type or capabilities..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full px-4 py-2 bg-gray-700/50 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">Filter</label>
            <select
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="w-full px-4 py-2 bg-gray-700/50 border border-gray-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All SkillNodes</option>
              <option value="validated">Validated Only</option>
              <option value="unvalidated">Unvalidated Only</option>
              <option value="high-performance">High Performance</option>
            </select>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="text-center">
            <div className="text-3xl font-bold text-blue-400">{skills.length}</div>
            <div className="text-gray-400">Total SkillNodes</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-green-400">{filteredSkills.length}</div>
            <div className="text-gray-400">Filtered Results</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-purple-400">{page}</div>
            <div className="text-gray-400">Current Page</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-orange-400">{totalPages}</div>
            <div className="text-gray-400">Total Pages</div>
          </div>
        </div>
      </div>

      {/* SkillNodes List */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <LoadingSpinner size="large" />
        </div>
      ) : error ? (
        <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-6 text-center">
          <div className="text-red-400 text-lg font-medium mb-2">Error Loading SkillNodes</div>
          <div className="text-gray-400">{error}</div>
        </div>
      ) : paginatedSkills.length === 0 ? (
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-12 border border-gray-700/50 text-center">
          <Brain className="w-16 h-16 mx-auto mb-4 text-gray-500" />
          <h3 className="text-xl font-semibold text-gray-400 mb-2">No SkillNodes Found</h3>
          <p className="text-gray-500">Try adjusting your search or filter criteria.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {paginatedSkills.map((skill) => (
            <SkillNodeCard key={skill.id} skill={skill} />
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center space-x-4 mt-8">
          <button
            onClick={() => setPage(Math.max(1, page - 1))}
            disabled={page === 1}
            className="flex items-center space-x-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 disabled:bg-gray-800 disabled:text-gray-500 text-white rounded-lg transition-colors"
          >
            <ChevronLeft className="w-4 h-4" />
            <span>Previous</span>
          </button>
          
          <div className="flex items-center space-x-2">
            {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
              const pageNum = Math.max(1, Math.min(totalPages, page - 2 + i));
              return (
                <button
                  key={pageNum}
                  onClick={() => setPage(pageNum)}
                  className={`px-3 py-2 rounded-lg transition-colors ${
                    page === pageNum
                      ? 'bg-blue-500 text-white'
                      : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
                  }`}
                >
                  {pageNum}
                </button>
              );
            })}
          </div>
          
          <button
            onClick={() => setPage(Math.min(totalPages, page + 1))}
            disabled={page === totalPages}
            className="flex items-center space-x-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 disabled:bg-gray-800 disabled:text-gray-500 text-white rounded-lg transition-colors"
          >
            <span>Next</span>
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      )}
    </div>
  );
};

export default SkillNodes;