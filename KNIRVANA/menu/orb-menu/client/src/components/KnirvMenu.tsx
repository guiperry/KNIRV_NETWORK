import { useState, useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import { animated, useSpring } from '@react-spring/three';
import PulsatingOrb from './PulsatingOrb';
import RingSystem from './RingSystem';
import { useKnirvMenu } from '../hooks/useKnirvMenu';
import * as THREE from 'three';

interface KnirvMenuProps {
  onServiceClick: (service: string) => void;
}

const KnirvMenu = ({ onServiceClick }: KnirvMenuProps) => {
  const { isExpanded, toggleExpansion } = useKnirvMenu();
  const groupRef = useRef<THREE.Group>(null);

  console.log('KnirvMenu rendering, isExpanded:', isExpanded);

  // Smooth transition animation
  const { scale, opacity } = useSpring({
    scale: isExpanded ? 1 : 0.1,
    opacity: isExpanded ? 1 : 0,
    config: { tension: 120, friction: 20 }
  });

  // Gentle rotation animation
  useFrame((state) => {
    if (groupRef.current) {
      groupRef.current.rotation.z += 0.001;
    }
  });

  const handleOrbClick = () => {
    console.log('Orb clicked, current isExpanded:', isExpanded);
    console.log('Calling toggleExpansion...');
    toggleExpansion();
    console.log('toggleExpansion called');
  };

  return (
    <group ref={groupRef}>
      {!isExpanded && (
        <PulsatingOrb onClick={handleOrbClick} />
      )}
      
      {isExpanded && (
        <animated.group scale={scale}>
          <RingSystem 
            onServiceClick={onServiceClick}
            opacity={opacity}
          />
        </animated.group>
      )}
    </group>
  );
};

export default KnirvMenu;
