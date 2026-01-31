import React, { useRef } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";
import { Agent as AgentType } from "./stores/useKnirvana";

interface AIAgentProps {
  agent: AgentType;
  isSelected: boolean;
  onSelect: () => void;
  onStage?: () => void;
}

export default function AIAgent({ agent, isSelected, onSelect, onStage }: AIAgentProps) {
  const meshRef = useRef<THREE.Group>(null);

  useFrame((state) => {
    if (meshRef.current) {
      // Idle animation
      const time = state.clock.elapsedTime;
      const bounceHeight = agent.status === 'idle' ? Math.sin(time * 3) * 0.1 : 0;
      meshRef.current.position.y = agent.position.y + bounceHeight;
      
      // Movement animation for working agents
      if (agent.status === 'working') {
        meshRef.current.rotation.y += 0.05;
      }
    }
  });

  const getColor = () => {
    switch (agent.status) {
      case 'idle': return '#00ff00';
      case 'moving': return '#ffff00';
      case 'working': return '#ff00ff';
      case 'upgrading': return '#00ffff';
      default: return '#ffffff';
    }
  };

  return (
    <group
      ref={meshRef}
      position={[agent.position.x, agent.position.y, agent.position.z]}
      onClick={onSelect}
    >
      {/* Agent body */}
      <mesh castShadow receiveShadow>
        <boxGeometry args={[0.8, 1.2, 0.8]} />
        <meshStandardMaterial
          color={getColor()}
          emissive={getColor()}
          emissiveIntensity={isSelected ? 0.5 : 0.2}
          roughness={0.3}
          metalness={0.8}
        />
      </mesh>
      
      {/* Agent indicator light */}
      <pointLight
        position={[0, 1, 0]}
        color={getColor()}
        intensity={isSelected ? 1 : 0.6}
        distance={3}
        decay={2}
      />
      
      {/* Selection indicator */}
      {isSelected && (
        <mesh position={[0, 2, 0]}>
          <ringGeometry args={[1.2, 1.4, 8]} />
          <meshBasicMaterial color="#ffffff" transparent opacity={0.8} />
        </mesh>
      )}

      {/* Stage Button - visual indicator without text (avoids CSP worker issues) */}
      {isSelected && onStage && (
        <group position={[0, -1.2, 0]} onClick={(e) => { e.stopPropagation(); onStage(); }}>
          <mesh>
            <boxGeometry args={[0.8, 0.25, 0.1]} />
            <meshStandardMaterial color="#ff6b35" emissive="#ff6b35" emissiveIntensity={0.6} />
          </mesh>
          {/* Arrow indicator pointing up to agent */}
          <mesh position={[0, 0.25, 0]} rotation={[0, 0, Math.PI]}>
            <coneGeometry args={[0.15, 0.2, 4]} />
            <meshStandardMaterial color="#ff6b35" emissive="#ff6b35" emissiveIntensity={0.6} />
          </mesh>
        </group>
      )}
    </group>
  );
}
