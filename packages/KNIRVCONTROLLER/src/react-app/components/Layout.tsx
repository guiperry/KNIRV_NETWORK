import { ReactNode, useEffect } from 'react';
import { useLocation, Link, useNavigate } from 'react-router';
import { ArrowLeft, Cpu, Zap, BadgeCheck, MessageSquare, Settings, Mic } from 'lucide-react';
import { EdgeColoring } from './EdgeColoring';
import { VoiceControl } from './VoiceControl';
import { useVoiceIntegration } from '../hooks/useVoiceIntegration';
import { registerBackHandler } from '../platform/backHandler';

interface LayoutProps { children: ReactNode; }

export default function Layout({ children }: LayoutProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const { isVoiceActive, cognitiveMode, edgeColor, edgeIntensity, handleVoiceCommand, toggleVoice } = useVoiceIntegration();

  const handleBack = () => {
    if (window.history.length > 1) { navigate(-1); return; }
    navigate('/dves');
  };

  useEffect(() => {
    const unregister = registerBackHandler(() => {
      if (window.history.length > 1 && location.pathname !== '/dves') {
        navigate(-1);
        return true;
      }
      return false;
    });
    return unregister;
  }, [navigate, location.pathname]);

  return (
    <div className="min-h-screen bg-[#0a0a0c] relative">
      <EdgeColoring color={edgeColor} intensity={edgeIntensity} />
      <div className="fixed inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-blue-600/10 via-transparent to-transparent"></div>
      <div className="fixed inset-0 bg-[linear-gradient(45deg,transparent_25%,rgba(68,68,68,.2)_50%,transparent_75%,transparent_100%)] bg-[length:20px_20px] opacity-10"></div>

      <header className="relative z-10 border-b border-white/5 backdrop-blur-xl bg-[#0a0a0c]/80" style={{ paddingTop: 'env(safe-area-inset-top)' }}>
        <div className="px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <button type="button" onClick={handleBack}
                className="flex h-9 w-9 items-center justify-center rounded-full border border-white/10 bg-white/5 text-slate-400 transition-colors hover:border-blue-500/40 hover:text-white hover:bg-white/10"
                aria-label="Go back">
                <ArrowLeft className="h-4 w-4" />
              </button>
              <div className="relative">
                <div className="w-8 h-8 bg-blue-600 rounded-sm transform rotate-45"></div>
                <div className="absolute inset-0 bg-blue-600 rounded-sm transform rotate-45 animate-pulse opacity-75"></div>
              </div>
              <div>
                <h1 className="text-xl font-extrabold tracking-tighter uppercase text-white">
                  KNIRV <span className="text-blue-500 font-light">CONTROLLER</span>
                </h1>
                <p className="text-[10px] uppercase tracking-widest text-slate-500 font-bold">Autonomous Gateway</p>
              </div>
            </div>

            <div className="flex-1 flex justify-center">
              {location.pathname.match(/^\/dves\/([^/]+)/) && (
                <div className="px-3 py-1 rounded-full bg-purple-500/20 border border-purple-500/30">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-purple-400 rounded-full animate-pulse"></div>
                    <span className="text-xs text-purple-400 font-medium font-mono uppercase tracking-wider">
                      DVE: {location.pathname.match(/^\/dves\/([^/]+)/)?.[1] ?? 'Unknown'}
                    </span>
                  </div>
                </div>
              )}
            </div>

            <div className="flex items-center space-x-2">
              <div className="px-2 py-1 rounded-full bg-amber-500/20 border border-amber-500/30">
                <div className="flex items-center space-x-1">
                  <Zap className="w-3 h-3 text-amber-400" />
                  <span className="text-xs text-amber-400 font-medium font-mono">18,247 NRN</span>
                </div>
              </div>
              <button onClick={() => toggleVoice(isVoiceActive)}
                className={`p-2 rounded-full transition-all ${
                  isVoiceActive
                    ? 'bg-blue-500/20 border border-blue-500/30 text-blue-400'
                    : 'bg-white/5 border border-white/10 text-slate-500 hover:text-slate-300 hover:bg-white/10'
                }`}>
                <Mic className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="relative z-10">{children}</main>

      <VoiceControl isActive={isVoiceActive} onVoiceCommand={handleVoiceCommand} onToggle={toggleVoice} cognitiveMode={cognitiveMode} />

      <nav className="fixed bottom-0 left-0 right-0 z-20 border-t border-white/5 backdrop-blur-xl bg-[#0a0a0c]/90" style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}>
        <div className="grid grid-cols-5 px-2 py-2">
          <NavItem to="/dves" icon={Cpu} label="DVEs" active={location.pathname.startsWith('/dves')} />
          <NavItem to="/vault" icon={Zap} label="Vault" active={location.pathname === '/vault'} />
          <NavItem to="/badges" icon={BadgeCheck} label="Badges" active={location.pathname === '/badges'} />
          <NavItem to="/cognitive" icon={MessageSquare} label="Chat" active={location.pathname === '/cognitive'} badge />
          <NavItem to="/settings" icon={Settings} label="Settings" active={location.pathname === '/settings'} />
        </div>
      </nav>
    </div>
  );
}

interface NavItemProps { to: string; icon: React.ComponentType<{ className?: string }>; label: string; active?: boolean; badge?: boolean; }

function NavItem({ to, icon: Icon, label, active = false, badge = false }: NavItemProps) {
  return (
    <Link to={to} className={`flex flex-col items-center space-y-1 py-2 px-1 rounded-lg transition-all relative ${
      active ? 'bg-blue-500/20 text-blue-400' : 'text-slate-500 hover:text-slate-300 hover:bg-white/5'
    }`}>
      <Icon className="w-5 h-5" />
      <span className="text-xs font-medium font-mono uppercase tracking-wider">{label}</span>
      {badge && <span className="absolute -top-0.5 -right-0.5 w-2.5 h-2.5 bg-blue-400 rounded-full animate-pulse"></span>}
    </Link>
  );
}
