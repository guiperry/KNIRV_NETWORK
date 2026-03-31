import { useRef, useState } from 'react';
import { useFrame } from '@react-three/fiber';
import { Text } from '@react-three/drei';
import * as THREE from 'three';
import { useKnirvana } from '../../lib/stores/useKnirvana';

interface SkillNodeProps {
  id: string;
  position: { x: number; y: number; z: number };
  name: string;
  creator: string;
  usageCount: number;
  category?: string;
  value?: number;
}

export default function SkillNode({
  id,
  position,
  name,
  creator,
  usageCount,
  category = 'General',
  value = 100
}: SkillNodeProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const auraRef = useRef<THREE.Mesh>(null);
  const [hovered, setHovered] = useState(false);
  const [isBeingUsed, setIsBeingUsed] = useState(false);

  const { selectAgent } = useKnirvana();

  useFrame((state) => {
    if (meshRef.current) {
      // Smooth rotation based on usage
      meshRef.current.rotation.y += (usageCount / 1000) * 0.01;
      meshRef.current.rotation.x = Math.sin(state.clock.elapsedTime * 2) * 0.1;
      
      // Pulsing scale based on value
      const pulse = 0.8 + Math.sin(state.clock.elapsedTime * 3) * 0.2;
      const valueScale = 1 + (value / 500);
      meshRef.current.scale.setScalar(pulse * valueScale);
    }
    
    if (auraRef.current) {
      // Aura rotation
      auraRef.current.rotation.z += 0.02;
      
      // Aura opacity based on usage
      const material = auraRef.current.material as THREE.MeshBasicMaterial;
      material.opacity = 0.2 + (usageCount / 100) * 0.3;
    }
  });

  const handleClick = () => {
    console.log(`SkillNode ${id} (${name}) clicked`);
    // Implement skill interaction logic
    setIsBeingUsed(true);
    setTimeout(() => setIsBeingUsed(false), 2000);
  };

  const getCategoryColor = () => {
    switch (category) {
      case 'Processing': return '#00ff88';
      case 'Analysis': return '#0088ff';
      case 'Validation': return '#8800ff';
      case 'Optimization': return '#ffaa00';
      default: return '#44ff88';
    }
  };

  const getValueIntensity = () => {
    return Math.min(1, value / 300);
  };

  return (
    <group position={[position.x, position.y + 0.5, position.z]}>
      {/* Main skill node */}
      <mesh
        ref={meshRef}
        onClick={handleClick}
        onPointerOver={() => setHovered(true)}
        onPointerOut={() => setHovered(false)}
        castShadow
      >
        <octahedronGeometry args={[0.4, 1]} />
        <meshStandardMaterial
          color={getCategoryColor()}
          emissive={getCategoryColor()}
          emissiveIntensity={hovered ? 0.4 : 0.2}
          metalness={0.8}
          roughness={0.1}
          transparent={true}
          opacity={0.9}
        />
      </mesh>

      {/* Aura ring */}
      <mesh 
        ref={auraRef}
        rotation={[Math.PI / 2, 0, 0]} 
        position={[0, -0.2, 0]}
      >
        <ringGeometry args={[0.6, 1.0, 32]} />
        <meshBasicMaterial 
          color={getCategoryColor()}
          transparent 
          opacity={0.3}
          side={THREE.DoubleSide}
        />
      </mesh>

      {/* Usage indicator particles */}
      {usageCount > 10 && (
        <mesh>
          <bufferGeometry>
            <bufferAttribute
              attach="attributes-position"
              count={Math.min(usageCount, 50)}
              array={new Float32Array(
                Array.from({ length: Math.min(usageCount, 50) * 3 }, (_, i) => {
                  if (i % 3 === 1) return Math.random() * 2 + 0.5;
                  return (Math.random() - 0.5) * 3;
                })
              )}
              itemSize={3}
            />
          </bufferGeometry>
          <pointsMaterial 
            size={0.03} 
            color={getCategoryColor()}
            transparent={true}
            opacity={0.8}
          />
        </mesh>
      )}

      {/* Skill info display */}
      {(hovered || isBeingUsed) && (
        <group position={[0, 1.2, 0]}>
          <Text
            fontSize={0.12}
            color="#00ffaa"
            anchorX="center"
            anchorY="bottom"
          >
            {name}
          </Text>
          <Text
            position={[0, -0.2, 0]}
            fontSize={0.08}
            color="#88ffaa"
            anchorX="center"
            anchorY="top"
          >
            {`${category} • ${usageCount} uses`}
          </Text>
          <Text
            position={[0, -0.35, 0]}
            fontSize={0.06}
            color="#aaffaa"
            anchorX="center"
            anchorY="top"
          >
            {`Created by: ${creator}`}
          </Text>
          <Text
            position={[0, -0.45, 0]}
            fontSize={0.06}
            color="#ffaa88"
            anchorX="center"
            anchorY="top"
          >
            {`Value: ${value} NRN`}
          </Text>
        </group>
      )}

      {/* Base glow */}
      <mesh position={[0, -0.8, 0]} rotation={[Math.PI / 2, 0, 0]}>
        <circleGeometry args={[1.5, 32]} />
        <meshBasicMaterial 
          color={getCategoryColor()} 
          transparent 
          opacity={0.1 * getValueIntensity()}
        />
      </mesh>

      {/* Being used effect */}
      {isBeingUsed && (
        <group>
          <mesh>
            <sphereGeometry args={[0.8, 16, 16]} />
            <meshBasicMaterial 
              color="#ffffff"
              transparent
              opacity={0.3}
              wireframe
            />
          </mesh>
          
          {/* Energy burst */}
          <mesh>
            <bufferGeometry>
              <bufferAttribute
                attach="attributes-position"
                count={100}
                array={new Float32Array(
                  Array.from({ length: 300 }, () => (Math.random() - 0.5) * 6)
                )}
                itemSize={3}
              />
            </bufferGeometry>
            <pointsMaterial 
              size={0.05} 
              color="#ffffff" 
              transparent={true}
              opacity={0.9}
              blending={THREE.AdditiveBlending}
            />
          </mesh>
        </group>
      )}

      {/* Connection ports */}
      {[0, 1, 2, 3].map((i) => {
        const angle = (i / 4) * Math.PI * 2;
        return (
          <mesh 
            key={`port-${i}`}
            position={[
              Math.cos(angle) * 0.8, 
              0, 
              Math.sin(angle) * 0.8
            ]}
          >
            <sphereGeometry args={[0.05]} />
            <meshBasicMaterial 
              color="#00aaff"
              transparent
              opacity={0.8}
            />
          </mesh>
        );
      })}
    </group>
  );
}
