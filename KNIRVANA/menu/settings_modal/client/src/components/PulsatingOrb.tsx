import { useRef, useMemo, useEffect } from 'react';
import { useFrame } from '@react-three/fiber';
import { Points } from '@react-three/drei';
import * as THREE from 'three';
import { useAudio } from '../lib/stores/useAudio';

interface PulsatingOrbProps {
  onClick: () => void;
}

const PulsatingOrb = ({ onClick }: PulsatingOrbProps) => {
  const orbRef = useRef<THREE.Points>(null);
  const coreRef = useRef<THREE.Mesh>(null);
  const { playOrbPulse, playExpansion, isInitialized, isMuted, toggleMute, startAmbientMusic } = useAudio();
  
  const lastPulseTime = useRef<number>(0);
  
  // Add debug logging
  useEffect(() => {
    console.log('PulsatingOrb mounted');
    return () => console.log('PulsatingOrb unmounted');
  }, []);

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
    
    // Pulsating effect with sound
    if (coreRef.current) {
      const pulse = 1 + Math.sin(time * 2) * 0.3;
      coreRef.current.scale.setScalar(pulse);
      
      // Play pulse sound every 2 seconds if initialized
      if (isInitialized && !isMuted && time - lastPulseTime.current > 2) {
        playOrbPulse();
        lastPulseTime.current = time;
      }
    }

    // Particle rotation
    if (orbRef.current) {
      orbRef.current.rotation.y = time * 0.5;
      orbRef.current.rotation.x = Math.sin(time * 0.3) * 0.2;
    }
  });

  const handleClick = () => {
    console.log('PulsatingOrb clicked');
    
    // Always try to unmute on first click
    if (isMuted) {
      console.log('Unmuting audio');
      toggleMute();
    }
    
    // Play expansion sound
    if (isInitialized && !isMuted) {
      playExpansion();
    }
    
    onClick();
  };

  return (
    <group onClick={handleClick} visible={true}>
      {/* Core orb */}
      <mesh ref={coreRef}>
        <sphereGeometry args={[1, 32, 32]} />
        <meshBasicMaterial 
          color="#00ccff" 
          transparent 
          opacity={0.6}
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
