"use client";

import { useEffect, useState } from "react";
import { usePWAInstall } from "@/hooks/usePWAInstall";

const BTN: React.CSSProperties = {
  background: 'rgba(20, 60, 180, 0.5)',
  border: '1px solid rgba(72, 136, 255, 0.6)',
  color: '#4888ff',
  fontFamily: "'Courier New', monospace",
  fontSize: '10px',
  fontWeight: 'bold',
  letterSpacing: '2px',
  padding: '4px 12px',
  cursor: 'pointer',
  textTransform: 'uppercase' as const,
};

const CLOSE_BTN: React.CSSProperties = {
  background: 'none',
  border: 'none',
  color: 'rgba(72, 136, 255, 0.5)',
  fontFamily: "'Courier New', monospace",
  fontSize: '14px',
  cursor: 'pointer',
  padding: '2px 4px',
  lineHeight: 1,
};

const WRAP: React.CSSProperties = {
  position: 'fixed',
  bottom: '70px',
  left: '260px',
  zIndex: 1002,
  display: 'flex',
  alignItems: 'center',
  gap: '8px',
  background: 'rgba(0, 8, 24, 0.92)',
  border: '1px solid rgba(72, 136, 255, 0.6)',
  boxShadow: '0 0 20px rgba(72, 136, 255, 0.3)',
  padding: '8px 16px',
  fontFamily: "'Courier New', monospace",
  backdropFilter: 'blur(8px)',
};

const LABEL: React.CSSProperties = {
  fontSize: '10px',
  letterSpacing: '2px',
  color: 'rgba(72, 136, 255, 0.8)',
  textTransform: 'uppercase',
};

// Detect whether we're likely on iOS Safari (no beforeinstallprompt support)
function isIOSSafari() {
  if (typeof navigator === 'undefined') return false;
  return /iphone|ipad|ipod/i.test(navigator.userAgent) &&
    /safari/i.test(navigator.userAgent) &&
    !/crios|fxios/i.test(navigator.userAgent);
}

// Detect standalone mode (already installed)
function isStandalone() {
  if (typeof window === 'undefined') return false;
  return window.matchMedia('(display-mode: standalone)').matches ||
    ('standalone' in navigator && (navigator as unknown as { standalone: boolean }).standalone === true);
}

export function PWAInstallButton() {
  const { installPrompt, isInstalled, install, dismiss } = usePWAInstall();
  const [showHelp, setShowHelp] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [swReady, setSwReady] = useState(false);

  // Track service-worker readiness — a proxy for "PWA criteria met"
  useEffect(() => {
    if (!('serviceWorker' in navigator)) return;
    navigator.serviceWorker.ready.then(() => setSwReady(true)).catch(() => {});
  }, []);

  if (isInstalled || isStandalone() || dismissed) return null;

  // Chrome/Edge path: prompt available — show 1-click install banner
  if (installPrompt) {
    return (
      <div style={WRAP}>
        <span style={LABEL}>Install KNIRVSERVER Client</span>
        <button style={BTN} onClick={install}>&#8681; INSTALL</button>
        <button style={CLOSE_BTN} onClick={dismiss} title="Dismiss">&#215;</button>
      </div>
    );
  }

  // iOS Safari path: no prompt API, show "Add to Home Screen" instructions
  if (isIOSSafari() && swReady) {
    if (showHelp) {
      return (
        <div style={{ ...WRAP, flexDirection: 'column', alignItems: 'flex-start', gap: '6px', maxWidth: '340px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
            <span style={{ ...LABEL, fontSize: '11px' }}>Install on iOS</span>
            <button style={CLOSE_BTN} onClick={() => setShowHelp(false)}>&#215;</button>
          </div>
          <span style={{ fontSize: '10px', color: 'rgba(72,136,255,0.7)', lineHeight: '1.6' }}>
            Tap <strong style={{ color: '#4888ff' }}>Share &#x2197;</strong> in Safari,<br />
            then <strong style={{ color: '#4888ff' }}>Add to Home Screen</strong>.
          </span>
        </div>
      );
    }
    return (
      <div style={WRAP}>
        <span style={LABEL}>Install App</span>
        <button style={BTN} onClick={() => setShowHelp(true)}>&#8681; HOW TO</button>
        <button style={CLOSE_BTN} onClick={() => setDismissed(true)} title="Dismiss">&#215;</button>
      </div>
    );
  }

  // Chrome/Edge but prompt not yet fired (engagement heuristic, HTTP, etc.)
  // Show a manual instructions button after the SW is ready
  if (swReady && !isIOSSafari()) {
    if (showHelp) {
      return (
        <div style={{ ...WRAP, flexDirection: 'column', alignItems: 'flex-start', gap: '6px', maxWidth: '380px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
            <span style={{ ...LABEL, fontSize: '11px' }}>Install KNIRVSERVER</span>
            <button style={CLOSE_BTN} onClick={() => setShowHelp(false)}>&#215;</button>
          </div>
          <span style={{ fontSize: '10px', color: 'rgba(72,136,255,0.7)', lineHeight: '1.6' }}>
            In Chrome/Edge, click the <strong style={{ color: '#4888ff' }}>&#8862; install icon</strong> in the address bar.<br />
            Or open <strong style={{ color: '#4888ff' }}>Menu ⋮ → Install KNIRVSERVER…</strong>
          </span>
        </div>
      );
    }
    return (
      <div style={WRAP}>
        <span style={LABEL}>Install App</span>
        <button style={BTN} onClick={() => setShowHelp(true)}>&#8681; INSTALL</button>
        <button style={CLOSE_BTN} onClick={() => setDismissed(true)} title="Dismiss">&#215;</button>
      </div>
    );
  }

  return null;
}
