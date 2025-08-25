import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { graphChainApi, NRVVector } from '../services/api';
import { ArrowLeft, Clock, Hash, User, Target, Database, ChevronRight } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const VectorDetails: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [vector, setVector] = useState<NRVVector | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchVector = async () => {
      if (!id) return;

      setLoading(true);
      setError(null);

      try {
        // Note: This would need to be implemented in the API
        // const vectorData = await graphChainApi.getVector(id);
        // setVector(vectorData);
        
        // For now, create a mock vector
        const mockVector: NRVVector = {
          id,
          source_peer: 'peer_' + Math.random().toString(36).substr(2, 9),
          target_hash: 'hash_' + Math.random().toString(36).substr(2, 16),
          coordinates: [Math.random() * 100, Math.random() * 100, Math.random() * 100],
          confidence: Math.random(),
          timestamp: new Date().toISOString(),
          metadata: {
            type: 'skill_vector',
            priority: 'high'
          }
        };
        setVector(mockVector);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch vector');
      } finally {
        setLoading(false);
      }
    };

    fetchVector();
  }, [id]);

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatCoordinates = (coordinates: number[]) => {
    return coordinates.map(coord => coord.toFixed(4)).join(', ');
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <LoadingSpinner size="large" />
      </div>
    );
  }

  if (error || !vector) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-6 text-center">
          <div className="text-red-400 text-lg font-medium mb-2">Vector Not Found</div>
          <div className="text-gray-400 mb-4">{error || 'Vector not found'}</div>
          <Link
            to="/"
            className="inline-flex items-center space-x-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back to Dashboard</span>
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
          to="/"
          className="p-2 rounded-lg bg-gray-700/50 hover:bg-gray-700 text-gray-400 hover:text-white transition-colors"
        >
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div>
          <h1 className="text-3xl font-bold text-white">Vector Details</h1>
          <div className="flex items-center space-x-2 text-gray-400 mt-1">
            <Target className="w-4 h-4" />
            <span>Vector ID: {vector.id}</span>
          </div>
        </div>
      </div>

      {/* Vector Information */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Main Info */}
        <div className="lg:col-span-2 space-y-6">
          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h2 className="text-xl font-semibold mb-4 flex items-center space-x-2">
              <Database className="w-5 h-5 text-blue-400" />
              <span>Vector Information</span>
            </h2>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-4">
                <div>
                  <div className="text-sm text-gray-400 mb-1">Vector ID</div>
                  <div 
                    className="text-white font-mono text-sm bg-gray-700/30 rounded p-2 cursor-pointer hover:bg-gray-700/50 transition-colors"
                    onClick={() => copyToClipboard(vector.id)}
                  >
                    {vector.id}
                  </div>
                </div>
                
                <div>
                  <div className="text-sm text-gray-400 mb-1">Source Peer</div>
                  <div 
                    className="text-white font-mono text-sm bg-gray-700/30 rounded p-2 cursor-pointer hover:bg-gray-700/50 transition-colors"
                    onClick={() => copyToClipboard(vector.source_peer)}
                  >
                    {vector.source_peer}
                  </div>
                </div>

                <div>
                  <div className="text-sm text-gray-400 mb-1">Target Hash</div>
                  <div 
                    className="text-white font-mono text-sm bg-gray-700/30 rounded p-2 cursor-pointer hover:bg-gray-700/50 transition-colors"
                    onClick={() => copyToClipboard(vector.target_hash)}
                  >
                    {vector.target_hash}
                  </div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <div className="text-sm text-gray-400 mb-1">Confidence</div>
                  <div className="text-white">
                    <div className="flex items-center space-x-2">
                      <div className="flex-1 bg-gray-700 rounded-full h-2">
                        <div 
                          className="bg-gradient-to-r from-blue-500 to-purple-500 h-2 rounded-full"
                          style={{ width: `${vector.confidence * 100}%` }}
                        />
                      </div>
                      <span className="text-sm">{(vector.confidence * 100).toFixed(1)}%</span>
                    </div>
                  </div>
                </div>

                <div>
                  <div className="text-sm text-gray-400 mb-1">Coordinates</div>
                  <div className="text-white font-mono text-sm bg-gray-700/30 rounded p-2">
                    [{formatCoordinates(vector.coordinates)}]
                  </div>
                </div>

                <div>
                  <div className="text-sm text-gray-400 mb-1">Timestamp</div>
                  <div className="text-white flex items-center space-x-2">
                    <Clock className="w-4 h-4 text-gray-400" />
                    <span>{formatTime(vector.timestamp)}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Metadata */}
          {vector.metadata && Object.keys(vector.metadata).length > 0 && (
            <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
              <h2 className="text-xl font-semibold mb-4">Metadata</h2>
              <div className="space-y-2">
                {Object.entries(vector.metadata).map(([key, value]) => (
                  <div key={key} className="flex justify-between items-center py-2 border-b border-gray-700/30 last:border-b-0">
                    <span className="text-gray-400">{key}</span>
                    <span className="text-white font-mono text-sm">{String(value)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h3 className="text-lg font-semibold mb-4">Quick Actions</h3>
            <div className="space-y-3">
              <button
                onClick={() => copyToClipboard(vector.id)}
                className="w-full text-left px-4 py-2 bg-gray-700/30 hover:bg-gray-700/50 rounded-lg transition-colors"
              >
                <div className="flex items-center justify-between">
                  <span>Copy Vector ID</span>
                  <ChevronRight className="w-4 h-4" />
                </div>
              </button>
              
              <button
                onClick={() => copyToClipboard(vector.target_hash)}
                className="w-full text-left px-4 py-2 bg-gray-700/30 hover:bg-gray-700/50 rounded-lg transition-colors"
              >
                <div className="flex items-center justify-between">
                  <span>Copy Target Hash</span>
                  <ChevronRight className="w-4 h-4" />
                </div>
              </button>
            </div>
          </div>

          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <h3 className="text-lg font-semibold mb-4">Vector Stats</h3>
            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-gray-400">Dimensions</span>
                <span className="text-white">{vector.coordinates.length}D</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">Confidence Level</span>
                <span className="text-white">{vector.confidence > 0.8 ? 'High' : vector.confidence > 0.5 ? 'Medium' : 'Low'}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default VectorDetails;
