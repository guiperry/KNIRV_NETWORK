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

  const clearStoredAuth = () => {
    localStorage.removeItem('knirv_user_role');
    localStorage.removeItem('knirv_network');
    localStorage.removeItem('knirv_auth_token');
  };

  // Authenticate from a postMessage response (used when embedded in KNIRVSERVER iframe).
  const applyAuthMessage = (data) => {
    const normalizedRole = normalizeRole(data.role || 'admin');
    const net = data.network || 'local';
    const tok = data.token || '';
    setRole(normalizedRole);
    setNetwork(net);
    setIsAuthenticated(true);
    localStorage.setItem('knirv_user_role', normalizedRole);
    localStorage.setItem('knirv_network', net);
    if (tok) localStorage.setItem('knirv_auth_token', tok);
    console.log('[RoleContext] Authenticated via parent frame postMessage, role:', normalizedRole);
  };

  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const urlRole = urlParams.get('role');
    const urlNetwork = urlParams.get('network');
    const authToken = urlParams.get('token');

    if (!urlRole && !urlNetwork && !authToken && isAuthenticated) {
      return;
    }

    if (urlRole && urlNetwork && authToken) {
      const normalizedRole = normalizeRole(urlRole);
      setRole(normalizedRole);
      setNetwork(urlNetwork);
      setIsAuthenticated(true);

      localStorage.setItem('knirv_user_role', normalizedRole);
      localStorage.setItem('knirv_network', urlNetwork);
      localStorage.setItem('knirv_auth_token', authToken);

      window.history.replaceState({}, document.title, window.location.pathname);
    } else if (urlRole || urlNetwork || authToken) {
      clearStoredAuth();
      console.warn('[RoleContext] Ignoring incomplete authentication callback');
    } else if (localStorage.getItem('knirv_demo_mode') === 'true') {
      setRole('Root');
      setNetwork('demo');
      setIsAuthenticated(true);
      console.log('[RoleContext] Demo mode enabled with Root access');
    } else if (typeof window !== 'undefined' && window.self !== window.top) {
      clearStoredAuth();
      // Running inside a parent frame (e.g. KNIRVSERVER WebGUI modal).
      // Request the parent's current auth credentials via postMessage.
      const onAuthResponse = (event) => {
        if (!event.data || event.data.type !== 'KNIRV_AUTH_RESPONSE') return;
        window.removeEventListener('message', onAuthResponse);
        applyAuthMessage(event.data);
      };
      window.addEventListener('message', onAuthResponse);
      window.parent.postMessage({ type: 'KNIRV_AUTH_REQUEST' }, '*');

      // Timeout: if the parent doesn't respond in 2 s, remove the listener
      // so the normal unauthenticated flow (redirect / auth screen) can run.
      setTimeout(() => window.removeEventListener('message', onAuthResponse), 2000);
    } else if (serverInfo && serverInfo.role) {
      clearStoredAuth();
      setRole(normalizeRole(serverInfo.role));
    }
  }, [serverInfo, isAuthenticated]);

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
