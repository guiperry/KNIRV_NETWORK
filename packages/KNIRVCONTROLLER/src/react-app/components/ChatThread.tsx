import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Copy, Check } from 'lucide-react';
import type { AgentMessage } from '@/shared/types';

interface ChatThreadProps {
  messages: AgentMessage[];
  isStreaming: boolean;
  onCopyMessage: (id: string, content: string) => void;
}

function formatTimestamp(ts: string): string {
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '--:--';
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false });
}

function parseCodeBlocks(text: string): ReactNode[] {
  const parts = text.split(/(```[\s\S]*?```)/g);
  return parts.map((part, i) => {
    if (part.startsWith('```') && part.endsWith('```')) {
      const inner = part.slice(3, -3);
      const firstLine = inner.indexOf('\n');
      const lang = firstLine > 0 ? inner.slice(0, firstLine).trim() : '';
      const code = firstLine > 0 ? inner.slice(firstLine + 1) : inner;
      return (
        <pre
          key={i}
          className="mt-2 mb-2 p-3 rounded-xl bg-[#0d0d12] border border-white/5 overflow-x-auto font-mono text-sm leading-relaxed text-slate-300"
        >
          {lang && (
            <div className="text-[10px] text-slate-500 uppercase tracking-widest mb-1 font-semibold">
              {lang}
            </div>
          )}
          <code>{code}</code>
        </pre>
      );
    }
    return <span key={i} className="whitespace-pre-wrap">{part}</span>;
  });
}

function roleStyles(role: AgentMessage['role']) {
  switch (role) {
    case 'user':
      return {
        align: 'justify-end',
        bubble: 'bg-gradient-to-br from-blue-600/40 to-blue-700/30 border-blue-500/30 text-blue-100',
        label: 'User',
        labelColor: 'text-blue-400',
      };
    case 'agent':
      return {
        align: 'justify-start',
        bubble: 'bg-gradient-to-br from-purple-600/40 to-purple-700/30 border-purple-500/30 text-purple-100',
        label: 'Agent',
        labelColor: 'text-purple-400',
      };
    case 'cognitive':
      return {
        align: 'justify-start',
        bubble: 'bg-gradient-to-br from-teal-600/40 to-teal-700/30 border-teal-500/30 text-teal-100',
        label: 'Cognitive',
        labelColor: 'text-teal-400',
      };
    case 'system':
    default:
      return {
        align: 'justify-center',
        bubble: 'bg-transparent border-none text-slate-400 italic text-center text-xs',
        label: '',
        labelColor: '',
      };
  }
}

function CopyButton({ id, content, onCopy }: { id: string; content: string; onCopy: (id: string, content: string) => void }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    onCopy(id, content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <button
      onClick={handleCopy}
      className="p-1.5 rounded-lg opacity-0 group-hover:opacity-100 hover:bg-white/10 transition-all duration-200 text-slate-400 hover:text-slate-200"
      title="Copy message"
    >
      {copied ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
    </button>
  );
}

function StreamingIndicator() {
  return (
    <div className="flex justify-start px-4">
      <div className="glass-panel px-4 py-3 rounded-2xl border border-purple-500/20">
        <div className="flex items-center space-x-1.5">
          <span className="w-2 h-2 bg-purple-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
          <span className="w-2 h-2 bg-purple-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
          <span className="w-2 h-2 bg-purple-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
        </div>
      </div>
    </div>
  );
}

export default function ChatThread({ messages, isStreaming, onCopyMessage }: ChatThreadProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  return (
    <div className="flex flex-col space-y-3 px-4 py-4">
      {messages.length === 0 && !isStreaming && (
        <div className="flex-1 flex items-center justify-center py-12">
          <p className="text-slate-500 text-sm font-mono italic">No messages yet. Start a conversation.</p>
        </div>
      )}

      {messages.map((msg) => {
        const styles = roleStyles(msg.role);

        // System messages render as centered italic text, no bubble
        if (msg.role === 'system') {
          return (
            <div key={msg.id} className="flex justify-center px-4 py-1">
              <div className="flex items-center space-x-2">
                <div className="h-px w-8 bg-white/5" />
                <p className="text-xs text-slate-500 italic font-mono">{msg.content}</p>
                <div className="h-px w-8 bg-white/5" />
              </div>
            </div>
          );
        }

        return (
          <div key={msg.id} className={`flex ${styles.align} group`}>
            <div className={`max-w-[80%] lg:max-w-[70%]`}>
              {/* Role Label */}
              {styles.label && (
                <div className={`text-[10px] uppercase tracking-widest font-semibold mb-1 px-1 ${styles.labelColor}`}>
                  {styles.label}
                </div>
              )}

              {/* Bubble */}
              <div className={`relative glass-panel p-3 rounded-2xl border ${styles.bubble}`}>
                <div className="text-sm leading-relaxed">
                  {parseCodeBlocks(msg.content)}
                </div>

                {/* Footer: Timestamp + Copy */}
                <div className="flex items-center justify-between mt-2 pt-1 border-t border-white/5">
                  <span className="text-[10px] text-slate-500 font-mono">
                    {formatTimestamp(msg.timestamp)}
                  </span>
                  <CopyButton id={msg.id} content={msg.content} onCopy={onCopyMessage} />
                </div>
              </div>
            </div>
          </div>
        );
      })}

      {isStreaming && <StreamingIndicator />}

      <div ref={bottomRef} />
    </div>
  );
}
