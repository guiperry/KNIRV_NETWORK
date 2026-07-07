import { useEffect, useMemo, useRef, useState } from 'react';
import { CameraOff, LoaderCircle, QrCode } from 'lucide-react';
import { Capacitor } from '@capacitor/core';
import { scanNativeQrCode } from '@/react-app/platform/nativeQrScanner';
import { hasCameraSupport, hasSecureContext } from '@/react-app/platform/browserStorage';

interface QrScannerFrameProps {
  active: boolean;
  readerId: string;
  onScan: (decodedText: string) => void;
  onError?: (message: string) => void;
  className?: string;
  fallbackTitle?: string;
  fallbackDescription?: string;
}

export function QrScannerFrame({
  active,
  readerId,
  onScan,
  onError,
  className = '',
  fallbackTitle = 'Camera unavailable',
  fallbackDescription = 'This browser cannot access the vault scanner. Use a secure camera-enabled browser or device.',
}: QrScannerFrameProps) {
  const isNative = Capacitor.isNativePlatform();
  const scannerRef = useRef<any>(null);
  const onScanRef = useRef(onScan);
  const onErrorRef = useRef(onError);
  const [status, setStatus] = useState<'idle' | 'loading' | 'ready' | 'unsupported' | 'error'>('idle');
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  useEffect(() => {
    onScanRef.current = onScan;
  }, [onScan]);

  useEffect(() => {
    onErrorRef.current = onError;
  }, [onError]);

  const scanLabel = useMemo(() => (isNative ? 'Open Native Scanner' : 'Start Scanner'), [isNative]);

  useEffect(() => {
    if (!active) {
      setStatus('idle');
      setStatusMessage(null);
      scannerRef.current?.clear?.().catch(() => undefined);
      scannerRef.current = null;
      return undefined;
    }

    if (isNative) {
      let cancelled = false;

      const startNativeScan = async () => {
        setStatus('loading');
        try {
          const decodedText = await scanNativeQrCode();
          if (cancelled) {
            return;
          }

          onScanRef.current(decodedText);
          setStatus('ready');
        } catch (error) {
          if (cancelled) {
            return;
          }

          const message = error instanceof Error ? error.message : 'Failed to initialize the QR scanner.';
          setStatus('error');
          setStatusMessage(message);
          onErrorRef.current?.(message);
        }
      };

      void startNativeScan();

      return () => {
        cancelled = true;
      };
    }

    let cancelled = false;

    const boot = async () => {
      if (typeof window === 'undefined' || !hasSecureContext() || !hasCameraSupport()) {
        setStatus('unsupported');
        setStatusMessage('Secure camera access is required for QR signing.');
        onErrorRef.current?.('Secure camera access is required for QR signing.');
        return;
      }

      try {
        setStatus('loading');
        const module = await import('html5-qrcode');

        if (cancelled) {
          return;
        }

        const scanner = new module.Html5QrcodeScanner(
          readerId,
          { fps: 10, qrbox: { width: 260, height: 260 } },
          false,
        );

        scannerRef.current = scanner;

        scanner.render(
          (decodedText: string) => {
            onScanRef.current(decodedText);
            scannerRef.current?.clear?.().catch(() => undefined);
            scannerRef.current = null;
          },
          () => {
            // Ignore transient scan noise.
          },
        );

        setStatus('ready');
      } catch (error) {
        if (cancelled) {
          return;
        }

        const message = error instanceof Error ? error.message : 'Failed to initialize the QR scanner.';
        setStatus('error');
        setStatusMessage(message);
        onErrorRef.current?.(message);
      }
    };

    void boot();

    return () => {
      cancelled = true;
      scannerRef.current?.clear?.().catch(() => undefined);
      scannerRef.current = null;
    };
  }, [active, isNative, readerId]);

  if (!active) {
    return null;
  }

  if (isNative) {
    return (
      <div className={className}>
        <div className="rounded-xl border border-white/10 bg-[#020408]/60 p-5 text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full border border-white/10 bg-white/5">
            <QrCode className="h-5 w-5 text-[#b4c5ff]" />
          </div>
          <div className="mt-4 text-lg font-semibold text-white">Native QR Scanner Ready</div>
          <p className="mt-2 text-sm leading-6 text-slate-400">
            {status === 'error' ? statusMessage || fallbackDescription : 'The native camera will open when you start scanning.'}
          </p>
          <div className="mt-5 flex flex-col gap-3">
            <button
              type="button"
              onClick={async () => {
                setStatus('loading');
                try {
                  const decodedText = await scanNativeQrCode();
                  onScanRef.current(decodedText);
                } catch (error) {
                  const message = error instanceof Error ? error.message : 'Failed to open the native scanner.';
                  setStatus('error');
                  setStatusMessage(message);
                  onErrorRef.current?.(message);
                }
              }}
              className="flex items-center justify-center gap-2 rounded-xl bg-blue-600 px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-blue-500"
            >
              <QrCode className="h-4 w-4" />
              <span>{scanLabel}</span>
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={className}>
      <div id={readerId} className="min-h-[320px]" />

      {(status === 'loading' || status === 'idle') && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center rounded-xl border border-white/10 bg-[#020408]/70">
          <div className="flex flex-col items-center gap-3 text-center">
            <LoaderCircle className="h-6 w-6 animate-spin text-[#5de6ff]" />
            <div className="text-[12px] uppercase tracking-[0.3em] text-[#c2c6d2]">Initializing scanner</div>
          </div>
        </div>
      )}

      {(status === 'unsupported' || status === 'error') && (
        <div className="absolute inset-0 flex items-center justify-center rounded-xl border border-white/10 bg-[#020408]/85 p-6 text-center">
          <div className="max-w-sm space-y-3">
            <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full border border-white/10 bg-white/5">
              {status === 'unsupported' ? <CameraOff className="h-5 w-5 text-[#b4c5ff]" /> : <QrCode className="h-5 w-5 text-[#b4c5ff]" />}
            </div>
            <div className="text-lg font-semibold text-white">{fallbackTitle}</div>
            <p className="text-sm leading-6 text-slate-400">
              {statusMessage || fallbackDescription}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
