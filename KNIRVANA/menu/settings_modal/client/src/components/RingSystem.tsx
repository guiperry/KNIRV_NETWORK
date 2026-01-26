import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import { Text } from '@react-three/drei';
import { animated } from '@react-spring/three';
import ElectricShock from './ElectricShock';
import ElectricNetwork from './ElectricNetwork';
import * as THREE from 'three';
import { useAudio } from '../lib/stores/useAudio';

interface RingSystemProps {
  onServiceClick: (service: string) => void;
  opacity: any;
}

const services = [
  { name: 'KNIRVTESTNET', ring: 0, angle: 0 },
  { name: 'KNIRVSDK', ring: 0, angle: Math.PI / 2 },
  { name: 'KNIRVROUTER', ring: 0, angle: Math.PI },
  { name: 'KNIRVORACLE', ring: 0, angle: -Math.PI / 2 },
  { name: 'KNIRVNEXUS', ring: 1, angle: 0 },
  { name: 'KNIRVGRAPH', ring: 1, angle: Math.PI / 3 },
  { name: 'KNIRVGATEWAY', ring: 1, angle: 2 * Math.PI / 3 },
  { name: 'KNIRVCORTEX', ring: 1, angle: Math.PI },
  { name: 'KNIRVCONTROLLER', ring: 1, angle: -2 * Math.PI / 3 },
  { name: 'KNIRVCLI', ring: 1, angle: -Math.PI / 3 },
  { name: 'KNIRVCHAIN', ring: 2, angle: 0 },
];

const ringRadii = [4, 6, 8];
const ringSpeeds = [0.002, -0.001, 0.0015];

const RingSystem = ({ onServiceClick, opacity }: RingSystemProps) => {
  const ringsRef = useRef<THREE.Group[]>([]);
  const { playClick } = useAudio();

  useFrame((state) => {
    const time = state.clock.elapsedTime;
    
    ringsRef.current.forEach((ring, index) => {
      if (ring) {
        ring.rotation.z = time * ringSpeeds[index];
      }
    });
  });

  const ringGeometries = useMemo(() => {
    return ringRadii.map(radius => {
      const points = [];
      for (let i = 0; i <= 64; i++) {
        const angle = (i / 64) * Math.PI * 2;
        points.push(new THREE.Vector3(Math.cos(angle) * radius, Math.sin(angle) * radius, 0));
      }
      return new THREE.BufferGeometry().setFromPoints(points);
    });
  }, []);

  return (
    <animated.group>
      {/* Central KNIRVANA */}
      <Text
        position={[0, 0, 0.1]}
        fontSize={0.6}
        color="#00ffff"
        anchorX="center"
        anchorY="middle"
        font="/fonts/inter.json"
        onClick={() => {
          playClick();
          onServiceClick('KNIRVANA');
        }}
        onPointerOver={() => document.body.style.cursor = 'pointer'}
        onPointerOut={() => document.body.style.cursor = 'default'}
      >
        KNIRVANA
      </Text>

      {/* Rings */}
      {ringRadii.map((radius, ringIndex) => (
        <group
          key={ringIndex}
          ref={(el) => {
            if (el) ringsRef.current[ringIndex] = el;
          }}
        >
          {/* Ring geometry */}
          <lineLoop geometry={ringGeometries[ringIndex]}>
            <lineBasicMaterial 
              color="#00ccff" 
              transparent 
              opacity={0.3}
            />
          </lineLoop>

          {/* Ring glow */}
          <lineLoop geometry={ringGeometries[ringIndex]}>
            <lineBasicMaterial 
              color="#00ccff" 
              transparent 
              opacity={0.1}
            />
          </lineLoop>

          {/* Electric shocks */}
          <ElectricShock radius={radius} />
        </group>
      ))}

      {/* Electric network connections between rings */}
      <ElectricNetwork ringRadii={ringRadii} />

      {/* Service labels */}
      {services.map((service, index) => {
        const radius = ringRadii[service.ring];
        const x = Math.cos(service.angle) * radius;
        const y = Math.sin(service.angle) * radius;
        
        return (
          <group key={service.name} position={[x, y, 0]}>
            <Text
              position={[0, 0, 0.1]}
              fontSize={0.25}
              color="#ffffff"
              anchorX="center"
              anchorY="middle"
              font="/fonts/inter.json"
              onClick={() => {
                console.log(`Clicked on ${service.name}`);
                playClick();
                onServiceClick(service.name);
              }}
              onPointerOver={(e) => {
                e.stopPropagation();
                document.body.style.cursor = 'pointer';
              }}
              onPointerOut={(e) => {
                e.stopPropagation();
                document.body.style.cursor = 'default';
              }}
            >
              {service.name}
            </Text>
            
            {/* Service node */}
            <mesh>
              <circleGeometry args={[0.15, 16]} />
              <meshBasicMaterial 
                color="#00ccff" 
                transparent 
                opacity={0.5}
              />
            </mesh>
            
            {/* Service glow */}
            <mesh>
              <circleGeometry args={[0.3, 16]} />
              <meshBasicMaterial 
                color="#00ccff" 
                transparent 
                opacity={0.1}
              />
            </mesh>
          </group>
        );
      })}
    </animated.group>
  );
};

export default RingSystem;
