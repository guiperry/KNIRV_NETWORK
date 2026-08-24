import { forwardRef, useState } from 'react';
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
  Globe,
  Wallet,
  MessageSquare,
  Monitor,
  Brain,
  Briefcase,
  ShoppingBag,
  Layers,
  Server,
  ChevronDown,
  ChevronRight
} from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from './AuthContext';
import { useAppLogo } from '../hooks/useAssetPath';
import { ErrorNotificationBell } from './ErrorInferenceNotification';

type ActiveView = 'dashboard' | 'chat' | 'monitor' | 'models' | 'agents' | 'skills' | 'capabilities' | 'properties' | 'api' | 'targets' | 'workflows' | 'analytics' | 'settings' | 'web-connections' | 'wallet';

interface SidebarProps {
  activeView: ActiveView;
  setActiveView: (view: ActiveView) => void;
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
}

export const Sidebar = forwardRef<HTMLDivElement, SidebarProps>(
  ({ activeView, setActiveView, isOpen, setIsOpen }, ref) => {
    const navigate = useNavigate();
    const { user, logout, canAccessPage, canAccessSubPage } = useAuth();
    const logoPath = useAppLogo();
    const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());

    const navigation = [
      { id: 'dashboard', name: 'Dashboard', icon: LayoutDashboard, path: '/dashboard' },
      {
        id: 'chat',
        name: 'Chat',
        icon: MessageSquare,
        path: '/chat',
        subItems: [
          { id: 'chatchain', name: 'ChatChain', path: '/chat/chatchain' },
          { id: 'mychatbrain', name: 'MyChatBrain', path: '/chat/mychatbrain' }
        ]
      },
      {
        id: 'monitor',
        name: 'Monitor',
        icon: Monitor,
        path: '/monitor',
        subItems: [
          { id: 'network-monitor', name: 'Network Monitor', path: '/monitor/network-monitor' },
          { id: 'local-analytics', name: 'Local Analytics', path: '/monitor/local-analytics' },
          {
            id: 'network-explorers',
            name: 'Network Explorers',
            path: '/monitor/network-explorers',
            subItems: [
              { id: 'graph', name: 'Graph', path: '/monitor/network-explorers/graph' },
              { id: 'chain', name: 'Chain', path: '/monitor/network-explorers/chain' },
              { id: 'oracle', name: 'Oracle', path: '/monitor/network-explorers/oracle' },
              { id: 'router', name: 'Router', path: '/monitor/network-explorers/router' },
              { id: 'nexus', name: 'Nexus', path: '/monitor/network-explorers/nexus' }
            ]
          }
        ]
      },
      {
        id: 'models',
        name: 'Models',
        icon: Brain,
        path: '/models',
        subItems: [
          { id: 'codex-builder', name: 'Codex Builder', path: '/models/codex-builder' },
          { id: 'fallback-config', name: 'Optional Fallback API & HOM Config', path: '/models/fallback-config' },
          { id: 'dao-voting', name: 'DAO KNIRVCORTEX Shared Model voting', path: '/models/dao-voting' }
        ]
      },
      {
        id: 'agents',
        name: 'Agents',
        icon: Bot,
        path: '/agents',
        subItems: [
          { id: 'my-agents', name: 'My Agents', path: '/agents/my-agents' },
          { id: 'my-targets', name: 'My Targets', path: '/agents/my-targets' },
          { id: 'my-workflows', name: 'My Workflows', path: '/agents/my-workflows' }
        ]
      },
      {
        id: 'skills',
        name: 'Skills',
        icon: Briefcase,
        path: '/skills',
        subItems: [
          { id: 'skills-dex', name: 'Skills DEX', path: '/skills/skills-dex' }
        ]
      },
      {
        id: 'capabilities',
        name: 'Capabilities',
        icon: Zap,
        path: '/capabilities',
        subItems: [
          { id: 'capability-store', name: 'Capability Store', path: '/capabilities/capability-store' },
          { id: 'mcp-manager', name: 'MCP Manager', path: '/capabilities/mcp-manager' },
          { id: 'mcp-servers', name: 'MCP Servers', path: '/capabilities/mcp-servers' }
        ]
      },
      {
        id: 'properties',
        name: 'Properties',
        icon: Layers,
        path: '/properties',
        subItems: [
          { id: 'nft-ip-vault', name: 'NFT IP Vault', path: '/properties/nft-ip-vault' }
        ]
      },
      {
        id: 'api',
        name: 'API',
        icon: Server,
        path: '/api',
        subItems: [
          { id: 'personal-endpoints', name: 'Personal API Endpoints', path: '/api/personal-endpoints' }
        ]
      },
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

    const toggleExpanded = (itemId: string) => {
      const newExpanded = new Set(expandedItems);
      if (newExpanded.has(itemId)) {
        newExpanded.delete(itemId);
      } else {
        newExpanded.add(itemId);
      }
      setExpandedItems(newExpanded);
    };

    const renderSubItems = (parentItem: any, level: number = 1) => {
      if (!parentItem.subItems) return null;

      return (
        <div className={`ml-${level * 4} space-y-1`}>
          {parentItem.subItems.map((subItem: any) => {
            if (!canAccessSubPage(parentItem.id, subItem.id)) return null;

            return (
              <div key={subItem.id}>
                <button
                  onClick={() => handleNavigation(subItem.id as ActiveView, subItem.path)}
                  className="w-full flex items-center space-x-2 px-3 py-2 text-sm text-slate-400 hover:text-white hover:bg-slate-700/30 rounded-lg transition-all duration-200"
                >
                  <span className="w-2 h-2 bg-slate-500 rounded-full"></span>
                  <span>{subItem.name}</span>
                </button>
                {subItem.subItems && renderSubItems(subItem, level + 1)}
              </div>
            );
          })}
        </div>
      );
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
                  alt="KNIRV logo"
                  className="w-6 h-6 object-contain"
                />
              </div>
              <div>
                <h1 className="text-xl font-bold text-white">KNIRV Engine</h1>
                <p className="text-sm text-slate-400">Desktop Client</p>
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
                // Check if user can access this page
                if (!canAccessPage(item.id)) return null;

                const Icon = item.icon;
                const isActive = activeView === item.id;
                const isExpanded = expandedItems.has(item.id);
                const hasSubItems = item.subItems && item.subItems.length > 0;

                return (
                  <li key={item.id}>
                    <div className="space-y-1">
                      <button
                        onClick={() => {
                          if (hasSubItems) {
                            toggleExpanded(item.id);
                          } else {
                            handleNavigation(item.id as ActiveView, item.path);
                          }
                        }}
                        data-page={item.id}
                        className={`
                          w-full flex items-center justify-between px-3 py-2.5 rounded-lg transition-all duration-200
                          ${isActive
                            ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 text-white border border-purple-500/30'
                            : 'text-slate-300 hover:text-white hover:bg-slate-700/50'
                          }
                        `}
                      >
                        <div className="flex items-center space-x-3">
                          <Icon className="w-5 h-5" />
                          <span className="font-medium">{item.name}</span>
                        </div>
                        {hasSubItems && (
                          <div className="ml-auto">
                            {isExpanded ? (
                              <ChevronDown className="w-4 h-4" />
                            ) : (
                              <ChevronRight className="w-4 h-4" />
                            )}
                          </div>
                        )}
                      </button>

                      {hasSubItems && isExpanded && renderSubItems(item)}
                    </div>
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