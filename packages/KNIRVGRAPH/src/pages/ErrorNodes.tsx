import React, { useEffect, useState } from 'react';
import { graphChainApi, ErrorNode, SkillNode } from '../services/api';
import { useGraphChain } from '../context/GraphChainContext';
import { AlertTriangle, Clock, Hash, ArrowRight, Brain, CheckCircle, XCircle } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const ErrorNodes: React.FC = () => {
  const { currentHeight, isLoading } = useGraphChain();
  const [errors, setErrors] = useState<ErrorNode[]>([]);
  const [skills, setSkills] = useState<SkillNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedError, setSelectedError] = useState<ErrorNode | null>(null);
  const [relatedSkills, setRelatedSkills] = useState<SkillNode[]>([]);

  useEffect(() => {
    const fetchErrorNodes = async () => {
      setLoading(true);
      setError(null);

      try {
        const [fetchedErrors, fetchedSkills] = await Promise.all([
          graphChainApi.getAllErrors(),
          graphChainApi.getAllSkills(),
        ]);

        // Sort errors by timestamp (most recent first)
        fetchedErrors.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

        setErrors(fetchedErrors);
        setSkills(fetchedSkills);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch ErrorNodes');
      } finally {
        setLoading(false);
      }
    };

    fetchErrorNodes();
  }, [currentHeight]);

  // Fetch related skills when an error is selected
  useEffect(() => {
    const fetchRelatedSkills = async () => {
      if (selectedError) {
        try {
          const related = await graphChainApi.getSkillsForError(selectedError.error_type);
          setRelatedSkills(related);
        } catch (err) {
          console.error('Failed to fetch related skills:', err);
        }
      }
    };

    fetchRelatedSkills();
  }, [selectedError]);

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const getSeverityColor = (severity: number) => {
    if (severity >= 4) return 'text-red-400 bg-red-500/20';
    if (severity >= 3) return 'text-orange-400 bg-orange-500/20';
    if (severity >= 2) return 'text-yellow-400 bg-yellow-500/20';
    return 'text-blue-400 bg-blue-500/20';
  };

  const getSeverityLabel = (severity: number) => {
    if (severity >= 4) return 'CRITICAL';
    if (severity >= 3) return 'HIGH';
    if (severity >= 2) return 'MEDIUM';
    return 'LOW';
  };

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
        <AlertTriangle className="w-8 h-8 text-orange-400" />
        <div>
          <h1 className="text-3xl font-bold text-white">ErrorNodes</h1>
          <p className="text-gray-400">Explore ErrorNodes and their resolution paths</p>
        </div>
      </div>

      {/* Stats */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 mb-8">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="text-center">
            <div className="text-3xl font-bold text-orange-400">{errors.length}</div>
            <div className="text-gray-400">Total ErrorNodes</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-red-400">
              {errors.filter(e => e.severity >= 3).length}
            </div>
            <div className="text-gray-400">High Severity</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-green-400">
              {errors.filter(e => e.resolution_status === 'resolved').length}
            </div>
            <div className="text-gray-400">Resolved</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-blue-400">{skills.length}</div>
            <div className="text-gray-400">Available Skills</div>
          </div>
        </div>
      </div>

      {/* ErrorNodes List */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <LoadingSpinner size="large" />
        </div>
      ) : error ? (
        <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-6 text-center">
          <div className="text-red-400 text-lg font-medium mb-2">Error Loading ErrorNodes</div>
          <div className="text-gray-400">{error}</div>
        </div>
      ) : errors.length === 0 ? (
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-12 border border-gray-700/50 text-center">
          <AlertTriangle className="w-16 h-16 mx-auto mb-4 text-gray-500" />
          <h3 className="text-xl font-semibold text-gray-400 mb-2">No ErrorNodes Found</h3>
          <p className="text-gray-500">There are no ErrorNodes in the system.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {errors.map((errorNode) => (
            <div
              key={errorNode.id}
              className={`bg-gray-800/50 backdrop-blur-xl border border-gray-700/50 rounded-xl p-6 hover:border-gray-600/50 transition-all duration-200 cursor-pointer ${
                selectedError?.id === errorNode.id ? 'ring-2 ring-orange-500/50' : ''
              }`}
              onClick={() => setSelectedError(selectedError?.id === errorNode.id ? null : errorNode)}
            >
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-orange-500/20 rounded-lg flex items-center justify-center">
                    <AlertTriangle className="w-5 h-5 text-orange-400" />
                  </div>
                  <div>
                    <div className="text-white font-medium">{errorNode.error_type}</div>
                    <div className="text-gray-400 text-sm flex items-center space-x-1">
                      <Clock className="w-3 h-3" />
                      <span>{formatTime(errorNode.timestamp)}</span>
                    </div>
                  </div>
                </div>
                <div className="text-right flex items-center space-x-3">
                  <div className={`px-2 py-1 rounded text-xs font-medium ${getSeverityColor(errorNode.severity)}`}>
                    {getSeverityLabel(errorNode.severity)}
                  </div>
                  {errorNode.resolution_status && (
                    <div className="flex items-center space-x-1">
                      {errorNode.resolution_status === 'resolved' ? (
                        <CheckCircle className="w-4 h-4 text-green-400" />
                      ) : errorNode.resolution_status === 'failed' ? (
                        <XCircle className="w-4 h-4 text-red-400" />
                      ) : (
                        <Clock className="w-4 h-4 text-yellow-400" />
                      )}
                      <span className="text-sm text-gray-400 capitalize">{errorNode.resolution_status}</span>
                    </div>
                  )}
                </div>
              </div>

              <div className="mb-4">
                <div className="text-sm text-gray-400 mb-2">Description</div>
                <div className="text-gray-300 text-sm bg-gray-700/30 rounded p-3">
                  {errorNode.description}
                </div>
              </div>

              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">
                <div className="space-y-3">
                  <div>
                    <div className="flex items-center space-x-2 text-sm text-gray-400 mb-1">
                      <Hash className="w-3 h-3" />
                      <span>Error ID</span>
                    </div>
                    <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2">
                      {errorNode.id}
                    </div>
                  </div>
                  <div>
                    <div className="flex items-center space-x-2 text-sm text-gray-400 mb-1">
                      <AlertTriangle className="w-3 h-3" />
                      <span>Error Type</span>
                    </div>
                    <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2">
                      {errorNode.error_type}
                    </div>
                  </div>
                </div>
                <div className="space-y-3">
                  <div>
                    <div className="flex items-center space-x-2 text-sm text-gray-400 mb-1">
                      <span>Severity Level</span>
                    </div>
                    <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2">
                      {errorNode.severity}/5
                    </div>
                  </div>
                  {errorNode.resolved_by && errorNode.resolved_by.length > 0 && (
                    <div>
                      <div className="flex items-center space-x-2 text-sm text-gray-400 mb-1">
                        <Brain className="w-3 h-3" />
                        <span>Resolved By</span>
                      </div>
                      <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2">
                        {errorNode.resolved_by.join(', ')}
                      </div>
                    </div>
                  )}
                </div>
              </div>

              {errorNode.context && Object.keys(errorNode.context).length > 0 && (
                <div className="border-t border-gray-700/50 pt-4">
                  <div className="text-sm text-gray-400 mb-1">Error Context</div>
                  <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2 max-h-20 overflow-y-auto">
                    {JSON.stringify(errorNode.context, null, 2)}
                  </div>
                </div>
              )}

              {selectedError?.id === errorNode.id && relatedSkills.length > 0 && (
                <div className="border-t border-gray-700/50 pt-4 mt-4">
                  <div className="text-sm text-gray-400 mb-3">Related SkillNodes ({relatedSkills.length})</div>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {relatedSkills.slice(0, 4).map((skill) => (
                      <div key={skill.id} className="bg-blue-500/10 border border-blue-500/20 rounded p-3">
                        <div className="flex items-center space-x-2 mb-2">
                          <Brain className="w-4 h-4 text-blue-400" />
                          <span className="text-blue-300 font-medium text-sm">{skill.skill_type}</span>
                        </div>
                        <div className="text-xs text-gray-400">
                          {skill.capabilities.slice(0, 2).join(', ')}
                          {skill.capabilities.length > 2 && ` +${skill.capabilities.length - 2} more`}
                        </div>
                        {skill.performance && (
                          <div className="text-xs text-green-400 mt-1">
                            {(skill.performance.success_rate * 100).toFixed(0)}% success rate
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                  {relatedSkills.length > 4 && (
                    <div className="text-center mt-3">
                      <span className="text-sm text-gray-400">+{relatedSkills.length - 4} more skills available</span>
                    </div>
                  )}
                </div>
              )}

              <div className="flex items-center justify-between mt-4 pt-4 border-t border-gray-700/50">
                <div className="text-sm text-gray-400">
                  Click to {selectedError?.id === errorNode.id ? 'hide' : 'view'} related skills
                </div>
                <div className="flex items-center space-x-2 text-sm text-orange-400 hover:text-orange-300">
                  <span>View Details</span>
                  <ArrowRight className="w-3 h-3" />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default ErrorNodes;