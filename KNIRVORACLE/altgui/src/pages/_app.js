import React, { useState, useEffect } from 'react';
import '../styles/globals.css';
import OnboardingFlow from '../components/OnboardingFlowUpdated';
import { BackendProvider } from '../contexts/BackendContext';
import { BlockchainProvider } from '../contexts/BlockchainContext';
import { RoleProvider } from '../contexts/RoleContext';

// Global styles are now in globals.css

function MyApp({ Component, pageProps }) {
  const [showOnboarding, setShowOnboarding] = useState(false);
  const [onboardingCompleted, setOnboardingCompleted] = useState(false);

  // Check if this is the first time the user is visiting
  useEffect(() => {
    // In a real app, you'd check localStorage or a database
    const hasCompletedOnboarding = localStorage.getItem('onboardingCompleted');

    if (!hasCompletedOnboarding) {
      // Show onboarding after a short delay to ensure the app has loaded
      const timer = setTimeout(() => {
        setShowOnboarding(true);
      }, 1000);

      return () => clearTimeout(timer);
    } else {
      setOnboardingCompleted(true);
    }
  }, []);

  const handleOnboardingComplete = () => {
    setShowOnboarding(false);
    setOnboardingCompleted(true);
    localStorage.setItem('onboardingCompleted', 'true');
  };

  return (
    <BackendProvider>
      <BlockchainProvider>
        <RoleProvider>
          <Component {...pageProps} onboardingCompleted={onboardingCompleted} />
          {showOnboarding && <OnboardingFlow onComplete={handleOnboardingComplete} />}
        </RoleProvider>
      </BlockchainProvider>
    </BackendProvider>
  );
}

export default MyApp;