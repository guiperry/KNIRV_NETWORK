'use client';

import React, { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button";
import { Network } from "lucide-react";
import Hero from "@/components/Hero";
import AppConnector from "@/components/AppConnector";
import ModelConfig from "@/components/ModelConfig";
import ModelDeployer from "@/components/ModelDeployer";
import Dashboard from "@/components/Dashboard";
import StepIndicator from "@/components/StepIndicator";
import LoadingScreen from "@/components/LoadingScreen";
import LoginModal from "@/components/LoginModal";
import { useOnboarding } from "@/contexts/OnboardingContext";
import { useAuth } from "@/contexts/AuthContext";
import { useToast } from "@/hooks/use-toast";
import { ModelConfiguration } from "@/lib/cortex-compiler/CortexModelCompiler";

const Index = () => {
  const { state, updateState, saveProgress } = useOnboarding();
  const { isAuthenticated, user, logout } = useAuth();
  const { toast } = useToast();
  const [loginModalOpen, setLoginModalOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [loadingProgress, setLoadingProgress] = useState(0);

  // State for dashboard actions (status, download, settings)
  const [dashboardIsActive, setDashboardIsActive] = useState(true);
  const [downloadModalOpen, setDownloadModalOpen] = useState(false);
  const [settingsModalOpen, setSettingsModalOpen] = useState(false);
  
  // Extract values from the onboarding context
  const { currentStep, connectedApp, modelConfig } = state;

  // Initial loading effect
  useEffect(() => {
    // Simulate loading process
    const loadingInterval = setInterval(() => {
      setLoadingProgress(prev => {
        if (prev >= 100) {
          clearInterval(loadingInterval);
          setTimeout(() => setIsLoading(false), 500); // Small delay after reaching 100%
          return 100;
        }
        return prev + 10;
      });
    }, 300);

    return () => clearInterval(loadingInterval);
  }, []);

  // Save progress whenever state changes
  useEffect(() => {
    saveProgress();
  }, [state, saveProgress]);

  const handleAppConnect = (appData: {url: string, name: string, type: string}) => {
    updateState({
      connectedApp: appData,
      currentStep: 'configure'
    });
  };

  // Update to store model config and move to deploy step
  const handleModelConfigured = (config: ModelConfiguration) => {
    updateState({
      modelConfig: config,
      currentStep: 'deploy'
    });
  };

  // Add handler for deployment completion
  const handleModelDeployed = () => {
    updateState({
      currentStep: 'dashboard'
    });
  };

  const goBack = () => {
    let newStep: 'hero' | 'connect' | 'configure' | 'deploy' | 'dashboard' = currentStep;

    if (currentStep === 'configure') newStep = 'hero';
    if (currentStep === 'connect') newStep = 'configure';
    if (currentStep === 'deploy') newStep = 'configure';
    if (currentStep === 'dashboard') newStep = 'deploy';

    updateState({ currentStep: newStep });
  };

  return (
    <div className="bg-gradient-to-br from-slate-900 via-slate-800 to-black">
      {/* Full-screen loading overlay */}
      {isLoading && (
        <LoadingScreen
          isVisible={true}
          message="Starting KNIRV Cortex Builder..."
          progress={loadingProgress}
        />
      )}
      
      {/* Navigation */}
      <nav className="border-b border-white/10 bg-black/20 backdrop-blur-lg">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            {/* Left: Logo and left-side buttons */}
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2">
                <Network className="h-8 w-8 knirv-text-primary" />
                <span className="text-2xl font-bold knirv-gradient-text">
                  KNIRV
                </span>
                <span className="text-sm font-light text-gray-400 tracking-wider">
                  CORTEX
                </span>
              </div>
              {/* Back then Sign In button */}
              {currentStep !== 'hero' && (
                <Button 
                  variant="ghost" 
                  onClick={goBack} 
                  className="text-white/70 hover:text-white hover:bg-white/10"
                >
                  ← Back
                </Button>
              )}
              {isAuthenticated && user ? (
                <div className="flex items-center space-x-2">
                  <div className="flex items-center space-x-2 text-white/70">
                    {user.picture && (
                      <img
                        src={user.picture}
                        alt={user.name}
                        className="w-6 h-6 rounded-full"
                      />
                    )}
                    <span className="text-sm">Hello, {user.name}</span>
                  </div>
                  <Button
                    variant="ghost"
                    onClick={logout}
                    className="text-white/70 hover:text-white hover:bg-white/10 text-sm"
                  >
                    Logout
                  </Button>
                </div>
              ) : (
                <div className="flex items-center space-x-3">
                  <div className="bg-amber-500/20 border border-amber-500/30 rounded-full px-3 py-1">
                    <span className="text-amber-300 text-xs font-medium">Demo Mode</span>
                  </div>
                  <Button
                    variant="outline"
                    onClick={() => setLoginModalOpen(true)}
                    className="knirv-border-primary knirv-text-primary hover:bg-blue-400/10 hover:knirv-text-secondary"
                  >
                    Sign In
                  </Button>
                </div>
              )}
            </div>

            {/* Center: Step Indicator */}
            <div className="flex-1 flex justify-center">
              <StepIndicator currentStep={currentStep} />
            </div>

            {/* Right: Empty space for balance */}
            <div className="flex items-center space-x-4">
              {/* This space can be used for additional navigation items */}
            </div>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="relative">
        {currentStep === 'hero' && (
          <Hero onGetStarted={() => updateState({ currentStep: 'configure' })} />
        )}
        
        {currentStep === 'connect' && (
          <AppConnector onAppConnect={handleAppConnect} />
        )}
        
        {currentStep === 'configure' && (
          <ModelConfig
            connectedApp={connectedApp}
            onConfigured={handleModelConfigured}
            onConnectGitHub={() => updateState({ currentStep: 'connect' })}
            isActive={dashboardIsActive}
            downloadModalOpen={downloadModalOpen}
            setDownloadModalOpen={setDownloadModalOpen}
            onDownload={(platform) => {
              console.log(`Downloading KNIRV Engine for ${platform}`);
              // Trigger download via API
              window.open(`/api/download/engine/${platform}`, '_blank');
              setDownloadModalOpen(false);
            }}
            settingsModalOpen={settingsModalOpen}
            setSettingsModalOpen={setSettingsModalOpen}
          />
        )}
        
        {/* Deploy Step - Full ModelDeployer component */}
        {currentStep === 'deploy' && connectedApp && modelConfig && (
          <ModelDeployer
            modelConfig={modelConfig}
            onDeployed={handleModelDeployed}
          />
        )}
        
        {currentStep === 'dashboard' && connectedApp && (
          <Dashboard 
            connectedApp={connectedApp}
            isActive={dashboardIsActive}
            setIsActive={setDashboardIsActive}
            settingsModalOpen={settingsModalOpen}
            setSettingsModalOpen={setSettingsModalOpen}
          />
        )}
      </main>

      {/* Login Modal */}
      <LoginModal
        open={loginModalOpen}
        onOpenChange={setLoginModalOpen}
        onLoginSuccess={() => {
          // Handle any post-login actions if needed
          toast({
            title: "Login Successful",
            description: "Welcome back to KNIRV Cortex Builder!",
          });
        }}
      />
    </div>
  );
};

export default Index;
