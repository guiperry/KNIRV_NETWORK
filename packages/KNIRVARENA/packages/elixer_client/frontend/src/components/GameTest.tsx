import React from 'react';
import GameArena from './GameArena';

// Simple test component to verify game integration
export default function GameTest() {
  return (
    <div className="w-full h-screen bg-gray-900">
      <h1 className="text-white text-center p-4">KNIRVANA Gaming Arena Test</h1>
      <div className="w-full h-3/4 border border-cyan-500 rounded-lg overflow-hidden">
        <GameArena />
      </div>
    </div>
  );
}