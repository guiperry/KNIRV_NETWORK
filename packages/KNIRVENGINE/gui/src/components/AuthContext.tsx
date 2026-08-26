import React, { createContext, useState, useEffect, useContext, ReactNode } from 'react';

// User type definition
export interface User {
  id: number;
  username: string;
  email: string;
  role: string;
  permissions?: string[];
  created_at: string;
  updated_at: string;
}

// Auth context type definition
export interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (credentials: LoginCredentials) => Promise<AuthResult>;
  register: (userData: RegisterData) => Promise<AuthResult>;
  logout: () => Promise<void>;
  updateUser: (userData: Partial<User>) => void;
  hasPermission: (permission: string) => boolean;
  canAccessPage: (pageId: string) => boolean;
  canAccessSubPage: (parentPageId: string, subPageId: string) => boolean;
  refreshToken: () => Promise<boolean>;
  isAuthenticated: boolean;
}

// Login credentials type
export interface LoginCredentials {
  username: string;
  password: string;
}

// Register data type
export interface RegisterData {
  username: string;
  email: string;
  password: string;
}

// Auth result type
export interface AuthResult {
  success: boolean;
  error?: string;
}

// Auth provider props type
export interface AuthProviderProps {
  children: ReactNode;
}

// KNIRVENGINE has its own local API.  Do not fall back to the page origin: the
// GUI can be opened from a file URL or another KNIRV site, neither of which
// hosts the engine authentication service.
const getApiBaseUrl = (): string => {
  const configuredUrl = import.meta.env.VITE_API_BASE_URL?.trim();
  return (configuredUrl || (window.location.protocol === 'file:' ? 'http://localhost:8081' : '')).replace(/\/$/, '');
};

const getResponseError = async (response: Response, fallback: string): Promise<string> => {
  const body = await response.text();
  if (!body) return fallback;

  try {
    const parsed = JSON.parse(body) as { message?: string; error?: string };
    return parsed.message || parsed.error || fallback;
  } catch {
    return body.trim() || fallback;
  }
};

// Create the auth context
const AuthContext = createContext<AuthContextType | null>(null);

// Auth provider component
export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  // Initialize auth state from localStorage on component mount
  useEffect(() => {
    const initializeAuth = async () => {
      const token = localStorage.getItem('token');
      const storedUser = localStorage.getItem('user');

      if (token && storedUser) {
        try {
          const userData = JSON.parse(storedUser);

          // Verify token is still valid by making a test API call
          const response = await fetch(`${getApiBaseUrl()}/api/v1/auth/verify`, {
            method: 'GET',
            headers: {
              'Authorization': `Bearer ${token}`,
              'Content-Type': 'application/json',
            },
          });

          if (response.ok) {
            setUser(userData);
          } else {
            // Token is invalid, clear storage
            localStorage.removeItem('token');
            localStorage.removeItem('user');
            setUser(null);
          }
        } catch (error) {
          console.error('Failed to verify authentication:', error);
          // Clear invalid data
          localStorage.removeItem('token');
          localStorage.removeItem('user');
          setUser(null);
        }
      } else {
        setUser(null);
      }

      setLoading(false);
    };

    initializeAuth();
  }, []);

  // Login function
  const login = async (credentials: LoginCredentials): Promise<AuthResult> => {
    try {
      const response = await fetch(`${getApiBaseUrl()}/api/v1/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      if (!response.ok) {
        throw new Error(await getResponseError(response, 'Login failed'));
      }

      const data = await response.json();

      // Store token and user data
      localStorage.setItem('token', data.token);
      localStorage.setItem('user', JSON.stringify(data.user));
      setUser(data.user);

      return { success: true };
    } catch (error: unknown) {
      console.error('Login error:', error);
      const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
      return { success: false, error: errorMessage };
    }
  };

  // Logout function
  const logout = async (): Promise<void> => {
    try {
      // Call logout API (optional, as JWT is stateless)
      const token = localStorage.getItem('token');
      if (token) {
        await fetch(`${getApiBaseUrl()}/api/v1/auth/logout`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        });
      }
    } catch (error) {
      console.error('Logout API call failed:', error);
    } finally {
      // Clear local storage and state regardless of API call result
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      setUser(null);
    }
  };

  // Update user function
  const updateUser = (userData: Partial<User>): void => {
    if (!user) return;
    const updatedUser = { ...user, ...userData };
    setUser(updatedUser);
    localStorage.setItem('user', JSON.stringify(updatedUser));
  };

  // Check if user has a specific permission
  const hasPermission = (permission: string): boolean => {
    if (!user || !user.permissions) return false;
    // Allow all permissions for development user
    if (user.permissions.includes('*')) return true;
    return user.permissions.includes(permission);
  };

  // Role-based page access control. KNIRVENGINE is an operator toolbench —
  // every authenticated role gets the same tool access; 'root' additionally
  // gets network-admin.
  const toolPages = [
    'dashboard', 'proxy', 'instrumentation', 'reversing', 'fuzzing',
    'static-analysis', 'packet-capture', 'auth-audit', 'sandbox', 'settings'
  ];
  const pageAccess: Record<string, string[]> = {
    'root': [...toolPages, 'network-admin'],
    'bootnode': toolPages,
    'peer': toolPages,
    'client': toolPages,
    'user': toolPages
  };

  // Sub-page access control — same tool set for every role.
  const allRoles = ['root', 'bootnode', 'peer', 'client', 'user'];
  const uniformSubAccess = (subIds: string[]): Record<string, string[]> =>
    Object.fromEntries(allRoles.map(role => [role, subIds]));

  const subPageAccess: Record<string, Record<string, string[]>> = {
    'instrumentation': uniformSubAccess(['frida', 'proxychains-ng', 'bpftrace']),
	'reversing': uniformSubAccess(['cutter', 'ilspy', 'jadx']),
	'fuzzing': uniformSubAccess(['aflplusplus']),
	'static-analysis': uniformSubAccess(['semgrep', 'tree-sitter']),
    'packet-capture': uniformSubAccess(['wireshark', 'zeek']),
    'auth-audit': uniformSubAccess(['jwt-tool', 'saml-raider']),
    'sandbox': uniformSubAccess(['bubblewrap', 'novnc'])
  };

  // Check if user can access a specific page
  const canAccessPage = (pageId: string): boolean => {
    if (!user) return false;
    const userRole = user.role?.toLowerCase() || 'user';
    const accessList = pageAccess[userRole] || pageAccess['user'];
    return accessList.includes(pageId);
  };

  // Check if user can access a specific sub-page
  const canAccessSubPage = (parentPageId: string, subPageId: string): boolean => {
    if (!user) return false;
    const userRole = user.role?.toLowerCase() || 'user';
    const parentAccess = subPageAccess[parentPageId];
    if (!parentAccess) return false;
    const accessList = parentAccess[userRole] || parentAccess['user'] || [];
    return accessList.includes(subPageId);
  };

  // Register function
  const register = async (userData: RegisterData): Promise<AuthResult> => {
    try {
      const response = await fetch(`${getApiBaseUrl()}/api/v1/auth/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(userData),
      });

      if (!response.ok) {
        throw new Error(await getResponseError(response, 'Registration failed'));
      }

      const data = await response.json();

      // Store token and user data
      localStorage.setItem('token', data.token);
      localStorage.setItem('user', JSON.stringify(data.user));
      setUser(data.user);

      return { success: true };
    } catch (error: unknown) {
      console.error('Registration error:', error);
      const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
      return { success: false, error: errorMessage };
    }
  };

  // Refresh token function
  const refreshToken = async (): Promise<boolean> => {
    const token = localStorage.getItem('token');
    if (!token) return false;

    try {
      const response = await fetch(`${getApiBaseUrl()}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error('Token refresh failed');
      }

      const data = await response.json();
      localStorage.setItem('token', data.token);
      if (data.user) {
        localStorage.setItem('user', JSON.stringify(data.user));
        setUser(data.user);
      }

      return true;
    } catch (error) {
      console.error('Token refresh failed:', error);
      // If refresh fails, logout the user
      logout();
      return false;
    }
  };

  // Auth context value
  const value: AuthContextType = {
    user,
    loading,
    login,
    register,
    logout,
    updateUser,
    hasPermission,
    canAccessPage,
    canAccessSubPage,
    refreshToken,
    isAuthenticated: !!user,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

// Custom hook to use the auth context
export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === null) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
