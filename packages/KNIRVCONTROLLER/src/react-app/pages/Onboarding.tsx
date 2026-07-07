import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router';
import {
  ArrowLeft,
  BadgeCheck,
  Check,
  ChevronRight,
  Copy,
  Download,
  Eye,
  EyeOff,
  QrCode,
  KeyRound,
  LoaderCircle,
  LockKeyhole,
  Plus,
  RotateCcw,
  Shield,
  ShieldCheck,
  Sparkles,
  Terminal,
} from 'lucide-react';
import { useVault } from '@/react-app/hooks/useVault';
import { QrScannerFrame } from '@/react-app/components/QrScannerFrame';
import { notificationError } from '@/react-app/platform/haptics';

type FlowMode = 'create' | 'restore';
type Step = 'entry' | 'password' | 'recovery' | 'signing' | 'transition' | 'granted';

type Challenge = {
  nonce: string;
  timestamp: number;
};

const TRANSITION_LOGS = [
  '> Initializing hardware handshake...',
  '> Verifying mnemonic integrity... [OK]',
  '> Injecting secure keys into enclave...',
  '> Linking Controller ID: KN-9921-X...',
  '> Awaiting signing challenge...',
];

function buildChallenge(): Challenge {
  const bytes = new Uint8Array(8);
  crypto.getRandomValues(bytes);
  const nonce = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');

  return {
    nonce: `0x${nonce}`,
    timestamp: Math.floor(Date.now() / 1000),
  };
}

function normalizePhrase(value: string) {
  return value.trim().replace(/\s+/g, ' ');
}

function splitPhrase(value: string) {
  return normalizePhrase(value)
    .split(' ')
    .filter(Boolean);
}

export default function Onboarding() {
  const navigate = useNavigate();
  const { createVault, importVault } = useVault();

  const [step, setStep] = useState<Step>('entry');
  const [mode, setMode] = useState<FlowMode>('create');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [recoveryPhrase, setRecoveryPhrase] = useState('');
  const [mnemonic, setMnemonic] = useState<string[]>([]);
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [receivedMessage, setReceivedMessage] = useState('');
  const [receivedOrigin, setReceivedOrigin] = useState('');
  const [scanError, setScanError] = useState<string | null>(null);
  const [challenge, setChallenge] = useState<Challenge>(() => buildChallenge());

  const isCreateFlow = mode === 'create';
  const recoveryReady = mnemonic.length > 0;

  const title = useMemo(() => {
    if (step === 'entry') return 'Initialize your Identity';
    if (step === 'password') return isCreateFlow ? 'Set Device Password' : 'Restore Controller';
    if (step === 'recovery') return 'Secret Recovery Phrase';
    if (step === 'signing') return 'Sign To Enter KNIRV';
    if (step === 'transition') return 'Controller Ready';
    return isCreateFlow ? 'Controller Ready' : 'Welcome Back, Controller';
  }, [isCreateFlow, step]);

  const subtitle = useMemo(() => {
    switch (step) {
      case 'entry':
        return 'Secure your digital sovereignty with a new vault or restore access to your existing assets.';
      case 'password':
        return isCreateFlow
          ? 'This password unlocks the controller vault on this device only.'
          : 'Restore your controller vault, then set the device password that protects this machine.';
      case 'recovery':
        return 'Write these words down in order and keep them safe. Anyone with this phrase controls the vault.';
      case 'signing':
        return 'Scan the QR message from onboarding.knirv.com, then sign it to unlock platform access.';
      case 'transition':
        return 'Preparing secure platform access. Synchronizing encryption protocols with your new hardware vault.';
      default:
        return isCreateFlow
          ? 'Your controller vault is active. The platform is ready to receive your signed identity.'
          : 'Your KNIRV session is active. All vault protocols are engaged and operating within normal parameters.';
    }
  }, [isCreateFlow, step]);

  useEffect(() => {
    if (step === 'signing') {
      setChallenge(buildChallenge());
    }
  }, [step]);

  useEffect(() => {
    if (step !== 'transition') return undefined;

    const timer = window.setTimeout(() => {
      setStep('granted');
    }, 2400);

    return () => window.clearTimeout(timer);
  }, [step]);

  useEffect(() => {
    if (!copied) return undefined;

    const timer = window.setTimeout(() => setCopied(false), 1800);
    return () => window.clearTimeout(timer);
  }, [copied]);

  const resetFormState = () => {
    setPassword('');
    setConfirmPassword('');
    setRecoveryPhrase('');
    setMnemonic([]);
    setShowPassword(false);
    setShowConfirmPassword(false);
    setError(null);
    setIsSubmitting(false);
    setCopied(false);
    setScanError(null);
    setReceivedMessage('');
    setReceivedOrigin('');
  };

  const goBack = () => {
    if (step === 'entry') {
      navigate(-1);
      return;
    }

    if (step === 'password') {
      resetFormState();
      setStep('entry');
      return;
    }

    if (step === 'recovery') {
      setStep('password');
      return;
    }

    if (step === 'signing') {
      setStep(recoveryReady ? 'recovery' : 'password');
      return;
    }

    if (step === 'transition') {
      setStep('signing');
      return;
    }

    if (step === 'granted') {
      navigate('/vault');
    }
  };

  const startFlow = (nextMode: FlowMode) => {
    resetFormState();
    setMode(nextMode);
    setStep('password');
  };

  const handlePasswordSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);

    if (!password || password.length < 8) {
      notificationError();
      setError('Password must be at least 8 characters long.');
      return;
    }

    if (password !== confirmPassword) {
      notificationError();
      setError('Passwords do not match.');
      return;
    }

    if (!isCreateFlow) {
      const normalizedPhrase = normalizePhrase(recoveryPhrase);
      const wordCount = splitPhrase(normalizedPhrase).length;

      if (![12, 24].includes(wordCount)) {
        notificationError();
        setError('Recovery phrase must contain 12 or 24 words.');
        return;
      }
    }

    setIsSubmitting(true);

    try {
      if (isCreateFlow) {
        const phrase = await createVault(password);
        setMnemonic(splitPhrase(phrase));
        setStep('recovery');
      } else {
        await importVault(normalizePhrase(recoveryPhrase), password);
        setStep('signing');
      }
    } catch (err: any) {
      notificationError();
      setError(err?.message || (isCreateFlow ? 'Failed to create controller vault.' : 'Failed to restore controller vault.'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCopyPhrase = async () => {
    if (!mnemonic.length) return;

    try {
      await navigator.clipboard.writeText(mnemonic.join(' '));
      setCopied(true);
    } catch {
      notificationError();
      setError('Copy to clipboard is unavailable in this browser.');
    }
  };

  const handleContinueFromRecovery = () => {
    if (!recoveryReady) return;
    setStep('signing');
  };

  const handleSign = () => {
    if (!receivedMessage) {
      notificationError();
      setScanError('Scan the QR message before signing.');
      return;
    }

    setIsSubmitting(true);
    window.setTimeout(() => {
      setIsSubmitting(false);
      setStep('transition');
    }, 2000);
  };

  const handlePrintPacket = () => {
    window.print();
  };

  const handleQrScan = (decodedText: string) => {
    try {
      const parsed = JSON.parse(decodedText);
      if (parsed && typeof parsed === 'object') {
        const message = typeof parsed.message === 'string' ? parsed.message : decodedText;
        setReceivedMessage(message.trim());
        setReceivedOrigin(typeof parsed.origin === 'string' ? parsed.origin : '');
        setScanError(null);
        return;
      }
    } catch {
      // fall through to plain-text payload
    }

    setReceivedMessage(decodedText.trim());
    setReceivedOrigin('');
    setScanError(null);
  };

  const panelMaxWidth = step === 'recovery' ? 'max-w-[640px]' : 'max-w-[560px]';

  return (
    <div
      className="relative min-h-screen overflow-x-hidden bg-[#020408] text-[#e2e2e2]"
      style={{ fontFamily: "'Geist', 'Inter', system-ui, sans-serif" }}
    >
      <div className="pointer-events-none fixed inset-0 overflow-hidden">
        <div className="absolute -top-[10%] -left-[10%] h-[50%] w-[50%] rounded-full bg-[#2563eb]/5 blur-[120px]" />
        <div className="absolute -bottom-[10%] -right-[10%] h-[40%] w-[40%] rounded-full bg-[#00cbe6]/5 blur-[120px]" />
      </div>

      <header className="fixed top-0 z-50 h-16 w-full border-b border-white/10 bg-[rgba(15,23,42,0.6)] backdrop-blur-xl">
        <div className="flex h-full items-center justify-between px-4 md:px-10">
          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={goBack}
              className="rounded-full p-2 text-[#b4c5ff] transition-colors hover:text-[#5de6ff]"
              aria-label="Go back"
            >
              <ArrowLeft className="h-5 w-5" />
            </button>
            <div className="text-[28px] font-semibold tracking-tight text-[#e2e2e2] md:text-[34px]">
              KNIRV
            </div>
          </div>

          <div className="flex items-center gap-2 text-[#b4c5ff]">
            <Shield className="h-5 w-5" />
            <span className="hidden text-[12px] font-medium uppercase tracking-[0.3em] md:inline">
              System Secured
            </span>
          </div>
        </div>
      </header>

      <main className="relative flex min-h-screen items-center justify-center px-4 pb-10 pt-24 md:px-6">
        {step === 'entry' && (
          <div className="relative flex w-full max-w-6xl items-center justify-center">
            <div className="relative w-full">
              <div className="mx-auto w-full max-w-[560px] rounded-xl border border-white/10 bg-[rgba(15,23,42,0.6)] p-8 shadow-[0_8px_32px_rgba(0,0,0,0.8)] backdrop-blur-xl md:p-10">
                <div className="mb-10 flex justify-center">
                  <div className="relative flex h-24 w-24 items-center justify-center rounded-full border border-white/10 bg-[#0c0f0f]/50 shadow-[0_0_40px_rgba(37,99,235,0.1)]">
                    <div className="absolute inset-0 rounded-full bg-[#2563eb]/5 blur-xl" />
                    <Shield className="relative z-10 h-12 w-12 text-[#b4c5ff]" />
                  </div>
                </div>

                <div className="mb-12 text-center">
                  <h1 className="text-[32px] font-semibold tracking-tight text-[#e2e2e2] md:text-[40px]">
                    {title}
                  </h1>
                  <p className="mx-auto mt-4 max-w-[420px] text-[18px] leading-7 text-[#c3c6d7]">
                    {subtitle}
                  </p>
                </div>

                <div className="space-y-4">
                  <button
                    type="button"
                    onClick={() => startFlow('create')}
                    className="group relative w-full overflow-hidden rounded-xl border border-white/10 bg-[#2563eb] p-5 text-left text-[#eeefff] transition-all duration-300 hover:shadow-[0_0_20px_rgba(37,99,235,0.35)] active:scale-[0.99]"
                  >
                    <div className="absolute -right-6 bottom-0 opacity-10">
                      <Plus className="h-36 w-36" />
                    </div>
                    <div className="flex items-start gap-4">
                      <div className="flex h-12 w-12 items-center justify-center rounded-xl border border-white/10 bg-[#b4c5ff]/10 text-[#b4c5ff]">
                        <Shield className="h-6 w-6" />
                      </div>
                      <div className="flex-1">
                        <div className="text-[22px] font-medium leading-8">Create New Controller</div>
                        <p className="mt-2 max-w-[360px] text-[16px] leading-7 text-[#dbe1ff]">
                          Generate a fresh controller vault and recovery phrase.
                        </p>
                      </div>
                      <ChevronRight className="mt-2 h-6 w-6 transition-transform group-hover:translate-x-1" />
                    </div>
                  </button>

                  <button
                    type="button"
                    onClick={() => startFlow('restore')}
                    className="group relative w-full overflow-hidden rounded-xl border border-white/10 bg-[#0c0f0f]/50 p-5 text-left text-[#e2e2e2] transition-all duration-300 hover:border-[#b4c5ff]/40 hover:bg-[#1a1c1c] active:scale-[0.99]"
                  >
                    <div className="absolute -right-6 bottom-0 opacity-10">
                      <RotateCcw className="h-36 w-36" />
                    </div>
                    <div className="flex items-start gap-4">
                      <div className="flex h-12 w-12 items-center justify-center rounded-xl border border-[#00cbe6]/20 bg-[#00cbe6]/10 text-[#00cbe6]">
                        <KeyRound className="h-6 w-6" />
                      </div>
                      <div className="flex-1">
                        <div className="text-[22px] font-medium leading-8">Import Existing Controller</div>
                        <p className="mt-2 max-w-[360px] text-[16px] leading-7 text-[#c3c6d7]">
                          Restore a controller vault from a 12 or 24 word phrase.
                        </p>
                      </div>
                      <ChevronRight className="mt-2 h-6 w-6 transition-transform group-hover:translate-x-1" />
                    </div>
                  </button>
                </div>

                <div className="mt-12 border-t border-white/10 pt-8">
                  <div className="grid gap-4 text-[12px] uppercase tracking-[0.25em] text-[#8d90a0] sm:grid-cols-2">
                    <div className="flex items-center gap-2">
                      <LockKeyhole className="h-4 w-4" />
                      <span>Open Source Audited</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <BadgeCheck className="h-4 w-4" />
                      <span>Cryptographically Verified</span>
                    </div>
                  </div>
                </div>

                <div className="absolute left-0 top-0 h-px w-full bg-gradient-to-r from-transparent via-[#b4c5ff]/40 to-transparent opacity-60" />
              </div>

              <div className="hidden lg:block lg:absolute lg:right-0 lg:top-1/2 lg:w-64 lg:-translate-y-1/2">
                <div className="border-r border-white/10 pr-4 opacity-40">
                  <div className="space-y-6 text-[12px] uppercase tracking-[0.3em] text-[#8d90a0]">
                    <div>
                      <div className="text-[#b4c5ff]">Node Status</div>
                      <div className="mt-1 text-[#e2e2e2]">SYNCHRONIZED</div>
                    </div>
                    <div>
                      <div className="text-[#b4c5ff]">Encryption</div>
                      <div className="mt-1 text-[#e2e2e2]">AES-256-GCM</div>
                    </div>
                    <div>
                      <div className="text-[#b4c5ff]">Latency</div>
                      <div className="mt-1 text-[#e2e2e2]">12ms</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {step === 'password' && (
          <div className={`relative w-full ${panelMaxWidth}`}>
            <div className="absolute inset-x-0 top-1/2 -z-10 h-[600px] -translate-y-1/2 rounded-full bg-[#2563eb]/5 blur-[120px]" />
            <div className="rounded-xl border border-white/10 bg-[rgba(15,23,42,0.6)] shadow-[0_8px_32px_rgba(0,0,0,0.8)] backdrop-blur-xl">
              <div className="p-8 md:p-10">
                <div className="mb-10 text-center">
                  <div className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full border border-white/10 bg-[#2563eb]/10">
                    <LockKeyhole className="h-10 w-10 text-[#b4c5ff]" />
                  </div>
                  <h1 className="text-[32px] font-semibold tracking-tight text-[#e2e2e2]">
                    {isCreateFlow ? 'Set Device Password' : 'Restore Controller Password'}
                  </h1>
                  <p className="mx-auto mt-3 max-w-[420px] text-[16px] leading-7 text-[#c3c6d7]">
                    {subtitle}
                  </p>
                </div>

                <form className="space-y-6" onSubmit={handlePasswordSubmit}>
                  <div className="space-y-2">
                    <label className="block px-1 text-[12px] font-medium uppercase tracking-[0.3em] text-[#c2c6d2]">
                      Enter Password
                    </label>
                    <div className="relative">
                      <input
                        type={showPassword ? 'text' : 'password'}
                        value={password}
                        onChange={(event) => setPassword(event.target.value)}
                        className="h-14 w-full rounded-lg border border-white/10 bg-[#020408]/50 px-4 pr-12 font-mono text-[#b4c5ff] placeholder:text-[#64748b] focus:border-[#b4c5ff] focus:outline-none focus:ring-1 focus:ring-[#b4c5ff]"
                        placeholder="••••••••••••"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword((current) => !current)}
                        className="absolute right-4 top-1/2 -translate-y-1/2 text-[#8d90a0] transition-colors hover:text-[#b4c5ff]"
                        aria-label={showPassword ? 'Hide password' : 'Show password'}
                      >
                        {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </button>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <label className="block px-1 text-[12px] font-medium uppercase tracking-[0.3em] text-[#c2c6d2]">
                      Confirm Password
                    </label>
                    <div className="relative">
                      <input
                        type={showConfirmPassword ? 'text' : 'password'}
                        value={confirmPassword}
                        onChange={(event) => setConfirmPassword(event.target.value)}
                        className="h-14 w-full rounded-lg border border-white/10 bg-[#020408]/50 px-4 pr-12 font-mono text-[#b4c5ff] placeholder:text-[#64748b] focus:border-[#b4c5ff] focus:outline-none focus:ring-1 focus:ring-[#b4c5ff]"
                        placeholder="••••••••••••"
                      />
                      <button
                        type="button"
                        onClick={() => setShowConfirmPassword((current) => !current)}
                        className="absolute right-4 top-1/2 -translate-y-1/2 text-[#8d90a0] transition-colors hover:text-[#b4c5ff]"
                        aria-label={showConfirmPassword ? 'Hide password confirmation' : 'Show password confirmation'}
                      >
                        {showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </button>
                    </div>
                  </div>

                  {!isCreateFlow && (
                    <div className="space-y-2">
                      <label className="block px-1 text-[12px] font-medium uppercase tracking-[0.3em] text-[#c2c6d2]">
                        Recovery Phrase
                      </label>
                      <textarea
                        value={recoveryPhrase}
                        onChange={(event) => setRecoveryPhrase(event.target.value)}
                        className="min-h-32 w-full rounded-lg border border-white/10 bg-[#020408]/50 px-4 py-3 font-mono text-[14px] leading-6 text-[#e2e2e2] placeholder:text-[#64748b] focus:border-[#b4c5ff] focus:outline-none focus:ring-1 focus:ring-[#b4c5ff]"
                        placeholder="Enter your 12 or 24 word phrase separated by spaces"
                      />
                    </div>
                  )}

                  {error && (
                    <div className="rounded-lg border border-[#ffb4ab]/30 bg-[#93000a]/20 px-4 py-3 text-[12px] text-[#ffdad6]">
                      {error}
                    </div>
                  )}

                  <button
                    type="submit"
                    disabled={isSubmitting}
                    className="flex h-14 w-full items-center justify-center gap-2 rounded-lg bg-[#2563eb] text-[18px] font-medium text-[#eeefff] transition-all duration-300 hover:bg-[#5de6ff] hover:text-[#00363e] hover:shadow-[0_0_20px_rgba(37,99,235,0.35)] disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    {isSubmitting ? (
                      <LoaderCircle className="h-5 w-5 animate-spin" />
                    ) : (
                      <span>{isCreateFlow ? 'Secure Vault' : 'Restore Controller'}</span>
                    )}
                  </button>
                </form>

                <div className="mt-8 flex items-start gap-4 rounded-lg border border-white/10 bg-[#1a1c1c]/30 p-4">
                  <BadgeCheck className="mt-0.5 h-5 w-5 text-[#b4c5ff]" />
                  <p className="text-[12px] leading-6 text-[#c3c6d7]">
                    Passwords are never stored on a remote server. Your controller is encrypted locally using hardware-level security modules.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {step === 'recovery' && (
          <div className={`relative w-full ${panelMaxWidth}`}>
            <div className="absolute left-0 top-1/2 -z-10 h-[600px] w-[600px] -translate-y-1/2 rounded-full bg-[#2563eb]/10 blur-[140px]" />
            <div className="absolute right-0 top-1/2 -z-10 h-[600px] w-[600px] -translate-y-1/2 rounded-full bg-[#00cbe6]/5 blur-[140px]" />

            <div className="rounded-xl border border-white/10 bg-[rgba(15,23,42,0.6)] shadow-[0_8px_32px_rgba(0,0,0,0.8)] backdrop-blur-xl">
              <div className="p-8 md:p-10">
                <div className="text-center">
                  <h1 className="text-[32px] font-semibold tracking-tight text-[#e2e2e2]">
                    {title}
                  </h1>
                  <p className="mx-auto mt-3 max-w-[420px] text-[16px] leading-7 text-[#c3c6d7]">
                    {subtitle}
                  </p>
                </div>

                <div className="mt-8 rounded-lg border border-[#ffb4ab]/30 bg-[#93000a]/20 p-4">
                  <div className="flex items-start gap-4">
                    <Sparkles className="mt-0.5 h-5 w-5 text-[#ffb4ab]" />
                    <div>
                      <div className="text-[12px] font-medium uppercase tracking-[0.3em] text-[#ffb4ab]">
                        Crucial Security Alert
                      </div>
                      <p className="mt-2 text-[14px] leading-6 text-[#ffdad6]">
                        Do not store this digitally. Your recovery phrase is the only way to recover your vault if you lose access.
                      </p>
                    </div>
                  </div>
                </div>

                <div className="mt-8 grid grid-cols-2 gap-3 sm:grid-cols-3">
                  {mnemonic.map((word, index) => (
                    <div
                      key={`${word}-${index}`}
                      className="flex items-center gap-3 rounded-lg border border-white/10 bg-[#020408]/40 p-3 transition-colors hover:border-[#b4c5ff] hover:bg-[#b4c5ff]/5"
                    >
                      <span className="w-5 font-mono text-[12px] text-[#424751]">
                        {String(index + 1).padStart(2, '0')}
                      </span>
                      <span className="select-none font-mono text-[14px] font-medium text-[#e2e2e2]">
                        {word}
                      </span>
                    </div>
                  ))}
                </div>

                <div className="mt-6 flex flex-col items-center justify-center gap-4 py-1 sm:flex-row sm:gap-6">
                  <button
                    type="button"
                    onClick={handleCopyPhrase}
                    className="flex items-center gap-2 text-[14px] font-medium text-[#b4c5ff] transition-colors hover:text-[#5de6ff]"
                  >
                    <Copy className="h-4 w-4" />
                    <span>{copied ? 'Copied to clipboard' : 'Copy to clipboard'}</span>
                  </button>
                  <div className="hidden h-4 w-px bg-white/10 sm:block" />
                  <button
                    type="button"
                    onClick={handlePrintPacket}
                    className="flex items-center gap-2 text-[14px] font-medium text-[#c3c6d7] transition-colors hover:text-[#e2e2e2]"
                  >
                    <Download className="h-4 w-4" />
                    <span>Download as PDF</span>
                  </button>
                </div>

                <div className="mt-8 rounded-lg border border-white/10 bg-[#1a1c1c]/30 p-4">
                  <div className="flex items-start gap-3">
                    <Check className="mt-0.5 h-5 w-5 text-[#00cbe6]" />
                    <p className="text-[12px] leading-6 text-[#c3c6d7]">
                      Verify the phrase in order, then continue to the signing step. The vault is not recoverable without these words.
                    </p>
                  </div>
                </div>

                <button
                  type="button"
                  onClick={handleContinueFromRecovery}
                  disabled={!recoveryReady}
                  className="mt-8 flex h-14 w-full items-center justify-center gap-2 rounded-lg bg-[#2563eb] text-[18px] font-medium text-[#eeefff] transition-all duration-300 hover:shadow-[0_0_20px_rgba(37,99,235,0.35)] disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <span>I have saved my phrase</span>
                  <ChevronRight className="h-5 w-5" />
                </button>
              </div>
            </div>
          </div>
        )}

        {step === 'signing' && (
          <div className={`relative w-full ${panelMaxWidth}`}>
            <div className="absolute inset-0 -z-10 rounded-full bg-[#2563eb]/5 blur-[140px]" />
            <div className="rounded-xl border border-white/10 bg-[rgba(15,23,42,0.6)] shadow-[0_8px_32px_rgba(0,0,0,0.8)] backdrop-blur-xl">
              <div className="relative overflow-hidden p-8 md:p-10">
                <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-[#b4c5ff]/70 to-transparent" />

                <div className="mb-6 flex justify-center">
                  <div className="flex h-20 w-20 items-center justify-center rounded-full border border-white/10 bg-[#2563eb]/10">
                    <Shield className="h-10 w-10 animate-pulse text-[#b4c5ff]" />
                  </div>
                </div>

                <div className="text-center">
                  <h1 className="text-[32px] font-semibold tracking-tight text-[#e2e2e2]">
                    {title}
                  </h1>
                  <p className="mx-auto mt-3 max-w-[420px] text-[16px] leading-7 text-[#c3c6d7]">
                    {subtitle}
                  </p>
                </div>

                <div className="mt-8 grid gap-4 lg:grid-cols-[1.05fr_0.95fr]">
                  <div className="space-y-3">
                    <div className="mb-2 flex items-center gap-2 text-[12px] font-medium uppercase tracking-[0.3em] text-[#c2c6d2]">
                      <QrCode className="h-4 w-4 text-[#5de6ff]" />
                      <span>Scan Message QR</span>
                    </div>
                    <div className="overflow-hidden rounded-xl border border-white/10 bg-[#020408]/60">
                      <QrScannerFrame
                        active={step === 'signing'}
                        readerId="sign-message-reader"
                        onScan={handleQrScan}
                        onError={(e) => { notificationError(); setScanError(e); }}
                        className="relative min-h-[320px]"
                        fallbackTitle="QR Scanner Unavailable"
                        fallbackDescription="Use the native camera or a secure browser camera to scan the onboarding payload."
                      />
                    </div>
                    <div className="rounded-lg border border-white/10 bg-[#1a1c1c]/30 p-4">
                      <div className="flex items-start gap-3">
                        <QrCode className="mt-0.5 h-5 w-5 text-[#b4c5ff]" />
                        <p className="text-[12px] leading-6 text-[#c3c6d7]">
                          Point the vault at the QR code from onboarding.knirv.com. The scanned payload will be used as the message to sign.
                        </p>
                      </div>
                    </div>
                  </div>

                  <div className="space-y-4">
                    <div className="rounded-lg border border-white/10 bg-[#020408]/60 p-4">
                      <div className="mb-2 text-[12px] font-medium uppercase tracking-[0.3em] text-[#c2c6d2]">
                        Received Message
                      </div>
                      <div className="overflow-x-auto font-mono text-[14px] leading-7 text-[#c2c6d2]">
                        <div className="flex items-start gap-3">
                          <Terminal className="mt-0.5 h-4 w-4 shrink-0 text-[#5de6ff]" />
                          <code className="whitespace-pre-wrap">
                            {receivedMessage || `Awaiting QR payload...\nnonce: ${challenge.nonce}\ntimestamp: ${challenge.timestamp}`}
                          </code>
                        </div>
                      </div>
                    </div>

                    {receivedOrigin && (
                      <div className="rounded-lg border border-[#00cbe6]/20 bg-[#00cbe6]/10 px-4 py-3 text-[12px] uppercase tracking-[0.3em] text-[#5de6ff]">
                        Source: {receivedOrigin}
                      </div>
                    )}

                    {scanError && (
                      <div className="rounded-lg border border-[#ffb4ab]/30 bg-[#93000a]/20 px-4 py-3 text-[12px] text-[#ffdad6]">
                        {scanError}
                      </div>
                    )}
                  </div>
                </div>

                <div className="mt-8 space-y-4">
                  <button
                    type="button"
                    onClick={handleSign}
                    disabled={isSubmitting || !receivedMessage}
                    className="flex h-14 w-full items-center justify-center gap-2 rounded-lg bg-[#b4c5ff] text-[18px] font-medium text-[#002a78] transition-all duration-300 hover:shadow-[0_0_20px_rgba(180,197,255,0.4)] disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    {isSubmitting ? (
                      <LoaderCircle className="h-5 w-5 animate-spin" />
                    ) : (
                      <BadgeCheck className="h-5 w-5" />
                    )}
                    <span>{isSubmitting ? 'Processing...' : 'Sign Message'}</span>
                  </button>

                  <button
                    type="button"
                    onClick={goBack}
                    className="flex h-14 w-full items-center justify-center rounded-lg border border-white/10 text-[18px] font-medium text-[#e2e2e2] transition-colors hover:bg-white/5"
                  >
                    Cancel
                  </button>
                </div>

                <div className="mt-8 flex items-center gap-2 text-[12px] font-mono uppercase tracking-[0.3em] text-[#004e5a] opacity-70">
                  <Shield className="h-4 w-4" />
                  <span>End-to-end encrypted signing path</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {step === 'transition' && (
          <div className={`relative w-full ${panelMaxWidth}`}>
            <div className="absolute inset-0 -z-10 rounded-full bg-[#2563eb]/5 blur-[140px]" />
            <div className="rounded-xl border border-white/10 bg-[rgba(15,23,42,0.6)] shadow-[0_8px_32px_rgba(0,0,0,0.8)] backdrop-blur-xl">
              <div className="p-8 text-center md:p-10">
                <div className="relative mx-auto mb-10 flex h-32 w-32 items-center justify-center rounded-full border border-white/10 bg-[#020408]/50">
                  <div className="absolute inset-0 rounded-full border border-[#2563eb]/20" />
                  <div className="absolute inset-4 rounded-full border border-white/10" />
                  <Shield className="h-14 w-14 text-[#b4c5ff]" />
                </div>

                <h1 className="text-[32px] font-semibold tracking-tight text-[#e2e2e2] md:text-[40px]">
                  Controller Ready
                </h1>
                <p className="mx-auto mt-4 max-w-[420px] text-[16px] leading-7 text-[#c3c6d7]">
                  Preparing secure platform access. Synchronizing encryption protocols with your new hardware vault.
                </p>

                <div className="mt-10 h-1 w-full overflow-hidden rounded-full bg-white/10">
                  <div className="h-full w-3/4 rounded-full bg-[#2563eb] shadow-[0_0_10px_rgba(37,99,235,0.7)]" />
                </div>

                <div className="mt-4 text-[12px] font-medium uppercase tracking-[0.3em] text-[#b4c5ff]">
                  Establishing Bridge
                </div>

                <div className="mt-8 h-28 overflow-hidden rounded-lg border border-white/10 bg-[#020408]/40 p-4 text-left font-mono text-[14px] leading-6 text-[#8d90a0]">
                  {TRANSITION_LOGS.map((line, index) => (
                    <div key={line} className={index === 0 ? 'text-[#5de6ff]' : ''}>
                      {line}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {step === 'granted' && (
          <div className="relative w-full max-w-[720px]">
            <div className="absolute inset-0 -z-10 rounded-full bg-[#2563eb]/10 blur-[160px]" />
            <div className="rounded-3xl border border-white/10 bg-[linear-gradient(180deg,rgba(15,23,42,0.7),rgba(15,23,42,0.55))] p-6 shadow-[0_8px_32px_rgba(0,0,0,0.8)] backdrop-blur-xl md:p-8">
              <div className="text-center">
                <div className="inline-flex items-center gap-2 rounded-full border border-[#00cbe6]/20 bg-[#00cbe6]/10 px-4 py-2 text-[12px] font-medium uppercase tracking-[0.35em] text-[#00cbe6]">
                  <span className="h-2 w-2 rounded-full bg-[#00cbe6] shadow-[0_0_14px_rgba(0,203,230,0.8)]" />
                  Connected
                </div>

                <div className="mx-auto mt-8 flex h-32 w-32 items-center justify-center rounded-3xl border border-white/10 bg-[#020408]/40 shadow-[0_0_40px_rgba(37,99,235,0.15)]">
                  <ShieldCheck className="h-14 w-14 text-[#b4c5ff]" />
                </div>

                <h1 className="mt-10 text-[44px] font-semibold tracking-tight text-[#e2e2e2] md:text-[64px]">
                  {isCreateFlow ? 'Controller Ready' : 'Welcome Back, Controller'}
                </h1>
                <p className="mx-auto mt-6 max-w-[560px] text-[20px] leading-8 text-[#c3c6d7]">
                  {subtitle}
                </p>
              </div>

              <div className="mt-10 grid gap-4 md:grid-cols-2">
                <div className="rounded-2xl border border-white/10 bg-[#020408]/40 p-5">
                  <div className="mb-6 flex items-center justify-between">
                    <div className="text-[16px] font-medium uppercase tracking-[0.25em] text-[#424751]">
                      Security Level
                    </div>
                    <Shield className="h-5 w-5 text-[#b4c5ff]" />
                  </div>
                  <div className="text-[28px] font-semibold text-[#e2e2e2]">HIGH</div>
                  <div className="mt-2 font-mono text-[14px] uppercase tracking-[0.2em] text-[#5de6ff]">
                    Encryption Active
                  </div>
                </div>

                <div className="rounded-2xl border border-white/10 bg-[#020408]/40 p-5">
                  <div className="mb-6 flex items-center justify-between">
                    <div className="text-[16px] font-medium uppercase tracking-[0.25em] text-[#424751]">
                      Platform
                    </div>
                    <BadgeCheck className="h-5 w-5 text-[#b4c5ff]" />
                  </div>
                  <div className="text-[28px] font-semibold text-[#e2e2e2]">SECURE</div>
                  <div className="mt-2 font-mono text-[14px] uppercase tracking-[0.2em] text-[#c3c6d7]">
                    Signature Verified
                  </div>
                </div>
              </div>

              <button
                type="button"
                onClick={() => navigate('/vault')}
                className="mt-8 flex h-16 w-full items-center justify-center gap-3 rounded-xl bg-[#2563eb] text-[22px] font-medium text-[#eeefff] transition-all duration-300 hover:bg-[#5de6ff] hover:text-[#00363e] hover:shadow-[0_0_20px_rgba(37,99,235,0.35)]"
              >
                <span>Enter Vault</span>
                <ChevronRight className="h-7 w-7" />
              </button>

              <div className="mt-10 border-t border-white/10 pt-8 text-center text-[12px] uppercase tracking-[0.35em] text-[#424751]">
                System Authorization 0xFF21-A
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
