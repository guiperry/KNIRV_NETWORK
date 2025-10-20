"use client";

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

interface DHTContextType {
  isDHTEnabled: boolean;
  toggleDHT: () => void;
  setDHTEnabled: (enabled: boolean) => void;
}

const DHTContext = createContext<DHTContextType | undefined>(undefined);

interface DHTProviderProps {
  children: ReactNode;
}

export const DHTProvider: React.FC<DHTProviderProps> = ({ children }) => {
  const [isDHTEnabled, setIsDHTEnabled] = useState(false); // Default to disabled

  // Load DHT preference from localStorage on mount
  useEffect(() => {
    const savedDHTEnabled = localStorage.getItem('knirv-dht-enabled');
    if (savedDHTEnabled !== null) {
      try {
        setIsDHTEnabled(savedDHTEnabled === 'true');
      } catch (error) {
        // If there's any error parsing, default to false
        setIsDHTEnabled(false);
      }
    }
  }, []);

  // Save DHT preference to localStorage when it changes
  useEffect(() => {
    localStorage.setItem('knirv-dht-enabled', isDHTEnabled.toString());
  }, [isDHTEnabled]);

  const toggleDHT = () => {
    setIsDHTEnabled(prev => !prev);
  };

  const setDHTEnabled = (enabled: boolean) => {
    setIsDHTEnabled(enabled);
  };

  return (
    <DHTContext.Provider value={{ isDHTEnabled, toggleDHT, setDHTEnabled }}>
      {children}
    </DHTContext.Provider>
  );
};

export const useDHT = (): DHTContextType => {
  const context = useContext(DHTContext);
  if (context === undefined) {
    throw new Error('useDHT must be used within a DHTProvider');
  }
  return context;
};
