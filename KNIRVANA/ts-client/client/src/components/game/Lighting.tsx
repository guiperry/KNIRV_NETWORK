import React from 'react';

export const Lighting: React.FC = () => {
  return (
    <>
      {/* Ambient light for overall scene illumination */}
      <ambientLight intensity={0.3} color="#001122" />
      
      {/* Main directional light */}
      <directionalLight
        position={[50, 50, 50]}
        intensity={0.8}
        color="#00ffff"
        castShadow
        shadow-mapSize={[2048, 2048]}
        shadow-camera-near={1}
        shadow-camera-far={100}
        shadow-camera-left={-50}
        shadow-camera-right={50}
        shadow-camera-top={50}
        shadow-camera-bottom={-50}
      />
      
      {/* Secondary light for depth */}
      <directionalLight
        position={[-30, 30, -30]}
        intensity={0.4}
        color="#0088ff"
      />
      
      {/* Point light for dynamic effects */}
      <pointLight
        position={[0, 20, 0]}
        intensity={1}
        color="#00ffff"
        distance={100}
      />
      
      {/* Spotlight for dramatic effect */}
      <spotLight
        position={[0, 50, 0]}
        angle={0.15}
        penumbra={1}
        intensity={0.5}
        color="#ffffff"
        castShadow
      />
    </>
  );
};
