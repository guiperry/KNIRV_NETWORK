import React, { useState, useEffect } from 'react';
import { Monitor, Zap, Camera, Cpu, Network, Brain } from 'lucide-react';

interface KnirvShellProps {
  status: 'idle' | 'processing' | 'listening' | 'error';
  nrnBalance: number;
  onScreenshotCapture: () => void;
  onAnalyze: () => void;
  onNetworkToggle: () => void;
}

export const KnirvShell: React.FC<KnirvShellProps> = ({
  status,
  nrnBalance,
  onScreenshotCapture,
  onAnalyze,
  onNetworkToggle
}) => {
  const [currentTime, setCurrentTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);

    return () => clearInterval(timer);
  }, []);

  return (
    <div className="absolute inset-0 bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900">
      {/* Header */}
      <header className="absolute top-0 left-0 right-0 z-30 bg-gray-900/80 backdrop-blur-sm border-b border-gray-700/50">
        <div className="flex items-center justify-between p-4">
          <div className="flex items-center space-x-3">
            <div className="w-8 h-8 bg-gradient-to-r from-blue-500 to-teal-500 rounded-lg flex items-center justify-center">
              <Brain className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-white">KNIRV-SHELL</h1>
              <p className="text-xs text-gray-400">Adaptive Intelligence Interface</p>
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
              {/* Status Indicator */}
              <div className="relative w-32 h-32 mx-auto">
                <div className={`absolute inset-0 rounded-full transition-all duration-1000 ${
                  status === 'idle' ? 'bg-green-500/20 border-2 border-green-500/50' :
                  status === 'processing' ? 'bg-blue-500/20 border-2 border-blue-500/50 animate-pulse' :
                  status === 'listening' ? 'bg-teal-500/20 border-2 border-teal-500/50 animate-pulse' :
                  'bg-red-500/20 border-2 border-red-500/50'
                }`}>
                  <div className="absolute inset-4 rounded-full bg-gradient-to-r from-blue-500 to-teal-500 flex items-center justify-center">
                    <Cpu className="w-8 h-8 text-white" />
                  </div>
                </div>
                {status === 'processing' && (
                  <div className="absolute inset-0 rounded-full border-2 border-transparent border-t-blue-500 animate-spin"></div>
                )}
              </div>

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

              {/* Quick Actions */}
              <div className="flex justify-center space-x-4 mt-8">
                <button
                  onClick={onScreenshotCapture}
                  className="flex items-center space-x-2 px-4 py-2 bg-blue-500/20 text-blue-400 rounded-lg border border-blue-500/30 hover:bg-blue-500/30 transition-colors"
                >
                  <Camera className="w-4 h-4" />
                  <span>Capture</span>
                </button>
                <button
                  onClick={onAnalyze}
                  className="flex items-center space-x-2 px-4 py-2 bg-teal-500/20 text-teal-400 rounded-lg border border-teal-500/30 hover:bg-teal-500/30 transition-colors"
                >
                  <Monitor className="w-4 h-4" />
                  <span>Analyze</span>
                </button>
                <button
                  onClick={onNetworkToggle}
                  className="flex items-center space-x-2 px-4 py-2 bg-orange-500/20 text-orange-400 rounded-lg border border-orange-500/30 hover:bg-orange-500/30 transition-colors"
                >
                  <Network className="w-4 h-4" />
                  <span>Network</span>
                </button>
              </div>
            </div>
          </div>

          {/* Floating Elements */}
          <div className="absolute top-6 right-6 flex flex-col space-y-4">
            <div className="w-12 h-12 bg-gray-700/50 rounded-lg flex items-center justify-center backdrop-blur-sm">
              <Zap className="w-6 h-6 text-yellow-400" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};