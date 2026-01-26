import { Canvas } from "@react-three/fiber";
import { Suspense, useState, useEffect } from "react";
import { OrbitControls } from "@react-three/drei";
import "@fontsource/inter";
import * as THREE from "three";
import KnirvMenu from "./components/KnirvMenu";
import ServiceModal from "./components/ServiceModal";
import AudioManager from "./components/AudioManager";
import AudioControls from "./components/AudioControls";

// WebGL capability detection
const checkWebGLSupport = (): boolean => {
  try {
    const canvas = document.createElement('canvas');
    const gl = canvas.getContext('webgl2') || canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
    const supported = !!gl;
    console.log('WebGL support check:', supported);
    return supported;
  } catch (e) {
    console.error('WebGL check failed:', e);
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
  const [webglSupported, setWebglSupported] = useState<boolean | null>(true); // Start with true to show content immediately
  
  useEffect(() => {
    // Check WebGL support but don't block rendering
    const supported = checkWebGLSupport();
    setWebglSupported(supported);
    if (!supported) {
      console.warn('WebGL not supported, but continuing anyway');
    }
  }, []);

  return (
    <div style={{ width: '100vw', height: '100vh', position: 'relative', overflow: 'hidden', backgroundColor: '#0a0a0a' }}>
      <AudioManager />
      
      {webglSupported === false ? (
        <WebGLFallback />
      ) : (
        <Canvas
          dpr={[1, 2]}
          shadows={false}
          camera={{
            position: [0, 0, 15],
            fov: 45,
            near: 0.1,
            far: 1000
          }}
          gl={{
            antialias: false, // Disable for better performance
            alpha: true,
            powerPreference: 'default', // Change to default for better compatibility
            preserveDrawingBuffer: false,
            stencil: false,
          }}
          onCreated={({ gl, scene, camera }) => {
            console.log('Three.js Canvas created successfully');
            console.log('Scene:', scene);
            console.log('Camera:', camera);
            
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
          <ambientLight intensity={0.2} />
          <directionalLight 
            position={[10, 10, 5]} 
            intensity={0.5} 
          />
          <pointLight position={[0, 0, 10]} intensity={0.3} color="#00ccff" />

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
      )}

      <ServiceModal 
        service={selectedService}
        isOpen={selectedService !== null}
        onClose={() => setSelectedService(null)}
      />
      
      <AudioControls />
      
      {/* Debug info */}
      <div className="fixed bottom-4 right-4 text-white text-xs bg-black/50 p-2 rounded">
        WebGL: {webglSupported === null ? 'checking...' : webglSupported ? 'supported' : 'not supported'}
      </div>
    </div>
  );
}

export default App;
