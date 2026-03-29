'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';
import { API_BASE_URL } from '@/lib/api';
import type { APIKeyEntry } from '@/components/onboarding/modals/APIKeysModal';
import type { MCPServerEntry } from '@/components/onboarding/modals/MCPServersModal';
import type { PolicyCert, CustomRule } from '@/components/onboarding/modals/PolicyCertsModal';

// New simplified onboarding flow
// Old flow: hero -> connect -> configure -> deploy -> dashboard
// New flow: guide (4 steps in one) -> hosting -> [local download or cloud signup]

export type NewOnboardingStep =
  | 'guide'           // Steps 1-4: Identity, Connections, Governance, Sovereignty
  | 'hosting';        // Final: Choose between local sovereign edition or cloud hosting

export type LegacyStep = 
  | 'hero' 
  | 'connect' 
  | 'configure' 
  | 'deploy' 
  | 'dashboard';

export type OnboardingStep = NewOnboardingStep | LegacyStep;

// Configuration objects
export interface AppConfig {
  url: string;
  name: string;
  type: string;
  apiSpec?: Record<string, unknown>;
  endpoints?: Record<string, string>;
  authMethods?: string[];
  corsRestricted?: boolean;
  [key: string]: unknown;
}

export interface DataWalletConfig {
  walletName: string;
  fabricInputs: string[];
  guardrails: {
    networkDrift: boolean;
    filesystemAccess: boolean;
    computeCostCap: boolean;
  };
  connectionData: {
    apiKeys: APIKeyEntry[];
    mcpServers: MCPServerEntry[];
    policyCerts: PolicyCert[];
    customRules: CustomRule[];
  };
  completedConnections: string[];
}

export interface PrivacyPreferences {
  dataEncryption: boolean;
  localProcessing: boolean;
  anonymizeMetrics: boolean;
  shareErrorLogs: boolean;
  allowAnalytics: boolean;
  dataRetentionDays: number;
  autoDeleteInactive: boolean;
  thirdPartyIntegrations: boolean;
}

export interface CloudCortexConfig {
  instanceId: string;
  region: string;
  status: 'active' | 'inactive' | 'error';
  createdAt: string;
}

interface OnboardingState {
  // Flow control
  currentStep: OnboardingStep;
  isOnboardingComplete: boolean;
  
  // User configuration
  dataWalletConfig: DataWalletConfig | null;
  privacyPreferences: PrivacyPreferences | null;
  cloudCortexConfig: CloudCortexConfig | null;
  
  // Verification state
  isEmailVerified: boolean;
  isDeviceVerified: boolean;
  
  // Connection data (for backward compatibility during transition)
  connectionData: {
    apiKeys: APIKeyEntry[];
    mcpServers: MCPServerEntry[];
    policyCerts: PolicyCert[];
    customRules: CustomRule[];
  };
  completedConnections: string[];
  
  // Legacy support (for migration/compatibility)
  connectedApp: {url: string, name: string, type: string} | null;
  appConfig: Record<string, unknown> | null;
  modelConfig: Record<string, unknown> | null;
  deploymentConfig: Record<string, unknown> | null;
}

interface OnboardingContextType {
  state: OnboardingState;
  updateState: (updates: Partial<OnboardingState>) => void;
  resetOnboarding: () => void;
  saveProgress: () => void;
  loadProgress: () => boolean;
  
  // Helper methods
  goToStep: (step: OnboardingStep) => void;
  completeOnboarding: () => Promise<void>;
  
  // Legacy compatibility
  isLegacyFlow: () => boolean;
}

const defaultState: OnboardingState = {
  currentStep: 'hero',
  isOnboardingComplete: false,
  
  dataWalletConfig: null,
  privacyPreferences: null,
  cloudCortexConfig: null,
  
  isEmailVerified: false,
  isDeviceVerified: false,
  
  connectionData: {
    apiKeys: [],
    mcpServers: [],
    policyCerts: [],
    customRules: []
  },
  completedConnections: [],
  
  // Legacy defaults
  connectedApp: null,
  appConfig: null,
  modelConfig: null,
  deploymentConfig: null,
};

const OnboardingContext = createContext<OnboardingContextType | undefined>(undefined);

export const OnboardingProvider: React.FC<{children: React.ReactNode}> = ({ children }) => {
  const [state, setState] = useState<OnboardingState>(defaultState);

  // Load saved progress on initial mount
  useEffect(() => {
    const hasProgress = loadProgress();
    if (!hasProgress) {
      // First time user - initialize with defaults
      setState(defaultState);
    }
  }, []);

  const updateState = (updates: Partial<OnboardingState>) => {
    setState(prev => ({
      ...prev,
      ...updates
    }));
  };

  const resetOnboarding = () => {
    localStorage.removeItem('onboardingState');
    localStorage.removeItem('knirvOnboardingV2');
    setState(defaultState);
  };

  const saveProgress = async () => {
    try {
      // Save to localStorage for quick access
      localStorage.setItem('knirvOnboardingV2', JSON.stringify(state));
      localStorage.setItem('onboardingState', JSON.stringify(state));
      
      // Also save to backend database per user
      const token = localStorage.getItem('knirv_nexus_token');
      if (token) {
        try {
          const response = await fetch(`${API_BASE_URL}/api/auth/preferences`, {
            method: 'PUT',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${token}`,
            },
            body: JSON.stringify({
              onboarding_data: state,
            }),
          });
          if (response.ok) {
            console.log('Onboarding data saved to database');
          }
        } catch (dbError) {
          console.warn('Failed to save onboarding to database, using localStorage only:', dbError);
        }
      }
    } catch (e) {
      console.error('Failed to save onboarding state', e);
    }
  };

  const loadProgress = (): boolean => {
    try {
      // Try to load from backend database first (for cross-device persistence)
      const token = localStorage.getItem('knirv_nexus_token');
      let loadedFromDB = false;
      
      // We'll try to load from DB in the background but also check localStorage
      if (token) {
        fetch(`${API_BASE_URL}/api/auth/preferences`, {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${token}`,
          },
        })
        .then(response => {
          if (response.ok) {
            return response.json();
          }
          throw new Error('Failed to load preferences');
        })
        .then(data => {
          const saved = data.onboarding_data ?? data.onboardingData;
          if (saved) {
            setState(prev => ({ ...prev, ...saved }));
            // Also update localStorage
            localStorage.setItem('knirvOnboardingV2', JSON.stringify({ ...state, ...saved }));
          }
        })
        .catch(err => console.warn('Could not load from DB, using localStorage:', err));
      }
      
      // Try new format first from localStorage
      const savedV2 = localStorage.getItem('knirvOnboardingV2');
      if (savedV2) {
        const parsed = JSON.parse(savedV2);
        setState(prev => ({ ...prev, ...parsed }));
        return true;
      }
      
      // Fall back to old format
      const saved = localStorage.getItem('onboardingState');
      if (saved) {
        const parsed = JSON.parse(saved);
        // Migrate old state to new format if needed
        const migratedState = migrateOldState(parsed);
        setState(prev => ({ ...prev, ...migratedState }));
        return true;
      }
    } catch (e) {
      console.error('Failed to load onboarding state', e);
    }
    return false;
  };

  // Migrate old state format to new format
  const migrateOldState = (oldState: Record<string, unknown>): Partial<OnboardingState> => {
    // If old state has 'hero' or 'connect' steps, start fresh with new flow
    const oldStep = oldState.currentStep as string;
    if (['hero', 'connect', 'configure', 'deploy'].includes(oldStep)) {
      return {
        currentStep: 'guide',
        isOnboardingComplete: false
      };
    }
    return oldState as Partial<OnboardingState>;
  };

  const goToStep = (step: OnboardingStep) => {
    updateState({ currentStep: step });
  };

  const completeOnboarding = async () => {
    updateState({
      isOnboardingComplete: true,
      currentStep: 'hosting'
    });

    // Persist onboarding data to ICME for policy enforcement guardrails (Gap 11)
    const token = localStorage.getItem('knirv_nexus_token');
    const userStr = localStorage.getItem('knirv_nexus_user');
    const userId = userStr ? JSON.parse(userStr)?.id ?? '' : '';

    if (state.dataWalletConfig) {
      try {
        await fetch(`${API_BASE_URL}/api/icme/onboarding/complete`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...(token && { 'Authorization': `Bearer ${token}` }),
          },
          body: JSON.stringify({
            user_id: userId,
            wallet_name: state.dataWalletConfig.walletName,
            fabric_inputs: state.dataWalletConfig.fabricInputs,
            guardrails: state.dataWalletConfig.guardrails,
            connection_data: state.dataWalletConfig.connectionData,
            privacy_settings: state.privacyPreferences ?? {},
            commit_to_chain: false,
          }),
        });
      } catch (err) {
        console.warn('ICME onboarding commit failed (non-blocking):', err);
      }
    }
  };

  const isLegacyFlow = () => {
    return ['hero', 'connect', 'configure', 'deploy'].includes(state.currentStep);
  };

  // Auto-save on state changes
  useEffect(() => {
    saveProgress();
  }, [state]);

  return (
    <OnboardingContext.Provider value={{ 
      state, 
      updateState, 
      resetOnboarding,
      saveProgress,
      loadProgress,
      goToStep,
      completeOnboarding,
      isLegacyFlow
    }}>
      {children}
    </OnboardingContext.Provider>
  );
};

export const useOnboarding = () => {
  const context = useContext(OnboardingContext);
  if (context === undefined) {
    throw new Error('useOnboarding must be used within an OnboardingProvider');
  }
  return context;
};

export default OnboardingContext;
