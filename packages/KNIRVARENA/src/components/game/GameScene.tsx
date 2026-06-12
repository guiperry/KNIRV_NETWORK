import React from "react";
import { useFrame, ThreeEvent } from "@react-three/fiber";
import { useRef, useMemo } from "react";
import * as THREE from "three";
import GameLights from "./GameLights";
import KnirvGraph from "./KnirvGraph";
import CameraController from "./CameraController";
import RewardAnchor3D from "./RewardAnchor3D";
import { useKnirvana } from "./stores/useKnirvana";
import { useThemeStore } from "../../stores/useThemeStore";
import DeployAnimation from "./DeployAnimation";
import GridParticleSystem from "./GridParticleSystem";
import AnchorStraighteningSequence from "./AnchorStraighteningSequence";
import ArenaGrid from "./ArenaGrid";

export default function GameScene() {
  const sceneRef = useRef<THREE.Group>(null);

  const { selectedAgent, rewardAnchors, isSculpting, errorNodes } = useKnirvana();
  const { themeMode } = useThemeStore();

  // Floor clicks do nothing — anchors are placed only via spike clicks on ErrorNode
  const handleFloorClick = (_event: ThreeEvent<MouseEvent>) => {};

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

    // Pause rotation while placing anchors so the spike positions stay stable
    if (sceneRef.current && !isSculpting) {
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

      {/* TRON grid floor — theme aware, sinks parabolically beneath
          error nodes whose validation ring has been committed */}
      <ArenaGrid
        floorColor={themeColors.floor}
        gridColor={themeColors.grid}
        floorOpacity={themeColors.floorOpacity}
        gridOpacity={themeColors.gridOpacity}
        onFloorClick={handleFloorClick}
      />
      
      {/* Grid Particle System - electrical pulses shooting through grid */}
      <GridParticleSystem key={themeMode} errorNodes={errorNodes} />

      {/* KNIRV Graph representation */}
      <KnirvGraph />

      {/* Reward Anchors */}
      {rewardAnchors.map(anchor => (
        <RewardAnchor3D key={anchor.id} anchor={anchor} />
      ))}

      {/* Deploy animations */}
      <DeployAnimation />

      {/* Anchor straightening sequence — agents run from staging edge to horizontal anchors */}
      <AnchorStraighteningSequence />
    </group>
  );
}