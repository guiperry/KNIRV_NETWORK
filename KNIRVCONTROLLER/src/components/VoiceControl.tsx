import * as React from 'react';
import { useState, useEffect, useRef, useCallback } from 'react';
import { Mic, MicOff, Volume2, Brain } from 'lucide-react';
import { VoiceProcessor, VoiceConfig } from '../sensory-shell/VoiceProcessor';

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

  const initializeVoiceProcessor = useCallback(async () => {
    try {
      const config: VoiceConfig = {
        sampleRate: 16000,
        channels: 1,
        bufferSize: 4096,
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
      processor.on('speechDetected', (speech: unknown) => {
        const speechData = speech as { text: string; confidence: number };
        setTranscript(speechData.text);
        setConfidence(speechData.confidence);
        onVoiceCommand(speechData.text);
      });

      processor.on('commandRecognized', (command: unknown) => {
        const commandData = command as { originalText: string };
        console.log('Voice command recognized:', commandData);
        onVoiceCommand(commandData.originalText);
      });

      processor.on('recognitionStarted', () => {
        setIsListening(true);
      });

      processor.on('recognitionEnded', () => {
        setIsListening(false);
      });

      processor.on('recognitionError', (error) => {
        console.error('Voice recognition error:', error);
        setIsListening(false);
      });

    } catch (error) {
      console.error('Failed to initialize voice processor:', error);
      setIsSupported(false);
    }
  }, [onVoiceCommand]);

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
  }, [cognitiveMode, initializeVoiceProcessor, voiceProcessor]);

  const simulateVoiceRecognition = useCallback(() => {
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
      'Check system performance',
      'Start learning mode',
      'Save current adaptation',
      'Invoke skill analysis',
      'Toggle visual processing'
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
  }, [isActive, onVoiceCommand]);

  const handleRealVoiceToggle = useCallback(async (active: boolean) => {
    if (!voiceProcessor) return;

    try {
      if (active) {
        await voiceProcessor.start();
      } else {
        await voiceProcessor.stop();
      }
    } catch (error) {
      console.error('Error toggling voice processor:', error);
    }
  }, [voiceProcessor]);

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
  }, [isActive, cognitiveMode, voiceProcessor, handleRealVoiceToggle, simulateVoiceRecognition]);

  return (
    <div className="absolute bottom-4 right-4 z-40" data-testid="voice-control">
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
          className={`w-14 h-14 rounded-full flex items-center justify-center transition-all duration-300 relative ${
            isActive
              ? 'bg-teal-500 text-white shadow-lg shadow-teal-500/30'
              : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
          }`}
          data-testid="voice-control-toggle"
        >
          {isActive ? (
            <Mic className="w-6 h-6" />
          ) : (
            <MicOff className="w-6 h-6" />
          )}

          {/* Cognitive Mode Indicator */}
          {cognitiveMode && (
            <div className="absolute -top-1 -right-1 w-4 h-4 bg-purple-500 rounded-full flex items-center justify-center">
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