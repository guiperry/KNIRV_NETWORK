import React, { createContext, useContext, useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import { useBackend } from './BackendContext';

const RoleContext = createContext();

export const RoleProvider = ({ children }) => {
  const { serverInfo, isRunning } = useBackend();
  const router = useRouter();
  const [role, setRole] = useState('General'); // Default to most restricted role
  const [network, setNetwork] = useState('disconnected');
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    // Check for authentication from URL parameters (from main website)
    const urlParams = new URLSearchParams(window.location.search);
    const urlRole = urlParams.get('role');
    const urlNetwork = urlParams.get('network');
    const authToken = urlParams.get('token');

    // Check localStorage for existing session
    const storedRole = localStorage.getItem('knirv_user_role');
    const storedNetwork = localStorage.getItem('knirv_network');
    const storedAuth = localStorage.getItem('knirv_auth_token');

    if (urlRole && urlNetwork) {
      // New authentication from main website
      const normalizedRole = normalizeRole(urlRole);
      setRole(normalizedRole);
      setNetwork(urlNetwork);
      setIsAuthenticated(true);

      // Store in localStorage
      localStorage.setItem('knirv_user_role', normalizedRole);
      localStorage.setItem('knirv_network', urlNetwork);
      if (authToken) {
        localStorage.setItem('knirv_auth_token', authToken);
      }

      // Clean up URL
      const cleanUrl = window.location.pathname;
      window.history.replaceState({}, document.title, cleanUrl);

    } else if (storedRole && storedNetwork && storedAuth) {
      // Existing session
      setRole(storedRole);
      setNetwork(storedNetwork);
      setIsAuthenticated(true);
    } else if (serverInfo && serverInfo.role) {
      // Fallback to backend detection
      setRole(normalizeRole(serverInfo.role));
      setIsAuthenticated(true);
    }
  }, [serverInfo]);

  // Normalize role names from different sources
  const normalizeRole = (inputRole) => {
    const roleMap = {
      'root': 'Root',
      'bootnode': 'Bootnode',
      'dev': 'Dev',
      'developer': 'Dev',
      'general': 'General',
      'client': 'General',
      'peer': 'General'
    };

    return roleMap[inputRole.toLowerCase()] || 'General';
  };
  
  // Define page access by role
  const pageAccess = {
    Root: [
      'dashboard', 'inventory', 'vault', 'blockchain', 'dex', 'daos',
      'nft-vault', 'nft-capability-manager', 'add-capability',
      'settlement', 'network-admin', 'peers', 'explorer', 'capabilities'
    ],
    Bootnode: [
      'dashboard', 'inventory', 'vault', 'blockchain', 'dex', 'daos',
      'nft-vault', 'nft-capability-manager', 'add-capability',
      'settlement', 'peers', 'explorer', 'capabilities'
    ],
    Dev: [
      'dashboard', 'inventory', 'vault', 'blockchain', 'dex',
      'nft-vault', 'nft-capability-manager', 'add-capability',
      'explorer', 'capabilities'
    ],
    General: [
      'dashboard', 'inventory', 'dex', 'nft-capability-manager', 'capabilities'
    ]
  };
  
  // Check if a page is accessible for the current role
  const canAccess = (page) => {
    if (!isAuthenticated) return false;
    const accessList = pageAccess[role] || pageAccess.General;
    return accessList.includes(page);
  };

  // Logout function
  const logout = () => {
    setRole('General');
    setNetwork('disconnected');
    setIsAuthenticated(false);
    localStorage.removeItem('knirv_user_role');
    localStorage.removeItem('knirv_network');
    localStorage.removeItem('knirv_auth_token');
    router.push('/');
  };

  // Get user info for display
  const getUserInfo = () => ({
    role,
    network,
    isAuthenticated,
    displayName: role,
    networkDisplay: network === 'public-testnet' ? 'Public Testnet' :
                   network === 'private-testnet' ? 'Private Testnet' :
                   network === 'mainnet' ? 'Mainnet' : 'Disconnected'
  });

  return (
    <RoleContext.Provider value={{
      role,
      network,
      isAuthenticated,
      canAccess,
      logout,
      getUserInfo
    }}>
      {children}
    </RoleContext.Provider>
  );
};

export const useRole = () => useContext(RoleContext);