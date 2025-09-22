import { useRef, useMemo } from 'react';
import { useFrame, useThree } from '@react-three/fiber';
import * as THREE from 'three';

interface TrailParticle {
  position: THREE.Vector3;
  velocity: THREE.Vector3;
  age: number;
  maxAge: number;
  size: number;
}

const ParticleTrails = () => {
  const pointsRef = useRef<THREE.Points>(null);
  const { camera, mouse, viewport } = useThree();
  const mouseWorldPosRef = useRef(new THREE.Vector3());
  const particlesRef = useRef<TrailParticle[]>([]);
  const raycaster = useMemo(() => new THREE.Raycaster(), []);
  const targetPlane = useMemo(() => new THREE.Plane(new THREE.Vector3(0, 0, 1), 0), []); // z=0 plane
  const maxParticles = 500;

  // Create geometry, material, and cached arrays
  const { geometry, material, positions, colors, sizes, alphas } = useMemo(() => {
    const geometry = new THREE.BufferGeometry();
    const positions = new Float32Array(maxParticles * 3);
    const colors = new Float32Array(maxParticles * 3);
    const sizes = new Float32Array(maxParticles);
    const alphas = new Float32Array(maxParticles);

    geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    geometry.setAttribute('aColor', new THREE.BufferAttribute(colors, 3));
    geometry.setAttribute('size', new THREE.BufferAttribute(sizes, 1));
    geometry.setAttribute('alpha', new THREE.BufferAttribute(alphas, 1));

    const material = new THREE.ShaderMaterial({
      uniforms: {
        time: { value: 0 }
      },
      vertexShader: `
        precision mediump float;
        precision mediump int;
        
        attribute float size;
        attribute float alpha;
        attribute vec3 aColor;
        varying float vAlpha;
        varying vec3 vColor;
        
        void main() {
          vAlpha = alpha;
          vColor = aColor;
          
          vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
          gl_PointSize = max(1.0, min(128.0, size * (300.0 / max(0.0001, -mvPosition.z))));
          gl_Position = projectionMatrix * mvPosition;
        }
      `,
      fragmentShader: `
        precision mediump float;
        precision mediump int;
        
        uniform float time;
        varying float vAlpha;
        varying vec3 vColor;
        
        void main() {
          // Create a circular particle with soft edges
          vec2 center = gl_PointCoord - 0.5;
          float dist = length(center);
          
          if (dist > 0.5) discard;
          
          // Soft falloff for glowing effect
          float alpha = smoothstep(0.5, 0.1, dist) * vAlpha;
          
          // Add some glow and sparkle
          float glow = exp(-dist * 8.0);
          vec3 glowColor = vColor + vec3(0.3, 0.5, 1.0) * glow * 0.3;
          
          gl_FragColor = vec4(glowColor, alpha);
        }
      `,
      transparent: true,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
    });

    return { geometry, material, positions, colors, sizes, alphas };
  }, []); // Empty deps to ensure stable arrays

  // Convert mouse coordinates to world space and update particles
  useFrame((state) => {
    const { mouse, camera } = state;
    const delta = state.clock.getDelta();
    
    // Use raycaster to convert mouse to world coordinates on z=0 plane
    raycaster.setFromCamera(mouse, camera);
    const intersection = new THREE.Vector3();
    const hit = raycaster.ray.intersectPlane(targetPlane, intersection);
    
    if (hit) {
      mouseWorldPosRef.current.copy(hit);
    }
    
    // Update time uniform
    material.uniforms.time.value = state.clock.elapsedTime;
    
    // Update particles with delta time
    updateParticles(hit || mouseWorldPosRef.current, delta);
  });

  const updateParticles = (mousePos: THREE.Vector3, delta: number) => {
    const particles = particlesRef.current;
    
    // Spawn new particles near the mouse
    if (Math.random() < 0.3) { // 30% chance per frame to spawn
      const newParticle: TrailParticle = {
        position: new THREE.Vector3(
          mousePos.x + (Math.random() - 0.5) * 0.5,
          mousePos.y + (Math.random() - 0.5) * 0.5,
          mousePos.z + (Math.random() - 0.5) * 0.2
        ),
        velocity: new THREE.Vector3(
          (Math.random() - 0.5) * 0.02,
          (Math.random() - 0.5) * 0.02,
          (Math.random() - 0.5) * 0.01
        ),
        age: 0,
        maxAge: 2 + Math.random() * 3, // 2-5 seconds lifespan
        size: 5 + Math.random() * 10
      };
      
      if (particles.length < maxParticles) {
        particles.push(newParticle);
      } else {
        // Replace oldest particle
        const oldestIndex = particles.findIndex(p => p.age / p.maxAge > 0.9);
        if (oldestIndex !== -1) {
          particles[oldestIndex] = newParticle;
        }
      }
    }

    // Ensure attributes exist and rebind if needed
    const ensureAttribute = (name: string, array: Float32Array, itemSize: number) => {
      const attr = geometry.getAttribute(name);
      if (!attr || !(attr as any).array) {
        console.warn(`Rebinding attribute: ${name}`);
        geometry.setAttribute(name, new THREE.BufferAttribute(array, itemSize));
      }
    };

    ensureAttribute('position', positions, 3);
    ensureAttribute('aColor', colors, 3);
    ensureAttribute('size', sizes, 1);
    ensureAttribute('alpha', alphas, 1);

    particles.forEach((particle, index) => {
      if (index >= maxParticles) return;
      
      // Age the particle (frame-rate independent)
      particle.age += delta;
      
      // Update position with some drift (frame-rate independent)
      particle.position.addScaledVector(particle.velocity, delta * 60); // scale for 60fps baseline
      
      // Add some attraction back to mouse for trailing effect
      const mouseDirection = mousePos.clone().sub(particle.position);
      const distance = mouseDirection.length();
      if (distance > 0) {
        mouseDirection.normalize();
        const attractStrength = 0.001 * distance;
        particle.velocity.add(mouseDirection.multiplyScalar(attractStrength * delta * 60));
      }
      
      // Apply some dampening (frame-rate independent)
      const dampFactor = Math.pow(0.98, delta * 60);
      particle.velocity.multiplyScalar(dampFactor);
      
      // Calculate alpha based on age
      const ageRatio = particle.age / particle.maxAge;
      const alpha = Math.max(0, 1 - ageRatio);
      
      // Update buffer attributes
      positions[index * 3] = particle.position.x;
      positions[index * 3 + 1] = particle.position.y;
      positions[index * 3 + 2] = particle.position.z;
      
      // Color gradient from cyan to blue to purple
      const colorPhase = ageRatio * Math.PI;
      colors[index * 3] = 0.2 + Math.sin(colorPhase) * 0.5; // R
      colors[index * 3 + 1] = 0.7 + Math.sin(colorPhase + Math.PI/3) * 0.3; // G
      colors[index * 3 + 2] = 1.0; // B (always full blue)
      
      sizes[index] = particle.size * (1 - ageRatio * 0.5); // Shrink over time
      alphas[index] = alpha;
    });

    // Remove dead particles
    particlesRef.current = particles.filter(p => p.age < p.maxAge);
    
    // Get active particle count after filtering
    const activeCount = particlesRef.current.length;

    // Set draw range for active particles
    geometry.setDrawRange(0, activeCount);

    // Mark attributes for update if they exist
    const posAttr = geometry.getAttribute('position');
    const colorAttr = geometry.getAttribute('aColor'); 
    const sizeAttr = geometry.getAttribute('size');
    const alphaAttr = geometry.getAttribute('alpha');

    if (posAttr) posAttr.needsUpdate = true;
    if (colorAttr) colorAttr.needsUpdate = true;
    if (sizeAttr) sizeAttr.needsUpdate = true;
    if (alphaAttr) alphaAttr.needsUpdate = true;
  };

  return (
    <points ref={pointsRef} geometry={geometry} material={material} frustumCulled={false} />
  );
};

export default ParticleTrails;