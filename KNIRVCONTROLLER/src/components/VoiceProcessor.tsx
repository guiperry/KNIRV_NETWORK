import { useState, useRef, useEffect, useCallback } from 'react';
import { Mic, MicOff, Volume2, VolumeX } from 'lucide-react';

interface VoiceProcessorProps {
  onVoiceCommand: (command: string, confidence: number) => void;
  onAudioData: (audioData: Float32Array) => void;
  isActive: boolean;
}

interface SpeechRecognitionEvent extends Event {
  results: SpeechRecognitionResultList;
  resultIndex: number;
}

interface SpeechRecognitionErrorEvent extends Event {
  error: string;
  message: string;
}

interface SpeechRecognition extends EventTarget {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  maxAlternatives: number;
  serviceURI: string;
  grammars: unknown; // SpeechGrammarList not widely supported
  start(): void;
  stop(): void;
  abort(): void;
  onaudiostart: ((this: SpeechRecognition, ev: Event) => void) | null;
  onaudioend: ((this: SpeechRecognition, ev: Event) => void) | null;
  onend: ((this: SpeechRecognition, ev: Event) => void) | null;
  onerror: ((this: SpeechRecognition, ev: SpeechRecognitionErrorEvent) => void) | null;
  onnomatch: ((this: SpeechRecognition, ev: SpeechRecognitionEvent) => void) | null;
  onresult: ((this: SpeechRecognition, ev: SpeechRecognitionEvent) => void) | null;
  onsoundstart: ((this: SpeechRecognition, ev: Event) => void) | null;
  onsoundend: ((this: SpeechRecognition, ev: Event) => void) | null;
  onspeechstart: ((this: SpeechRecognition, ev: Event) => void) | null;
  onspeechend: ((this: SpeechRecognition, ev: Event) => void) | null;
  onstart: ((this: SpeechRecognition, ev: Event) => void) | null;
}

declare global {
  interface Window {
    SpeechRecognition: {
      new (): SpeechRecognition;
    };
    webkitSpeechRecognition: {
      new (): SpeechRecognition;
    };
  }
}

export default function VoiceProcessor({ onVoiceCommand, onAudioData, isActive }: VoiceProcessorProps) {
  const [isListening, setIsListening] = useState(false);
  const [isRecording, setIsRecording] = useState(false);
  const [audioLevel, setAudioLevel] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [lastCommand, setLastCommand] = useState<string>('');
  const [transcript, setTranscript] = useState<string>('');
  
  const recognitionRef = useRef<SpeechRecognition | null>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const microphoneRef = useRef<MediaStreamAudioSourceNode | null>(null);
  const streamRef = useRef<MediaStream | null>(null);

  const processVoiceCommand = useCallback((transcript: string, _confidence: number) => {
    const command = transcript.toLowerCase().trim();

    // Basic command processing
    if (command.includes('scan qr') || command.includes('scan code')) {
      console.log('QR scan command detected');
    } else if (command.includes('connect') || command.includes('link')) {
      console.log('Connection command detected');
    } else if (command.includes('wallet') || command.includes('transaction')) {
      console.log('Wallet command detected');
    }
  }, []);

  const stopVoiceRecognition = useCallback(() => {
    if (recognitionRef.current) {
      recognitionRef.current.stop();
      recognitionRef.current = null;
    }
    setIsListening(false);
  }, []);

  const stopAudioAnalysis = useCallback(() => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach(track => track.stop());
      streamRef.current = null;
    }
    setIsRecording(false);
    setAudioLevel(0);
  }, []);

  const initializeVoiceRecognition = useCallback(() => {
    if (!('SpeechRecognition' in window) && !('webkitSpeechRecognition' in window)) {
      setError('Speech recognition not supported in this browser');
      return;
    }

    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    const recognition = new SpeechRecognition();

    recognition.continuous = true;
    recognition.interimResults = true;
    recognition.lang = 'en-US';

    recognition.onstart = () => {
      setIsListening(true);
      setError(null);
    };

    recognition.onresult = (event) => {
      let finalTranscript = '';
      let interimTranscript = '';

      for (let i = event.resultIndex; i < event.results.length; i++) {
        const transcript = event.results[i][0].transcript;
        const confidence = event.results[i][0].confidence;

        if (event.results[i].isFinal) {
          finalTranscript += transcript;
          processVoiceCommand(transcript, confidence);
          onVoiceCommand(transcript, confidence);
        } else {
          interimTranscript += transcript;
        }
      }

      setTranscript(finalTranscript || interimTranscript);
    };

    recognition.onerror = (event) => {
      console.error('Speech recognition error:', event.error);
      setError(`Speech recognition error: ${event.error}`);
      setIsListening(false);
    };

    recognition.onend = () => {
      setIsListening(false);
      if (isActive) {
        // Restart recognition if still active
        setTimeout(() => recognition.start(), 100);
      }
    };

    recognitionRef.current = recognition;
    recognition.start();
  }, [isActive, onVoiceCommand, processVoiceCommand]);

  const initializeAudioAnalysis = useCallback(async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true
        }
      });

      streamRef.current = stream;
      setIsRecording(true);
      setError(null);

      // Create audio context for analysis
      const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
      const analyser = audioContext.createAnalyser();
      const source = audioContext.createMediaStreamSource(stream);

      source.connect(analyser);
      analyser.fftSize = 256;

      const bufferLength = analyser.frequencyBinCount;
      const dataArray = new Uint8Array(bufferLength);
      const floatArray = new Float32Array(bufferLength);

      const updateAudioLevel = () => {
        if (!isActive || !streamRef.current) return;

        analyser.getByteFrequencyData(dataArray);
        analyser.getFloatFrequencyData(floatArray);

        // Calculate average volume
        let sum = 0;
        for (let i = 0; i < bufferLength; i++) {
          sum += dataArray[i];
        }
        const average = sum / bufferLength;
        const normalizedLevel = average / 255;

        setAudioLevel(normalizedLevel);

        // Send audio data to parent
        onAudioData(floatArray);

        if (isActive) {
          requestAnimationFrame(updateAudioLevel);
        }
      };

      updateAudioLevel();
    } catch (error) {
      console.error('Failed to initialize audio analysis:', error);
      setError('Failed to access microphone. Please check permissions.');
    }
  }, [isActive, onAudioData]);

  useEffect(() => {
    if (isActive) {
      initializeVoiceRecognition();
      initializeAudioAnalysis();
    } else {
      stopVoiceRecognition();
      stopAudioAnalysis();
    }

    return () => {
      stopVoiceRecognition();
      stopAudioAnalysis();
    };
  }, [isActive, initializeAudioAnalysis, initializeVoiceRecognition, stopAudioAnalysis, stopVoiceRecognition]);

  const startAudioLevelMonitoring = useCallback(() => {
    if (!analyserRef.current) return;

    const analyser = analyserRef.current;
    const bufferLength = analyser.frequencyBinCount;
    const dataArray = new Uint8Array(bufferLength);
    const floatArray = new Float32Array(bufferLength);

    const updateAudioLevel = () => {
      if (!analyserRef.current || !isActive) return;

      analyser.getByteFrequencyData(dataArray);
      analyser.getFloatFrequencyData(floatArray);

      // Calculate audio level
      const sum = dataArray.reduce((a: number, b: number) => a + b, 0);
      const average = sum / bufferLength;
      const level = average / 255;

      setAudioLevel(level);

      // Send audio data for processing
      onAudioData(floatArray);

      requestAnimationFrame(updateAudioLevel);
    };

    updateAudioLevel();
  }, [isActive, onAudioData]);



  const toggleVoiceRecognition = () => {
    if (isListening) {
      stopVoiceRecognition();
    } else {
      initializeVoiceRecognition();
    }
  };

  return (
    <div className="bg-white rounded-lg shadow-lg p-4 m-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-800">Voice Processing</h3>
        <div className="flex items-center gap-2">
          {isRecording ? <Volume2 className="text-green-500" size={20} /> : <VolumeX className="text-gray-400" size={20} />}
          <button
            onClick={toggleVoiceRecognition}
            className={`p-2 rounded-full transition-colors ${
              isListening 
                ? 'bg-red-500 hover:bg-red-600 text-white' 
                : 'bg-blue-500 hover:bg-blue-600 text-white'
            }`}
          >
            {isListening ? <MicOff size={20} /> : <Mic size={20} />}
          </button>
        </div>
      </div>

      {/* Audio Level Indicator */}
      <div className="mb-4">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-sm text-gray-600">Audio Level:</span>
          <div className="flex-1 bg-gray-200 rounded-full h-2">
            <div 
              className="bg-green-500 h-2 rounded-full transition-all duration-100"
              style={{ width: `${audioLevel * 100}%` }}
            />
          </div>
        </div>
      </div>

      {/* Status */}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-600">Status:</span>
          <span className={`text-sm font-medium ${isListening ? 'text-green-600' : 'text-gray-400'}`}>
            {isListening ? 'Listening...' : 'Inactive'}
          </span>
        </div>
        
        {lastCommand && (
          <div className="text-sm">
            <span className="text-gray-600">Last Command:</span>
            <span className="ml-2 text-blue-600 font-medium">{lastCommand}</span>
          </div>
        )}
        
        {error && (
          <div className="text-sm text-red-600">
            {error}
          </div>
        )}
      </div>
    </div>
  );
}
