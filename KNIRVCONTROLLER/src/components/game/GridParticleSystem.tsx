import * as THREE from "three";
import React, { useRef, useMemo, useEffect, useRef as useRefCallback } from "react";
import { useFrame } from "@react-three/fiber";
import { useThemeStore } from "../../stores/useThemeStore";
import { KnirvanaErrorNode } from "../../services/KnirvanaBridgeService";

interface Particle {
  x: number;
  z: number;
  velocityX: number;
  velocityZ: number;
  isSwirling: boolean;
  swirlCenterX?: number;
  swirlCenterZ?: number;
  swirlAngle: number;
  lastMoveTime: number;
}

export default function GridParticleSystem({ errorNodes }: { errorNodes: KnirvanaErrorNode[] }) {
  const particleRef = useRef<THREE.Points>(null);
  const { themeMode } = useThemeStore();
  const particlesRef = useRef<Particle[]>([]);
  const positionsRef = useRef<Float32Array>();
  const colorsRef = useRef<Float32Array>();

  // Theme-based colors
  const themeColors = useMemo(() => {
    switch (themeMode) {
      case 'light':
        return { particle: '#4a5568', opacity: 0.6 };
      case 'light-blue':
        return { particle: '#3182ce', opacity: 0.7 };
      default: // dark
        return { particle: '#00ffff', opacity: 0.8 };
    }
  }, [themeMode]);

  // Initialize particles
  const particleCount = 150; // Reduced for performance
  const gridSize = 10;
  const arenaSize = 100;

  useMemo(() => {
    const initialParticles: Particle[] = [];
    const baseColor = new THREE.Color(themeColors.particle);

    for (let i = 0; i < particleCount; i++) {
      const isHorizontalLine = Math.random() < 0.5;
      let x = 0, z = 0;

      if (isHorizontalLine) {
        z = Math.floor((Math.random() - 0.5) * (arenaSize / gridSize)) * gridSize;
        x = (Math.random() - 0.5) * arenaSize;
      } else {
        x = Math.floor((Math.random() - 0.5) * (arenaSize / gridSize)) * gridSize;
        z = (Math.random() - 0.5) * arenaSize;
      }

      initialParticles.push({
        x,
        z,
        velocityX: Math.random() < 0.5 ? 1 : -1,
        velocityZ: Math.random() < 0.5 ? 1 : -1,
        isSwirling: false,
        swirlAngle: Math.random() * Math.PI * 2,
        lastMoveTime: Date.now()
      });
    }

    particlesRef.current = initialParticles;
  }, [themeColors.particle, particleCount]);

  // Create static geometry
  const particleGeometry = useMemo(() => {
    const positions = new Float32Array(particleCount * 3);
    const colors = new Float32Array(particleCount * 3);
    const baseColor = new THREE.Color(themeColors.particle);

    for (let i = 0; i < particleCount; i++) {
      positions[i * 3] = 0;
      positions[i * 3 + 1] = 0.1;
      positions[i * 3 + 2] = 0;
      
      colors[i * 3] = baseColor.r;
      colors[i * 3 + 1] = baseColor.g;
      colors[i * 3 + 2] = baseColor.b;
    }

    positionsRef.current = positions;
    colorsRef.current = colors;

    const geometry = new THREE.BufferGeometry();
    geometry.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
    geometry.setAttribute('color', new THREE.Float32BufferAttribute(colors, 3));
    
    return geometry;
  }, [themeColors.particle, particleCount]);

  // Update particles
  useFrame((state, delta) => {
    if (!particleRef.current || !positionsRef.current || !colorsRef.current) return;

    const currentTime = Date.now();
    const moveInterval = 100; // Slower updates for performance
    const positions = positionsRef.current;
    const colors = colorsRef.current;
    const particles = particlesRef.current;

    particles.forEach((particle, i) => {
      // Check for nearby error nodes
      if (!particle.isSwirling) {
        const nearbyErrorNode = errorNodes.find(node => {
          const distance = Math.sqrt(
            Math.pow(particle.x - node.position.x, 2) + 
            Math.pow(particle.z - node.position.z, 2)
          );
          return distance <= 8.0;
        });

        if (nearbyErrorNode) {
          particle.isSwirling = true;
          particle.swirlCenterX = nearbyErrorNode.position.x;
          particle.swirlCenterZ = nearbyErrorNode.position.z;
        }
      }

      if (particle.isSwirling && particle.swirlCenterX !== undefined) {
        // Swirling movement
        particle.swirlAngle += delta * 2;
        const swirlRadius = 2.5;
        
        particle.x = particle.swirlCenterX + Math.cos(particle.swirlAngle) * swirlRadius;
        particle.z = particle.swirlCenterZ + Math.sin(particle.swirlAngle) * swirlRadius;

        // Random chance to stop swirling
        if (Math.random() < 0.008) {
          particle.isSwirling = false;
          particle.swirlCenterX = undefined;
          particle.swirlCenterZ = undefined;
        }

        // Red color when swirling
        colors[i * 3] = 1.0;
        colors[i * 3 + 1] = 0.4;
        colors[i * 3 + 2] = 0.4;
      } else {
        // Grid-based movement
        if (currentTime - particle.lastMoveTime > moveInterval) {
          particle.lastMoveTime = currentTime;

          if (Math.random() < 0.15) {
            const dirs = [1, -1];
            particle.velocityX = dirs[Math.floor(Math.random() * 2)];
            particle.velocityZ = dirs[Math.floor(Math.random() * 2)];
          }

          const nextX = particle.x + particle.velocityX * 2;
          const nextZ = particle.z + particle.velocityZ * 2;

          if (nextX < -arenaSize / 2 || nextX > arenaSize / 2) {
            particle.velocityX *= -1;
          }
          if (nextZ < -arenaSize / 2 || nextZ > arenaSize / 2) {
            particle.velocityZ *= -1;
          }

          particle.x = Math.max(-arenaSize / 2, Math.min(arenaSize / 2, nextX));
          particle.z = Math.max(-arenaSize / 2, Math.min(arenaSize / 2, nextZ));

          // Snap to grid lines
          if (Math.abs(particle.velocityX) > 0) {
            particle.z = Math.round(particle.z / gridSize) * gridSize;
          } else {
            particle.x = Math.round(particle.x / gridSize) * gridSize;
          }
        }

        // Normal theme color
        const color = new THREE.Color(themeColors.particle);
        colors[i * 3] = color.r;
        colors[i * 3 + 1] = color.g;
        colors[i * 3 + 2] = color.b;
      }

      // Update position buffer
      positions[i * 3] = particle.x;
      positions[i * 3 + 2] = particle.z;
    });

    // Mark for update
    particleRef.current.geometry.attributes.position.needsUpdate = true;
    particleRef.current.geometry.attributes.color.needsUpdate = true;
  });

  return (
    <points 
      ref={particleRef} 
      geometry={particleGeometry}
      raycast={() => null}
    >
      <pointsMaterial
        size={0.3}
        transparent
        opacity={themeColors.opacity}
        sizeAttenuation
        vertexColors
        depthWrite={false}
        blending={THREE.AdditiveBlending}
      />
    </points>
  );
}