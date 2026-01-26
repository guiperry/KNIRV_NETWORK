import { Canvas } from "@react-three/fiber";
import { Suspense, useEffect } from "react";
import GameScene from "./GameScene";
import GameUI from "./GameUI";
import { useKnirvana } from "./stores/useKnirvana";

export default function KnirvanaGame() {
  const { gamePhase, selectedAgent } = useKnirvana();

  return (
    <>
      <Canvas
        shadows
        camera={{
          position: [15, 20, 15],
          fov: 60,
          near: 0.1,
          far: 1000
        }}
        gl={{
          antialias: true,
          powerPreference: "high-performance",
          alpha: false,
          preserveDrawingBuffer: true
        }}
        style={{
          background: '#000510',
          width: '100vw',
          height: '100vh',
          position: 'fixed',
          top: 0,
          left: 0,
          zIndex: 1
        }}

      >
        <color attach="background" args={["#000510"]} />

        <Suspense fallback={null}>
          <GameScene />
        </Suspense>
      </Canvas>
      
      <GameUI />
    </>
  );
}