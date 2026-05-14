import React, { useState, useEffect } from 'react';
import { Terminal, Cpu, Zap, Brain, Shield, Play, AlertTriangle } from 'lucide-react';

interface GameMenuProps {
  onStart: () => void;
  usingMockLLM: boolean;
}

export const GameMenu: React.FC<GameMenuProps> = ({ onStart, usingMockLLM }) => {
  const [loadingStep, setLoadingStep] = useState(0);
  const [showMenu, setShowMenu] = useState(false);
  const [terminalLines, setTerminalLines] = useState<string[]>([]);
  
  const loadingMessages = [
    "INITIALIZING COGNITIVE SHELL...",
    "LOADING NEURAL WEIGHTS (LORA)...",
    "ESTABLISHING KNIRVGRAPH CONNECTION...",
    "BOOTING WASM ORCHESTRATOR...",
    "SYNCHRONIZING WITH XION TESTNET...",
    "SCANNING FOR ERROR NODES...",
    "SYSTEM READY."
  ];

  useEffect(() => {
    if (loadingStep < loadingMessages.length) {
      const timer = setTimeout(() => {
        setTerminalLines(prev => [...prev, loadingMessages[loadingStep]]);
        setLoadingStep(prev => prev + 1);
      }, 400 + Math.random() * 600);
      return () => clearTimeout(timer);
    } else {
      const timer = setTimeout(() => {
        setShowMenu(true);
      }, 800);
      return () => clearTimeout(timer);
    }
  }, [loadingStep]);

  return (
    <div className="fixed inset-0 flex items-center justify-center z-[100001] bg-[#050505] overflow-hidden font-mono pointer-events-auto">
      {/* Animated Background Grid */}
      <div className="absolute inset-0 opacity-20">
        <svg width="100%" height="100%" xmlns="http://www.w3.org/2000/svg">
          <defs>
            <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
              <path d="M 40 0 L 0 0 0 40" fill="none" stroke="#00f2ff" strokeWidth="0.5" />
            </pattern>
            <radialGradient id="fade" cx="50%" cy="50%" r="50%" fx="50%" fy="50%">
              <stop offset="0%" stopColor="rgba(0, 242, 255, 0.2)" />
              <stop offset="100%" stopColor="rgba(5, 5, 5, 0)" />
            </radialGradient>
          </defs>
          <rect width="100%" height="100%" fill="url(#grid)" />
          <rect width="100%" height="100%" fill="url(#fade)" />
        </svg>
      </div>

      {/* Animated Pulses */}
      <div className="absolute inset-0 pointer-events-none">
        {[...Array(5)].map((_, i) => (
          <div
            key={i}
            className="absolute bg-cyan-500/30 rounded-full animate-ping"
            style={{
              left: `${Math.random() * 100}%`,
              top: `${Math.random() * 100}%`,
              width: `${Math.random() * 100 + 50}px`,
              height: `${Math.random() * 100 + 50}px`,
              animationDuration: `${Math.random() * 3 + 2}s`,
              animationDelay: `${Math.random() * 5}s`
            }}
          />
        ))}
      </div>

      {!showMenu ? (
        /* Boot Sequence */
        <div className="relative z-10 w-full max-w-lg p-8 bg-black/60 border border-cyan-900/50 rounded shadow-[0_0_30px_rgba(0,242,255,0.1)]">
          <div className="flex items-center gap-3 mb-6 border-b border-cyan-900/50 pb-4">
            <Terminal className="text-cyan-400 w-6 h-6 animate-pulse" />
            <h2 className="text-cyan-400 font-bold tracking-widest text-lg">ERGO_BOOT_v1.0.4</h2>
          </div>
          
          <div className="space-y-2 h-48 overflow-hidden">
            {terminalLines.map((line, i) => (
              <div key={i} className="flex gap-3 text-sm">
                <span className="text-cyan-800">[{i.toString().padStart(2, '0')}]</span>
                <span className={i === terminalLines.length - 1 ? "text-cyan-400" : "text-cyan-700"}>
                  {line}
                </span>
                {i === terminalLines.length - 1 && loadingStep < loadingMessages.length && (
                  <span className="w-2 h-4 bg-cyan-400 animate-bounce" />
                )}
              </div>
            ))}
          </div>

          <div className="mt-8">
            <div className="w-full bg-cyan-950 h-1 rounded-full overflow-hidden">
              <div 
                className="bg-cyan-400 h-full transition-all duration-300 shadow-[0_0_10px_rgba(0,242,255,0.8)]"
                style={{ width: `${(loadingStep / loadingMessages.length) * 100}%` }}
              />
            </div>
          </div>
        </div>
      ) : (
        /* Main Menu */
        <div className="relative z-10 flex flex-col items-center animate-menu-fade-in">
          {/* Logo Section */}
          <div className="relative mb-12 group">
            <div className="absolute -inset-4 bg-cyan-500/20 blur-xl rounded-full group-hover:bg-cyan-500/30 transition-all duration-500 animate-pulse" />
            <svg width="320" height="100" viewBox="0 0 320 100" className="relative">
              <defs>
                <linearGradient id="logo-grad" x1="0%" y1="0%" x2="100%" y2="0%">
                  <stop offset="0%" stopColor="#00f2ff" />
                  <stop offset="100%" stopColor="#bc00ff" />
                </linearGradient>
              </defs>
              <text 
                x="50%" 
                y="70%" 
                textAnchor="middle" 
                className="text-7xl font-black tracking-[0.2em] fill-white stroke-[url(#logo-grad)] stroke-2 animate-logo-dash"
                style={{ 
                  strokeDasharray: '400', 
                  strokeDashoffset: '400',
                  paintOrder: 'stroke',
                  filter: 'drop-shadow(0 0 8px rgba(0, 242, 255, 0.5))'
                }}
              >
                ERGO
              </text>
              <style>{`
                @keyframes dash {
                  to { strokeDashoffset: 0; }
                }
                @keyframes menuFadeIn {
                  from { opacity: 0; transform: scale(0.95); }
                  to { opacity: 1; transform: scale(1); }
                }
                .animate-menu-fade-in {
                  animation: menuFadeIn 0.8s ease-out forwards;
                }
                .animate-logo-dash {
                  animation: dash 4s linear infinite alternate;
                }
              `}</style>
            </svg>
            <div className="text-cyan-500/80 tracking-[0.5em] text-xs text-center mt-2 font-bold uppercase">
              Neural Intelligence Arena
            </div>
          </div>

          {/* Menu Card */}
          <div className="bg-black/40 backdrop-blur-md border border-white/10 p-8 rounded-2xl w-full max-w-sm shadow-2xl relative overflow-hidden group">
            <div className="absolute inset-0 bg-gradient-to-b from-cyan-500/5 to-purple-500/5 pointer-events-none" />
            
            <p className="text-gray-400 text-center text-sm mb-8 leading-relaxed">
              Transform AI errors into collective knowledge.
              <span className="block mt-2 text-xs text-gray-500 italic">
                LLM agents compete to fix bugs — you sculpt the arena.
              </span>
            </p>

            {usingMockLLM && (
              <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-xl p-3 mb-6 flex items-start gap-3">
                <AlertTriangle className="text-yellow-500 w-4 h-4 mt-0.5 flex-shrink-0" />
                <p className="text-[10px] text-yellow-500/80 leading-tight">
                  MOCK MODE ACTIVE. No API keys detected. Configure VITE_OPENAI_API_KEY for live LLM gameplay.
                </p>
              </div>
            )}

            <div className="space-y-4">
              <button
                onClick={onStart}
                className="w-full group/btn relative overflow-hidden bg-cyan-600 hover:bg-cyan-500 text-white py-4 rounded-xl font-bold tracking-widest transition-all duration-300 shadow-[0_0_20px_rgba(8,145,178,0.3)] hover:shadow-[0_0_30px_rgba(6,182,212,0.5)] active:scale-95"
              >
                <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover/btn:translate-x-full transition-transform duration-700" />
                <div className="flex items-center justify-center gap-2">
                  <Play className="w-5 h-5 fill-current" />
                  INITIATE TOURNAMENT
                </div>
              </button>
              
              <div className="grid grid-cols-3 gap-2">
                <div className="flex flex-col items-center p-2 rounded-lg bg-white/5 border border-white/5">
                  <Brain className="text-purple-400 w-4 h-4 mb-1" />
                  <span className="text-[8px] text-gray-500 uppercase">Neural</span>
                </div>
                <div className="flex flex-col items-center p-2 rounded-lg bg-white/5 border border-white/5">
                  <Cpu className="text-cyan-400 w-4 h-4 mb-1" />
                  <span className="text-[8px] text-gray-500 uppercase">WASM</span>
                </div>
                <div className="flex flex-col items-center p-2 rounded-lg bg-white/5 border border-white/5">
                  <Shield className="text-green-400 w-4 h-4 mb-1" />
                  <span className="text-[8px] text-gray-500 uppercase">XION</span>
                </div>
              </div>
            </div>
          </div>

          <div className="mt-12 text-[10px] text-gray-600 flex items-center gap-4">
            <span className="flex items-center gap-1"><div className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" /> NETWORK ONLINE</span>
            <span>VER: 1.0.4-MVP</span>
            <span>© 2026 KNIRV_NETWORK</span>
          </div>
        </div>
      )}
    </div>
  );
};
