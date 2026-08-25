import React, { lazy, Suspense, useState, useEffect } from 'react';
import { BrowserRouter, HashRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Dashboard } from './components/Dashboard';
import { AuthProvider } from './components/AuthContext';
import LoginPage from './components/LoginPage';
import ProtectedRoute from './components/ProtectedRoute';
import UnauthorizedPage from './components/UnauthorizedPage';
import { initializeWebSocket, cleanupWebSocket } from './utils/websocket';
import { initializeAccessibility } from './utils/accessibility';
import { enableIntelligentErrorAnalysis, handleApiError } from './utils/errorHandler';
import { OnboardingProvider } from './components/onboarding/OnboardingProvider';
import OnboardingSequence from './components/onboarding/OnboardingSequence';
import LoadingScreen from './components/LoadingScreen';
import { SandboxProvider } from './components/SandboxContext';
import { AppLayout } from './components/layout/AppLayout';
import { RequireSandbox } from './components/RequireSandbox';
import { getApiBaseUrl } from './utils/apiBase';
import './components/onboarding/onboarding.css';

// Tooling is visited on demand. Splitting these route modules keeps the
// dashboard responsive and avoids downloading noVNC with every application load.
const Proxy = lazy(() => import('./components/tools/Proxy').then((module) => ({ default: module.Proxy })));
const Instrumentation = lazy(() => import('./components/tools/Instrumentation').then((module) => ({ default: module.Instrumentation })));
const Reversing = lazy(() => import('./components/tools/Reversing').then((module) => ({ default: module.Reversing })));
const Fuzzing = lazy(() => import('./components/tools/Fuzzing').then((module) => ({ default: module.Fuzzing })));
const StaticAnalysis = lazy(() => import('./components/tools/StaticAnalysis').then((module) => ({ default: module.StaticAnalysis })));
const PacketCapture = lazy(() => import('./components/tools/PacketCapture').then((module) => ({ default: module.PacketCapture })));
const AuthAudit = lazy(() => import('./components/tools/AuthAudit').then((module) => ({ default: module.AuthAudit })));
const Sandbox = lazy(() => import('./components/tools/Sandbox').then((module) => ({ default: module.Sandbox })));
const Settings = lazy(() => import('./components/Settings'));

// Detect if we're running in Electron or web browser
const isElectron = () => {
  // Check for file protocol (Electron loads files locally)
  if (window.location.protocol === 'file:') return true;

  // Check for Electron process
  try {
    const electronWindow = window as { process?: { type?: string } };
    if (electronWindow.process && electronWindow.process.type === 'renderer') return true;
  } catch {
    // Ignore errors accessing process
  }

  // Check user agent
  if (typeof navigator !== 'undefined' && navigator.userAgent.toLowerCase().indexOf('electron') > -1) return true;

  return false;
};

// Dynamic Router component that chooses the appropriate router based on environment
const DynamicRouter: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  if (isElectron()) {
    return <HashRouter>{children}</HashRouter>;
  } else {
    return <BrowserRouter>{children}</BrowserRouter>;
  }
};

function App() {
  const [isLoading, setIsLoading] = useState(true);
  const [loadingMessage, setLoadingMessage] = useState('Starting KNIRVENGINE...');
  const [loadingProgress, setLoadingProgress] = useState(0);

  // Application initialization with loading screen
  useEffect(() => {
    let cancelled = false;

    const initializeApp = async () => {
      try {
        setLoadingMessage('Initializing error handling...');
        setLoadingProgress(25);

        // Initialize error handling
        enableIntelligentErrorAnalysis(true);

        setLoadingMessage('Setting up accessibility...');
        setLoadingProgress(20);

        // Initialize accessibility features
        initializeAccessibility();

        setLoadingMessage('Connecting to backend...');
        setLoadingProgress(50);

        // Keep first paint responsive. A backend that is starting or busy must
        // not hold the application on its splash screen for multiple retries.
        const apiBaseUrl = getApiBaseUrl();
        let backendReady = false;
        try {
          const healthResponse = await fetch(`${apiBaseUrl}/api/v1/health`, {
            signal: AbortSignal.timeout(1500),
          });
          backendReady = healthResponse.ok;
          if (!backendReady) {
            console.warn(`Backend health check returned ${healthResponse.status}; continuing while it starts`);
          }
        } catch (error) {
          console.warn('Backend is not ready yet; continuing with limited functionality:', error);
        }

        if (cancelled) return;

        setLoadingMessage(backendReady ? 'Backend connected' : 'Starting in limited mode');
        setLoadingProgress(100);
        setIsLoading(false);

        // Realtime updates are optional during first paint. Reconnect attempts
        // continue in the background without blocking navigation.
        if (backendReady) {
          void initializeWebSocket().catch(error => {
            console.error('Failed to initialize WebSocket:', error);
            handleApiError(error, {
              operation: 'websocket_initialization',
              component: 'App',
              timestamp: new Date().toISOString(),
              context: 'Failed to initialize WebSocket connection',
            });
          });
        }

      } catch (error) {
        console.error('App initialization error:', error);
        handleApiError(error, {
          operation: 'app_initialization',
          component: 'App',
          timestamp: new Date().toISOString(),
          context: 'Application initialization failed'
        });

        // Still allow app to load even if some initialization fails.
        if (!cancelled) {
          setLoadingMessage('Starting with limited functionality...');
          setIsLoading(false);
        }
      }
    };

    initializeApp();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    // Global error handlers and cleanup (separate from initialization)

    // Set up global error handlers with auto-reload for critical errors
    const handleGlobalError = (event: ErrorEvent) => {
      console.error('Global error caught:', event.error, event);

      handleApiError(event.error || new Error(event.message), {
        operation: 'global_error',
        component: 'App',
        filename: event.filename,
        lineno: event.lineno,
        colno: event.colno,
        timestamp: new Date().toISOString(),
        context: 'Unhandled JavaScript error',
        autoReload: true
      });

      // Auto-reload for critical errors like DOMException
      if (event.error?.name === 'DOMException' ||
          event.message?.includes('querySelector') ||
          event.message?.includes('not a valid selector')) {
        console.log('Critical DOM error detected, triggering auto-reload...');
        setTimeout(() => {
          window.location.reload();
        }, 2000);
      }
    };

    const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
      console.error('Unhandled promise rejection:', event.reason);

      handleApiError(event.reason, {
        operation: 'unhandled_promise_rejection',
        component: 'App',
        timestamp: new Date().toISOString(),
        context: 'Unhandled promise rejection',
        autoReload: true
      });

      // Auto-reload for critical promise rejections
      if (event.reason?.name === 'DOMException' ||
          event.reason?.message?.includes('querySelector')) {
        console.log('Critical promise rejection detected, triggering auto-reload...');
        setTimeout(() => {
          window.location.reload();
        }, 2000);
      }
    };

    window.addEventListener('error', handleGlobalError);
    window.addEventListener('unhandledrejection', handleUnhandledRejection);

    const handleUnload = () => {
      // Clean up WebSocket connection
      cleanupWebSocket();

      // navigator.sendBeacon is designed for this use case.
      // It sends a POST request with the specified data.
      // The browser attempts to send it even if the page is unloading.
      if (navigator.sendBeacon) {
        // Ensure the URL is correct for your API endpoint
        const status = navigator.sendBeacon('/api/shutdown', ''); // Empty body is fine
        console.log(`Shutdown signal sent to backend via sendBeacon. Status: ${status}`);
      } else {
        // Fallback for older browsers (less reliable during unload)
        console.warn('navigator.sendBeacon not supported, falling back to fetch (less reliable for unload).');
        fetch('/api/shutdown', {
          method: 'POST',
          keepalive: true, // Important for fetch during unload
          // No body needed if your backend doesn't expect one for this endpoint
        }).catch(err => console.error('Error sending shutdown signal via fetch:', err));
      }
    };

    window.addEventListener('unload', handleUnload);

    return () => {
      window.removeEventListener('unload', handleUnload);
      window.removeEventListener('error', handleGlobalError);
      window.removeEventListener('unhandledrejection', handleUnhandledRejection);
      cleanupWebSocket();
    };
  }, []); // Empty dependency array ensures this runs once on mount and cleans up on unmount

  return (
    <>
      {/* Loading Screen */}
      <LoadingScreen
        isVisible={isLoading}
        message={loadingMessage}
        progress={loadingProgress}
      />

      {/* Main Application */}
      <OnboardingProvider>
        <AuthProvider>
          <DynamicRouter>
        <Suspense fallback={<div className="min-h-screen bg-slate-900" />}>
        <Routes>
          {/* Public routes */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/unauthorized" element={<UnauthorizedPage />} />

          {/* Protected routes — one layout shell with a persistent dock.
              The 7 tool routes are gated by RequireSandbox; the catch-all
              lives inside the layout route so unauthenticated hits still
              pass through ProtectedRoute. */}
          <Route element={
            <ProtectedRoute>
              <SandboxProvider>
                <AppLayout />
              </SandboxProvider>
            </ProtectedRoute>
          }>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/proxy" element={<RequireSandbox><Proxy /></RequireSandbox>} />
            <Route path="/instrumentation/*" element={<RequireSandbox><Instrumentation /></RequireSandbox>} />
            <Route path="/reversing/*" element={<RequireSandbox><Reversing /></RequireSandbox>} />
            <Route path="/fuzzing/*" element={<RequireSandbox><Fuzzing /></RequireSandbox>} />
            <Route path="/static-analysis/*" element={<RequireSandbox><StaticAnalysis /></RequireSandbox>} />
            <Route path="/packet-capture/*" element={<RequireSandbox><PacketCapture /></RequireSandbox>} />
            <Route path="/auth-audit/*" element={<RequireSandbox><AuthAudit /></RequireSandbox>} />
            <Route path="/sandbox/*" element={<Sandbox />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Route>
        </Routes>
        </Suspense>
        <OnboardingSequence />
          </DynamicRouter>
        </AuthProvider>
      </OnboardingProvider>
    </>
  );
}

export default App;
