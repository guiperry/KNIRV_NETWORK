import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useGraphChain } from '../context/GraphChainContext';
import { graphChainApi, SkillNode, ErrorNode } from '../services/api';
import { Search as SearchIcon, Brain, AlertTriangle, Network, ArrowRight } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const Search: React.FC = () => {
  const { query } = useParams<{ query: string }>();
  const { currentHeight } = useGraphChain();
  const [loading, setLoading] = useState(true);
  const [results, setResults] = useState<{
    skills: SkillNode[];
    errors: ErrorNode[];
    type: 'skills' | 'errors' | 'mixed' | 'unknown';
  } | null>(null);

  useEffect(() => {
    const performSearch = async () => {
      if (!query) return;

      setLoading(true);
      setResults(null);

      try {
        const [allSkills, allErrors] = await Promise.all([
          graphChainApi.getAllSkills(),
          graphChainApi.getAllErrors(),
        ]);

        const searchTerm = query.toLowerCase();

        // Search in SkillNodes
        const matchingSkills = allSkills.filter(skill =>
          skill.skill_type.toLowerCase().includes(searchTerm) ||
          skill.capabilities.some(cap => cap.toLowerCase().includes(searchTerm)) ||
          skill.id.toLowerCase().includes(searchTerm)
        );

        // Search in ErrorNodes
        const matchingErrors = allErrors.filter(error =>
          error.error_type.toLowerCase().includes(searchTerm) ||
          error.description.toLowerCase().includes(searchTerm) ||
          error.id.toLowerCase().includes(searchTerm)
        );

        // Determine result type
        let resultType: 'skills' | 'errors' | 'mixed' | 'unknown';
        if (matchingSkills.length > 0 && matchingErrors.length > 0) {
          resultType = 'mixed';
        } else if (matchingSkills.length > 0) {
          resultType = 'skills';
        } else if (matchingErrors.length > 0) {
          resultType = 'errors';
        } else {
          resultType = 'unknown';
        }

        setResults({
          skills: matchingSkills,
          errors: matchingErrors,
          type: resultType,
        });

      } catch (error) {
        setResults({ skills: [], errors: [], type: 'unknown' });
      } finally {
        setLoading(false);
      }
    };

    performSearch();
  }, [query, currentHeight]);

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  if (loading) {
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
        <SearchIcon className="w-8 h-8 text-blue-400" />
        <div>
          <h1 className="text-3xl font-bold text-white">Search Results</h1>
          <p className="text-gray-400">Results for: <span className="font-mono text-gray-300">"{query}"</span></p>
        </div>
      </div>

      {/* Results */}
      {!results || results.type === 'unknown' ? (
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-12 border border-gray-700/50 text-center">
          <SearchIcon className="w-16 h-16 mx-auto mb-4 text-gray-500" />
          <h3 className="text-xl font-semibold text-gray-400 mb-2">No Results Found</h3>
          <p className="text-gray-500 mb-6">
            We couldn't find any SkillNodes or ErrorNodes matching your search.
          </p>
          <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 text-left max-w-md mx-auto">
            <h4 className="text-blue-400 font-medium mb-2">Search Tips:</h4>
            <ul className="text-sm text-gray-300 space-y-1">
              <li>• SkillNode type: e.g., "authentication", "validation"</li>
              <li>• Capabilities: e.g., "error handling", "data processing"</li>
              <li>• ErrorNode type: e.g., "network", "validation"</li>
              <li>• Node ID: Enter the full or partial ID</li>
            </ul>
          </div>
        </div>
      ) : results.type === 'skills' || results.type === 'mixed' ? (
        <div className="space-y-6">
          {/* SkillNodes Results */}
          {results.skills.length > 0 && (
            <>
              <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 flex items-center space-x-3">
                <Brain className="w-6 h-6 text-blue-400" />
                <div>
                  <div className="text-blue-400 font-medium">SkillNodes Found</div>
                  <div className="text-gray-400 text-sm">Found {results.skills.length} matching SkillNode{results.skills.length !== 1 ? 's' : ''}</div>
                </div>
              </div>

              <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
                <h2 className="text-xl font-semibold text-white mb-6">SkillNodes ({results.skills.length})</h2>
                <div className="space-y-4">
                  {results.skills.slice(0, 5).map((skill) => (
                    <Link
                      key={skill.id}
                      to={`/skill/${skill.id}`}
                      className="block p-4 bg-blue-500/10 border border-blue-500/20 rounded-lg hover:bg-blue-500/20 transition-colors"
                    >
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center space-x-3">
                          <Brain className="w-5 h-5 text-blue-400" />
                          <span className="text-white font-medium">{skill.skill_type}</span>
                        </div>
                        <ArrowRight className="w-4 h-4 text-blue-400" />
                      </div>
                      <div className="text-sm text-gray-400 mb-2">
                        {skill.capabilities.slice(0, 3).join(', ')}
                        {skill.capabilities.length > 3 && ` +${skill.capabilities.length - 3} more`}
                      </div>
                      {skill.performance && (
                        <div className="text-xs text-green-400">
                          {(skill.performance.success_rate * 100).toFixed(0)}% success rate
                        </div>
                      )}
                    </Link>
                  ))}
                  {results.skills.length > 5 && (
                    <div className="text-center py-4">
                      <Link
                        to="/skills"
                        className="text-blue-400 hover:text-blue-300 text-sm"
                      >
                        View all {results.skills.length} SkillNodes →
                      </Link>
                    </div>
                  )}
                </div>
              </div>
            </>
          )}

          {/* ErrorNodes Results */}
          {results.errors.length > 0 && (
            <>
              <div className="bg-orange-500/10 border border-orange-500/20 rounded-lg p-4 flex items-center space-x-3">
                <AlertTriangle className="w-6 h-6 text-orange-400" />
                <div>
                  <div className="text-orange-400 font-medium">ErrorNodes Found</div>
                  <div className="text-gray-400 text-sm">Found {results.errors.length} matching ErrorNode{results.errors.length !== 1 ? 's' : ''}</div>
                </div>
              </div>

              <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
                <h2 className="text-xl font-semibold text-white mb-6">ErrorNodes ({results.errors.length})</h2>
                <div className="space-y-4">
                  {results.errors.slice(0, 5).map((error) => (
                    <Link
                      key={error.id}
                      to={`/error/${error.id}`}
                      className="block p-4 bg-orange-500/10 border border-orange-500/20 rounded-lg hover:bg-orange-500/20 transition-colors"
                    >
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center space-x-3">
                          <AlertTriangle className="w-5 h-5 text-orange-400" />
                          <span className="text-white font-medium">{error.error_type}</span>
                        </div>
                        <div className="flex items-center space-x-2">
                          <span className="text-xs px-2 py-1 bg-orange-500/20 text-orange-300 rounded">
                            Severity {error.severity}/5
                          </span>
                          <ArrowRight className="w-4 h-4 text-orange-400" />
                        </div>
                      </div>
                      <div className="text-sm text-gray-400">
                        {error.description}
                      </div>
                    </Link>
                  ))}
                  {results.errors.length > 5 && (
                    <div className="text-center py-4">
                      <Link
                        to="/errors"
                        className="text-orange-400 hover:text-orange-300 text-sm"
                      >
                        View all {results.errors.length} ErrorNodes →
                      </Link>
                    </div>
                  )}
                </div>
              </div>
            </>
          )}
        </div>
      ) : results.type === 'errors' ? (
        <div className="space-y-6">
          <div className="bg-orange-500/10 border border-orange-500/20 rounded-lg p-4 flex items-center space-x-3">
            <AlertTriangle className="w-6 h-6 text-orange-400" />
            <div>
              <div className="text-orange-400 font-medium">ErrorNodes Found</div>
              <div className="text-gray-400 text-sm">Found {results.errors.length} matching ErrorNode{results.errors.length !== 1 ? 's' : ''}</div>
            </div>
          </div>

          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h2 className="text-xl font-semibold text-white mb-6">ErrorNodes ({results.errors.length})</h2>
            <div className="space-y-4">
              {results.errors.slice(0, 5).map((error) => (
                <Link
                  key={error.id}
                  to={`/error/${error.id}`}
                  className="block p-4 bg-orange-500/10 border border-orange-500/20 rounded-lg hover:bg-orange-500/20 transition-colors"
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center space-x-3">
                      <AlertTriangle className="w-5 h-5 text-orange-400" />
                      <span className="text-white font-medium">{error.error_type}</span>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className="text-xs px-2 py-1 bg-orange-500/20 text-orange-300 rounded">
                        Severity {error.severity}/5
                      </span>
                      <ArrowRight className="w-4 h-4 text-orange-400" />
                    </div>
                  </div>
                  <div className="text-sm text-gray-400">
                    {error.description}
                  </div>
                </Link>
              ))}
              {results.errors.length > 5 && (
                <div className="text-center py-4">
                  <Link
                    to="/errors"
                    className="text-orange-400 hover:text-orange-300 text-sm"
                  >
                    View all {results.errors.length} ErrorNodes →
                  </Link>
                </div>
              )}
            </div>
          </div>
        </div>
      ) : null}

      {/* Search Suggestions */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 mt-8">
        <h3 className="text-lg font-semibold text-white mb-4">Try These Searches</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Link
            to="/search/authentication"
            className="group bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 hover:border-blue-500/40 transition-all"
          >
            <Brain className="w-6 h-6 text-blue-400 mb-2" />
            <div className="text-white font-medium">Authentication</div>
            <div className="text-gray-400 text-sm">Search SkillNodes</div>
          </Link>

          <Link
            to="/search/network"
            className="group bg-orange-500/10 border border-orange-500/20 rounded-lg p-4 hover:border-orange-500/40 transition-all"
          >
            <AlertTriangle className="w-6 h-6 text-orange-400 mb-2" />
            <div className="text-white font-medium">Network Error</div>
            <div className="text-gray-400 text-sm">Search ErrorNodes</div>
          </Link>

          <Link
            to="/search/validation"
            className="group bg-purple-500/10 border border-purple-500/20 rounded-lg p-4 hover:border-purple-500/40 transition-all"
          >
            <Network className="w-6 h-6 text-purple-400 mb-2" />
            <div className="text-white font-medium">Validation</div>
            <div className="text-gray-400 text-sm">Search both types</div>
          </Link>
        </div>
      </div>
    </div>
  );
};

export default Search;