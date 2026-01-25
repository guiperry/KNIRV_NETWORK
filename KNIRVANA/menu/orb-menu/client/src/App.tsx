import { Canvas } from "@react-three/fiber";
import { Suspense, useState, useEffect } from "react";
import { OrbitControls } from "@react-three/drei";
import "@fontsource/inter";
import * as THREE from "three";
import KnirvMenu from "./components/KnirvMenu";
import ServiceModal from "./components/ServiceModal";

// WebGL capability detection
const checkWebGLSupport = (): boolean => {
  try {
    const canvas = document.createElement('canvas');
    const gl = canvas.getContext('webgl2') || canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
    return !!gl;
  } catch (e) {
    return false;
  }
};

// Fallback UI for when WebGL is not available
const WebGLFallback = () => (
  <div className="flex items-center justify-center w-full h-full bg-gray-900 text-white">
    <div className="text-center p-8">
      <h2 className="text-2xl font-bold mb-4">WebGL Not Available</h2>
      <p className="text-gray-300 mb-4">
        This application requires WebGL support to display 3D graphics.
      </p>
      <p className="text-gray-400 text-sm">
        Please try using a different browser or enabling hardware acceleration.
      </p>
    </div>
  </div>
);

function App() {
  const [selectedService, setSelectedService] = useState<string | null>(null);
  const [webglSupported, setWebglSupported] = useState<boolean | null>(null);
  
  useEffect(() => {
    setWebglSupported(checkWebGLSupport());
  }, []);

  // Show fallback if WebGL is not supported
  if (webglSupported === false) {
    return <WebGLFallback />;
  }
  
  // Show loading while checking WebGL support
  if (webglSupported === null) {
    return (
      <div className="flex items-center justify-center w-full h-full bg-gray-900 text-white">
        <div>Checking WebGL support...</div>
      </div>
    );
  }

  return (
    <div style={{ width: '100vw', height: '100vh', position: 'relative', overflow: 'hidden', backgroundColor: '#0a0a0a' }}>
      <Canvas
        dpr={[1, 1]}
        shadows={false}
        camera={{
          position: [0, 0, 15],
          fov: 45,
          near: 0.1,
          far: 1000
        }}
        gl={{
          antialias: true,
          alpha: true,
          powerPreference: 'high-performance',
          preserveDrawingBuffer: false,
          stencil: false,
        }}
        onCreated={({ gl }) => {
          console.log('WebGL context created successfully:', gl.getContext());
          
          // Handle context loss
          gl.domElement.addEventListener('webglcontextlost', (event) => {
            event.preventDefault();
            console.log('WebGL context lost');
          });
          
          gl.domElement.addEventListener('webglcontextrestored', () => {
            console.log('WebGL context restored');
          });
        }}
        onError={(error) => {
          console.error('Three.js Canvas error:', error);
        }}
      >
        <color attach="background" args={["#0a0a0a"]} />
        
        {/* Lighting */}
        <ambientLight intensity={0.5} />
        <directionalLight
          position={[10, 10, 5]}
          intensity={0.8}
        />
        <pointLight position={[0, 0, 10]} intensity={0.5} color="#00ccff" />

        {/* Astrological background pattern */}
        <group position={[0, 0, -15]}>
          {/* Create a grid of dots for astrological background */}
          {Array.from({ length: 50 }, (_, i) => {
            const x = (Math.random() - 0.5) * 40;
            const y = (Math.random() - 0.5) * 40;
            const z = (Math.random() - 0.5) * 10;
            return (
              <mesh key={i} position={[x, y, z]}>
                <circleGeometry args={[0.02, 8]} />
                <meshBasicMaterial
                  color="#00ccff"
                  transparent
                  opacity={0.3}
                />
              </mesh>
            );
          })}

          {/* Constellation lines */}
          {Array.from({ length: 20 }, (_, i) => {
            const points = [];
            const startX = (Math.random() - 0.5) * 30;
            const startY = (Math.random() - 0.5) * 30;
            const endX = startX + (Math.random() - 0.5) * 10;
            const endY = startY + (Math.random() - 0.5) * 10;
            points.push(new THREE.Vector3(startX, startY, 0));
            points.push(new THREE.Vector3(endX, endY, 0));

            const geometry = new THREE.BufferGeometry().setFromPoints(points);
            return (
              <line key={i} geometry={geometry}>
                <lineBasicMaterial
                  color="#00ccff"
                  transparent
                  opacity={0.2}
                />
              </line>
            );
          })}
        </group>

        <Suspense fallback={null}>
          <KnirvMenu onServiceClick={setSelectedService} />
        </Suspense>

        <OrbitControls 
          enablePan={false}
          enableZoom={true}
          maxDistance={30}
          minDistance={5}
          enableDamping
          dampingFactor={0.05}
        />
      </Canvas>

      <ServiceModal
        service={selectedService}
        isOpen={selectedService !== null}
        onClose={() => setSelectedService(null)}
      />
    </div>
  );
}

export default App;
