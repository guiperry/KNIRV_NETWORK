import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useGraphChain } from '../context/GraphChainContext';
import { graphChainApi, SkillNode, ErrorNode } from '../services/api';
import { Brain, ArrowLeft, CheckCircle, AlertCircle, TrendingUp, Clock, Users } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const SkillNodeDetails: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { isLoading } = useGraphChain();
  const [skill, setSkill] = useState<SkillNode | null>(null);
  const [relatedErrors, setRelatedErrors] = useState<ErrorNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchSkillDetails = async () => {
      if (!id) return;
      
      setLoading(true);
      setError(null);

      try {
        // For now, we'll get all skills and find the one we need
        // In a real implementation, there would be a getSkill(id) endpoint
        const skills = await graphChainApi.getAllSkills();
        const foundSkill = skills.find(s => s.id === id);
        
        if (!foundSkill) {
          setError('SkillNode not found');
          return;
        }

        setSkill(foundSkill);

        // Get all errors to find related ones
        const errors = await graphChainApi.getAllErrors();
        const related = errors.filter(error => 
          foundSkill.capabilities.some(cap => 
            error.error_type.toLowerCase().includes(cap.toLowerCase()) ||
            cap.toLowerCase().includes(error.error_type.toLowerCase())
          )
        );
        setRelatedErrors(related);

      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch SkillNode details');
      } finally {
        setLoading(false);
      }
    };

    fetchSkillDetails();
  }, [id]);

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const getValidationColor = (validation?: SkillNode['validation']) => {
    if (!validation) return 'text-gray-400';
    if (validation.is_validated && validation.validation_score > 0.8) return 'text-green-400';
    if (validation.is_validated && validation.validation_score > 0.6) return 'text-yellow-400';
    return 'text-red-400';
  };

  if (isLoading || loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <LoadingSpinner size="large" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-6 text-center">
          <div className="text-red-400 text-lg font-medium mb-2">Error</div>
          <div className="text-gray-400 mb-4">{error}</div>
          <Link
            to="/skills"
            className="inline-flex items-center space-x-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back to SkillNodes</span>
          </Link>
        </div>
      </div>
    );
  }

  if (!skill) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="text-center py-12">
          <Brain className="w-16 h-16 mx-auto mb-4 text-gray-500" />
          <h3 className="text-xl font-semibold text-gray-400 mb-2">SkillNode Not Found</h3>
          <p className="text-gray-500 mb-6">The requested SkillNode could not be found.</p>
          <Link
            to="/skills"
            className="inline-flex items-center space-x-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back to SkillNodes</span>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center space-x-4 mb-8">
        <Link
          to="/skills"
          className="p-2 bg-gray-700/50 hover:bg-gray-700 rounded-lg transition-colors"
        >
          <ArrowLeft className="w-5 h-5 text-gray-400" />
        </Link>
        <div className="flex items-center space-x-3">
          <Brain className="w-8 h-8 text-blue-400" />
          <div>
            <h1 className="text-3xl font-bold text-white">{skill.skill_type}</h1>
            <p className="text-gray-400">SkillNode Details</p>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Column - Main Details */}
        <div className="lg:col-span-2 space-y-6">
          {/* Basic Info */}
          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h2 className="text-xl font-semibold text-white mb-4">Basic Information</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <div className="text-sm text-gray-400 mb-1">Skill ID</div>
                <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2">
                  {skill.id}
                </div>
              </div>
              <div>
                <div className="text-sm text-gray-400 mb-1">Created</div>
                <div className="text-gray-300 text-sm">
                  {formatTime(skill.timestamp)}
                </div>
              </div>
            </div>
          </div>

          {/* Capabilities */}
          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h2 className="text-xl font-semibold text-white mb-4">Capabilities</h2>
            <div className="flex flex-wrap gap-2">
              {skill.capabilities.map((capability, index) => (
                <span 
                  key={index}
                  className="px-3 py-1 bg-blue-500/20 text-blue-300 rounded-full text-sm"
                >
                  {capability}
                </span>
              ))}
            </div>
          </div>

          {/* Requirements */}
          {skill.requirements && Object.keys(skill.requirements).length > 0 && (
            <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
              <h2 className="text-xl font-semibold text-white mb-4">Requirements</h2>
              <div className="bg-gray-700/30 rounded p-4">
                <pre className="text-gray-300 text-sm overflow-x-auto">
                  {JSON.stringify(skill.requirements, null, 2)}
                </pre>
              </div>
            </div>
          )}

          {/* Related ErrorNodes */}
          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h2 className="text-xl font-semibold text-white mb-4">
              Related ErrorNodes ({relatedErrors.length})
            </h2>
            {relatedErrors.length > 0 ? (
              <div className="space-y-3">
                {relatedErrors.slice(0, 5).map((error) => (
                  <Link
                    key={error.id}
                    to={`/error/${error.id}`}
                    className="block p-3 bg-orange-500/10 border border-orange-500/20 rounded hover:bg-orange-500/20 transition-colors"
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="text-orange-300 font-medium">{error.error_type}</div>
                        <div className="text-gray-400 text-sm">{error.description}</div>
                      </div>
                      <div className="text-orange-400 text-sm">
                        Severity: {error.severity}/5
                      </div>
                    </div>
                  </Link>
                ))}
                {relatedErrors.length > 5 && (
                  <div className="text-center text-gray-400 text-sm">
                    +{relatedErrors.length - 5} more related errors
                  </div>
                )}
              </div>
            ) : (
              <div className="text-center py-8 text-gray-400">
                <AlertCircle className="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>No related ErrorNodes found</p>
              </div>
            )}
          </div>
        </div>

        {/* Right Column - Stats & Validation */}
        <div className="space-y-6">
          {/* Validation Status */}
          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h3 className="text-lg font-semibold text-white mb-4">Validation Status</h3>
            {skill.validation ? (
              <div className="space-y-3">
                <div className="flex items-center space-x-2">
                  {skill.validation.is_validated ? (
                    <CheckCircle className="w-5 h-5 text-green-400" />
                  ) : (
                    <AlertCircle className="w-5 h-5 text-red-400" />
                  )}
                  <span className={getValidationColor(skill.validation)}>
                    {skill.validation.is_validated ? 'Validated' : 'Not Validated'}
                  </span>
                </div>
                <div>
                  <div className="text-sm text-gray-400 mb-1">Validation Score</div>
                  <div className="text-2xl font-bold text-white">
                    {(skill.validation.validation_score * 100).toFixed(1)}%
                  </div>
                </div>
                {skill.validation.validated_by && skill.validation.validated_by.length > 0 && (
                  <div>
                    <div className="text-sm text-gray-400 mb-1">Validated By</div>
                    <div className="text-gray-300 text-sm">
                      {skill.validation.validated_by.join(', ')}
                    </div>
                  </div>
                )}
                <div>
                  <div className="text-sm text-gray-400 mb-1">Last Validated</div>
                  <div className="text-gray-300 text-sm">
                    {formatTime(skill.validation.last_validated)}
                  </div>
                </div>
              </div>
            ) : (
              <div className="text-center py-4 text-gray-400">
                <AlertCircle className="w-8 h-8 mx-auto mb-2 opacity-50" />
                <p>No validation data available</p>
              </div>
            )}
          </div>

          {/* Performance Metrics */}
          {skill.performance && (
            <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
              <h3 className="text-lg font-semibold text-white mb-4">Performance Metrics</h3>
              <div className="space-y-4">
                <div>
                  <div className="flex items-center space-x-2 mb-1">
                    <TrendingUp className="w-4 h-4 text-green-400" />
                    <span className="text-sm text-gray-400">Success Rate</span>
                  </div>
                  <div className="text-2xl font-bold text-green-400">
                    {(skill.performance.success_rate * 100).toFixed(1)}%
                  </div>
                </div>
                <div>
                  <div className="flex items-center space-x-2 mb-1">
                    <Clock className="w-4 h-4 text-blue-400" />
                    <span className="text-sm text-gray-400">Avg Resolution Time</span>
                  </div>
                  <div className="text-2xl font-bold text-blue-400">
                    {skill.performance.avg_resolution_time.toFixed(1)}s
                  </div>
                </div>
                <div>
                  <div className="flex items-center space-x-2 mb-1">
                    <Users className="w-4 h-4 text-purple-400" />
                    <span className="text-sm text-gray-400">Total Resolutions</span>
                  </div>
                  <div className="text-2xl font-bold text-purple-400">
                    {skill.performance.total_resolutions.toLocaleString()}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default SkillNodeDetails;
