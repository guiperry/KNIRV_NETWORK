import React, { useState, useEffect } from 'react';
import { Mic, MicOff, Volume2 } from 'lucide-react';

interface VoiceControlProps {
  isActive: boolean;
  onVoiceCommand: (command: string) => void;
  onToggle: (active: boolean) => void;
}

export const VoiceControl: React.FC<VoiceControlProps> = ({
  isActive,
  onVoiceCommand,
  onToggle
}) => {
  const [isListening, setIsListening] = useState(false);
  const [transcript, setTranscript] = useState('');
  const [confidence, setConfidence] = useState(0);

  const simulateVoiceRecognition = () => {
    if (!isActive) return;

    setIsListening(true);
    setTranscript('');
    setConfidence(0);

    // Simulate voice recognition process
    const phrases = [
      'Identify problems in the interface',
      'Show network status',
      'Assign agents to fix this',
      'Capture screenshot and analyze',
      'Map NRV to graph',
      'Check system performance'
    ];

    let currentText = '';
    const targetPhrase = phrases[Math.floor(Math.random() * phrases.length)];
    
    const words = targetPhrase.split(' ');
    let wordIndex = 0;

    const addWord = () => {
      if (wordIndex < words.length) {
        currentText += (wordIndex > 0 ? ' ' : '') + words[wordIndex];
        setTranscript(currentText);
        setConfidence(Math.min(0.95, 0.3 + (wordIndex / words.length) * 0.65));
        wordIndex++;
        setTimeout(addWord, 200 + Math.random() * 300);
      } else {
        setIsListening(false);
        setTimeout(() => {
          onVoiceCommand(currentText);
          setTranscript('');
          setConfidence(0);
        }, 500);
      }
    };

    setTimeout(addWord, 500);
  };

  useEffect(() => {
    if (isActive) {
      const interval = setInterval(() => {
        if (Math.random() < 0.3) { // 30% chance to trigger voice recognition
          simulateVoiceRecognition();
        }
      }, 3000);

      return () => clearInterval(interval);
    }
  }, [isActive]);

  return (
    <div className="absolute bottom-4 right-4 z-40">
      <div className="flex flex-col items-end space-y-2">
        {/* Voice Transcript */}
        {(isListening || transcript) && (
          <div className="bg-gray-800/90 backdrop-blur-sm rounded-lg p-3 max-w-xs border border-gray-700/50">
            <div className="flex items-center space-x-2 mb-2">
              <Volume2 className="w-4 h-4 text-teal-400" />
              <span className="text-xs text-gray-400">
                Confidence: {Math.round(confidence * 100)}%
              </span>
            </div>
            <p className="text-sm text-white">{transcript || 'Listening...'}</p>
            {isListening && (
              <div className="flex space-x-1 mt-2">
                <div className="w-2 h-2 bg-teal-400 rounded-full animate-pulse"></div>
                <div className="w-2 h-2 bg-teal-400 rounded-full animate-pulse" style={{ animationDelay: '0.2s' }}></div>
                <div className="w-2 h-2 bg-teal-400 rounded-full animate-pulse" style={{ animationDelay: '0.4s' }}></div>
              </div>
            )}
          </div>
        )}

        {/* Voice Toggle Button */}
        <button
          onClick={() => onToggle(!isActive)}
          className={`w-14 h-14 rounded-full flex items-center justify-center transition-all duration-300 ${
            isActive 
              ? 'bg-teal-500 text-white shadow-lg shadow-teal-500/30' 
              : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
          }`}
        >
          {isActive ? (
            <Mic className="w-6 h-6" />
          ) : (
            <MicOff className="w-6 h-6" />
          )}
        </button>
      </div>
    </div>
  );
};