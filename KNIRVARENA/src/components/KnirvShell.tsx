import React from 'react';

import { useState, useEffect } from 'react';
import { Cpu, Zap, AlertTriangle, FileText, Lightbulb, Plus, Camera, Mic, Shield, Wallet, Eye, Lock, ShieldAlert, Brain } from 'lucide-react';
import GameArena from './GameArena';
import { useThemeStore } from '../stores/useThemeStore';

interface KnirvShellProps {
  status: 'idle' | 'processing' | 'listening' | 'error';
  nrnBalance: number;
  onScreenshotCapture: () => void;
  cognitiveMode: boolean;
  isVoiceActive?: boolean;
  onSubmitError?: () => void;
  onSubmitContext?: () => void;
  onSubmitIdea?: () => void;
  onSubmitDemo?: () => void;
  onSkillsOpen?: () => void;
  onUDCOpen?: () => void;
  onWalletOpen?: () => void;
}

export const KnirvShell: React.FC<KnirvShellProps> = ({
  status,
  nrnBalance,
  onScreenshotCapture,
  cognitiveMode,
  isVoiceActive = false,
  onSubmitError,
  onSubmitContext,
  onSubmitIdea,
  onSubmitDemo,
  onSkillsOpen,
  onUDCOpen,
  onWalletOpen
}) => {
  const [currentTime, setCurrentTime] = useState(new Date());
  const [isExpanded, setIsExpanded] = useState(false);
  const [isAdversarySolving, setIsAdversarySolving] = useState(false);
  const { themeMode, toggleTheme } = useThemeStore();

  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);

    // Simulate adversary solving state
    const adversaryTimer = setInterval(() => {
      setIsAdversarySolving(prev => !prev);
    }, 5000);

    return () => {
      clearInterval(timer);
      clearInterval(adversaryTimer);
    };
  }, []);

  const handleNoiseInjection = () => {
    console.log('Noise Injection activated');
    setIsExpanded(false);
  };

  const handleWeightHijacking = () => {
    console.log('Weight Hijacking activated');
    setIsExpanded(false);
  };

  const handleContextPoisoning = () => {
    console.log('Context Poisoning activated');
    setIsExpanded(false);
  };

  const handleBackpropPulse = () => {
    console.log('Backprop Pulse activated');
    setIsExpanded(false);
  };

  return (
    <div className="absolute inset-0 bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900" data-testid="knirv-shell">
      {/* Header */}
      <header className="absolute top-0 left-0 right-0 z-30 bg-gray-900/80 backdrop-blur-sm border-b border-gray-700/50">
        <div className="flex items-center justify-between p-4">
          <div className="flex items-center space-x-3">
            <div className="w-8 h-8 bg-gradient-to-r from-blue-500 to-teal-500 rounded-lg flex items-center justify-center">
              <Cpu className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-white">KNIRV Cortex</h1>
              <p className="text-xs text-gray-400">Train Your Own SLM Agent</p>
            </div>
          </div>
          
          <div className="flex items-center space-x-4">
            <div className="text-right">
              <p className="text-sm font-medium text-white">{nrnBalance.toLocaleString()} NRN</p>
              <p className="text-xs text-gray-400">Balance</p>
            </div>
            <div className="text-right">
              <p className="text-sm font-medium text-white">{currentTime.toLocaleTimeString()}</p>
              <p className="text-xs text-gray-400">{currentTime.toLocaleDateString()}</p>
            </div>
            <div className="flex items-center space-x-2">
              <button 
                onClick={onSkillsOpen}
                className="p-2 rounded-lg bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 text-blue-400 hover:text-blue-300 transition-colors"
                title="Skills"
              >
                <Zap className="w-4 h-4" />
              </button>
              <button 
                onClick={onUDCOpen}
                className="p-2 rounded-lg bg-purple-600/20 hover:bg-purple-600/30 border border-purple-500/30 text-purple-400 hover:text-purple-300 transition-colors"
                title="User Delegation Certificate"
              >
                <Shield className="w-4 h-4" />
              </button>
              <button 
                onClick={onWalletOpen}
                className="p-2 rounded-lg bg-green-600/20 hover:bg-green-600/30 border border-green-500/30 text-green-400 hover:text-green-300 transition-colors"
                title="Wallet"
              >
                <Wallet className="w-4 h-4" />
              </button>
              <button
                onClick={toggleTheme}
                className="p-2 rounded-lg bg-gray-700/20 hover:bg-gray-700/30 border border-gray-600/30 text-gray-400 hover:text-gray-300 transition-colors"
                title="Toggle Theme"
              >
                {themeMode === 'dark' && '🌙'}
                {themeMode === 'light' && '☀️'}
                {themeMode === 'light-blue' && '💧'}
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <div className="absolute inset-0 pt-20 pb-16 px-8">
        <div className="relative w-full h-full bg-gray-800/30 backdrop-blur-sm rounded-2xl border border-gray-700/50 overflow-hidden">
          {/* Central Interface - Gaming Arena */}
          <div className="absolute inset-0 flex items-center justify-center">
            <GameArena />
          </div>

          {/* Floating Action Icons */}
          <div className="absolute top-6 left-6 flex flex-col space-y-4">
            {/* Cognitive Mode Indicator */}
            <div className={`w-12 h-12 rounded-lg flex items-center justify-center backdrop-blur-sm transition-all duration-300 ${
              cognitiveMode
                ? 'bg-yellow-500/30 border border-yellow-500/50'
                : 'bg-gray-700/50'
            }`}>
              <Zap className={`w-6 h-6 transition-all duration-300 ${
                cognitiveMode ? 'text-yellow-300' : 'text-yellow-400'
              }`} fill={cognitiveMode ? 'currentColor' : 'none'} />
            </div>

            {/* Screenshot Capture Button */}
            <button
              onClick={onScreenshotCapture}
              className="w-12 h-12 rounded-lg flex items-center justify-center backdrop-blur-sm bg-gray-700/50 hover:bg-gray-600/50 transition-all duration-300 border border-gray-600/50 hover:border-gray-500/50"
              title="Capture Screenshot"
            >
              <Camera className="w-6 h-6 text-gray-300 hover:text-white transition-colors" />
            </button>
          </div>
        </div>
      </div>

      {/* Sabotage Button - Moved to Bottom Right */}
      <div className="absolute bottom-4 right-4 z-40">
        {/* Main Button Container */}
        <div className="relative w-14 h-14">
          {/* Radial Nested Buttons - Semi-circle arc from top to left */}
          {isExpanded && (
            <>
              {/* Noise Injection - Position 1 (top, 90°) */}
              <button
                onClick={handleNoiseInjection}
                className="absolute w-12 h-12 cursor-pointer transition-all duration-300 hover:scale-110 group/radial"
                style={{
                  top: '-66px',
                  left: '4px',
                }}
                title="Noise Injection"
              >
                <div className="absolute inset-0 rounded-full bg-yellow-500/90 border-2 border-yellow-400/50 group-hover/radial:bg-yellow-500 shadow-lg">
                  <div className="absolute inset-1.5 rounded-full bg-gradient-to-r from-yellow-500 to-yellow-600 flex items-center justify-center group-hover/radial:from-yellow-400 group-hover/radial:to-yellow-500 transition-all">
                    <Eye className="w-5 h-5 text-white" />
                  </div>
                </div>
              </button>

              {/* Weight Hijacking - Position 2 (upper-left, 120°) */}
              <button
                onClick={handleWeightHijacking}
                className="absolute w-12 h-12 cursor-pointer transition-all duration-300 hover:scale-110 group/radial"
                style={{
                  top: '-57px',
                  left: '-31px',
                }}
                title="Weight Hijacking"
              >
                <div className="absolute inset-0 rounded-full bg-red-500/90 border-2 border-red-400/50 group-hover/radial:bg-red-500 shadow-lg">
                  <div className="absolute inset-1.5 rounded-full bg-gradient-to-r from-red-500 to-red-600 flex items-center justify-center group-hover/radial:from-red-400 group-hover/radial:to-red-500 transition-all">
                    <Lock className="w-5 h-5 text-white" />
                  </div>
                </div>
              </button>

              {/* Context Poisoning - Position 3 (left-upper, 150°) */}
              <button
                onClick={handleContextPoisoning}
                className="absolute w-12 h-12 cursor-pointer transition-all duration-300 hover:scale-110 group/radial"
                style={{
                  top: '-31px',
                  left: '-57px',
                }}
                title="Context Poisoning"
              >
                <div className="absolute inset-0 rounded-full bg-purple-500/90 border-2 border-purple-400/50 group-hover/radial:bg-purple-500 shadow-lg">
                  <div className="absolute inset-1.5 rounded-full bg-gradient-to-r from-purple-500 to-purple-600 flex items-center justify-center group-hover/radial:from-purple-400 group-hover/radial:to-purple-500 transition-all">
                    <Brain className="w-5 h-5 text-white" />
                  </div>
                </div>
              </button>

              {/* Backprop Pulse - Position 4 (left, 180°) */}
              <button
                onClick={handleBackpropPulse}
                className="absolute w-12 h-12 cursor-pointer transition-all duration-300 hover:scale-110 group/radial"
                style={{
                  top: '4px',
                  left: '-66px',
                }}
                title="Backprop Pulse"
              >
                <div className="absolute inset-0 rounded-full bg-orange-500/90 border-2 border-orange-400/50 group-hover/radial:bg-orange-500 shadow-lg">
                  <div className="absolute inset-1.5 rounded-full bg-gradient-to-r from-orange-500 to-orange-600 flex items-center justify-center group-hover/radial:from-orange-400 group-hover/radial:to-orange-500 transition-all">
                    <ShieldAlert className="w-5 h-5 text-white" />
                  </div>
                </div>
              </button>
            </>
          )}

          {/* Main Sabotage Button */}
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="relative w-14 h-14 group cursor-pointer transition-transform hover:scale-105 z-20"
          >
            <div className={`absolute inset-0 rounded-full transition-all duration-1000 ${
              isAdversarySolving
                ? 'bg-red-500/20 border-2 border-red-500/50 animate-pulse'
                : 'bg-gray-700/20 border-2 border-gray-600/50 group-hover:bg-gray-700/30'
            }`}>
              <div className="absolute inset-2 rounded-full bg-gradient-to-r from-red-500 to-orange-500 flex items-center justify-center group-hover:from-red-400 group-hover:to-orange-400 transition-all">
                <ShieldAlert className={`w-6 h-6 text-white transition-transform duration-300 ${isExpanded ? 'rotate-45' : ''}`} />
              </div>
            </div>
            {status === 'processing' && (
              <div className="absolute inset-0 rounded-full border-2 border-transparent border-t-blue-500 animate-spin"></div>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
