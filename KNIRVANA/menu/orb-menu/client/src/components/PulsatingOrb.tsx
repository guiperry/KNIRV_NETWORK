import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import { Points } from '@react-three/drei';
import * as THREE from 'three';

interface PulsatingOrbProps {
  onClick: () => void;
}

const PulsatingOrb = ({ onClick }: PulsatingOrbProps) => {
  const orbRef = useRef<THREE.Points>(null);
  const coreRef = useRef<THREE.Mesh>(null);

  // Generate particle positions for the orb
  const particlePositions = useMemo(() => {
    const positions = new Float32Array(1000 * 3);
    for (let i = 0; i < 1000; i++) {
      const radius = Math.random() * 2 + 0.5;
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(2 * Math.random() - 1);
      
      positions[i * 3] = radius * Math.sin(phi) * Math.cos(theta);
      positions[i * 3 + 1] = radius * Math.sin(phi) * Math.sin(theta);
      positions[i * 3 + 2] = radius * Math.cos(phi);
    }
    return positions;
  }, []);

  useFrame((state) => {
    const time = state.clock.elapsedTime;

    // Pulsating effect
    if (coreRef.current) {
      const pulse = 1 + Math.sin(time * 2) * 0.3;
      coreRef.current.scale.setScalar(pulse);
    }

    // Particle rotation
    if (orbRef.current) {
      orbRef.current.rotation.y = time * 0.5;
      orbRef.current.rotation.x = Math.sin(time * 0.3) * 0.2;
    }
  });

  const handleClick = () => {
    console.log('PulsatingOrb clicked!');
    console.log('Calling onClick callback...');
    onClick();
  };

  console.log('PulsatingOrb rendering...');

  return (
    <group>
      {/* Core orb */}
      <mesh
        ref={coreRef}
        position={[0, 0, 0]}
        onClick={handleClick}
        onPointerOver={() => document.body.style.cursor = 'pointer'}
        onPointerOut={() => document.body.style.cursor = 'default'}
      >
        <sphereGeometry args={[2, 32, 32]} />
        <meshBasicMaterial
          color="#00ccff"
          transparent={false}
          opacity={1.0}
        />
      </mesh>

      {/* Particle system */}
      <Points ref={orbRef}>
        <bufferGeometry>
          <bufferAttribute
            attach="attributes-position"
            count={1000}
            array={particlePositions}
            itemSize={3}
          />
        </bufferGeometry>
        <pointsMaterial
          color="#00ccff"
          size={0.05}
          sizeAttenuation
          transparent
          opacity={0.8}
          blending={THREE.AdditiveBlending}
        />
      </Points>

      {/* Outer glow */}
      <mesh>
        <sphereGeometry args={[2.5, 16, 16]} />
        <meshBasicMaterial 
          color="#00ccff" 
          transparent 
          opacity={0.1}
          side={THREE.BackSide}
        />
      </mesh>
    </group>
  );
};

export default PulsatingOrb;
