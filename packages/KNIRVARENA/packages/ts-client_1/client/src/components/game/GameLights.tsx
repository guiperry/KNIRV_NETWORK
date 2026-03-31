import { useRef } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";

export default function GameLights() {
  const lightRef = useRef<THREE.DirectionalLight>(null);
  const spotLightRef = useRef<THREE.SpotLight>(null);

  useFrame((state) => {
    // Animate main light for dynamic shadows
    if (lightRef.current) {
      lightRef.current.position.x = Math.sin(state.clock.elapsedTime * 0.5) * 10;
      lightRef.current.position.z = Math.cos(state.clock.elapsedTime * 0.5) * 10;
    }
    
    // Animate spot light for dramatic effect
    if (spotLightRef.current) {
      spotLightRef.current.angle = 0.3 + Math.sin(state.clock.elapsedTime * 2) * 0.1;
    }
  });

  return (
    <>
      {/* Ambient light for overall illumination */}
      <ambientLight intensity={0.2} color="#001144" />
      
      {/* Main directional light */}
      <directionalLight
        ref={lightRef}
        position={[10, 15, 5]}
        intensity={1}
        color="#4488ff"
        castShadow
        shadow-mapSize-width={2048}
        shadow-mapSize-height={2048}
        shadow-camera-far={50}
        shadow-camera-left={-20}
        shadow-camera-right={20}
        shadow-camera-top={20}
        shadow-camera-bottom={-20}
      />
      
      {/* Spot light for dramatic effect */}
      <spotLight
        ref={spotLightRef}
        position={[0, 20, 0]}
        angle={0.3}
        penumbra={0.5}
        intensity={0.8}
        color="#00ffff"
        castShadow
      />
      
      {/* Point lights for node illumination */}
      <pointLight position={[-10, 5, -10]} intensity={0.5} color="#ff4488" />
      <pointLight position={[10, 5, 10]} intensity={0.5} color="#44ff88" />
      <pointLight position={[-10, 5, 10]} intensity={0.5} color="#8844ff" />
      <pointLight position={[10, 5, -10]} intensity={0.5} color="#ffaa44" />
      
      {/* Hemisphere light for sky effect */}
      <hemisphereLight
        args={["#001122", "#000011", 0.3]}
      />
    </>
  );
}
