import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import { useRole } from '../contexts/RoleContext';

export default function RoleProtectedRoute({ children }) {
  const { role, isAuthenticated, canAccess, getUserInfo } = useRole();
  const router = useRouter();
  const [isLoading, setIsLoading] = useState(true);
  const [redirectAttempted, setRedirectAttempted] = useState(false);
  const currentPage = router.pathname.substring(1) || 'dashboard';

  useEffect(() => {
    // Wait a moment for authentication to be checked
    const timer = setTimeout(() => {
      setIsLoading(false);
    }, 500);

    return () => clearTimeout(timer);
  }, []);

  useEffect(() => {
    if (!isLoading) {
      // If not authenticated, check if we should allow demo mode
      if (!isAuthenticated) {
        // Check if we're in development mode or if demo mode is enabled
        const isDevelopment = process.env.NODE_ENV === 'development';
        const allowDemo = localStorage.getItem('knirv_demo_mode') === 'true';

        if (isDevelopment || allowDemo) {
          // Allow demo access with General role
          console.log('[WebGUI] Demo mode enabled - allowing access with General role');
          return;
        }

        // Prevent redirect loop - only redirect once
        if (!redirectAttempted && !window.location.search.includes('redirect=webgui')) {
          setRedirectAttempted(true);
          // In production, redirect to main website for authentication
          const mainWebsiteUrl = window.location.origin;
          console.log('[WebGUI] Redirecting to main website for authentication');
          window.location.href = `${mainWebsiteUrl}?redirect=webgui`;
          return;
        }

        // If we've already attempted redirect or are in a redirect loop, show auth screen
        console.log('[WebGUI] Showing authentication screen to prevent redirect loop');
        return;
      }

      // If the user can't access this page, redirect to a page they can access
      if (!canAccess(currentPage)) {
        // Find the first accessible page for their role
        const accessiblePages = {
          Root: 'network-admin',
          Bootnode: 'peers',
          Dev: 'dashboard',
          General: 'dashboard'
        };

        const redirectPage = accessiblePages[role] || 'dashboard';
        router.push(`/${redirectPage}`);
      }
    }
  }, [role, currentPage, canAccess, router, isAuthenticated, isLoading, redirectAttempted]);

  // Show loading while checking authentication
  if (isLoading) {
    return (
      <div style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: '100vh',
        background: 'linear-gradient(135deg, #0a0a23 0%, #1a1a3e 100%)',
        color: 'white',
        fontFamily: 'Arial, sans-serif'
      }}>
        <div style={{ textAlign: 'center' }}>
          <div style={{
            width: '50px',
            height: '50px',
            border: '3px solid rgba(255,255,255,0.3)',
            borderTop: '3px solid white',
            borderRadius: '50%',
            animation: 'spin 1s linear infinite',
            margin: '0 auto 20px'
          }}></div>
          <h3>Loading KNIRV WebGUI...</h3>
          <p>Checking authentication...</p>
        </div>
        <style jsx>{`
          @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
          }
        `}</style>
      </div>
    );
  }

  // Show unauthorized message if not authenticated
  if (!isAuthenticated) {
    const isDevelopment = process.env.NODE_ENV === 'development';

    return (
      <div style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: '100vh',
        background: 'linear-gradient(135deg, #0a0a23 0%, #1a1a3e 100%)',
        color: 'white',
        fontFamily: 'Arial, sans-serif'
      }}>
        <div style={{ textAlign: 'center', maxWidth: '400px', padding: '20px' }}>
          <h2>Authentication Required</h2>
          <p>Please log in through the main KNIRV website to access the WebGUI.</p>
          <div style={{ marginTop: '20px' }}>
            <button
              onClick={() => window.location.href = window.location.origin}
              style={{
                background: 'rgba(255,255,255,0.2)',
                border: '1px solid rgba(255,255,255,0.3)',
                color: 'white',
                padding: '10px 20px',
                borderRadius: '5px',
                cursor: 'pointer',
                marginRight: '10px'
              }}
            >
              Go to Main Website
            </button>
            {isDevelopment && (
              <button
                onClick={() => {
                  localStorage.setItem('knirv_demo_mode', 'true');
                  window.location.reload();
                }}
                style={{
                  background: 'rgba(255,255,255,0.1)',
                  border: '1px solid rgba(255,255,255,0.2)',
                  color: 'white',
                  padding: '10px 20px',
                  borderRadius: '5px',
                  cursor: 'pointer'
                }}
              >
                Demo Mode
              </button>
            )}
          </div>
        </div>
      </div>
    );
  }

  // If they can access the page, render the children
  return canAccess(currentPage) ? children : null;
}