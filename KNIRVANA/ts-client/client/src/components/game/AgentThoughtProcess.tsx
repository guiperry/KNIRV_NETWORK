import React, { useState, useEffect } from 'react';
import { Text } from '@react-three/drei';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface AgentThoughtProcessProps {
  agentId: string;
  errorNodeId: string;
  position: [number, number, number];
  isVisible: boolean;
}

export default function AgentThoughtProcess({ 
  agentId, 
  errorNodeId, 
  position, 
  isVisible 
}: AgentThoughtProcessProps) {
  const [currentThought, setCurrentThought] = useState(0);
  const [thoughts] = useState([
    "Analyzing error patterns in KNIRVGRAPH...",
    "Accessing relevant SkillNodes database...",
    "Computing optimal solution path...",
    "Applying learned heuristics from previous tasks...",
    "Validating solution against knowledge constraints...",
    "Executing resolution sequence...",
    "Updating neural weights based on outcome..."
  ]);

  useFrame((state) => {
    // Cycle through thoughts every 2 seconds
    const cycleTime = Math.floor(state.clock.elapsedTime / 2);
    setCurrentThought(cycleTime % thoughts.length);
  });

  if (!isVisible) return null;

  return (
    <group position={position}>
      {/* Thought bubble background */}
      <mesh position={[0, 0, 0]}>
        <planeGeometry args={[4, 2]} />
        <meshBasicMaterial 
          color="#000000"
          transparent
          opacity={0.85}
        />
      </mesh>

      {/* Border glow */}
      <mesh position={[0, 0, -0.01]}>
        <planeGeometry args={[4.1, 2.1]} />
        <meshBasicMaterial 
          color="#00ffaa"
          transparent
          opacity={0.3}
        />
      </mesh>

      {/* Agent ID header */}
      <Text
        position={[0, 0.7, 0.01]}
        fontSize={0.08}
        color="#00ffff"
        anchorX="center"
        anchorY="middle"
      >
        {`Agent ${agentId.slice(-4)} • Processing ErrorNode ${errorNodeId.slice(-4)}`}
      </Text>

      {/* Current thought */}
      <Text
        position={[0, 0.2, 0.01]}
        fontSize={0.06}
        color="#00ff88"
        anchorX="center"
        anchorY="middle"
        maxWidth={3.5}
      >
        {`→ ${thoughts[currentThought]}`}
      </Text>

      {/* Progress indicator */}
      <Text
        position={[0, -0.3, 0.01]}
        fontSize={0.05}
        color="#ffaa00"
        anchorX="center"
        anchorY="middle"
      >
        {`Step ${currentThought + 1}/${thoughts.length}`}
      </Text>

      {/* Thinking particles */}
      <mesh position={[0, -0.6, 0.01]}>
        <bufferGeometry>
          <bufferAttribute
            attach="attributes-position"
            count={15}
            array={new Float32Array(
              Array.from({ length: 45 }, (_, i) => {
                if (i % 3 === 1) return Math.random() * 0.2 + 0.1;
                return (Math.random() - 0.5) * 1.5;
              })
            )}
            itemSize={3}
          />
        </bufferGeometry>
        <pointsMaterial 
          size={0.02} 
          color="#00ff88" 
          transparent={true}
          opacity={0.7}
        />
      </mesh>
    </group>
  );
}