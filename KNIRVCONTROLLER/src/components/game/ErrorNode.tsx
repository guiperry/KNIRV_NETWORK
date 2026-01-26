import { useRef } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";
import { ErrorNode as ErrorNodeType } from "./stores/useKnirvana";

interface ErrorNodeProps {
  node: ErrorNodeType;
  isSelected: boolean;
  onSelect: () => void;
}

export default function ErrorNode({ node, isSelected, onSelect }: ErrorNodeProps) {
  const meshRef = useRef<THREE.Mesh>(null);

  useFrame((state) => {
    if (meshRef.current) {
      // Pulsing effect for active nodes
      const time = state.clock.elapsedTime;
      const scale = isSelected ? 1.2 + Math.sin(time * 4) * 0.1 : 1;
      meshRef.current.scale.setScalar(scale);
      
      // Rotation animation
      meshRef.current.rotation.y += 0.01;
    }
  });

  const color = node.isBeingSolved ? "#ff6600" : "#ff0000";
  const progress = node.progress;

  return (
    <mesh
      ref={meshRef}
      position={[node.position.x, node.position.y + 1, node.position.z]}
      onClick={onSelect}
      castShadow
      receiveShadow
    >
      {/* Main error node sphere */}
      <sphereGeometry args={[0.8, 16, 16]} />
      <meshStandardMaterial
        color={color}
        emissive={color}
        emissiveIntensity={isSelected ? 0.5 : 0.2}
        roughness={0.3}
        metalness={0.7}
      />
      
      {/* Progress indicator */}
      {node.isBeingSolved && (
        <mesh position={[0, 1.2, 0]}>
          <cylinderGeometry args={[0.3, 0.3, 2, 8]} />
          <meshBasicMaterial
            color="#00ff00"
            transparent
            opacity={0.6}
          />
        </mesh>
      )}
      
      {/* Error glow effect */}
      <pointLight
        position={[0, 0.5, 0]}
        color={color}
        intensity={isSelected ? 1 : 0.5}
        distance={5}
        decay={2}
      />
    </mesh>
  );
}