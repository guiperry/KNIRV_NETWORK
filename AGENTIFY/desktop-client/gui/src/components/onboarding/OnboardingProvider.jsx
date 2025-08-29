import React, { createContext, useContext, useState, useEffect } from 'react';

const OnboardingContext = createContext();

export const useOnboarding = () => {
  const context = useContext(OnboardingContext);
  if (!context) {
    throw new Error('useOnboarding must be used within an OnboardingProvider');
  }
  return context;
};

export const OnboardingProvider = ({ children }) => {
  const [isOnboardingActive, setIsOnboardingActive] = useState(false);
  const [currentStep, setCurrentStep] = useState(0);
  const [isAutoMode, setIsAutoMode] = useState(false);
  const [hasCompletedOnboarding, setHasCompletedOnboarding] = useState(false);

  // Check if user has completed onboarding before
  useEffect(() => {
    const completed = localStorage.getItem('knirv-engine-onboarding-completed');
    setHasCompletedOnboarding(completed === 'true');
  }, []);

  const startOnboarding = (autoMode = false) => {
    setIsOnboardingActive(true);
    setCurrentStep(0);
    setIsAutoMode(autoMode);
  };

  const nextStep = () => {
    setCurrentStep(prev => prev + 1);
  };

  const previousStep = () => {
    setCurrentStep(prev => Math.max(0, prev - 1));
  };

  const skipOnboarding = () => {
    setIsOnboardingActive(false);
    setCurrentStep(0);
    localStorage.setItem('knirv-engine-onboarding-completed', 'true');
    setHasCompletedOnboarding(true);
  };

  const completeOnboarding = () => {
    setIsOnboardingActive(false);
    setCurrentStep(0);
    localStorage.setItem('knirv-engine-onboarding-completed', 'true');
    setHasCompletedOnboarding(true);
  };

  const restartOnboarding = (autoMode = false) => {
    localStorage.removeItem('knirv-engine-onboarding-completed');
    setHasCompletedOnboarding(false);
    startOnboarding(autoMode);
  };

  const value = {
    isOnboardingActive,
    currentStep,
    isAutoMode,
    hasCompletedOnboarding,
    startOnboarding,
    nextStep,
    previousStep,
    skipOnboarding,
    completeOnboarding,
    restartOnboarding,
    setIsAutoMode,
  };

  return (
    <OnboardingContext.Provider value={value}>
      {children}
    </OnboardingContext.Provider>
  );
};
