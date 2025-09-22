import React, { useState, useEffect } from 'react';
import { useAppLogo } from '../hooks/useAssetPath';
import KnirvNetworkLogo from './KnirvNetworkLogo.jsx';

const LoadingScreen = ({
  isVisible = true,
  message = "Starting KNIRV Cortex Builder...",
  progress = null,
  onComplete = null
}) => {
  const { logoPath, isLoaded } = useAppLogo();
  const [dots, setDots] = useState('');
  const [currentMessage, setCurrentMessage] = useState(message);

  // Animate loading dots
  useEffect(() => {
    const interval = setInterval(() => {
      setDots(prev => {
        if (prev === '...') return '';
        return prev + '.';
      });
    }, 500);

    return () => clearInterval(interval);
  }, []);

  // Update message when prop changes
  useEffect(() => {
    setCurrentMessage(message);
  }, [message]);

  // Auto-hide after minimum display time if onComplete is provided
  useEffect(() => {
    if (onComplete && isVisible) {
      const timer = setTimeout(() => {
        onComplete();
      }, 2000); // Minimum 2 seconds display time

      return () => clearTimeout(timer);
    }
  }, [isVisible, onComplete]);

  if (!isVisible) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 bg-gradient-to-br from-slate-900 via-slate-800 to-black flex items-center justify-center">
      {/* Background pattern */}
      <div className="absolute inset-0 opacity-20">
        <div className="absolute inset-0 knirv-gradient animate-pulse"></div>
        <div className="absolute top-0 left-0 w-full h-full bg-[radial-gradient(circle_at_50%_50%,rgba(120,119,198,0.1),transparent_50%)]"></div>
      </div>

      {/* Loading content */}
      <div className="relative z-10 flex flex-col items-center space-y-8 max-w-lg mx-auto px-6">
        {/* KNIRV Animated Logo */}
        <div className="relative w-80 h-40 bg-gradient-to-br from-slate-900 via-slate-800 to-black rounded-2xl p-4 shadow-2xl knirv-glow">
          <KnirvNetworkLogo />

          {/* Animated ring around logo */}
          <div className="absolute inset-0 w-full h-full border-2 knirv-border-primary/30 rounded-2xl animate-pulse"></div>
          <div className="absolute inset-0 w-full h-full border-2 border-transparent border-t-blue-400 rounded-2xl animate-spin"></div>
        </div>

        {/* App title */}
        <div className="text-center">
          <h1 className="text-3xl font-bold knirv-gradient-text mb-2">KNIRV CORTEX</h1>
          <p className="text-slate-300 text-lg">Neural Intelligence Platform</p>
        </div>

        {/* Loading message and spinner */}
        <div className="text-center space-y-4">
          <div className="flex items-center justify-center space-x-2">
            <div className="w-6 h-6 border-2 knirv-border-primary/30 border-t-blue-400 rounded-full animate-spin"></div>
            <span className="text-white font-medium">{currentMessage}{dots}</span>
          </div>

          {/* Progress bar if provided */}
          {progress !== null && (
            <div className="w-64 bg-slate-700 rounded-full h-2">
              <div
                className="knirv-gradient h-2 rounded-full transition-all duration-500 ease-out"
                style={{ width: `${Math.max(0, Math.min(100, progress))}%` }}
              ></div>
            </div>
          )}

          {/* Loading steps indicator */}
          <div className="flex justify-center space-x-2 mt-6">
            <div className="w-2 h-2 knirv-bg-primary rounded-full animate-pulse"></div>
            <div className="w-2 h-2 bg-blue-400/50 rounded-full animate-pulse" style={{ animationDelay: '0.2s' }}></div>
            <div className="w-2 h-2 bg-blue-400/30 rounded-full animate-pulse" style={{ animationDelay: '0.4s' }}></div>
          </div>
        </div>

        {/* Version info */}
        <div className="text-center text-slate-400 text-sm">
          <p>KNIRV Cortex Builder v1.0.0</p>
          <p>Initializing neural services...</p>
        </div>
      </div>

      {/* Subtle background animation */}
      <div className="absolute bottom-0 left-0 w-full h-1 knirv-gradient opacity-50">
        <div className="h-full bg-white/20 animate-pulse"></div>
      </div>
    </div>
  );
};

export default LoadingScreen;
