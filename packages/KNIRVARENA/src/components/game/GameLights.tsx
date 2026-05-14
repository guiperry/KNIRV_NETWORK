import { Canvas } from "@react-three/fiber";
import { Suspense, useEffect } from "react";

export default function GameLights() {
  return (
    <>
      {/* Ambient lighting for overall scene visibility */}
      <ambientLight intensity={0.3} color="#000033" />
      
      {/* Primary directional light with shadows */}
      <directionalLight
        position={[20, 40, 20]}
        intensity={1.2}
        color="#4400ff"
        castShadow
        shadow-mapSize-width={2048}
        shadow-mapSize-height={2048}
        shadow-camera-far={100}
        shadow-camera-left={-50}
        shadow-camera-right={50}
        shadow-camera-top={50}
        shadow-camera-bottom={-50}
      />
      
      {/* Secondary directional light for fill */}
      <directionalLight
        position={[-20, 20, -20]}
        intensity={0.6}
        color="#00ffff"
      />
      
      {/* Cyan point lights for cyberpunk feel */}
      <pointLight
        position={[10, 10, 10]}
        intensity={0.8}
        color="#00ffff"
        distance={30}
        decay={2}
      />
      
      <pointLight
        position={[-10, 10, -10]}
        intensity={0.8}
        color="#ff00ff"
        distance={30}
        decay={2}
      />
      
      {/* Purple spotlights for dramatic effect */}
      <spotLight
        position={[0, 50, 0]}
        intensity={0.5}
        color="#8800ff"
        angle={Math.PI / 6}
        penumbra={0.5}
        distance={100}
        decay={2}
      />
    </>
  );
}