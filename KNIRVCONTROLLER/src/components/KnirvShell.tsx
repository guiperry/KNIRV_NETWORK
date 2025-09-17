import * as React from 'react';
import { useState, useEffect } from 'react';
import { Cpu, Zap, AlertTriangle, FileText, Lightbulb, Plus, X, Camera } from 'lucide-react';

interface KnirvShellProps {
  status: 'idle' | 'processing' | 'listening' | 'error';
  nrnBalance: number;
  onScreenshotCapture: () => void;
  cognitiveMode: boolean;
  onSubmitError?: () => void;
  onSubmitContext?: () => void;
  onSubmitIdea?: () => void;
}

export const KnirvShell: React.FC<KnirvShellProps> = ({
  status,
  nrnBalance,
  onScreenshotCapture,
  cognitiveMode,
  onSubmitError,
  onSubmitContext,
  onSubmitIdea
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
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <div className="absolute inset-0 pt-20 pb-16 px-8">
        <div className="relative w-full h-full bg-gray-800/30 backdrop-blur-sm rounded-2xl border border-gray-700/50 overflow-hidden">
          {/* Central Interface */}
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center space-y-6">
              {/* Main Action Button - Expandable */}
              <div className="relative">
                {!isExpanded ? (
                  /* Main Button */
                  <button
                    onClick={() => setIsExpanded(true)}
                    className="relative w-32 h-32 mx-auto group cursor-pointer transition-transform hover:scale-105"
                  >
                    <div className={`absolute inset-0 rounded-full transition-all duration-1000 ${
                      status === 'idle' ? 'bg-green-500/20 border-2 border-green-500/50 group-hover:bg-green-500/30' :
                      status === 'processing' ? 'bg-blue-500/20 border-2 border-blue-500/50 animate-pulse' :
                      status === 'listening' ? 'bg-teal-500/20 border-2 border-teal-500/50 animate-pulse' :
                      'bg-red-500/20 border-2 border-red-500/50'
                    }`}>
                      <div className="absolute inset-4 rounded-full bg-gradient-to-r from-blue-500 to-teal-500 flex items-center justify-center group-hover:from-blue-400 group-hover:to-teal-400 transition-all">
                        <Plus className="w-8 h-8 text-white" />
                      </div>
                    </div>
                    {status === 'processing' && (
                      <div className="absolute inset-0 rounded-full border-2 border-transparent border-t-blue-500 animate-spin"></div>
                    )}
                  </button>
                ) : (
                  /* Expanded 3-Button Interface */
                  <div className="relative mb-16">
                    {/* Close Button */}
                    <button
                      onClick={() => setIsExpanded(false)}
                      className="absolute -top-4 -right-4 w-8 h-8 bg-gray-700/80 hover:bg-gray-600/80 rounded-full flex items-center justify-center transition-colors z-10"
                    >
                      <X className="w-4 h-4 text-gray-300" />
                    </button>

                    {/* Three Action Buttons */}
                    <div className="flex items-center justify-center space-x-6">
                      {/* Submit Error */}
                      <button
                        onClick={() => {
                          onSubmitError?.();
                          setIsExpanded(false);
                        }}
                        className="group relative w-24 h-24 cursor-pointer transition-transform hover:scale-105"
                      >
                        <div className="absolute inset-0 rounded-full bg-red-500/20 border-2 border-red-500/50 group-hover:bg-red-500/30 transition-all">
                          <div className="absolute inset-3 rounded-full bg-gradient-to-r from-red-500 to-red-600 flex items-center justify-center group-hover:from-red-400 group-hover:to-red-500 transition-all">
                            <AlertTriangle className="w-6 h-6 text-white" />
                          </div>
                        </div>
                        <div className="absolute -bottom-12 left-1/2 transform -translate-x-1/2 text-xs text-red-400 font-medium whitespace-nowrap text-center">
                          <div>Submit Error</div>
                          <div className="text-gray-500 text-[10px]">→ Train Skill</div>
                        </div>
                      </button>

                      {/* Submit Context */}
                      <button
                        onClick={() => {
                          onSubmitContext?.();
                          setIsExpanded(false);
                        }}
                        className="group relative w-24 h-24 cursor-pointer transition-transform hover:scale-105"
                      >
                        <div className="absolute inset-0 rounded-full bg-blue-500/20 border-2 border-blue-500/50 group-hover:bg-blue-500/30 transition-all">
                          <div className="absolute inset-3 rounded-full bg-gradient-to-r from-blue-500 to-blue-600 flex items-center justify-center group-hover:from-blue-400 group-hover:to-blue-500 transition-all">
                            <FileText className="w-6 h-6 text-white" />
                          </div>
                        </div>
                        <div className="absolute -bottom-12 left-1/2 transform -translate-x-1/2 text-xs text-blue-400 font-medium whitespace-nowrap text-center">
                          <div>Submit Context</div>
                          <div className="text-gray-500 text-[10px]">→ Train Capability</div>
                        </div>
                      </button>

                      {/* Submit Idea */}
                      <button
                        onClick={() => {
                          onSubmitIdea?.();
                          setIsExpanded(false);
                        }}
                        className="group relative w-24 h-24 cursor-pointer transition-transform hover:scale-105"
                      >
                        <div className="absolute inset-0 rounded-full bg-yellow-500/20 border-2 border-yellow-500/50 group-hover:bg-yellow-500/30 transition-all">
                          <div className="absolute inset-3 rounded-full bg-gradient-to-r from-yellow-500 to-yellow-600 flex items-center justify-center group-hover:from-yellow-400 group-hover:to-yellow-500 transition-all">
                            <Lightbulb className="w-6 h-6 text-white" />
                          </div>
                        </div>
                        <div className="absolute -bottom-12 left-1/2 transform -translate-x-1/2 text-xs text-yellow-400 font-medium whitespace-nowrap text-center">
                          <div>Submit Idea</div>
                          <div className="text-gray-500 text-[10px]">→ Train Property</div>
                        </div>
                      </button>
                    </div>
                  </div>
                )}
              </div>

              {/* Status Message */}
              <div className="space-y-2">
                <h2 className="text-2xl font-bold text-white">
                  {!isExpanded && status === 'idle' && 'Ready for Data Capture'}
                  {!isExpanded && status === 'processing' && 'Processing Request'}
                  {!isExpanded && status === 'listening' && 'Listening...'}
                  {!isExpanded && status === 'error' && 'Error Detected'}
                  {isExpanded && 'Choose Data Type'}
                </h2>
                <p className="text-gray-400 max-w-md mx-auto">
                  {!isExpanded && status === 'idle' && 'Train your own KNIRVCORTEX by capturing errors, context, or ideas. Each submission helps build your personalized SLM agent.'}
                  {!isExpanded && status === 'processing' && 'Training your KNIRVCORTEX: The Fabric algorithm is analyzing your input and generating specialized neural pathways.'}
                  {!isExpanded && status === 'listening' && 'Speak clearly to train your KNIRVCORTEX with voice commands and natural language patterns.'}
                  {!isExpanded && status === 'error' && 'Error detected - this will help train your KNIRVCORTEX to handle similar issues in the future.'}
                  {isExpanded && 'Train your KNIRVCORTEX: Errors become Skills, Context becomes Capabilities, Ideas become Properties.'}
                </p>
              </div>


            </div>
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
    </div>
  );
};
