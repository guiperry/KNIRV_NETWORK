import React, { useState, useEffect } from 'react';
import { Cpu, Zap } from 'lucide-react';

interface KnirvShellProps {
  status: 'idle' | 'processing' | 'listening' | 'error';
  nrnBalance: number;
  onScreenshotCapture: () => void;
  cognitiveMode: boolean;
}

export const KnirvShell: React.FC<KnirvShellProps> = ({
  status,
  nrnBalance,
  onScreenshotCapture,
  cognitiveMode
}) => {
  const [currentTime, setCurrentTime] = useState(new Date());

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
              <p className="text-xs text-gray-400">AI Agent Framework</p>
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
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <div className="absolute inset-0 pt-20 pb-16 px-8">
        <div className="relative w-full h-full bg-gray-800/30 backdrop-blur-sm rounded-2xl border border-gray-700/50 overflow-hidden">
          {/* Central Interface */}
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center space-y-6">
              {/* Status Indicator - Now Clickable */}
              <button
                onClick={onScreenshotCapture}
                className="relative w-32 h-32 mx-auto group cursor-pointer transition-transform hover:scale-105"
              >
                <div className={`absolute inset-0 rounded-full transition-all duration-1000 ${
                  status === 'idle' ? 'bg-green-500/20 border-2 border-green-500/50 group-hover:bg-green-500/30' :
                  status === 'processing' ? 'bg-blue-500/20 border-2 border-blue-500/50 animate-pulse' :
                  status === 'listening' ? 'bg-teal-500/20 border-2 border-teal-500/50 animate-pulse' :
                  'bg-red-500/20 border-2 border-red-500/50'
                }`}>
                  <div className="absolute inset-4 rounded-full bg-gradient-to-r from-blue-500 to-teal-500 flex items-center justify-center group-hover:from-blue-400 group-hover:to-teal-400 transition-all">
                    <Cpu className="w-8 h-8 text-white" />
                  </div>
                </div>
                {status === 'processing' && (
                  <div className="absolute inset-0 rounded-full border-2 border-transparent border-t-blue-500 animate-spin"></div>
                )}
              </button>

              {/* Status Message */}
              <div className="space-y-2">
                <h2 className="text-2xl font-bold text-white">
                  {status === 'idle' && 'Ready for Input'}
                  {status === 'processing' && 'Processing Request'}
                  {status === 'listening' && 'Listening...'}
                  {status === 'error' && 'Error Detected'}
                </h2>
                <p className="text-gray-400 max-w-md mx-auto">
                  {status === 'idle' && 'Use voice commands or visual input to identify problems and assign agents.'}
                  {status === 'processing' && 'The Fabric algorithm is analyzing your input and generating NRV objects.'}
                  {status === 'listening' && 'Speak clearly to interact with the KNIRV network.'}
                  {status === 'error' && 'An error has occurred. Please try again or check network connections.'}
                </p>
              </div>


            </div>
          </div>

          {/* Floating Lightning Icon */}
          <div className="absolute top-6 left-6 flex flex-col space-y-4">
            <div className={`w-12 h-12 rounded-lg flex items-center justify-center backdrop-blur-sm transition-all duration-300 ${
              cognitiveMode
                ? 'bg-yellow-500/30 border border-yellow-500/50'
                : 'bg-gray-700/50'
            }`}>
              <Zap className={`w-6 h-6 transition-all duration-300 ${
                cognitiveMode ? 'text-yellow-300' : 'text-yellow-400'
              }`} fill={cognitiveMode ? 'currentColor' : 'none'} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};