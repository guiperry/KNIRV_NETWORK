import { Canvas } from "@react-three/fiber";
import { Suspense } from "react";
import GameScene from "./GameScene";
import GameUI from "./GameUI";
import { useKnirvana } from "./stores/useKnirvana";
import { useThemeStore } from "../../stores/useThemeStore";

export default function KnirvanaGame() {
  const { gamePhase } = useKnirvana();
  const { themeMode } = useThemeStore();

  // Theme-aware background colors
  const getBackgroundColors = () => {
    switch (themeMode) {
      case 'light':
        return { style: '#f5f5f5', color: '#f5f5f5' };
      case 'light-blue':
        return { style: '#001a33', color: '#001a33' };
      default: // dark
        return { style: '#000510', color: '#000510' };
    }
  };

  const backgrounds = getBackgroundColors();

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
          background: backgrounds.style,
          width: '100vw',
          height: '100vh',
          position: 'fixed',
          top: 0,
          left: 0,
          zIndex: 1
        }}
      >
        <color attach="background" args={[backgrounds.color]} />

        <Suspense fallback={null}>
          <GameScene />
        </Suspense>
      </Canvas>
      
      <GameUI />
    </>
  );
}