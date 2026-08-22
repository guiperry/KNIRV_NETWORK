import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/router';
import styles from './constellation.module.css';

const innerLabels = [
  ['KNIRVCLI', 210], ['KNIRVCTCLI', 250], ['KNIRVCHAIN', 290],
  ['KNIRVCCHAIN', 330], ['KNIRVCINEXUS', 10], ['KNIRVROUTER', 50],
  ['KNIRVCONTROLLER', 90], ['KNIRVCLLS', 130], ['KNIRVCWORKER', 170],
];

const middleLabels = [
  ['KNIRVCORPAY', 230], ['KNIRVASDK', 270], ['KNIRVCOGINS', 310],
  ['KNIRVCORTEX', 350], ['KNIRVGATEWAY', 30], ['KNIRVCONTROLLER', 70],
  ['KNIRVROUTET', 110], ['KNIRVBALI', 150], ['KNIRVTESTNET', 190],
];

const destinations = [
  { label: 'Setup', icon: '△', href: '/dashboard', angle: 0 },
  { label: 'WebGUI', icon: '◎', href: '/dashboard', angle: 45 },
  { label: 'Admin', icon: '◉', href: '/network-admin', angle: 90 },
  { label: 'Cognitive', icon: '◌', href: '/models', angle: 135 },
  { label: 'DVE Nodes', icon: '▦', href: '/dve-list', angle: 180 },
  { label: 'Settings', icon: '⚙', angle: 225 },
  { label: 'Reports', icon: '◈', href: '/network-monitor', angle: 270 },
  { label: 'Badge Lab', icon: '□', href: '/my-badges', angle: 315 },
];

function polar(angle, radius) {
  const radians = (angle - 90) * Math.PI / 180;
  return { x: Math.cos(radians) * radius, y: Math.sin(radians) * radius };
}

export default function ConstellationPage() {
  const router = useRouter();
  const [ready, setReady] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [zooming, setZooming] = useState(false);
  const stars = useMemo(() => Array.from({ length: 150 }, (_, index) => ({
    id: index,
    left: `${(index * 37.7) % 100}%`,
    top: `${(index * 61.3) % 100}%`,
    size: index % 11 === 0 ? 2 : 1,
    delay: `${(index % 9) * 0.35}s`,
  })), []);

  useEffect(() => {
    const timer = window.setTimeout(() => setReady(true), 400);
    return () => window.clearTimeout(timer);
  }, []);

  useEffect(() => {
    if (!zooming) return undefined;
    const timer = window.setTimeout(() => router.push('/arena'), 900);
    return () => window.clearTimeout(timer);
  }, [router, zooming]);

  const chooseDestination = (item) => {
    if (!item.href) {
      setSettingsOpen(true);
      return;
    }
    router.push(item.href);
  };

  return (
    <main className={styles.constellation} aria-label="KNIRV constellation menu">
      <div className={styles.stars} aria-hidden="true">
        {stars.map((star) => <i key={star.id} style={{ left: star.left, top: star.top, width: star.size, height: star.size, animationDelay: star.delay }} />)}
      </div>
      <div className={styles.horizon} aria-hidden="true" />
      <section className={`${styles.orbit} ${ready ? styles.ready : ''}`}>
        <svg className={styles.rings} viewBox="-450 -450 900 900" aria-hidden="true">
          {[120, 180, 250, 320, 355, 365].map((radius, index) => <circle key={radius} r={radius} className={`${styles.ring} ${index === 2 ? styles.rotatingRing : ''}`} />)}
          {Array.from({ length: 12 }, (_, index) => {
            const end = polar(index * 30, 340);
            return <line key={index} x1="0" y1="0" x2={end.x} y2={end.y} className={styles.radial} />;
          })}
        </svg>

        {innerLabels.map(([label, angle]) => {
          const point = polar(angle, 155);
          return <span key={label} className={`${styles.label} ${styles.innerLabel}`} style={{ transform: `translate(calc(-50% + ${point.x}px), calc(-50% + ${point.y}px))` }}>{label}</span>;
        })}
        {middleLabels.map(([label, angle]) => {
          const point = polar(angle, 220);
          return <span key={label} className={`${styles.label} ${styles.middleLabel}`} style={{ transform: `translate(calc(-50% + ${point.x}px), calc(-50% + ${point.y}px))` }}>{label}</span>;
        })}

        {destinations.map((item) => {
          const point = polar(item.angle, 360);
          return (
            <button key={item.label} type="button" className={styles.destination} style={{ transform: `translate(calc(-50% + ${point.x}px), calc(-50% + ${point.y}px))` }} onClick={() => chooseDestination(item)}>
              <span className={styles.destinationIcon}>{item.icon}</span>
              <span>{item.label}</span>
            </button>
          );
        })}

        <button type="button" className={styles.core} onClick={() => setZooming(true)} aria-label="Open KNIRV Arena">
          <span className={styles.coreMesh} aria-hidden="true" />
          <strong>KNIRVARENA</strong>
        </button>
        <div className={styles.brand}>KNIRV.COM</div>
      </section>

      {zooming && <div className={styles.zoomOverlay}><strong>KNIRVARENA</strong></div>}
      {settingsOpen && (
        <div className={styles.modalBackdrop} role="presentation" onMouseDown={() => setSettingsOpen(false)}>
          <section className={styles.modal} role="dialog" aria-modal="true" aria-label="Constellation settings" onMouseDown={(event) => event.stopPropagation()}>
            <h1>Constellation settings</h1>
            <p>Visual and audio preferences are managed in the WebGUI settings page.</p>
            <div><button type="button" onClick={() => router.push('/settings')}>Open settings</button><button type="button" onClick={() => setSettingsOpen(false)}>Close</button></div>
          </section>
        </div>
      )}
    </main>
  );
}
