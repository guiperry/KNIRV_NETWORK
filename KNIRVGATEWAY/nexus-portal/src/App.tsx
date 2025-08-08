import React, { useState, useEffect } from 'react';
import { LoginForm } from './components/auth/LoginForm';
import { DashboardWrapper } from './components/dashboard/DashboardWrapper';
import { ValidatorDashboard } from './components/dashboard/ValidatorDashboard';
import { AdminDashboard } from './components/dashboard/AdminDashboard';
import { ObserverDashboard } from './components/dashboard/ObserverDashboard';
import { useNexusSystem, useNexusDVE, useNexusValidation } from './hooks/use-realtime';
import { Shield, Server, Activity, Eye, User, Settings } from 'lucide-react';

// Types
interface AuthUser {
  user: string;
  role: 'admin' | 'validator' | 'observer';
  permissions: string[];
  nexus_access: string[];
  node_id?: string;
  authenticated: boolean;
}

interface SystemStatus {
  dve_manager: {
    status: string;
    service: string;
  };
  validation_core: {
    status: string;
    service: string;
  };
  timestamp: number;
}

function App() {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [loginError, setLoginError] = useState('');

  // Temporarily disable real-time connections to test basic loading
  // const systemRealtime = useNexusSystem(localStorage.getItem('knirv_nexus_token') || undefined);
  // const dveRealtime = useNexusDVE(localStorage.getItem('knirv_nexus_token') || undefined);
  // const validationRealtime = useNexusValidation(localStorage.getItem('knirv_nexus_token') || undefined);

  // Check for existing authentication
  useEffect(() => {
    const savedToken = localStorage.getItem('knirv_nexus_token');
    if (savedToken) {
      validateToken(savedToken);
    }
  }, []);

  // Fetch system status
  useEffect(() => {
    if (user?.authenticated) {
      fetchSystemStatus();
      const interval = setInterval(fetchSystemStatus, 30000); // Update every 30 seconds
      return () => clearInterval(interval);
    }
  }, [user]);

  const validateToken = async (authToken: string) => {
    try {
      // For testnet, directly validate known tokens without API call
      const mockUsers: Record<string, AuthUser> = {
        'testnet-admin-123': {
          user: 'testnet-admin',
          role: 'admin',
          permissions: ['*:*'],
          nexus_access: ['dve:*', 'validation:*', 'system:*'],
          authenticated: true
        },
        'testnet-validator-456': {
          user: 'testnet-validator',
          role: 'validator',
          permissions: ['nexus:read', 'nexus:validate'],
          nexus_access: ['dve:read', 'validation:read', 'validation:execute'],
          node_id: 'validator-node-001',
          authenticated: true
        },
        'testnet-observer-789': {
          user: 'testnet-observer',
          role: 'observer',
          permissions: ['*:read'],
          nexus_access: ['dve:read', 'validation:read', 'system:read'],
          authenticated: true
        }
      };

      if (mockUsers[authToken]) {
        setUser(mockUsers[authToken]);
        setIsConnected(true);
        return true;
      } else {
        // Try API call for production tokens
        try {
          const response = await fetch('/gateway/nexus/system/status', {
            headers: {
              'Authorization': `Bearer ${authToken}`
            }
          });

          if (response.ok) {
            setUser({
              user: 'production-user',
              role: 'observer',
              permissions: ['*:read'],
              nexus_access: ['dve:read', 'validation:read'],
              authenticated: true
            });
            setIsConnected(true);
            return true;
          }
        } catch (apiError) {
          console.log('API not available, using testnet mode');
        }

        localStorage.removeItem('knirv_nexus_token');
        setIsConnected(false);
        return false;
      }
    } catch (error) {
      console.error('Token validation failed:', error);
      setIsConnected(false);
      return false;
    }
  };

  const fetchSystemStatus = async () => {
    try {
      const token = localStorage.getItem('knirv_nexus_token');
      const response = await fetch('/gateway/nexus/system/status', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const status = await response.json();
        setSystemStatus(status);
        setIsConnected(true);
      }
    } catch (error) {
      console.error('Failed to fetch system status:', error);
      setIsConnected(false);
    }
  };

  const handleLogin = async (credentials: { username: string; password: string; role: string }) => {
    setLoginError('');

    try {
      // Mock authentication for testnet
      const testnetCredentials = {
        'admin': { username: 'admin', password: 'admin123', token: 'testnet-admin-123' },
        'validator': { username: 'validator', password: 'val123', token: 'testnet-validator-456' },
        'observer': { username: 'observer', password: 'obs123', token: 'testnet-observer-789' }
      };

      const expectedCreds = testnetCredentials[credentials.role as keyof typeof testnetCredentials];

      if (expectedCreds &&
          credentials.username === expectedCreds.username &&
          credentials.password === expectedCreds.password) {

        const success = await validateToken(expectedCreds.token);
        if (success) {
          localStorage.setItem('knirv_nexus_token', expectedCreds.token);
        } else {
          setLoginError('Authentication failed. Please try again.');
        }
      } else {
        setLoginError('Invalid credentials. Please check your username and password.');
      }
    } catch (error) {
      setLoginError('Login failed. Please try again.');
    }
  };

  const handleLogout = () => {
    setUser(null);
    setSystemStatus(null);
    setIsConnected(false);
    setLoginError('');
    localStorage.removeItem('knirv_nexus_token');
  };

  const useTestnetToken = (testToken: string) => {
    console.log('useTestnetToken called with:', testToken);

    // Directly set user based on token without async validation
    const mockUsers: Record<string, AuthUser> = {
      'testnet-admin-123': {
        user: 'testnet-admin',
        role: 'admin',
        permissions: ['*:*'],
        nexus_access: ['dve:*', 'validation:*', 'system:*'],
        authenticated: true
      },
      'testnet-validator-456': {
        user: 'testnet-validator',
        role: 'validator',
        permissions: ['nexus:read', 'nexus:validate'],
        nexus_access: ['dve:read', 'validation:read', 'validation:execute'],
        node_id: 'validator-node-001',
        authenticated: true
      },
      'testnet-observer-789': {
        user: 'testnet-observer',
        role: 'observer',
        permissions: ['*:read'],
        nexus_access: ['dve:read', 'validation:read', 'system:read'],
        authenticated: true
      }
    };

    if (mockUsers[testToken]) {
      console.log('Setting user to:', mockUsers[testToken]);
      setUser(mockUsers[testToken]);
      setIsConnected(true);
      localStorage.setItem('knirv_nexus_token', testToken);
      setLoginError('');
    } else {
      setLoginError('Invalid testnet token.');
    }
  };

  console.log('Render state:', { user, isAuthenticated: user?.authenticated });

  // If user is not authenticated, show login form (not the landing page)
  if (!user?.authenticated) {
    return <LoginForm onLogin={handleLogin} />;
  }

  // If authenticated, show the dashboard
  return (
    <DashboardWrapper
      user={user}
      onLogout={handleLogout}
      isConnected={isConnected}
      alerts={0}
    />
  );
}

export default App;
