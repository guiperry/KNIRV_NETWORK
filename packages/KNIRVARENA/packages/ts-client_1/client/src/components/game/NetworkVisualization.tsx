import React, { useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';
import { useKnirvana } from '../../lib/stores/useKnirvana';

export default function NetworkVisualization() {
  const { errorNodes, skillNodes, gameTime } = useKnirvana();
  const connectionLinesRef = useRef<THREE.Group>(null);
  const dataFlowRef = useRef<THREE.Points>(null);

  useFrame((state) => {
    // Animate network connections
    if (connectionLinesRef.current) {
      connectionLinesRef.current.children.forEach((line, index) => {
        const material = (line as THREE.Mesh).material as THREE.LineBasicMaterial;
        material.opacity = 0.3 + Math.sin(state.clock.elapsedTime * 2 + index) * 0.2;
      });
    }

    // Animate data flow particles
    if (dataFlowRef.current) {
      const positions = dataFlowRef.current.geometry.attributes.position.array as Float32Array;
      for (let i = 0; i < positions.length; i += 3) {
        positions[i + 1] += Math.sin(state.clock.elapsedTime + i) * 0.01;
      }
      dataFlowRef.current.geometry.attributes.position.needsUpdate = true;
    }
  });

  // Generate connection lines between nodes
  const generateConnections = () => {
    const connections = [];
    const allNodes = [...errorNodes, ...skillNodes];
    
    for (let i = 0; i < allNodes.length - 1; i++) {
      for (let j = i + 1; j < Math.min(i + 3, allNodes.length); j++) {
        const nodeA = allNodes[i];
        const nodeB = allNodes[j];
        
        const distance = Math.sqrt(
          Math.pow(nodeA.position.x - nodeB.position.x, 2) +
          Math.pow(nodeA.position.z - nodeB.position.z, 2)
        );
        
        // Only connect nearby nodes
        if (distance < 8) {
          const geometry = new THREE.BufferGeometry().setFromPoints([
            new THREE.Vector3(nodeA.position.x, nodeA.position.y + 0.5, nodeA.position.z),
            new THREE.Vector3(nodeB.position.x, nodeB.position.y + 0.5, nodeB.position.z)
          ]);
          
          connections.push(
            <line key={`connection-${i}-${j}`} geometry={geometry}>
              <lineBasicMaterial 
                color="#004477" 
                transparent 
                opacity={0.3}
              />
            </line>
          );
        }
      }
    }
    return connections;
  };

  // Generate data flow particles
  const generateDataFlow = () => {
    const particleCount = 200;
    const particles = new Float32Array(particleCount * 3);
    
    for (let i = 0; i < particleCount * 3; i += 3) {
      particles[i] = (Math.random() - 0.5) * 40;     // x
      particles[i + 1] = Math.random() * 20 + 1;     // y  
      particles[i + 2] = (Math.random() - 0.5) * 40; // z
    }
    
    return particles;
  };

  return (
    <group>
      {/* Network connection lines */}
      <group ref={connectionLinesRef}>
        {generateConnections()}
      </group>

      {/* Data flow visualization */}
      <points ref={dataFlowRef}>
        <bufferGeometry>
          <bufferAttribute
            attach="attributes-position"
            count={200}
            array={generateDataFlow()}
            itemSize={3}
          />
        </bufferGeometry>
        <pointsMaterial 
          size={0.02}
          color="#0088cc"
          transparent
          opacity={0.6}
          sizeAttenuation={true}
        />
      </points>

      {/* Network health visualization */}
      <mesh position={[0, 25, 0]}>
        <sphereGeometry args={[0.5, 16, 16]} />
        <meshStandardMaterial
          color="#00ffff"
          emissive="#00ffff"
          emissiveIntensity={0.4 + Math.sin(gameTime * 0.01) * 0.2}
          transparent
          opacity={0.8}
        />
      </mesh>

      {/* Data streams from central node */}
      {[0, 60, 120, 180, 240, 300].map((angle, index) => (
        <group key={`stream-${index}`} rotation={[0, (angle * Math.PI) / 180, 0]}>
          <mesh position={[0, 25, 15]}>
            <cylinderGeometry args={[0.01, 0.01, 20]} />
            <meshBasicMaterial 
              color="#0066cc"
              transparent
              opacity={0.4}
            />
          </mesh>
        </group>
      ))}

      {/* Knowledge graph grid overlay */}
      <mesh rotation={[Math.PI / 2, 0, 0]} position={[0, 0.1, 0]}>
        <planeGeometry args={[60, 60, 20, 20]} />
        <meshBasicMaterial 
          color="#002244"
          transparent
          opacity={0.1}
          wireframe
        />
      </mesh>

      {/* Sector boundary indicators */}
      {[-15, -5, 5, 15].map((x, i) =>
        [-15, -5, 5, 15].map((z, j) => (
          <mesh key={`sector-${i}-${j}`} position={[x, 0.2, z]}>
            <cylinderGeometry args={[0.1, 0.1, 0.2]} />
            <meshBasicMaterial 
              color="#003366"
              transparent
              opacity={0.5}
            />
          </mesh>
        ))
      )}
    </group>
  );
}