import React, { useState } from 'react';
import { Outlet } from 'react-router-dom';
import { Sidebar } from '../Sidebar';
import RealTimeNotifications from '../RealTimeNotifications';
import SandboxDock from './SandboxDock';

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
  | 'settings';

/**
 * Consolidated application shell. Replaces the 9x-duplicated per-route layout
 * blocks in App.tsx: renders the Sidebar + main content region (hosting the
 * active child route via <Outlet/>) + the persistent SandboxDock below it.
 * The dock sits outside <main> so its RFB connection survives every
 * child-route swap.
 */
export const AppLayout: React.FC = () => {
  const [activeView, setActiveView] = useState<ActiveView>('dashboard');
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="flex min-h-screen flex-col bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900">
      <div className="flex flex-1">
        <Sidebar
          activeView={activeView}
          setActiveView={setActiveView}
          isOpen={sidebarOpen}
          setIsOpen={setSidebarOpen}
        />
        <main
          id="main-content"
          className="flex flex-1 flex-col lg:ml-64"
          role="main"
          aria-label="Main content"
        >
          <div className="lg:hidden">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="fixed left-4 top-4 z-50 rounded-lg bg-slate-800 p-2 text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              aria-label="Toggle navigation menu"
              aria-expanded={sidebarOpen}
            >
              <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
          </div>
          <div className="min-h-0 flex-1">
            <Outlet />
          </div>
        </main>
      </div>
      <SandboxDock />
      <div className="fixed right-4 top-4 z-50">
        <RealTimeNotifications />
      </div>
    </div>
  );
};

export default AppLayout;
