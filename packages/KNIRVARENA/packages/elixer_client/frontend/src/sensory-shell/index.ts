// KNIRV Cognitive Shell - Main Export Module
// Month 7 Implementation of KNIRV_D-TEN Comprehensive Implementation Plan

// Core Engine
export { CognitiveEngine } from './CognitiveEngine';
export type { 
  CognitiveState, 
  LearningEvent, 
  CognitiveConfig 
} from './CognitiveEngine';

// SEAL Framework
export { SEALFramework } from './SEALFramework';
export type { 
  SEALConfig, 
  SEALAgent, 
  AgentPerformance, 
  SkillInvocation 
} from './SEALFramework';

// Fabric Algorithm
export { FabricAlgorithm } from './FabricAlgorithm';
export type { 
  FabricConfig, 
  FabricContext, 
  ProcessingMetrics, 
  AttentionMechanism 
} from './FabricAlgorithm';

// Voice Processing
export { VoiceProcessor } from './VoiceProcessor';
export type { 
  VoiceConfig, 
  SpeechRecognitionResult, 
  VoiceCommand 
} from './VoiceProcessor';

// Visual Processing
export { VisualProcessor } from './VisualProcessor';
export type { 
  VisualConfig, 
  DetectedObject, 
  BoundingBox, 
  GestureEvent, 
  OCRResult 
} from './VisualProcessor';

// LoRA Adapter
export { LoRAAdapter } from './LoRAAdapter';
export type {
  LoRAConfig,
  LoRAWeights,
  TrainingData,
  AdaptationMetrics
} from './LoRAAdapter';

// Event System
export { EventEmitter } from './EventEmitter';

// Demo System
export { CognitiveShellDemo, cognitiveDemo } from './demo';

// Version and metadata
export const COGNITIVE_SHELL_VERSION = '1.0.0';
export const IMPLEMENTATION_MONTH = 7;
export const KNIRV_D_TEN_COMPLIANCE = true;

/**
 * KNIRV Cognitive Shell Architecture
 * 
 * This module implements Month 7 of the KNIRV_D-TEN Comprehensive Implementation Plan,
 * providing a complete cognitive shell architecture with:
 * 
 * - Multi-modal input processing (voice, visual, text)
 * - Adaptive learning and skill invocation
 * - SEAL Framework for agent management
 * - Fabric Algorithm for intelligent processing
 * - LoRA adaptation for model personalization
 * - Real-time context management
 * - Browser-compatible event system
 * 
 * Key Features:
 * - Event-driven architecture
 * - Real-time adaptation
 * - Performance monitoring
 * - Context-aware processing
 * - Skill-based capabilities
 * - Learning from feedback
 * 
 * Usage:
 * ```typescript
 * import { CognitiveEngine, CognitiveConfig } from '@knirv/cognitive-shell';
 * 
 * const config: CognitiveConfig = {
 *   maxContextSize: 100,
 *   learningRate: 0.01,
 *   adaptationThreshold: 0.3,
 *   skillTimeout: 30000,
 *   voiceEnabled: true,
 *   visualEnabled: true,
 *   loraEnabled: true,
 * };
 * 
 * const engine = new CognitiveEngine(config);
 * await engine.start();
 * 
 * // Process input
 * const result = await engine.processInput('analyze system performance', 'text');
 * 
 * // Invoke skills
 * const skillResult = await engine.invokeSkill('text_analysis', { text: 'sample' });
 * 
 * // Enable learning
 * await engine.startLearningMode();
 * 
 * // Provide feedback
 * engine.provideFeedback(0, 0.8);
 * ```
 * 
 * @version 1.0.0
 * @author KNIRV Development Team
 * @license MIT
 */
