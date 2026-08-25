import { forwardRef, useState } from 'react';
import {
  LayoutDashboard,
  Radio,
  Cpu,
  Binary,
  Bug,
  ScanSearch,
  Waves,
  KeyRound,
  Box,
  Settings,
  X,
  LogOut,
  ChevronDown,
  ChevronRight
} from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from './AuthContext';
import { useAppLogo } from '../hooks/useAssetPath';
import { ErrorNotificationBell } from './ErrorInferenceNotification';
import type { LucideIcon } from 'lucide-react';

type ActiveView =
  | 'dashboard'
  | 'proxy'
  | 'instrumentation'
  | 'reversing'
  | 'fuzzing'
  | 'static-analysis'
  | 'packet-capture'
  | 'auth-audit'
  | 'sandbox'
  | 'settings'
  | 'frida' | 'proxychains-ng' | 'bpftrace'
  | 'ghidra' | 'cutter' | 'ilspy' | 'jadx'
  | 'libafl' | 'aflplusplus'
  | 'semgrep' | 'tree-sitter' | 'trufflehog'
  | 'wireshark' | 'zeek'
  | 'jwt-tool' | 'saml-raider'
  | 'bubblewrap' | 'novnc';

interface SidebarProps {
  activeView: ActiveView;
  setActiveView: (view: ActiveView) => void;
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
}

interface NavigationItem {
  id: ActiveView;
  name: string;
  path: string;
  icon?: LucideIcon;
  subItems?: NavigationItem[];
}

export const Sidebar = forwardRef<HTMLDivElement, SidebarProps>(
  ({ activeView, setActiveView, isOpen, setIsOpen }, ref) => {
    const navigate = useNavigate();
    const { user, logout, canAccessPage, canAccessSubPage } = useAuth();
    const logoPath = useAppLogo();
    const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());

    const navigation: NavigationItem[] = [
      { id: 'dashboard', name: 'Dashboard', icon: LayoutDashboard, path: '/dashboard' },
      { id: 'sandbox', name: 'Sandbox', icon: Box, path: '/sandbox', subItems: [ { id: 'bubblewrap', name: 'Bubblewrap', path: '/sandbox/bubblewrap' }, { id: 'novnc', name: 'noVNC', path: '/sandbox/novnc' } ] },
      { id: 'proxy', name: 'Proxy', icon: Radio, path: '/proxy' },
      {
        id: 'instrumentation',
        name: 'Instrumentation',
        icon: Cpu,
        path: '/instrumentation',
        subItems: [
          { id: 'frida', name: 'Frida', path: '/instrumentation/frida' },
          { id: 'proxychains-ng', name: 'proxychains-ng', path: '/instrumentation/proxychains-ng' },
          { id: 'bpftrace', name: 'bpftrace', path: '/instrumentation/bpftrace' }
        ]
      },
      {
        id: 'reversing',
        name: 'Reversing',
        icon: Binary,
        path: '/reversing',
        subItems: [
          { id: 'ghidra', name: 'Ghidra', path: '/reversing/ghidra' },
          { id: 'cutter', name: 'Cutter', path: '/reversing/cutter' },
          { id: 'ilspy', name: 'ILSpy', path: '/reversing/ilspy' },
          { id: 'jadx', name: 'JADX', path: '/reversing/jadx' }
        ]
      },
      {
        id: 'fuzzing',
        name: 'Fuzzing',
        icon: Bug,
        path: '/fuzzing',
        subItems: [
          { id: 'libafl', name: 'LibAFL', path: '/fuzzing/libafl' },
          { id: 'aflplusplus', name: 'AFL++', path: '/fuzzing/aflplusplus' }
        ]
      },
      {
        id: 'static-analysis',
        name: 'Static Analysis',
        icon: ScanSearch,
        path: '/static-analysis',
        subItems: [
          { id: 'semgrep', name: 'Semgrep', path: '/static-analysis/semgrep' },
          { id: 'tree-sitter', name: 'Tree-sitter', path: '/static-analysis/tree-sitter' },
          { id: 'trufflehog', name: 'TruffleHog', path: '/static-analysis/trufflehog' }
        ]
      },
      {
        id: 'packet-capture',
        name: 'Packet Capture',
        icon: Waves,
        path: '/packet-capture',
        subItems: [
          { id: 'wireshark', name: 'Wireshark (TShark)', path: '/packet-capture/wireshark' },
          { id: 'zeek', name: 'Zeek', path: '/packet-capture/zeek' }
        ]
      },
      {
        id: 'auth-audit',
        name: 'Auth Audit',
        icon: KeyRound,
        path: '/auth-audit',
        subItems: [
          { id: 'jwt-tool', name: 'jwt_tool', path: '/auth-audit/jwt-tool' },
          { id: 'saml-raider', name: 'SAML Raider', path: '/auth-audit/saml-raider' }
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

    const renderSubItems = (parentItem: NavigationItem, level: number = 1) => {
      if (!parentItem.subItems) return null;

      return (
        <div className={`ml-${level * 4} space-y-1`}>
          {parentItem.subItems.map((subItem) => {
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
                          {Icon && <Icon className="w-5 h-5" />}
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
