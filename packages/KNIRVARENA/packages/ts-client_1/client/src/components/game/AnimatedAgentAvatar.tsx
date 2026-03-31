import React, { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import { useTexture } from '@react-three/drei';
import * as THREE from 'three';

interface AnimatedAgentAvatarProps {
  position: [number, number, number];
  type: string;
  status: 'idle' | 'moving' | 'working' | 'upgrading';
  efficiency: number;
  isSelected: boolean;
  onClick: () => void;
  onHover: (hovered: boolean) => void;
}

export default function AnimatedAgentAvatar({
  position,
  type,
  status,
  efficiency,
  isSelected,
  onClick,
  onHover
}: AnimatedAgentAvatarProps) {
  const groupRef = useRef<THREE.Group>(null);
  const avatarRef = useRef<THREE.Mesh>(null);
  const eyesRef = useRef<THREE.Group>(null);
  
  // Load the robot avatar texture
  const avatarTexture = useTexture('/textures/agent_avatar.png');
  
  // Get colors based on agent type and status
  const getAgentColors = useMemo(() => {
    const baseColors = {
      'Analyzer': { primary: '#ff6b6b', secondary: '#ff8e8e', eyes: '#ffaaaa' },
      'Optimizer': { primary: '#4ecdc4', secondary: '#6ee7df', eyes: '#8ff5f0' },
      'Synthesizer': { primary: '#45b7d1', secondary: '#6bc7e1', eyes: '#91d7f0' },
      'Debugger': { primary: '#f7ca18', secondary: '#f9d943', eyes: '#fbe96e' },
      'default': { primary: '#95a5a6', secondary: '#b8c7c8', eyes: '#dae5e6' }
    };
    
    const colors = baseColors[type as keyof typeof baseColors] || baseColors.default;
    
    // Modify colors based on status
    if (status === 'working') {
      return {
        primary: colors.primary,
        secondary: colors.secondary,
        eyes: '#00ffff', // Bright cyan when working
        intensity: 0.8
      };
    } else if (status === 'upgrading') {
      return {
        primary: colors.primary,
        secondary: colors.secondary,
        eyes: '#ff00ff', // Magenta when upgrading
        intensity: 0.6
      };
    } else if (status === 'moving') {
      return {
        primary: colors.primary,
        secondary: colors.secondary,
        eyes: '#00ff00', // Green when moving
        intensity: 0.4
      };
    }
    
    return {
      primary: colors.primary,
      secondary: colors.secondary,
      eyes: colors.eyes,
      intensity: 0.2
    };
  }, [type, status]);

  useFrame((state) => {
    if (!groupRef.current || !avatarRef.current || !eyesRef.current) return;

    const time = state.clock.elapsedTime;
    
    // Floating animation
    groupRef.current.position.y = position[1] + Math.sin(time * 2) * 0.1;
    
    // Rotation based on status
    if (status === 'working') {
      groupRef.current.rotation.y += 0.02;
      // Working head bob
      avatarRef.current.rotation.x = Math.sin(time * 4) * 0.1;
    } else if (status === 'moving') {
      // Walking animation
      avatarRef.current.rotation.z = Math.sin(time * 6) * 0.05;
    } else if (status === 'upgrading') {
      // Upgrading glow pulse
      const scale = 1 + Math.sin(time * 3) * 0.05;
      avatarRef.current.scale.setScalar(scale);
    }
    
    // Eye animation (blinking and looking)
    const blinkTime = Math.sin(time * 0.5);
    if (blinkTime > 0.95) {
      eyesRef.current.scale.y = 0.1; // Blink
    } else {
      eyesRef.current.scale.y = 1;
    }
    
    // Eyes look around when idle
    if (status === 'idle') {
      eyesRef.current.rotation.y = Math.sin(time * 0.3) * 0.2;
    }
  });

  return (
    <group 
      ref={groupRef}
      position={position}
      onClick={onClick}
      onPointerOver={() => onHover(true)}
      onPointerOut={() => onHover(false)}
    >
      {/* Main avatar body - make it much more visible */}
      <mesh ref={avatarRef} castShadow receiveShadow>
        <boxGeometry args={[1, 1.2, 0.8]} />
        <meshStandardMaterial
          color={getAgentColors.primary}
          emissive={getAgentColors.primary}
          emissiveIntensity={0.8}
          metalness={0.3}
          roughness={0.7}
        />
      </mesh>
      
      {/* Animated eyes */}
      <group ref={eyesRef} position={[0, 0.1, 0.21]}>
        {/* Left eye */}
        <mesh position={[-0.06, 0, 0]}>
          <boxGeometry args={[0.04, 0.03, 0.01]} />
          <meshStandardMaterial
            color={getAgentColors.eyes}
            emissive={getAgentColors.eyes}
            emissiveIntensity={2.0}
          />
        </mesh>
        
        {/* Right eye */}
        <mesh position={[0.06, 0, 0]}>
          <boxGeometry args={[0.04, 0.03, 0.01]} />
          <meshStandardMaterial
            color={getAgentColors.eyes}
            emissive={getAgentColors.eyes}
            emissiveIntensity={2.0}
          />
        </mesh>
      </group>
      
      {/* Selection ring */}
      {isSelected && (
        <mesh rotation={[Math.PI / 2, 0, 0]} position={[0, -0.8, 0]}>
          <ringGeometry args={[0.6, 0.8, 32]} />
          <meshBasicMaterial color="#00ffff" transparent opacity={0.6} />
        </mesh>
      )}
      
      {/* Status indicator halo */}
      <mesh position={[0, 1, 0]} rotation={[Math.PI / 2, 0, 0]}>
        <ringGeometry args={[0.4, 0.5, 16]} />
        <meshBasicMaterial 
          color={getAgentColors.primary}
          transparent 
          opacity={0.4 + efficiency * 0.4}
        />
      </mesh>
      
      {/* Efficiency indicator particles */}
      {efficiency > 0.7 && (
        <mesh position={[0, 0.8, 0]}>
          <bufferGeometry>
            <bufferAttribute
              attach="attributes-position"
              count={Math.floor(efficiency * 20)}
              array={new Float32Array(
                Array.from({ length: Math.floor(efficiency * 60) }, (_, i) => {
                  if (i % 3 === 1) return Math.random() * 0.5 + 0.3;
                  return (Math.random() - 0.5) * 1;
                })
              )}
              itemSize={3}
            />
          </bufferGeometry>
          <pointsMaterial 
            size={0.03} 
            color={getAgentColors.secondary} 
            transparent 
            opacity={0.8}
          />
        </mesh>
      )}
      
      {/* Working data stream effect */}
      {status === 'working' && (
        <group>
          <mesh position={[0, 1.5, 0]}>
            <cylinderGeometry args={[0.02, 0.02, 1]} />
            <meshBasicMaterial 
              color="#00ffff"
              transparent
              opacity={0.6}
            />
          </mesh>
          
          {/* Data bits flowing upward */}
          {[0.2, 0.4, 0.6, 0.8].map((height, i) => (
            <mesh key={i} position={[0, 1 + height, 0]}>
              <boxGeometry args={[0.04, 0.04, 0.04]} />
              <meshBasicMaterial 
                color="#00ffff"
                transparent
                opacity={0.8 - height * 0.8}
              />
            </mesh>
          ))}
        </group>
      )}
      
      {/* Upgrading energy field */}
      {status === 'upgrading' && (
        <mesh>
          <sphereGeometry args={[1.2, 16, 16]} />
          <meshBasicMaterial 
            color="#ff00ff"
            transparent
            opacity={0.15}
            wireframe
          />
        </mesh>
      )}
      
      {/* Moving trail effect */}
      {status === 'moving' && (
        <group position={[0, -0.5, 0]}>
          {[0, 0.2, 0.4].map((z, i) => (
            <mesh key={i} position={[0, 0, -z]} rotation={[Math.PI / 2, 0, 0]}>
              <ringGeometry args={[0.3, 0.4, 8]} />
              <meshBasicMaterial 
                color="#00ff00"
                transparent
                opacity={0.5 - i * 0.15}
              />
            </mesh>
          ))}
        </group>
      )}
    </group>
  );
}