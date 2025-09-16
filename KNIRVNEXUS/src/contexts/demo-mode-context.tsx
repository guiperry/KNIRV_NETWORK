"use client";

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

interface DemoData {
  nodes: Array<{
    id: string;
    name: string;
    status: string;
    capabilities: string[];
    stake: number;
    location: string;
  }>;
  tasks: Array<{
    id: string;
    type: string;
    status: string;
    priority: number;
  }>;
  metrics: {
    totalNodes: number;
    activeNodes: number;
    totalStake: number;
    networkHealth: number;
  };
}

interface DemoModeContextType {
  isDemoMode: boolean;
  toggleDemoMode: () => void;
  setDemoMode: (enabled: boolean) => void;
  demoData: DemoData | {};
}

const DemoModeContext = createContext<DemoModeContextType | undefined>(undefined);

interface DemoModeProviderProps {
  children: ReactNode;
}

const generateDemoData = (): DemoData => ({
  nodes: [
    {
      id: 'demo-node-1',
      name: 'Demo Node 1',
      status: 'active',
      capabilities: ['compute', 'storage'],
      stake: 1000,
      location: 'US-East'
    },
    {
      id: 'demo-node-2',
      name: 'Demo Node 2',
      status: 'active',
      capabilities: ['compute'],
      stake: 500,
      location: 'EU-West'
    }
  ],
  tasks: [
    {
      id: 'demo-task-1',
      type: 'validation',
      status: 'pending',
      priority: 5
    },
    {
      id: 'demo-task-2',
      type: 'computation',
      status: 'running',
      priority: 3
    }
  ],
  metrics: {
    totalNodes: 2,
    activeNodes: 2,
    totalStake: 1500,
    networkHealth: 95
  }
});

export const DemoModeProvider: React.FC<DemoModeProviderProps> = ({ children }) => {
  const [isDemoMode, setIsDemoMode] = useState(true); // Default to demo mode

  // Load demo mode preference from localStorage on mount
  useEffect(() => {
    const savedDemoMode = localStorage.getItem('knirv-demo-mode');
    if (savedDemoMode !== null) {
      try {
        setIsDemoMode(savedDemoMode === 'true');
      } catch (error) {
        // If there's any error parsing, default to true
        setIsDemoMode(true);
      }
    }
  }, []);

  // Save demo mode preference to localStorage when it changes
  useEffect(() => {
    localStorage.setItem('knirv-demo-mode', isDemoMode.toString());
  }, [isDemoMode]);

  const toggleDemoMode = () => {
    setIsDemoMode(prev => !prev);
  };

  const setDemoMode = (enabled: boolean) => {
    setIsDemoMode(enabled);
  };

  const demoData = isDemoMode ? generateDemoData() : {};

  return (
    <DemoModeContext.Provider value={{ isDemoMode, toggleDemoMode, setDemoMode, demoData }}>
      {children}
    </DemoModeContext.Provider>
  );
};

export const useDemoMode = (): DemoModeContextType => {
  const context = useContext(DemoModeContext);
  if (context === undefined) {
    throw new Error('useDemoMode must be used within a DemoModeProvider');
  }
  return context;
};
