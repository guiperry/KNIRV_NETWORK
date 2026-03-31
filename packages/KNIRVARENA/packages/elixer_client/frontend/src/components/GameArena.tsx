import React, { useEffect } from 'react';
import { KeyboardControls } from '@react-three/drei';
import { useKnirvana } from './game/stores/useKnirvana';
import { useAudio } from './game/stores/useAudio';
import KnirvanaGame from './game/KnirvanaGame';

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
  { name: "nextError", keys: ["Tab"] },
  { name: "prevError", keys: ["ShiftLeft", "Tab"] },
];

export default function GameArena() {
  const { gamePhase, startGame } = useKnirvana();
  const { setBackgroundMusic, setHitSound, setSuccessSound } = useAudio();

  useEffect(() => {
    // Initialize audio assets
    const backgroundMusic = new Audio('/sounds/background.mp3');
    backgroundMusic.loop = true;
    backgroundMusic.volume = 0.2;
    setBackgroundMusic(backgroundMusic);

    const hitSound = new Audio('/sounds/hit.mp3');
    setHitSound(hitSound);

    const successSound = new Audio('/sounds/success.mp3');
    setSuccessSound(successSound);

    // Auto-start game when component mounts
    if (gamePhase === "menu") {
      startGame();
    }
  }, [gamePhase, startGame, setBackgroundMusic, setHitSound, setSuccessSound]);

  return (
    <div className="w-full h-full rounded-2xl overflow-hidden relative">
      <KeyboardControls map={controls}>
        <KnirvanaGame />
      </KeyboardControls>
    </div>
  );
}