import * as THREE from "three";
import { useRef, useMemo } from "react";
import { useFrame } from "@react-three/fiber";
import { useThemeStore } from "../../stores/useThemeStore";
import { ErrorNode } from "./stores/useKnirvana";
import { sampleGridDip } from "./gridDepthField";

interface Particle {
  x: number;
  z: number;
  vx: number;
  vz: number;
  isSwirling: boolean;
  swirlCenterX: number;
  swirlCenterZ: number;
  swirlAngle: number;
  pulseOffset: number;
}

interface Photon {
  lineIndex: number;
  isHorizontal: boolean;
  position: number;
  speed: number;
  flashPhase: number;
}

const PARTICLE_COUNT = 80;
const PHOTON_COUNT = 120;
const GRID_SIZE = 10;
const ARENA_SIZE = 100;
const SWIRL_RADIUS = 2.5;
const FLOOR_Y = 0.05; // All particles locked to floor level

export default function GridParticleSystem({ errorNodes }: { errorNodes: ErrorNode[] }) {
  const pointsRef = useRef<THREE.Points>(null);
  const photonsRef = useRef<THREE.Points>(null);
  const { themeMode } = useThemeStore();
  const particlesRef = useRef<Particle[]>([]);
  const photonsDataRef = useRef<Photon[]>([]);

  const themeColors = useMemo(() => {
    switch (themeMode) {
      case 'light':
        // Orange colors for visibility on white background
        return { 
          normal: new THREE.Color('#ff8800'), 
          swirl: new THREE.Color('#ff4400'),
          photon: new THREE.Color('#ffaa00')
        };
      case 'light-blue':
        return { 
          normal: new THREE.Color('#3182ce'), 
          swirl: new THREE.Color('#ff4444'),
          photon: new THREE.Color('#aaddff')
        };
      default:
        return { 
          normal: new THREE.Color('#00ffff'), 
          swirl: new THREE.Color('#ff0066'),
          photon: new THREE.Color('#ccffff')
        };
    }
  }, [themeMode]);

  // Initialize dynamic particles - recreate when theme changes
  const particleGeometry = useMemo(() => {
    const particles: Particle[] = [];
    const positions = new Float32Array(PARTICLE_COUNT * 3);
    const colors = new Float32Array(PARTICLE_COUNT * 3);
    
    // Get color based on current theme
    let initialColor: THREE.Color;
    if (themeMode === 'light') {
      initialColor = new THREE.Color('#ff8800'); // Orange for light theme
    } else if (themeMode === 'light-blue') {
      initialColor = new THREE.Color('#3182ce');
    } else {
      initialColor = new THREE.Color('#00ffff');
    }

    for (let i = 0; i < PARTICLE_COUNT; i++) {
      const isHorizontal = Math.random() < 0.5;
      let x = 0, z = 0;

      if (isHorizontal) {
        z = Math.floor((Math.random() - 0.5) * (ARENA_SIZE / GRID_SIZE)) * GRID_SIZE;
        x = (Math.random() - 0.5) * ARENA_SIZE;
      } else {
        x = Math.floor((Math.random() - 0.5) * (ARENA_SIZE / GRID_SIZE)) * GRID_SIZE;
        z = (Math.random() - 0.5) * ARENA_SIZE;
      }

      particles.push({
        x, z,
        vx: Math.random() < 0.5 ? 1 : -1,
        vz: Math.random() < 0.5 ? 1 : -1,
        isSwirling: false,
        swirlCenterX: 0,
        swirlCenterZ: 0,
        swirlAngle: Math.random() * Math.PI * 2,
        pulseOffset: Math.random() * Math.PI * 2
      });

      positions[i * 3] = x;
      positions[i * 3 + 1] = FLOOR_Y;
      positions[i * 3 + 2] = z;

      colors[i * 3] = initialColor.r;
      colors[i * 3 + 1] = initialColor.g;
      colors[i * 3 + 2] = initialColor.b;
    }

    particlesRef.current = particles;

    const geom = new THREE.BufferGeometry();
    geom.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    geom.setAttribute('color', new THREE.BufferAttribute(colors, 3));
    return geom;
  }, [themeMode]); // Recreate on theme change

  // Initialize photon particles - recreate when theme changes
  const photonGeometry = useMemo(() => {
    const photons: Photon[] = [];
    const positions = new Float32Array(PHOTON_COUNT * 3);
    const colors = new Float32Array(PHOTON_COUNT * 3);
    
    // Get color based on current theme
    let initialColor: THREE.Color;
    if (themeMode === 'light') {
      initialColor = new THREE.Color('#ffaa00'); // Bright orange for light theme
    } else if (themeMode === 'light-blue') {
      initialColor = new THREE.Color('#aaddff');
    } else {
      initialColor = new THREE.Color('#ccffff');
    }

    const gridLines = 21;

    for (let i = 0; i < PHOTON_COUNT; i++) {
      const isHorizontal = Math.random() < 0.5;
      const lineIndex = Math.floor(Math.random() * gridLines);

      photons.push({
        lineIndex,
        isHorizontal,
        position: (Math.random() - 0.5) * ARENA_SIZE,
        speed: 2.0 + Math.random() * 3.0,
        flashPhase: Math.random() * Math.PI * 2
      });

      if (isHorizontal) {
        positions[i * 3] = (Math.random() - 0.5) * ARENA_SIZE;
        positions[i * 3 + 1] = FLOOR_Y;
        positions[i * 3 + 2] = (lineIndex - 10) * GRID_SIZE;
      } else {
        positions[i * 3] = (lineIndex - 10) * GRID_SIZE;
        positions[i * 3 + 1] = FLOOR_Y;
        positions[i * 3 + 2] = (Math.random() - 0.5) * ARENA_SIZE;
      }

      colors[i * 3] = initialColor.r;
      colors[i * 3 + 1] = initialColor.g;
      colors[i * 3 + 2] = initialColor.b;
    }

    photonsDataRef.current = photons;

    const geom = new THREE.BufferGeometry();
    geom.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    geom.setAttribute('color', new THREE.BufferAttribute(colors, 3));
    return geom;
  }, [themeMode]); // Recreate on theme change

  // Higher brightness multiplier for light theme to make orange visible on white
  const isLightTheme = themeMode === 'light';
  const brightnessMultiplier = isLightTheme ? 2.0 : 1.0;
  
  // Larger particles for light theme to improve visibility on white background
  const particleSize = isLightTheme ? 0.2 : 0.12;
  const photonSize = isLightTheme ? 0.28 : 0.18;

  // Update loop
  useFrame((state) => {
    const time = state.clock.elapsedTime;

    // Update dynamic particles
    if (pointsRef.current) {
      const positions = pointsRef.current.geometry.attributes.position.array as Float32Array;
      const colors = pointsRef.current.geometry.attributes.color.array as Float32Array;
      const particles = particlesRef.current;
      const globalPulse = 0.7 + Math.sin(time * 2) * 0.2;

      for (let i = 0; i < PARTICLE_COUNT; i++) {
        const p = particles[i];
        const idx = i * 3;

        // Check distance to error nodes
        if (!p.isSwirling && errorNodes.length > 0) {
          for (const node of errorNodes) {
            const dx = p.x - node.position.x;
            const dz = p.z - node.position.z;
            const distSq = dx * dx + dz * dz;
            
            if (distSq <= 64) {
              p.isSwirling = true;
              p.swirlCenterX = node.position.x;
              p.swirlCenterZ = node.position.z;
              break;
            }
          }
        }

        if (p.isSwirling) {
          // Swirl beneath error node - keep y locked to floor
          p.swirlAngle += 0.08;
          const radius = SWIRL_RADIUS + Math.sin(p.swirlAngle * 2) * 0.3;
          
          p.x = p.swirlCenterX + Math.cos(p.swirlAngle) * radius;
          p.z = p.swirlCenterZ + Math.sin(p.swirlAngle) * radius;

          if (Math.random() < 0.008) {
            p.isSwirling = false;
            const angle = Math.random() * Math.PI * 2;
            p.vx = Math.cos(angle) > 0 ? 1 : -1;
            p.vz = Math.sin(angle) > 0 ? 1 : -1;
          }

          const brightness = (1.0 + Math.sin(time * 3 + p.pulseOffset) * 0.3) * brightnessMultiplier;
          colors[idx] = themeColors.swirl.r * brightness;
          colors[idx + 1] = themeColors.swirl.g * brightness;
          colors[idx + 2] = themeColors.swirl.b * brightness;

          positions[idx] = p.x;
          positions[idx + 1] = FLOOR_Y - sampleGridDip(p.x, p.z); // Ride the grid surface
          positions[idx + 2] = p.z;
          
        } else {
          // Grid movement
          if (Math.random() < 0.1) {
            if (Math.random() < 0.15) {
              if (Math.random() < 0.5) {
                p.vx = p.vx === 0 ? (Math.random() < 0.5 ? 1 : -1) : 0;
              } else {
                p.vz = p.vz === 0 ? (Math.random() < 0.5 ? 1 : -1) : 0;
              }
            }

            const speed = 0.8;
            p.x += p.vx * speed;
            p.z += p.vz * speed;

            if (p.x < -ARENA_SIZE / 2 || p.x > ARENA_SIZE / 2) p.vx *= -1;
            if (p.z < -ARENA_SIZE / 2 || p.z > ARENA_SIZE / 2) p.vz *= -1;

            p.x = Math.max(-ARENA_SIZE / 2, Math.min(ARENA_SIZE / 2, p.x));
            p.z = Math.max(-ARENA_SIZE / 2, Math.min(ARENA_SIZE / 2, p.z));

            if (p.vx !== 0) {
              p.z = Math.round(p.z / GRID_SIZE) * GRID_SIZE;
            } else if (p.vz !== 0) {
              p.x = Math.round(p.x / GRID_SIZE) * GRID_SIZE;
            }
          }

          const brightness = (0.6 + Math.sin(time * 2 + p.pulseOffset) * 0.2 * globalPulse) * brightnessMultiplier;
          colors[idx] = themeColors.normal.r * brightness;
          colors[idx + 1] = themeColors.normal.g * brightness;
          colors[idx + 2] = themeColors.normal.b * brightness;
          
          positions[idx] = p.x;
          positions[idx + 1] = FLOOR_Y - sampleGridDip(p.x, p.z); // Ride the grid surface
          positions[idx + 2] = p.z;
        }
      }

      pointsRef.current.geometry.attributes.position.needsUpdate = true;
      pointsRef.current.geometry.attributes.color.needsUpdate = true;
    }

    // Update photon particles
    if (photonsRef.current) {
      const positions = photonsRef.current.geometry.attributes.position.array as Float32Array;
      const colors = photonsRef.current.geometry.attributes.color.array as Float32Array;
      const photons = photonsDataRef.current;

      for (let i = 0; i < PHOTON_COUNT; i++) {
        const ph = photons[i];
        const idx = i * 3;

        ph.position += ph.speed * 0.5;

        if (ph.position > ARENA_SIZE / 2) {
          ph.position = -ARENA_SIZE / 2;
          ph.flashPhase = Math.random() * Math.PI * 2;
        } else if (ph.position < -ARENA_SIZE / 2) {
          ph.position = ARENA_SIZE / 2;
          ph.flashPhase = Math.random() * Math.PI * 2;
        }

        if (ph.isHorizontal) {
          positions[idx] = ph.position;
          positions[idx + 2] = (ph.lineIndex - 10) * GRID_SIZE;
        } else {
          positions[idx] = (ph.lineIndex - 10) * GRID_SIZE;
          positions[idx + 2] = ph.position;
        }
        positions[idx + 1] = FLOOR_Y - sampleGridDip(positions[idx], positions[idx + 2]); // Ride the grid surface

        const flash = Math.sin(time * 8 + ph.flashPhase);
        const baseBrightness = flash > 0.5 ? 1.5 + flash * 0.5 : 0.3;
        const brightness = baseBrightness * brightnessMultiplier;
        
        colors[idx] = themeColors.photon.r * brightness;
        colors[idx + 1] = themeColors.photon.g * brightness;
        colors[idx + 2] = themeColors.photon.b * brightness;
      }

      photonsRef.current.geometry.attributes.position.needsUpdate = true;
      photonsRef.current.geometry.attributes.color.needsUpdate = true;
    }
  });

  return (
    <>
      <points
        ref={pointsRef}
        geometry={particleGeometry}
        raycast={() => null}
        frustumCulled={false}
      >
        <pointsMaterial
          size={particleSize}
          transparent
          opacity={0.85}
          sizeAttenuation
          vertexColors
          depthWrite={false}
          blending={THREE.AdditiveBlending}
        />
      </points>
      
      <points
        ref={photonsRef}
        geometry={photonGeometry}
        raycast={() => null}
        frustumCulled={false}
      >
        <pointsMaterial
          size={photonSize}
          transparent
          opacity={0.95}
          sizeAttenuation
          vertexColors
          depthWrite={false}
          blending={THREE.AdditiveBlending}
        />
      </points>
    </>
  );
}
