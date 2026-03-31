import { Canvas } from "@react-three/fiber";
import { Suspense, useEffect, useState } from "react";
import { KeyboardControls } from "@react-three/drei";
import { useAudio } from "./lib/stores/useAudio";
import "@fontsource/inter";
import KnirvanaGame from "./components/game/KnirvanaGame";

// Define control keys for the RTS game
const controls = [
  { name: "forward", keys: ["KeyW", "ArrowUp"] },
  { name: "backward", keys: ["KeyS", "ArrowDown"] },
  { name: "leftward", keys: ["KeyA", "ArrowLeft"] },
  { name: "rightward", keys: ["KeyD", "ArrowRight"] },
  { name: "rotateLeft", keys: ["KeyQ"] },
  { name: "rotateRight", keys: ["KeyE"] },
  { name: "zoomIn", keys: ["Equal", "NumpadAdd"] },
  { name: "zoomOut", keys: ["Minus", "NumpadSubtract"] },
  { name: "select", keys: ["Space"] },
  { name: "deploy", keys: ["KeyR"] },
];

// Main App component
function App() {
  const [showCanvas, setShowCanvas] = useState(false);

  // Show the canvas once everything is loaded
  useEffect(() => {
    setShowCanvas(true);
  }, []);

  return (
    <div style={{ width: '100vw', height: '100vh', position: 'relative', overflow: 'hidden' }}>
      {showCanvas && (
        <KeyboardControls map={controls}>
          <KnirvanaGame />
        </KeyboardControls>
      )}
    </div>
  );
}

export default App;
