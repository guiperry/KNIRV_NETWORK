import React, { useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useOnboarding } from './OnboardingProvider';
import OnboardingModal from './OnboardingModal';
import { onboardingSteps, executeStepAction } from './onboardingSteps';

const OnboardingSequence = () => {
  const navigate = useNavigate();
  const {
    isOnboardingActive,
    currentStep,
    isAutoMode,
    nextStep,
    previousStep,
    skipOnboarding,
    completeOnboarding,
    setIsAutoMode,
  } = useOnboarding();

  const currentStepData = onboardingSteps[currentStep];
  const totalSteps = onboardingSteps.length;

  // Auto-advance logic
  useEffect(() => {
    if (!isOnboardingActive || !isAutoMode || !currentStepData) return;

    const timer = setTimeout(() => {
      if (currentStep < totalSteps - 1) {
        handleNext();
      } else {
        completeOnboarding();
      }
    }, currentStepData.autoDelay || 4000);

    return () => clearTimeout(timer);
  }, [isOnboardingActive, isAutoMode, currentStep, currentStepData, totalSteps]);

  // Execute step actions
  useEffect(() => {
    if (!isOnboardingActive || !currentStepData?.action) return;

    const timer = setTimeout(() => {
      try {
        executeStepAction(currentStepData.action, navigate);
      } catch (error) {
        console.error('Error executing onboarding step action:', error);
        // Continue to next step instead of breaking the sequence
        handleNext();
      }
    }, 500); // Increased delay to ensure modal is positioned and previous highlights are cleaned up

    return () => clearTimeout(timer);
  }, [currentStep, isOnboardingActive, currentStepData, navigate]);

  const handleNext = useCallback(() => {
    if (currentStep < totalSteps - 1) {
      nextStep();
    } else {
      completeOnboarding();
    }
  }, [currentStep, totalSteps, nextStep, completeOnboarding]);

  const handlePrevious = useCallback(() => {
    if (currentStep > 0) {
      previousStep();
    }
  }, [currentStep, previousStep]);

  const handleToggleAutoMode = useCallback(() => {
    setIsAutoMode(!isAutoMode);
  }, [isAutoMode, setIsAutoMode]);

  // Add/remove onboarding-active class to body
  useEffect(() => {
    if (isOnboardingActive) {
      document.body.classList.add('onboarding-active');
    } else {
      document.body.classList.remove('onboarding-active');
    }

    // Cleanup on unmount
    return () => {
      document.body.classList.remove('onboarding-active');
    };
  }, [isOnboardingActive]);

  // Keyboard navigation
  useEffect(() => {
    if (!isOnboardingActive) return;

    const handleKeyPress = (e) => {
      switch (e.key) {
        case 'ArrowRight':
        case ' ':
          e.preventDefault();
          handleNext();
          break;
        case 'ArrowLeft':
          e.preventDefault();
          handlePrevious();
          break;
        case 'Escape':
          e.preventDefault();
          skipOnboarding();
          break;
        case 'a':
        case 'A':
          e.preventDefault();
          handleToggleAutoMode();
          break;
        default:
          break;
      }
    };

    window.addEventListener('keydown', handleKeyPress);
    return () => window.removeEventListener('keydown', handleKeyPress);
  }, [isOnboardingActive, handleNext, handlePrevious, skipOnboarding, handleToggleAutoMode]);

  if (!isOnboardingActive) return null;

  return (
    <OnboardingModal
      step={currentStepData}
      isVisible={isOnboardingActive}
      onNext={handleNext}
      onPrevious={handlePrevious}
      onSkip={skipOnboarding}
      onComplete={completeOnboarding}
      isAutoMode={isAutoMode}
      onToggleAutoMode={handleToggleAutoMode}
      totalSteps={totalSteps}
    />
  );
};

export default OnboardingSequence;
