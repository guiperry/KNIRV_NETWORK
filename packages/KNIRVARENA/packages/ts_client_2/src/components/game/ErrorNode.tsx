import { useRef } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";
import { ErrorNode as ErrorNodeType, useKnirvana } from "./stores/useKnirvana";

interface ErrorNodeProps {
  node: ErrorNodeType;
  isSelected: boolean;
  onSelect: () => void;
}

export default function ErrorNode({ node, isSelected, onSelect }: ErrorNodeProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const ringRef = useRef<THREE.Mesh>(null);
  const isAnalyzing = useKnirvana(state => state.isAnalyzing);

  useFrame((state) => {
    if (meshRef.current) {
      // Pulsing effect for active nodes
      const time = state.clock.elapsedTime;
      const baseScale = isAnalyzing ? 1.3 : 1;
      const scale = isSelected ? baseScale + 0.2 + Math.sin(time * 4) * 0.1 : baseScale;
      meshRef.current.scale.setScalar(scale);

      // Rotation animation
      meshRef.current.rotation.y += 0.01;
    }

    // Animate the analyze ring
    if (ringRef.current && isAnalyzing) {
      const time = state.clock.elapsedTime;
      ringRef.current.rotation.z = time * 2;
      ringRef.current.scale.setScalar(1 + Math.sin(time * 3) * 0.15);
    }
  });

  const color = node.isBeingSolved ? "#ff6600" : "#ff0000";
  const progress = node.progress;

  return (
    <group position={[node.position.x, node.position.y + 1, node.position.z]}>
      <mesh
        ref={meshRef}
        onClick={onSelect}
        castShadow
        receiveShadow
      >
        {/* Main error node sphere */}
        <sphereGeometry args={[0.8, 16, 16]} />
        <meshStandardMaterial
          color={color}
          emissive={color}
          emissiveIntensity={isAnalyzing ? 0.8 : (isSelected ? 0.5 : 0.2)}
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
          intensity={isAnalyzing ? 2 : (isSelected ? 1 : 0.5)}
          distance={isAnalyzing ? 8 : 5}
          decay={2}
        />
      </mesh>

      {/* Analyze mode - red pulsing ring around error nodes */}
      {isAnalyzing && (
        <mesh ref={ringRef} rotation={[Math.PI / 2, 0, 0]}>
          <ringGeometry args={[1.5, 1.8, 32]} />
          <meshBasicMaterial
            color="#ff0000"
            transparent
            opacity={0.7}
            side={THREE.DoubleSide}
          />
        </mesh>
      )}

      {/* Analyze mode - outer glow ring */}
      {isAnalyzing && (
        <mesh rotation={[Math.PI / 2, 0, 0]} position={[0, -0.5, 0]}>
          <ringGeometry args={[2.0, 2.3, 32]} />
          <meshBasicMaterial
            color="#ff3333"
            transparent
            opacity={0.4}
            side={THREE.DoubleSide}
          />
        </mesh>
      )}
    </group>
  );
}