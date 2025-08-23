import React, { useState, useEffect, useRef } from 'react';
import { Mic, MicOff, Volume2, Brain } from 'lucide-react';
import { VoiceProcessor, VoiceConfig } from '../../shared/cognitive-shell/VoiceProcessor';

interface VoiceControlProps {
  isActive: boolean;
  onVoiceCommand: (command: string) => void;
  onToggle: (active: boolean) => void;
  cognitiveMode?: boolean;
}

export const VoiceControl: React.FC<VoiceControlProps> = ({
  isActive,
  onVoiceCommand,
  onToggle,
  cognitiveMode = false
}) => {
  const [isListening, setIsListening] = useState(false);
  const [transcript, setTranscript] = useState('');
  const [confidence, setConfidence] = useState(0);
  const [voiceProcessor, setVoiceProcessor] = useState<VoiceProcessor | null>(null);
  const [isSupported, setIsSupported] = useState(false);
  const processorRef = useRef<VoiceProcessor | null>(null);

  // Initialize voice processor when cognitive mode is enabled
  useEffect(() => {
    if (cognitiveMode && !voiceProcessor) {
      initializeVoiceProcessor();
    }

    return () => {
      if (processorRef.current) {
        processorRef.current.stop();
      }
    };
  }, [cognitiveMode]);

  const initializeVoiceProcessor = async () => {
    try {
      const config: VoiceConfig = {
        sampleRate: 16000,
        channels: 1,
        language: 'en-US',
        enableWakeWord: true,
        wakeWord: 'knirv',
        noiseReduction: true,
      };

      const processor = new VoiceProcessor(config);
      processorRef.current = processor;
      setVoiceProcessor(processor);
      setIsSupported(processor.isSupported());

      // Set up event listeners
      processor.on('speechDetected', (speech: any) => {
        setTranscript(speech.text);
        setConfidence(speech.confidence);
        onVoiceCommand(speech.text);
      });

      processor.on('commandRecognized', (command: any) => {
        console.log('Voice command recognized:', command);
        onVoiceCommand(command.originalText);
      });

      processor.on('recognitionStarted', () => {
        setIsListening(true);
      });

      processor.on('recognitionEnded', () => {
        setIsListening(false);
      });

      processor.on('recognitionError', (error: any) => {
        console.error('Voice recognition error:', error);
        setIsListening(false);
      });

    } catch (error) {
      console.error('Failed to initialize voice processor:', error);
      setIsSupported(false);
    }
  };

  const simulateVoiceRecognition = () => {
    if (!isActive) return;

    setIsListening(true);
    setTranscript('');
    setConfidence(0);

    // Simulate voice recognition process
    const phrases = [
      'Show skills page',
      'Navigate to wallet',
      'Check agent status',
      'Open UDC panel',
      'Show network status',
      'Activate skill analysis',
      'Check NRN balance',
      'Toggle voice mode'
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
      if (cognitiveMode && voiceProcessor) {
        // Use real voice processor
        handleRealVoiceToggle(true);
      } else {
        // Use simulation
        const interval = setInterval(() => {
          if (Math.random() < 0.3) { // 30% chance to trigger voice recognition
            simulateVoiceRecognition();
          }
        }, 3000);

        return () => clearInterval(interval);
      }
    } else if (voiceProcessor) {
      handleRealVoiceToggle(false);
    }
  }, [isActive, cognitiveMode, voiceProcessor]);

  const handleRealVoiceToggle = async (active: boolean) => {
    if (!voiceProcessor) return;

    try {
      if (active) {
        await voiceProcessor.start();
      } else {
        await voiceProcessor.stop();
      }
    } catch (error) {
      console.error('Voice processor toggle error:', error);
    }
  };

  return (
    <div className="absolute bottom-20 right-4 z-40">
      <div className="flex flex-col items-end space-y-2">
        {/* Voice Transcript */}
        {(isListening || transcript) && (
          <div className="bg-slate-800/90 backdrop-blur-sm rounded-lg p-3 max-w-xs border border-slate-700/50">
            <div className="flex items-center space-x-2 mb-2">
              <Volume2 className="w-4 h-4 text-purple-400" />
              <span className="text-xs text-slate-400">
                Confidence: {Math.round(confidence * 100)}%
              </span>
            </div>
            <p className="text-sm text-white">{transcript || 'Listening...'}</p>
            {isListening && (
              <div className="flex space-x-1 mt-2">
                <div className="w-2 h-2 bg-purple-400 rounded-full animate-pulse"></div>
                <div className="w-2 h-2 bg-purple-400 rounded-full animate-pulse" style={{ animationDelay: '0.2s' }}></div>
                <div className="w-2 h-2 bg-purple-400 rounded-full animate-pulse" style={{ animationDelay: '0.4s' }}></div>
              </div>
            )}
          </div>
        )}

        {/* Voice Toggle Button */}
        <button
          onClick={() => onToggle(!isActive)}
          className={`w-14 h-14 rounded-full flex items-center justify-center transition-all duration-300 relative ${
            isActive
              ? 'bg-purple-500 text-white shadow-lg shadow-purple-500/30'
              : 'bg-slate-700 text-slate-400 hover:bg-slate-600'
          }`}
        >
          {isActive ? (
            <Mic className="w-6 h-6" />
          ) : (
            <MicOff className="w-6 h-6" />
          )}

          {/* Cognitive Mode Indicator */}
          {cognitiveMode && (
            <div className="absolute -top-1 -right-1 w-4 h-4 bg-cyan-500 rounded-full flex items-center justify-center">
              <Brain className="w-2 h-2 text-white" />
            </div>
          )}

          {/* Unsupported Indicator */}
          {cognitiveMode && !isSupported && (
            <div className="absolute -bottom-1 -right-1 w-3 h-3 bg-red-500 rounded-full"></div>
          )}
        </button>
      </div>
    </div>
  );
};
