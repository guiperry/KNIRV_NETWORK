'use client';

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { API_BASE_URL } from '@/lib/api';

// Role definitions matching the backend
export const ROLES = {
  admin: {
    permissions: ['*:*'],
    nexus_access: ['dve:*', 'validation:*', 'system:*', 'fabric:*'],
    description: 'Full administrative access',
    displayName: 'Root'
  },
  validator: {
    permissions: ['server:read', 'server:validate', 'server:update_assigned'],
    nexus_access: ['dve:read', 'validation:read', 'validation:execute', 'system:read', 'fabric:read'],
    description: 'Validator node operator with scoped access',
    displayName: 'Operator'
  },
  observer: {
    permissions: ['*:read'],
    nexus_access: ['dve:read', 'validation:read', 'system:read', 'fabric:read'],
    description: 'Read-only access to all services',
    displayName: 'Developer'
  }
} as const;

export type Role = keyof typeof ROLES;

export interface AuthUser {
  user: string;
  role: Role;
  permissions: readonly string[];
  nexus_access: readonly string[];
  node_id?: string;
  authenticated: boolean;
}

interface AuthContextType {
  user: AuthUser | null;
  login: (token: string) => Promise<boolean>;
  loginWithCredentials: (username: string, password: string) => Promise<boolean>;
  logout: () => void;
  hasPermission: (service: string, operation: string) => boolean;
  hasNodeAccess: (nodeId?: string) => boolean;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// The desktop client used the `knirv_auth_*` keys before the Nexus UI was
// introduced. Keep the aliases in sync while they are supported: clearing only
// one token lets a rejected session repeatedly redirect between `/` and
// `/login`.
export const getStoredAuthToken = (): string | null =>
  localStorage.getItem('knirv_nexus_token') || localStorage.getItem('knirv_auth_token');

export const persistStoredAuth = (token: string, role?: string, username?: string) => {
  localStorage.setItem('knirv_nexus_token', token);
  localStorage.setItem('knirv_auth_token', token);
  if (role) {
    localStorage.setItem('knirv_nexus_role', role);
    localStorage.setItem('knirv_auth_role', role);
  }
  if (username) localStorage.setItem('knirv_nexus_user', username);
};

export const clearStoredAuth = () => {
  [
    'knirv_nexus_token',
    'knirv_auth_token',
    'knirv_nexus_role',
    'knirv_auth_role',
    'knirv_nexus_user',
  ].forEach(key => localStorage.removeItem(key));
};

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Check for existing token on mount
  useEffect(() => {
    const token = getStoredAuthToken();
    if (token) {
      validateToken(token);
    } else {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    const handleDesktopAuth = (event: MessageEvent) => {
      const data = event.data;
      if (!data || data.type !== 'desktop-auth' || typeof data.token !== 'string' || !data.token) {
        return;
      }

      persistStoredAuth(data.token, typeof data.role === 'string' ? data.role : undefined);
      validateToken(data.token);
    };

    window.addEventListener('message', handleDesktopAuth);
    return () => window.removeEventListener('message', handleDesktopAuth);
  }, []);

  const validateToken = async (token: string) => {
    // Handle testnet tokens without a backend round-trip
    const testnetTokens: Record<string, AuthUser> = {
      'TESTNET_ADMIN_TOKEN': {
        user: 'testnet-admin',
        role: 'admin',
        permissions: ROLES.admin.permissions,
        nexus_access: ROLES.admin.nexus_access,
        authenticated: true
      },
      'TESTNET_VALIDATOR_TOKEN': {
        user: 'testnet-validator',
        role: 'validator',
        permissions: ROLES.validator.permissions,
        nexus_access: ROLES.validator.nexus_access,
        node_id: 'validator-node-001',
        authenticated: true
      },
      'TESTNET_OBSERVER_TOKEN': {
        user: 'testnet-observer',
        role: 'observer',
        permissions: ROLES.observer.permissions,
        nexus_access: ROLES.observer.nexus_access,
        authenticated: true
      }
    };

    if (testnetTokens[token]) {
      setUser(testnetTokens[token]);
      setIsLoading(false);
      return;
    }

    // For JWTs, check expiry client-side before hitting the backend
    if (typeof token === 'string' && token.startsWith('ey')) {
      try {
        const payload = JSON.parse(atob(token.split('.')[1]));
        if (payload.exp && payload.exp * 1000 < Date.now()) {
          clearStoredAuth();
          setIsLoading(false);
          return;
        }
      } catch {
        // Malformed token — let the backend reject it
      }
    }

    try {
      const response = await fetch(`${API_BASE_URL}/api/auth/me`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const userData = await response.json();
        setUser({
          user: userData.username,
          role: userData.role as Role,
          permissions: ROLES[userData.role as Role]?.permissions || [],
          nexus_access: ROLES[userData.role as Role]?.nexus_access || [],
          authenticated: true
        });
      } else {
        clearStoredAuth();
      }
    } catch (error) {
      console.error('Token validation failed:', error);
      // Backend unreachable — try to restore session from cached role.
      const cachedRole = localStorage.getItem('knirv_nexus_role') as Role | null;
      const cachedUser = localStorage.getItem('knirv_nexus_user');
      if (cachedRole && ROLES[cachedRole]) {
        setUser({
          user: cachedUser || 'user',
          role: cachedRole,
          permissions: ROLES[cachedRole].permissions,
          nexus_access: ROLES[cachedRole].nexus_access,
          authenticated: true,
        });
      } else {
        clearStoredAuth();
      }
    } finally {
      setIsLoading(false);
    }
  };

  const loginWithCredentials = async (username: string, password: string): Promise<boolean> => {
    try {
      setIsLoading(true);

      const response = await fetch(`${API_BASE_URL}/api/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, password }),
      });

      if (response.ok) {
        const loginData = await response.json();
        const token = loginData.token;

        // Get user details with the token
        const userResponse = await fetch(`${API_BASE_URL}/api/auth/me`, {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
          }
        });

        if (userResponse.ok) {
          const userData = await userResponse.json();
          setUser({
            user: userData.username,
            role: userData.role as Role,
            permissions: ROLES[userData.role as Role]?.permissions || [],
            nexus_access: ROLES[userData.role as Role]?.nexus_access || [],
            authenticated: true
          });
          persistStoredAuth(token, userData.role, userData.username);
          return true;
        }
      }

      return false;
    } catch (error) {
      console.error('Login with credentials failed:', error);
      return false;
    } finally {
      setIsLoading(false);
    }
  };

  const login = async (token: string): Promise<boolean> => {
    try {
      setIsLoading(true);

      // For testnet mode, use predefined tokens
      const testnetTokens: Record<string, AuthUser> = {
        'TESTNET_ADMIN_TOKEN': {
          user: 'testnet-admin',
          role: 'admin',
          permissions: ROLES.admin.permissions,
          nexus_access: ROLES.admin.nexus_access,
          authenticated: true
        },
        'TESTNET_VALIDATOR_TOKEN': {
          user: 'testnet-validator',
          role: 'validator',
          permissions: ROLES.validator.permissions,
          nexus_access: ROLES.validator.nexus_access,
          node_id: 'validator-node-001',
          authenticated: true
        },
        'TESTNET_OBSERVER_TOKEN': {
          user: 'testnet-observer',
          role: 'observer',
          permissions: ROLES.observer.permissions,
          nexus_access: ROLES.observer.nexus_access,
          authenticated: true
        }
      };

      // Check if it's a testnet token
      if (testnetTokens[token]) {
        setUser(testnetTokens[token]);
        persistStoredAuth(token);
        return true;
      }

      // Validate with backend
      const response = await fetch(`${API_BASE_URL}/api/auth/me`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const userData = await response.json();
        setUser({
          user: userData.username,
          role: userData.role as Role,
          permissions: ROLES[userData.role as Role]?.permissions || [],
          nexus_access: ROLES[userData.role as Role]?.nexus_access || [],
          authenticated: true
        });
        persistStoredAuth(token, userData.role, userData.username);
        return true;
      }

      return false;
    } catch (error) {
      console.error('Login failed:', error);
      return false;
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    setUser(null);
    clearStoredAuth();
  };

  const hasPermission = (service: string, operation: string): boolean => {
    if (!user?.authenticated) return false;
    
    // Admin has full access
    if (user.role === 'admin') return true;
    
    // Check server-specific permissions
    const requiredPermission = `${service}:${operation}`;
    const hasWildcard = user.nexus_access.includes(`${service}:*`);
    const hasSpecific = user.nexus_access.includes(requiredPermission);
    
    return hasWildcard || hasSpecific;
  };

  const hasNodeAccess = (nodeId?: string): boolean => {
    if (!user?.authenticated) return false;
    
    // Admin and observer have access to all nodes
    if (user.role === 'admin' || user.role === 'observer') return true;
    
    // Validator is scoped to specific nodes
    if (user.role === 'validator') {
      return !user.node_id || !nodeId || user.node_id === nodeId;
    }
    
    return false;
  };

  const value: AuthContextType = {
    user,
    login,
    loginWithCredentials,
    logout,
    hasPermission,
    hasNodeAccess,
    isLoading
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}
