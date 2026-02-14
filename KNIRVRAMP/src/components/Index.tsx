'use client';

import React, { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button";
import { Network } from "lucide-react";

// New Onboarding Components
import OnboardingGuide from "@/components/OnboardingGuide";
import DataPreferencesForm from "@/components/DataPreferencesForm";
import WelcomePage from "@/components/WelcomePage";
import VerificationQRModal from "@/components/VerificationQRModal";
import CloudCortexInfoCard from "@/components/CloudCortexInfoCard";

// Legacy Components (for backward compatibility)
import Hero from "@/components/Hero";
import AppConnector from "@/components/AppConnector";
import ModelConfig from "@/components/ModelConfig";
import ModelDeployer from "@/components/ModelDeployer";
import Dashboard from "@/components/Dashboard";

// Shared Components
import StepIndicator from "@/components/StepIndicator";
import LoadingScreen from "@/components/LoadingScreen";
import LoginModal from "@/components/LoginModal";

// Contexts
import { useOnboarding, DataWalletConfig, PrivacyPreferences, OnboardingStep } from "@/contexts/OnboardingContext";
import { useAuth } from "@/contexts/AuthContext";
import { useToast } from "@/hooks/use-toast";
import { ModelConfiguration } from "@/lib/cortex-compiler/CortexModelCompiler";

const Index = () => {
  const { state, updateState, saveProgress, resetOnboarding, goToStep } = useOnboarding();
  const { isAuthenticated, user, logout } = useAuth();
  const { toast } = useToast();
  
  // UI State
  const [loginModalOpen, setLoginModalOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [loadingProgress, setLoadingProgress] = useState(0);
  const [verificationModalOpen, setVerificationModalOpen] = useState(false);
  
  // Legacy state support
  const [dashboardIsActive, setDashboardIsActive] = useState(true);
  const [downloadModalOpen, setDownloadModalOpen] = useState(false);
  const [settingsModalOpen, setSettingsModalOpen] = useState(false);

  // Extract current step
  const { currentStep } = state;

  // Initial loading effect - simple and reliable
  useEffect(() => {
    const timer = setTimeout(() => {
      setIsLoading(false);
      setLoadingProgress(100);
    }, 2000); // Show loading for 2 seconds

    // Progress animation
    const progressInterval = setInterval(() => {
      setLoadingProgress(prev => {
        if (prev >= 90) return prev;
        return prev + 10;
      });
    }, 150);

    return () => {
      clearTimeout(timer);
      clearInterval(progressInterval);
    };
  }, []);

  // Save progress whenever state changes
  useEffect(() => {
    saveProgress();
  }, [state, saveProgress]);

  // ============ NEW ONBOARDING FLOW HANDLERS ============

  // Step 1-4: Onboarding Guide Complete
  const handleGuideComplete = (walletConfig: DataWalletConfig) => {
    updateState({ 
      dataWalletConfig: walletConfig,
      currentStep: 'preferences'
    });
    toast({
      title: "Setup Guide Complete",
      description: "Now let's configure your privacy preferences",
    });
  };

  // Step 5: Data Preferences Form Submit
  const handlePreferencesSubmit = (preferences: PrivacyPreferences) => {
    updateState({ 
      privacyPreferences: preferences,
      isEmailVerified: true, // Simulated - in real app, wait for email click
      currentStep: 'welcome'
    });
    toast({
      title: "Preferences Saved",
      description: "Confirmation email sent. Please check your inbox.",
    });
  };

  const handlePreferencesBack = () => {
    goToStep('guide');
  };

  const handleWelcomeBack = () => {
    goToStep('preferences');
  };

  // Welcome Page: Continue to Verification
  const handleWelcomeContinue = () => {
    setVerificationModalOpen(true);
  };

  // Verification Complete
  const handleVerificationComplete = () => {
    setVerificationModalOpen(false);
    updateState({ 
      isDeviceVerified: true,
      currentStep: 'cortex'
    });
    toast({
      title: "Device Verified",
      description: "Welcome to your Cloud Cortex!",
    });
  };

  // ============ LEGACY FLOW HANDLERS ============

  const handleReset = () => {
    resetOnboarding();
  };

  const handleAppConnect = (appData: {url: string, name: string, type: string}) => {
    updateState({
      connectedApp: appData,
      currentStep: 'configure'
    });
  };

  const handleModelConfigured = (config: ModelConfiguration) => {
    updateState({
      modelConfig: config as unknown as Record<string, unknown>,
      currentStep: 'deploy'
    });
  };

  const handleModelDeployed = () => {
    updateState({
      currentStep: 'dashboard'
    });
  };

  const goBack = () => {
    const stepMap: Record<string, OnboardingStep> = {
      'configure': 'hero',
      'connect': 'configure',
      'deploy': 'configure',
      'dashboard': 'deploy'
    };
    
    if (stepMap[currentStep]) {
      goToStep(stepMap[currentStep]);
    }
  };

  // ============ RENDER HELPERS ============

  const isLegacyStep = ['hero', 'connect', 'configure', 'deploy', 'dashboard'].includes(currentStep);

  return (
    <div className="min-h-screen bg-[#0a0a0c]">
      {/* Full-screen loading overlay */}
      {isLoading && (
        <LoadingScreen
          isVisible={true}
          message="Initializing KNIRV Data Vault..."
          progress={loadingProgress}
        />
      )}
      
      {/* Navigation - Only show for legacy flow or when needed */}
      {isLegacyStep && (
        <nav className="border-b border-white/10 bg-black/40 backdrop-blur-md relative z-50">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between items-center h-16">
              {/* Left: Logo */}
              <div className="flex items-center space-x-4">
                <div 
                  className="flex items-center space-x-2 cursor-pointer hover:opacity-80 transition-opacity"
                  onClick={handleReset}
                  title="Return to home screen"
                >
                  <Network className="h-8 w-8 text-blue-500" />
                  <span className="text-2xl font-bold text-white">
                    KNIRV
                  </span>
                  <span className="text-sm font-light text-gray-400 tracking-wider hidden sm:block">
                    Data Fabric
                  </span>
                </div>


              </div>

              {/* Center: Step Indicator */}
              <div className="flex-1 flex justify-center">
                <StepIndicator currentStep={currentStep} />
              </div>

              {/* Right: Empty space for balance */}
              <div className="flex items-center space-x-4">
              </div>
            </div>
          </div>
        </nav>
      )}

      {/* Main Content - New Onboarding Flow */}
      <main className="relative">
        {/* Step 1-4: Onboarding Guide */}
        {currentStep === 'guide' && (
          <OnboardingGuide 
            onComplete={(walletConfig: DataWalletConfig) => handleGuideComplete(walletConfig)}
            onReset={handleReset}
          />
        )}

        {/* Step 5: Data Preferences Form */}
        {currentStep === 'preferences' && (
          <DataPreferencesForm 
            onSubmit={() => handlePreferencesSubmit({
              dataEncryption: true,
              localProcessing: true,
              anonymizeMetrics: true,
              shareErrorLogs: false,
              allowAnalytics: false,
              dataRetentionDays: 90,
              autoDeleteInactive: true,
              thirdPartyIntegrations: false
            })}
            onBack={handlePreferencesBack}
            onReset={handleReset}
          />
        )}

        {/* Step 6: Welcome Page with QR Code */}
        {currentStep === 'welcome' && (
          <WelcomePage 
            onContinue={handleWelcomeContinue}
            onBack={handleWelcomeBack}
            onReset={handleReset}
          />
        )}

        {/* Step 7: Cloud Cortex Info Card (Final) */}
        {currentStep === 'cortex' && (
          <CloudCortexInfoCard onReset={handleReset} />
        )}

        {/* ============ LEGACY FLOW (for backward compatibility) ============ */}
        
        {currentStep === 'hero' && (
          <Hero onGetStarted={() => goToStep('guide')} />
        )}
        
        {currentStep === 'connect' && (
          <AppConnector onAppConnect={handleAppConnect} />
        )}
        
        {currentStep === 'configure' && (
          <ModelConfig
            connectedApp={state.connectedApp}
            onConfigured={handleModelConfigured}
            onConnectGitHub={() => goToStep('connect' as const)}
            isActive={dashboardIsActive}
            downloadModalOpen={downloadModalOpen}
            setDownloadModalOpen={setDownloadModalOpen}
            onDownload={(platform) => {
              window.open(`/api/download/engine/${platform}`, '_blank');
              setDownloadModalOpen(false);
            }}
            settingsModalOpen={settingsModalOpen}
            setSettingsModalOpen={setSettingsModalOpen}
            goBack={goBack}
            onReset={handleReset}
          />
        )}
        
        {currentStep === 'deploy' && state.modelConfig && (
          <ModelDeployer
            modelConfig={state.modelConfig as unknown as ModelConfiguration}
            onDeployed={handleModelDeployed}
            onConnectToTargets={() => goToStep('dashboard')}
            onReset={handleReset}
          />
        )}
        
        {currentStep === 'dashboard' && (
          <Dashboard
            connectedApp={state.connectedApp}
            isActive={dashboardIsActive}
            setIsActive={setDashboardIsActive}
            settingsModalOpen={settingsModalOpen}
            setSettingsModalOpen={setSettingsModalOpen}
            onConnectApp={(appData) => {
              updateState({ connectedApp: appData });
            }}
            modelConfig={state.modelConfig as unknown as ModelConfiguration}
            onReset={handleReset}
          />
        )}
      </main>

      {/* Verification QR Modal */}
      <VerificationQRModal
        isOpen={verificationModalOpen}
        onClose={() => setVerificationModalOpen(false)}
        onVerified={handleVerificationComplete}
      />

      {/* Login Modal */}
      <LoginModal
        open={loginModalOpen}
        onOpenChange={setLoginModalOpen}
        onLoginSuccess={() => {
          toast({
            title: "Login Successful",
            description: "Welcome back to KNIRV!",
          });
        }}
      />
    </div>
  );
};

export default Index;
