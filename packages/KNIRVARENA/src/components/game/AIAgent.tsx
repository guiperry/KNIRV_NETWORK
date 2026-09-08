/* eslint-disable react/no-unknown-property */

import React, { useRef, useEffect } from "react";
import { useFrame } from "@react-three/fiber";
import { useGLTF, useAnimations } from "@react-three/drei";
import * as THREE from "three";
import { clone } from "three/examples/jsm/utils/SkeletonUtils.js";
import { Agent as AgentType, useKnirvana } from "./stores/useKnirvana";

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
  // A skeleton must be cloned with its mesh so bots can play different clips.
  const clonedScene = React.useMemo(() => clone(scene), [scene]);
  const { actions } = useAnimations(animations, groupRef);
  const activeAction = useRef<THREE.AnimationAction | null>(null);
  const targetNode = useKnirvana(s => s.errorNodes.find(n => n.id === agent.target));
  const destination = useRef(new THREE.Vector3());
  const heading = useRef(new THREE.Quaternion());
  const up = React.useMemo(() => new THREE.Vector3(0, 1, 0), []);

  useEffect(() => {
    const name = isStaged ? 'Idle' : agent.status === 'moving' ? 'Run'
      : agent.status === 'working' ? 'Work' : 'Idle';
    const next = actions[name];
    if (!next || activeAction.current === next) return;
    const previous = activeAction.current;
    next.reset().setEffectiveTimeScale(1).setEffectiveWeight(1).fadeIn(0.18).play();
    previous?.fadeOut(0.18);
    activeAction.current = next;
  }, [agent.status, isStaged, actions]);

  useFrame((_, delta) => {
    const group = groupRef.current;
    if (!group || isStaged || !targetNode) return;
    const dx = targetNode.position.x - group.position.x;
    const dz = targetNode.position.z - group.position.z;
    const distance = Math.hypot(dx, dz);
    if (distance > 0.001) {
      heading.current.setFromAxisAngle(up, Math.atan2(dx, dz));
      group.quaternion.rotateTowards(heading.current, delta * 8);
    }
    if (agent.status !== 'moving') return;
    // Stop beside the node, with both hands facing it. Clips run in place;
    // world travel belongs to the game so it remains frame-rate independent.
    const stopDistance = 1.4;
    const step = Math.min(Math.max(0, distance - stopDistance), delta * 2.8);
    if (distance > 0.001) {
      destination.current.set(dx / distance * step, 0, dz / distance * step);
      group.position.add(destination.current);
    }
    if (distance - step <= stopDistance + 0.01) {
      const position = { x: group.position.x, y: agent.position.y, z: group.position.z };
      useKnirvana.setState(s => ({ agents: s.agents.map(a =>
        a.id === agent.id && a.target === targetNode.id && a.status === 'moving'
          ? { ...a, position, status: 'working' as const } : a
      ) }));
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

  // Staged agents are displayed much larger at the arena edge
  const modelScale = isStaged ? 3.0 : 0.5;

  // Source feet are at y=-1; place them on the grid at either display scale.
  const primitiveYOffset = modelScale - agent.position.y;

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
