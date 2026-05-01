import { useState, useRef, useEffect } from 'react';
import { ArrowUp, Mic, MicOff, Loader2 } from 'lucide-react';

type VoiceStatus = 'idle' | 'listening' | 'processing' | 'speaking' | 'error';

interface VoiceChatBarProps {
  onSendMessage: (text: string) => void;
  disabled?: boolean;
  placeholder?: string;
  voiceStatus?: VoiceStatus;
}

const STATUS_COLORS: Record<VoiceStatus, string> = {
  idle: '#10B981',
  listening: '#14B8A6',
  processing: '#3B82F6',
  speaking: '#8B5CF6',
  error: '#EF4444',
};

const STATUS_LABELS: Record<VoiceStatus, string> = {
  idle: 'Idle',
  listening: 'Listening...',
  processing: 'Processing...',
  speaking: 'Speaking...',
  error: 'Voice Error',
};

export default function VoiceChatBar({
  onSendMessage,
  disabled = false,
  placeholder = 'Type or speak a command...',
  voiceStatus = 'idle',
}: VoiceChatBarProps) {
  const [text, setText] = useState('');
  const [micActive, setMicActive] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const isMicOn = micActive || voiceStatus === 'listening' || voiceStatus === 'processing';

  useEffect(() => {
    if (!disabled && inputRef.current) {
      inputRef.current.focus();
    }
  }, [disabled]);

  const handleSend = () => {
    const trimmed = text.trim();
    if (!trimmed || disabled) return;
    onSendMessage(trimmed);
    setText('');
    inputRef.current?.focus();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const toggleMic = () => {
    if (disabled) return;
    setMicActive((prev) => !prev);
  };

  const borderColor = disabled ? '#334155' : STATUS_COLORS[voiceStatus];

  return (
    <div
      className="bg-slate-900/80 backdrop-blur-xl border-t border-white/5 rounded-t-2xl px-4 py-3 transition-all duration-300"
      style={{
        borderLeft: `2px solid ${borderColor}`,
        borderRight: '1px solid rgba(255,255,255,0.05)',
        borderTop: '1px solid rgba(255,255,255,0.05)',
        borderBottom: 'none',
        opacity: disabled ? 0.5 : 1,
      }}
    >
      <div className="flex items-center gap-2">
        {/* Microphone Toggle */}
        <button
          type="button"
          onClick={toggleMic}
          disabled={disabled}
          className={`relative flex items-center justify-center w-10 h-10 rounded-xl transition-all duration-200 ${
            disabled
              ? 'text-slate-600 cursor-not-allowed'
              : isMicOn
                ? 'bg-teal-500/20 text-teal-400 hover:bg-teal-500/30'
                : 'text-slate-400 hover:text-slate-300 hover:bg-white/5'
          }`}
          title={isMicOn ? 'Deactivate microphone' : 'Activate microphone'}
        >
          {isMicOn ? (
            <Mic className="w-4 h-4" />
          ) : (
            <MicOff className="w-4 h-4" />
          )}

          {/* Listening indicator dots */}
          {isMicOn && (
            <span className="absolute -top-0.5 -right-0.5 flex h-2.5 w-2.5">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-teal-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-teal-500"></span>
            </span>
          )}
        </button>

        {/* Text Input */}
        <div className="flex-1 relative">
          <input
            ref={inputRef}
            type="text"
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={voiceStatus === 'listening' ? 'Listening...' : placeholder}
            disabled={disabled}
            autoFocus
            className="w-full bg-slate-950 text-white placeholder-slate-500 rounded-xl px-4 py-2.5 pr-10 text-sm outline-none border border-white/5 focus:border-white/10 transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          />

          {/* Voice status indicator text */}
          {isMicOn && voiceStatus !== 'idle' && (
            <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none">
              {voiceStatus === 'processing' ? (
                <Loader2 className="w-4 h-4 text-blue-400 animate-spin" />
              ) : voiceStatus === 'listening' ? (
                <span className="flex gap-0.5">
                  <span className="w-0.5 h-3 bg-teal-400 rounded-full animate-pulse" />
                  <span className="w-0.5 h-2 bg-teal-400 rounded-full animate-pulse" style={{ animationDelay: '0.15s' }} />
                  <span className="w-0.5 h-3 bg-teal-400 rounded-full animate-pulse" style={{ animationDelay: '0.3s' }} />
                </span>
              ) : null}
            </div>
          )}
        </div>

        {/* Send Button */}
        <button
          type="button"
          onClick={handleSend}
          disabled={disabled || !text.trim()}
          className={`flex items-center justify-center w-10 h-10 rounded-xl transition-all duration-200 ${
            disabled || !text.trim()
              ? 'bg-slate-800 text-slate-600 cursor-not-allowed'
              : 'bg-blue-600 text-white hover:bg-blue-500 active:bg-blue-700 shadow-lg shadow-blue-600/20'
          }`}
          title="Send message"
        >
          <ArrowUp className="w-4 h-4" />
        </button>
      </div>

      {/* Voice status label bar */}
      {voiceStatus !== 'idle' && !disabled && (
        <div className="flex items-center gap-1.5 mt-2">
          <div
            className="w-1.5 h-1.5 rounded-full"
            style={{ backgroundColor: STATUS_COLORS[voiceStatus] }}
          />
          <span
            className="text-[11px] font-medium uppercase tracking-wider"
            style={{ color: STATUS_COLORS[voiceStatus] }}
          >
            {STATUS_LABELS[voiceStatus]}
          </span>
        </div>
      )}
    </div>
  );
}
