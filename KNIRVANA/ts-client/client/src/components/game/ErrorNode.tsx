import { useRef, useState } from "react";
import { useFrame } from "@react-three/fiber";
import { Text } from "@react-three/drei";
import * as THREE from "three";
import { useKnirvana } from "../../lib/stores/useKnirvana";

interface ErrorNodeProps {
  id: string;
  position: { x: number; y: number; z: number };
  type: string;
  difficulty: number;
  bounty: number;
  isBeingSolved: boolean;
  progress: number;
}

export default function ErrorNode({
  id,
  position,
  type,
  difficulty,
  bounty,
  isBeingSolved,
  progress
}: ErrorNodeProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const [hovered, setHovered] = useState(false);
  const { selectErrorNode, selectedErrorNode } = useKnirvana();

  const isSelected = selectedErrorNode === id;

  useFrame((state) => {
    if (meshRef.current) {
      // Pulsing animation
      const pulse = Math.sin(state.clock.elapsedTime * 3) * 0.1 + 1;
      meshRef.current.scale.setScalar(pulse);
      
      // Rotation based on difficulty
      meshRef.current.rotation.y += difficulty * 0.01;
      
      // Color interpolation based on progress
      const material = meshRef.current.material as THREE.MeshStandardMaterial;
      if (isBeingSolved) {
        const progressColor = new THREE.Color().lerpColors(
          new THREE.Color("#ff4444"),
          new THREE.Color("#ffaa00"),
          progress
        );
        material.color = progressColor;
      }
    }
  });

  const handleClick = (event: any) => {
    // Prevent event from bubbling to camera controller
    event.stopPropagation();
    console.log(`ErrorNode ${id} clicked`);
    selectErrorNode(id);
  };

  const getDifficultyColor = () => {
    if (difficulty < 0.3) return "#ff4444";
    if (difficulty < 0.7) return "#ffaa00";
    return "#ff0088";
  };

  const getScale = () => {
    return 0.5 + difficulty * 0.5;
  };

  return (
    <group position={[position.x, 0.5, position.z]}>
      <mesh
        ref={meshRef}
        onClick={handleClick}
        onPointerOver={() => setHovered(true)}
        onPointerOut={() => setHovered(false)}
        scale={getScale()}
        castShadow
      >
        <icosahedronGeometry args={[0.5, 1]} />
        <meshStandardMaterial
          color={getDifficultyColor()}
          emissive={getDifficultyColor()}
          emissiveIntensity={isSelected ? 0.3 : 0.1}
          transparent={true}
          opacity={hovered ? 0.9 : 0.7}
          wireframe={!isBeingSolved}
        />
      </mesh>
      
      {/* Progress ring for nodes being solved */}
      {isBeingSolved && (
        <mesh rotation={[Math.PI / 2, 0, 0]} position={[0, -0.1, 0]}>
          <ringGeometry args={[0.6, 0.8, 32, 1, 0, Math.PI * 2 * progress]} />
          <meshBasicMaterial color="#00ffff" transparent opacity={0.8} />
        </mesh>
      )}
      
      {/* Floating text labels */}
      {(hovered || isSelected) && (
        <Text
          position={[0, 1.5, 0]}
          fontSize={0.2}
          color="#00ffff"
          anchorX="center"
          anchorY="middle"
        >
          {`${type}\nDifficulty: ${Math.round(difficulty * 100)}%\nBounty: ${bounty} NRN`}
        </Text>
      )}
      
      {/* Glowing base */}
      <mesh position={[0, -0.8, 0]} rotation={[Math.PI / 2, 0, 0]}>
        <ringGeometry args={[0.2, 1.2, 32]} />
        <meshBasicMaterial 
          color={getDifficultyColor()} 
          transparent 
          opacity={0.2}
        />
      </mesh>
    </group>
  );
}
