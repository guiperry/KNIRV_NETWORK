import { useFrame } from "@react-three/fiber";
import { useRef, useEffect } from "react";
import * as THREE from "three";
import GameLights from "./GameLights";
import KnirvGraph from "./KnirvGraph";
import CameraController from "./CameraController";
import { useKnirvana } from "../../lib/stores/useKnirvana";

export default function GameScene() {
  const sceneRef = useRef<THREE.Group>(null);
  const { gameTime, selectedAgent } = useKnirvana();



  useFrame((state, delta) => {
    // Update game time
    useKnirvana.getState().updateGameTime(delta);

    // Subtle scene rotation for dynamic feel (slower when agent is selected)
    if (sceneRef.current) {
      const rotationSpeed = selectedAgent ? 0.01 : 0.02;
      sceneRef.current.rotation.y += delta * rotationSpeed;
    }

    // Ensure scene is always visible
    if (sceneRef.current) {
      sceneRef.current.visible = true;
    }
  });

  return (
    <group ref={sceneRef}>
      <GameLights />
      <CameraController />



      {/* Dark TRON grid floor */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.1, 0]} receiveShadow>
        <planeGeometry args={[100, 100, 50, 50]} />
        <meshStandardMaterial
          color="#000008"
          wireframe={true}
          transparent={true}
          opacity={0.4}
        />
      </mesh>
      
      {/* Purple grid lines */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, 0, 0]}>
        <planeGeometry args={[100, 100, 25, 25]} />
        <meshBasicMaterial 
          color="#4400ff"
          transparent
          opacity={0.3}
          wireframe
        />
      </mesh>
      
      {/* KNIRV Graph representation */}
      <KnirvGraph />
      
      {/* Ambient particles */}
      <mesh>
        <bufferGeometry>
          <bufferAttribute
            attach="attributes-position"
            count={1000}
            array={new Float32Array(
              Array.from({ length: 3000 }, () => (Math.random() - 0.5) * 200)
            )}
            itemSize={3}
          />
        </bufferGeometry>
        <pointsMaterial 
          size={0.1} 
          color="#00ffff" 
          transparent={true}
          opacity={0.5}
          sizeAttenuation={true}
        />
      </mesh>
    </group>
  );
}
