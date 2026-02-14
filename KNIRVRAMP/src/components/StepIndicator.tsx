'use client';

import React from 'react';

// Support both legacy and new onboarding steps
type LegacyStep = 'hero' | 'connect' | 'configure' | 'deploy' | 'dashboard';
type NewStep = 'guide' | 'preferences' | 'welcome' | 'verification' | 'cortex';
type Step = LegacyStep | NewStep;

interface StepIndicatorProps {
  currentStep: Step;
}

const StepIndicator: React.FC<StepIndicatorProps> = ({ currentStep }) => {
  // New flow steps
  const newSteps = [
    { id: 'guide', label: 'Setup Guide', steps: ['guide'] },
    { id: 'preferences', label: 'Preferences', steps: ['preferences'] },
    { id: 'welcome', label: 'Welcome', steps: ['welcome'] },
    { id: 'cortex', label: 'Cloud Cortex', steps: ['verification', 'cortex'] }
  ];

  // Legacy flow steps
  const legacySteps = [
    { id: 'configure', label: 'Configure Model', steps: ['connect', 'configure', 'deploy', 'results', 'dashboard'] },
    { id: 'deploy', label: 'Compile & Pre-Train', steps: ['deploy', 'results', 'dashboard'] },
    { id: 'results', label: 'Download & Connect', steps: ['results', 'dashboard'] },
    { id: 'dashboard', label: 'Deployment Dashboard', steps: ['dashboard'] }
  ];

  // Determine which flow we're in
  const isNewFlow = ['guide', 'preferences', 'welcome', 'verification', 'cortex'].includes(currentStep);
  const steps = isNewFlow ? newSteps : legacySteps;

  // Determine which steps are active based on the current step
  const isStepActive = (stepId: string) => {
    // Hide indicator on initial pages
    if (currentStep === 'hero' || currentStep === 'guide') return false;

    const step = steps.find(s => s.id === stepId);
    return step ? step.steps.includes(currentStep) : false;
  };

  const isStepCompleted = (stepIndex: number) => {
    const step = steps[stepIndex];
    if (!step) return false;
    
    // For new flow, check if we've passed this step
    if (isNewFlow) {
      const currentIndex = newSteps.findIndex(s => s.steps.includes(currentStep));
      return stepIndex < currentIndex;
    }
    
    return false;
  };

  // Don't show on initial landing pages
  if (currentStep === 'hero' || currentStep === 'guide') return null;

  return (
    <div className="hidden md:flex items-center space-x-4">
      {steps.map((step, index) => (
        <div key={step.id} className="flex items-center">
          {/* Step circle with number */}
          <div 
            className={`h-8 w-8 rounded-full flex items-center justify-center mr-2 transition-all ${
              isStepActive(step.id)
                ? 'bg-blue-500 text-white shadow-[0_0_10px_rgba(59,130,246,0.5)]'
                : isStepCompleted(index)
                  ? 'bg-green-500 text-white'
                  : 'bg-slate-700 text-slate-400'
            }`}
          >
            {isStepCompleted(index) ? (
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            ) : (
              index + 1
            )}
          </div>
          
          {/* Step label */}
          <span className={`text-sm font-medium transition-colors ${
            isStepActive(step.id) ? 'text-white' : 'text-slate-500'
          }`}>
            {step.label}
          </span>
          
          {/* Arrow between steps (except after the last step) */}
          {index < steps.length - 1 && (
            <div className="mx-3 text-slate-700">→</div>
          )}
        </div>
      ))}
    </div>
  );
};

export default StepIndicator;
