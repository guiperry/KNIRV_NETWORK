/**
 * KNIRVANA Graph Visualization Component
 * 3D graph visualization for personal KNIRVGRAPH with Three.js
 */

import React, { useRef, useEffect, useState, useCallback } from 'react';
import { Canvas } from '@react-three/fiber';
import { OrbitControls, Text } from '@react-three/drei';
import * as THREE from 'three';
import { personalKNIRVGRAPHService, GraphNode, GraphEdge } from '../services/PersonalKNIRVGRAPHService';

interface GraphNodeProps {
  node: GraphNode;
  onClick?: (node: GraphNode) => void;
}

function GraphNodeComponent({ node, onClick }: GraphNodeProps) {
  const meshRef = useRef<THREE.Mesh>(null);

  // Node colors based on type
  const getNodeColor = (type: string): string => {
    switch (type) {
      case 'error': return '#ef4444'; // red
      case 'skill': return '#10b981'; // green
      case 'capability': return '#3b82f6'; // blue
      case 'property': return '#f59e0b'; // amber
      case 'connection': return '#8b5cf6'; // purple
      case 'agent': return '#06b6d4'; // cyan
      default: return '#6b7280'; // gray
    }
  };

  // Node sizes based on type
  const getNodeSize = (type: string): number => {
    switch (type) {
      case 'error': return 0.8;
      case 'skill': return 0.6;
      case 'capability': return 0.7;
      case 'property': return 0.9;
      case 'connection': return 0.4;
      case 'agent': return 0.7;
      default: return 0.5;
    }
  };

  return (
    <group position={[node.position.x, node.position.y, node.position.z]}>
      <mesh
        ref={meshRef}
        onClick={() => onClick?.(node)}
        onPointerOver={(e: React.PointerEvent<THREE.Mesh>) => {
          e.stopPropagation();
          document.body.style.cursor = 'pointer';
        }}
        onPointerOut={(e: React.PointerEvent<THREE.Mesh>) => {
          e.stopPropagation();
          document.body.style.cursor = 'auto';
        }}
      >
        <sphereGeometry args={[getNodeSize(node.type), 16, 16]} />
        <meshStandardMaterial color={getNodeColor(node.type)} />
      </mesh>

      {/* Node label */}
      <Text
        position={[0, getNodeSize(node.type) + 0.5, 0]}
        fontSize={0.3}
        color="white"
        anchorX="center"
        anchorY="middle"
        maxWidth={3}
      >
        {node.label.length > 15 ? node.label.substring(0, 12) + '...' : node.label}
      </Text>
    </group>
  );
}

interface GraphEdgeProps {
  edge: GraphEdge;
  nodes: GraphNode[];
}

function GraphEdgeComponent({ edge, nodes }: GraphEdgeProps) {
  const sourceNode = nodes.find(n => n.id === edge.source);
  const targetNode = nodes.find(n => n.id === edge.target);

  if (!sourceNode || !targetNode) return null;

  // Calculate edge geometry
  const sourcePos = sourceNode.position;
  const targetPos = targetNode.position;

  // Create line geometry
  const points = [
    new THREE.Vector3(sourcePos.x, sourcePos.y, sourcePos.z),
    new THREE.Vector3(targetPos.x, targetPos.y, targetPos.z)
  ];

  const geometry = new THREE.BufferGeometry().setFromPoints(points);

  const getEdgeColor = (edgeType: string): string => {
    switch (edgeType) {
      case 'error_to_skill': return '#f97316'; // orange
      case 'context_to_capability': return '#3b82f6'; // blue
      case 'idea_to_property': return '#f59e0b'; // amber
      case 'collaboration': return '#8b5cf6'; // purple
      case 'skill_chain': return '#10b981'; // green
      case 'agent_connection': return '#06b6d4'; // cyan
      default: return '#94a3b8'; // gray
    }
  };

  const material = new THREE.LineBasicMaterial({
    color: getEdgeColor(edge.type),
    linewidth: 2
  });

  return (
    <primitive object={new THREE.Line(geometry, material)} />
  );
}

interface GraphSceneProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  onNodeClick?: (node: GraphNode) => void;
}

function GraphScene({ nodes, edges, onNodeClick }: GraphSceneProps) {
  return (
    <>
      <ambientLight intensity={0.6} />
      <pointLight position={[10, 10, 10]} intensity={0.5} />
      <pointLight position={[-10, -10, -10]} intensity={0.3} />

      {/* Render edges first */}
      {edges.map((edge) => (
        <GraphEdgeComponent
          key={edge.id}
          edge={edge}
          nodes={nodes}
        />
      ))}

      {/* Render nodes */}
      {nodes.map((node) => (
        <GraphNodeComponent
          key={node.id}
          node={node}
          onClick={onNodeClick}
        />
      ))}
    </>
  );
}

export default function KNIRVANAGraphVisualization() {
  const [graphData, setGraphData] = useState<{ nodes: GraphNode[]; edges: GraphEdge[] } | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [stats, setStats] = useState<{ nodeCount: number; edgeCount: number; complexity: number; nodeTypes: Record<string, number> } | null>(null);

  const initializeGraph = useCallback(async () => {
    try {
      setIsLoading(true);

      // Load or create personal graph
      await personalKNIRVGRAPHService.loadPersonalGraph('current_user');

      // Add some sample data
      await addSampleData();

      // Get graph data for visualization
      const data = personalKNIRVGRAPHService.exportGraphData();
      const graphStats = personalKNIRVGRAPHService.getGraphStats();

      setGraphData(data);
      setStats(graphStats);
    } catch (error) {
      console.error('Failed to initialize graph:', error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    initializeGraph();
  }, [initializeGraph]);

  const addSampleData = async () => {
    try {
      // Add some sample error
      await personalKNIRVGRAPHService.addErrorNode({
        errorId: 'sample_error_1',
        errorType: 'TypeScript Error',
        description: 'Cannot find module react',
        context: { file: 'App.tsx', line: 5 },
        timestamp: Date.now()
      });

      // Add some sample skills
      await personalKNIRVGRAPHService.addSkillNode({
        skillId: 'typescript',
        skillName: 'TypeScript',
        description: 'TypeScript development',
        category: 'programming',
        proficiency: 0.8
      });

      await personalKNIRVGRAPHService.addSkillNode({
        skillId: 'react',
        skillName: 'React',
        description: 'React development',
        category: 'frontend',
        proficiency: 0.9
      });

      // Create connections
      const graph = personalKNIRVGRAPHService.getCurrentGraph();
      if (graph) {
        const errorNode = graph.nodes.find(n => n.type === 'error');
        const tsNode = graph.nodes.find(n => n.type === 'skill' && 'skillName' in n.data && n.data.skillName === 'TypeScript');

        if (errorNode && tsNode) {
          await personalKNIRVGRAPHService.createConnection(
            errorNode.id,
            tsNode.id,
            'error_to_skill'
          );
        }
      }
    } catch (error) {
      console.error('Failed to add sample data:', error);
    }
  };

  const handleNodeClick = (node: GraphNode) => {
    setSelectedNode(node);
  };

  const handleRefresh = async () => {
    await initializeGraph();
  };

  const handleReset = async () => {
    await personalKNIRVGRAPHService.resetGraph();
    setGraphData(null);
    setStats(null);
    await initializeGraph();
  };

  if (isLoading) {
    return (
      <div className="bg-gray-800/80 backdrop-blur-xl rounded-xl p-6 border border-gray-600/50">
        <div className="flex items-center justify-center h-64">
          <div className="text-white">Loading KNIRVANA Graph...</div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800/80 backdrop-blur-xl rounded-xl p-6 border border-gray-600/50">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 bg-gradient-to-br from-purple-500 to-cyan-500 rounded-lg flex items-center justify-center">
            <svg className="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 2L13.09 8.26L20 9L13.09 9.74L12 16L10.91 9.74L4 9L10.91 8.26L12 2Z"/>
            </svg>
          </div>
          <div>
            <h3 className="text-lg font-semibold text-white">KNIRVANA Graph Visualization</h3>
            <p className="text-sm text-gray-400">Personal knowledge graph with error mapping</p>
          </div>
        </div>

        <div className="flex space-x-2">
          <button
            onClick={handleRefresh}
            className="px-3 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-white text-sm"
          >
            Refresh
          </button>
          <button
            onClick={handleReset}
            className="px-3 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-white text-sm"
          >
            Reset
          </button>
        </div>
      </div>

      {/* Graph Statistics */}
      {stats && (
        <div className="mb-4 grid grid-cols-4 gap-4">
          <div className="p-3 bg-gray-700/50 rounded-lg">
            <div className="text-xs text-gray-400">Nodes</div>
            <div className="text-lg font-semibold text-white">{stats.nodeCount}</div>
          </div>
          <div className="p-3 bg-gray-700/50 rounded-lg">
            <div className="text-xs text-gray-400">Edges</div>
            <div className="text-lg font-semibold text-white">{stats.edgeCount}</div>
          </div>
          <div className="p-3 bg-gray-700/50 rounded-lg">
            <div className="text-xs text-gray-400">Complexity</div>
            <div className="text-lg font-semibold text-white">{stats.complexity}</div>
          </div>
          <div className="p-3 bg-gray-700/50 rounded-lg">
            <div className="text-xs text-gray-400">Types</div>
            <div className="text-lg font-semibold text-white">{Object.keys(stats.nodeTypes).length}</div>
          </div>
        </div>
      )}

      {/* 3D Graph Canvas */}
      <div className="h-96 bg-gray-900/50 rounded-lg overflow-hidden">
        {graphData && graphData.nodes.length > 0 ? (
          <Canvas camera={{ position: [0, 0, 200], fov: 75 }}>
            <OrbitControls enablePan={true} enableZoom={true} enableRotate={true} />
            <GraphScene
              nodes={graphData.nodes}
              edges={graphData.edges}
              onNodeClick={handleNodeClick}
            />
          </Canvas>
        ) : (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <div className="text-gray-400 mb-2">No graph data available</div>
              <button
                onClick={addSampleData}
                className="px-4 py-2 bg-purple-600 hover:bg-purple-700 rounded-lg text-white text-sm"
              >
                Add Sample Data
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Node Details Panel */}
      {selectedNode && (
        <div className="mt-4 p-4 bg-gray-700/50 rounded-lg">
          <h4 className="text-white font-semibold mb-2">Selected Node</h4>
          <div className="space-y-2 text-sm">
            <div><span className="text-gray-400">Type:</span> <span className="text-white capitalize">{selectedNode.type}</span></div>
            <div><span className="text-gray-400">Label:</span> <span className="text-white">{selectedNode.label}</span></div>
            {selectedNode.data && (
              <div>
                <span className="text-gray-400">Details:</span>
                <pre className="text-xs text-gray-300 mt-1 p-2 bg-gray-800/50 rounded">
                  {JSON.stringify(selectedNode.data, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Legend */}
      <div className="mt-4 flex flex-wrap gap-4 text-xs">
        <div className="flex items-center space-x-2">
          <div className="w-3 h-3 bg-red-500 rounded-full"></div>
          <span className="text-gray-400">Error Nodes</span>
        </div>
        <div className="flex items-center space-x-2">
          <div className="w-3 h-3 bg-green-500 rounded-full"></div>
          <span className="text-gray-400">Skill Nodes</span>
        </div>
        <div className="flex items-center space-x-2">
          <div className="w-3 h-3 bg-blue-500 rounded-full"></div>
          <span className="text-gray-400">Connection Nodes</span>
        </div>
        <div className="flex items-center space-x-2">
          <div className="w-3 h-3 bg-amber-500 rounded-full"></div>
          <span className="text-gray-400">Agent Nodes</span>
        </div>
      </div>
    </div>
  );
}
