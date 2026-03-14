import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useGraphChain } from '../context/GraphChainContext';
import { graphChainApi, ErrorNode, SkillNode } from '../services/api';
import { AlertTriangle, ArrowLeft, Brain, CheckCircle, XCircle, Clock } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const ErrorNodeDetails: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { isLoading } = useGraphChain();
  const [error, setError] = useState<ErrorNode | null>(null);
  const [relatedSkills, setRelatedSkills] = useState<SkillNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);

  useEffect(() => {
    const fetchErrorDetails = async () => {
      if (!id) return;
      
      setLoading(true);
      setFetchError(null);

      try {
        // For now, we'll get all errors and find the one we need
        const errors = await graphChainApi.getAllErrors();
        const foundError = errors.find(e => e.id === id);
        
        if (!foundError) {
          setFetchError('ErrorNode not found');
          return;
        }

        setError(foundError);

        // Get related skills
        const related = await graphChainApi.getSkillsForError(foundError.error_type);
        setRelatedSkills(related);

      } catch (err) {
        setFetchError(err instanceof Error ? err.message : 'Failed to fetch ErrorNode details');
      } finally {
        setLoading(false);
      }
    };

    fetchErrorDetails();
  }, [id]);

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

  const getStatusIcon = (status?: string) => {
    switch (status) {
      case 'resolved':
        return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-400" />;
      default:
        return <Clock className="w-5 h-5 text-yellow-400" />;
    }
  };

  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'resolved':
        return 'text-green-400';
      case 'failed':
        return 'text-red-400';
      default:
        return 'text-yellow-400';
    }
  };

  if (isLoading || loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <LoadingSpinner size="large" />
      </div>
    );
  }

  if (fetchError) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-6 text-center">
          <div className="text-red-400 text-lg font-medium mb-2">Error</div>
          <div className="text-gray-400 mb-4">{fetchError}</div>
          <Link
            to="/errors"
            className="inline-flex items-center space-x-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back to ErrorNodes</span>
          </Link>
        </div>
      </div>
    );
  }

  if (!error) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="text-center py-12">
          <AlertTriangle className="w-16 h-16 mx-auto mb-4 text-gray-500" />
          <h3 className="text-xl font-semibold text-gray-400 mb-2">ErrorNode Not Found</h3>
          <p className="text-gray-500 mb-6">The requested ErrorNode could not be found.</p>
          <Link
            to="/errors"
            className="inline-flex items-center space-x-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back to ErrorNodes</span>
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
          to="/errors"
          className="p-2 bg-gray-700/50 hover:bg-gray-700 rounded-lg transition-colors"
        >
          <ArrowLeft className="w-5 h-5 text-gray-400" />
        </Link>
        <div className="flex items-center space-x-3">
          <AlertTriangle className="w-8 h-8 text-orange-400" />
          <div>
            <h1 className="text-3xl font-bold text-white">{error.error_type}</h1>
            <p className="text-gray-400">ErrorNode Details</p>
          </div>
        </div>
        <div className="ml-auto flex items-center space-x-3">
          <div className={`px-3 py-1 rounded text-sm font-medium ${getSeverityColor(error.severity)}`}>
            {getSeverityLabel(error.severity)}
          </div>
          {error.resolution_status && (
            <div className="flex items-center space-x-2">
              {getStatusIcon(error.resolution_status)}
              <span className={`text-sm font-medium ${getStatusColor(error.resolution_status)}`}>
                {error.resolution_status.charAt(0).toUpperCase() + error.resolution_status.slice(1)}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Main Content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Column - Main Details */}
        <div className="lg:col-span-2 space-y-6">
          {/* Description */}
          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h2 className="text-xl font-semibold text-white mb-4">Description</h2>
            <div className="text-gray-300 leading-relaxed">
              {error.description}
            </div>
          </div>

          {/* Basic Info */}
          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h2 className="text-xl font-semibold text-white mb-4">Basic Information</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <div className="text-sm text-gray-400 mb-1">Error ID</div>
                <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2">
                  {error.id}
                </div>
              </div>
              <div>
                <div className="text-sm text-gray-400 mb-1">Error Type</div>
                <div className="text-gray-300 text-sm">
                  {error.error_type}
                </div>
              </div>
              <div>
                <div className="text-sm text-gray-400 mb-1">Severity Level</div>
                <div className="text-gray-300 text-sm">
                  {error.severity}/5
                </div>
              </div>
              <div>
                <div className="text-sm text-gray-400 mb-1">Created</div>
                <div className="text-gray-300 text-sm">
                  {formatTime(error.timestamp)}
                </div>
              </div>
            </div>
          </div>

          {/* Context */}
          {error.context && Object.keys(error.context).length > 0 && (
            <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
              <h2 className="text-xl font-semibold text-white mb-4">Error Context</h2>
              <div className="bg-gray-700/30 rounded p-4">
                <pre className="text-gray-300 text-sm overflow-x-auto">
                  {JSON.stringify(error.context, null, 2)}
                </pre>
              </div>
            </div>
          )}

          {/* Resolution Information */}
          {error.resolved_by && error.resolved_by.length > 0 && (
            <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
              <h2 className="text-xl font-semibold text-white mb-4">Resolution Information</h2>
              <div>
                <div className="text-sm text-gray-400 mb-2">Resolved By</div>
                <div className="flex flex-wrap gap-2">
                  {error.resolved_by.map((resolver, index) => (
                    <span 
                      key={index}
                      className="px-3 py-1 bg-green-500/20 text-green-300 rounded-full text-sm"
                    >
                      {resolver}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Right Column - Related Skills */}
        <div className="space-y-6">
          {/* Related SkillNodes */}
          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h3 className="text-lg font-semibold text-white mb-4">
              Related SkillNodes ({relatedSkills.length})
            </h3>
            {relatedSkills.length > 0 ? (
              <div className="space-y-3">
                {relatedSkills.map((skill) => (
                  <Link
                    key={skill.id}
                    to={`/skill/${skill.id}`}
                    className="block p-3 bg-blue-500/10 border border-blue-500/20 rounded hover:bg-blue-500/20 transition-colors"
                  >
                    <div className="flex items-center space-x-2 mb-2">
                      <Brain className="w-4 h-4 text-blue-400" />
                      <span className="text-blue-300 font-medium text-sm">{skill.skill_type}</span>
                    </div>
                    <div className="text-xs text-gray-400 mb-2">
                      {skill.capabilities.slice(0, 2).join(', ')}
                      {skill.capabilities.length > 2 && ` +${skill.capabilities.length - 2} more`}
                    </div>
                    {skill.performance && (
                      <div className="text-xs text-green-400">
                        {(skill.performance.success_rate * 100).toFixed(0)}% success rate
                      </div>
                    )}
                    {skill.validation && (
                      <div className="text-xs text-blue-400">
                        Validation: {(skill.validation.validation_score * 100).toFixed(0)}%
                      </div>
                    )}
                  </Link>
                ))}
              </div>
            ) : (
              <div className="text-center py-8 text-gray-400">
                <Brain className="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>No related SkillNodes found</p>
              </div>
            )}
          </div>

          {/* Quick Actions */}
          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h3 className="text-lg font-semibold text-white mb-4">Quick Actions</h3>
            <div className="space-y-3">
              <Link
                to="/graph"
                className="block w-full px-4 py-2 bg-purple-500/20 border border-purple-500/30 text-purple-300 rounded hover:bg-purple-500/30 transition-colors text-center"
              >
                View in Graph
              </Link>
              <button className="block w-full px-4 py-2 bg-green-500/20 border border-green-500/30 text-green-300 rounded hover:bg-green-500/30 transition-colors">
                Mark as Resolved
              </button>
              <button className="block w-full px-4 py-2 bg-blue-500/20 border border-blue-500/30 text-blue-300 rounded hover:bg-blue-500/30 transition-colors">
                Find Resolution Path
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ErrorNodeDetails;
