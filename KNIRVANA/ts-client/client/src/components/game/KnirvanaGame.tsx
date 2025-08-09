import { Canvas } from "@react-three/fiber";
import { Suspense } from "react";
import GameScene from "./GameScene";
import GameUI from "../ui/GameUI";
import { useKnirvana } from "../../lib/stores/useKnirvana";

export default function KnirvanaGame() {
  const { gamePhase } = useKnirvana();

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
          powerPreference: "high-performance"
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
