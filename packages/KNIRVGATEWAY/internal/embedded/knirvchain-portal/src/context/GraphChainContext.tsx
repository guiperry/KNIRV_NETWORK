import React, { createContext, useContext, useEffect, useState } from 'react';
import { graphChainApi, GraphNode, SkillNode, ErrorNode } from '../services/api';

interface GraphChainContextType {
  currentDensity: number;
  isLoading: boolean;
  error: string | null;
  refreshData: () => void;
  getNode: (nodeId: string) => Promise<GraphNode | null>;
  getSkills: () => Promise<SkillNode[]>;
  getErrors: () => Promise<ErrorNode[]>;
  createSkill: (skill: Partial<SkillNode>) => Promise<boolean>;
  createError: (error: Partial<ErrorNode>) => Promise<boolean>;
}

const GraphChainContext = createContext<GraphChainContextType | undefined>(undefined);

// eslint-disable-next-line react-refresh/only-export-components
export const useGraphChain = () => {
  const context = useContext(GraphChainContext);
  if (context === undefined) {
    throw new Error('useGraphChain must be used within a GraphChainProvider');
  }
  return context;
};

interface GraphChainProviderProps {
  children: React.ReactNode;
}

export const GraphChainProvider: React.FC<GraphChainProviderProps> = ({ children }) => {
  const [currentDensity, setCurrentDensity] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchCurrentDensity = async () => {
    try {
      setError(null);
      const density = await graphChainApi.getCurrentDensity();
      setCurrentDensity(density);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch GraphChain density');
    } finally {
      setIsLoading(false);
    }
  };

  const getNode = async (nodeId: string): Promise<GraphNode | null> => {
    try {
      return await graphChainApi.getNode(nodeId);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch node');
      return null;
    }
  };

  const getSkills = async (): Promise<SkillNode[]> => {
    try {
      return await graphChainApi.getAllSkills();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch skills');
      return [];
    }
  };

  const getErrors = async (): Promise<ErrorNode[]> => {
    try {
      return await graphChainApi.getAllErrors();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch errors');
      return [];
    }
  };

  const createSkill = async (skill: Partial<SkillNode>): Promise<boolean> => {
    try {
      await graphChainApi.createSkill(skill);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create skill');
      return false;
    }
  };

  const createError = async (error: Partial<ErrorNode>): Promise<boolean> => {
    try {
      await graphChainApi.createError(error);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create error');
      return false;
    }
  };

  const refreshData = () => {
    setIsLoading(true);
    fetchCurrentDensity();
  };

  useEffect(() => {
    fetchCurrentDensity();

    // Auto-refresh every 10 seconds (less frequent than blockchain)
    const interval = setInterval(fetchCurrentDensity, 10000);
    return () => clearInterval(interval);
  }, []);

  const value: GraphChainContextType = {
    currentDensity,
    isLoading,
    error,
    refreshData,
    getNode,
    getSkills,
    getErrors,
    createSkill,
    createError,
  };

  return (
    <GraphChainContext.Provider value={value}>
      {children}
    </GraphChainContext.Provider>
  );
};
