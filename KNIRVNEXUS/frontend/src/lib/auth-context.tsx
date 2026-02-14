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
    permissions: ['nexus:read', 'nexus:validate', 'nexus:update_assigned'],
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
    const token = localStorage.getItem('knirv_nexus_token');
    if (token) {
      validateToken(token);
    } else {
      setIsLoading(false);
    }
  }, []);

  const validateToken = async (token: string) => {
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
        localStorage.removeItem('knirv_nexus_token');
      }
    } catch (error) {
      console.error('Token validation failed:', error);
      localStorage.removeItem('knirv_nexus_token');
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
          localStorage.setItem('knirv_nexus_token', token);
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
        'testnet-admin-123': {
          user: 'testnet-admin',
          role: 'admin',
          permissions: ROLES.admin.permissions,
          nexus_access: ROLES.admin.nexus_access,
          authenticated: true
        },
        'testnet-validator-456': {
          user: 'testnet-validator',
          role: 'validator',
          permissions: ROLES.validator.permissions,
          nexus_access: ROLES.validator.nexus_access,
          node_id: 'validator-node-001',
          authenticated: true
        },
        'testnet-observer-789': {
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
        localStorage.setItem('knirv_nexus_token', token);
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
        localStorage.setItem('knirv_nexus_token', token);
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
    localStorage.removeItem('knirv_nexus_token');
  };

  const hasPermission = (service: string, operation: string): boolean => {
    if (!user?.authenticated) return false;
    
    // Admin has full access
    if (user.role === 'admin') return true;
    
    // Check nexus-specific permissions
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
