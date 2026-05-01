import { useState } from 'react';
import { useNavigate } from 'react-router';
import { Cpu, Shield, Check, ArrowLeft, ArrowRight, BadgeCheck, Zap, Loader } from 'lucide-react';
import Layout from '@/react-app/components/Layout';

// ─── Types ───────────────────────────────────────────────────────────────────

type Step = 1 | 2 | 3 | 4 | 5;
type TEEType = 'hardware' | 'browser-extension';

interface BadgeOption {
  id: string;
  label: string;
  color: 'blue' | 'green' | 'purple' | 'amber' | 'cyan' | 'red';
  description: string;
}

// ─── Mock Data ───────────────────────────────────────────────────────────────

const BADGE_INVENTORY: BadgeOption[] = [
  { id: 'attested', label: 'ATTESTED', color: 'green', description: 'TEE attestation verified' },
  { id: 'sgx-20', label: 'SGX 2.0', color: 'blue', description: 'Intel SGX 2.0 support' },
  { id: 'low-latency', label: 'LOW LATENCY', color: 'cyan', description: 'Sub-10ms response time' },
  { id: 'verified', label: 'VERIFIED', color: 'green', description: 'Identity verified' },
  { id: 'confidential', label: 'CONFIDENTIAL', color: 'purple', description: 'Confidential compute enabled' },
  { id: 'browser', label: 'BROWSER', color: 'purple', description: 'Browser extension compatible' },
  { id: 'dev-mode', label: 'DEV MODE', color: 'amber', description: 'Development mode enabled' },
  { id: 'high-availability', label: 'HIGH AVAIL', color: 'red', description: '99.99% uptime SLA' },
  { id: 'audited', label: 'AUDITED', color: 'blue', description: 'Security audit passed' },
];

const STEPS: { num: Step; label: string }[] = [
  { num: 1, label: 'Name & Type' },
  { num: 2, label: 'Stake' },
  { num: 3, label: 'Badges' },
  { num: 4, label: 'Confirm' },
  { num: 5, label: 'Submit' },
];

// ─── Badge Chip Colors ───────────────────────────────────────────────────────

const badgeColorMap: Record<string, string> = {
  blue: 'bg-blue-500/15 text-blue-400 border-blue-500/25',
  green: 'bg-green-500/15 text-green-400 border-green-500/25',
  purple: 'bg-purple-500/15 text-purple-400 border-purple-500/25',
  amber: 'bg-amber-500/15 text-amber-400 border-amber-500/25',
  cyan: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/25',
  red: 'bg-red-500/15 text-red-400 border-red-500/25',
};

const badgeColorBgMap: Record<string, string> = {
  blue: 'bg-blue-500/10 border-blue-500/20',
  green: 'bg-green-500/10 border-green-500/20',
  purple: 'bg-purple-500/10 border-purple-500/20',
  amber: 'bg-amber-500/10 border-amber-500/20',
  cyan: 'bg-cyan-500/10 border-cyan-500/20',
  red: 'bg-red-500/10 border-red-500/20',
};

// ─── Sub-Components ──────────────────────────────────────────────────────────

const StepIndicator: React.FC<{ current: Step }> = ({ current }) => (
  <div className="flex items-center justify-center space-x-2 mb-8">
    {STEPS.map((s, idx) => (
      <div key={s.num} className="flex items-center">
        <div
          className={`flex items-center justify-center w-8 h-8 rounded-full text-xs font-bold font-mono transition-all duration-300 ${
            current === s.num
              ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/30 scale-110'
              : current > s.num
              ? 'bg-green-500/20 text-green-400 border border-green-500/30'
              : 'bg-slate-800 text-slate-500 border border-slate-700'
          }`}
        >
          {current > s.num ? <Check className="w-3.5 h-3.5" /> : s.num}
        </div>
        <span
          className={`ml-1.5 text-[9px] font-bold uppercase tracking-widest hidden sm:block ${
            current === s.num ? 'text-blue-400' : current > s.num ? 'text-green-400' : 'text-slate-600'
          }`}
        >
          {s.label}
        </span>
        {idx < STEPS.length - 1 && (
          <div
            className={`w-6 sm:w-10 h-px mx-1.5 ${
              current > s.num ? 'bg-green-500/40' : 'bg-slate-700'
            }`}
          />
        )}
      </div>
    ))}
  </div>
);

const StyledInput: React.FC<{
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: string;
  disabled?: boolean;
}> = ({ label, value, onChange, placeholder, type = 'text', disabled }) => (
  <div className="space-y-1.5">
    <label className="text-[10px] font-black text-slate-400 uppercase tracking-widest">{label}</label>
    <input
      type={type}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      disabled={disabled}
      className="w-full bg-slate-900 border border-slate-700 rounded-xl p-3.5 text-white text-sm font-mono focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500/30 transition-all placeholder:text-slate-600 disabled:opacity-50 disabled:cursor-not-allowed"
    />
  </div>
);

const StyledNumberInput: React.FC<{
  label: string;
  value: number;
  onChange: (v: number) => void;
  min?: number;
  max?: number;
  suffix?: string;
}> = ({ label, value, onChange, min = 0, max, suffix }) => (
  <div className="space-y-1.5">
    <label className="text-[10px] font-black text-slate-400 uppercase tracking-widest">{label}</label>
    <div className="relative">
      <input
        type="number"
        value={value}
        onChange={(e) => onChange(Math.max(min, parseInt(e.target.value) || 0))}
        min={min}
        max={max}
        className="w-full bg-slate-900 border border-slate-700 rounded-xl p-3.5 text-white text-sm font-mono focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500/30 transition-all placeholder:text-slate-600 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
        placeholder="0"
      />
      {suffix && (
        <span className="absolute right-3.5 top-1/2 -translate-y-1/2 text-xs font-bold text-blue-400">{suffix}</span>
      )}
    </div>
  </div>
);

const NavButtons: React.FC<{
  onBack?: () => void;
  onNext?: () => void;
  backLabel?: string;
  nextLabel?: string;
  nextDisabled?: boolean;
  nextLoading?: boolean;
  showBack?: boolean;
  showNext?: boolean;
}> = ({ onBack, onNext, backLabel = 'Back', nextLabel = 'Next', nextDisabled = false, nextLoading = false, showBack = true, showNext = true }) => (
  <div className="flex items-center justify-between pt-6 border-t border-white/5 mt-6">
    {showBack ? (
      <button
        onClick={onBack}
        className="inline-flex items-center space-x-1.5 px-4 py-2.5 rounded-xl border border-slate-700 text-slate-300 text-xs font-bold hover:bg-slate-800 hover:text-white transition-all uppercase tracking-wider"
      >
        <ArrowLeft className="w-3.5 h-3.5" />
        <span>{backLabel}</span>
      </button>
    ) : (
      <div />
    )}
    {showNext && (
      <button
        onClick={onNext}
        disabled={nextDisabled || nextLoading}
        className="inline-flex items-center space-x-1.5 px-5 py-2.5 rounded-xl bg-blue-600 text-white text-xs font-bold hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all uppercase tracking-wider shadow-lg shadow-blue-600/20"
      >
        {nextLoading ? (
          <Loader className="w-3.5 h-3.5 animate-spin" />
        ) : (
          <>
            <span>{nextLabel}</span>
            <ArrowRight className="w-3.5 h-3.5" />
          </>
        )}
      </button>
    )}
  </div>
);

// ─── Step Renderers ──────────────────────────────────────────────────────────

interface DVECreateState {
  name: string;
  teeType: TEEType | null;
  stakeAmount: number;
  selectedBadges: string[];
}

const renderNameType = (
  state: DVECreateState,
  setState: React.Dispatch<React.SetStateAction<DVECreateState>>
) => (
  <div className="space-y-6">
    <div className="text-center">
      <div className="mx-auto w-12 h-12 bg-blue-600/20 rounded-2xl flex items-center justify-center border border-blue-500/30 mb-3">
        <Cpu className="w-6 h-6 text-blue-400" />
      </div>
      <h3 className="text-lg font-bold text-white">Name Your DVE</h3>
      <p className="text-slate-400 text-xs mt-1">Choose a name and trusted execution environment type</p>
    </div>

    <StyledInput
      label="DVE Name"
      value={state.name}
      onChange={(v) => setState((s) => ({ ...s, name: v }))}
      placeholder="e.g. DVE-Alpha"
    />

    <div className="space-y-2">
      <label className="text-[10px] font-black text-slate-400 uppercase tracking-widest">TEE Type</label>
      <div className="grid grid-cols-2 gap-3">
        <button
          onClick={() => setState((s) => ({ ...s, teeType: 'hardware' }))}
          className={`relative p-4 rounded-xl border text-left transition-all ${
            state.teeType === 'hardware'
              ? 'bg-blue-600/20 border-blue-500/50 shadow-lg shadow-blue-600/10'
              : 'bg-slate-900/80 border-slate-700 hover:border-slate-500'
          }`}
        >
          <div className={`w-10 h-10 rounded-lg flex items-center justify-center mb-2 ${
            state.teeType === 'hardware' ? 'bg-blue-600/30' : 'bg-slate-800'
          }`}>
            <Shield className={`w-5 h-5 ${state.teeType === 'hardware' ? 'text-blue-400' : 'text-slate-500'}`} />
          </div>
          <h4 className={`text-sm font-bold ${state.teeType === 'hardware' ? 'text-white' : 'text-slate-300'}`}>Hardware TEE</h4>
          <p className="text-[10px] text-slate-500 mt-1">SGX / TDX / SEV-SNP</p>
          {state.teeType === 'hardware' && (
            <div className="absolute top-2 right-2 w-5 h-5 bg-blue-600 rounded-full flex items-center justify-center">
              <Check className="w-3 h-3 text-white" />
            </div>
          )}
        </button>
        <button
          onClick={() => setState((s) => ({ ...s, teeType: 'browser-extension' }))}
          className={`relative p-4 rounded-xl border text-left transition-all ${
            state.teeType === 'browser-extension'
              ? 'bg-purple-600/20 border-purple-500/50 shadow-lg shadow-purple-600/10'
              : 'bg-slate-900/80 border-slate-700 hover:border-slate-500'
          }`}
        >
          <div className={`w-10 h-10 rounded-lg flex items-center justify-center mb-2 ${
            state.teeType === 'browser-extension' ? 'bg-purple-600/30' : 'bg-slate-800'
          }`}>
            <Cpu className={`w-5 h-5 ${state.teeType === 'browser-extension' ? 'text-purple-400' : 'text-slate-500'}`} />
          </div>
          <h4 className={`text-sm font-bold ${state.teeType === 'browser-extension' ? 'text-white' : 'text-slate-300'}`}>Browser Extension</h4>
          <p className="text-[10px] text-slate-500 mt-1">Lightweight client</p>
          {state.teeType === 'browser-extension' && (
            <div className="absolute top-2 right-2 w-5 h-5 bg-purple-600 rounded-full flex items-center justify-center">
              <Check className="w-3 h-3 text-white" />
            </div>
          )}
        </button>
      </div>
    </div>
  </div>
);

const renderStake = (
  state: DVECreateState,
  setState: React.Dispatch<React.SetStateAction<DVECreateState>>
) => (
  <div className="space-y-6">
    <div className="text-center">
      <div className="mx-auto w-12 h-12 bg-blue-600/20 rounded-2xl flex items-center justify-center border border-blue-500/30 mb-3">
        <Zap className="w-6 h-6 text-blue-400" />
      </div>
      <h3 className="text-lg font-bold text-white">Stake NRN</h3>
      <p className="text-slate-400 text-xs mt-1">Allocate NRN tokens from your Vault to this DVE</p>
    </div>

    {/* Vault Balance Info */}
    <div className="bg-slate-900/80 border border-slate-700 rounded-xl p-3.5 flex items-center justify-between">
      <div className="flex items-center space-x-2.5">
        <div className="w-8 h-8 bg-blue-600/20 rounded-lg flex items-center justify-center">
          <Zap className="w-4 h-4 text-blue-400" />
        </div>
        <div>
          <p className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Available in Vault</p>
          <p className="text-sm font-bold text-white font-mono">18,247 NRN</p>
        </div>
      </div>
      <button
        onClick={() => setState((s) => ({ ...s, stakeAmount: 18247 }))}
        className="text-[9px] font-bold text-blue-400 uppercase tracking-wider px-2 py-1 rounded-lg bg-blue-500/10 border border-blue-500/20 hover:bg-blue-500/20 transition-all"
      >
        Max
      </button>
    </div>

    <StyledNumberInput
      label="Stake Amount"
      value={state.stakeAmount}
      onChange={(v) => setState((s) => ({ ...s, stakeAmount: v }))}
      min={100}
      max={18247}
      suffix="NRN"
    />

    <div className="flex items-center space-x-2 text-[10px] text-slate-500">
      <Shield className="w-3 h-3 text-amber-400" />
      <span>Minimum stake: <span className="text-amber-300 font-bold">100 NRN</span></span>
    </div>

    {/* Stake tiers */}
    <div className="space-y-2">
      <label className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Recommended Tiers</label>
      <div className="grid grid-cols-3 gap-2">
        {[
          { label: 'Starter', amount: 500 },
          { label: 'Standard', amount: 5000 },
          { label: 'Premium', amount: 12000 },
        ].map((tier) => (
          <button
            key={tier.label}
            onClick={() => setState((s) => ({ ...s, stakeAmount: tier.amount }))}
            className={`p-2.5 rounded-xl border text-center transition-all ${
              state.stakeAmount === tier.amount
                ? 'bg-blue-600/20 border-blue-500/40'
                : 'bg-slate-900/60 border-slate-700 hover:border-slate-500'
            }`}
          >
            <p className={`text-[9px] font-black uppercase tracking-widest ${
              state.stakeAmount === tier.amount ? 'text-blue-400' : 'text-slate-400'
            }`}>{tier.label}</p>
            <p className={`text-xs font-bold font-mono ${
              state.stakeAmount === tier.amount ? 'text-white' : 'text-slate-300'
            }`}>{tier.amount.toLocaleString()}</p>
            <p className="text-[8px] text-slate-600">NRN</p>
          </button>
        ))}
      </div>
    </div>
  </div>
);

const renderBadges = (
  state: DVECreateState,
  setState: React.Dispatch<React.SetStateAction<DVECreateState>>
) => {
  const toggleBadge = (id: string) => {
    setState((s) => ({
      ...s,
      selectedBadges: s.selectedBadges.includes(id)
        ? s.selectedBadges.filter((b) => b !== id)
        : [...s.selectedBadges, id],
    }));
  };

  return (
    <div className="space-y-6">
      <div className="text-center">
        <div className="mx-auto w-12 h-12 bg-blue-600/20 rounded-2xl flex items-center justify-center border border-blue-500/30 mb-3">
          <BadgeCheck className="w-6 h-6 text-blue-400" />
        </div>
        <h3 className="text-lg font-bold text-white">Select Badges</h3>
        <p className="text-slate-400 text-xs mt-1">
          Choose capability badges from your inventory ({state.selectedBadges.length} selected)
        </p>
      </div>

      <div className="space-y-2">
        {BADGE_INVENTORY.map((badge) => {
          const selected = state.selectedBadges.includes(badge.id);
          return (
            <button
              key={badge.id}
              onClick={() => toggleBadge(badge.id)}
              className={`w-full flex items-center justify-between p-3 rounded-xl border transition-all ${
                selected
                  ? `${badgeColorBgMap[badge.color]} shadow-lg`
                  : 'bg-slate-900/60 border-slate-700 hover:border-slate-500'
              }`}
            >
              <div className="flex items-center space-x-3">
                <div
                  className={`w-8 h-8 rounded-lg flex items-center justify-center border ${
                    selected
                      ? badgeColorBgMap[badge.color]
                      : 'bg-slate-800 border-slate-700'
                  }`}
                >
                  <BadgeCheck className={`w-4 h-4 ${
                    selected ? 'text-green-400' : 'text-slate-500'
                  }`} />
                </div>
                <div className="text-left">
                  <span
                    className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[9px] font-black uppercase tracking-wider ${
                      selected
                        ? badgeColorMap[badge.color]
                        : 'bg-slate-800/60 text-slate-500 border-slate-700'
                    }`}
                  >
                    {badge.label}
                  </span>
                  <p className="text-[10px] text-slate-500 mt-0.5">{badge.description}</p>
                </div>
              </div>
              <div
                className={`w-5 h-5 rounded-full border-2 flex items-center justify-center transition-all ${
                  selected
                    ? 'bg-green-500 border-green-500'
                    : 'border-slate-600 bg-transparent'
                }`}
              >
                {selected && <Check className="w-3 h-3 text-white" />}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
};

const renderConfirm = (state: DVECreateState) => {
  const teeLabel = state.teeType === 'hardware' ? 'Hardware TEE (SGX/TDX/SEV-SNP)' : 'Browser Extension';

  // Derived mock DVE wallet address
  const derivedWallet = `0x${Array.from({ length: 40 }, () =>
    Math.floor(Math.random() * 16).toString(16)
  ).join('')}`;

  // Estimated reputation based on stake + badges
  const estReputation = Math.min(1000, Math.floor(100 + (state.stakeAmount / 18247) * 600 + state.selectedBadges.length * 30));

  const selectedBadgeItems = BADGE_INVENTORY.filter((b) => state.selectedBadges.includes(b.id));

  return (
    <div className="space-y-5">
      <div className="text-center">
        <div className="mx-auto w-12 h-12 bg-blue-600/20 rounded-2xl flex items-center justify-center border border-blue-500/30 mb-3">
          <Shield className="w-6 h-6 text-blue-400" />
        </div>
        <h3 className="text-lg font-bold text-white">Confirm DVE</h3>
        <p className="text-slate-400 text-xs mt-1">Review your DVE configuration before submitting</p>
      </div>

      {/* Derived Wallet */}
      <div className="bg-slate-900/80 border border-slate-700 rounded-xl p-3.5 space-y-1.5">
        <label className="text-[9px] font-black text-slate-500 uppercase tracking-widest">DVE Wallet Address</label>
        <div className="flex items-center space-x-2">
          <div className="w-7 h-7 bg-blue-600/20 rounded-lg flex items-center justify-center">
            <Cpu className="w-3.5 h-3.5 text-blue-400" />
          </div>
          <span className="text-xs font-mono text-blue-300 font-bold truncate">{derivedWallet}</span>
        </div>
        <p className="text-[9px] text-slate-600">Derived from your vault seed + DVE identity</p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 gap-3">
        <div className="bg-slate-900/80 border border-slate-700 rounded-xl p-3 space-y-1">
          <label className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Name</label>
          <p className="text-sm font-bold text-white font-mono">{state.name || '—'}</p>
        </div>
        <div className="bg-slate-900/80 border border-slate-700 rounded-xl p-3 space-y-1">
          <label className="text-[9px] font-black text-slate-500 uppercase tracking-widest">TEE Type</label>
          <div className="flex items-center space-x-1.5">
            {state.teeType === 'hardware' ? (
              <Shield className="w-3.5 h-3.5 text-blue-400" />
            ) : (
              <Cpu className="w-3.5 h-3.5 text-purple-400" />
            )}
            <span className="text-xs font-bold text-white">{teeLabel}</span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="bg-slate-900/80 border border-slate-700 rounded-xl p-3 space-y-1">
          <label className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Stake Amount</label>
          <p className="text-sm font-bold text-amber-300 font-mono">{state.stakeAmount.toLocaleString()} NRN</p>
        </div>
        <div className="bg-slate-900/80 border border-slate-700 rounded-xl p-3 space-y-1">
          <label className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Est. Reputation</label>
          <div className="flex items-center space-x-1.5">
            <ActivityIcon />
            <p className="text-sm font-bold text-green-400 font-mono">{estReputation}/1000</p>
          </div>
        </div>
      </div>

      {/* Badges Summary */}
      <div className="bg-slate-900/80 border border-slate-700 rounded-xl p-3 space-y-2">
        <div className="flex items-center justify-between">
          <label className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Badges ({selectedBadgeItems.length})</label>
        </div>
        {selectedBadgeItems.length > 0 ? (
          <div className="flex flex-wrap gap-1.5">
            {selectedBadgeItems.map((b) => (
              <span
                key={b.id}
                className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[9px] font-black uppercase tracking-wider ${badgeColorMap[b.color]}`}
              >
                <BadgeCheck className="w-2.5 h-2.5 mr-0.5" />
                {b.label}
              </span>
            ))}
          </div>
        ) : (
          <p className="text-xs text-slate-500">No badges selected</p>
        )}
      </div>

      {/* Capabilities Summary */}
      <div className="bg-slate-900/80 border border-slate-700 rounded-xl p-3 space-y-2">
        <label className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Capabilities</label>
        <ul className="space-y-1">
          <CapabilityItem text={`${state.teeType === 'hardware' ? 'Hardware-grade' : 'Browser-based'} trusted execution`} />
          <CapabilityItem text={`${state.stakeAmount.toLocaleString()} NRN staked for network participation`} />
          <CapabilityItem text={`${state.selectedBadges.length} capability badge${state.selectedBadges.length !== 1 ? 's' : ''} applied`} />
          <CapabilityItem text="D-TEN network discovery enabled" />
        </ul>
      </div>
    </div>
  );
};

const ActivityIcon = () => (
  <svg className="w-3.5 h-3.5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
    <path strokeLinecap="round" strokeLinejoin="round" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
  </svg>
);

const CapabilityItem: React.FC<{ text: string }> = ({ text }) => (
  <li className="flex items-center space-x-2 text-[10px] text-slate-400">
    <Check className="w-3 h-3 text-green-500 flex-shrink-0" />
    <span>{text}</span>
  </li>
);

const renderSubmit = (
  state: DVECreateState,
  isSubmitting: boolean,
  submitError: string | null,
  onSubmit: () => void
) => {
  const teeLabel = state.teeType === 'hardware' ? 'Hardware TEE' : 'Browser Extension';
  const selectedBadgeItems = BADGE_INVENTORY.filter((b) => state.selectedBadges.includes(b.id));

  return (
    <div className="space-y-6 text-center">
      <div className="mx-auto w-16 h-16 bg-gradient-to-br from-blue-600/30 to-blue-800/30 rounded-2xl flex items-center justify-center border border-blue-500/30">
        <Zap className="w-8 h-8 text-blue-400" />
      </div>
      <div>
        <h3 className="text-xl font-bold text-white mb-2">Ready to Deploy</h3>
        <p className="text-slate-400 text-sm">Your DVE is configured and ready to join the D-TEN network</p>
      </div>

      {/* Final Summary */}
      <div className="bg-slate-900/80 border border-slate-700 rounded-xl p-4 space-y-3 text-left">
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500">Name</span>
          <span className="text-white font-bold font-mono">{state.name}</span>
        </div>
        <div className="border-t border-slate-800" />
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500">TEE Type</span>
          <span className="text-white font-bold">{teeLabel}</span>
        </div>
        <div className="border-t border-slate-800" />
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500">Stake</span>
          <span className="text-amber-300 font-bold font-mono">{state.stakeAmount.toLocaleString()} NRN</span>
        </div>
        <div className="border-t border-slate-800" />
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500">Badges</span>
          <span className="text-white font-bold">{selectedBadgeItems.length}</span>
        </div>
        {selectedBadgeItems.length > 0 && (
          <>
            <div className="border-t border-slate-800" />
            <div className="flex flex-wrap gap-1">
              {selectedBadgeItems.map((b) => (
                <span
                  key={b.id}
                  className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[8px] font-black uppercase tracking-wider ${badgeColorMap[b.color]}`}
                >
                  {b.label}
                </span>
              ))}
            </div>
          </>
        )}
      </div>

      {submitError && (
        <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-xs text-center">
          {submitError}
        </div>
      )}

      <button
        onClick={onSubmit}
        disabled={isSubmitting}
        className="w-full py-3.5 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-500 hover:to-blue-600 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-xl font-bold text-sm uppercase tracking-wider transition-all shadow-lg shadow-blue-600/30 flex items-center justify-center space-x-2"
      >
        {isSubmitting ? (
          <>
            <Loader className="w-4 h-4 animate-spin" />
            <span>Deploying DVE...</span>
          </>
        ) : (
          <>
            <Zap className="w-4 h-4" />
            <span>Deploy DVE</span>
          </>
        )}
      </button>

      <p className="text-[10px] text-slate-600">
        This will submit a transaction to deploy your DVE on the D-TEN network.
        A small gas fee (~0.001 ETH) will be required.
      </p>
    </div>
  );
};

// ─── Main Component ──────────────────────────────────────────────────────────

export default function DVECreate() {
  const navigate = useNavigate();
  const [step, setStep] = useState<Step>(1);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const [state, setState] = useState<DVECreateState>({
    name: '',
    teeType: null,
    stakeAmount: 0,
    selectedBadges: [],
  });

  const canAdvanceFromStep = (s: Step): boolean => {
    switch (s) {
      case 1:
        return state.name.trim().length >= 2 && state.teeType !== null;
      case 2:
        return state.stakeAmount >= 100;
      case 3:
        return true; // badges are optional
      case 4:
        return true;
      default:
        return false;
    }
  };

  const handleNext = () => {
    if (step < 5) {
      setStep((step + 1) as Step);
    }
  };

  const handleBack = () => {
    if (step > 1) {
      setStep((step - 1) as Step);
    }
  };

  const handleSubmit = async () => {
    setIsSubmitting(true);
    setSubmitError(null);

    try {
      // Mock API call with artificial delay
      await new Promise((resolve, reject) => {
        setTimeout(() => {
          // Simulate 90% success rate
          if (Math.random() > 0.1) {
            resolve(true);
          } else {
            reject(new Error('Network congestion. Please try again.'));
          }
        }, 2000);
      });

      // Navigate to home on success
      navigate('/');
    } catch (err: any) {
      setSubmitError(err.message || 'Failed to deploy DVE. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const renderStepContent = () => {
    switch (step) {
      case 1:
        return renderNameType(state, setState);
      case 2:
        return renderStake(state, setState);
      case 3:
        return renderBadges(state, setState);
      case 4:
        return renderConfirm(state);
      case 5:
        return renderSubmit(state, isSubmitting, submitError, handleSubmit);
      default:
        return null;
    }
  };

  return (
    <Layout>
      <div className="p-4 pb-24 space-y-4 max-w-lg mx-auto">
        {/* Page Header */}
        <div className="text-center py-2">
          <div className="flex items-center justify-center space-x-2 mb-1">
            <div className="w-5 h-5 bg-blue-600 rounded-sm transform rotate-45" />
            <h2 className="text-xl font-extrabold tracking-tighter uppercase text-white">
              Create <span className="text-blue-500 font-light">DVE</span>
            </h2>
          </div>
          <p className="text-slate-500 text-[10px] font-mono uppercase tracking-wider">
            New Distributed Verification Environment
          </p>
        </div>

        {/* Glass Panel Wizard */}
        <div className="relative group">
          <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/20 to-blue-800/20 rounded-2xl blur opacity-20" />
          <div className="relative glass-panel p-5 glow-hover">
            <StepIndicator current={step} />

            {/* Step Counter */}
            <div className="text-center mb-4">
              <span className="text-[9px] font-black text-slate-500 uppercase tracking-widest">
                Step {step} of 5
              </span>
            </div>

            {renderStepContent()}

            {/* Navigation Buttons (not on step 5 — step 5 has its own submit button) */}
            {step < 5 && (
              <NavButtons
                onBack={handleBack}
                onNext={handleNext}
                showBack={step > 1}
                nextDisabled={!canAdvanceFromStep(step)}
                backLabel="Previous"
              />
            )}

            {step === 5 && !isSubmitting && (
              <div className="flex items-center justify-center pt-4">
                <button
                  onClick={handleBack}
                  className="inline-flex items-center space-x-1.5 px-4 py-2 rounded-xl border border-slate-700 text-slate-300 text-xs font-bold hover:bg-slate-800 hover:text-white transition-all uppercase tracking-wider"
                >
                  <ArrowLeft className="w-3.5 h-3.5" />
                  <span>Back to Confirm</span>
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </Layout>
  );
}
