import { useRef } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";
import { SkillNode as SkillNodeType } from "./stores/useKnirvana";

interface SkillNodeProps {
  node: SkillNodeType;
}

export default function SkillNode({ node }: SkillNodeProps) {
  const meshRef = useRef<THREE.Mesh>(null);

  useFrame((state) => {
    if (meshRef.current) {
      // Gentle floating animation
      const time = state.clock.elapsedTime;
      meshRef.current.position.y = node.position.y + Math.sin(time * 2 + parseFloat(node.id)) * 0.2;
      
      // Slow rotation
      meshRef.current.rotation.y += 0.005;
    }
  });

  return (
    <mesh
      ref={meshRef}
      position={[node.position.x, node.position.y + 0.5, node.position.z]}
      castShadow
      receiveShadow
    >
      {/* Skill node - diamond shape */}
      <octahedronGeometry args={[0.6, 0]} />
      <meshStandardMaterial
        color="#00ffff"
        emissive="#00ffff"
        emissiveIntensity={0.3}
        roughness={0.1}
        metalness={0.9}
      />
      
      {/* Skill glow effect */}
      <pointLight
        position={[0, 0, 0]}
        color="#00ffff"
        intensity={0.8}
        distance={4}
        decay={2}
      />
    </mesh>
  );
}