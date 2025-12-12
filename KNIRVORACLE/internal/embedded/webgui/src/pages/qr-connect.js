import React, { useEffect, useState } from 'react';

// Minimal QR Connect page: accepts a controller URL via query or manual input
// Future enhancement: display QR and parse scanned payloads
export default function QRConnectPage() {
  const [controllerUrl, setControllerUrl] = useState('');
  const [status, setStatus] = useState('');

  useEffect(() => {
    // Allow setting via query ?controllerUrl=https://...
    try {
      const url = new URL(window.location.href);
      const q = url.searchParams.get('controllerUrl');
      if (q) setControllerUrl(q);
    } catch (_) {}
  }, []);

  async function save() {
    setStatus('');
    try {
      const r = await fetch('/session/controller', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ controllerUrl })
      });
      const j = await r.json();
      if (!r.ok) throw new Error(j?.error || 'Failed');
      setStatus('Saved. You can now open My API Endpoints.');
    } catch (e) {
      setStatus(`Error: ${e.message}`);
    }
  }

  return (
    <div style={styles.page}>
      <h1 style={styles.title}>QR Connect</h1>
      <p style={styles.desc}>Paste your KNIRVCONTROLLER base URL (e.g., http://localhost:3000) or arrive via QR deep link.</p>

      <div style={styles.row}>
        <input
          style={styles.input}
          value={controllerUrl}
          onChange={(e) => setControllerUrl(e.target.value)}
          placeholder="https://your-controller-host"
        />
        <button style={styles.button} onClick={save}>Save</button>
      </div>

      {status && <div style={styles.status}>{status}</div>}

      <div style={styles.tip}>
        Tip: Once saved, "My API Endpoints" will open your controller at <code>/controller</code> through the oracle.
      </div>
    </div>
  );
}

const styles = {
  page: { padding: 24, color: '#e6efff' },
  title: { fontSize: 24, marginBottom: 8 },
  desc: { opacity: 0.8 },
  row: { display: 'flex', gap: 8, marginTop: 16 },
  input: {
    flex: 1,
    padding: '10px 12px',
    borderRadius: 8,
    border: '1px solid rgba(255,255,255,0.2)',
    background: 'rgba(255,255,255,0.06)',
    color: '#e6efff'
  },
  button: {
    padding: '10px 16px',
    borderRadius: 8,
    border: '1px solid rgba(255,255,255,0.2)',
    background: '#0b5cff',
    color: 'white',
    cursor: 'pointer'
  },
  status: { marginTop: 12 },
  tip: { marginTop: 16, opacity: 0.8 }
};