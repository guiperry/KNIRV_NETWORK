import React, { useEffect, useRef, useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import { useRole } from '../contexts/RoleContext';

// Connection types the KNIRVCONTROLLER can scan from this gateway.
const TARGETS = [
  { id: 'webgui',  label: 'WebGUI Session',    icon: '🖥️',  description: 'Authenticate your KNIRVCONTROLLER wallet into this dashboard.' },
  { id: 'server',  label: 'KNIRVSERVER',        icon: '🌐',  description: 'Connect the controller to the backing KNIRVSERVER node.' },
  { id: 'bridge',  label: 'KNIRVBRIDGE',        icon: '🌉',  description: 'Link to the KNIRVBRIDGE cross-chain relay.' },
];

function generateSessionToken() {
  const arr = new Uint8Array(24);
  crypto.getRandomValues(arr);
  return Array.from(arr).map(b => b.toString(16).padStart(2, '0')).join('');
}

export default function QRConnectPage() {
  const { activePage } = useNavigation('qr-connect');
  const { role } = useRole();

  const [selectedTarget, setSelectedTarget] = useState('webgui');
  const [sessionToken, setSessionToken] = useState(() => generateSessionToken());
  const [copied, setCopied] = useState(false);
  const [connected, setConnected] = useState(false);
  const [connectedAt, setConnectedAt] = useState('');
  const pollRef = useRef(null);

  // Build the knirv:// URI that the controller will scan.
  const gatewayOrigin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080';
  const knirvUri = `knirv://connect?v=1&type=${selectedTarget}&endpoint=${encodeURIComponent(gatewayOrigin)}&token=${sessionToken}&network=local`;

  // Poll /session/controller to detect when the KNIRVCONTROLLER has connected.
  useEffect(() => {
    if (connected) return;

    const check = async () => {
      try {
        const res = await fetch('/session/controller', { credentials: 'include' });
        if (!res.ok) return;
        const data = await res.json();
        if (data && data.controllerUrl) {
          setConnected(true);
          setConnectedAt(new Date().toLocaleTimeString());
          clearInterval(pollRef.current);
        }
      } catch {}
    };

    pollRef.current = setInterval(check, 3000);
    return () => clearInterval(pollRef.current);
  }, [connected, selectedTarget]);

  // Reset session when target changes.
  useEffect(() => {
    setSessionToken(generateSessionToken());
    setConnected(false);
    setConnectedAt('');
  }, [selectedTarget]);

  function regenerate() {
    setSessionToken(generateSessionToken());
    setConnected(false);
    setConnectedAt('');
  }

  function copyUri() {
    navigator.clipboard.writeText(knirvUri).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <PageLayout activePage={activePage} pageTitle="QR Connect">
      <PageHeader
        title="QR Connect"
        subtitle="Scan a QR code with KNIRVCONTROLLER to link your wallet to this gateway or a network service."
      />

      {/* Target selector */}
      <GlassyCard title="Select Connection Target">
        <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
          {TARGETS.map(t => (
            <button
              key={t.id}
              onClick={() => setSelectedTarget(t.id)}
              style={{
                flex: '1 1 160px',
                padding: '14px 16px',
                borderRadius: 10,
                border: `1px solid ${selectedTarget === t.id ? 'rgba(0,192,250,0.6)' : 'rgba(255,255,255,0.1)'}`,
                background: selectedTarget === t.id ? 'rgba(0,192,250,0.12)' : 'rgba(255,255,255,0.03)',
                color: selectedTarget === t.id ? '#00c0fa' : 'rgba(255,255,255,0.7)',
                cursor: 'pointer',
                textAlign: 'left',
                transition: 'all 0.15s',
              }}
            >
              <div style={{ fontSize: 22, marginBottom: 6 }}>{t.icon}</div>
              <div style={{ fontWeight: 600, fontSize: 13, marginBottom: 4 }}>{t.label}</div>
              <div style={{ fontSize: 11, opacity: 0.7, lineHeight: 1.4 }}>{t.description}</div>
            </button>
          ))}
        </div>
      </GlassyCard>

      {/* QR code display */}
      <GlassyCard title="Scan with KNIRVCONTROLLER">
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 20 }}>
          {/* Status badge */}
          {connected ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 16px', borderRadius: 20, background: 'rgba(46,204,113,0.15)', border: '1px solid rgba(46,204,113,0.4)' }}>
              <span style={{ width: 8, height: 8, borderRadius: '50%', background: '#2ecc71', display: 'inline-block' }} />
              <span style={{ color: '#2ecc71', fontSize: 13, fontWeight: 600 }}>Controller connected at {connectedAt}</span>
            </div>
          ) : (
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 16px', borderRadius: 20, background: 'rgba(0,192,250,0.08)', border: '1px solid rgba(0,192,250,0.2)' }}>
              <span style={{ width: 8, height: 8, borderRadius: '50%', background: '#00c0fa', display: 'inline-block', animation: 'pulse 1.4s ease-in-out infinite' }} />
              <span style={{ color: 'rgba(0,192,250,0.9)', fontSize: 13 }}>Waiting for scan…</span>
            </div>
          )}

          {/* QR */}
          <div style={{ background: '#fff', padding: 16, borderRadius: 12 }}>
            <QRCodeSVG
              value={knirvUri}
              size={220}
              level="H"
              includeMargin={false}
            />
          </div>

          <div style={{ textAlign: 'center', maxWidth: 340 }}>
            <p style={{ color: 'rgba(255,255,255,0.6)', fontSize: 13, margin: '0 0 4px' }}>
              Open KNIRVCONTROLLER → tap <strong style={{ color: '#fff' }}>Scan QR</strong>
            </p>
            <p style={{ color: 'rgba(255,255,255,0.35)', fontSize: 11, margin: 0 }}>
              Session token rotates every time you change target or regenerate.
            </p>
          </div>

          {/* URI display + copy */}
          <div style={{ width: '100%', maxWidth: 440, background: 'rgba(0,0,0,0.25)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 8, padding: '10px 14px' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
              <span style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'rgba(255,255,255,0.35)' }}>URI</span>
              <button
                onClick={copyUri}
                style={{ background: 'none', border: 'none', color: copied ? '#2ecc71' : 'rgba(255,255,255,0.5)', fontSize: 11, cursor: 'pointer', padding: '2px 6px' }}
              >
                {copied ? '✓ Copied' : 'Copy'}
              </button>
            </div>
            <code style={{ fontSize: 10, color: 'rgba(255,255,255,0.6)', wordBreak: 'break-all', display: 'block', lineHeight: 1.6 }}>
              {knirvUri}
            </code>
          </div>

          {/* Actions */}
          <div style={{ display: 'flex', gap: 10 }}>
            <button
              onClick={regenerate}
              style={{ padding: '9px 20px', border: '1px solid rgba(255,255,255,0.2)', borderRadius: 8, background: 'transparent', color: 'rgba(255,255,255,0.7)', cursor: 'pointer', fontSize: 13, transition: 'all 0.15s' }}
            >
              ↺ New Token
            </button>
          </div>
        </div>
      </GlassyCard>

      {/* How it works */}
      <GlassyCard title="How it works">
        <ol style={{ margin: 0, paddingLeft: 20, color: 'rgba(255,255,255,0.6)', fontSize: 13, lineHeight: 1.8 }}>
          <li>Select the connection target above.</li>
          <li>Open KNIRVCONTROLLER on your phone and tap <strong style={{ color: '#fff' }}>Scan QR</strong>.</li>
          <li>Point the camera at the code — the app will parse the <code style={{ color: '#00c0fa' }}>knirv://</code> URI automatically.</li>
          <li>Confirm the connection in the app. Your wallet address and role are then available in this session.</li>
          <li>The token is single-use. Tap <strong style={{ color: '#fff' }}>New Token</strong> to invalidate the current code and generate a fresh one.</li>
        </ol>
      </GlassyCard>
    </PageLayout>
  );
}
