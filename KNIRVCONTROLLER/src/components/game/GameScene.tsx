import React from "react";
import { useFrame, useThree } from "@react-three/fiber";
import { useRef, useMemo } from "react";
import * as THREE from "three";
import GameLights from "./GameLights";
import KnirvGraph from "./KnirvGraph";
import CameraController from "./CameraController";
import MountainTerrain from "./MountainTerrain";
import RewardAnchor3D from "./RewardAnchor3D";
import { useKnirvana } from "./stores/useKnirvana";
import { useThemeStore } from "../../stores/useThemeStore";
import DeployAnimation from "./DeployAnimation";

export default function GameScene() {
  const sceneRef = useRef<THREE.Group>(null);
  const { selectedAgent, rewardAnchors, isSculpting, selectedErrorNode, errorNodes, addRewardAnchor, setSculpting } = useKnirvana();
  const { themeMode } = useThemeStore();

  // Handle floor click for placing reward anchors in sculpt mode
  const handleFloorClick = (event: THREE.Event) => {
    if (!isSculpting) return;

    event.stopPropagation();
    const point = event.point;

    // Get linked error node data if one is selected
    const errorNodeData = selectedErrorNode ? errorNodes.find(n => n.id === selectedErrorNode) : null;
    const metadata = errorNodeData ? {
      logs: ['Error: Memory allocation failed', 'Stack trace at line 42'],
      traces: ['Component: DataProcessor', 'Method: processBatch'],
      severity: 'high',
      description: 'Memory leak detected in processing pipeline'
    } : undefined;

    const newAnchor = {
      id: `anchor-${Date.now()}`,
      position: { x: point.x, y: 0.5, z: point.z },
      weights: { w_c: 0.6, w_l: 0.3, w_s: 0.1 },
      constraints: '// Define constraints here',
      linkedErrorNode: selectedErrorNode || undefined,
      metadata
    };

    addRewardAnchor(newAnchor);
    setSculpting(false);
  };

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

      {/* TRON grid floor - theme aware, clickable for sculpt mode */}
      <mesh
        rotation={[-Math.PI / 2, 0, 0]}
        position={[0, -0.1, 0]}
        receiveShadow
        onClick={handleFloorClick}
      >
        <planeGeometry args={[100, 100, 50, 50]} />
        <meshStandardMaterial
          color={themeColors.floor}
          wireframe={true}
          transparent={true}
          opacity={themeColors.floorOpacity}
        />
      </mesh>

      {/* Solid floor for raycasting clicks */}
      <mesh
        rotation={[-Math.PI / 2, 0, 0]}
        position={[0, -0.11, 0]}
        onClick={handleFloorClick}
        visible={false}
      >
        <planeGeometry args={[100, 100]} />
        <meshBasicMaterial transparent opacity={0} />
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

      {/* Reward Anchors */}
      {rewardAnchors.map(anchor => (
        <RewardAnchor3D key={anchor.id} anchor={anchor} />
      ))}

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