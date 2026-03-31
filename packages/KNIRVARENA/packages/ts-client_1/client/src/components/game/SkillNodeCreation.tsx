import React, { useState } from 'react';
import { useFrame } from '@react-three/fiber';
import { Text } from '@react-three/drei';
import * as THREE from 'three';

interface SkillNodeCreationProps {
  position: [number, number, number];
  skillName: string;
  creatorAgent: string;
  onCreationComplete: () => void;
}

export default function SkillNodeCreation({
  position,
  skillName,
  creatorAgent,
  onCreationComplete
}: SkillNodeCreationProps) {
  const [creationProgress, setCreationProgress] = useState(0);
  const [particles, setParticles] = useState<number[]>([]);

  useFrame((state, delta) => {
    // Progress creation animation
    setCreationProgress(prev => {
      const newProgress = Math.min(1, prev + delta * 0.3);
      if (newProgress >= 1) {
        setTimeout(onCreationComplete, 1000);
      }
      return newProgress;
    });

    // Generate creation particles
    if (creationProgress < 1) {
      setParticles(prev => {
        const newParticles = [];
        for (let i = 0; i < 30; i++) {
          const angle = (i / 30) * Math.PI * 2;
          const radius = 1 + Math.sin(state.clock.elapsedTime * 3) * 0.3;
          newParticles.push(
            Math.cos(angle) * radius,
            Math.sin(state.clock.elapsedTime * 2 + i) * 0.5,
            Math.sin(angle) * radius
          );
        }
        return newParticles;
      });
    }
  });

  return (
    <group position={position}>
      {/* Central core being formed */}
      <mesh>
        <sphereGeometry args={[0.5 * creationProgress, 16, 16]} />
        <meshStandardMaterial
          color="#00ffaa"
          emissive="#00ffaa"
          emissiveIntensity={0.5 + creationProgress * 0.5}
          transparent
          opacity={0.8}
        />
      </mesh>

      {/* Outer energy shell */}
      <mesh scale={1.5 + creationProgress * 0.5}>
        <sphereGeometry args={[0.8, 12, 12]} />
        <meshBasicMaterial
          color="#88ffaa"
          transparent
          opacity={0.2}
          wireframe
        />
      </mesh>

      {/* Formation particles */}
      {particles.length > 0 && (
        <mesh>
          <bufferGeometry>
            <bufferAttribute
              attach="attributes-position"
              count={particles.length / 3}
              array={new Float32Array(particles)}
              itemSize={3}
            />
          </bufferGeometry>
          <pointsMaterial 
            size={0.04} 
            color="#00ff88" 
            transparent={true}
            opacity={0.8}
          />
        </mesh>
      )}

      {/* Creation progress ring */}
      <mesh rotation={[Math.PI / 2, 0, 0]}>
        <ringGeometry args={[1.0, 1.2, 32, 1, 0, Math.PI * 2 * creationProgress]} />
        <meshBasicMaterial color="#00ffff" transparent opacity={0.8} />
      </mesh>

      {/* Creation info */}
      <group position={[0, 2, 0]}>
        <Text
          fontSize={0.15}
          color="#00ffaa"
          anchorX="center"
          anchorY="bottom"
        >
          Creating SkillNode
        </Text>
        
        <Text
          position={[0, -0.3, 0]}
          fontSize={0.1}
          color="#88ffaa"
          anchorX="center"
          anchorY="top"
        >
          {skillName}
        </Text>
        
        <Text
          position={[0, -0.5, 0]}
          fontSize={0.08}
          color="#aaffaa"
          anchorX="center"
          anchorY="top"
        >
          {`Created by Agent ${creatorAgent.slice(-4)}`}
        </Text>
        
        <Text
          position={[0, -0.7, 0]}
          fontSize={0.08}
          color="#ffaa88"
          anchorX="center"
          anchorY="top"
        >
          {`Progress: ${Math.round(creationProgress * 100)}%`}
        </Text>
      </group>

      {/* Energy beams during creation */}
      {creationProgress < 1 && (
        <group>
          <mesh position={[0, 1.5, 0]}>
            <cylinderGeometry args={[0.02, 0.02, 1]} />
            <meshBasicMaterial 
              color="#00ffff"
              transparent
              opacity={0.6}
            />
          </mesh>
          <mesh position={[0, -1.5, 0]}>
            <cylinderGeometry args={[0.02, 0.02, 1]} />
            <meshBasicMaterial 
              color="#00ffff"
              transparent
              opacity={0.6}
            />
          </mesh>
        </group>
      )}

      {/* Success glow when complete */}
      {creationProgress >= 1 && (
        <mesh>
          <sphereGeometry args={[2, 16, 16]} />
          <meshBasicMaterial 
            color="#00ffaa"
            transparent
            opacity={0.1}
          />
        </mesh>
      )}
    </group>
  );
}