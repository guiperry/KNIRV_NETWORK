/**
 * KNIRVANA Game Visualization Component
 * Enhanced 3D graph visualization with game mechanics and agent deployment
 * Bridges personal KNIRVGRAPH with KNIRVANA collective game mechanics
 */

import React, { useRef, useEffect, useState, useCallback } from 'react';
import { Canvas } from '@react-three/fiber';
import { OrbitControls, Text, Html } from '@react-three/drei';
import * as THREE from 'three';
import { knirvanaBridgeService, KnirvanaErrorNode, KnirvanaSkillNode, KnirvanaAgent } from '../services/KnirvanaBridgeService';

interface GameNodeProps {
  node: KnirvanaErrorNode | KnirvanaSkillNode;
  onClick?: (node: KnirvanaErrorNode | KnirvanaSkillNode) => void;
  isSelected?: boolean;
  gameState?: string;
}

function GameNode({ node, onClick, isSelected, gameState: _gameState }: GameNodeProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const [hovered, setHovered] = useState(false);

  // Determine if this is an error node or skill node
  const isErrorNode = 'bounty' in node;
  const errorNode = node as KnirvanaErrorNode;
  const skillNode = node as KnirvanaSkillNode;

  // Node colors based on type and state
  const getNodeColor = (): string => {
    if (isErrorNode) {
      if (errorNode.isBeingSolved) {
        // Color interpolation based on progress
        return errorNode.progress > 0.5 ? '#ffaa00' : '#ff4444';
      }
      return '#ef4444'; // Red for unsolved errors
    }
    return skillNode.category === 'collective' ? '#00aaff' : '#10b981'; // Blue for collective, green for skills
  };

  // Node sizes based on type and state
  const getNodeSize = (): number => {
    if (isErrorNode) {
      return 0.6 + (errorNode.difficulty * 0.3) + (errorNode.isBeingSolved ? errorNode.progress * 0.2 : 0);
    }
    return skillNode.category === 'collective' ? 0.8 : 0.6;
  };

  // Pulsing animation for active nodes
  useEffect(() => {
    if (meshRef.current && isErrorNode && errorNode.isBeingSolved) {
      const animate = () => {
        if (meshRef.current) {
          meshRef.current.rotation.y += 0.01;
          const pulse = 1 + Math.sin(Date.now() * 0.005) * 0.1;
          meshRef.current.scale.setScalar(pulse);
        }
        requestAnimationFrame(animate);
      };
      animate();
    }
  }, [isErrorNode, errorNode?.isBeingSolved]);

  return (
    <group position={[node.position.x, node.position.y, node.position.z]}>
      <mesh
        ref={meshRef}
        onClick={() => onClick?.(node)}
        onPointerOver={() => setHovered(true)}
        onPointerOut={() => setHovered(false)}
        scale={getNodeSize()}
        castShadow
      >
        <sphereGeometry args={[0.5, 16, 16]} />
        <meshStandardMaterial
          color={getNodeColor()}
          emissive={isSelected ? getNodeColor() : '#000000'}
          emissiveIntensity={isSelected ? 0.3 : 0.1}
          transparent={true}
          opacity={hovered ? 0.9 : 0.8}
        />
      </mesh>

      {/* Progress ring for solving errors */}
      {isErrorNode && errorNode.isBeingSolved && (
        <mesh rotation={[Math.PI / 2, 0, 0]} position={[0, -0.1, 0]}>
          <ringGeometry args={[0.6, 0.8, 32, 1, 0, Math.PI * 2 * errorNode.progress]} />
          <meshBasicMaterial color="#00ff88" transparent opacity={0.8} />
        </mesh>
      )}

      {/* Agent indicator */}
      {isErrorNode && errorNode.solverAgent && (
        <mesh position={[0.8, 0.5, 0]}>
          <octahedronGeometry args={[0.2]} />
          <meshStandardMaterial color="#ffaa00" emissive="#442200" />
        </mesh>
      )}

      {/* Node label */}
      {(hovered || isSelected) && (
        <Html distanceFactor={10} position={[0, getNodeSize() + 0.8, 0]}>
          <div className="bg-gray-900/90 text-white px-3 py-2 rounded-lg shadow-lg max-w-xs">
            <div className="font-semibold text-sm">{node.id}</div>
            {isErrorNode ? (
              <div className="text-xs space-y-1">
                <div>Type: {errorNode.type}</div>
                <div>Difficulty: {(errorNode.difficulty * 100).toFixed(0)}%</div>
                <div>Bounty: {errorNode.bounty} NRN</div>
                {errorNode.isBeingSolved && (
                  <div>Progress: {(errorNode.progress * 100).toFixed(0)}%</div>
                )}
              </div>
            ) : (
              <div className="text-xs space-y-1">
                <div>Skill: {skillNode.name}</div>
                <div>Proficiency: {(skillNode.proficiency * 100).toFixed(0)}%</div>
                <div>Category: {skillNode.category}</div>
              </div>
            )}
          </div>
        </Html>
      )}

      {/* Selection indicator */}
      {isSelected && (
        <mesh position={[0, -1, 0]} rotation={[Math.PI / 2, 0, 0]}>
          <ringGeometry args={[1.2, 1.4, 32]} />
          <meshBasicMaterial color="#00aaff" transparent opacity={0.6} />
        </mesh>
      )}
    </group>
  );
}

interface AgentUnitProps {
  agent: KnirvanaAgent;
  onClick?: (agent: KnirvanaAgent) => void;
  isSelected?: boolean;
}

function AgentUnit({ agent, onClick, isSelected }: AgentUnitProps) {
  const meshRef = useRef<THREE.Mesh>(null);

  const getAgentColor = (): string => {
    switch (agent.type.toLowerCase()) {
      case 'analyzer': return '#10b981'; // Green
      case 'optimizer': return '#f59e0b'; // Amber
      case 'debugger': return '#8b5cf6'; // Purple
      default: return '#6b7280'; // Gray
    }
  };

  // Animate agent movement if it has a target
  useEffect(() => {
    if (meshRef.current && agent.target && agent.status === 'working') {
      const animate = () => {
        if (meshRef.current) {
          meshRef.current.rotation.y += 0.05;
        }
        requestAnimationFrame(animate);
      };
      animate();
    }
  }, [agent.target, agent.status]);

  return (
    <group position={[agent.position.x, agent.position.y, agent.position.z]}>
      <mesh
        ref={meshRef}
        onClick={() => onClick?.(agent)}
        castShadow
      >
        <boxGeometry args={[0.4, 0.8, 0.4]} />
        <meshStandardMaterial
          color={getAgentColor()}
          emissive={isSelected ? getAgentColor() : '#000000'}
          emissiveIntensity={isSelected ? 0.2 : 0}
        />
      </mesh>

      {/* Status indicator */}
      <mesh position={[0, 1, 0]}>
        <sphereGeometry args={[0.1]} />
        <meshBasicMaterial
          color={agent.status === 'idle' ? '#6b7280' : agent.status === 'working' ? '#10b981' : '#f59e0b'}
        />
      </mesh>

      {/* Agent label */}
      <Text
        position={[0, 1.2, 0]}
        fontSize={0.2}
        color="white"
        anchorX="center"
        anchorY="middle"
        maxWidth={2}
      >
        {agent.type}
      </Text>

      {/* Target indicator line */}
      {agent.target && (
        <line>
          <bufferGeometry>
            <bufferAttribute
              attach="attributes-position"
              args={[new Float32Array([
                0, 0, 0,
                Math.sin(Date.now() * 0.001) * 2, 0, Math.cos(Date.now() * 0.001) * 2
              ]), 3]}
              count={2}
              array={new Float32Array([
                0, 0, 0,
                Math.sin(Date.now() * 0.001) * 2, 0, Math.cos(Date.now() * 0.001) * 2
              ])}
              itemSize={3}
            />
          </bufferGeometry>
          <lineBasicMaterial color="#ffaa00" transparent opacity={0.5} />
        </line>
      )}
    </group>
  );
}

function GameScene({
  errorNodes,
  skillNodes,
  agents,
  selectedNodeId,
  selectedAgentId,
  onNodeClick,
  onAgentClick,
  gameState
}: {
  errorNodes: KnirvanaErrorNode[];
  skillNodes: KnirvanaSkillNode[];
  agents: KnirvanaAgent[];
  selectedNodeId: string | null;
  selectedAgentId: string | null;
  onNodeClick?: (node: KnirvanaErrorNode | KnirvanaSkillNode) => void;
  onAgentClick?: (agent: KnirvanaAgent) => void;
  gameState: string;
}) {
  return (
    <>
      <ambientLight intensity={0.4} />
      <directionalLight
        position={[10, 10, 5]}
        intensity={0.8}
        castShadow
        shadow-mapSize={[2048, 2048]}
      />
      <pointLight position={[-10, -10, -5]} intensity={0.3} />

      {/* Grid floor */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.5, 0]} receiveShadow>
        <planeGeometry args={[200, 200]} />
        <meshStandardMaterial color="#1a1a1a" transparent opacity={0.3} />
      </mesh>

      {/* Grid lines */}
      {Array.from({ length: 21 }, (_, i) => (
        <React.Fragment key={`grid-${i}`}>
      <line>
        <bufferGeometry>
          <bufferAttribute
            attach="attributes-position"
            count={2}
            array={new Float32Array([-100, 0, i * 10 - 100, 100, 0, i * 10 - 100])}
            itemSize={3}
            args={[new Float32Array([-100, 0, i * 10 - 100, 100, 0, i * 10 - 100]), 3]}
          />
        </bufferGeometry>
        <lineBasicMaterial color="#333" transparent opacity={0.2} />
      </line>
          <line>
            <bufferGeometry>
              <bufferAttribute
                attach="attributes-position"
                count={2}
                array={new Float32Array([i * 10 - 100, 0, -100, i * 10 - 100, 0, 100])}
                itemSize={3}
                args={[new Float32Array([i * 10 - 100, 0, -100, i * 10 - 100, 0, 100]), 3]}
              />
            </bufferGeometry>
            <lineBasicMaterial color="#333" transparent opacity={0.2} />
          </line>
        </React.Fragment>
      ))}

      {/* Render error nodes */}
      {errorNodes.map((node) => (
        <GameNode
          key={node.id}
          node={node}
          onClick={onNodeClick}
          isSelected={selectedNodeId === node.id}
          gameState={gameState}
        />
      ))}

      {/* Render skill nodes */}
      {skillNodes.map((node) => (
        <GameNode
          key={node.id}
          node={node}
          onClick={onNodeClick}
          isSelected={selectedNodeId === node.id}
          gameState={gameState}
        />
      ))}

      {/* Render agents */}
      {agents.map((agent) => (
        <AgentUnit
          key={agent.id}
          agent={agent}
          onClick={onAgentClick}
          isSelected={selectedAgentId === agent.id}
        />
      ))}
    </>
  );
}

export default function KNIRVANAGameVisualization() {
  const [gameState, setGameState] = useState(knirvanaBridgeService.getGameState());
  const [isInitialized, setIsInitialized] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);

  // Initialize bridge service
  const initializeGame = useCallback(async () => {
    if (!isInitialized) {
      await knirvanaBridgeService.initialize();
      setGameState(knirvanaBridgeService.getGameState());
      setIsInitialized(true);
    }
  }, [isInitialized]);

  useEffect(() => {
    initializeGame();
  }, [initializeGame]);

  // Handle game controls
  const handleStartGame = () => {
    knirvanaBridgeService.startGame();
    setIsPlaying(true);
    setGameState(knirvanaBridgeService.getGameState());
  };

  const handlePauseGame = () => {
    knirvanaBridgeService.pauseGame();
    setIsPlaying(false);
    setGameState(knirvanaBridgeService.getGameState());
  };

  const handleNodeClick = (node: KnirvanaErrorNode | KnirvanaSkillNode) => {
    if ('bounty' in node) {
      // Error node
      knirvanaBridgeService.selectErrorNode(node.id);
    } else {
      // Skill node - could implement skill learning here
      console.log('Skill node clicked:', node.name);
    }
    setGameState(knirvanaBridgeService.getGameState());
  };

  const handleAgentClick = (agent: KnirvanaAgent) => {
    knirvanaBridgeService.selectAgent(agent.id);
    setGameState(knirvanaBridgeService.getGameState());
  };

  const handleDeployAgent = async (agentId: string, nodeId: string) => {
    const success = await knirvanaBridgeService.deployAgent(agentId, nodeId);
    if (success) {
      setGameState(knirvanaBridgeService.getGameState());
    }
  };

  const handleCreateAgent = async (type: string) => {
    const success = await knirvanaBridgeService.createAgent(type);
    if (success) {
      setGameState(knirvanaBridgeService.getGameState());
    }
  };

  const handleMergeToCollective = async () => {
    await knirvanaBridgeService.mergeToCollectiveNetwork({});
    setGameState(knirvanaBridgeService.getGameState());
  };

  // Auto-update game state
  useEffect(() => {
    if (!isPlaying) return;

    const interval = setInterval(() => {
      setGameState(knirvanaBridgeService.getGameState());
    }, 100);

    return () => clearInterval(interval);
  }, [isPlaying]);

  if (!isInitialized) {
    return (
      <div className="bg-gray-800/80 backdrop-blur-xl rounded-xl p-6 border border-gray-600/50">
        <div className="flex items-center justify-center h-64">
          <div className="text-white">Initializing KNIRVANA Game Bridge...</div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800/80 backdrop-blur-xl rounded-xl p-6 border border-gray-600/50">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 bg-gradient-to-br from-purple-500 to-cyan-500 rounded-lg flex items-center justify-center">
            <svg className="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 2L13.09 8.26L20 9L13.09 9.74L12 16L10.91 9.74L4 9L10.91 8.26L12 2Z"/>
            </svg>
          </div>
          <div>
            <h3 className="text-lg font-semibold text-white">KNIRVANA Game Bridge</h3>
            <p className="text-sm text-gray-400">Personal graph meets collective game mechanics</p>
          </div>
        </div>

        <div className="flex space-x-2">
          {!isPlaying ? (
            <button
              onClick={handleStartGame}
              className="px-3 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-white text-sm"
            >
              Start Game
            </button>
          ) : (
            <button
              onClick={handlePauseGame}
              className="px-3 py-2 bg-yellow-600 hover:bg-yellow-700 rounded-lg text-white text-sm"
            >
              Pause
            </button>
          )}
          <button
            onClick={handleMergeToCollective}
            className="px-3 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-white text-sm"
          >
            Merge Collective
          </button>
        </div>
      </div>

      {/* Game Stats */}
      <div className="mb-4 grid grid-cols-5 gap-4">
        <div className="p-3 bg-gray-700/50 rounded-lg">
          <div className="text-xs text-gray-400">NRN Balance</div>
          <div className="text-lg font-semibold text-white">{gameState.nrnBalance}</div>
        </div>
        <div className="p-3 bg-gray-700/50 rounded-lg">
          <div className="text-xs text-gray-400">Errors Resolved</div>
          <div className="text-lg font-semibold text-white">{gameState.errorsResolved}</div>
        </div>
        <div className="p-3 bg-gray-700/50 rounded-lg">
          <div className="text-xs text-gray-400">Agents</div>
          <div className="text-lg font-semibold text-white">{gameState.agents.length}</div>
        </div>
        <div className="p-3 bg-gray-700/50 rounded-lg">
          <div className="text-xs text-gray-400">Error Nodes</div>
          <div className="text-lg font-semibold text-white">{gameState.errorNodes.length}</div>
        </div>
        <div className="p-3 bg-gray-700/50 rounded-lg">
          <div className="text-xs text-gray-400">Skill Nodes</div>
          <div className="text-lg font-semibold text-white">{gameState.skillNodes.length}</div>
        </div>
      </div>

      {/* Game Controls */}
      <div className="mb-4 flex flex-wrap gap-2">
        <button
          onClick={() => handleCreateAgent('Analyzer')}
          className="px-3 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-white text-sm"
        >
          + Analyzer (50 NRN)
        </button>
        <button
          onClick={() => handleCreateAgent('Optimizer')}
          className="px-3 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-white text-sm"
        >
          + Optimizer (50 NRN)
        </button>
        {gameState.selectedAgent && gameState.selectedErrorNode && (
          <button
            onClick={() => handleDeployAgent(gameState.selectedAgent!, gameState.selectedErrorNode!)}
            className="px-3 py-2 bg-purple-600 hover:bg-purple-700 rounded-lg text-white text-sm"
          >
            Deploy Agent (10 NRN)
          </button>
        )}
      </div>

      {/* 3D Game Canvas */}
      <div className="h-96 bg-gray-900/50 rounded-lg overflow-hidden">
        <Canvas
          camera={{ position: [0, 5, 20], fov: 75 }}
          shadows
        >
          <OrbitControls
            enablePan={true}
            enableZoom={true}
            enableRotate={true}
            minDistance={5}
            maxDistance={100}
          />
          <GameScene
            errorNodes={gameState.errorNodes}
            skillNodes={gameState.skillNodes}
            agents={gameState.agents}
            selectedNodeId={gameState.selectedErrorNode}
            selectedAgentId={gameState.selectedAgent}
            onNodeClick={handleNodeClick}
            onAgentClick={handleAgentClick}
            gameState={gameState.gamePhase}
          />
        </Canvas>
      </div>

      {/* Status Messages */}
      {gameState.selectedErrorNode && (
        <div className="mt-4 p-3 bg-blue-900/30 border border-blue-500/30 rounded-lg">
          <div className="text-sm text-blue-300">
            Selected Error: {gameState.selectedErrorNode}
          </div>
        </div>
      )}

      {gameState.selectedAgent && (
        <div className="mt-2 p-3 bg-green-900/30 border border-green-500/30 rounded-lg">
          <div className="text-sm text-green-300">
            Selected Agent: {gameState.selectedAgent}
          </div>
        </div>
      )}

      {/* Instructions */}
      <div className="mt-4 p-4 bg-gray-700/30 rounded-lg">
        <h4 className="text-white font-semibold mb-2">How to Play:</h4>
        <ul className="text-sm text-gray-300 space-y-1">
          <li>• Click error nodes to select them (red spheres)</li>
          <li>• Click agents to select them for deployment</li>
          <li>• Deploy agents to solve errors and earn NRN tokens</li>
          <li>• Create new agents to handle more complex errors</li>
          <li>• Merge with collective to gain advanced skills</li>
        </ul>
      </div>
    </div>
  );
}
