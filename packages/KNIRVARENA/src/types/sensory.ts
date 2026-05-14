/**
 * TypeScript interfaces for sensory processing components
 * Replaces 'any' types in visual, audio, and other sensory processing
 */

// Visual processing interfaces
export interface VisualProcessingConfig {
  resolution: {
    width: number;
    height: number;
  };
  frameRate: number;
  enableObjectDetection: boolean;
  enableSceneAnalysis: boolean;
  enableFaceRecognition: boolean;
  modelPaths: {
    objectDetection?: string;
    sceneAnalysis?: string;
    faceRecognition?: string;
  };
  optimization: {
    useWebGL: boolean;
    batchSize: number;
    confidenceThreshold: number;
  };
}

export interface VisualFrame {
  imageData: ImageData;
  timestamp: Date;
  frameNumber: number;
  metadata?: {
    exposure?: number;
    brightness?: number;
    contrast?: number;
    saturation?: number;
  };
}

export interface ObjectDetectionResult {
  objects: DetectedObject[];
  processingTime: number;
  confidence: number;
  modelVersion: string;
  metadata?: {
    totalObjects: number;
    averageConfidence: number;
    processingMethod: string;
  };
}

export interface DetectedObject {
  id: string;
  type: string;
  label: string;
  confidence: number;
  boundingBox: BoundingBox;
  features: number[];
  attributes?: Record<string, unknown>;
  tracking?: {
    trackingId: string;
    velocity: { x: number; y: number };
    trajectory: Array<{ x: number; y: number; timestamp: Date }>;
  };
}

export interface BoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
  normalized?: boolean;
}

export interface SceneAnalysis {
  sceneType: string;
  confidence: number;
  objects: DetectedObject[];
  lighting: 'natural' | 'artificial' | 'mixed' | 'low' | 'bright';
  setting: 'indoor' | 'outdoor' | 'unknown';
  mood: 'positive' | 'negative' | 'neutral';
  complexity: number;
  timestamp: Date;
  metadata?: {
    dominantColors?: string[];
    textureAnalysis?: Record<string, number>;
    spatialLayout?: string;
    weatherConditions?: string;
  };
}

export interface FaceRecognitionResult {
  faces: DetectedFace[];
  processingTime: number;
  totalFaces: number;
  metadata?: {
    averageConfidence: number;
    recognizedFaces: number;
    unknownFaces: number;
  };
}

export interface DetectedFace {
  id: string;
  boundingBox: BoundingBox;
  confidence: number;
  landmarks: FaceLandmark[];
  identity?: {
    personId: string;
    name?: string;
    confidence: number;
  };
  attributes?: {
    age?: number;
    gender?: string;
    emotion?: string;
    glasses?: boolean;
    beard?: boolean;
  };
}

export interface FaceLandmark {
  type: 'eye' | 'nose' | 'mouth' | 'eyebrow' | 'chin';
  points: Array<{ x: number; y: number }>;
  confidence: number;
}

// Audio processing interfaces
export interface AudioProcessingConfig {
  sampleRate: number;
  channels: number;
  bitDepth: number;
  bufferSize: number;
  enableSpeechRecognition: boolean;
  enableSpeakerIdentification: boolean;
  enableEmotionDetection: boolean;
  noiseReduction: boolean;
  echoCancellation: boolean;
}

export interface AudioFrame {
  audioData: Float32Array;
  timestamp: Date;
  sampleRate: number;
  channels: number;
  duration: number;
  metadata?: {
    volume?: number;
    frequency?: number;
    quality?: number;
  };
}

export interface SpeechRecognitionResult {
  transcript: string;
  confidence: number;
  alternatives: Array<{
    transcript: string;
    confidence: number;
  }>;
  language: string;
  processingTime: number;
  metadata?: {
    wordCount: number;
    speakingRate: number;
    pauseCount: number;
    voiceActivity: boolean;
  };
}

export interface SpeakerIdentificationResult {
  speakerId: string;
  confidence: number;
  voiceprint: number[];
  characteristics: {
    gender?: string;
    ageRange?: string;
    accent?: string;
    emotionalState?: string;
  };
  metadata?: {
    voiceprintVersion: string;
    comparisonMethod: string;
    enrollmentStatus: boolean;
  };
}

export interface AudioEmotionResult {
  emotion: string;
  confidence: number;
  valence: number; // -1 to 1 (negative to positive)
  arousal: number; // 0 to 1 (calm to excited)
  dominance: number; // 0 to 1 (submissive to dominant)
  metadata?: {
    emotionDistribution: Record<string, number>;
    voiceFeatures: Record<string, number>;
    processingMethod: string;
  };
}

// Sensor data interfaces
export interface SensorReading {
  sensorId: string;
  type: 'accelerometer' | 'gyroscope' | 'magnetometer' | 'proximity' | 'light' | 'temperature' | 'pressure';
  value: number | number[] | Record<string, number>;
  unit: string;
  timestamp: Date;
  accuracy: number;
  metadata?: {
    calibrationStatus?: boolean;
    batteryLevel?: number;
    signalStrength?: number;
  };
}

export interface MotionData {
  acceleration: { x: number; y: number; z: number };
  rotation: { alpha: number; beta: number; gamma: number };
  orientation: string;
  timestamp: Date;
  confidence: number;
  metadata?: {
    deviceOrientation: string;
    movementType: string;
    intensity: number;
  };
}

export interface EnvironmentalData {
  temperature?: number;
  humidity?: number;
  pressure?: number;
  lightLevel?: number;
  noiseLevel?: number;
  airQuality?: number;
  timestamp: Date;
  location?: {
    latitude: number;
    longitude: number;
    altitude?: number;
    accuracy: number;
  };
}

// Processing pipeline interfaces
export interface SensoryProcessor<TInput, TOutput> {
  process(input: TInput): Promise<TOutput>;
  configure(config: Record<string, unknown>): void;
  getCapabilities(): string[];
  isReady(): boolean;
  getMetrics(): ProcessingMetrics;
}

export interface ProcessingMetrics {
  throughput: number; // items per second
  latency: number; // milliseconds
  accuracy: number; // 0 to 1
  errorRate: number; // 0 to 1
  resourceUsage: {
    cpu: number; // percentage
    memory: number; // bytes
    gpu?: number; // percentage
  };
  uptime: number; // milliseconds
}

export interface SensoryFusion {
  visualData?: VisualFrame;
  audioData?: AudioFrame;
  sensorData?: SensorReading[];
  environmentalData?: EnvironmentalData;
  timestamp: Date;
  correlationId: string;
  confidence: number;
  metadata?: {
    fusionMethod: string;
    dataQuality: Record<string, number>;
    synchronization: boolean;
  };
}

export interface MultimodalResult {
  visual?: ObjectDetectionResult | SceneAnalysis | FaceRecognitionResult;
  audio?: SpeechRecognitionResult | SpeakerIdentificationResult | AudioEmotionResult;
  motion?: MotionData;
  environmental?: EnvironmentalData;
  fusion?: {
    confidence: number;
    correlations: Record<string, number>;
    insights: string[];
  };
  timestamp: Date;
  processingTime: number;
}

// Event interfaces for sensory data
export interface SensoryEvent {
  type: 'visual' | 'audio' | 'motion' | 'environmental' | 'fusion';
  data: unknown;
  timestamp: Date;
  source: string;
  priority: 'low' | 'medium' | 'high' | 'critical';
  metadata?: Record<string, unknown>;
}

export interface SensoryEventHandler {
  canHandle(event: SensoryEvent): boolean;
  handle(event: SensoryEvent): Promise<void>;
  getPriority(): number;
}

// Type guards for sensory data
export function isVisualFrame(obj: unknown): obj is VisualFrame {
  return typeof obj === 'object' && obj !== null && 'imageData' in obj && 'timestamp' in obj;
}

export function isAudioFrame(obj: unknown): obj is AudioFrame {
  return typeof obj === 'object' && obj !== null && 'audioData' in obj && 'sampleRate' in obj;
}

export function isDetectedObject(obj: unknown): obj is DetectedObject {
  return typeof obj === 'object' && obj !== null && 'id' in obj && 'boundingBox' in obj && 'confidence' in obj;
}

export function isSensorReading(obj: unknown): obj is SensorReading {
  return typeof obj === 'object' && obj !== null && 'sensorId' in obj && 'type' in obj && 'value' in obj;
}
