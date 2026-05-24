/* eslint-disable react/no-unknown-property */

import React, { useRef, useEffect } from "react";
import { useFrame } from "@react-three/fiber";
import { useGLTF, useAnimations } from "@react-three/drei";
import * as THREE from "three";
import { Agent as AgentType } from "./stores/useKnirvana";

interface AIAgentProps {
  agent: AgentType;
  isSelected: boolean;
  onSelect: () => void;
  onStage?: () => void;
  isStaged?: boolean;
  stagingIndex?: number;
}

const GLB_PATH = `${import.meta.env.BASE_URL}assets/avatar/Green_Bot_Explorer.glb`;
useGLTF.preload(GLB_PATH);

export default function AIAgent({ agent, isSelected, onSelect, onStage, isStaged = false }: AIAgentProps) {
  const groupRef = useRef<THREE.Group>(null);
  const { scene, animations } = useGLTF(GLB_PATH);
  const { actions, names } = useAnimations(animations, groupRef);

  // Play idle animation by default if available
  useEffect(() => {
    if (names.length > 0 && actions[names[0]]) {
      actions[names[0]]?.play();
    }
  }, [actions, names]);

  // Handle status-based animations
  useEffect(() => {
    if (names.length > 0) {
      // Stop all current animations
      names.forEach(name => actions[name]?.stop());
      
      // Play appropriate animation based on status
      switch (agent.status) {
        case 'idle':
          // Play first animation (usually idle) or create floating effect
          if (actions[names[0]]) {
            actions[names[0]]?.play();
          }
          break;
        case 'working': {
          // Try to find a working/walk/run animation, fallback to first
          const workAnim = names.find(n => n.toLowerCase().includes('walk') || n.toLowerCase().includes('run') || n.toLowerCase().includes('work'));
          actions[workAnim || names[0]]?.play();
          break;
        }
        case 'upgrading': {
          // Try to find an upgrade/celebrate animation
          const upgradeAnim = names.find(n => n.toLowerCase().includes('celebrate') || n.toLowerCase().includes('dance') || n.toLowerCase().includes('upgrade'));
          actions[upgradeAnim || names[0]]?.play();
          break;
        }
        default:
          actions[names[0]]?.play();
      }
    }
  }, [agent.status, actions, names]);

  useFrame((state) => {
    if (groupRef.current) {
      const time = state.clock.elapsedTime;
      if (isStaged) {
        // Staged agents bob slowly at the arena edge
        groupRef.current.position.y = agent.position.y + Math.sin(time * 1.5) * 0.1;
      } else {
        const bounceHeight = agent.status === 'idle' ? Math.sin(time * 3) * 0.05 : 0;
        groupRef.current.position.y = agent.position.y + bounceHeight;
        if (agent.status === 'working') {
          groupRef.current.rotation.y += 0.02;
        }
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

  // Clone the scene to allow individual instances
  const clonedScene = React.useMemo(() => scene.clone(), [scene]);

  // Staged agents are displayed much larger at the arena edge
  const modelScale = isStaged ? 3.0 : 0.5;

  // At scale 3.0 the model center (waist) sits at the group origin.
  // Deployed agents work at group.y=1 with scale 0.5 — feet ~y=0.
  // Scale factor 6× means feet would sink 5 units underground when staged.
  // Offset the primitive upward by 5 units so feet stay on the grid.
  const primitiveYOffset = isStaged ? 5.0 : 0;

  return (
    <group
      ref={groupRef}
      position={[agent.position.x, agent.position.y, agent.position.z]}
      onClick={onSelect}
    >
      {/* 3D Model Avatar */}
      <primitive
        object={clonedScene}
        scale={modelScale}
        position={[0, primitiveYOffset, 0]}
        castShadow
        receiveShadow
      />
      
      {/* Status glow effect */}
      <pointLight
        position={[0, 0.5, 0]}
        color={getColor()}
        intensity={isSelected ? 1.5 : 0.8}
        distance={4}
        decay={2}
      />
      
      {/* Selection indicator ring */}
      {isSelected && (
        <mesh position={[0, 1.5, 0]} rotation={[Math.PI / 2, 0, 0]}>
          <ringGeometry args={[0.8, 1.0, 32]} />
          <meshBasicMaterial color="#00ff00" transparent opacity={0.8} side={THREE.DoubleSide} />
        </mesh>
      )}

      {/* Stage Button - visual indicator */}
      {isSelected && onStage && (
        <group position={[0, -0.8, 0]} onClick={(e) => { e.stopPropagation(); onStage(); }}>
          <mesh>
            <boxGeometry args={[0.6, 0.2, 0.1]} />
            <meshStandardMaterial color="#ff6b35" emissive="#ff6b35" emissiveIntensity={0.6} />
          </mesh>
          {/* Arrow indicator pointing up to agent */}
          <mesh position={[0, 0.2, 0]} rotation={[0, 0, Math.PI]}>
            <coneGeometry args={[0.12, 0.15, 4]} />
            <meshStandardMaterial color="#ff6b35" emissive="#ff6b35" emissiveIntensity={0.6} />
          </mesh>
        </group>
      )}
    </group>
  );
}
