import { EventEmitter } from './EventEmitter';
import { SEALFramework } from './SEALFramework';
import { FabricAlgorithm } from './FabricAlgorithm';
import { VoiceProcessor } from './VoiceProcessor';
import { VisualProcessor } from './VisualProcessor';
import { LoRAAdapter } from './LoRAAdapter';
import { EnhancedLoRAAdapter } from './EnhancedLoRAAdapter';
import { HRMBridge, HRMConfig } from './HRMBridge';
import { HRMLoRABridge } from './HRMLoRABridge';
import { AdaptiveLearningPipeline } from './AdaptiveLearningPipeline';
import { KNIRVWalletIntegration } from './KNIRVWalletIntegration';
import { KNIRVChainIntegration } from './KNIRVChainIntegration';
import { EcosystemCommunicationLayer } from './EcosystemCommunicationLayer';

export interface CognitiveState {
  currentContext: Map<string, any>;
  activeSkills: string[];
  learningHistory: LearningEvent[];
  confidenceLevel: number;
  adaptationLevel: number;
}

export interface LearningEvent {
  timestamp: Date;
  eventType: string;
  input: any;
  output: any;
  feedback: number; // -1 to 1
  adaptationApplied: boolean;
}

export interface CognitiveConfig {
  maxContextSize: number;
  learningRate: number;
  adaptationThreshold: number;
  skillTimeout: number;
  voiceEnabled: boolean;
  visualEnabled: boolean;
  loraEnabled: boolean;
  enhancedLoraEnabled: boolean;
  hrmEnabled: boolean;
  hrmConfig?: HRMConfig;
  adaptiveLearningEnabled: boolean;
  walletIntegrationEnabled: boolean;
  chainIntegrationEnabled: boolean;
  ecosystemCommunicationEnabled: boolean;
}

export class CognitiveEngine extends EventEmitter {
  private state: CognitiveState;
  private config: CognitiveConfig;
  private sealFramework: SEALFramework;
  private fabricAlgorithm: FabricAlgorithm;
  private voiceProcessor: VoiceProcessor;
  private visualProcessor: VisualProcessor;
  private loraAdapter: LoRAAdapter;
  private enhancedLoraAdapter: EnhancedLoRAAdapter;
  private hrmBridge: HRMBridge;
  private hrmLoraBridge: HRMLoRABridge;
  private adaptiveLearningPipeline: AdaptiveLearningPipeline;
  private walletIntegration: KNIRVWalletIntegration;
  private chainIntegration: KNIRVChainIntegration;
  private ecosystemCommunication: EcosystemCommunicationLayer;
  private isRunning: boolean = false;

  constructor(config: CognitiveConfig) {
    super();
    this.config = config;
    this.state = {
      currentContext: new Map(),
      activeSkills: [],
      learningHistory: [],
      confidenceLevel: 0.5,
      adaptationLevel: 0.0,
    };

    this.initializeComponents();
  }

  private async initializeComponents(): Promise<void> {
    // Initialize SEAL Framework
    this.sealFramework = new SEALFramework({
      maxAgents: 10,
      learningRate: this.config.learningRate,
      adaptationThreshold: this.config.adaptationThreshold,
      skillTimeout: this.config.skillTimeout,
      hrmIntegration: this.config.hrmEnabled,
    });

    // Initialize Fabric Algorithm
    this.fabricAlgorithm = new FabricAlgorithm({
      contextSize: this.config.maxContextSize,
      processingMode: 'adaptive',
      memoryDepth: 50,
      attentionHeads: 8,
      learningRate: this.config.learningRate,
      hrmIntegration: this.config.hrmEnabled,
    });

    // Initialize input processors
    if (this.config.voiceEnabled) {
      this.voiceProcessor = new VoiceProcessor({
        sampleRate: 16000,
        channels: 1,
        language: 'en-US',
        enableWakeWord: true,
        wakeWord: 'knirv',
        noiseReduction: true,
      });
    }

    if (this.config.visualEnabled) {
      this.visualProcessor = new VisualProcessor({
        resolution: '1920x1080',
        frameRate: 30,
        objectDetection: true,
        faceRecognition: true,
        gestureRecognition: true,
        ocrEnabled: true,
        enableSceneAnalysis: true,
        enableHRMGuidance: this.config.hrmEnabled,
        maxImageSize: 1024,
        confidenceThreshold: 0.5,
        enableRealTimeProcessing: true,
      });
    }

    // Initialize LoRA adapter
    if (this.config.loraEnabled) {
      this.loraAdapter = new LoRAAdapter({
        rank: 16,
        alpha: 32,
        dropout: 0.1,
        targetModules: ['attention', 'feedforward'],
        taskType: 'cognitive_processing',
      });
    }

    // Initialize Enhanced LoRA adapter
    if (this.config.enhancedLoraEnabled) {
      this.enhancedLoraAdapter = new EnhancedLoRAAdapter(
        {
          rank: 16,
          alpha: 32,
          dropout: 0.1,
          targetModules: ['base_hidden_1', 'base_hidden_2', 'base_output'],
          taskType: 'cognitive_processing',
        },
        {
          inputDim: 512,
          hiddenDim: 256,
          outputDim: 512,
          learningRate: 0.001,
          batchSize: 16,
          epochs: 5,
        },
        {
          enableHRMGuidance: this.config.hrmEnabled,
          hrmWeightInfluence: 0.3,
          adaptationThreshold: 0.7,
        }
      );
    }

    // Initialize HRM Bridge
    if (this.config.hrmEnabled) {
      const hrmConfig: HRMConfig = this.config.hrmConfig || {
        l_module_count: 8,
        h_module_count: 4,
        enable_adaptation: true,
        processing_timeout: 5000,
      };

      this.hrmBridge = new HRMBridge(hrmConfig);
    }

    // Initialize HRM-LoRA Bridge if both HRM and Enhanced LoRA are enabled
    if (this.config.hrmEnabled && this.config.enhancedLoraEnabled) {
      this.hrmLoraBridge = new HRMLoRABridge({
        syncFrequency: 3000, // 3 seconds
        adaptationThreshold: 0.1,
        maxWeightChange: 0.3,
        enableBidirectional: true,
      });
    }

    // Initialize Adaptive Learning Pipeline
    if (this.config.adaptiveLearningEnabled) {
      this.adaptiveLearningPipeline = new AdaptiveLearningPipeline({
        minInteractionsForPattern: 3,
        adaptationThreshold: 0.6,
        maxPatternsStored: 1000,
        learningRateDecay: 0.95,
        feedbackWeight: 0.7,
        hrmInfluenceWeight: 0.3,
        realTimeAdaptation: true,
      });
    }

    // Initialize KNIRV Wallet Integration
    if (this.config.walletIntegrationEnabled) {
      this.walletIntegration = new KNIRVWalletIntegration({
        apiBaseUrl: 'http://localhost:8083/api/v1',
        chainId: 'knirv-mainnet-1',
        rpcUrl: 'https://rpc.knirv.com',
        enableCrossPlatform: true,
        autoConnectMobile: false,
        qrCodeTimeout: 300000,
      });
    }

    // Initialize KNIRV Chain Integration
    if (this.config.chainIntegrationEnabled) {
      this.chainIntegration = new KNIRVChainIntegration({
        rpcUrl: 'http://localhost:8080',
        chainId: 'knirv-chain-1',
        networkName: 'KNIRV Network',
        contractAddresses: {
          nrnToken: '0x1234567890123456789012345678901234567890',
          llmRegistry: '0x2345678901234567890123456789012345678901',
          skillRegistry: '0x3456789012345678901234567890123456789012',
        },
        gasPrice: '20000000000',
        gasLimit: '500000',
      });
    }

    // Initialize Ecosystem Communication Layer
    if (this.config.ecosystemCommunicationEnabled) {
      this.ecosystemCommunication = new EcosystemCommunicationLayer({
        enableWalletIntegration: this.config.walletIntegrationEnabled,
        enableChainIntegration: this.config.chainIntegrationEnabled,
        enableNexusIntegration: true,
        enableGatewayIntegration: true,
        enableShellIntegration: true,
        communicationProtocol: 'http',
        heartbeatInterval: 30000,
        timeoutDuration: 10000,
        retryAttempts: 3,
      });
    }

    this.setupEventHandlers();
  }

  private setupEventHandlers(): void {
    // Voice input events
    if (this.voiceProcessor) {
      this.voiceProcessor.on('speechDetected', (speech) => {
        this.processVoiceInput(speech);
      });

      this.voiceProcessor.on('commandRecognized', (command) => {
        this.executeVoiceCommand(command);
      });
    }

    // Visual input events
    if (this.visualProcessor) {
      this.visualProcessor.on('objectDetected', (objects) => {
        this.processVisualInput(objects);
      });

      this.visualProcessor.on('gestureRecognized', (gesture) => {
        this.executeGestureCommand(gesture);
      });

      // Enhanced AI visual processing events
      this.visualProcessor.on('visualProcessorInitialized', () => {
        this.emit('visualProcessorInitialized');
      });

      this.visualProcessor.on('imageProcessedWithAI', (result) => {
        this.emit('visualImageProcessedWithAI', result);
        this.processEnhancedVisualInput(result);
      });

      this.visualProcessor.on('visualProcessorDisposed', () => {
        this.emit('visualProcessorDisposed');
      });
    }

    // SEAL Framework events
    this.sealFramework.on('agentCreated', (agent) => {
      this.emit('cognitiveEvent', {
        type: 'agent_created',
        data: agent,
      });
    });

    this.sealFramework.on('adaptationComplete', (adaptation) => {
      this.applyAdaptation(adaptation);
    });

    // LoRA events
    if (this.loraAdapter) {
      this.loraAdapter.on('adaptationReady', (weights) => {
        this.applyLoRAAdaptation(weights);
      });

      this.loraAdapter.on('trainingStepComplete', (metrics) => {
        this.emit('loraTrainingUpdate', metrics);
      });

      this.loraAdapter.on('batchTrainingComplete', (result) => {
        this.emit('loraBatchComplete', result);
      });
    }

    // Enhanced LoRA events
    if (this.enhancedLoraAdapter) {
      this.enhancedLoraAdapter.on('enhancedLoraStarted', () => {
        this.emit('enhancedLoraStarted');
      });

      this.enhancedLoraAdapter.on('trainingStepComplete', (metrics) => {
        this.emit('enhancedLoraTrainingUpdate', metrics);
      });

      this.enhancedLoraAdapter.on('batchTrainingComplete', (result) => {
        this.emit('enhancedLoraBatchComplete', result);
      });

      this.enhancedLoraAdapter.on('epochComplete', (data) => {
        this.emit('enhancedLoraEpochComplete', data);
      });
    }

    // HRM-LoRA Bridge events
    if (this.hrmLoraBridge) {
      this.hrmLoraBridge.on('bridgeStarted', () => {
        this.emit('hrmLoraBridgeStarted');
      });

      this.hrmLoraBridge.on('weightsSynced', (data) => {
        this.emit('hrmLoraWeightsSynced', data);
      });

      this.hrmLoraBridge.on('mappingSynced', (data) => {
        this.emit('hrmLoraMappingSynced', data);
      });

      this.hrmLoraBridge.on('syncError', (error) => {
        this.emit('hrmLoraSyncError', error);
      });
    }

    // Adaptive Learning Pipeline events
    if (this.adaptiveLearningPipeline) {
      this.adaptiveLearningPipeline.on('pipelineStarted', () => {
        this.emit('adaptiveLearningStarted');
      });

      this.adaptiveLearningPipeline.on('interactionRecorded', (interaction) => {
        this.emit('learningInteractionRecorded', interaction);
      });

      this.adaptiveLearningPipeline.on('patternCreated', (pattern) => {
        this.emit('learningPatternCreated', pattern);
      });

      this.adaptiveLearningPipeline.on('adaptationTriggered', (data) => {
        this.emit('learningAdaptationTriggered', data);
      });

      this.adaptiveLearningPipeline.on('metricsUpdated', (metrics) => {
        this.emit('learningMetricsUpdated', metrics);
      });
    }

    // KNIRV Wallet Integration events
    if (this.walletIntegration) {
      this.walletIntegration.on('walletInitialized', () => {
        this.emit('walletInitialized');
      });

      this.walletIntegration.on('accountSwitched', (account) => {
        this.emit('walletAccountSwitched', account);
      });

      this.walletIntegration.on('transactionCreated', (transaction) => {
        this.emit('walletTransactionCreated', transaction);
      });

      this.walletIntegration.on('transactionConfirmed', (transaction) => {
        this.emit('walletTransactionConfirmed', transaction);
      });

      this.walletIntegration.on('skillInvoked', (data) => {
        this.emit('walletSkillInvoked', data);
      });

      this.walletIntegration.on('qrCodeGenerated', (data) => {
        this.emit('walletQRCodeGenerated', data);
      });
    }

    // KNIRV Chain Integration events
    if (this.chainIntegration) {
      this.chainIntegration.on('chainInitialized', () => {
        this.emit('chainInitialized');
      });

      this.chainIntegration.on('skillsLoaded', (skills) => {
        this.emit('chainSkillsLoaded', skills);
      });

      this.chainIntegration.on('llmModelsLoaded', (models) => {
        this.emit('chainLLMModelsLoaded', models);
      });

      this.chainIntegration.on('contractCallExecuted', (data) => {
        this.emit('chainContractCallExecuted', data);
      });

      this.chainIntegration.on('skillInvokedOnChain', (invocation) => {
        this.emit('chainSkillInvoked', invocation);
      });

      this.chainIntegration.on('skillRegistered', (skill) => {
        this.emit('chainSkillRegistered', skill);
      });

      this.chainIntegration.on('llmModelRegistered', (model) => {
        this.emit('chainLLMModelRegistered', model);
      });

      this.chainIntegration.on('newBlocks', (data) => {
        this.emit('chainNewBlocks', data);
      });

      this.chainIntegration.on('skillValidationUpdated', (data) => {
        this.emit('chainSkillValidationUpdated', data);
      });

      this.chainIntegration.on('nrnTransferred', (data) => {
        this.emit('chainNRNTransferred', data);
      });
    }

    // Ecosystem Communication Layer events
    if (this.ecosystemCommunication) {
      this.ecosystemCommunication.on('ecosystemInitialized', () => {
        this.emit('ecosystemInitialized');
      });

      this.ecosystemCommunication.on('componentRegistered', (component) => {
        this.emit('ecosystemComponentRegistered', component);
      });

      this.ecosystemCommunication.on('connectionEstablished', (endpoint) => {
        this.emit('ecosystemConnectionEstablished', endpoint);
      });

      this.ecosystemCommunication.on('connectionFailed', (data) => {
        this.emit('ecosystemConnectionFailed', data);
      });

      this.ecosystemCommunication.on('componentOffline', (component) => {
        this.emit('ecosystemComponentOffline', component);
      });

      this.ecosystemCommunication.on('messageSent', (message) => {
        this.emit('ecosystemMessageSent', message);
      });

      this.ecosystemCommunication.on('messageProcessed', (message) => {
        this.emit('ecosystemMessageProcessed', message);
      });

      this.ecosystemCommunication.on('heartbeatComplete', (data) => {
        this.emit('ecosystemHeartbeatComplete', data);
      });
    }

    // HRM Bridge events
    if (this.hrmBridge) {
      this.hrmBridge.on('initialized', () => {
        this.emit('hrmInitialized');
      });

      this.hrmBridge.on('inputProcessed', (data) => {
        this.emit('hrmProcessed', data);
      });

      this.hrmBridge.on('error', (error) => {
        this.emit('hrmError', error);
      });

      this.hrmBridge.on('weightsLoaded', () => {
        this.emit('hrmWeightsLoaded');
      });
    }
  }

  public async start(): Promise<void> {
    if (this.isRunning) {
      throw new Error('Cognitive Engine is already running');
    }

    console.log('Starting Cognitive Engine...');

    // Start all components
    await this.sealFramework.start();
    await this.fabricAlgorithm.start();

    if (this.voiceProcessor) {
      await this.voiceProcessor.start();
    }

    if (this.visualProcessor) {
      await this.visualProcessor.start();
    }

    if (this.loraAdapter) {
      await this.loraAdapter.start();
    }

    if (this.enhancedLoraAdapter) {
      await this.enhancedLoraAdapter.start();
    }

    if (this.hrmBridge) {
      await this.hrmBridge.initialize();
      // Inject HRM bridge into SEAL framework for enhanced reasoning
      this.sealFramework.setHRMBridge(this.hrmBridge);
      // Inject HRM bridge into Fabric Algorithm for enhanced NRV generation
      this.fabricAlgorithm.setHRMBridge(this.hrmBridge);
      // Inject HRM bridge into Enhanced LoRA adapter
      if (this.enhancedLoraAdapter) {
        this.enhancedLoraAdapter.setHRMBridge(this.hrmBridge);
      }

      // Set up HRM-LoRA Bridge connections
      if (this.hrmLoraBridge) {
        this.hrmLoraBridge.setHRMBridge(this.hrmBridge);
        this.hrmLoraBridge.setEnhancedLoRAAdapter(this.enhancedLoraAdapter);
        await this.hrmLoraBridge.start();
      }

      // Set up Adaptive Learning Pipeline connections
      if (this.adaptiveLearningPipeline) {
        this.adaptiveLearningPipeline.setHRMBridge(this.hrmBridge);
        if (this.enhancedLoraAdapter) {
          this.adaptiveLearningPipeline.setEnhancedLoRAAdapter(this.enhancedLoraAdapter);
        }
        if (this.hrmLoraBridge) {
          this.adaptiveLearningPipeline.setHRMLoRABridge(this.hrmLoraBridge);
        }
        await this.adaptiveLearningPipeline.loadLearnedPatterns();
        await this.adaptiveLearningPipeline.start();
      }

      // Initialize KNIRV Wallet Integration
      if (this.walletIntegration) {
        await this.walletIntegration.initialize();
      }

      // Initialize KNIRV Chain Integration
      if (this.chainIntegration) {
        await this.chainIntegration.initialize();
      }

      // Initialize enhanced Visual Processor with AI capabilities
      if (this.visualProcessor) {
        this.visualProcessor.setHRMBridge(this.hrmBridge);
        await this.visualProcessor.initialize();
      }

      // Initialize Ecosystem Communication Layer
      if (this.ecosystemCommunication) {
        await this.ecosystemCommunication.initialize();
      }
    }

    this.isRunning = true;
    this.emit('engineStarted');
    console.log('Cognitive Engine started successfully');
  }

  public async stop(): Promise<void> {
    if (!this.isRunning) {
      return;
    }

    console.log('Stopping Cognitive Engine...');

    // Stop all components
    await this.sealFramework.stop();
    await this.fabricAlgorithm.stop();

    if (this.voiceProcessor) {
      await this.voiceProcessor.stop();
    }

    if (this.visualProcessor) {
      await this.visualProcessor.stop();
    }

    if (this.loraAdapter) {
      await this.loraAdapter.stop();
    }

    if (this.enhancedLoraAdapter) {
      await this.enhancedLoraAdapter.stop();
    }

    if (this.hrmLoraBridge) {
      await this.hrmLoraBridge.stop();
    }

    if (this.adaptiveLearningPipeline) {
      await this.adaptiveLearningPipeline.stop();
    }

    if (this.walletIntegration) {
      await this.walletIntegration.disconnect();
    }

    if (this.chainIntegration) {
      await this.chainIntegration.disconnect();
    }

    if (this.ecosystemCommunication) {
      await this.ecosystemCommunication.shutdown();
    }

    if (this.hrmBridge) {
      await this.hrmBridge.destroy();
    }

    this.isRunning = false;
    this.emit('engineStopped');
    console.log('Cognitive Engine stopped');
  }

  public async processInput(input: any, inputType: string): Promise<any> {
    const startTime = Date.now();

    try {
      // Update context
      this.updateContext(inputType, input);

      let response: any;

      // Use HRM for cognitive processing if available
      if (this.hrmBridge && this.hrmBridge.isReady()) {
        response = await this.processWithHRM(input, inputType);
      } else {
        // Fallback to original processing pipeline
        // Process through Fabric Algorithm
        const fabricResult = await this.fabricAlgorithm.process(input, {
          context: this.state.currentContext,
          inputType,
        });

        // Generate response using SEAL Framework
        response = await this.sealFramework.generateResponse(fabricResult, {
          confidenceLevel: this.state.confidenceLevel,
          activeSkills: this.state.activeSkills,
        });
      }

      // Record interaction for adaptive learning
      if (this.adaptiveLearningPipeline) {
        await this.recordInteractionForLearning(input, inputType, response);
      }

      const processingTime = Date.now() - startTime;
      this.emit('inputProcessed', {
        inputType,
        processingTime,
        response,
      });

      return response;

    } catch (error) {
      console.error('Error processing input:', error);
      this.emit('processingError', {
        inputType,
        error: error.message,
      });
      throw error;
    }
  }

  private async processWithHRM(input: any, inputType: string): Promise<any> {
    console.log('Processing with HRM:', inputType);

    try {
      let hrmOutput;

      // Route to appropriate HRM processing method based on input type
      switch (inputType) {
        case 'voice':
          // Convert voice input to numerical data for HRM
          const audioData = this.convertVoiceToData(input);
          hrmOutput = await this.hrmBridge.processVoiceInput(audioData, {
            context: Object.fromEntries(this.state.currentContext),
            confidenceLevel: this.state.confidenceLevel,
          });
          break;

        case 'visual':
          // Convert visual input to numerical data for HRM
          const visualData = this.convertVisualToData(input);
          hrmOutput = await this.hrmBridge.processVisualInput(visualData, {
            context: Object.fromEntries(this.state.currentContext),
            confidenceLevel: this.state.confidenceLevel,
          });
          break;

        case 'text':
        default:
          // Process text input through HRM
          const textInput = typeof input === 'string' ? input : JSON.stringify(input);
          hrmOutput = await this.hrmBridge.processTextInput(textInput, {
            context: Object.fromEntries(this.state.currentContext),
            confidenceLevel: this.state.confidenceLevel,
          });
          break;
      }

      // Convert HRM output to standard response format
      const response = {
        text: hrmOutput.reasoning_result,
        confidence: hrmOutput.confidence,
        processingTime: hrmOutput.processing_time,
        source: 'hrm',
        metadata: {
          l_module_activations: hrmOutput.l_module_activations,
          h_module_activations: hrmOutput.h_module_activations,
        },
        shouldSpeak: inputType === 'voice' && hrmOutput.confidence > 0.7,
      };

      // Update confidence level based on HRM output
      this.state.confidenceLevel = (this.state.confidenceLevel + hrmOutput.confidence) / 2;

      return response;

    } catch (error) {
      console.error('Error processing with HRM:', error);
      // Fallback to original processing
      throw error;
    }
  }

  private convertVoiceToData(speech: any): number[] {
    // Convert speech data to numerical array for HRM processing
    if (speech.audioData && Array.isArray(speech.audioData)) {
      return speech.audioData;
    }

    // Fallback: convert text to numerical representation
    if (speech.text) {
      return this.textToNumerical(speech.text);
    }

    return new Array(512).fill(0);
  }

  private convertVisualToData(objects: any[]): number[] {
    // Convert visual objects to numerical array for HRM processing
    const features: number[] = [];

    objects.forEach(obj => {
      if (obj.bbox) {
        features.push(...obj.bbox); // x, y, width, height
      }
      if (obj.confidence) {
        features.push(obj.confidence);
      }
      if (obj.classId) {
        features.push(obj.classId);
      }
    });

    // Pad or truncate to fixed size
    const maxLength = 512;
    if (features.length > maxLength) {
      return features.slice(0, maxLength);
    } else {
      return [...features, ...new Array(maxLength - features.length).fill(0)];
    }
  }

  private textToNumerical(text: string): number[] {
    // Simple text to numerical conversion
    const encoder = new TextEncoder();
    const bytes = encoder.encode(text);
    const normalized = Array.from(bytes).map(b => b / 255.0);

    const maxLength = 512;
    if (normalized.length > maxLength) {
      return normalized.slice(0, maxLength);
    } else {
      return [...normalized, ...new Array(maxLength - normalized.length).fill(0)];
    }
  }

  private async processVoiceInput(speech: any): Promise<void> {
    console.log('Processing voice input:', speech);

    const response = await this.processInput(speech, 'voice');

    // Convert response to speech if needed
    if (this.voiceProcessor && response.shouldSpeak) {
      await this.voiceProcessor.speak(response.text);
    }
  }

  private async processVisualInput(objects: any[]): Promise<void> {
    console.log('Processing visual input:', objects);

    const response = await this.processInput(objects, 'visual');

    // Update visual context
    this.state.currentContext.set('lastVisualObjects', objects);
  }

  private async processEnhancedVisualInput(result: any): Promise<void> {
    console.log('Processing enhanced visual input with AI:', result);

    try {
      // Update context with enhanced visual information
      this.state.currentContext.set('visualObjects', result.objects);
      this.state.currentContext.set('visualFaces', result.faces);
      this.state.currentContext.set('visualText', result.textRegions);
      this.state.currentContext.set('visualScene', result.sceneAnalysis);
      this.state.currentContext.set('visualGestures', result.gestures);

      // Record interaction for adaptive learning
      if (this.adaptiveLearningPipeline) {
        await this.recordInteractionForLearning(
          result,
          'visual_processing',
          { processed: true, hrmEnhanced: result.hrmEnhanced }
        );
      }

      // Trigger cognitive processing if significant visual input detected
      if (result.objects.length > 0 || result.faces.length > 0 || result.gestures.length > 0) {
        const cognitiveInput = {
          type: 'visual',
          data: result,
          confidence: this.calculateVisualConfidence(result),
          timestamp: new Date(),
        };

        // Process through SEAL framework
        const sealResponse = await this.sealFramework.processInput(cognitiveInput);

        // Generate NRV through Fabric Algorithm
        const nrv = await this.fabricAlgorithm.generateNRV(cognitiveInput, sealResponse);

        this.emit('enhancedVisualProcessed', {
          input: result,
          sealResponse,
          nrv,
          timestamp: new Date(),
        });
      }

    } catch (error) {
      console.error('Error processing enhanced visual input:', error);
    }
  }

  private calculateVisualConfidence(result: any): number {
    let confidence = 0;
    let count = 0;

    // Average confidence from objects
    if (result.objects.length > 0) {
      confidence += result.objects.reduce((sum: number, obj: any) => sum + obj.confidence, 0) / result.objects.length;
      count++;
    }

    // Average confidence from faces
    if (result.faces.length > 0) {
      confidence += result.faces.reduce((sum: number, face: any) => sum + face.confidence, 0) / result.faces.length;
      count++;
    }

    // Scene analysis confidence
    if (result.sceneAnalysis.confidence > 0) {
      confidence += result.sceneAnalysis.confidence;
      count++;
    }

    // Text recognition confidence
    if (result.textRegions.length > 0) {
      confidence += result.textRegions.reduce((sum: number, text: any) => sum + text.confidence, 0) / result.textRegions.length;
      count++;
    }

    return count > 0 ? confidence / count : 0;
  }

  private async executeVoiceCommand(command: any): Promise<void> {
    console.log('Executing voice command:', command);

    switch (command.type) {
      case 'invoke_skill':
        await this.invokeSkill(command.skillId, command.parameters);
        break;
      case 'start_learning':
        await this.startLearningMode();
        break;
      case 'save_adaptation':
        await this.saveCurrentAdaptation();
        break;
      default:
        console.warn('Unknown voice command:', command.type);
    }
  }

  private async executeGestureCommand(gesture: any): Promise<void> {
    console.log('Executing gesture command:', gesture);

    switch (gesture.type) {
      case 'point':
        await this.focusOnObject(gesture.target);
        break;
      case 'swipe':
        await this.navigateInterface(gesture.direction);
        break;
      case 'pinch':
        await this.adjustScale(gesture.scale);
        break;
      default:
        console.warn('Unknown gesture:', gesture.type);
    }
  }

  private updateContext(inputType: string, input: any): void {
    this.state.currentContext.set(`last_${inputType}`, input);
    this.state.currentContext.set('last_update', new Date());

    // Maintain context size limit
    if (this.state.currentContext.size > this.config.maxContextSize) {
      const oldestKey = this.state.currentContext.keys().next().value;
      this.state.currentContext.delete(oldestKey);
    }
  }

  private shouldTriggerAdaptation(): boolean {
    const recentEvents = this.state.learningHistory.slice(-10);
    if (recentEvents.length === 0) return false;

    const avgFeedback = recentEvents.reduce((sum, event) => sum + event.feedback, 0) / recentEvents.length;
    return avgFeedback < this.config.adaptationThreshold;
  }

  private async triggerAdaptation(): Promise<void> {
    console.log('Triggering cognitive adaptation...');

    const recentHistory = this.state.learningHistory.slice(-50);
    const adaptation = await this.sealFramework.generateAdaptation(recentHistory);

    if (adaptation && this.loraAdapter) {
      // Convert adaptation to TrainingData format
      const trainingData = {
        input: adaptation.input || recentHistory,
        output: adaptation.output || adaptation.expectedOutput,
        feedback: adaptation.feedback || 0.8, // Default positive feedback
        timestamp: new Date(),
      };

      await this.loraAdapter.addTrainingData(trainingData);
    }

    this.state.adaptationLevel += 0.1;
    this.emit('adaptationTriggered', { adaptationLevel: this.state.adaptationLevel });
  }

  private async applyAdaptation(adaptation: any): Promise<void> {
    console.log('Applying cognitive adaptation:', adaptation);

    // Update confidence level based on adaptation success
    this.state.confidenceLevel = Math.min(this.state.confidenceLevel + 0.05, 1.0);

    // Mark recent events as adapted
    this.state.learningHistory.slice(-10).forEach(event => {
      event.adaptationApplied = true;
    });

    this.emit('adaptationApplied', adaptation);
  }

  private async applyLoRAAdaptation(weights: any): Promise<void> {
    console.log('Applying LoRA adaptation weights');

    // This would integrate with the actual model weights
    // For now, we'll just update the adaptation level
    this.state.adaptationLevel = Math.min(this.state.adaptationLevel + 0.2, 1.0);

    this.emit('loraAdaptationApplied', {
      adaptationLevel: this.state.adaptationLevel,
      weights,
    });
  }

  public async invokeSkill(skillId: string, parameters: any): Promise<any> {
    console.log(`Invoking skill: ${skillId}`, parameters);

    // Add to active skills
    if (!this.state.activeSkills.includes(skillId)) {
      this.state.activeSkills.push(skillId);
    }

    try {
      // This would integrate with KNIRVCHAIN for skill invocation
      const result = await this.sealFramework.invokeSkill(skillId, parameters);

      this.emit('skillInvoked', {
        skillId,
        parameters,
        result,
      });

      return result;

    } catch (error) {
      console.error(`Error invoking skill ${skillId}:`, error);
      throw error;
    } finally {
      // Remove from active skills
      const index = this.state.activeSkills.indexOf(skillId);
      if (index > -1) {
        this.state.activeSkills.splice(index, 1);
      }
    }
  }

  public async startLearningMode(): Promise<void> {
    console.log('Starting learning mode...');

    await this.sealFramework.enableLearningMode();

    if (this.loraAdapter) {
      await this.loraAdapter.enableTraining();
    }

    this.emit('learningModeStarted');
  }

  public async saveCurrentAdaptation(): Promise<void> {
    console.log('Saving current adaptation...');

    if (this.loraAdapter) {
      const weights = await this.loraAdapter.exportWeights();

      // Save to local storage or send to KNIRVCHAIN
      localStorage.setItem('cognitive_adaptation', JSON.stringify({
        weights,
        adaptationLevel: this.state.adaptationLevel,
        timestamp: new Date(),
      }));
    }

    this.emit('adaptationSaved');
  }



  public getState(): CognitiveState {
    return { ...this.state };
  }

  public getMetrics(): any {
    return {
      isRunning: this.isRunning,
      confidenceLevel: this.state.confidenceLevel,
      adaptationLevel: this.state.adaptationLevel,
      activeSkills: this.state.activeSkills.length,
      learningEvents: this.state.learningHistory.length,
      contextSize: this.state.currentContext.size,
    };
  }

  private async focusOnObject(target: any): Promise<void> {
    console.log('Focusing on object:', target);
    this.state.currentContext.set('focusTarget', target);
  }

  private async navigateInterface(direction: string): Promise<void> {
    console.log('Navigating interface:', direction);
    this.emit('navigationRequest', { direction });
  }

  private async adjustScale(scale: number): Promise<void> {
    console.log('Adjusting scale:', scale);
    this.emit('scaleAdjustment', { scale });
  }

  // Month 9 getter methods for demo access
  public getVisualProcessor(): any {
    return this.visualProcessor;
  }

  public getLoRAAdapter(): any {
    return this.loraAdapter;
  }

  public getVoiceProcessor(): any {
    return this.voiceProcessor;
  }

  public getFabricAlgorithm(): any {
    return this.fabricAlgorithm;
  }

  public getHRMBridge(): any {
    return this.hrmBridge;
  }

  public async loadHRMWeights(weightsData: Uint8Array): Promise<boolean> {
    if (!this.hrmBridge) {
      console.warn('HRM bridge not initialized');
      return false;
    }

    try {
      const success = await this.hrmBridge.loadModelWeights(weightsData);
      if (success) {
        console.log('HRM model weights loaded successfully');
        this.emit('hrmWeightsLoaded');
      }
      return success;
    } catch (error) {
      console.error('Failed to load HRM weights:', error);
      return false;
    }
  }

  public getHRMModelInfo(): any {
    if (!this.hrmBridge) {
      return null;
    }
    return this.hrmBridge.getModelInfo();
  }

  public isHRMReady(): boolean {
    return this.hrmBridge ? this.hrmBridge.isReady() : false;
  }

  public getEnhancedLoRAAdapter(): any {
    return this.enhancedLoraAdapter;
  }

  public isEnhancedLoRAReady(): boolean {
    return this.enhancedLoraAdapter ? this.enhancedLoraAdapter.isAdapterReady() : false;
  }

  public async trainEnhancedLoRA(trainingData: any[]): Promise<void> {
    if (!this.enhancedLoraAdapter) {
      console.warn('Enhanced LoRA adapter not initialized');
      return;
    }

    try {
      this.enhancedLoraAdapter.enableTraining();
      await this.enhancedLoraAdapter.trainOnBatch(trainingData);
      console.log('Enhanced LoRA training completed');
    } catch (error) {
      console.error('Enhanced LoRA training failed:', error);
      throw error;
    }
  }

  public async adaptWithEnhancedLoRA(input: any, expectedOutput: any, feedback: number): Promise<any> {
    if (!this.enhancedLoraAdapter) {
      console.warn('Enhanced LoRA adapter not available');
      return input;
    }

    try {
      return await this.enhancedLoraAdapter.adapt(input, expectedOutput, feedback);
    } catch (error) {
      console.error('Enhanced LoRA adaptation failed:', error);
      return input;
    }
  }

  public getEnhancedLoRAMetrics(): any {
    if (!this.enhancedLoraAdapter) {
      return null;
    }
    return this.enhancedLoraAdapter.getEnhancedMetrics();
  }

  public async saveEnhancedLoRAModel(modelName: string): Promise<void> {
    if (!this.enhancedLoraAdapter) {
      throw new Error('Enhanced LoRA adapter not initialized');
    }

    try {
      await this.enhancedLoraAdapter.saveModel(modelName);
      console.log(`Enhanced LoRA model saved as ${modelName}`);
    } catch (error) {
      console.error('Failed to save Enhanced LoRA model:', error);
      throw error;
    }
  }

  public async loadEnhancedLoRAModel(modelName: string): Promise<void> {
    if (!this.enhancedLoraAdapter) {
      throw new Error('Enhanced LoRA adapter not initialized');
    }

    try {
      await this.enhancedLoraAdapter.loadModel(modelName);
      console.log(`Enhanced LoRA model loaded from ${modelName}`);
    } catch (error) {
      console.error('Failed to load Enhanced LoRA model:', error);
      throw error;
    }
  }

  public exportEnhancedLoRAWeights(): any {
    if (!this.enhancedLoraAdapter) {
      return null;
    }
    return this.enhancedLoraAdapter.exportWeights();
  }

  public async importEnhancedLoRAWeights(weights: any): Promise<void> {
    if (!this.enhancedLoraAdapter) {
      throw new Error('Enhanced LoRA adapter not initialized');
    }

    try {
      await this.enhancedLoraAdapter.importWeights(weights);
      console.log('Enhanced LoRA weights imported successfully');
    } catch (error) {
      console.error('Failed to import Enhanced LoRA weights:', error);
      throw error;
    }
  }

  public getTensorFlowInfo(): any {
    if (!this.enhancedLoraAdapter) {
      return null;
    }
    return this.enhancedLoraAdapter.getTensorFlowInfo();
  }

  public getHRMLoRABridge(): any {
    return this.hrmLoraBridge;
  }

  public isHRMLoRABridgeReady(): boolean {
    return this.hrmLoraBridge ? this.hrmLoraBridge.getStatus().isRunning : false;
  }

  public getHRMLoRAMappings(): any {
    if (!this.hrmLoraBridge) {
      return null;
    }
    return Object.fromEntries(this.hrmLoraBridge.getMappings());
  }

  public async forceHRMLoRASync(): Promise<void> {
    if (!this.hrmLoraBridge) {
      throw new Error('HRM-LoRA Bridge not initialized');
    }

    try {
      await this.hrmLoraBridge.forceSyncNow();
      console.log('HRM-LoRA synchronization forced successfully');
    } catch (error) {
      console.error('Failed to force HRM-LoRA synchronization:', error);
      throw error;
    }
  }

  public updateHRMLoRASyncConfig(config: any): void {
    if (!this.hrmLoraBridge) {
      console.warn('HRM-LoRA Bridge not initialized');
      return;
    }

    this.hrmLoraBridge.updateSyncConfig(config);
    console.log('HRM-LoRA sync configuration updated');
  }

  public getHRMLoRAStatus(): any {
    if (!this.hrmLoraBridge) {
      return {
        available: false,
        reason: 'Bridge not initialized',
      };
    }

    return {
      available: true,
      ...this.hrmLoraBridge.getStatus(),
    };
  }

  public getComprehensiveStatus(): any {
    return {
      engine: {
        isRunning: this.isRunning,
        confidenceLevel: this.state.confidenceLevel,
        activeSkills: this.state.activeSkills.length,
      },
      hrm: {
        enabled: this.config.hrmEnabled,
        ready: this.isHRMReady(),
        modelInfo: this.getHRMModelInfo(),
      },
      lora: {
        basicEnabled: this.config.loraEnabled,
        enhancedEnabled: this.config.enhancedLoraEnabled,
        enhancedReady: this.isEnhancedLoRAReady(),
        metrics: this.getEnhancedLoRAMetrics(),
      },
      hrmLoraBridge: this.getHRMLoRAStatus(),
      wallet: this.getWalletStatus(),
      chain: this.getChainStatus(),
      ecosystem: this.getEcosystemStatus(),
      adaptiveLearning: this.getAdaptiveLearningStatus(),
      tensorflow: this.getTensorFlowInfo(),
      seal: this.sealFramework.getMetrics(),
      fabric: this.fabricAlgorithm.getEnhancedMetrics(),
    };
  }

  private async recordInteractionForLearning(input: any, inputType: string, response: any): Promise<void> {
    try {
      await this.adaptiveLearningPipeline.recordInteraction({
        inputType: inputType as any,
        input,
        output: response,
        context: Object.fromEntries(this.state.currentContext),
      });
    } catch (error) {
      console.error('Error recording interaction for learning:', error);
    }
  }

  public async provideFeedback(interactionId: string, feedback: number): Promise<void> {
    if (!this.adaptiveLearningPipeline) {
      console.warn('Adaptive learning pipeline not available');
      return;
    }

    // This would ideally update the specific interaction
    // For now, we'll record it as a new feedback interaction
    try {
      await this.adaptiveLearningPipeline.recordInteraction({
        inputType: 'feedback',
        input: { interactionId, feedback },
        output: { acknowledged: true },
        userFeedback: feedback,
        context: { type: 'user_feedback' },
      });

      console.log(`Feedback recorded: ${feedback} for interaction ${interactionId}`);
    } catch (error) {
      console.error('Error providing feedback:', error);
    }
  }

  public getAdaptiveLearningPipeline(): any {
    return this.adaptiveLearningPipeline;
  }

  public isAdaptiveLearningReady(): boolean {
    return this.adaptiveLearningPipeline ? this.adaptiveLearningPipeline.getStatus().isRunning : false;
  }

  public getAdaptiveLearningMetrics(): any {
    if (!this.adaptiveLearningPipeline) {
      return null;
    }
    return this.adaptiveLearningPipeline.getMetrics();
  }

  public getAdaptiveLearningPatterns(): any[] {
    if (!this.adaptiveLearningPipeline) {
      return [];
    }
    return this.adaptiveLearningPipeline.getPatterns();
  }

  public updateAdaptiveLearningConfig(config: any): void {
    if (!this.adaptiveLearningPipeline) {
      console.warn('Adaptive learning pipeline not available');
      return;
    }

    this.adaptiveLearningPipeline.updateConfig(config);
    console.log('Adaptive learning configuration updated');
  }

  public clearAdaptiveLearningPatterns(): void {
    if (!this.adaptiveLearningPipeline) {
      console.warn('Adaptive learning pipeline not available');
      return;
    }

    this.adaptiveLearningPipeline.clearPatterns();
    console.log('Adaptive learning patterns cleared');
  }

  public getAdaptiveLearningStatus(): any {
    if (!this.adaptiveLearningPipeline) {
      return {
        available: false,
        reason: 'Pipeline not initialized',
      };
    }

    return {
      available: true,
      ...this.adaptiveLearningPipeline.getStatus(),
    };
  }

  // KNIRV Wallet Integration Methods

  public getWalletIntegration(): any {
    return this.walletIntegration;
  }

  public isWalletConnected(): boolean {
    return this.walletIntegration ? this.walletIntegration.isWalletConnected() : false;
  }

  public getWalletAccounts(): any[] {
    if (!this.walletIntegration) {
      return [];
    }
    return this.walletIntegration.getAccounts();
  }

  public getCurrentWalletAccount(): any {
    if (!this.walletIntegration) {
      return null;
    }
    return this.walletIntegration.getCurrentAccount();
  }

  public async switchWalletAccount(accountId: string): Promise<void> {
    if (!this.walletIntegration) {
      throw new Error('Wallet integration not available');
    }

    try {
      await this.walletIntegration.switchAccount(accountId);
      console.log(`Switched to wallet account: ${accountId}`);
    } catch (error) {
      console.error('Failed to switch wallet account:', error);
      throw error;
    }
  }

  public async getWalletBalance(accountId?: string): Promise<any> {
    if (!this.walletIntegration) {
      throw new Error('Wallet integration not available');
    }

    try {
      return await this.walletIntegration.getBalance(accountId);
    } catch (error) {
      console.error('Failed to get wallet balance:', error);
      throw error;
    }
  }

  public async getNRNBalance(accountId?: string): Promise<any> {
    if (!this.walletIntegration) {
      throw new Error('Wallet integration not available');
    }

    try {
      return await this.walletIntegration.getNRNBalance(accountId);
    } catch (error) {
      console.error('Failed to get NRN balance:', error);
      throw error;
    }
  }

  public async createWalletTransaction(request: any): Promise<string> {
    if (!this.walletIntegration) {
      throw new Error('Wallet integration not available');
    }

    try {
      const transactionId = await this.walletIntegration.createTransaction(request);
      console.log(`Created wallet transaction: ${transactionId}`);
      return transactionId;
    } catch (error) {
      console.error('Failed to create wallet transaction:', error);
      throw error;
    }
  }

  public async invokeSkillWithWallet(skillInvocation: any): Promise<string> {
    if (!this.walletIntegration) {
      throw new Error('Wallet integration not available');
    }

    try {
      const transactionId = await this.walletIntegration.invokeSkill(skillInvocation);
      console.log(`Invoked skill with wallet: ${skillInvocation.skillName}`);

      // Record this as a learning interaction
      if (this.adaptiveLearningPipeline) {
        await this.recordInteractionForLearning(
          skillInvocation,
          'skill_invocation',
          { transactionId, status: 'initiated' }
        );
      }

      return transactionId;
    } catch (error) {
      console.error('Failed to invoke skill with wallet:', error);
      throw error;
    }
  }

  public async getWalletTransaction(transactionId: string): Promise<any> {
    if (!this.walletIntegration) {
      throw new Error('Wallet integration not available');
    }

    try {
      return await this.walletIntegration.getTransaction(transactionId);
    } catch (error) {
      console.error('Failed to get wallet transaction:', error);
      throw error;
    }
  }

  public getWalletTransactions(): any[] {
    if (!this.walletIntegration) {
      return [];
    }
    return this.walletIntegration.getTransactions();
  }

  public async checkWalletTransactionStatus(transactionId: string): Promise<any> {
    if (!this.walletIntegration) {
      throw new Error('Wallet integration not available');
    }

    try {
      return await this.walletIntegration.checkTransactionStatus(transactionId);
    } catch (error) {
      console.error('Failed to check wallet transaction status:', error);
      throw error;
    }
  }

  public getWalletStatus(): any {
    if (!this.walletIntegration) {
      return {
        available: false,
        reason: 'Wallet integration not initialized',
      };
    }

    return {
      available: true,
      ...this.walletIntegration.getStatus(),
    };
  }

  public updateWalletConfig(config: any): void {
    if (!this.walletIntegration) {
      console.warn('Wallet integration not available');
      return;
    }

    this.walletIntegration.updateConfig(config);
    console.log('Wallet configuration updated');
  }

  // KNIRV Chain Integration Methods

  public getChainIntegration(): any {
    return this.chainIntegration;
  }

  public isChainConnected(): boolean {
    return this.chainIntegration ? this.chainIntegration.isChainConnected() : false;
  }

  public async executeChainContractCall(call: any): Promise<any> {
    if (!this.chainIntegration) {
      throw new Error('Chain integration not available');
    }

    try {
      const result = await this.chainIntegration.executeContractCall(call);
      console.log(`Executed contract call: ${call.contract}.${call.method}`);
      return result;
    } catch (error) {
      console.error('Failed to execute contract call:', error);
      throw error;
    }
  }

  public async verifySkillOnChain(skillId: string): Promise<boolean> {
    if (!this.chainIntegration) {
      throw new Error('Chain integration not available');
    }

    try {
      return await this.chainIntegration.verifySkill(skillId);
    } catch (error) {
      console.error('Failed to verify skill on chain:', error);
      return false;
    }
  }

  public async invokeSkillOnChain(
    skillId: string,
    userAddress: string,
    nrnAmount: string,
    parameters: any
  ): Promise<string> {
    if (!this.chainIntegration) {
      throw new Error('Chain integration not available');
    }

    try {
      const transactionHash = await this.chainIntegration.invokeSkillOnChain(
        skillId,
        userAddress,
        nrnAmount,
        parameters
      );

      // Record this as a learning interaction
      if (this.adaptiveLearningPipeline) {
        await this.recordInteractionForLearning(
          { skillId, parameters },
          'chain_skill_invocation',
          { transactionHash, status: 'initiated' }
        );
      }

      console.log(`Invoked skill ${skillId} on chain: ${transactionHash}`);
      return transactionHash;
    } catch (error) {
      console.error('Failed to invoke skill on chain:', error);
      throw error;
    }
  }

  public async registerSkillOnChain(skillMetadata: any): Promise<string> {
    if (!this.chainIntegration) {
      throw new Error('Chain integration not available');
    }

    try {
      const skillId = await this.chainIntegration.registerSkill(skillMetadata);
      console.log(`Registered skill on chain: ${skillId}`);
      return skillId;
    } catch (error) {
      console.error('Failed to register skill on chain:', error);
      throw error;
    }
  }

  public async registerLLMModelOnChain(llmMetadata: any): Promise<string> {
    if (!this.chainIntegration) {
      throw new Error('Chain integration not available');
    }

    try {
      const modelId = await this.chainIntegration.registerLLMModel(llmMetadata);
      console.log(`Registered LLM model on chain: ${modelId}`);
      return modelId;
    } catch (error) {
      console.error('Failed to register LLM model on chain:', error);
      throw error;
    }
  }

  public async getChainNRNBalance(address: string): Promise<string> {
    if (!this.chainIntegration) {
      throw new Error('Chain integration not available');
    }

    try {
      return await this.chainIntegration.getNRNBalance(address);
    } catch (error) {
      console.error('Failed to get NRN balance from chain:', error);
      return '0';
    }
  }

  public async transferNRNOnChain(from: string, to: string, amount: string): Promise<string> {
    if (!this.chainIntegration) {
      throw new Error('Chain integration not available');
    }

    try {
      const transactionHash = await this.chainIntegration.transferNRN(from, to, amount);
      console.log(`Transferred ${amount} NRN on chain: ${transactionHash}`);
      return transactionHash;
    } catch (error) {
      console.error('Failed to transfer NRN on chain:', error);
      throw error;
    }
  }

  public async getNetworkConsensus(): Promise<any> {
    if (!this.chainIntegration) {
      throw new Error('Chain integration not available');
    }

    try {
      return await this.chainIntegration.getNetworkConsensus();
    } catch (error) {
      console.error('Failed to get network consensus:', error);
      throw error;
    }
  }

  public getChainSkills(): any[] {
    if (!this.chainIntegration) {
      return [];
    }
    return this.chainIntegration.getSkills();
  }

  public getChainSkill(skillId: string): any {
    if (!this.chainIntegration) {
      return null;
    }
    return this.chainIntegration.getSkill(skillId);
  }

  public getChainLLMModels(): any[] {
    if (!this.chainIntegration) {
      return [];
    }
    return this.chainIntegration.getLLMModels();
  }

  public getChainLLMModel(modelId: string): any {
    if (!this.chainIntegration) {
      return null;
    }
    return this.chainIntegration.getLLMModel(modelId);
  }

  public getChainSkillInvocations(skillId: string): any[] {
    if (!this.chainIntegration) {
      return [];
    }
    return this.chainIntegration.getSkillInvocations(skillId);
  }

  public getChainStatus(): any {
    if (!this.chainIntegration) {
      return {
        available: false,
        reason: 'Chain integration not initialized',
      };
    }

    return {
      available: true,
      ...this.chainIntegration.getStatus(),
    };
  }

  public updateChainConfig(config: any): void {
    if (!this.chainIntegration) {
      console.warn('Chain integration not available');
      return;
    }

    this.chainIntegration.updateConfig(config);
    console.log('Chain configuration updated');
  }

  // Unified skill invocation that uses both wallet and chain
  public async invokeSkillUnified(
    skillId: string,
    parameters: any,
    nrnAmount?: string
  ): Promise<{ walletTransactionId?: string; chainTransactionHash?: string }> {
    const result: { walletTransactionId?: string; chainTransactionHash?: string } = {};

    try {
      // First verify skill exists on chain
      if (this.chainIntegration) {
        const isVerified = await this.verifySkillOnChain(skillId);
        if (!isVerified) {
          throw new Error('Skill not verified on chain');
        }
      }

      // Get current wallet account
      const currentAccount = this.getCurrentWalletAccount();
      if (!currentAccount) {
        throw new Error('No active wallet account');
      }

      // Determine NRN amount if not provided
      let finalNrnAmount = nrnAmount;
      if (!finalNrnAmount && this.chainIntegration) {
        const skill = this.getChainSkill(skillId);
        if (skill) {
          finalNrnAmount = skill.usageFee;
        }
      }

      // Execute wallet transaction for skill invocation
      if (this.walletIntegration && finalNrnAmount) {
        const walletTransactionId = await this.invokeSkillWithWallet({
          skillId,
          skillName: skillId,
          nrnCost: finalNrnAmount,
          parameters,
          expectedOutput: {},
          timeout: 30000,
        });
        result.walletTransactionId = walletTransactionId;
      }

      // Execute chain transaction for skill invocation
      if (this.chainIntegration && finalNrnAmount) {
        const chainTransactionHash = await this.invokeSkillOnChain(
          skillId,
          currentAccount.address,
          finalNrnAmount,
          parameters
        );
        result.chainTransactionHash = chainTransactionHash;
      }

      this.emit('skillInvokedUnified', {
        skillId,
        parameters,
        nrnAmount: finalNrnAmount,
        result,
        timestamp: Date.now(),
      });

      return result;

    } catch (error) {
      console.error('Failed to invoke skill unified:', error);
      throw error;
    }
  }

  // Ecosystem Communication Methods

  public getEcosystemCommunication(): any {
    return this.ecosystemCommunication;
  }

  public isEcosystemConnected(): boolean {
    return this.ecosystemCommunication ? this.ecosystemCommunication.getEcosystemStatus().isRunning : false;
  }

  public async sendEcosystemMessage(messageData: any): Promise<any> {
    if (!this.ecosystemCommunication) {
      throw new Error('Ecosystem communication not available');
    }

    try {
      const response = await this.ecosystemCommunication.sendMessage({
        from: 'knirv-cortex',
        ...messageData,
      });

      console.log('Ecosystem message sent:', messageData.type);
      return response;
    } catch (error) {
      console.error('Failed to send ecosystem message:', error);
      throw error;
    }
  }

  public async executeSkillThroughEcosystem(skillId: string, parameters: any): Promise<any> {
    if (!this.ecosystemCommunication) {
      throw new Error('Ecosystem communication not available');
    }

    try {
      const response = await this.sendEcosystemMessage({
        to: 'knirv-nexus',
        type: 'command',
        payload: {
          action: 'execute_skill',
          skillId,
          parameters,
        },
        priority: 'high',
        requiresResponse: true,
      });

      // Record this as a learning interaction
      if (this.adaptiveLearningPipeline) {
        await this.recordInteractionForLearning(
          { skillId, parameters },
          'ecosystem_skill_execution',
          response
        );
      }

      console.log(`Executed skill ${skillId} through ecosystem`);
      return response;
    } catch (error) {
      console.error('Failed to execute skill through ecosystem:', error);
      throw error;
    }
  }

  public async performWalletOperationThroughEcosystem(operation: any): Promise<any> {
    if (!this.ecosystemCommunication) {
      throw new Error('Ecosystem communication not available');
    }

    try {
      const response = await this.sendEcosystemMessage({
        to: 'knirv-wallet',
        type: 'command',
        payload: operation,
        priority: 'normal',
        requiresResponse: true,
      });

      console.log('Wallet operation executed through ecosystem:', operation.type);
      return response;
    } catch (error) {
      console.error('Failed to execute wallet operation through ecosystem:', error);
      throw error;
    }
  }

  public async performBlockchainOperationThroughEcosystem(operation: any): Promise<any> {
    if (!this.ecosystemCommunication) {
      throw new Error('Ecosystem communication not available');
    }

    try {
      const response = await this.sendEcosystemMessage({
        to: 'knirv-chain',
        type: 'command',
        payload: operation,
        priority: 'normal',
        requiresResponse: true,
      });

      console.log('Blockchain operation executed through ecosystem:', operation.type);
      return response;
    } catch (error) {
      console.error('Failed to execute blockchain operation through ecosystem:', error);
      throw error;
    }
  }

  public getEcosystemComponents(): any[] {
    if (!this.ecosystemCommunication) {
      return [];
    }
    return this.ecosystemCommunication.getComponents();
  }

  public getEcosystemEndpoints(): any[] {
    if (!this.ecosystemCommunication) {
      return [];
    }
    return this.ecosystemCommunication.getEndpoints();
  }

  public isEcosystemComponentOnline(componentId: string): boolean {
    if (!this.ecosystemCommunication) {
      return false;
    }
    return this.ecosystemCommunication.isComponentOnline(componentId);
  }

  public getEcosystemStatus(): any {
    if (!this.ecosystemCommunication) {
      return {
        available: false,
        reason: 'Ecosystem communication not initialized',
      };
    }

    return {
      available: true,
      ...this.ecosystemCommunication.getEcosystemStatus(),
    };
  }

  public updateEcosystemConfig(config: any): void {
    if (!this.ecosystemCommunication) {
      console.warn('Ecosystem communication not available');
      return;
    }

    this.ecosystemCommunication.updateConfig(config);
    console.log('Ecosystem communication configuration updated');
  }

  // Unified ecosystem operation that coordinates multiple services
  public async performUnifiedEcosystemOperation(operation: {
    type: 'skill_with_payment' | 'cross_chain_transfer' | 'multi_service_query';
    payload: any;
  }): Promise<any> {
    if (!this.ecosystemCommunication) {
      throw new Error('Ecosystem communication not available');
    }

    console.log('Performing unified ecosystem operation:', operation.type);

    try {
      switch (operation.type) {
        case 'skill_with_payment':
          return await this.performSkillWithPayment(operation.payload);

        case 'cross_chain_transfer':
          return await this.performCrossChainTransfer(operation.payload);

        case 'multi_service_query':
          return await this.performMultiServiceQuery(operation.payload);

        default:
          throw new Error(`Unknown operation type: ${operation.type}`);
      }

    } catch (error) {
      console.error('Unified ecosystem operation failed:', error);
      throw error;
    }
  }

  private async performSkillWithPayment(payload: any): Promise<any> {
    // 1. Check wallet balance
    const walletResponse = await this.performWalletOperationThroughEcosystem({
      type: 'get_balance',
      accountId: payload.accountId,
    });

    if (!walletResponse.success || parseFloat(walletResponse.data.nrnBalance) < parseFloat(payload.nrnCost)) {
      throw new Error('Insufficient NRN balance');
    }

    // 2. Execute skill
    const skillResponse = await this.executeSkillThroughEcosystem(payload.skillId, payload.parameters);

    if (!skillResponse.success) {
      throw new Error('Skill execution failed');
    }

    // 3. Process payment
    const paymentResponse = await this.performWalletOperationThroughEcosystem({
      type: 'create_transaction',
      from: payload.accountId,
      to: 'skill_contract',
      nrnAmount: payload.nrnCost,
      skillId: payload.skillId,
    });

    return {
      success: true,
      skillResult: skillResponse.data,
      paymentTransaction: paymentResponse.data,
      timestamp: Date.now(),
    };
  }

  private async performCrossChainTransfer(payload: any): Promise<any> {
    // 1. Initiate wallet transaction
    const walletResponse = await this.performWalletOperationThroughEcosystem({
      type: 'create_transaction',
      ...payload,
    });

    // 2. Record on blockchain
    const chainResponse = await this.performBlockchainOperationThroughEcosystem({
      type: 'record_transaction',
      transactionData: walletResponse.data,
    });

    return {
      success: true,
      walletTransaction: walletResponse.data,
      blockchainRecord: chainResponse.data,
      timestamp: Date.now(),
    };
  }

  private async performMultiServiceQuery(payload: any): Promise<any> {
    const results: any = {};

    // Query multiple services in parallel
    const queries = payload.services.map(async (service: any) => {
      try {
        const response = await this.sendEcosystemMessage({
          to: service.componentId,
          type: 'query',
          payload: service.query,
          priority: 'normal',
          requiresResponse: true,
        });
        results[service.componentId] = response;
      } catch (error) {
        results[service.componentId] = { success: false, error: error.message };
      }
    });

    await Promise.all(queries);

    return {
      success: true,
      results,
      timestamp: Date.now(),
    };
  }
}
