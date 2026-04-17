"use client";

import { usePWAInstall } from "@/hooks/usePWAInstall";

export function PWAInstallButton() {
  const { installPrompt, isInstalled, install, dismiss } = usePWAInstall();

  if (isInstalled || !installPrompt) {
    return null;
  }

  return (
    <div
      style={{
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
      }}
    >
      <span
        style={{
          fontSize: '10px',
          letterSpacing: '2px',
          color: 'rgba(72, 136, 255, 0.8)',
          textTransform: 'uppercase',
        }}
      >
        Install KNIRVSERVER Client
      </span>
      <button
        onClick={install}
        style={{
          background: 'rgba(20, 60, 180, 0.5)',
          border: '1px solid rgba(72, 136, 255, 0.6)',
          color: '#4888ff',
          fontFamily: "'Courier New', monospace",
          fontSize: '10px',
          fontWeight: 'bold',
          letterSpacing: '2px',
          padding: '4px 12px',
          cursor: 'pointer',
          transition: 'all 0.2s',
          textTransform: 'uppercase',
        }}
      >
        &#8681; INSTALL
      </button>
      <button
        onClick={dismiss}
        style={{
          background: 'none',
          border: 'none',
          color: 'rgba(72, 136, 255, 0.5)',
          fontFamily: "'Courier New', monospace",
          fontSize: '14px',
          cursor: 'pointer',
          padding: '2px 4px',
          lineHeight: 1,
        }}
        title="Dismiss"
      >
        &#215;
      </button>
    </div>
  );
}
