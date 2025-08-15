import React, { useEffect, useState } from 'react';
import { useGraphChain } from '../context/GraphChainContext';
import { graphChainApi, SkillNode, ErrorNode } from '../services/api';
import { Network, Brain, AlertTriangle, Zap } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const GraphVisualization: React.FC = () => {
  const { currentHeight, isLoading } = useGraphChain();
  const [skills, setSkills] = useState<SkillNode[]>([]);
  const [errors, setErrors] = useState<ErrorNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [filter, setFilter] = useState<'all' | 'skills' | 'errors'>('all');

  useEffect(() => {
    const fetchGraphData = async () => {
      setLoading(true);
      setError(null);

      try {
        const [skillsData, errorsData] = await Promise.all([
          graphChainApi.getAllSkills(),
          graphChainApi.getAllErrors(),
        ]);

        setSkills(skillsData);
        setErrors(errorsData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch graph data');
      } finally {
        setLoading(false);
      }
    };

    fetchGraphData();
  }, [currentHeight]);

  const getNodeConnections = (nodeId: string, nodeType: 'skill' | 'error') => {
    if (nodeType === 'skill') {
      const skill = skills.find(s => s.id === nodeId);
      if (!skill) return [];
      
      // Find errors that this skill can resolve
      return errors.filter(error => 
        skill.capabilities.some(cap => 
          error.error_type.toLowerCase().includes(cap.toLowerCase()) ||
          cap.toLowerCase().includes(error.error_type.toLowerCase())
        )
      );
    } else {
      const errorNode = errors.find(e => e.id === nodeId);
      if (!errorNode) return [];
      
      // Find skills that can resolve this error
      return skills.filter(skill =>
        skill.capabilities.some(cap => 
          errorNode.error_type.toLowerCase().includes(cap.toLowerCase()) ||
          cap.toLowerCase().includes(errorNode.error_type.toLowerCase())
        )
      );
    }
  };

  const renderNode = (node: SkillNode | ErrorNode, type: 'skill' | 'error') => {
    const isSelected = selectedNode === node.id;
    const connections = getNodeConnections(node.id, type);
    
    return (
      <div
        key={node.id}
        className={`relative p-4 rounded-lg border cursor-pointer transition-all duration-200 ${
          type === 'skill'
            ? 'bg-blue-500/10 border-blue-500/30 hover:bg-blue-500/20'
            : 'bg-orange-500/10 border-orange-500/30 hover:bg-orange-500/20'
        } ${isSelected ? 'ring-2 ring-white/50 scale-105' : ''}`}
        onClick={() => setSelectedNode(isSelected ? null : node.id)}
      >
        <div className="flex items-center space-x-2 mb-2">
          {type === 'skill' ? (
            <Brain className="w-5 h-5 text-blue-400" />
          ) : (
            <AlertTriangle className="w-5 h-5 text-orange-400" />
          )}
          <span className="text-white font-medium text-sm">
            {type === 'skill' ? (node as SkillNode).skill_type : (node as ErrorNode).error_type}
          </span>
        </div>
        
        <div className="text-xs text-gray-400 mb-2">
          {type === 'skill' 
            ? `${(node as SkillNode).capabilities.length} capabilities`
            : `Severity: ${(node as ErrorNode).severity}/5`
          }
        </div>
        
        <div className="flex items-center space-x-1 text-xs">
          <Zap className="w-3 h-3 text-green-400" />
          <span className="text-green-400">{connections.length} connections</span>
        </div>

        {/* Connection lines (simplified visualization) */}
        {isSelected && connections.length > 0 && (
          <div className="absolute top-full left-1/2 transform -translate-x-1/2 mt-2 z-10">
            <div className="bg-gray-800 border border-gray-600 rounded p-2 shadow-lg min-w-48">
              <div className="text-xs text-gray-400 mb-1">Connected to:</div>
              {connections.slice(0, 3).map((conn) => (
                <div key={conn.id} className="text-xs text-white mb-1">
                  {type === 'skill' 
                    ? (conn as ErrorNode).error_type 
                    : (conn as SkillNode).skill_type
                  }
                </div>
              ))}
              {connections.length > 3 && (
                <div className="text-xs text-gray-400">
                  +{connections.length - 3} more
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    );
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
          <div className="text-red-400 text-lg font-medium mb-2">Error Loading Graph</div>
          <div className="text-gray-400">{error}</div>
        </div>
      </div>
    );
  }

  const filteredSkills = filter === 'errors' ? [] : skills;
  const filteredErrors = filter === 'skills' ? [] : errors;

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div className="flex items-center space-x-3">
          <Network className="w-8 h-8 text-purple-400" />
          <div>
            <h1 className="text-3xl font-bold text-white">Graph Visualization</h1>
            <p className="text-gray-400">SkillNode-ErrorNode relationship map</p>
          </div>
        </div>
        
        {/* Filter Controls */}
        <div className="flex items-center space-x-4">
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value as 'all' | 'skills' | 'errors')}
            className="px-3 py-2 bg-gray-700/50 border border-gray-600 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-purple-500"
          >
            <option value="all">All Nodes</option>
            <option value="skills">SkillNodes Only</option>
            <option value="errors">ErrorNodes Only</option>
          </select>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 text-center">
          <Brain className="w-8 h-8 text-blue-400 mx-auto mb-2" />
          <div className="text-2xl font-bold text-blue-400">{skills.length}</div>
          <div className="text-gray-400 text-sm">SkillNodes</div>
        </div>
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 text-center">
          <AlertTriangle className="w-8 h-8 text-orange-400 mx-auto mb-2" />
          <div className="text-2xl font-bold text-orange-400">{errors.length}</div>
          <div className="text-gray-400 text-sm">ErrorNodes</div>
        </div>
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 text-center">
          <Zap className="w-8 h-8 text-green-400 mx-auto mb-2" />
          <div className="text-2xl font-bold text-green-400">
            {skills.reduce((total, skill) => 
              total + getNodeConnections(skill.id, 'skill').length, 0
            )}
          </div>
          <div className="text-gray-400 text-sm">Connections</div>
        </div>
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 text-center">
          <Network className="w-8 h-8 text-purple-400 mx-auto mb-2" />
          <div className="text-2xl font-bold text-purple-400">{skills.length + errors.length}</div>
          <div className="text-gray-400 text-sm">Total Nodes</div>
        </div>
      </div>

      {/* Graph Visualization Area */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-white">Node Network</h2>
          <div className="text-sm text-gray-400">
            Click on nodes to see connections
          </div>
        </div>

        {/* Simple Grid Layout */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 relative">
          {/* SkillNodes */}
          {filteredSkills.map((skill) => renderNode(skill, 'skill'))}
          
          {/* ErrorNodes */}
          {filteredErrors.map((error) => renderNode(error, 'error'))}
        </div>

        {filteredSkills.length === 0 && filteredErrors.length === 0 && (
          <div className="text-center py-12 text-gray-400">
            <Network className="w-16 h-16 mx-auto mb-4 opacity-50" />
            <h3 className="text-xl font-semibold mb-2">No Nodes to Display</h3>
            <p>Adjust your filter to see nodes in the graph.</p>
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="mt-8 bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
        <h3 className="text-lg font-semibold text-white mb-4">Legend</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="flex items-center space-x-3">
            <div className="w-4 h-4 bg-blue-500/20 border border-blue-500/30 rounded"></div>
            <span className="text-gray-300">SkillNodes - Capabilities that can resolve errors</span>
          </div>
          <div className="flex items-center space-x-3">
            <div className="w-4 h-4 bg-orange-500/20 border border-orange-500/30 rounded"></div>
            <span className="text-gray-300">ErrorNodes - Problems that need resolution</span>
          </div>
          <div className="flex items-center space-x-3">
            <Zap className="w-4 h-4 text-green-400" />
            <span className="text-gray-300">Connections - Relationships between nodes</span>
          </div>
          <div className="flex items-center space-x-3">
            <div className="w-4 h-4 border-2 border-white/50 rounded"></div>
            <span className="text-gray-300">Selected node with visible connections</span>
          </div>
        </div>
      </div>

      {/* Future Enhancement Notice */}
      <div className="mt-8 bg-blue-500/10 border border-blue-500/20 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-blue-300 mb-2">🚀 Coming Soon</h3>
        <p className="text-gray-300 text-sm">
          Advanced graph visualization with interactive network diagrams, force-directed layouts, 
          and real-time relationship mapping using libraries like D3.js or React Flow.
        </p>
      </div>
    </div>
  );
};

export default GraphVisualization;
