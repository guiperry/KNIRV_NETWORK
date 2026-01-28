import React from "react";
import { useFrame } from "@react-three/fiber";
import { useRef, useMemo } from "react";
import * as THREE from "three";
import GameLights from "./GameLights";
import KnirvGraph from "./KnirvGraph";
import CameraController from "./CameraController";
import MountainTerrain from "./MountainTerrain";
import { useKnirvana } from "./stores/useKnirvana";
import { useThemeStore } from "../../stores/useThemeStore";
import DeployAnimation from "./DeployAnimation";

export default function GameScene() {
  const sceneRef = useRef<THREE.Group>(null);
  const { selectedAgent } = useKnirvana();
  const { themeMode } = useThemeStore();

  // Theme-aware colors
  const themeColors = useMemo(() => {
    switch (themeMode) {
      case 'light':
        return {
          floor: '#f0f0f0',
          grid: '#888888',
          particles: '#333333',
          floorOpacity: 0.3,
          gridOpacity: 0.2,
          particleOpacity: 0.4
        };
      case 'light-blue':
        return {
          floor: '#001133',
          grid: '#0066aa',
          particles: '#00ffff',
          floorOpacity: 0.4,
          gridOpacity: 0.3,
          particleOpacity: 0.6
        };
      default: // dark
        return {
          floor: '#000008',
          grid: '#4400ff',
          particles: '#00ffff',
          floorOpacity: 0.4,
          gridOpacity: 0.3,
          particleOpacity: 0.5
        };
    }
  }, [themeMode]);

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

      {/* TRON grid floor - theme aware */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.1, 0]} receiveShadow>
        <planeGeometry args={[100, 100, 50, 50]} />
        <meshStandardMaterial
          color={themeColors.floor}
          wireframe={true}
          transparent={true}
          opacity={themeColors.floorOpacity}
        />
      </mesh>
      
      {/* Grid lines - theme aware */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, 0, 0]}>
        <planeGeometry args={[100, 100, 25, 25]} />
        <meshBasicMaterial 
          color={themeColors.grid}
          transparent
          opacity={themeColors.gridOpacity}
          wireframe
        />
      </mesh>
      
      {/* Mountain terrain - static geometric shapes at edges */}
      <MountainTerrain />
      
      {/* KNIRV Graph representation */}
      <KnirvGraph />
      
      {/* Ambient particles - theme aware */}
      <points>
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
          color={themeColors.particles}
          transparent={true}
          opacity={themeColors.particleOpacity}
          sizeAttenuation={true}
        />
      </points>

      {/* Deploy animations */}
      <DeployAnimation />
    </group>
  );
}