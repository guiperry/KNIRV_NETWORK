import React, { useEffect, useState } from 'react';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';

export default function ControllerStatus() {
  const [activePage] = useState('controller-status');
  const [sessionController, setSessionController] = useState(null);
  const [health, setHealth] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const r = await fetch('/session/controller', { credentials: 'include' });
        const j = await r.json();
        if (!mounted) return;
        setSessionController(j?.controllerUrl || null);
        if (j?.controllerUrl) {
          try {
            const h = await fetch('/controller/health', { credentials: 'include' });
            const hv = await h.json().catch(() => null);
            if (!mounted) return;
            setHealth({ ok: h.ok, body: hv });
          } catch (_) {
            if (!mounted) return;
            setHealth({ ok: false, body: null });
          }
        }
      } catch (_) {
        if (!mounted) return;
        setSessionController(null);
      } finally {
        if (mounted) setLoading(false);
      }
    })();
    return () => { mounted = false; };
  }, []);

  return (
    <PageLayout activePage={activePage} pageTitle="Controller Status" onSearch={() => {}}>
      <PageHeader title="KNIRVCONTROLLER Connection" subtitle="Session and health status" />
      <div style={{ padding: 16, color: '#e6efff' }}>
        {loading && <div>Loading...</div>}
        {!loading && (
          <>
            <div style={{ marginBottom: 12 }}>
              <strong>Status:</strong>{' '}
              {sessionController ? (
                <span style={{ color: '#28a745' }}>Connected</span>
              ) : (
                <span style={{ color: '#dc3545' }}>Disconnected</span>
              )}
            </div>
            <div style={{ marginBottom: 12 }}>
              <strong>Controller URL:</strong>{' '}
              {sessionController ? sessionController : '—'}
            </div>
            {sessionController && (
              <div style={{ marginBottom: 12 }}>
                <strong>Health:</strong>{' '}
                {health ? (health.ok ? 'OK' : 'Unavailable') : '—'}
              </div>
            )}
            {!sessionController && (
              <button onClick={() => (window.location.href = '/qr-connect')} style={{ padding: '8px 12px', borderRadius: 6, background: '#0b5cff', color: '#fff', border: '1px solid rgba(255,255,255,0.2)' }}>
                Connect via QR
              </button>
            )}
            {sessionController && (
              <button onClick={() => window.open('/controller', '_blank', 'noopener')} style={{ padding: '8px 12px', borderRadius: 6, background: 'transparent', color: '#e6efff', border: '1px solid rgba(255,255,255,0.2)', marginLeft: 8 }}>
                Open My Controller
              </button>
            )}
          </>
        )}
      </div>
    </PageLayout>
  );
}