import { useState, type ReactElement } from 'react';
import { Check, Globe, Lock, Server, ShieldCheck } from 'lucide-react';
import { NETWORK_OPTIONS, type NetworkId } from '@/react-app/platform/networkConfig';

interface NetworkSelectorDialogProps {
  isOpen: boolean;
  currentNetworkId: NetworkId;
  currentDevServerUrl: string;
  onSelect: (id: NetworkId, devServerUrl?: string) => void;
  onClose?: () => void;
}

const NETWORK_ICONS: Record<NetworkId, ReactElement> = {
  mainnet: <Lock className="h-6 w-6" />,
  testnet: <Globe className="h-6 w-6" />,
  dev: <Server className="h-6 w-6" />,
};

export default function NetworkSelectorDialog({
  isOpen,
  currentNetworkId,
  currentDevServerUrl,
  onSelect,
  onClose,
}: NetworkSelectorDialogProps) {
  const [pendingId, setPendingId] = useState<NetworkId>(currentNetworkId);
  const [devUrl, setDevUrl] = useState(currentDevServerUrl);

  if (!isOpen) return null;

  const handleConfirm = () => {
    onSelect(pendingId, pendingId === 'dev' ? devUrl.trim() : undefined);
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-[#020408]/80 p-4 backdrop-blur-sm">
      <div className="w-full max-w-[520px] rounded-xl border border-white/10 bg-[rgba(15,23,42,0.9)] p-8 shadow-[0_8px_32px_rgba(0,0,0,0.8)] backdrop-blur-xl">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-5 flex h-16 w-16 items-center justify-center rounded-full border border-white/10 bg-[#2563eb]/10">
            <ShieldCheck className="h-8 w-8 text-[#b4c5ff]" />
          </div>
          <h1 className="text-[26px] font-semibold tracking-tight text-[#e2e2e2]">Choose Your Network</h1>
          <p className="mx-auto mt-2 max-w-[380px] text-[14px] leading-6 text-[#c3c6d7]">
            Select which KNIRVSERVER this controller should connect to.
          </p>
        </div>

        <div className="space-y-3">
          {NETWORK_OPTIONS.map((option) => {
            const selected = pendingId === option.id;
            const displayUrl = option.id === 'dev' ? devUrl : option.url;

            return (
              <button
                key={option.id}
                type="button"
                disabled={option.disabled}
                onClick={() => setPendingId(option.id)}
                className={`w-full rounded-xl border p-4 text-left transition-all ${
                  option.disabled
                    ? 'cursor-not-allowed border-white/5 bg-[#020408]/30 opacity-50'
                    : selected
                      ? 'border-[#2563eb]/60 bg-[#2563eb]/10'
                      : 'border-white/10 bg-[#020408]/40 hover:border-[#b4c5ff]/30 hover:bg-white/5'
                }`}
              >
                <div className="flex items-start gap-4">
                  <div
                    className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-lg border ${
                      selected && !option.disabled
                        ? 'border-[#2563eb]/40 bg-[#2563eb]/20 text-[#b4c5ff]'
                        : 'border-white/10 bg-white/5 text-[#8d90a0]'
                    }`}
                  >
                    {NETWORK_ICONS[option.id]}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-[16px] font-medium text-[#e2e2e2]">{option.name}</span>
                      {option.badge && (
                        <span className="rounded-full border border-white/10 bg-white/5 px-2 py-0.5 text-[10px] font-medium uppercase tracking-[0.2em] text-[#8d90a0]">
                          {option.badge}
                        </span>
                      )}
                    </div>
                    <p className="mt-1 text-[13px] leading-5 text-[#c3c6d7]">{option.description}</p>
                    {option.editable && selected ? (
                      <input
                        type="text"
                        value={devUrl}
                        onChange={(event) => setDevUrl(event.target.value)}
                        onClick={(event) => event.stopPropagation()}
                        placeholder={option.url}
                        className="mt-3 w-full rounded-lg border border-white/10 bg-[#020408]/60 px-3 py-2 font-mono text-[13px] text-[#b4c5ff] placeholder:text-[#64748b] focus:border-[#b4c5ff] focus:outline-none focus:ring-1 focus:ring-[#b4c5ff]"
                      />
                    ) : (
                      <p className="mt-1 truncate font-mono text-[11px] text-[#64748b]">{displayUrl}</p>
                    )}
                  </div>
                  {selected && !option.disabled && (
                    <Check className="mt-1 h-5 w-5 shrink-0 text-[#2563eb]" />
                  )}
                </div>
              </button>
            );
          })}
        </div>

        <div className="mt-8 flex gap-3">
          {onClose && (
            <button
              type="button"
              onClick={onClose}
              className="h-12 flex-1 rounded-lg border border-white/10 text-[15px] font-medium text-[#e2e2e2] transition-colors hover:bg-white/5"
            >
              Cancel
            </button>
          )}
          <button
            type="button"
            onClick={handleConfirm}
            className="h-12 flex-1 rounded-lg bg-[#2563eb] text-[15px] font-medium text-[#eeefff] transition-all duration-300 hover:shadow-[0_0_20px_rgba(37,99,235,0.35)]"
          >
            Connect
          </button>
        </div>
      </div>
    </div>
  );
}
