import { useRef } from "react";
import { useFrame, ThreeEvent } from "@react-three/fiber";
import * as THREE from "three";
import { RewardAnchor, useKnirvana } from "./stores/useKnirvana";

interface RewardAnchor3DProps {
  anchor: RewardAnchor;
}

export default function RewardAnchor3D({ anchor }: RewardAnchor3DProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const ringRef = useRef<THREE.Mesh>(null);
  const selectedRewardAnchor = useKnirvana(state => state.selectedRewardAnchor);
  const selectRewardAnchor = useKnirvana(state => state.selectRewardAnchor);

  const isSelected = selectedRewardAnchor === anchor.id;

  useFrame((state) => {
    const time = state.clock.elapsedTime;

    // Floating animation
    if (meshRef.current) {
      meshRef.current.position.y = anchor.position.y + Math.sin(time * 2) * 0.15;
      meshRef.current.rotation.y = time * 0.5;
    }

    // Ring animation when selected
    if (ringRef.current) {
      ringRef.current.rotation.z = time * 2;
      if (isSelected) {
        ringRef.current.scale.setScalar(1 + Math.sin(time * 4) * 0.1);
      }
    }
  });

  const handleClick = (e: ThreeEvent<MouseEvent>) => {
    e.stopPropagation();
    selectRewardAnchor(anchor.id);
  };

  return (
    <group position={[anchor.position.x, anchor.position.y, anchor.position.z]}>
      {/* Main anchor crystal/diamond shape */}
      <mesh
        ref={meshRef}
        onClick={handleClick}
        castShadow
      >
        <octahedronGeometry args={[0.4, 0]} />
        <meshStandardMaterial
          color="#ffcc00"
          emissive="#ffaa00"
          emissiveIntensity={isSelected ? 0.8 : 0.4}
          roughness={0.2}
          metalness={0.9}
          transparent
          opacity={0.9}
        />
      </mesh>

      {/* Selection ring */}
      <mesh
        ref={ringRef}
        rotation={[Math.PI / 2, 0, 0]}
        position={[0, 0, 0]}
      >
        <ringGeometry args={[0.6, 0.75, 6]} />
        <meshBasicMaterial
          color={isSelected ? "#00ff00" : "#ffcc00"}
          transparent
          opacity={isSelected ? 0.8 : 0.4}
          side={THREE.DoubleSide}
        />
      </mesh>

      {/* Outer glow ring when selected */}
      {isSelected && (
        <mesh rotation={[Math.PI / 2, 0, 0]} position={[0, -0.1, 0]}>
          <ringGeometry args={[0.9, 1.1, 6]} />
          <meshBasicMaterial
            color="#00ff00"
            transparent
            opacity={0.3}
            side={THREE.DoubleSide}
          />
        </mesh>
      )}

      {/* Vertical beam indicator */}
      <mesh position={[0, 1.5, 0]}>
        <cylinderGeometry args={[0.02, 0.02, 3, 8]} />
        <meshBasicMaterial
          color={isSelected ? "#00ff00" : "#ffcc00"}
          transparent
          opacity={isSelected ? 0.6 : 0.3}
        />
      </mesh>

      {/* Point light for glow effect */}
      <pointLight
        color="#ffcc00"
        intensity={isSelected ? 1.5 : 0.5}
        distance={5}
        decay={2}
      />
    </group>
  );
}
