import { EventEmitter } from './EventEmitter';

// Speech Recognition API types
interface SpeechRecognitionEvent {
  results: SpeechRecognitionResultList;
  resultIndex: number;
}

interface SpeechRecognitionErrorEvent {
  error: string;
  message?: string;
}

interface SpeechRecognitionInterface {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  onstart: (() => void) | null;
  onresult: ((event: SpeechRecognitionEvent) => void) | null;
  onerror: ((event: SpeechRecognitionErrorEvent) => void) | null;
  onend: (() => void) | null;
  start(): void;
  stop(): void;
}

interface SpeechRecognitionConstructor {
  new(): SpeechRecognitionInterface;
}

export interface VoiceConfig {
  sampleRate: number;
  channels: number;
  language: string;
  enableWakeWord: boolean;
  wakeWord?: string;
  noiseReduction: boolean;
}

export interface SpeechRecognitionResult {
  text: string;
  confidence: number;
  language: string;
  timestamp: Date;
  duration: number;
}

export interface VoiceCommand {
  type: string;
  parameters: unknown;
  confidence: number;
  originalText: string;
}

export class VoiceProcessor extends EventEmitter {
  private config: VoiceConfig;
  private isListening: boolean = false;
  private isRecording: boolean = false;
  private mediaRecorder: MediaRecorder | null = null;
  private audioContext: AudioContext | null = null;
  private recognition: SpeechRecognitionInterface | null = null;
  private synthesis: SpeechSynthesis | null = null;

  constructor(config: VoiceConfig) {
    super();
    this.config = config;
    this.initializeWebAPIs();
  }

  private initializeWebAPIs(): void {
    // Initialize Web Speech API
    if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
      const windowWithSpeech = window as { SpeechRecognition?: SpeechRecognitionConstructor; webkitSpeechRecognition?: SpeechRecognitionConstructor };
      const SpeechRecognition = windowWithSpeech.SpeechRecognition || windowWithSpeech.webkitSpeechRecognition;
      if (SpeechRecognition) {
        this.recognition = new SpeechRecognition();

        this.recognition.continuous = true;
        this.recognition.interimResults = true;
        this.recognition.lang = this.config.language;

        this.setupRecognitionHandlers();
      }
    }

    // Initialize Speech Synthesis
    if ('speechSynthesis' in window) {
      this.synthesis = window.speechSynthesis;
    }

    // Initialize Audio Context
    if ('AudioContext' in window || 'webkitAudioContext' in window) {
      const windowWithAudio = window as { AudioContext?: typeof AudioContext; webkitAudioContext?: typeof AudioContext };
      const AudioContextConstructor = windowWithAudio.AudioContext || windowWithAudio.webkitAudioContext;
      if (AudioContextConstructor) {
        this.audioContext = new AudioContextConstructor();
      }
    }
  }

  private setupRecognitionHandlers(): void {
    if (!this.recognition) return;

    this.recognition.onstart = () => {
      console.log('Speech recognition started');
      this.emit('recognitionStarted');
    };

    this.recognition.onresult = (event: SpeechRecognitionEvent) => {
      const results = Array.from(event.results);
      const latestResult = results[results.length - 1];

      if (latestResult.isFinal) {
        const result: SpeechRecognitionResult = {
          text: latestResult[0].transcript,
          confidence: latestResult[0].confidence,
          language: this.config.language,
          timestamp: new Date(),
          duration: 0, // Would be calculated from audio
        };

        this.processSpeechResult(result);
      }
    };

    this.recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      console.error('Speech recognition error:', event.error);
      this.emit('recognitionError', event.error);
    };

    this.recognition.onend = () => {
      console.log('Speech recognition ended');
      this.emit('recognitionEnded');

      // Restart if still listening
      if (this.isListening) {
        setTimeout(() => {
          if (this.isListening && this.recognition) {
            this.recognition.start();
          }
        }, 100);
      }
    };
  }

  public async start(): Promise<void> {
    console.log('Starting Voice Processor...');

    try {
      // Request microphone permission
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          sampleRate: this.config.sampleRate,
          channelCount: this.config.channels,
          echoCancellation: true,
          noiseSuppression: this.config.noiseReduction,
        }
      });

      // Initialize MediaRecorder
      this.mediaRecorder = new MediaRecorder(stream);
      this.setupMediaRecorderHandlers();

      // Start speech recognition
      if (this.recognition) {
        this.isListening = true;
        this.recognition.start();
      }

      this.emit('voiceProcessorStarted');
      console.log('Voice Processor started successfully');

    } catch (error) {
      console.error('Failed to start Voice Processor:', error);
      throw error;
    }
  }

  public async stop(): Promise<void> {
    console.log('Stopping Voice Processor...');

    this.isListening = false;

    if (this.recognition) {
      this.recognition.stop();
    }

    if (this.mediaRecorder && this.isRecording) {
      this.mediaRecorder.stop();
    }

    if (this.audioContext) {
      await this.audioContext.close();
    }

    this.emit('voiceProcessorStopped');
    console.log('Voice Processor stopped');
  }

  private setupMediaRecorderHandlers(): void {
    if (!this.mediaRecorder) return;

    this.mediaRecorder.ondataavailable = (event) => {
      if (event.data.size > 0) {
        this.processAudioData(event.data);
      }
    };

    this.mediaRecorder.onstart = () => {
      this.isRecording = true;
      this.emit('recordingStarted');
    };

    this.mediaRecorder.onstop = () => {
      this.isRecording = false;
      this.emit('recordingStopped');
    };
  }

  private async processAudioData(audioData: Blob): Promise<void> {
    // Process raw audio data for additional analysis
    // This could include noise detection, volume analysis, etc.

    const audioBuffer = await audioData.arrayBuffer();

    this.emit('audioDataProcessed', {
      size: audioBuffer.byteLength,
      timestamp: new Date(),
    });
  }

  private processSpeechResult(result: SpeechRecognitionResult): void {
    console.log('Speech recognized:', result.text);

    // Check for wake word if enabled
    if (this.config.enableWakeWord && this.config.wakeWord) {
      if (!result.text.toLowerCase().includes(this.config.wakeWord.toLowerCase())) {
        return; // Ignore if wake word not detected
      }
    }

    // Emit speech detection event
    this.emit('speechDetected', result);

    // Try to parse as command
    const command = this.parseVoiceCommand(result.text);
    if (command) {
      this.emit('commandRecognized', command);
    }
  }

  private parseVoiceCommand(text: string): VoiceCommand | null {
    const lowerText = text.toLowerCase().trim();

    // Define command patterns
    const commandPatterns = [
      {
        pattern: /invoke skill (.+)/,
        type: 'invoke_skill',
        extractor: (match: RegExpMatchArray) => ({
          skillId: match[1],
        }),
      },
      {
        pattern: /start learning/,
        type: 'start_learning',
        extractor: () => ({}),
      },
      {
        pattern: /save adaptation/,
        type: 'save_adaptation',
        extractor: () => ({}),
      },
      {
        pattern: /show (.+)/,
        type: 'show_interface',
        extractor: (match: RegExpMatchArray) => ({
          target: match[1],
        }),
      },
      {
        pattern: /help with (.+)/,
        type: 'request_help',
        extractor: (match: RegExpMatchArray) => ({
          topic: match[1],
        }),
      },
      {
        pattern: /analyze (.+)/,
        type: 'analyze_input',
        extractor: (match: RegExpMatchArray) => ({
          target: match[1],
        }),
      },
      {
        pattern: /capture screen/,
        type: 'capture_screen',
        extractor: () => ({}),
      },
      {
        pattern: /toggle network/,
        type: 'toggle_network',
        extractor: () => ({}),
      },
    ];

    // Try to match command patterns
    for (const pattern of commandPatterns) {
      const match = lowerText.match(pattern.pattern);
      if (match) {
        return {
          type: pattern.type,
          parameters: pattern.extractor(match),
          confidence: 0.8, // Could be improved with ML
          originalText: text,
        };
      }
    }

    return null;
  }

  public async speak(text: string, options: unknown = {}): Promise<void> {
    if (!this.synthesis) {
      throw new Error('Speech synthesis not available');
    }

    return new Promise((resolve, reject) => {
      const utterance = new SpeechSynthesisUtterance(text);

      const opts = options as { language?: string; rate?: number; pitch?: number; volume?: number };
      utterance.lang = opts?.language || this.config.language;
      utterance.rate = opts?.rate || 1.0;
      utterance.pitch = opts?.pitch || 1.0;
      utterance.volume = opts?.volume || 1.0;

      utterance.onend = () => {
        this.emit('speechEnded', { text });
        resolve();
      };

      utterance.onerror = (event) => {
        this.emit('speechError', event);
        reject(new Error(`Speech synthesis error: ${event.error}`));
      };

      utterance.onstart = () => {
        this.emit('speechStarted', { text });
      };

      this.synthesis!.speak(utterance);
    });
  }

  public startRecording(): void {
    if (this.mediaRecorder && !this.isRecording) {
      this.mediaRecorder.start(1000); // Collect data every second
    }
  }

  public stopRecording(): void {
    if (this.mediaRecorder && this.isRecording) {
      this.mediaRecorder.stop();
    }
  }

  public setLanguage(language: string): void {
    this.config.language = language;
    if (this.recognition) {
      this.recognition.lang = language;
    }
  }

  public enableWakeWord(wakeWord: string): void {
    this.config.enableWakeWord = true;
    this.config.wakeWord = wakeWord;
  }

  public disableWakeWord(): void {
    this.config.enableWakeWord = false;
    this.config.wakeWord = undefined;
  }

  public getAvailableVoices(): SpeechSynthesisVoice[] {
    if (!this.synthesis) {
      return [];
    }
    return this.synthesis.getVoices();
  }

  public isSupported(): boolean {
    return !!(this.recognition && this.synthesis && this.audioContext);
  }

  public getMetrics(): unknown {
    return {
      isListening: this.isListening,
      isRecording: this.isRecording,
      isSupported: this.isSupported(),
      language: this.config.language,
      wakeWordEnabled: this.config.enableWakeWord,
    };
  }
}
