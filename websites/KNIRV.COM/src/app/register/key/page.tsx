'use client';

import React, { useState } from 'react';
import { Key, Wallet, Shield, Download, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import KnirvLogo from '@/components/KnirvLogo';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

type KeyType = 'enterprise' | 'bootnode';
type Step = 'select' | 'connect' | 'sign' | 'generate' | 'download';

interface EthereumProvider {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
  isMetaMask?: boolean;
}

declare global {
  interface Window {
    ethereum?: EthereumProvider;
  }
}

const KEY_DESCRIPTIONS: Record<KeyType, { title: string; desc: string; tier: string; badge: string }> = {
  enterprise: {
    title: 'Enterprise Key',
    desc: 'Authorizes your KNIRVSERVER instance as an enterprise operator node. Unlocks all Professional and Enterprise tier features.',
    tier: 'Professional / Enterprise',
    badge: 'Pro+',
  },
  bootnode: {
    title: 'Bootnode Key',
    desc: 'Authorizes your KNIRVSERVER instance as a network seed node. Required for operating a bootnode in the KNIRV peer-to-peer network.',
    tier: 'Enterprise / Custom',
    badge: 'Enterprise+',
  },
};

const SIGN_MESSAGE_TEMPLATE = (wallet: string, keyType: KeyType, nonce: string) =>
  `KNIRV Network Key Registration\n\nKey Type: ${keyType}\nWallet: ${wallet}\nNonce: ${nonce}\n\nBy signing this message you authorize KNIRV Corp to issue a ${keyType}.key bound to this wallet address. The signature hash will serve as the cryptographic root password for your KNIRVSERVER instance. Never share this signature.`;

export default function KeyRegistrationPage() {
  const [step, setStep] = useState<Step>('select');
  const [keyType, setKeyType] = useState<KeyType>('enterprise');
  const [walletAddress, setWalletAddress] = useState('');
  const [signature, setSignature] = useState('');
  const [nonce] = useState(() => Math.random().toString(36).slice(2) + Date.now().toString(36));
  const [keyData, setKeyData] = useState<string | null>(null);
  const [keyFileName, setKeyFileName] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const connectWallet = async () => {
    setError('');
    if (!window.ethereum) {
      setError('No Ethereum wallet detected. Please install MetaMask or another EIP-1193 compatible wallet.');
      return;
    }
    try {
      setIsLoading(true);
      const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' }) as string[];
      if (!accounts.length) {
        setError('No accounts returned from wallet.');
        return;
      }
      setWalletAddress(accounts[0]);
      setStep('sign');
    } catch (e: unknown) {
      setError((e as Error).message || 'Wallet connection failed.');
    } finally {
      setIsLoading(false);
    }
  };

  const signMessage = async () => {
    setError('');
    try {
      setIsLoading(true);
      const message = SIGN_MESSAGE_TEMPLATE(walletAddress, keyType, nonce);
      const sig = await window.ethereum!.request({
        method: 'personal_sign',
        params: [message, walletAddress],
      }) as string;
      setSignature(sig);
      setStep('generate');
      await generateKey(sig);
    } catch (e: unknown) {
      setError((e as Error).message || 'Signing failed.');
    } finally {
      setIsLoading(false);
    }
  };

  const generateKey = async (sig: string) => {
    setError('');
    try {
      setIsLoading(true);
      const res = await fetch('/api/register-key', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          keyType,
          walletAddress,
          signature: sig,
          nonce,
          message: SIGN_MESSAGE_TEMPLATE(walletAddress, keyType, nonce),
        }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: 'Server error' }));
        throw new Error(body.error || `HTTP ${res.status}`);
      }

      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const fileName = `${keyType}.key`;
      setKeyData(url);
      setKeyFileName(fileName);
      setStep('download');
    } catch (e: unknown) {
      setError((e as Error).message || 'Key generation failed.');
      setStep('sign');
    } finally {
      setIsLoading(false);
    }
  };

  const downloadKey = () => {
    if (!keyData) return;
    const a = document.createElement('a');
    a.href = keyData;
    a.download = keyFileName;
    a.click();
  };

  return (
    <div className="dve-page min-h-screen">
      <nav className="dve-nav">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <KnirvLogo />
          </div>
        </div>
      </nav>

      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-2xl mx-auto">
          <div className="text-center mb-12">
            <Key className="h-12 w-12 mx-auto mb-4 text-purple-400" />
            <h1 className="text-4xl font-bold knirv-gradient-text mb-3">Node Key Registration</h1>
            <p className="text-white/60 text-lg">
              Generate your wallet-bound node identity key. Your wallet signature becomes the cryptographic root password for your KNIRVSERVER instance.
            </p>
          </div>

          {/* Progress steps */}
          <div className="flex items-center justify-center gap-2 mb-10">
            {(['select', 'connect', 'sign', 'generate', 'download'] as Step[]).map((s, i) => (
              <React.Fragment key={s}>
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold
                  ${step === s ? 'bg-purple-500 text-white' :
                    ['select','connect','sign','generate','download'].indexOf(step) > i
                      ? 'bg-green-500 text-white'
                      : 'bg-white/10 text-white/40'}`}>
                  {['select','connect','sign','generate','download'].indexOf(step) > i
                    ? <CheckCircle className="h-4 w-4" /> : i + 1}
                </div>
                {i < 4 && <div className="w-8 h-px bg-white/20" />}
              </React.Fragment>
            ))}
          </div>

          {/* Error banner */}
          {error && (
            <Card className="mb-6 bg-red-500/10 border-red-500/30">
              <CardContent className="p-4 flex gap-3 items-start">
                <AlertCircle className="h-5 w-5 text-red-400 flex-shrink-0 mt-0.5" />
                <p className="text-red-300 text-sm">{error}</p>
              </CardContent>
            </Card>
          )}

          {/* Step: Select key type */}
          {step === 'select' && (
            <Card className="bg-white/5 border-white/10">
              <CardHeader>
                <CardTitle className="text-white">Choose Key Type</CardTitle>
                <CardDescription className="text-white/60">
                  Select the node identity key that matches your subscription tier.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {(Object.entries(KEY_DESCRIPTIONS) as [KeyType, typeof KEY_DESCRIPTIONS['enterprise']][]).map(([type, meta]) => (
                  <button
                    key={type}
                    onClick={() => setKeyType(type)}
                    className={`w-full text-left p-4 rounded-lg border transition-all ${
                      keyType === type
                        ? 'border-purple-500/60 bg-purple-500/10'
                        : 'border-white/10 bg-white/5 hover:bg-white/10'
                    }`}
                  >
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="font-semibold text-white flex items-center gap-2">
                          <Shield className="h-4 w-4 text-purple-400" />
                          {meta.title}
                        </div>
                        <p className="text-white/60 text-sm mt-1">{meta.desc}</p>
                        <p className="text-white/40 text-xs mt-1">Required tier: {meta.tier}</p>
                      </div>
                      <Badge className="bg-purple-500/20 text-purple-300 border-purple-500/30 ml-3 flex-shrink-0">
                        {meta.badge}
                      </Badge>
                    </div>
                  </button>
                ))}

                <Button
                  className="w-full mt-4 bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 text-white"
                  onClick={() => setStep('connect')}
                >
                  Continue with {KEY_DESCRIPTIONS[keyType].title}
                </Button>
              </CardContent>
            </Card>
          )}

          {/* Step: Connect wallet */}
          {step === 'connect' && (
            <Card className="bg-white/5 border-white/10">
              <CardHeader>
                <CardTitle className="text-white flex items-center gap-2">
                  <Wallet className="h-5 w-5 text-purple-400" />
                  Connect Your Wallet
                </CardTitle>
                <CardDescription className="text-white/60">
                  Connect the wallet that will be cryptographically bound to your{' '}
                  <strong className="text-white/80">{KEY_DESCRIPTIONS[keyType].title}</strong>.
                  This address will be your node&apos;s permanent on-chain identity.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4">
                  <p className="text-yellow-300 text-sm">
                    ⚠ Use the wallet address you intend to operate your KNIRVSERVER node with.
                    The key cannot be transferred to a different wallet after generation.
                  </p>
                </div>
                <Button
                  className="w-full bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 text-white"
                  onClick={connectWallet}
                  disabled={isLoading}
                >
                  {isLoading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : <Wallet className="h-4 w-4 mr-2" />}
                  Connect Wallet
                </Button>
                <p className="text-white/40 text-xs text-center">
                  Compatible with MetaMask, WalletConnect, Coinbase Wallet, and any EIP-1193 provider.
                </p>
              </CardContent>
            </Card>
          )}

          {/* Step: Sign message */}
          {step === 'sign' && (
            <Card className="bg-white/5 border-white/10">
              <CardHeader>
                <CardTitle className="text-white">Sign to Authorize Key</CardTitle>
                <CardDescription className="text-white/60">
                  Sign the authorization message with wallet <code className="text-purple-300 text-xs bg-purple-500/10 px-1 rounded">{walletAddress.slice(0, 6)}...{walletAddress.slice(-4)}</code>.
                  This signature becomes the root password for your KNIRVSERVER instance — store it securely.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="bg-white/5 border border-white/10 rounded-lg p-4 font-mono text-xs text-white/60 whitespace-pre-wrap break-all">
                  {SIGN_MESSAGE_TEMPLATE(walletAddress, keyType, nonce)}
                </div>
                <Button
                  className="w-full bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 text-white"
                  onClick={signMessage}
                  disabled={isLoading}
                >
                  {isLoading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : <Key className="h-4 w-4 mr-2" />}
                  Sign & Generate Key
                </Button>
              </CardContent>
            </Card>
          )}

          {/* Step: Generating */}
          {step === 'generate' && (
            <Card className="bg-white/5 border-white/10">
              <CardContent className="py-16 flex flex-col items-center gap-4">
                <Loader2 className="h-12 w-12 text-purple-400 animate-spin" />
                <p className="text-white/70">Generating your {KEY_DESCRIPTIONS[keyType].title}...</p>
                <p className="text-white/40 text-sm">Verifying subscription tier and compiling key parameters.</p>
              </CardContent>
            </Card>
          )}

          {/* Step: Download */}
          {step === 'download' && (
            <Card className="bg-white/5 border-white/10">
              <CardHeader>
                <CardTitle className="text-white flex items-center gap-2">
                  <CheckCircle className="h-5 w-5 text-green-400" />
                  Key Ready
                </CardTitle>
                <CardDescription className="text-white/60">
                  Your <strong className="text-white/80">{keyFileName}</strong> has been generated and is ready for download.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="bg-green-500/10 border border-green-500/30 rounded-lg p-4 space-y-2">
                  <p className="text-green-300 text-sm font-medium">Installation instructions:</p>
                  <ol className="text-white/60 text-sm space-y-1 list-decimal list-inside">
                    <li>Download the key file below.</li>
                    <li>When the KNIRVSERVER installer prompts for a key file path, provide the path to this file.</li>
                    <li>The installer will place it in <code className="text-purple-300 text-xs bg-purple-500/10 px-1 rounded">/var/lib/knirvserver/{keyFileName}</code>.</li>
                    <li>Keep your wallet signature secure — it serves as the root password for your instance.</li>
                  </ol>
                </div>

                <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4">
                  <p className="text-yellow-300 text-sm">
                    ⚠ This key will not be stored by KNIRV Corp. Download and back it up immediately.
                    A lost key requires re-registration.
                  </p>
                </div>

                <Button
                  className="w-full bg-gradient-to-r from-green-600 to-emerald-600 hover:from-green-500 hover:to-emerald-500 text-white"
                  onClick={downloadKey}
                >
                  <Download className="h-4 w-4 mr-2" />
                  Download {keyFileName}
                </Button>

                <div className="text-center">
                  <button
                    className="text-white/40 text-xs hover:text-white/70 transition-colors"
                    onClick={() => { setStep('select'); setKeyData(null); setSignature(''); setWalletAddress(''); }}
                  >
                    Register another key
                  </button>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </section>
    </div>
  );
}
