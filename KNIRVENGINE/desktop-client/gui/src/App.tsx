import React, { useState, useEffect } from 'react';
import { BrowserRouter, HashRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Sidebar } from './components/Sidebar';
import { Dashboard } from './components/Dashboard';
import { AgentManager } from './components/AgentManager';
import { CapabilityStore } from './components/CapabilityStore';
import { TargetManager } from './components/TargetManager';

import { WorkflowOrchestrator } from './components/WorkflowOrchestrator';
import { Analytics } from './components/Analytics';
import { Settings } from './components/Settings';
import { WebConnections } from './components/WebConnections';
import { Wallet } from './components/Wallet';
import { AuthProvider } from './components/AuthContext';
import LoginPage from './components/LoginPage';
import ProtectedRoute from './components/ProtectedRoute';
import UnauthorizedPage from './components/UnauthorizedPage';
import RealTimeNotifications from './components/RealTimeNotifications';
import ErrorBoundary from './components/ErrorBoundary';
import { initializeWebSocket, cleanupWebSocket } from './utils/websocket';
import { initializeAccessibility } from './utils/accessibility';
import { enableIntelligentErrorAnalysis, handleApiError } from './utils/errorHandler';
import { OnboardingProvider } from './components/onboarding/OnboardingProvider';
import OnboardingSequence from './components/onboarding/OnboardingSequence';
import LoadingScreen from './components/LoadingScreen';
import './components/onboarding/onboarding.css';

type ActiveView = 'dashboard' | 'agents' | 'capabilities' | 'targets' | 'workflows' | 'analytics' | 'settings' | 'web-connections' | 'wallet';

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
  const [activeView, setActiveView] = useState<ActiveView>('dashboard');
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [loadingMessage, setLoadingMessage] = useState('Starting KNIRVENGINE...');
  const [loadingProgress, setLoadingProgress] = useState(0);

  // Application initialization with loading screen
  useEffect(() => {
    const initializeApp = async () => {
      try {
        setLoadingMessage('Initializing error handling...');
        setLoadingProgress(10);

        // Initialize error handling
        enableIntelligentErrorAnalysis(true);

        setLoadingMessage('Setting up accessibility...');
        setLoadingProgress(20);

        // Initialize accessibility features
        initializeAccessibility();

        setLoadingMessage('Connecting to backend...');
        setLoadingProgress(40);

        // Check backend connectivity with retry logic
        const apiBaseUrl = isElectron() ? 'http://localhost:8081' : '';
        let backendReady = false;
        const maxRetries = 10;
        const retryDelay = 1000; // 1 second

        for (let attempt = 1; attempt <= maxRetries; attempt++) {
          try {
            setLoadingMessage(`Connecting to backend... (attempt ${attempt}/${maxRetries})`);
            const healthResponse = await fetch(`${apiBaseUrl}/api/v1/health`, {
              signal: AbortSignal.timeout(3000)
            });
            if (healthResponse.ok) {
              backendReady = true;
              console.log('Backend health check successful');
              break;
            } else {
              throw new Error(`Backend health check failed: ${healthResponse.status}`);
            }
          } catch (error) {
            console.warn(`Backend connection attempt ${attempt} failed:`, error);
            if (attempt < maxRetries) {
              // Wait before retrying
              await new Promise(resolve => setTimeout(resolve, retryDelay));
            } else {
              console.error('Backend connection failed after all retries, continuing with limited functionality');
              handleApiError(error, {
                operation: 'backend_health_check',
                component: 'App',
                attempt: attempt,
                maxRetries: maxRetries,
                timestamp: new Date().toISOString(),
                context: 'Failed to connect to backend server during initialization'
              });
            }
          }
        }

        if (backendReady) {
          setLoadingMessage('Backend connected successfully!');
        } else {
          setLoadingMessage('Starting with limited functionality...');
        }

        setLoadingMessage('Establishing WebSocket connection...');
        setLoadingProgress(60);

        // Initialize WebSocket connection (only if backend is ready)
        if (backendReady) {
          try {
            await initializeWebSocket();
            setLoadingMessage('WebSocket connected successfully!');
          } catch (error) {
            console.error('Failed to initialize WebSocket:', error);
            handleApiError(error, {
              operation: 'websocket_initialization',
              component: 'App',
              timestamp: new Date().toISOString(),
              context: 'Failed to initialize WebSocket connection'
            });
            setLoadingMessage('WebSocket connection failed, continuing...');
          }
        } else {
          setLoadingMessage('Skipping WebSocket connection (backend not ready)...');
        }

        setLoadingMessage('Loading application data...');
        setLoadingProgress(80);

        // Simulate loading time for other services
        await new Promise(resolve => setTimeout(resolve, 1000));

        setLoadingMessage('Finalizing initialization...');
        setLoadingProgress(90);

        // Additional loading time to show off the cool loading screen
        await new Promise(resolve => setTimeout(resolve, 2000));

        setLoadingMessage('Ready!');
        setLoadingProgress(100);

        // Extended delay to enjoy the loading screen
        setTimeout(() => {
          setIsLoading(false);
        }, 3500); // Extended from 500ms to 3500ms (3 more seconds)

      } catch (error) {
        console.error('App initialization error:', error);
        handleApiError(error, {
          operation: 'app_initialization',
          component: 'App',
          timestamp: new Date().toISOString(),
          context: 'Application initialization failed'
        });

        // Still allow app to load even if some initialization fails
        setLoadingMessage('Starting with limited functionality...');
        setTimeout(() => {
          setIsLoading(false);
        }, 1000);
      }
    };

    initializeApp();
  }, []);

  useEffect(() => {
    // Global error handlers and cleanup (separate from initialization)

    // Set up global error handlers with auto-reload for critical errors
    const handleGlobalError = (event: ErrorEvent) => {
      console.error('Global error caught:', event.error, event);

      const errorDetails = handleApiError(event.error || new Error(event.message), {
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

      const errorDetails = handleApiError(event.reason, {
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

    // Initialize WebSocket connection
    initializeWebSocket().catch(error => {
      console.error('Failed to initialize WebSocket:', error);
      handleApiError(error, {
        operation: 'websocket_initialization',
        component: 'App',
        timestamp: new Date().toISOString(),
        context: 'Failed to initialize WebSocket connection'
      });
    });

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
        <Routes>
          {/* Public routes */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/unauthorized" element={<UnauthorizedPage />} />
          
          {/* Protected routes */}
          <Route path="/" element={
            <ProtectedRoute>
              <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
                <div className="flex">
                  <Sidebar 
                    activeView={activeView} 
                    setActiveView={setActiveView}
                    isOpen={sidebarOpen}
                    setIsOpen={setSidebarOpen}
                  />
                  <main id="main-content" className="flex-1 lg:ml-64" role="main" aria-label="Main content">
                    <div className="lg:hidden">
                      <button
                        onClick={() => setSidebarOpen(!sidebarOpen)}
                        className="fixed top-4 left-4 z-50 p-2 bg-slate-800 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                        aria-label="Toggle navigation menu"
                        aria-expanded={sidebarOpen}
                      >
                        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                        </svg>
                      </button>
                    </div>
                    {/* Real-time notifications */}
                    <div className="fixed top-4 right-4 z-50">
                      <RealTimeNotifications />
                    </div>
                    <Dashboard />
                  </main>
                </div>
              </div>
            </ProtectedRoute>
          } />
          
          <Route path="/dashboard" element={
            <ProtectedRoute>
              <ErrorBoundary componentName="Dashboard">
                <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
                  <div className="flex">
                    <Sidebar
                      activeView={activeView}
                      setActiveView={setActiveView}
                      isOpen={sidebarOpen}
                      setIsOpen={setSidebarOpen}
                    />
                    <main className="flex-1 lg:ml-64">
                      <div className="lg:hidden">
                        <button
                          onClick={() => setSidebarOpen(!sidebarOpen)}
                          className="fixed top-4 left-4 z-50 p-2 bg-slate-800 rounded-lg text-white"
                        >
                          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                          </svg>
                        </button>
                      </div>
                      <Dashboard />
                    </main>
                  </div>
                </div>
              </ErrorBoundary>
            </ProtectedRoute>
          } />
          
          <Route path="/agents" element={
            <ProtectedRoute>
              <ErrorBoundary componentName="AgentManager">
                <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
                  <div className="flex">
                    <Sidebar
                      activeView={activeView}
                      setActiveView={setActiveView}
                      isOpen={sidebarOpen}
                      setIsOpen={setSidebarOpen}
                    />
                    <main className="flex-1 lg:ml-64">
                      <div className="lg:hidden">
                        <button
                          onClick={() => setSidebarOpen(!sidebarOpen)}
                          className="fixed top-4 left-4 z-50 p-2 bg-slate-800 rounded-lg text-white"
                        >
                          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                          </svg>
                        </button>
                      </div>
                      <AgentManager />
                    </main>
                  </div>
                </div>
              </ErrorBoundary>
            </ProtectedRoute>
          } />
          
          <Route path="/capabilities" element={
            <ProtectedRoute>
              <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
                <div className="flex">
                  <Sidebar 
                    activeView={activeView} 
                    setActiveView={setActiveView}
                    isOpen={sidebarOpen}
                    setIsOpen={setSidebarOpen}
                  />
                  <main className="flex-1 lg:ml-64">
                    <div className="lg:hidden">
                      <button
                        onClick={() => setSidebarOpen(!sidebarOpen)}
                        className="fixed top-4 left-4 z-50 p-2 bg-slate-800 rounded-lg text-white"
                      >
                        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                        </svg>
                      </button>
                    </div>
                    <CapabilityStore />
                  </main>
                </div>
              </div>
            </ProtectedRoute>
          } />
          
          <Route path="/targets" element={
            <ProtectedRoute>
              <ErrorBoundary componentName="TargetManager">
                <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
                  <div className="flex">
                    <Sidebar
                      activeView={activeView}
                      setActiveView={setActiveView}
                      isOpen={sidebarOpen}
                      setIsOpen={setSidebarOpen}
                    />
                    <main className="flex-1 lg:ml-64">
                      <div className="lg:hidden">
                        <button
                          onClick={() => setSidebarOpen(!sidebarOpen)}
                          className="fixed top-4 left-4 z-50 p-2 bg-slate-800 rounded-lg text-white"
                        >
                          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                          </svg>
                        </button>
                      </div>
                      <TargetManager />
                    </main>
                  </div>
                </div>
              </ErrorBoundary>
            </ProtectedRoute>
          } />
          

          
          <Route path="/workflows" element={
            <ProtectedRoute>
              <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
                <div className="flex">
                  <Sidebar 
                    activeView={activeView} 
                    setActiveView={setActiveView}
                    isOpen={sidebarOpen}
                    setIsOpen={setSidebarOpen}
                  />
                  <main className="flex-1 lg:ml-64">
                    <div className="lg:hidden">
                      <button
                        onClick={() => setSidebarOpen(!sidebarOpen)}
                        className="fixed top-4 left-4 z-50 p-2 bg-slate-800 rounded-lg text-white"
                      >
                        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                        </svg>
                      </button>
                    </div>
                    <WorkflowOrchestrator />
                  </main>
                </div>
              </div>
            </ProtectedRoute>
          } />
          
          <Route path="/analytics" element={
            <ProtectedRoute>
              <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
                <div className="flex">
                  <Sidebar 
                    activeView={activeView} 
                    setActiveView={setActiveView}
                    isOpen={sidebarOpen}
                    setIsOpen={setSidebarOpen}
                  />
                  <main className="flex-1 lg:ml-64">
                    <div className="lg:hidden">
                      <button
                        onClick={() => setSidebarOpen(!sidebarOpen)}
                        className="fixed top-4 left-4 z-50 p-2 bg-slate-800 rounded-lg text-white"
                      >
                        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                        </svg>
                      </button>
                    </div>
                    <Analytics />
                  </main>
                </div>
              </div>
            </ProtectedRoute>
          } />

          <Route path="/web-connections" element={
            <ProtectedRoute>
              <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
                <div className="flex">
                  <Sidebar
                    activeView={activeView}
                    setActiveView={setActiveView}
                    isOpen={sidebarOpen}
                    setIsOpen={setSidebarOpen}
                  />
                  <main className="flex-1 lg:ml-64">
                    <div className="lg:hidden">
                      <button
                        onClick={() => setSidebarOpen(!sidebarOpen)}
                        className="fixed top-4 left-4 z-50 p-2 bg-slate-800 rounded-lg text-white"
                      >
                        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                        </svg>
                      </button>
                    </div>
                    <WebConnections />
                  </main>
                </div>
              </div>
            </ProtectedRoute>
          } />

          <Route path="/wallet" element={
            <ProtectedRoute>
              <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
                <div className="flex">
                  <Sidebar
                    activeView={activeView}
                    setActiveView={setActiveView}
                    isOpen={sidebarOpen}
                    setIsOpen={setSidebarOpen}
                  />
                  <main className="flex-1 lg:ml-64">
                    <div className="lg:hidden">
                      <button
                        onClick={() => setSidebarOpen(!sidebarOpen)}
                        className="fixed top-4 left-4 z-50 p-2 bg-slate-800 rounded-lg text-white"
                      >
                        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                        </svg>
                      </button>
                    </div>
                    <Wallet />
                  </main>
                </div>
              </div>
            </ProtectedRoute>
          } />

          <Route path="/settings" element={
            <ProtectedRoute>
              <ErrorBoundary componentName="Settings">
                <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
                  <div className="flex">
                    <Sidebar
                      activeView={activeView}
                      setActiveView={setActiveView}
                      isOpen={sidebarOpen}
                      setIsOpen={setSidebarOpen}
                    />
                    <main className="flex-1 lg:ml-64">
                      <div className="lg:hidden">
                        <button
                          onClick={() => setSidebarOpen(!sidebarOpen)}
                          className="fixed top-4 left-4 z-50 p-2 bg-slate-800 rounded-lg text-white"
                        >
                          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                          </svg>
                        </button>
                      </div>
                      <Settings />
                    </main>
                  </div>
                </div>
              </ErrorBoundary>
            </ProtectedRoute>
          } />
          
          {/* Redirect any other path to dashboard */}
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
        <OnboardingSequence />
          </DynamicRouter>
        </AuthProvider>
      </OnboardingProvider>
    </>
  );
}

export default App;