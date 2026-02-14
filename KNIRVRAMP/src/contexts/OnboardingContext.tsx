'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';

// New simplified onboarding flow
// Old flow: hero -> connect -> configure -> deploy -> dashboard
// New flow: guide -> preferences -> welcome -> verification -> cortex

export type NewOnboardingStep = 
  | 'guide'           // Step 1-4: Animated guide for data wallet setup
  | 'preferences'     // Step 5: Private Data Management Preferences form
  | 'welcome'         // Welcome page with QR code for mobile app download
  | 'verification'    // QR code verification modal
  | 'cortex';         // Final: Cloud Cortex info card/dashboard

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
  completeOnboarding: () => void;
  
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

  const saveProgress = () => {
    try {
      localStorage.setItem('knirvOnboardingV2', JSON.stringify(state));
      // Also save to old key for backward compatibility
      localStorage.setItem('onboardingState', JSON.stringify(state));
    } catch (e) {
      console.error('Failed to save onboarding state', e);
    }
  };

  const loadProgress = (): boolean => {
    try {
      // Try new format first
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

  const completeOnboarding = () => {
    updateState({ 
      isOnboardingComplete: true,
      currentStep: 'cortex'
    });
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

// eslint-disable-next-line react-refresh/only-export-components
export const useOnboarding = () => {
  const context = useContext(OnboardingContext);
  if (context === undefined) {
    throw new Error('useOnboarding must be used within an OnboardingProvider');
  }
  return context;
};

export default OnboardingContext;
