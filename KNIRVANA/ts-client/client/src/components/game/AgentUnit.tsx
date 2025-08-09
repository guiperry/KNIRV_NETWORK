import { useRef, useState } from "react";
import { useFrame } from "@react-three/fiber";
import { Text } from "@react-three/drei";
import * as THREE from "three";
import { useKnirvana } from "../../lib/stores/useKnirvana";

interface AgentUnitProps {
  id: string;
  position: { x: number; y: number; z: number };
  target: string | null;
  status: 'idle' | 'moving' | 'working' | 'upgrading';
  type: string;
  efficiency: number;
}

export default function AgentUnit({
  id,
  position,
  target,
  status,
  type,
  efficiency
}: AgentUnitProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const [hovered, setHovered] = useState(false);
  const { selectAgent, selectedAgent, moveAgent } = useKnirvana();

  const isSelected = selectedAgent === id;

  useFrame((state) => {
    if (meshRef.current) {
      // Hovering animation
      meshRef.current.position.y = position.y + Math.sin(state.clock.elapsedTime * 4) * 0.1;
      
      // Rotation based on status
      if (status === 'working') {
        meshRef.current.rotation.y += 0.05;
      } else if (status === 'moving') {
        meshRef.current.rotation.x += 0.02;
      }
      
      // Pulsing based on efficiency
      const pulse = 0.8 + efficiency * 0.4 + Math.sin(state.clock.elapsedTime * 2) * 0.1;
      meshRef.current.scale.setScalar(pulse);
    }
  });

  const handleClick = () => {
    console.log(`Agent ${id} clicked`);
    selectAgent(id);
  };

  const getStatusColor = () => {
    switch (status) {
      case 'idle': return "#aaaaaa";
      case 'moving': return "#00aaff";
      case 'working': return "#ffaa00";
      case 'upgrading': return "#aa00ff";
      default: return "#ffffff";
    }
  };

  const getTypeShape = () => {
    switch (type) {
      case 'Analyzer':
        return <octahedronGeometry args={[0.3]} />;
      case 'Optimizer':
        return <boxGeometry args={[0.4, 0.4, 0.4]} />;
      case 'Synthesizer':
        return <coneGeometry args={[0.3, 0.6, 8]} />;
      default:
        return <sphereGeometry args={[0.3]} />;
    }
  };

  return (
    <group position={[position.x, position.y, position.z]}>
      <mesh
        ref={meshRef}
        onClick={handleClick}
        onPointerOver={() => setHovered(true)}
        onPointerOut={() => setHovered(false)}
        castShadow
      >
        {getTypeShape()}
        <meshStandardMaterial
          color={getStatusColor()}
          emissive={getStatusColor()}
          emissiveIntensity={isSelected ? 0.4 : 0.2}
          metalness={0.8}
          roughness={0.2}
        />
      </mesh>
      
      {/* Selection ring */}
      {isSelected && (
        <mesh rotation={[Math.PI / 2, 0, 0]} position={[0, -0.5, 0]}>
          <ringGeometry args={[0.5, 0.7, 32]} />
          <meshBasicMaterial color="#00ffff" transparent opacity={0.6} />
        </mesh>
      )}
      
      {/* Efficiency indicator trails */}
      <mesh position={[0, 0.8, 0]}>
        <sphereGeometry args={[0.05]} />
        <meshBasicMaterial 
          color="#00ff00" 
          transparent 
          opacity={efficiency}
        />
      </mesh>
      
      {/* Floating info text */}
      {(hovered || isSelected) && (
        <Text
          position={[0, 1.2, 0]}
          fontSize={0.15}
          color="#00ffff"
          anchorX="center"
          anchorY="middle"
        >
          {`${type} Agent\nStatus: ${status}\nEfficiency: ${Math.round(efficiency * 100)}%`}
        </Text>
      )}
      
      {/* Working particles */}
      {status === 'working' && (
        <mesh>
          <bufferGeometry>
            <bufferAttribute
              attach="attributes-position"
              count={20}
              array={new Float32Array(
                Array.from({ length: 60 }, () => (Math.random() - 0.5) * 2)
              )}
              itemSize={3}
            />
          </bufferGeometry>
          <pointsMaterial 
            size={0.05} 
            color="#ffaa00" 
            transparent={true}
            opacity={0.8}
          />
        </mesh>
      )}
    </group>
  );
}
