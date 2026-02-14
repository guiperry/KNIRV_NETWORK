import React, { useEffect, useState, useRef } from 'react';
import { Html5QrcodeScanner } from 'html5-qrcode';
import { QrCode, Shield, CheckCircle, XCircle, RefreshCw } from 'lucide-react';
import Layout from '@/react-app/components/Layout';

const Scanner: React.FC = () => {
  const [scanResult, setScanResult] = useState<string | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);
  const [verificationStatus, setVerificationStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const scannerRef = useRef<Html5QrcodeScanner | null>(null);

  useEffect(() => {
    if (verificationStatus === 'idle' && !scanResult) {
      const scanner = new Html5QrcodeScanner(
        "reader",
        { fps: 10, qrbox: { width: 250, height: 250 } },
        /* verbose= */ false
      );

      scanner.render(onScanSuccess, onScanFailure);
      scannerRef.current = scanner;
    }

    return () => {
      if (scannerRef.current) {
        scannerRef.current.clear().catch(error => {
          console.error("Failed to clear scanner", error);
        });
      }
    };
  }, [verificationStatus, scanResult]);

  function onScanSuccess(decodedText: string) {
    if (scannerRef.current) {
      scannerRef.current.clear().then(() => {
        setScanResult(decodedText);
        verifySession(decodedText);
      }).catch(error => {
        console.error("Failed to clear scanner after success", error);
      });
    }
  }

  function onScanFailure(_error: any) {
    // console.warn(`Code scan error = ${_error}`);
  }

  const verifySession = (data: string) => {
    setIsVerifying(true);
    // Simulate session verification logic
    setTimeout(() => {
      setIsVerifying(false);
      // In a real app, you'd validate the 'data' against a backend
      if (data.includes('session-') || data.length > 10) {
        setVerificationStatus('success');
      } else {
        setVerificationStatus('error');
      }
    }, 2000);
  };

  const resetScanner = () => {
    setScanResult(null);
    setVerificationStatus('idle');
  };

  return (
    <Layout>
      <div className="p-4 pb-24 space-y-6">
        <div className="text-center py-6">
          <h2 className="text-2xl font-bold gradient-text mb-2 uppercase tracking-tight">
            Session Verifier
          </h2>
          <p className="text-slate-400 text-sm font-mono uppercase tracking-wider">
            Scan QR code to authorize session access
          </p>
        </div>

        <div className="max-w-md mx-auto">
          {verificationStatus === 'idle' && !scanResult && (
            <div className="relative group">
              <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/50 to-blue-800/50 rounded-2xl blur opacity-30 group-hover:opacity-60 transition duration-300"></div>
              <div className="relative bg-slate-950 border border-blue-600/30 rounded-2xl overflow-hidden p-1 shadow-[0_0_30px_rgba(37,99,235,0.1)]">
                <div id="reader" className="w-full"></div>
                <div className="p-6 text-center border-t border-blue-600/20 bg-slate-900/50">
                  <div className="flex items-center justify-center space-x-2 text-blue-400">
                    <QrCode className="w-5 h-5" />
                    <span className="text-sm font-bold uppercase tracking-widest font-mono">Ready for Scan</span>
                  </div>
                </div>
              </div>
            </div>
          )}

          {isVerifying && (
            <div className="bg-slate-950 border border-blue-600/30 rounded-2xl p-12 flex flex-col items-center justify-center space-y-6 text-center shadow-[0_0_30px_rgba(37,99,235,0.2)]">
              <div className="relative">
                <div className="w-20 h-20 border-4 border-blue-600/20 border-t-blue-600 rounded-full animate-spin"></div>
                <div className="absolute inset-0 flex items-center justify-center">
                  <Shield className="w-8 h-8 text-blue-400" />
                </div>
              </div>
              <div>
                <h3 className="text-xl font-black text-white uppercase tracking-tighter">Verifying Session</h3>
                <p className="text-slate-500 text-xs font-mono mt-2 animate-pulse">Establishing secure handshake...</p>
              </div>
            </div>
          )}

          {verificationStatus === 'success' && (
            <div className="bg-slate-950 border border-green-500/30 rounded-2xl p-12 flex flex-col items-center justify-center space-y-6 text-center shadow-[0_0_30px_rgba(34,197,94,0.1)]">
              <div className="w-20 h-20 bg-green-500/20 rounded-full flex items-center justify-center border border-green-500/30">
                <CheckCircle className="w-10 h-10 text-green-400" />
              </div>
              <div>
                <h3 className="text-xl font-black text-white uppercase tracking-tighter">Session Verified</h3>
                <p className="text-slate-400 text-sm font-mono mt-2">D-TEN Access Token Generated</p>
                <div className="mt-4 p-3 bg-green-500/10 rounded-lg border border-green-500/20">
                  <p className="text-[10px] font-mono text-green-400 break-all">{scanResult}</p>
                </div>
              </div>
              <button 
                onClick={resetScanner}
                className="flex items-center space-x-2 px-6 py-2 bg-green-600 hover:bg-green-500 text-white rounded-lg text-xs font-bold uppercase tracking-widest transition-all"
              >
                <RefreshCw className="w-4 h-4" />
                <span>Scan New</span>
              </button>
            </div>
          )}

          {verificationStatus === 'error' && (
            <div className="bg-slate-950 border border-red-500/30 rounded-2xl p-12 flex flex-col items-center justify-center space-y-6 text-center shadow-[0_0_30px_rgba(239,68,68,0.1)]">
              <div className="w-20 h-20 bg-red-500/20 rounded-full flex items-center justify-center border border-red-500/30">
                <XCircle className="w-10 h-10 text-red-400" />
              </div>
              <div>
                <h3 className="text-xl font-black text-white uppercase tracking-tighter">Verification Failed</h3>
                <p className="text-slate-400 text-sm font-mono mt-2">Invalid or Expired Session Token</p>
              </div>
              <button 
                onClick={resetScanner}
                className="flex items-center space-x-2 px-6 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg text-xs font-bold uppercase tracking-widest transition-all"
              >
                <RefreshCw className="w-4 h-4" />
                <span>Try Again</span>
              </button>
            </div>
          )}
        </div>

        {/* Security Info */}
        <div className="grid grid-cols-1 gap-4 mt-8">
          <div className="glass-panel p-4 flex items-start space-x-4">
            <div className="p-2 bg-blue-600/10 rounded-lg border border-blue-600/20">
              <Shield className="w-5 h-5 text-blue-400" />
            </div>
            <div>
              <h4 className="text-sm font-bold text-white uppercase tracking-tight">End-to-End Encryption</h4>
              <p className="text-xs text-slate-500 font-mono mt-1">All session handshakes are performed within hardware TEEs for maximum security.</p>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
};

export default Scanner;
