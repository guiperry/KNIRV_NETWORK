import React from 'react';
import { Play, RotateCcw, HelpCircle } from 'lucide-react';
import { useOnboarding } from './OnboardingProvider';

const OnboardingTrigger = ({ className = "" }) => {
  const { hasCompletedOnboarding, startOnboarding, restartOnboarding } = useOnboarding();

  const handleStartTour = (autoMode = false) => {
    if (hasCompletedOnboarding) {
      restartOnboarding(autoMode);
    } else {
      startOnboarding(autoMode);
    }
  };

  return (
    <div className={`flex items-center space-x-2 ${className}`}>
      {/* Auto Tour Button */}
      <button
        onClick={() => handleStartTour(true)}
        className="flex items-center space-x-2 px-3 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm"
        title="Start automatic guided tour"
      >
        <Play size={16} />
        <span>Auto Tour</span>
      </button>

      {/* Manual Tour Button */}
      <button
        onClick={() => handleStartTour(false)}
        className="flex items-center space-x-2 px-3 py-2 bg-slate-600 text-white rounded-lg hover:bg-slate-700 transition-colors text-sm"
        title="Start manual guided tour"
      >
        <HelpCircle size={16} />
        <span>Manual Tour</span>
      </button>

      {/* Restart Tour Button (only show if completed) */}
      {hasCompletedOnboarding && (
        <button
          onClick={() => restartOnboarding()}
          className="flex items-center space-x-2 px-3 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors text-sm"
          title="Restart the guided tour"
        >
          <RotateCcw size={16} />
          <span>Restart</span>
        </button>
      )}
    </div>
  );
};

export default OnboardingTrigger;
