import React, { createContext, useContext, useState, useEffect } from 'react';
import { useBackend } from './BackendContext';

const RoleContext = createContext();

export const RoleProvider = ({ children }) => {
  const { serverInfo, isRunning } = useBackend();
  const [role, setRole] = useState('Client'); // Default to most restricted role
  
  useEffect(() => {
    if (serverInfo && serverInfo.role) {
      setRole(serverInfo.role);
    }
  }, [serverInfo]);
  
  // Define page access by role
  const pageAccess = {
    Root: [
      'dashboard', 'inventory', 'vault', 'blockchain', 'dex', 'daos', 
      'nft-vault', 'nft-capability-manager', 'add-capability', 
      'devs', 'settlement', 'network-admin'
    ],
    Bootnode: [
      'dashboard', 'inventory', 'vault', 'blockchain', 'dex', 'daos', 
      'nft-vault', 'nft-capability-manager', 'add-capability', 
      'devs', 'settlement'
    ],
    Peer: [
      'dashboard', 'inventory', 'vault', 'blockchain', 'dex',
      'nft-vault', 'nft-capability-manager', 'add-capability', 
      'devs'
    ],
    Client: [
      'inventory', 'dex', 'nft-capability-manager'
    ]
  };
  
  // Check if a page is accessible for the current role
  const canAccess = (page) => {
    const accessList = pageAccess[role] || pageAccess.Client;
    return accessList.includes(page);
  };
  
  return (
    <RoleContext.Provider value={{ role, canAccess }}>
      {children}
    </RoleContext.Provider>
  );
};

export const useRole = () => useContext(RoleContext);