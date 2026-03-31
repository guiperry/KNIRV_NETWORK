import React, { useMemo } from 'react';
import * as THREE from 'three';

export const GridFloor: React.FC = () => {
  const gridHelper = useMemo(() => {
    const size = 100;
    const divisions = 50;
    return new THREE.GridHelper(size, divisions, '#00ffff', '#003344');
  }, []);

  return (
    <>
      {/* Main floor plane */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, 0, 0]} receiveShadow>
        <planeGeometry args={[200, 200]} />
        <meshStandardMaterial 
          color="#000011" 
          transparent 
          opacity={0.8}
          metalness={0.8}
          roughness={0.2}
        />
      </mesh>
      
      {/* Grid lines */}
      <primitive object={gridHelper} />
      
      {/* Animated grid lines for TRON effect */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, 0.01, 0]}>
        <planeGeometry args={[200, 200]} />
        <meshBasicMaterial 
          color="#00ffff"
          transparent
          opacity={0.1}
          wireframe
        />
      </mesh>
    </>
  );
};
