import React from 'react';

import { useState, useEffect } from 'react';
import { Cpu, Zap, AlertTriangle, FileText, Lightbulb, Plus, Camera, Mic, Shield, Wallet } from 'lucide-react';
import GameArena from './GameArena';

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

  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);

    return () => clearInterval(timer);
  }, []);

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

      {/* Main Action Button - Moved to Bottom Right */}
      <div className="absolute bottom-4 right-4 z-40">
        {/* Main Button Container */}
        <div className="relative w-14 h-14">
          {/* Radial Nested Buttons - Semi-circle arc from top to left */}
          {isExpanded && (
            <>
              {/* Submit Error - Position 1 (top, 90°) */}
              <button
                onClick={() => {
                  onSubmitError?.();
                  setIsExpanded(false);
                }}
                className="absolute w-12 h-12 cursor-pointer transition-all duration-300 hover:scale-110 group/radial"
                style={{
                  top: '-66px',
                  left: '4px',
                }}
                title="Submit Error"
              >
                <div className="absolute inset-0 rounded-full bg-red-500/90 border-2 border-red-400/50 group-hover/radial:bg-red-500 shadow-lg">
                  <div className="absolute inset-1.5 rounded-full bg-gradient-to-r from-red-500 to-red-600 flex items-center justify-center group-hover/radial:from-red-400 group-hover/radial:to-red-500 transition-all">
                    <AlertTriangle className="w-5 h-5 text-white" />
                  </div>
                </div>
              </button>

              {/* Submit Context - Position 2 (upper-left, 120°) */}
              <button
                onClick={() => {
                  onSubmitContext?.();
                  setIsExpanded(false);
                }}
                className="absolute w-12 h-12 cursor-pointer transition-all duration-300 hover:scale-110 group/radial"
                style={{
                  top: '-57px',
                  left: '-31px',
                }}
                title="Submit Context"
              >
                <div className="absolute inset-0 rounded-full bg-blue-500/90 border-2 border-blue-400/50 group-hover/radial:bg-blue-500 shadow-lg">
                  <div className="absolute inset-1.5 rounded-full bg-gradient-to-r from-blue-500 to-blue-600 flex items-center justify-center group-hover/radial:from-blue-400 group-hover/radial:to-blue-500 transition-all">
                    <FileText className="w-5 h-5 text-white" />
                  </div>
                </div>
              </button>

              {/* Submit Idea - Position 3 (left-upper, 150°) */}
              <button
                onClick={() => {
                  onSubmitIdea?.();
                  setIsExpanded(false);
                }}
                className="absolute w-12 h-12 cursor-pointer transition-all duration-300 hover:scale-110 group/radial"
                style={{
                  top: '-31px',
                  left: '-57px',
                }}
                title="Submit Idea"
              >
                <div className="absolute inset-0 rounded-full bg-yellow-500/90 border-2 border-yellow-400/50 group-hover/radial:bg-yellow-500 shadow-lg">
                  <div className="absolute inset-1.5 rounded-full bg-gradient-to-r from-yellow-500 to-yellow-600 flex items-center justify-center group-hover/radial:from-yellow-400 group-hover/radial:to-yellow-500 transition-all">
                    <Lightbulb className="w-5 h-5 text-white" />
                  </div>
                </div>
              </button>

              {/* Auto Demo - Position 4 (left, 180°) */}
              <button
                onClick={() => {
                  onSubmitDemo?.();
                  setIsExpanded(false);
                }}
                className="absolute w-12 h-12 cursor-pointer transition-all duration-300 hover:scale-110 group/radial"
                style={{
                  top: '4px',
                  left: '-66px',
                }}
                title="Auto Demo"
              >
                <div className={`absolute inset-0 rounded-full border-2 shadow-lg transition-all duration-300 ${
                  isVoiceActive
                    ? 'bg-teal-500/90 border-teal-400/50 group-hover/radial:bg-teal-600 animate-pulse'
                    : 'bg-teal-500/90 border-teal-400/50 group-hover/radial:bg-teal-500'
                }`}>
                  <div className={`absolute inset-1.5 rounded-full bg-gradient-to-r flex items-center justify-center transition-all duration-300 ${
                    isVoiceActive
                      ? 'from-teal-600 to-purple-700'
                      : 'from-teal-500 to-purple-600 group-hover/radial:from-teal-400 group-hover/radial:to-purple-500'
                  }`}>
                    <Mic className="w-5 h-5 text-white" />
                  </div>
                </div>
              </button>
            </>
          )}

          {/* Main Button */}
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="relative w-14 h-14 group cursor-pointer transition-transform hover:scale-105 z-20"
          >
            <div className={`absolute inset-0 rounded-full transition-all duration-1000 ${
              status === 'idle' ? 'bg-green-500/20 border-2 border-green-500/50 group-hover:bg-green-500/30' :
              status === 'processing' ? 'bg-blue-500/20 border-2 border-blue-500/50 animate-pulse' :
              status === 'listening' ? 'bg-teal-500/20 border-2 border-teal-500/50 animate-pulse' :
              'bg-red-500/20 border-2 border-red-500/50'
            }`}>
              <div className="absolute inset-2 rounded-full bg-gradient-to-r from-blue-500 to-teal-500 flex items-center justify-center group-hover:from-blue-400 group-hover:to-teal-400 transition-all">
                <Plus className={`w-6 h-6 text-white transition-transform duration-300 ${isExpanded ? 'rotate-45' : ''}`} />
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
