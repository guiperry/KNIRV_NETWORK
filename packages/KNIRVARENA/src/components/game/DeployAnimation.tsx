import React, { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import { useKnirvana } from './stores/useKnirvana';
import * as THREE from 'three';
import { useThemeStore } from '../../stores/useThemeStore';

export default function DeployAnimation() {
  const meshRef = useRef<THREE.Mesh>(null);
  const { deployAnimations } = useKnirvana();
  const { themeMode } = useThemeStore();

  // Theme-aware colors
  const colors = useMemo(() => {
    switch (themeMode) {
      case 'light':
        return {
          main: '#3b82f6',
          glow: '#93c5fd',
          trail: '#bfdbfe'
        };
      case 'light-blue':
        return {
          main: '#00ffff',
          glow: '#0099ff',
          trail: '#0066ff'
        };
      default: // dark
        return {
          main: '#00ffff',
          glow: '#0066ff',
          trail: '#003366'
        };
    }
  }, [themeMode]);

  useFrame((state, delta) => {
    const currentTime = Date.now();

    deployAnimations.forEach(anim => {
      const elapsed = currentTime - anim.startTime;
      const progress = Math.min(elapsed / anim.duration, 1);

      // Easing function - bounce effect
      const easeProgress = 1 - Math.pow(1 - progress, 3);

      // Calculate current position with arc
      const arcHeight = 8;
      const x = anim.startPosition.x + (anim.endPosition.x - anim.startPosition.x) * easeProgress;
      const z = anim.startPosition.z + (anim.endPosition.z - anim.startPosition.z) * easeProgress;
      const y = anim.startPosition.y + (anim.endPosition.y - anim.startPosition.y) * easeProgress + 
                Math.sin(easeProgress * Math.PI) * arcHeight;

      // Find and update the animation mesh
      if (meshRef.current && meshRef.current.userData.animationId === anim.id) {
        meshRef.current.position.set(x, y, z);
        
        // Scale animation
        const scale = 1 + Math.sin(easeProgress * Math.PI) * 0.3;
        meshRef.current.scale.set(scale, scale, scale);
        
        // Rotation animation
        meshRef.current.rotation.x += delta * 2;
        meshRef.current.rotation.y += delta * 3;
        
        // Opacity animation
        const material = meshRef.current.material as THREE.MeshBasicMaterial;
        if (material) {
          material.opacity = easeProgress < 0.9 ? 1 : 1 - (easeProgress - 0.9) * 10;
        }
      }
    });
  });

  return (
    <group>
      {deployAnimations.map(anim => {
        const elapsed = Date.now() - anim.startTime;
        const progress = Math.min(elapsed / anim.duration, 1);

        // Initial position
        const initialX = anim.startPosition.x;
        const initialZ = anim.startPosition.z;
        const initialY = anim.startPosition.y;

        return (
          <mesh
            key={anim.id}
            ref={meshRef}
            position={[initialX, initialY, initialZ]}
            userData={{ animationId: anim.id }}
          >
            {/* Agent body */}
            <icosahedronGeometry args={[0.8, 1]} />
            <meshBasicMaterial 
              color={colors.main}
              transparent
              opacity={1}
              wireframe
            />
            
            {/* Inner core */}
            <mesh position={[0, 0, 0]}>
              <sphereGeometry args={[0.4, 8, 8]} />
              <meshBasicMaterial 
                color={colors.glow}
                transparent
                opacity={0.8}
              />
            </mesh>

            {/* Glow effect */}
            <mesh position={[0, 0, 0]}>
              <sphereGeometry args={[1.2, 8, 8]} />
              <meshBasicMaterial 
                color={colors.trail}
                transparent
                opacity={0.3}
                wireframe
              />
            </mesh>

            {/* Trailing particles */}
            {Array.from({ length: 3 }).map((_, i) => (
              <mesh
                key={`particle-${i}`}
                position={[
                  -i * 0.5, 
                  Math.sin(i) * 0.2, 
                  Math.cos(i) * 0.3
                ]}
              >
                <sphereGeometry args={[0.1, 4, 4]} />
                <meshBasicMaterial 
                  color={colors.glow}
                  transparent
                  opacity={1 - (i * 0.3)}
                />
              </mesh>
            ))}
          </mesh>
        );
      })}
    </group>
  );
}
