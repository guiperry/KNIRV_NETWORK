import React, { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

export default function Environment() {
  const gridRef = useRef<THREE.Group>(null);
  const particlesRef = useRef<THREE.Points>(null);
  const glowRingsRef = useRef<THREE.Group>(null);

  // Generate ambient particle system
  const particleSystem = useMemo(() => {
    const particleCount = 2000;
    const positions = new Float32Array(particleCount * 3);
    const colors = new Float32Array(particleCount * 3);
    
    for (let i = 0; i < particleCount; i++) {
      const i3 = i * 3;
      
      // Random positions in a large sphere
      positions[i3] = (Math.random() - 0.5) * 200;
      positions[i3 + 1] = Math.random() * 30 + 1;
      positions[i3 + 2] = (Math.random() - 0.5) * 200;
      
      // Cyan-blue color variations
      colors[i3] = Math.random() * 0.5; // Red
      colors[i3 + 1] = 0.8 + Math.random() * 0.2; // Green 
      colors[i3 + 2] = 1; // Blue
    }
    
    return { positions, colors };
  }, []);

  // Generate procedural glow rings
  const glowRings = useMemo(() => {
    const rings = [];
    for (let i = 0; i < 8; i++) {
      const radius = 10 + i * 8;
      const height = Math.random() * 2;
      rings.push({ radius, height, rotation: Math.random() * Math.PI * 2 });
    }
    return rings;
  }, []);

  useFrame((state) => {
    const time = state.clock.elapsedTime;
    
    // Animate grid
    if (gridRef.current) {
      gridRef.current.rotation.y = time * 0.01;
      gridRef.current.position.y = Math.sin(time * 0.5) * 0.2;
    }
    
    // Animate particles
    if (particlesRef.current) {
      particlesRef.current.rotation.y = time * 0.02;
      
      // Update particle positions for floating effect
      const positions = particlesRef.current.geometry.attributes.position.array as Float32Array;
      for (let i = 1; i < positions.length; i += 3) {
        positions[i] += Math.sin(time + i) * 0.01;
      }
      particlesRef.current.geometry.attributes.position.needsUpdate = true;
    }
    
    // Animate glow rings
    if (glowRingsRef.current) {
      glowRingsRef.current.children.forEach((ring, index) => {
        ring.rotation.y = time * (0.1 + index * 0.05);
        ring.position.y = Math.sin(time + index) * 0.5;
      });
    }
  });

  return (
    <group>
      {/* Main floor with TRON grid */}
      <group ref={gridRef}>
        {/* Base floor */}
        <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.1, 0]} receiveShadow>
          <planeGeometry args={[200, 200]} />
          <meshStandardMaterial 
            color="#000005" 
            transparent 
            opacity={0.95}
            metalness={0.9}
            roughness={0.1}
          />
        </mesh>
        
        {/* Grid lines - primary */}
        <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, 0, 0]}>
          <planeGeometry args={[200, 200, 40, 40]} />
          <meshBasicMaterial 
            color="#4400ff"
            transparent
            opacity={0.25}
            wireframe
          />
        </mesh>
        
        {/* Grid lines - secondary */}
        <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, 0.01, 0]}>
          <planeGeometry args={[200, 200, 20, 20]} />
          <meshBasicMaterial 
            color="#0088ff"
            transparent
            opacity={0.2}
            wireframe
          />
        </mesh>
      </group>

      {/* Ambient particle system */}
      <points ref={particlesRef}>
        <bufferGeometry>
          <bufferAttribute
            attach="attributes-position"
            count={particleSystem.positions.length / 3}
            array={particleSystem.positions}
            itemSize={3}
          />
          <bufferAttribute
            attach="attributes-color"
            count={particleSystem.colors.length / 3}
            array={particleSystem.colors}
            itemSize={3}
          />
        </bufferGeometry>
        <pointsMaterial 
          size={0.1} 
          transparent={true}
          opacity={0.6}
          sizeAttenuation={true}
          vertexColors={true}
          blending={THREE.AdditiveBlending}
        />
      </points>

      {/* Procedural glow rings */}
      <group ref={glowRingsRef}>
        {glowRings.map((ring, index) => (
          <mesh 
            key={index}
            rotation={[Math.PI / 2, 0, ring.rotation]} 
            position={[0, ring.height, 0]}
          >
            <ringGeometry args={[ring.radius, ring.radius + 0.5, 64]} />
            <meshBasicMaterial 
              color={`hsl(${180 + index * 20}, 100%, 50%)`}
              transparent 
              opacity={0.1 + Math.sin(index) * 0.05}
              side={THREE.DoubleSide}
            />
          </mesh>
        ))}
      </group>

      {/* Central energy core */}
      <group position={[0, 5, 0]}>
        <mesh>
          <sphereGeometry args={[2, 32, 32]} />
          <meshBasicMaterial 
            color="#00ffff"
            transparent
            opacity={0.1}
            wireframe
          />
        </mesh>
        
        {/* Core glow */}
        <pointLight 
          position={[0, 0, 0]} 
          intensity={1.5} 
          color="#00ffff" 
          distance={20}
        />
      </group>

      {/* Boundary walls */}
      {[-100, 100].map((x, i) => (
        <mesh key={`wall-x-${i}`} position={[x, 10, 0]}>
          <planeGeometry args={[200, 20]} />
          <meshBasicMaterial 
            color="#001122"
            transparent
            opacity={0.3}
            side={THREE.DoubleSide}
          />
        </mesh>
      ))}
      
      {[-100, 100].map((z, i) => (
        <mesh key={`wall-z-${i}`} position={[0, 10, z]} rotation={[0, Math.PI / 2, 0]}>
          <planeGeometry args={[200, 20]} />
          <meshBasicMaterial 
            color="#001122"
            transparent
            opacity={0.3}
            side={THREE.DoubleSide}
          />
        </mesh>
      ))}

      {/* Data streams - animated lines */}
      <group>
        {Array.from({ length: 12 }, (_, i) => {
          const angle = (i / 12) * Math.PI * 2;
          const radius = 30;
          return (
            <mesh 
              key={`stream-${i}`}
              position={[
                Math.cos(angle) * radius, 
                1, 
                Math.sin(angle) * radius
              ]}
              rotation={[0, angle, 0]}
            >
              <cylinderGeometry args={[0.02, 0.02, 15]} />
              <meshBasicMaterial 
                color="#44aaff"
                transparent
                opacity={0.6}
              />
            </mesh>
          );
        })}
      </group>
    </group>
  );
}
