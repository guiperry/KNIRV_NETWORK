import React, { createContext, useContext, useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import { useBackend } from './BackendContext';

const RoleContext = createContext();

export const DEFAULT_PAGE_ACCESS = {
  Root: [
    'dashboard', 'controller-status', 'qr-connect', 'my-endpoints', 'payment-gateway',
    'environments', 'dve-list', 'models', 'codex-builder', 'models-dex', 'arena',
    'monitor', 'network-monitor', 'graph-explorer', 'chain-explorer', 'chain-explorer-new',
    'oracle-explorer', 'peers', 'operator-registry', 'tunnel-registry', 'error-explorer',
    'transaction-explorer', 'validation-explorer',
    'governance', 'bootnode-dao', 'network-inference-dao',
    'graphchain', 'graphchain-dashboard', 'graphchain-errors', 'graphchain-skills',
    'marketplace', 'skills', 'capabilities', 'properties', 'settlement',
    'my-models', 'my-wallets', 'my-skills', 'my-capabilities', 'my-properties', 'nft-property-explorer',
    'tools', 'basic', 'advanced',
    'settings', 'inventory', 'blockchain', 'dex', 'daos',
    'nft-capability-manager', 'add-capability',
    'network-admin', 'auth-test',
  ],
  Bootnode: [
    'dashboard', 'controller-status', 'qr-connect', 'my-endpoints', 'payment-gateway',
    'environments', 'dve-list', 'models', 'codex-builder', 'models-dex', 'arena',
    'monitor', 'network-monitor', 'graph-explorer', 'chain-explorer', 'chain-explorer-new',
    'oracle-explorer', 'peers', 'operator-registry', 'tunnel-registry', 'error-explorer',
    'transaction-explorer', 'validation-explorer',
    'governance', 'bootnode-dao', 'network-inference-dao',
    'graphchain', 'graphchain-dashboard', 'graphchain-errors', 'graphchain-skills',
    'marketplace', 'skills', 'capabilities', 'properties', 'settlement',
    'my-models', 'my-wallets', 'my-skills', 'my-capabilities', 'my-properties', 'nft-property-explorer',
    'tools', 'basic', 'advanced',
    'settings', 'inventory', 'blockchain', 'dex', 'daos',
    'nft-capability-manager', 'add-capability',
    'auth-test',
  ],
  Dev: [
    'dashboard', 'controller-status', 'qr-connect', 'my-endpoints', 'payment-gateway',
    'environments', 'dve-list', 'models', 'codex-builder', 'models-dex', 'arena',
    'monitor', 'network-monitor', 'graph-explorer', 'chain-explorer', 'chain-explorer-new',
    'oracle-explorer', 'peers', 'operator-registry', 'tunnel-registry', 'error-explorer',
    'transaction-explorer', 'validation-explorer',
    'governance', 'bootnode-dao', 'network-inference-dao',
    'graphchain', 'graphchain-dashboard', 'graphchain-errors', 'graphchain-skills',
    'marketplace', 'skills', 'capabilities', 'properties', 'settlement',
    'my-models', 'my-wallets', 'my-skills', 'my-capabilities', 'my-properties', 'nft-property-explorer',
    'tools', 'basic', 'advanced',
    'settings', 'inventory', 'blockchain', 'dex',
    'nft-capability-manager', 'add-capability',
    'explorer', 'capabilities', 'auth-test',
  ],
  General: [
    'dashboard', 'controller-status', 'qr-connect', 'my-endpoints', 'payment-gateway',
    'environments', 'dve-list', 'models', 'codex-builder', 'models-dex', 'arena',
    'monitor', 'network-monitor', 'graph-explorer', 'chain-explorer', 'chain-explorer-new',
    'oracle-explorer', 'peers', 'operator-registry', 'tunnel-registry', 'error-explorer',
    'transaction-explorer', 'validation-explorer',
    'governance', 'bootnode-dao', 'network-inference-dao',
    'graphchain', 'graphchain-dashboard', 'graphchain-errors', 'graphchain-skills',
    'marketplace', 'skills', 'capabilities', 'properties', 'settlement',
    'my-models', 'my-wallets', 'my-skills', 'my-capabilities', 'my-properties', 'nft-property-explorer',
    'tools', 'basic', 'advanced',
    'settings', 'inventory', 'dex', 'nft-capability-manager', 'capabilities', 'auth-test',
  ],
};

export const RoleProvider = ({ children }) => {
  const { serverInfo } = useBackend();
  const router = useRouter();
  const [role, setRole] = useState('General');
  const [network, setNetwork] = useState('disconnected');
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  // Load persisted page access config from localStorage, fallback to defaults
  const [pageAccess, setPageAccess] = useState(() => {
    try {
      if (typeof window !== 'undefined') {
        const stored = localStorage.getItem('knirv_page_access');
        if (stored) return JSON.parse(stored);
      }
    } catch {}
    return DEFAULT_PAGE_ACCESS;
  });

  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const urlRole = urlParams.get('role');
    const urlNetwork = urlParams.get('network');
    const authToken = urlParams.get('token');

    const storedRole = localStorage.getItem('knirv_user_role');
    const storedNetwork = localStorage.getItem('knirv_network');
    const storedAuth = localStorage.getItem('knirv_auth_token');

    if (urlRole && urlNetwork) {
      const normalizedRole = normalizeRole(urlRole);
      setRole(normalizedRole);
      setNetwork(urlNetwork);
      setIsAuthenticated(true);

      localStorage.setItem('knirv_user_role', normalizedRole);
      localStorage.setItem('knirv_network', urlNetwork);
      if (authToken) localStorage.setItem('knirv_auth_token', authToken);

      window.history.replaceState({}, document.title, window.location.pathname);
    } else if (storedRole && storedNetwork && storedAuth) {
      setRole(storedRole);
      setNetwork(storedNetwork);
      setIsAuthenticated(true);
    } else if (serverInfo && serverInfo.role) {
      setRole(normalizeRole(serverInfo.role));
    } else if (localStorage.getItem('knirv_demo_mode') === 'true') {
      setRole('Root');
      setNetwork('demo');
      setIsAuthenticated(true);
      console.log('[RoleContext] Demo mode enabled with Root access');
    }
  }, [serverInfo]);

  const normalizeRole = (inputRole) => {
    const roleMap = {
      'root': 'Root',
      'admin': 'Root',
      'bootnode': 'Bootnode',
      'dev': 'Dev',
      'developer': 'Dev',
      'general': 'General',
      'user': 'General',
      'client': 'General',
      'peer': 'General',
    };
    return roleMap[inputRole.toLowerCase()] || 'General';
  };

  const updatePageAccess = (newAccess) => {
    setPageAccess(newAccess);
    try {
      localStorage.setItem('knirv_page_access', JSON.stringify(newAccess));
    } catch (e) {
      console.warn('[RoleContext] Failed to persist page access:', e);
    }
  };

  const canAccess = (page) => {
    if (!isAuthenticated) return false;
    const accessList = pageAccess[role] || pageAccess.General;
    return accessList.includes(page);
  };

  const logout = () => {
    setRole('General');
    setNetwork('disconnected');
    setIsAuthenticated(false);
    localStorage.removeItem('knirv_user_role');
    localStorage.removeItem('knirv_network');
    localStorage.removeItem('knirv_auth_token');
    localStorage.removeItem('knirv_demo_mode');
    router.push('/');
  };

  const getUserInfo = () => ({
    role,
    network,
    isAuthenticated,
    displayName: role,
    networkDisplay:
      network === 'public-testnet' ? 'Public Testnet' :
      network === 'private-testnet' ? 'Private Testnet' :
      network === 'mainnet' ? 'Mainnet' :
      network === 'demo' ? 'Demo Mode' : 'Disconnected',
  });

  return (
    <RoleContext.Provider value={{
      role,
      network,
      isAuthenticated,
      canAccess,
      logout,
      getUserInfo,
      pageAccess,
      updatePageAccess,
      defaultPageAccess: DEFAULT_PAGE_ACCESS,
    }}>
      {children}
    </RoleContext.Provider>
  );
};

export const useRole = () => useContext(RoleContext);
