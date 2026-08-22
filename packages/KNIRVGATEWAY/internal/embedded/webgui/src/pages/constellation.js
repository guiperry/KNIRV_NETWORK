import dynamic from 'next/dynamic';

// The constellation menu (with its star-supernova intro) is restored verbatim
// from packages/KNIRVSERVER/frontend/src/components/menu/constellation-menu.tsx
// as it existed before commit 9e480917fa432b6c8d5b926fe853f6ca09398c2c migrated
// (and simplified) it into this WebGUI.
const ConstellationMenu = dynamic(
  () => import('../components/menu/constellation-menu'),
  { loading: () => <div style={{ minHeight: '100vh', background: '#030a18' }} /> }
);

export default function ConstellationPage() {
  return <ConstellationMenu />;
}
