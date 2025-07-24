import { forwardRef } from 'react';
import {
  LayoutDashboard,
  Bot,
  Zap,
  Target,
  BarChart3,
  Settings,
  X,
  GitMerge,
  LogOut,
  Globe
} from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from './AuthContext';
import { useAppLogo } from '../hooks/useAssetPath';
import { ErrorNotificationBell } from './ErrorInferenceNotification';

type ActiveView = 'dashboard' | 'agents' | 'capabilities' | 'targets' | 'workflows' | 'analytics' | 'settings' | 'web-connections';

interface SidebarProps {
  activeView: ActiveView;
  setActiveView: (view: ActiveView) => void;
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
}

export const Sidebar = forwardRef<HTMLDivElement, SidebarProps>(
  ({ activeView, setActiveView, isOpen, setIsOpen }, ref) => {
    const navigate = useNavigate();
    const { user, logout } = useAuth();
  const logoPath = useAppLogo();

    const navigation = [
      { id: 'dashboard', name: 'Dashboard', icon: LayoutDashboard, path: '/dashboard' },
      { id: 'agents', name: 'NFT-Agents', icon: Bot, path: '/agents' },
      { id: 'capabilities', name: 'Capabilities', icon: Zap, path: '/capabilities' },
      { id: 'targets', name: 'Target Systems', icon: Target, path: '/targets' },
      { id: 'web-connections', name: 'Web Connections', icon: Globe, path: '/web-connections' },
      { id: 'workflows', name: 'Workflows', icon: GitMerge, path: '/workflows' },
      { id: 'analytics', name: 'Analytics', icon: BarChart3, path: '/analytics' },
      { id: 'settings', name: 'Settings', icon: Settings, path: '/settings' },
    ];

    const handleNavigation = (view: ActiveView, path: string) => {
      setActiveView(view);
      setIsOpen(false);
      navigate(path);
    };

    const handleLogout = async () => {
      await logout();
      navigate('/login');
    };

    return (
      <>
        {/* Mobile overlay */}
        {isOpen && (
          <div 
            className="fixed inset-0 bg-black bg-opacity-50 z-40 lg:hidden"
            onClick={() => setIsOpen(false)}
          />
        )}
        
        {/* Sidebar */}
        <div ref={ref} className={`
          fixed inset-y-0 left-0 z-50 w-64 bg-slate-800/95 backdrop-blur-xl border-r border-slate-700/50
          transform transition-transform duration-300 ease-in-out lg:translate-x-0
          ${isOpen ? 'translate-x-0' : '-translate-x-full'}
        `}>
          <div className="flex items-center justify-between p-6 border-b border-slate-700/50 main-header">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-white rounded-lg">
                <img
                  src={logoPath}
                  alt="Agentify logo"
                  className="w-6 h-6 object-contain"
                />
              </div>
              <div>
                <h1 className="text-xl font-bold text-white">Agentify</h1>
                <p className="text-sm text-slate-400">Agentic Engine</p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              {/* AI Error Engine Notification Bell */}
              <ErrorNotificationBell className="text-slate-400 hover:text-white error-notification-bell" />
              <button
                onClick={() => setIsOpen(false)}
                className="p-1 text-slate-400 hover:text-white lg:hidden"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
          </div>
          
          <nav className="mt-6 px-3">
            <ul className="space-y-2">
              {navigation.map((item) => {
                const Icon = item.icon;
                const isActive = activeView === item.id;
                
                return (
                  <li key={item.id}>
                    <button
                      onClick={() => handleNavigation(item.id as ActiveView, item.path)}
                      data-page={item.id}
                      className={`
                        w-full flex items-center space-x-3 px-3 py-2.5 rounded-lg transition-all duration-200
                        ${isActive
                          ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 text-white border border-purple-500/30'
                          : 'text-slate-300 hover:text-white hover:bg-slate-700/50'
                        }
                      `}
                    >
                      <Icon className="w-5 h-5" />
                      <span className="font-medium">{item.name}</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          </nav>
          
          <div className="absolute bottom-6 left-3 right-3">
            <div className="bg-slate-700/50 rounded-lg p-4 border border-slate-600/50">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="w-8 h-8 bg-gradient-to-r from-green-400 to-blue-500 rounded-full flex items-center justify-center">
                    <span className="text-xs font-bold text-white">{user?.username?.charAt(0).toUpperCase() || 'A'}</span>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-white">{user?.username || 'User'}</p>
                    <p className="text-xs text-slate-400 capitalize">{user?.role || 'User'}</p>
                  </div>
                </div>
                <button 
                  onClick={handleLogout}
                  className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
                  title="Logout"
                >
                  <LogOut className="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </>
    );
  }
);

Sidebar.displayName = 'Sidebar';