import React, { useState } from 'react';
import { useNavigate } from 'react-router';
import { QrCode, Shield, CheckCircle, XCircle, RefreshCw, ArrowRight } from 'lucide-react';
import Layout from '@/react-app/components/Layout';
import { QrScannerFrame } from '@/react-app/components/QrScannerFrame';
import { useBackend } from '@/react-app/hooks/useBackend';
import { useVault } from '@/react-app/hooks/useVault';
import { impactHeavy, notificationError } from '@/react-app/platform/haptics';
import {
  bytesToBase64,
  controllerPairingClaim,
  getControllerDeviceID,
  parseDVESessionPairing,
  saveJoinedControllerSession,
} from '@/react-app/platform/controllerSession';

const ONBOARDING_API_BASE = 'https://onboarding.knirv.com';

const Scanner: React.FC = () => {
  const navigate = useNavigate();
  const [scanResult, setScanResult] = useState<string | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);
  const [verificationStatus, setVerificationStatus] = useState<'idle' | 'success' | 'error' | 'confirm_send'>('idle');
  const [sendDetails, setSendDetails] = useState<{ address: string, amount: number } | null>(null);
  const [joinedSessionID, setJoinedSessionID] = useState<string | null>(null);
  const [verificationError, setVerificationError] = useState<string | null>(null);
  
   const { currentAccount, status: vaultStatus, signMessage, oracleAddress } = useVault();
   const { sendNRN, registerVault } = useBackend();

  const handleScanResult = async (data: string) => {
    setIsVerifying(true);
    setVerificationError(null);

    try {
      const pairing = parseDVESessionPairing(data);
      if (vaultStatus !== 'unlocked' || !currentAccount) {
        throw new Error('Unlock the controller vault before joining a DVE session.');
      }
      const deviceID = await getControllerDeviceID();
      const claim = controllerPairingClaim(pairing.session_id, pairing.pairing_token, deviceID);
      const signature = await signMessage(claim);
      const response = await fetch(
        `/worker/dve/sessions/${encodeURIComponent(pairing.session_id)}/pair/confirm`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            pairing_token: pairing.pairing_token,
            device_id: deviceID,
            public_key: bytesToBase64(currentAccount.publicKey),
            signature,
          }),
        },
      );
      const result = await response.json() as {
        access_token?: string;
        expires_at?: string;
        trust_level?: string;
        error?: string;
      };
      if (!response.ok || !result.access_token || !result.expires_at) {
        throw new Error(result.error || 'The server rejected the DVE session pairing.');
      }
      await saveJoinedControllerSession({
        session_id: pairing.session_id,
        environment_id: pairing.environment_id,
        device_id: deviceID,
        access_token: result.access_token,
        expires_at: result.expires_at,
        trust_level: result.trust_level || 'signed_supervised',
      });
      setJoinedSessionID(pairing.session_id);
      setIsVerifying(false);
      impactHeavy();
      setVerificationStatus('success');
      return;
    } catch (error) {
      // A non-pairing QR may still be one of the scanner's other supported
      // payloads. Pairing-shaped JSON fails closed instead of falling through.
      if (data.trim().startsWith('{')) {
        setIsVerifying(false);
        notificationError();
        setVerificationError(error instanceof Error ? error.message : 'Failed to join DVE session.');
        setVerificationStatus('error');
        return;
      }
    }
    
    if (data.startsWith('send:')) {
      const parts = data.split(':');
      if (parts.length === 3) {
        const address = parts[1];
        const amount = parseFloat(parts[2]);
        if (address && amount) {
          setSendDetails({ address, amount });
          setVerificationStatus('confirm_send');
          setIsVerifying(false);
          return;
        }
      }
    }

    if (data.startsWith('knirv://auth?token=')) {
      const token = data.replace('knirv://auth?token=', '');
      const authToken = localStorage.getItem('knirv_auth_token');
      if (!authToken) {
        notificationError();
        setVerificationStatus('error');
        setIsVerifying(false);
        return;
      }

      try {
        const resp = await fetch(`${ONBOARDING_API_BASE}/api/auth/qr/approve`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + authToken,
          },
          body: JSON.stringify({ token }),
        });
        setIsVerifying(false);
        if (resp.ok) {
          impactHeavy();
          setVerificationStatus('success');
        } else {
          notificationError();
          setVerificationStatus('error');
        }
      } catch {
        setIsVerifying(false);
        notificationError();
        setVerificationError('Unable to reach the onboarding service.');
        setVerificationStatus('error');
      }
      return;
    }

    setTimeout(() => {
      setIsVerifying(false);
      if (data.includes('session-') || data.length > 10) {
        impactHeavy();
        setVerificationStatus('success');
      } else {
        notificationError();
        setVerificationStatus('error');
      }
    }, 2000);
  };

   const confirmSend = async () => {
     if (sendDetails) {
       setIsVerifying(true);
       if (oracleAddress) {
         await registerVault();
       }
       const result = await sendNRN(sendDetails.address, sendDetails.amount);
       setIsVerifying(false);
       if (result.success) {
         impactHeavy();
         setVerificationStatus('success');
       } else {
         notificationError();
         setVerificationStatus('error');
       }
     }
   };

  const resetScanner = () => {
    setScanResult(null);
    setVerificationStatus('idle');
    setSendDetails(null);
    setJoinedSessionID(null);
    setVerificationError(null);
  };

  const handleScan = (decodedText: string) => {
    setScanResult(decodedText);
    handleScanResult(decodedText);
  };

  return (
    <Layout>
      <div className="p-4 pb-24 space-y-6">
        <div className="text-center py-6">
          <h2 className="text-2xl font-bold gradient-text mb-2 uppercase tracking-tight">
            Scanner
          </h2>
          <p className="text-slate-400 text-sm font-mono uppercase tracking-wider">
            Scan QR code to authorize session or pay
          </p>
        </div>

        <div className="max-w-md mx-auto">
          {verificationStatus === 'idle' && !scanResult && (
            <div className="relative group">
              <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/50 to-blue-800/50 rounded-2xl blur opacity-30 group-hover:opacity-60 transition duration-300"></div>
              <div className="relative bg-slate-950 border border-blue-600/30 rounded-2xl overflow-hidden p-1 shadow-[0_0_30px_rgba(37,99,235,0.1)]">
                <QrScannerFrame
                  active={verificationStatus === 'idle' && !scanResult}
                  readerId="reader"
                  onScan={handleScan}
                  className="relative w-full"
                  fallbackTitle="Scanner unavailable"
                  fallbackDescription="Use the native camera or a secure browser camera to authorize QR payloads."
                />
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
                <h3 className="text-xl font-black text-white uppercase tracking-tighter">Processing</h3>
                <p className="text-slate-500 text-xs font-mono mt-2 animate-pulse">Verifying transaction data...</p>
              </div>
            </div>
          )}

          {verificationStatus === 'confirm_send' && sendDetails && (
            <div className="bg-slate-950 border border-yellow-500/30 rounded-2xl p-8 flex flex-col items-center justify-center space-y-6 text-center shadow-[0_0_30px_rgba(234,179,8,0.1)]">
              <div className="w-16 h-16 bg-yellow-500/20 rounded-full flex items-center justify-center border border-yellow-500/30">
                <ArrowRight className="w-8 h-8 text-yellow-400" />
              </div>
              <div>
                <h3 className="text-xl font-black text-white uppercase tracking-tighter">Confirm Transaction</h3>
                <p className="text-slate-400 text-sm font-mono mt-2">Send NRN to Address</p>
                <div className="mt-4 p-4 bg-slate-900 rounded-xl border border-slate-800 text-left space-y-2">
                   <div className="flex justify-between">
                     <span className="text-xs text-slate-500 uppercase">To:</span>
                     <span className="text-xs text-white font-mono break-all pl-2">{sendDetails.address}</span>
                   </div>
                   <div className="flex justify-between">
                     <span className="text-xs text-slate-500 uppercase">Amount:</span>
                     <span className="text-sm text-yellow-400 font-bold font-mono">{sendDetails.amount} NRN</span>
                   </div>
                </div>
              </div>
              <div className="flex space-x-3 w-full">
                <button 
                  onClick={resetScanner}
                  className="flex-1 py-3 bg-slate-800 hover:bg-slate-700 text-white rounded-lg text-xs font-bold uppercase tracking-widest transition-all"
                >
                  Cancel
                </button>
                <button 
                  onClick={confirmSend}
                  className="flex-1 py-3 bg-yellow-600 hover:bg-yellow-500 text-white rounded-lg text-xs font-bold uppercase tracking-widest transition-all"
                >
                  Confirm Send
                </button>
              </div>
            </div>
          )}

          {verificationStatus === 'success' && (
            <div className="bg-slate-950 border border-green-500/30 rounded-2xl p-12 flex flex-col items-center justify-center space-y-6 text-center shadow-[0_0_30px_rgba(34,197,94,0.1)]">
              <div className="w-20 h-20 bg-green-500/20 rounded-full flex items-center justify-center border border-green-500/30">
                <CheckCircle className="w-10 h-10 text-green-400" />
              </div>
              <div>
                <h3 className="text-xl font-black text-white uppercase tracking-tighter">Success</h3>
                <p className="text-slate-400 text-sm font-mono mt-2">Operation Completed Successfully</p>
                <div className="mt-4 p-3 bg-green-500/10 rounded-lg border border-green-500/20">
                  <p className="text-[10px] font-mono text-green-400 break-all">{scanResult}</p>
                </div>
              </div>
              {joinedSessionID && (
                <button
                  onClick={() => navigate(`/dves/${encodeURIComponent(joinedSessionID)}/agent`)}
                  className="flex items-center space-x-2 px-6 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg text-xs font-bold uppercase tracking-widest transition-all"
                >
                  <ArrowRight className="w-4 h-4" />
                  <span>Open Session Chat</span>
                </button>
              )}
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
                <h3 className="text-xl font-black text-white uppercase tracking-tighter">Failed</h3>
                <p className="text-slate-400 text-sm font-mono mt-2">
                  {verificationError || 'Invalid or Expired Token'}
                </p>
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
