import * as THREE from "three";
import React, { useRef, useMemo } from "react";
import { useThemeStore } from "../../stores/useThemeStore";

export default function MountainTerrain() {
  const mountainRef = useRef<THREE.Points>(null);
  const { themeMode } = useThemeStore();

  // Theme-based colors
  const themeColors = useMemo(() => {
    switch (themeMode) {
      case 'light':
        return {
          mountain: '#ffffff',
          opacity: 0.3
        };
      case 'light-blue':
        return {
          mountain: '#00cccc',
          opacity: 0.4
        };
      default: // dark
        return {
          mountain: '#00ffff',
          opacity: 0.5
        };
    }
  }, [themeMode]);

  // Create static mountain particles at arena edges only
  const mountainParticles = useMemo(() => {
    const positions: number[] = [];
    const colors: number[] = [];
    const particleCount = 800; // More particles for denser mountain effect

    // Color for particles (same as mountain color)
    const color = new THREE.Color(themeColors.mountain);
    
    for (let i = 0; i < particleCount; i++) {
      // Position particles only at arena edges (±50 to ±60 units)
      const edge = Math.floor(Math.random() * 4); // 0: top, 1: right, 2: bottom, 3: left
      let x = 0, z = 0;
      
      switch (edge) {
        case 0: // Top edge
          x = (Math.random() - 0.5) * 100;
          z = -50 - Math.random() * 10; // -50 to -60
          break;
        case 1: // Right edge
          x = 50 + Math.random() * 10; // 50 to 60
          z = (Math.random() - 0.5) * 100;
          break;
        case 2: // Bottom edge
          x = (Math.random() - 0.5) * 100;
          z = 50 + Math.random() * 10; // 50 to 60
          break;
        case 3: // Left edge
          x = -50 - Math.random() * 10; // -50 to -60
          z = (Math.random() - 0.5) * 100;
          break;
      }
      
      // Y position for mountain height (5 to 25 units tall)
      const y = Math.random() * 20 + 5;
      
      positions.push(x, y, z);
      colors.push(color.r, color.g, color.b);
    }

    const geometry = new THREE.BufferGeometry();
    geometry.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
    geometry.setAttribute('color', new THREE.Float32BufferAttribute(colors, 3));
    
    return geometry;
  }, [themeColors.mountain]);

  return (
    <points ref={mountainRef} geometry={mountainParticles}>
      <pointsMaterial
        size={0.8} // Larger size for mountain effect
        color={themeColors.mountain}
        transparent={true}
        opacity={themeColors.opacity}
        sizeAttenuation={true}
        vertexColors={true}
      />
    </points>
  );
}