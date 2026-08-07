import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useWalletContext } from '@hooks/use-context';
import { useCurrentAccount } from '@hooks/use-current-account';
import { RoutePath } from '@types';

// ---------------------------------------------------------------------------
// knirv:// URI scheme
//
// Connection:
//   knirv://connect?v=1&type=<target>&endpoint=<url>&token=<token>
//   type values: webgui | server | bridge | website
//
// Registration (new account, from knirv.com/get-started):
//   knirv://register?v=1&nonce=<hex>&callback=<url>&network=<id>
// ---------------------------------------------------------------------------

type ConnectTarget = 'webgui' | 'server' | 'bridge' | 'website';

interface KnirvConnectPayload {
  kind: 'connect';
  type: ConnectTarget;
  endpoint: string;
  token: string;
  network?: string;
}

interface KnirvRegisterPayload {
  kind: 'register';
  nonce: string;
  callback: string;
  network: string;
}

type KnirvPayload = KnirvConnectPayload | KnirvRegisterPayload;

function parseKnirvUri(raw: string): KnirvPayload | null {
  try {
    const url = new URL(raw);
    if (url.protocol !== 'knirv:') return null;

    const params = url.searchParams;

    if (url.hostname === 'connect') {
      const type = params.get('type') as ConnectTarget | null;
      const endpoint = params.get('endpoint');
      const token = params.get('token') ?? '';
      if (!type || !endpoint) return null;
      return { kind: 'connect', type, endpoint, token, network: params.get('network') ?? undefined };
    }

    if (url.hostname === 'register') {
      const nonce = params.get('nonce');
      const callback = params.get('callback');
      const network = params.get('network') ?? 'public-testnet';
      if (!nonce || !callback) return null;
      return { kind: 'register', nonce, callback: decodeURIComponent(callback), network };
    }

    return null;
  } catch {
    return null;
  }
}

const TARGET_LABELS: Record<ConnectTarget, { label: string; icon: string; description: string }> = {
  webgui:  { label: 'KNIRVGATEWAY WebGUI', icon: '🖥️',  description: 'Connect wallet to your local KNIRVGATEWAY dashboard.' },
  server:  { label: 'KNIRVSERVER',          icon: '🌐',  description: 'Authenticate with a KNIRVSERVER node.' },
  bridge:  { label: 'KNIRVBRIDGE',          icon: '🌉',  description: 'Connect to the KNIRVBRIDGE cross-chain service.' },
  website: { label: 'KNIRV.COM',            icon: '🌍',  description: 'Sign in to the KNIRV website.' },
};

// ---------------------------------------------------------------------------
// Hook: access the camera and decode QR frames via BarcodeDetector (modern
// browsers/Chromium extensions) with a jsQR fallback shim when unavailable.
// ---------------------------------------------------------------------------
function useQrCamera(onDecode: (raw: string) => void) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const rafRef = useRef<number>(0);
  const [cameraError, setCameraError] = useState('');

  const tick = useCallback(async (detector: BarcodeDetector | null, ctx: CanvasRenderingContext2D, canvas: HTMLCanvasElement, video: HTMLVideoElement) => {
    if (video.readyState < 2) {
      rafRef.current = requestAnimationFrame(() => tick(detector, ctx, canvas, video));
      return;
    }
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    ctx.drawImage(video, 0, 0);

    try {
      if (detector) {
        const results = await detector.detect(canvas);
        if (results.length > 0) {
          onDecode(results[0].rawValue);
          return;
        }
      }
    } catch {}

    rafRef.current = requestAnimationFrame(() => tick(detector, ctx, canvas, video));
  }, [onDecode]);

  useEffect(() => {
    let alive = true;
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d')!;

    (async () => {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } });
        if (!alive) { stream.getTracks().forEach(t => t.stop()); return; }
        streamRef.current = stream;
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          await videoRef.current.play();
        }

        let detector: BarcodeDetector | null = null;
        if ('BarcodeDetector' in window) {
          detector = new (window as unknown as { BarcodeDetector: typeof BarcodeDetector }).BarcodeDetector({ formats: ['qr_code'] });
        }
        rafRef.current = requestAnimationFrame(() => tick(detector, ctx, canvas, videoRef.current!));
      } catch (err) {
        if (alive) setCameraError((err as Error).message ?? 'Camera access denied');
      }
    })();

    return () => {
      alive = false;
      cancelAnimationFrame(rafRef.current);
      streamRef.current?.getTracks().forEach(t => t.stop());
    };
  }, [tick]);

  return { videoRef, cameraError };
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------
const QrScannerPage: React.FC = () => {
  const navigate = useNavigate();
  const { wallet } = useWalletContext();
  const { currentAccount } = useCurrentAccount();
  const [scanned, setScanned] = useState<KnirvPayload | null>(null);
  const [scanError, setScanError] = useState('');
  const [status, setStatus] = useState<'idle' | 'connecting' | 'done' | 'error'>('idle');
  const [statusMsg, setStatusMsg] = useState('');
  const decoded = useRef(false);

  const handleDecode = useCallback((raw: string) => {
    if (decoded.current) return;
    const payload = parseKnirvUri(raw);
    if (!payload) {
      setScanError(`Unrecognised QR code. Expected a knirv:// URI.\nGot: ${raw.slice(0, 80)}`);
      return;
    }
    decoded.current = true;
    setScanned(payload);
    setStatus('idle');
  }, []);

  const { videoRef, cameraError } = useQrCamera(handleDecode);

  // -----------------------------------------------------------------------
  // Execute the scanned action
  // -----------------------------------------------------------------------
  const handleConfirm = useCallback(async () => {
    if (!scanned) return;
    setStatus('connecting');

    try {
      if (scanned.kind === 'connect') {
        // Store the connection in localStorage so other parts of the extension
        // can find the active endpoint for each service type.
        const key = `knirv_conn_${scanned.type}`;
        localStorage.setItem(key, JSON.stringify({
          endpoint: scanned.endpoint,
          token: scanned.token,
          network: scanned.network ?? '',
          connectedAt: Date.now(),
        }));

        // Verify the endpoint is reachable before declaring success.
        const healthUrl = `${scanned.endpoint.replace(/\/$/, '')}/health`;
        const res = await fetch(healthUrl, { signal: AbortSignal.timeout(5000) });
        if (!res.ok) throw new Error(`Endpoint returned ${res.status}`);

        setStatusMsg(`Connected to ${TARGET_LABELS[scanned.type].label}`);
        setStatus('done');
      } else {
        if (!wallet || !currentAccount) {
          throw new Error('Unlock KNIRVCONTROLLER and select an account before approving registration');
        }

        const now = Math.floor(Date.now() / 1000);
        const signed = JSON.parse(await wallet.signOracleMessageByAccountId(currentAccount.id, {
          domain: 'knirv.controller',
          purpose: 'account-registration',
          chainId: scanned.network || 'knirv-1',
          nonce: scanned.nonce,
          issuedAtUnix: now,
          expiresAtUnix: now + 300,
          payload: new TextEncoder().encode(scanned.nonce),
        })) as { envelope: string; signature: string; public_key: string; address: string };

        const res = await fetch(scanned.callback, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            nonce: scanned.nonce,
            signature: signed.signature,
            pubkey: signed.public_key,
            address: signed.address,
            envelope: signed.envelope,
            signed_message: signed,
            role: 'General',
          }),
        });
        if (!res.ok) throw new Error(`Registration callback failed: ${res.status}`);

        setStatusMsg('Account registered! Return to KNIRV.COM to continue.');
        setStatus('done');
      }
    } catch (e) {
      setStatus('error');
      setStatusMsg((e as Error).message ?? 'Connection failed');
    }
  }, [currentAccount, scanned, wallet]);

  const reset = () => {
    decoded.current = false;
    setScanned(null);
    setScanError('');
    setStatus('idle');
    setStatusMsg('');
  };

  // -----------------------------------------------------------------------
  // Render
  // -----------------------------------------------------------------------
  return (
    <div style={styles.page}>
      {/* Header */}
      <div style={styles.header}>
        <button style={styles.backBtn} onClick={() => navigate(RoutePath.Wallet)}>←</button>
        <span style={styles.headerTitle}>Scan QR Code</span>
        <div style={{ width: 32 }} />
      </div>

      {!scanned && status === 'idle' && (
        <>
          {/* Viewfinder */}
          <div style={styles.viewfinder}>
            <video ref={videoRef} style={styles.video} playsInline muted />
            <div style={styles.overlay}>
              <div style={styles.cutout} />
              <div style={styles.corner('tl')} />
              <div style={styles.corner('tr')} />
              <div style={styles.corner('bl')} />
              <div style={styles.corner('br')} />
            </div>
          </div>

          {cameraError && (
            <div style={styles.errorBox}>{cameraError}</div>
          )}
          {scanError && (
            <div style={styles.errorBox}>{scanError}
              <button style={styles.retryBtn} onClick={reset}>Try again</button>
            </div>
          )}

          <p style={styles.hint}>
            Point at a <strong>knirv://</strong> QR from KNIRVGATEWAY, KNIRVSERVER, KNIRVBRIDGE, or KNIRV.COM
          </p>
        </>
      )}

      {/* Confirmation card */}
      {scanned && status === 'idle' && (
        <div style={styles.card}>
          {scanned.kind === 'connect' ? (
            <>
              <div style={styles.icon}>{TARGET_LABELS[scanned.type].icon}</div>
              <h3 style={styles.cardTitle}>Connect to {TARGET_LABELS[scanned.type].label}</h3>
              <p style={styles.cardDesc}>{TARGET_LABELS[scanned.type].description}</p>
              <div style={styles.field}>
                <span style={styles.fieldLabel}>Endpoint</span>
                <span style={styles.fieldValue}>{scanned.endpoint}</span>
              </div>
              {scanned.network && (
                <div style={styles.field}>
                  <span style={styles.fieldLabel}>Network</span>
                  <span style={styles.fieldValue}>{scanned.network}</span>
                </div>
              )}
            </>
          ) : (
            <>
              <div style={styles.icon}>🆕</div>
              <h3 style={styles.cardTitle}>Create KNIRV Account</h3>
              <p style={styles.cardDesc}>
                Your wallet will sign a one-time nonce to register your address on the KNIRV network.
              </p>
              <div style={styles.field}>
                <span style={styles.fieldLabel}>Network</span>
                <span style={styles.fieldValue}>{scanned.network}</span>
              </div>
            </>
          )}

          <div style={styles.btnRow}>
            <button style={styles.cancelBtn} onClick={reset}>Cancel</button>
            <button style={styles.confirmBtn} onClick={handleConfirm}>
              {scanned.kind === 'connect' ? 'Connect' : 'Sign & Register'}
            </button>
          </div>
        </div>
      )}

      {/* Connecting spinner */}
      {status === 'connecting' && (
        <div style={styles.statusBox}>
          <div style={styles.spinner} />
          <p style={styles.statusText}>
            {scanned?.kind === 'register' ? 'Signing & registering…' : 'Connecting…'}
          </p>
        </div>
      )}

      {/* Done */}
      {status === 'done' && (
        <div style={styles.card}>
          <div style={styles.icon}>✅</div>
          <h3 style={styles.cardTitle}>Success</h3>
          <p style={styles.cardDesc}>{statusMsg}</p>
          <button style={{ ...styles.confirmBtn, width: '100%' }} onClick={() => navigate(RoutePath.Wallet)}>
            Go to Wallet
          </button>
        </div>
      )}

      {/* Error */}
      {status === 'error' && (
        <div style={styles.card}>
          <div style={styles.icon}>❌</div>
          <h3 style={styles.cardTitle}>Failed</h3>
          <p style={styles.cardDesc}>{statusMsg}</p>
          <button style={{ ...styles.confirmBtn, width: '100%' }} onClick={reset}>Try again</button>
        </div>
      )}
    </div>
  );
};

// ---------------------------------------------------------------------------
// Inline styles (matches extension's dark theme without requiring styled-
// components or CSS modules — keeping dependencies minimal)
// ---------------------------------------------------------------------------
const C = {
  bg: '#1a1a2e',
  surface: '#16213e',
  border: '#2a2a4a',
  primary: '#00c0fa',
  danger: '#e74c3c',
  text: '#e0e0ff',
  muted: 'rgba(224,224,255,0.5)',
} as const;

const styles = {
  page: { display: 'flex', flexDirection: 'column' as const, height: '100%', background: C.bg, color: C.text, fontFamily: 'system-ui, sans-serif' },
  header: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 16px', borderBottom: `1px solid ${C.border}` },
  headerTitle: { fontWeight: 600, fontSize: 16 },
  backBtn: { background: 'none', border: 'none', color: C.text, fontSize: 20, cursor: 'pointer', padding: '0 4px', lineHeight: 1 },
  viewfinder: { position: 'relative' as const, flex: 1, background: '#000', overflow: 'hidden', maxHeight: 300 },
  video: { width: '100%', height: '100%', objectFit: 'cover' as const, display: 'block' },
  overlay: { position: 'absolute' as const, inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' },
  cutout: { width: 200, height: 200, border: `2px solid ${C.primary}`, borderRadius: 12, boxShadow: '0 0 0 9999px rgba(0,0,0,0.5)' },
  corner: (pos: 'tl'|'tr'|'bl'|'br') => ({
    position: 'absolute' as const,
    width: 20, height: 20,
    borderColor: C.primary, borderStyle: 'solid' as const,
    ...(pos === 'tl' ? { top: 'calc(50% - 100px)', left: 'calc(50% - 100px)', borderWidth: '3px 0 0 3px', borderRadius: '6px 0 0 0' } : {}),
    ...(pos === 'tr' ? { top: 'calc(50% - 100px)', right: 'calc(50% - 100px)', borderWidth: '3px 3px 0 0', borderRadius: '0 6px 0 0' } : {}),
    ...(pos === 'bl' ? { bottom: 'calc(50% - 100px)', left: 'calc(50% - 100px)', borderWidth: '0 0 3px 3px', borderRadius: '0 0 0 6px' } : {}),
    ...(pos === 'br' ? { bottom: 'calc(50% - 100px)', right: 'calc(50% - 100px)', borderWidth: '0 3px 3px 0', borderRadius: '0 0 6px 0' } : {}),
  }),
  hint: { textAlign: 'center' as const, fontSize: 12, color: C.muted, padding: '12px 20px' },
  errorBox: { margin: '8px 16px', padding: '10px 14px', background: 'rgba(231,76,60,0.15)', border: `1px solid ${C.danger}`, borderRadius: 8, fontSize: 12, color: C.danger, display: 'flex', flexDirection: 'column' as const, gap: 8 },
  retryBtn: { alignSelf: 'flex-start' as const, background: 'none', border: `1px solid ${C.danger}`, color: C.danger, borderRadius: 4, padding: '4px 10px', cursor: 'pointer', fontSize: 11 },
  card: { margin: 16, padding: 20, background: C.surface, border: `1px solid ${C.border}`, borderRadius: 16, display: 'flex', flexDirection: 'column' as const, gap: 12 },
  icon: { fontSize: 36, textAlign: 'center' as const },
  cardTitle: { fontSize: 17, fontWeight: 700, textAlign: 'center' as const, margin: 0 },
  cardDesc: { fontSize: 13, color: C.muted, textAlign: 'center' as const, margin: 0 },
  field: { display: 'flex', flexDirection: 'column' as const, gap: 2, background: 'rgba(255,255,255,0.04)', borderRadius: 8, padding: '8px 12px' },
  fieldLabel: { fontSize: 10, textTransform: 'uppercase' as const, letterSpacing: '0.5px', color: C.muted },
  fieldValue: { fontSize: 12, fontFamily: 'monospace', wordBreak: 'break-all' as const },
  btnRow: { display: 'flex', gap: 10, marginTop: 4 },
  cancelBtn: { flex: 1, padding: '10px 0', background: 'transparent', border: `1px solid ${C.border}`, color: C.text, borderRadius: 8, cursor: 'pointer', fontWeight: 600 },
  confirmBtn: { flex: 1, padding: '10px 0', background: C.primary, border: 'none', color: '#000', borderRadius: 8, cursor: 'pointer', fontWeight: 700 },
  statusBox: { flex: 1, display: 'flex', flexDirection: 'column' as const, alignItems: 'center', justifyContent: 'center', gap: 16 },
  spinner: { width: 40, height: 40, border: `3px solid rgba(0,192,250,0.2)`, borderTop: `3px solid ${C.primary}`, borderRadius: '50%', animation: 'spin 0.8s linear infinite' },
  statusText: { color: C.muted, fontSize: 14 },
};

export default QrScannerPage;
