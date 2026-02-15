import { useState } from 'react';
import { useNavigate } from 'react-router';
import { Shield, Key, Download, ArrowRight, Loader, Eye, EyeOff } from 'lucide-react';
import Layout from '@/react-app/components/Layout';
import { useWallet } from '@/react-app/hooks/useWallet';

type Step = 'welcome' | 'create' | 'import' | 'password' | 'mnemonic' | 'confirm' | 'success';

export default function Onboarding() {
  const navigate = useNavigate();
  const { createWallet, importWallet } = useWallet();
  const [step, setStep] = useState<Step>('welcome');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [mnemonic, setMnemonic] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleCreateWallet = async () => {
    if (password !== confirmPassword) {
      setError("Passwords don't match");
      return;
    }
    if (password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }

    setIsLoading(true);
    try {
      const phrase = await createWallet(password);
      setMnemonic(phrase);
      setStep('mnemonic');
    } catch (err: any) {
      setError(err.message || "Failed to create wallet");
    } finally {
      setIsLoading(false);
    }
  };

  const handleImportWallet = async () => {
    if (password !== confirmPassword) {
      setError("Passwords don't match");
      return;
    }
    if (mnemonic.split(' ').length !== 12 && mnemonic.split(' ').length !== 24) {
      setError("Invalid mnemonic phrase (must be 12 or 24 words)");
      return;
    }

    setIsLoading(true);
    try {
      await importWallet(mnemonic, password);
      setStep('success');
      setTimeout(() => navigate('/wallet'), 2000);
    } catch (err: any) {
      setError(err.message || "Failed to import wallet");
    } finally {
      setIsLoading(false);
    }
  };

  const handleContinue = () => {
    setStep('success');
    setTimeout(() => navigate('/wallet'), 2000);
  };

  const renderWelcome = () => (
    <div className="space-y-6 text-center">
      <div className="mx-auto w-16 h-16 bg-blue-600/20 rounded-2xl flex items-center justify-center border border-blue-500/30">
        <Shield className="w-8 h-8 text-blue-400" />
      </div>
      <div>
        <h2 className="text-2xl font-bold text-white mb-2">Welcome to KNIRV</h2>
        <p className="text-slate-400 text-sm">Secure your digital identity and assets</p>
      </div>
      <div className="grid gap-4">
        <button
          onClick={() => setStep('create')}
          className="p-4 bg-blue-600 hover:bg-blue-500 text-white rounded-xl font-medium transition-all flex items-center justify-between group"
        >
          <div className="text-left">
            <div className="font-bold text-sm">Create New Wallet</div>
            <div className="text-xs text-blue-200 mt-1">Generate a new 12-word seed phrase</div>
          </div>
          <ArrowRight className="w-5 h-5 transform group-hover:translate-x-1 transition-transform" />
        </button>
        <button
          onClick={() => setStep('import')}
          className="p-4 bg-slate-800 hover:bg-slate-700 text-white rounded-xl font-medium transition-all flex items-center justify-between group border border-slate-700"
        >
          <div className="text-left">
            <div className="font-bold text-sm">Import Existing Wallet</div>
            <div className="text-xs text-slate-400 mt-1">Restore using your secret recovery phrase</div>
          </div>
          <Download className="w-5 h-5 text-slate-400 group-hover:text-white transition-colors" />
        </button>
      </div>
    </div>
  );

  const renderPassword = (action: 'create' | 'import') => (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-xl font-bold text-white mb-2">Set Password</h2>
        <p className="text-slate-400 text-sm">This password will unlock your KNIRV wallet on this device only.</p>
      </div>

      <div className="space-y-4">
        <div className="space-y-2">
          <label className="text-xs font-mono text-slate-500 uppercase">New Password</label>
          <div className="relative">
            <input
              type={showPassword ? "text" : "password"}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-slate-900 border border-slate-700 rounded-lg p-3 text-white focus:border-blue-500 focus:outline-none transition-colors"
              placeholder="Min. 8 characters"
            />
            <button
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-3 text-slate-500 hover:text-white"
            >
              {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          </div>
        </div>
        <div className="space-y-2">
          <label className="text-xs font-mono text-slate-500 uppercase">Confirm Password</label>
          <input
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            className="w-full bg-slate-900 border border-slate-700 rounded-lg p-3 text-white focus:border-blue-500 focus:outline-none transition-colors"
            placeholder="Confirm password"
          />
        </div>

        {error && (
          <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-xs text-center">
            {error}
          </div>
        )}

        {action === 'import' && (
           <div className="space-y-2">
           <label className="text-xs font-mono text-slate-500 uppercase">Recovery Phrase</label>
           <textarea
             value={mnemonic}
             onChange={(e) => setMnemonic(e.target.value)}
             className="w-full bg-slate-900 border border-slate-700 rounded-lg p-3 text-white focus:border-blue-500 focus:outline-none transition-colors h-24 text-sm font-mono"
             placeholder="Enter your 12 or 24 word phrase separated by spaces"
           />
         </div>
        )}

        <button
          onClick={action === 'create' ? handleCreateWallet : handleImportWallet}
          disabled={isLoading || !password || !confirmPassword || (action === 'import' && !mnemonic)}
          className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-xl font-bold text-sm uppercase tracking-wide transition-all flex items-center justify-center space-x-2"
        >
          {isLoading ? (
            <Loader className="w-4 h-4 animate-spin" />
          ) : (
            <span>{action === 'create' ? 'Create Wallet' : 'Import Wallet'}</span>
          )}
        </button>
      </div>
    </div>
  );

  const renderMnemonic = () => (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-xl font-bold text-white mb-2">Secret Recovery Phrase</h2>
        <p className="text-slate-400 text-sm">Write down these 12 words in order and keep them safe. You will need them to recover your wallet.</p>
      </div>

      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4">
        <div className="grid grid-cols-3 gap-3">
          {mnemonic.split(' ').map((word, index) => (
            <div key={index} className="flex items-center space-x-2 p-2 bg-slate-950 rounded border border-slate-800">
              <span className="text-slate-600 text-xs font-mono w-4">{index + 1}.</span>
              <span className="text-blue-300 text-sm font-bold">{word}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="p-4 bg-yellow-500/10 border border-yellow-500/20 rounded-xl flex items-start space-x-3">
        <Key className="w-5 h-5 text-yellow-500 flex-shrink-0 mt-0.5" />
        <p className="text-xs text-yellow-200">
          Warning: Never disclose your recovery phrase. Anyone with this phrase can take your assets forever.
        </p>
      </div>

      <button
        onClick={handleContinue}
        className="w-full py-3 bg-blue-600 hover:bg-blue-500 text-white rounded-xl font-bold text-sm uppercase tracking-wide transition-all"
      >
        I have saved my phrase
      </button>
    </div>
  );

  const renderSuccess = () => (
    <div className="text-center space-y-6 py-8">
      <div className="mx-auto w-20 h-20 bg-green-500/20 rounded-full flex items-center justify-center border border-green-500/30 animate-pulse">
        <Shield className="w-10 h-10 text-green-400" />
      </div>
      <div>
        <h2 className="text-2xl font-bold text-white mb-2">Wallet Ready!</h2>
        <p className="text-slate-400 text-sm">Redirecting to dashboard...</p>
      </div>
    </div>
  );

  return (
    <Layout>
      <div className="min-h-[80vh] flex items-center justify-center p-4">
        <div className="w-full max-w-md bg-slate-950/50 backdrop-blur-xl border border-white/5 rounded-3xl p-6 shadow-2xl">
          {step === 'welcome' && renderWelcome()}
          {step === 'create' && renderPassword('create')}
          {step === 'import' && renderPassword('import')}
          {step === 'mnemonic' && renderMnemonic()}
          {step === 'success' && renderSuccess()}
        </div>
      </div>
    </Layout>
  );
}
