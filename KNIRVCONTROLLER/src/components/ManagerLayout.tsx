import * as React from 'react';
import { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Cpu, Shield, Wallet, Mic, Brain } from 'lucide-react';
import { EdgeColoring } from './EdgeColoring';
import { VoiceControl } from './VoiceControl';
import { useVoiceIntegration } from '../hooks/useVoiceIntegration';

interface LayoutProps {
  children: ReactNode;
}

export default function Layout({ children }: LayoutProps) {
  const location = useLocation();
  const {
    isVoiceActive,
    voiceStatus,
    cognitiveMode,
    edgeColor,
    edgeIntensity,
    handleVoiceCommand,
    toggleVoice
  } = useVoiceIntegration();

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900 relative">
      {/* Edge Coloring for Voice Status */}
      <EdgeColoring color={edgeColor} intensity={edgeIntensity} />

      {/* Background Effects */}
      <div className="fixed inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-blue-400/20 via-transparent to-transparent"></div>
      <div className="fixed inset-0 bg-[linear-gradient(45deg,transparent_25%,rgba(68,68,68,.2)_50%,transparent_75%,transparent_100%)] bg-[length:20px_20px] opacity-20"></div>
      
      {/* Header */}
      <header className="relative z-10 border-b border-blue-500/20 backdrop-blur-xl bg-slate-900/50">
        <div className="px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="relative">
                <div className="w-8 h-8 bg-gradient-to-br from-blue-400 to-cyan-400 rounded-lg shadow-lg shadow-blue-500/25"></div>
                <div className="absolute inset-0 bg-gradient-to-br from-blue-400 to-cyan-400 rounded-lg animate-pulse opacity-75"></div>
              </div>
              <div>
                <h1 className="text-xl font-bold bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent">
                  KNIRV-AGENTIFIER
                </h1>
                <p className="text-xs text-slate-400">Autonomous Gateway</p>
              </div>
            </div>
            
            <div className="flex items-center space-x-2">
              {/* Voice Status Indicator */}
              {isVoiceActive && (
                <div className="px-2 py-1 rounded-full bg-blue-500/20 border border-blue-500/30">
                  <div className="flex items-center space-x-1">
                    <Mic className="w-3 h-3 text-blue-400" />
                    <span className="text-xs text-blue-400 font-medium">
                      {voiceStatus === 'listening' ? 'Listening' :
                       voiceStatus === 'processing' ? 'Processing' :
                       voiceStatus === 'speaking' ? 'Speaking' : 'Voice'}
                    </span>
                    {cognitiveMode && <Brain className="w-3 h-3 text-cyan-400" />}
                  </div>
                </div>
              )}

              <div className="px-2 py-1 rounded-full bg-green-500/20 border border-green-500/30">
                <div className="flex items-center space-x-1">
                  <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                  <span className="text-xs text-green-400 font-medium">D-TEN</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="relative z-10">
        {children}
      </main>

      {/* Voice Control */}
      <VoiceControl
        isActive={isVoiceActive}
        onVoiceCommand={handleVoiceCommand}
        onToggle={toggleVoice}
        cognitiveMode={cognitiveMode}
      />

      {/* Bottom Navigation */}
      <nav className="fixed bottom-0 left-0 right-0 z-20 border-t border-blue-500/20 backdrop-blur-xl bg-slate-900/80">
        <div className="grid grid-cols-4 px-2 py-2">
          <NavItem to="/manager" icon={Cpu} label="Agents" active={location.pathname === '/manager'} />
          <NavItem to="/manager/skills" icon={Brain} label="Skills" active={location.pathname === '/manager/skills'} />
          <NavItem to="/manager/udc" icon={Shield} label="UDC" active={location.pathname === '/manager/udc'} />
          <NavItem to="/manager/wallet" icon={Wallet} label="Wallet" active={location.pathname === '/manager/wallet'} />
        </div>
      </nav>
    </div>
  );
}

interface NavItemProps {
  to: string;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  active?: boolean;
}

function NavItem({ to, icon: Icon, label, active = false }: NavItemProps) {
  return (
    <Link to={to} className={`flex flex-col items-center space-y-1 py-2 px-1 rounded-lg transition-all ${
      active
        ? 'bg-blue-500/20 text-blue-400'
        : 'text-slate-400 hover:text-slate-300 hover:bg-slate-800/50'
    }`}>
      <Icon className="w-5 h-5" />
      <span className="text-xs font-medium">{label}</span>
    </Link>
  );
}
