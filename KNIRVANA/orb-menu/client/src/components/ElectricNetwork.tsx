import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface ElectricNetworkProps {
  ringRadii: number[];
}

interface ElectricBranch {
  startRadius: number;
  endRadius: number;
  startAngle: number;
  endAngle: number;
  segments: THREE.Vector3[];
  intensity: number;
  age: number;
  maxAge: number;
}

const ElectricNetwork = ({ ringRadii }: ElectricNetworkProps) => {
  const groupRef = useRef<THREE.Group>(null);
  const branchesRef = useRef<ElectricBranch[]>([]);
  const lastBranchTime = useRef<number>(0);
  const maxBranches = 8;

  // Create line geometry for electric branches
  const lineGeometry = useMemo(() => new THREE.BufferGeometry(), []);
  
  const material = useMemo(() => {
    return new THREE.ShaderMaterial({
      uniforms: {
        time: { value: 0 },
        intensity: { value: 1.0 }
      },
      vertexShader: `
        precision mediump float;
        precision mediump int;
        
        attribute float alpha;
        attribute float segmentIntensity;
        varying float vAlpha;
        varying float vIntensity;
        
        void main() {
          vAlpha = alpha;
          vIntensity = segmentIntensity;
          
          vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
          gl_Position = projectionMatrix * mvPosition;
        }
      `,
      fragmentShader: `
        precision mediump float;
        precision mediump int;
        
        uniform float time;
        uniform float intensity;
        varying float vAlpha;
        varying float vIntensity;
        
        void main() {
          // Electric blue to white based on intensity
          vec3 color = mix(
            vec3(0.0, 0.7, 1.0), 
            vec3(1.0, 1.0, 1.0), 
            vIntensity
          );
          
          // Add flickering effect
          float flicker = sin(time * 20.0 + vIntensity * 10.0) * 0.3 + 0.7;
          
          gl_FragColor = vec4(color * flicker, vAlpha * intensity);
        }
      `,
      transparent: true,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
    });
  }, []);

  const generateBranchSegments = (startRadius: number, endRadius: number, startAngle: number, endAngle: number): THREE.Vector3[] => {
    const segments: THREE.Vector3[] = [];
    const segmentCount = 12; // More segments for more detailed branching
    
    for (let i = 0; i <= segmentCount; i++) {
      const t = i / segmentCount;
      
      // Linear interpolation between start and end points
      const radius = THREE.MathUtils.lerp(startRadius, endRadius, t);
      const angle = THREE.MathUtils.lerp(startAngle, endAngle, t);
      
      // Add some random deviation for lightning-like effect
      const deviation = (Math.random() - 0.5) * 0.3 * Math.sin(t * Math.PI); // Peak deviation in middle
      const radiusDeviation = (Math.random() - 0.5) * 0.2;
      
      const x = Math.cos(angle + deviation) * (radius + radiusDeviation);
      const y = Math.sin(angle + deviation) * (radius + radiusDeviation);
      const z = (Math.random() - 0.5) * 0.1; // Small z variation
      
      segments.push(new THREE.Vector3(x, y, z));
    }
    
    return segments;
  };

  const spawnElectricBranch = () => {
    if (branchesRef.current.length >= maxBranches || ringRadii.length < 2) return;
    
    // Pick two different rings to connect
    const startRingIndex = Math.floor(Math.random() * ringRadii.length);
    let endRingIndex = Math.floor(Math.random() * ringRadii.length);
    while (endRingIndex === startRingIndex) {
      endRingIndex = Math.floor(Math.random() * ringRadii.length);
    }
    
    const startRadius = ringRadii[startRingIndex];
    const endRadius = ringRadii[endRingIndex];
    
    // Random angles for connection points
    const startAngle = Math.random() * Math.PI * 2;
    const endAngle = Math.random() * Math.PI * 2;
    
    const branch: ElectricBranch = {
      startRadius,
      endRadius,
      startAngle,
      endAngle,
      segments: generateBranchSegments(startRadius, endRadius, startAngle, endAngle),
      intensity: 0.7 + Math.random() * 0.3,
      age: 0,
      maxAge: 0.5 + Math.random() * 1.0 // 0.5 to 1.5 seconds
    };
    
    branchesRef.current.push(branch);
  };

  useFrame((state) => {
    const time = state.clock.elapsedTime;
    const delta = state.clock.getDelta();
    
    // Update material time uniform
    material.uniforms.time.value = time;
    
    // Spawn new branches occasionally
    if (time - lastBranchTime.current > 0.8 + Math.random() * 1.2) { // Every 0.8-2.0 seconds
      if (Math.random() < 0.6) { // 60% chance to spawn
        spawnElectricBranch();
        
        // Electric network visual effect only (no audio)
        
        lastBranchTime.current = time;
      }
    }
    
    // Update existing branches
    const branches = branchesRef.current;
    const activeBranches: ElectricBranch[] = [];
    
    branches.forEach(branch => {
      branch.age += delta;
      
      if (branch.age < branch.maxAge) {
        // Update branch intensity based on age
        const ageRatio = branch.age / branch.maxAge;
        branch.intensity = Math.max(0, 1 - ageRatio * ageRatio); // Fade out with quadratic curve
        activeBranches.push(branch);
      }
    });
    
    branchesRef.current = activeBranches;
    
    // Update geometry with active branches
    updateBranchGeometry();
  });

  const updateBranchGeometry = () => {
    const branches = branchesRef.current;
    
    if (branches.length === 0) {
      // Hide geometry when no branches
      if (lineGeometry.attributes.position) {
        lineGeometry.setDrawRange(0, 0);
      }
      return;
    }
    
    // Calculate total vertices needed
    const totalVertices = branches.reduce((sum, branch) => sum + branch.segments.length, 0);
    
    if (totalVertices === 0) return;
    
    // Create arrays for all branch data
    const positions = new Float32Array(totalVertices * 3);
    const alphas = new Float32Array(totalVertices);
    const intensities = new Float32Array(totalVertices);
    
    let vertexIndex = 0;
    
    branches.forEach(branch => {
      branch.segments.forEach((segment, segIndex) => {
        positions[vertexIndex * 3] = segment.x;
        positions[vertexIndex * 3 + 1] = segment.y;
        positions[vertexIndex * 3 + 2] = segment.z;
        
        // Alpha fades from center to edges of branch
        const segmentRatio = segIndex / (branch.segments.length - 1);
        const centerFade = 1 - Math.abs(segmentRatio - 0.5) * 2; // Peak at center
        alphas[vertexIndex] = centerFade * branch.intensity;
        
        intensities[vertexIndex] = branch.intensity;
        
        vertexIndex++;
      });
    });
    
    // Update geometry attributes
    if (lineGeometry.attributes.position && lineGeometry.attributes.position.count === totalVertices) {
      // Update existing attributes if they're the same size
      (lineGeometry.attributes.position.array as Float32Array).set(positions);
      lineGeometry.attributes.position.needsUpdate = true;
      (lineGeometry.attributes.alpha.array as Float32Array).set(alphas);
      lineGeometry.attributes.alpha.needsUpdate = true;
      (lineGeometry.attributes.segmentIntensity.array as Float32Array).set(intensities);
      lineGeometry.attributes.segmentIntensity.needsUpdate = true;
    } else {
      // Create new attributes if they don't exist or are different size
      lineGeometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
      lineGeometry.setAttribute('alpha', new THREE.BufferAttribute(alphas, 1));
      lineGeometry.setAttribute('segmentIntensity', new THREE.BufferAttribute(intensities, 1));
    }
    
    lineGeometry.setDrawRange(0, totalVertices);
  };

  return (
    <group ref={groupRef}>
      <points geometry={lineGeometry} material={material} />
    </group>
  );
};

export default ElectricNetwork;