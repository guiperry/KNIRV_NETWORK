import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface ElectricShockProps {
  radius: number;
}

const ElectricShock = ({ radius }: ElectricShockProps) => {
  const shockRef = useRef<THREE.Points>(null);
  const materialRef = useRef<THREE.PointsMaterial>(null);

  // Generate shock particles
  const { positions, colors } = useMemo(() => {
    const particleCount = 50;
    const positions = new Float32Array(particleCount * 3);
    const colors = new Float32Array(particleCount * 3);
    
    for (let i = 0; i < particleCount; i++) {
      const angle = (i / particleCount) * Math.PI * 2;
      positions[i * 3] = Math.cos(angle) * radius;
      positions[i * 3 + 1] = Math.sin(angle) * radius;
      positions[i * 3 + 2] = 0;
      
      // Electric blue to white gradient
      colors[i * 3] = 0.5 + Math.random() * 0.5; // R
      colors[i * 3 + 1] = 0.8 + Math.random() * 0.2; // G  
      colors[i * 3 + 2] = 1; // B
    }
    
    return { positions, colors };
  }, [radius]);

  useFrame((state) => {
    const time = state.clock.elapsedTime;
    
    if (shockRef.current && materialRef.current) {
      // Move shock around the ring
      const shockPosition = (time * 0.5) % (Math.PI * 2);
      
      // Update visibility based on random intervals
      const shouldShow = Math.sin(time * 3 + radius) > 0.7;
      const wasVisible = materialRef.current.opacity > 0;
      materialRef.current.opacity = shouldShow ? 0.8 : 0;
      
      // Electric shock visual effect only (no audio)
      
      // Rotate the shock
      shockRef.current.rotation.z = shockPosition;
    }
  });

  return (
    <points ref={shockRef}>
      <bufferGeometry>
        <bufferAttribute
          attach="attributes-position"
          count={positions.length / 3}
          array={positions}
          itemSize={3}
        />
        <bufferAttribute
          attach="attributes-color"
          count={colors.length / 3}
          array={colors}
          itemSize={3}
        />
      </bufferGeometry>
      <pointsMaterial
        ref={materialRef}
        size={0.1}
        transparent
        opacity={0.8}
        blending={THREE.AdditiveBlending}
        vertexColors={true}
      />
    </points>
  );
};

export default ElectricShock;
