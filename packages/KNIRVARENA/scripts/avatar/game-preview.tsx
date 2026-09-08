import React, { Suspense } from 'react';
import { createRoot } from 'react-dom/client';
import { Canvas } from '@react-three/fiber';
import { OrbitControls } from '@react-three/drei';
import AIAgent from '../../src/components/game/AIAgent';
import { useKnirvana } from '../../src/components/game/stores/useKnirvana';

const original = useKnirvana.getState().agents[0];
function reset() {
  useKnirvana.setState({
    nrnBalance: 100,
    agents: [{ ...original, id: 'preview-bot', staged: false, target: null,
      status: 'idle', position: { x: -4, y: 1, z: -3 } }],
    errorNodes: [{ id: 'preview-error', position: { x: 0, y: 0, z: 0 },
      type: 'Runtime Error', difficulty: 1, bounty: 10, progress: 0, isBeingSolved: false }],
  });
}
reset();
Object.assign(window, { arenaPreviewStore: useKnirvana });
function Preview() {
  const agent = useKnirvana(s => s.agents[0]);
  return <><aside><h2>Game integration · {agent.status}</h2><button onClick={() => {
    reset(); useKnirvana.getState().deployAgent('preview-bot', 'preview-error');
  }}>Deploy to error node</button></aside><Canvas camera={{ position: [5, 5, 8], fov: 45 }}>
    <ambientLight intensity={2} /><directionalLight position={[3, 6, 5]} intensity={3} />
    <gridHelper args={[30, 30]} /><OrbitControls target={[-1, .6, -1]} />
    <mesh position={[0, 1, 0]}><sphereGeometry args={[.8, 16, 16]} /><meshStandardMaterial color="#ff5544" wireframe /></mesh>
    <Suspense fallback={null}><AIAgent agent={agent} isSelected={false} onSelect={() => {}} /></Suspense>
  </Canvas></>;
}
createRoot(document.getElementById('root')!).render(<Preview />);
