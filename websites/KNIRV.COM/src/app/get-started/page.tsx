'use client';

import React, { useEffect, useRef, useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import KnirvLogo from '@/components/KnirvLogo';
import Link from 'next/link';

// ---------------------------------------------------------------------------
// Cloudflare CDN download URLs — swap these for real R2 / Pages URLs once
// the builds are uploaded.
// ---------------------------------------------------------------------------
const CDN_ANDROID = 'https://cdn.knirv.com/controller/android/knirvcontroller-latest.apk';
const CDN_IOS_MANIFEST = 'itms-services://?action=download-manifest&url=https://cdn.knirv.com/controller/ios/manifest.plist';
const CDN_IOS_TESTFLIGHT = 'https://testflight.apple.com/join/KNIRV_TF_TOKEN'; // replace with real token

// ---------------------------------------------------------------------------
// Detect mobile platform from user-agent (client-side only)
// ---------------------------------------------------------------------------
function detectPlatform(): 'android' | 'ios' | 'desktop' {
  if (typeof navigator === 'undefined') return 'desktop';
  const ua = navigator.userAgent.toLowerCase();
  if (/iphone|ipad|ipod/.test(ua)) return 'ios';
  if (/android/.test(ua)) return 'android';
  return 'desktop';
}

type Step = 'download' | 'auth' | 'done';

export default function GetStarted() {
  const [platform, setPlatform] = useState<'android' | 'ios' | 'desktop'>('desktop');
  const [step, setStep] = useState<Step>('download');
  const [sessionId, setSessionId] = useState('');
  const [nonce, setNonce] = useState('');
  const [authQr, setAuthQr] = useState('');
  const [authAddress, setAuthAddress] = useState('');
  const [authRole, setAuthRole] = useState('');
  const [authToken, setAuthToken] = useState('');
  const [error, setError] = useState('');
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    setPlatform(detectPlatform());
  }, []);

  // -----------------------------------------------------------------------
  // Step 1 → Step 2: initialise a registration session and build the auth QR
  // -----------------------------------------------------------------------
  async function initAuth() {
    setError('');
    try {
      const res = await fetch('/api/register/init', { method: 'POST' });
      if (!res.ok) throw new Error('Failed to start session');
      const { nonce: n, sessionId: sid } = await res.json();
      setNonce(n);
      setSessionId(sid);

      // knirv://register encodes everything the controller needs to sign and
      // POST back to /api/register/verify.
      const qr = `knirv://register?v=1&nonce=${n}&callback=${encodeURIComponent('https://knirv.com/api/register/verify')}&network=public-testnet`;
      setAuthQr(qr);
      setStep('auth');
      startPolling(sid);
    } catch (e: unknown) {
      setError((e as Error).message ?? 'Unknown error');
    }
  }

  function startPolling(sid: string) {
    if (pollRef.current) clearInterval(pollRef.current);
    pollRef.current = setInterval(async () => {
      try {
        const res = await fetch(`/api/register/verify?sessionId=${sid}`);
        if (!res.ok) return;
        const data = await res.json();
        if (data.completed) {
          clearInterval(pollRef.current!);
          setAuthAddress(data.address);
          setAuthRole(data.role);
          setAuthToken(data.token);
          setStep('done');
        }
      } catch {}
    }, 2000);
  }

  useEffect(() => () => { if (pollRef.current) clearInterval(pollRef.current); }, []);

  // -----------------------------------------------------------------------
  // Download QR value — platform-aware
  // -----------------------------------------------------------------------
  const downloadQr =
    platform === 'android' ? CDN_ANDROID :
    platform === 'ios' ? CDN_IOS_MANIFEST :
    CDN_ANDROID; // desktop shows Android by default; user can switch

  const [showIos, setShowIos] = useState(platform === 'ios');
  const activeDownloadQr = showIos ? CDN_IOS_MANIFEST : CDN_ANDROID;
  const activeDownloadLabel = showIos
    ? 'Scan with iPhone camera to install via TestFlight'
    : 'Scan with Android camera to download APK';

  return (
    <div className="dve-page min-h-screen">
      {/* Nav */}
      <nav className="dve-nav">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <KnirvLogo />
            <Link href="/pricing" className="text-white/60 hover:text-white text-sm transition-colors">
              ← Back to pricing
            </Link>
          </div>
        </div>
      </nav>

      <div className="max-w-3xl mx-auto px-4 py-16">
        {/* Header */}
        <div className="text-center mb-12">
          <h1 className="text-4xl md:text-5xl font-bold knirv-gradient-text mb-4">
            Get Started with KNIRV
          </h1>
          <p className="text-white/60 text-lg">
            KNIRVCONTROLLER is your NRN wallet and signing tool for the KNIRV network.
            Follow the two steps below to install it and create your account.
          </p>
        </div>

        {/* Progress */}
        <div className="flex items-center justify-center gap-4 mb-12">
          {(['download', 'auth', 'done'] as Step[]).map((s, i) => (
            <React.Fragment key={s}>
              <div className={`flex items-center gap-2 ${step === s ? 'text-white' : step === 'done' || (s === 'download' && step !== 'download') || (s === 'auth' && step === 'done') ? 'text-green-400' : 'text-white/30'}`}>
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold border-2 ${
                  step === s ? 'border-blue-400 bg-blue-400/20' :
                  (s === 'download' && (step === 'auth' || step === 'done')) || (s === 'auth' && step === 'done')
                    ? 'border-green-400 bg-green-400/20' : 'border-white/20'
                }`}>
                  {(s === 'download' && (step === 'auth' || step === 'done')) || (s === 'auth' && step === 'done')
                    ? '✓' : i + 1}
                </div>
                <span className="text-sm font-medium hidden sm:block">
                  {s === 'download' ? 'Install App' : s === 'auth' ? 'Create Account' : 'Ready'}
                </span>
              </div>
              {i < 2 && <div className="flex-1 max-w-16 h-px bg-white/20" />}
            </React.Fragment>
          ))}
        </div>

        {/* ── STEP 1: Download ── */}
        {step === 'download' && (
          <div className="bg-white/5 border border-white/10 rounded-2xl p-8 text-center">
            <h2 className="text-2xl font-bold text-white mb-2">Step 1 — Install KNIRVCONTROLLER</h2>
            <p className="text-white/60 mb-8">
              Scan the QR code with your phone camera to download and install the app.
            </p>

            {/* Platform toggle */}
            <div className="flex justify-center mb-6">
              <div className="flex bg-white/5 border border-white/10 rounded-lg p-1 gap-1">
                <button
                  onClick={() => setShowIos(false)}
                  className={`px-4 py-2 rounded-md text-sm font-medium transition-all ${!showIos ? 'bg-blue-500/30 text-white border border-blue-400/40' : 'text-white/50 hover:text-white'}`}
                >
                  Android
                </button>
                <button
                  onClick={() => setShowIos(true)}
                  className={`px-4 py-2 rounded-md text-sm font-medium transition-all ${showIos ? 'bg-blue-500/30 text-white border border-blue-400/40' : 'text-white/50 hover:text-white'}`}
                >
                  iOS
                </button>
              </div>
            </div>

            {/* QR code */}
            <div className="flex justify-center mb-6">
              <div className="bg-white p-4 rounded-xl inline-block">
                <QRCodeSVG value={activeDownloadQr} size={200} level="M" />
              </div>
            </div>

            <p className="text-white/50 text-sm mb-2">{activeDownloadLabel}</p>

            {showIos && (
              <p className="text-white/40 text-xs mb-6">
                iOS distribution requires TestFlight. The QR links directly to our TestFlight invite.
              </p>
            )}
            {!showIos && (
              <p className="text-white/40 text-xs mb-6">
                Android: enable "Install unknown apps" for your browser in Settings → Apps before installing the APK.
              </p>
            )}

            <button
              onClick={initAuth}
              className="w-full py-3 px-6 rounded-xl bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold hover:from-blue-600 hover:to-purple-700 transition-all"
            >
              I've installed it — Create my account →
            </button>
            {error && <p className="text-red-400 text-sm mt-3">{error}</p>}
          </div>
        )}

        {/* ── STEP 2: Auth / account creation ── */}
        {step === 'auth' && (
          <div className="bg-white/5 border border-white/10 rounded-2xl p-8 text-center">
            <h2 className="text-2xl font-bold text-white mb-2">Step 2 — Create Your Account</h2>
            <p className="text-white/60 mb-2">
              Open KNIRVCONTROLLER, tap <strong className="text-white">Scan QR</strong>, then point your camera at this code.
            </p>
            <p className="text-white/40 text-sm mb-8">
              Your wallet signs the registration with your private key — no password needed.
            </p>

            <div className="flex justify-center mb-6">
              <div className="bg-white p-4 rounded-xl inline-block relative">
                <QRCodeSVG value={authQr} size={220} level="H" />
                {/* Pulsing ring to draw attention */}
                <div className="absolute inset-0 rounded-xl ring-2 ring-blue-400 animate-pulse pointer-events-none" />
              </div>
            </div>

            <div className="flex items-center justify-center gap-2 text-white/50 text-sm">
              <span className="w-2 h-2 rounded-full bg-blue-400 animate-pulse inline-block" />
              Waiting for KNIRVCONTROLLER to scan…
            </div>

            <p className="text-white/30 text-xs mt-4">
              Session expires in 10 minutes. The QR encodes a one-time nonce — it is safe to display.
            </p>
          </div>
        )}

        {/* ── STEP 3: Done ── */}
        {step === 'done' && (
          <div className="bg-white/5 border border-green-400/20 rounded-2xl p-8 text-center">
            <div className="text-5xl mb-4">🎉</div>
            <h2 className="text-2xl font-bold text-white mb-2">Account Created!</h2>
            <p className="text-white/60 mb-6">
              Your KNIRV account is live. Your wallet address and role are shown below.
            </p>

            <div className="bg-white/5 border border-white/10 rounded-xl p-4 mb-6 text-left">
              <div className="flex justify-between items-center mb-2">
                <span className="text-white/50 text-xs uppercase tracking-wider">Wallet Address</span>
              </div>
              <p className="text-white font-mono text-sm break-all">{authAddress}</p>
              <div className="flex justify-between items-center mt-3 mb-1">
                <span className="text-white/50 text-xs uppercase tracking-wider">Role</span>
              </div>
              <p className="text-green-400 font-semibold">{authRole}</p>
            </div>

            <div className="flex flex-col sm:flex-row gap-3 justify-center">
              <Link
                href={`/dashboard?token=${authToken}&role=${authRole}`}
                className="py-3 px-6 rounded-xl bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold hover:from-blue-600 hover:to-purple-700 transition-all text-center"
              >
                Go to Dashboard →
              </Link>
              <Link
                href="/pricing"
                className="py-3 px-6 rounded-xl bg-white/10 hover:bg-white/20 text-white font-semibold border border-white/20 transition-all text-center"
              >
                View Plans
              </Link>
            </div>
          </div>
        )}

        {/* What KNIRVCONTROLLER does */}
        <div className="mt-12 grid grid-cols-1 sm:grid-cols-2 gap-4">
          {[
            { icon: '🔐', title: 'Sign Transactions', body: 'Every KNIRV operation — DVE policy commits, NRN transfers, governance votes — is signed by your controller.' },
            { icon: '🌐', title: 'Connect Anywhere', body: 'Scan a QR from KNIRVSERVER, KNIRVGATEWAY, KNIRVBRIDGE, or the KNIRV website to authenticate instantly.' },
            { icon: '💰', title: 'NRN Wallet', body: 'Hold, send, and receive NRN tokens. Fund your DVEs directly from the app.' },
            { icon: '📱', title: 'Mobile-First', body: 'Available for Android and iOS. Works offline for signing; only needs connectivity to broadcast.' },
          ].map(({ icon, title, body }) => (
            <div key={title} className="bg-white/5 border border-white/10 rounded-xl p-4 flex gap-3">
              <span className="text-2xl flex-shrink-0">{icon}</span>
              <div>
                <h3 className="text-white font-semibold mb-1">{title}</h3>
                <p className="text-white/50 text-sm">{body}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
