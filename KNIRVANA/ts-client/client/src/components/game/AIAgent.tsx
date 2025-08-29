import { useRef, useState } from 'react';
import { useFrame } from '@react-three/fiber';
import { Text } from '@react-three/drei';
import * as THREE from 'three';
import { useKnirvana } from '../../lib/stores/useKnirvana';
import AnimatedAgentAvatar from './AnimatedAgentAvatar';

interface AIAgentProps {
  id: string;
  position: { x: number; y: number; z: number };
  target: string | null;
  status: 'idle' | 'moving' | 'working' | 'upgrading';
  type: string;
  efficiency: number;
  experience?: number;
}

export default function AIAgent({
  id,
  position,
  target,
  status,
  type,
  efficiency,
  experience = 0
}: AIAgentProps) {
  const agentRef = useRef<THREE.Group>(null);
  const coreRef = useRef<THREE.Mesh>(null);
  const trailRef = useRef<THREE.Points>(null);
  const [hovered, setHovered] = useState(false);
  const [showThoughts, setShowThoughts] = useState(false);
  
  const { selectAgent, selectedAgent } = useKnirvana();
  
  const isSelected = selectedAgent === id;

  // Agent thoughts based on status
  const getThoughts = () => {
    switch (status) {
      case 'working':
        return [
          'Analyzing error patterns...',
          'Accessing KNIRVGRAPH database...',
          'Computing optimal solution...',
          'Applying learned heuristics...'
        ];
      case 'moving':
        return [
          'Navigating to target...',
          'Optimizing path...',
          'Preparing skill set...'
        ];
      case 'upgrading':
        return [
          'Updating neural weights...',
          'Integrating new knowledge...',
          'Expanding capability matrix...'
        ];
      default:
        return [
          'Monitoring network status...',
          'Ready for deployment...',
          'Awaiting instructions...'
        ];
    }
  };

  useFrame((state) => {
    if (agentRef.current) {
      // Floating animation
      agentRef.current.position.y = position.y + Math.sin(state.clock.elapsedTime * 3 + parseFloat(id.slice(-3))) * 0.15;
      
      // Rotation based on status
      if (status === 'working') {
        agentRef.current.rotation.y += 0.08;
      } else if (status === 'moving') {
        agentRef.current.rotation.x = Math.sin(state.clock.elapsedTime * 4) * 0.2;
      }
    }
    
    if (coreRef.current) {
      // Core pulsing based on efficiency
      const pulse = 0.7 + efficiency * 0.5 + Math.sin(state.clock.elapsedTime * 5) * 0.1;
      coreRef.current.scale.setScalar(pulse);
      
      // Color intensity based on status
      const material = coreRef.current.material as THREE.MeshStandardMaterial;
      const intensity = status === 'working' ? 0.8 : status === 'moving' ? 0.4 : 0.2;
      material.emissiveIntensity = intensity;
    }
  });

  const handleClick = (event: any) => {
    // Prevent event from bubbling to camera controller
    event.stopPropagation();
    console.log(`AI Agent ${id} (${type}) clicked - Status: ${status}`);
    selectAgent(id);
    setShowThoughts(!showThoughts);
  };

  const getStatusColor = () => {
    switch (status) {
      case 'idle': return '#888888';
      case 'moving': return '#00aaff';
      case 'working': return '#ffaa00';
      case 'upgrading': return '#aa00ff';
      default: return '#ffffff';
    }
  };

  const getAgentShape = () => {
    switch (type) {
      case 'Analyzer':
        return <icosahedronGeometry args={[0.3, 1]} />;
      case 'Optimizer':
        return <boxGeometry args={[0.35, 0.35, 0.35]} />;
      case 'Synthesizer':
        return <coneGeometry args={[0.25, 0.5, 8]} />;
      case 'Debugger':
        return <tetrahedronGeometry args={[0.3]} />;
      default:
        return <sphereGeometry args={[0.3]} />;
    }
  };

  const getExperienceLevel = () => {
    return Math.floor(experience / 25) + 1;
  };

  return (
    <group 
      ref={agentRef}
      position={[position.x, position.y, position.z]}
    >
      {/* Bright visible robot agent */}
      <mesh
        onClick={handleClick}
        onPointerOver={() => setHovered(true)}
        onPointerOut={() => setHovered(false)}
        castShadow
        receiveShadow
        position={[0, 0.5, 0]}
      >
        <boxGeometry args={[0.8, 1.2, 0.6]} />
        <meshStandardMaterial
          color={getStatusColor()}
          emissive={getStatusColor()}
          emissiveIntensity={1.0}
          metalness={0.2}
          roughness={0.8}
        />
      </mesh>

      {/* Robot head */}
      <mesh position={[0, 1.1, 0]}>
        <boxGeometry args={[0.6, 0.5, 0.5]} />
        <meshStandardMaterial
          color={getStatusColor()}
          emissive={getStatusColor()}
          emissiveIntensity={0.8}
        />
      </mesh>

      {/* Bright glowing eyes */}
      <mesh position={[-0.15, 1.15, 0.26]}>
        <sphereGeometry args={[0.06]} />
        <meshBasicMaterial color="#00ffff" />
      </mesh>
      <mesh position={[0.15, 1.15, 0.26]}>
        <sphereGeometry args={[0.06]} />
        <meshBasicMaterial color="#00ffff" />
      </mesh>

      {/* Bright selection indicator */}
      {isSelected && (
        <mesh rotation={[Math.PI / 2, 0, 0]} position={[0, 0.1, 0]}>
          <ringGeometry args={[0.8, 1.0, 32]} />
          <meshBasicMaterial color="#00ffff" transparent opacity={0.8} />
        </mesh>
      )}

      {/* Agent name text */}
      {(hovered || isSelected) && (
        <Text
          position={[0, 1.8, 0]}
          fontSize={0.15}
          color="#ffffff"
          anchorX="center"
          anchorY="middle"
        >
          {type} Agent
        </Text>
      )}

      {/* Experience level indicators around avatar */}
      {Array.from({ length: getExperienceLevel() }, (_, i) => (
        <mesh 
          key={`exp-${i}`}
          position={[
            Math.cos((i / getExperienceLevel()) * Math.PI * 2) * 0.8,
            1.2,
            Math.sin((i / getExperienceLevel()) * Math.PI * 2) * 0.8
          ]}
        >
          <sphereGeometry args={[0.03]} />
          <meshBasicMaterial color="#ffaa00" />
        </mesh>
      ))}



      {/* Agent information display */}
      {(hovered || isSelected) && (
        <group position={[0, 1.5, 0]}>
          <Text
            fontSize={0.1}
            color="#00ffff"
            anchorX="center"
            anchorY="bottom"
          >
            {`${type} Agent`}
          </Text>
          <Text
            position={[0, -0.15, 0]}
            fontSize={0.06}
            color="#88ffff"
            anchorX="center"
            anchorY="top"
          >
            {`Status: ${status}`}
          </Text>
          <Text
            position={[0, -0.25, 0]}
            fontSize={0.06}
            color="#aaffff"
            anchorX="center"
            anchorY="top"
          >
            {`Efficiency: ${Math.round(efficiency * 100)}%`}
          </Text>
          <Text
            position={[0, -0.35, 0]}
            fontSize={0.06}
            color="#ffffaa"
            anchorX="center"
            anchorY="top"
          >
            {`Level: ${getExperienceLevel()}`}
          </Text>
        </group>
      )}

      {/* Thought process display */}
      {showThoughts && status === 'working' && (
        <group position={[1.2, 0.5, 0]}>
          <mesh>
            <planeGeometry args={[2, 1]} />
            <meshBasicMaterial 
              color="#000000"
              transparent
              opacity={0.8}
            />
          </mesh>
          {getThoughts().map((thought, index) => (
            <Text
              key={index}
              position={[0, 0.3 - index * 0.15, 0.01]}
              fontSize={0.05}
              color="#00ff88"
              anchorX="center"
              anchorY="middle"
            >
              {`→ ${thought}`}
            </Text>
          ))}
        </group>
      )}

      {/* Base glow around avatar */}
      <mesh position={[0, -0.2, 0]} rotation={[Math.PI / 2, 0, 0]}>
        <circleGeometry args={[1, 16]} />
        <meshBasicMaterial 
          color={getStatusColor()} 
          transparent 
          opacity={0.15}
        />
      </mesh>
    </group>
  );
}
