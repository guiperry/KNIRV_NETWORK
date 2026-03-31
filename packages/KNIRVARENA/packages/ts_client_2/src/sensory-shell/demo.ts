// KNIRV Cognitive Shell Demo Script
// This script demonstrates the cognitive shell capabilities

import { CognitiveEngine, CognitiveConfig } from './CognitiveEngine';
import {
  CognitiveMetrics,
  VisualProcessor,
  LoRAAdapter,
  TrainingData
} from '../types/events';

export class CognitiveShellDemo {
  private engine: CognitiveEngine | null = null;

  async initializeDemo(): Promise<void> {
    console.log('🧠 Initializing KNIRV Cognitive Shell Demo...');

    const config: CognitiveConfig = {
      maxContextSize: 50,
      learningRate: 0.02,
      adaptationThreshold: 0.4,
      skillTimeout: 15000,
      voiceEnabled: true,
      visualEnabled: true,
      loraEnabled: true,
      enhancedLoraEnabled: true,
      hrmEnabled: false,
      wasmAgentsEnabled: false,
      typeScriptCompilerEnabled: false,
      ecosystemCommunicationEnabled: false,
      sealFrameworkEnabled: false,
      fabricAlgorithmEnabled: false,
      knirvChainIntegrationEnabled: false,
      knirvRouterIntegrationEnabled: false,
      adaptiveLearningEnabled: false,
      walletIntegrationEnabled: false,
      chainIntegrationEnabled: false,
      errorContextEnabled: false,
    } as CognitiveConfig;

    this.engine = new CognitiveEngine(config);

    // Set up demo event listeners
    this.setupDemoEventListeners();

    try {
      await this.engine.start();
      console.log('✅ Cognitive Shell started successfully');
    } catch (error) {
      console.error('❌ Failed to start Cognitive Shell:', error);
    }
  }

  private setupDemoEventListeners(): void {
    if (!this.engine) return;

    this.engine.on('engineStarted', () => {
      console.log('🚀 Engine Status: ONLINE');
      this.displayCapabilities();
    });

    this.engine.on('inputProcessed', ((data: {
      inputType: string;
      processingTime: number;
      response?: { type?: string };
    }) => {
      console.log('📝 Input Processed:', {
        type: data.inputType,
        processingTime: `${data.processingTime}ms`,
        response: data.response?.type || 'unknown'
      });
    }) as any);

    this.engine.on('skillInvoked', ((data: {
      skillId: string;
      parameters: Record<string, unknown>;
      result?: { result?: unknown };
    }) => {
      console.log('🎯 Skill Invoked:', {
        skillId: data.skillId,
        parameters: data.parameters,
        result: data.result?.result || 'completed'
      });
    }) as any);

    this.engine.on('adaptationTriggered', ((data: {
      metrics?: { adaptationLevel?: number };
    }) => {
      console.log('🔄 Adaptation Triggered:', {
        adaptationLevel: `${Math.round((data.metrics?.adaptationLevel || 0) * 100)}%`
      });
    }) as any);

    this.engine.on('learningModeStarted', () => {
      console.log('📚 Learning Mode: ACTIVE');
    });

    this.engine.on('cognitiveEvent', ((event: {
      type: string;
      data: unknown;
    }) => {
      console.log('🧠 Cognitive Event:', event.type, event.data);
    }) as any);
  }

  private displayCapabilities(): void {
    console.log('\n🎛️  KNIRV Cognitive Shell Capabilities:');
    console.log('   • Multi-modal input processing (voice, visual, text)');
    console.log('   • Adaptive learning and skill invocation');
    console.log('   • Real-time context management');
    console.log('   • LoRA-based model adaptation');
    console.log('   • SEAL framework agent management');
    console.log('   • Fabric algorithm processing');
    console.log('\n💬 Try these voice commands:');
    console.log('   • "invoke skill analysis"');
    console.log('   • "start learning"');
    console.log('   • "save adaptation"');
    console.log('   • "show network status"');
    console.log('   • "help with debugging"');
  }

  async runDemoSequence(): Promise<void> {
    if (!this.engine) {
      console.error('❌ Engine not initialized');
      return;
    }

    console.log('\n🎬 Starting Demo Sequence...\n');

    // Demo 1: Text Processing
    await this.demoTextProcessing();
    await this.delay(2000);

    // Demo 2: Voice Command Simulation
    await this.demoVoiceCommands();
    await this.delay(2000);

    // Demo 3: Skill Invocation
    await this.demoSkillInvocation();
    await this.delay(2000);

    // Demo 4: Learning Mode
    await this.demoLearningMode();
    await this.delay(2000);

    // Demo 5: Adaptation
    await this.demoAdaptation();

    console.log('\n🎉 Demo sequence completed!');
    this.displayMetrics();
  }

  private async demoTextProcessing(): Promise<void> {
    console.log('📝 Demo 1: Text Processing');
    
    const testInputs = [
      'Analyze the network performance metrics',
      'Generate a summary of system errors',
      'Identify optimization opportunities'
    ];

    for (const input of testInputs) {
      console.log(`   Input: "${input}"`);
      try {
        const result = await this.engine!.processInput(input, 'text') as {
          type?: string;
          confidence?: number;
        };
        console.log(`   Output: ${result.type || 'unknown'} (confidence: ${Math.round((result.confidence || 0) * 100)}%)`);
      } catch (error) {
        console.error(`   Error: ${error instanceof Error ? error.message : String(error)}`);
      }
      await this.delay(1000);
    }
  }

  private async demoVoiceCommands(): Promise<void> {
    console.log('🎤 Demo 2: Voice Command Processing');
    
    const voiceCommands = [
      'invoke skill network_analysis',
      'show system status',
      'help with performance tuning'
    ];

    for (const command of voiceCommands) {
      console.log(`   Voice: "${command}"`);
      try {
        const result = await this.engine!.processInput(command, 'voice') as {
          result?: { text?: string };
        };
        console.log(`   Response: ${result.result?.text || 'Command processed'}`);
      } catch (error) {
        console.error(`   Error: ${error instanceof Error ? error.message : String(error)}`);
      }
      await this.delay(1500);
    }
  }

  private async demoSkillInvocation(): Promise<void> {
    console.log('🎯 Demo 3: Skill Invocation');
    
    const skills = [
      { id: 'text_analysis', params: { text: 'Sample text for analysis' } },
      { id: 'code_generation', params: { language: 'typescript', task: 'create function' } },
      { id: 'problem_solving', params: { problem: 'optimize database queries' } }
    ];

    for (const skill of skills) {
      console.log(`   Invoking: ${skill.id}`);
      try {
        const result = await this.engine!.invokeSkill(skill.id, skill.params) as {
          result?: unknown;
        };
        console.log(`   Result: ${result.result || 'Skill executed successfully'}`);
      } catch (error) {
        console.error(`   Error: ${error instanceof Error ? error.message : String(error)}`);
      }
      await this.delay(1000);
    }
  }

  private async demoLearningMode(): Promise<void> {
    console.log('📚 Demo 4: Learning Mode');
    
    try {
      await this.engine!.startLearningMode();
      console.log('   Learning mode activated');
      
      // Simulate learning with feedback
      const learningInputs = [
        { input: 'optimize performance', feedback: 0.8 },
        { input: 'debug error', feedback: 0.6 },
        { input: 'generate report', feedback: 0.9 }
      ];

      for (let i = 0; i < learningInputs.length; i++) {
        const { input, feedback } = learningInputs[i];
        await this.engine!.processInput(input, 'text');
        this.engine!.provideFeedback(i.toString(), feedback);
        console.log(`   Learning from: "${input}" (feedback: ${feedback})`);
        await this.delay(800);
      }
      
    } catch (error) {
      console.error(`   Error: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  private async demoAdaptation(): Promise<void> {
    console.log('🔄 Demo 5: Adaptation');
    
    try {
      await this.engine!.saveCurrentAdaptation();
      console.log('   Adaptation saved to local storage');
      
      const state = this.engine!.getState();
      console.log(`   Adaptation level: ${Math.round(state.adaptationLevel * 100)}%`);
      console.log(`   Confidence level: ${Math.round(state.confidenceLevel * 100)}%`);
      
    } catch (error) {
      console.error(`   Error: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  private displayMetrics(): void {
    if (!this.engine) return;

    const metrics = this.engine.getMetrics() as CognitiveMetrics;
    console.log('\n📊 Final Metrics:');
    console.log(`   • Confidence Level: ${Math.round(metrics.confidenceLevel * 100)}%`);
    console.log(`   • Adaptation Level: ${Math.round(metrics.adaptationLevel * 100)}%`);
    console.log(`   • Active Skills: ${metrics.activeSkills}`);
    console.log(`   • Learning Events: ${metrics.learningEvents}`);
    console.log(`   • Context Size: ${metrics.contextSize}`);
    console.log(`   • Engine Status: ${metrics.isRunning ? 'RUNNING' : 'STOPPED'}`);
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  async stopDemo(): Promise<void> {
    if (this.engine) {
      await this.engine.stop();
      console.log('🛑 Cognitive Shell demo stopped');
    }
  }

  // Month 9 specific demo methods
  async testVisualProcessing(): Promise<void> {
    console.log('\n🎥 Testing Visual Processing System (Month 9)');
    console.log('='.repeat(50));

    try {
      const visualProcessor = this.engine?.getVisualProcessor();
      if (!visualProcessor) {
        console.log('❌ Visual processor not available');
        return;
      }

      console.log('📹 Visual Processor Status:');
      const metrics = (visualProcessor as VisualProcessor).getMetrics();
      console.log(`   • Supported: ${metrics.isSupported ? 'YES' : 'NO'}`);
      console.log(`   • Resolution: ${metrics.resolution}`);
      console.log(`   • Frame Rate: ${metrics.frameRate} fps`);
      console.log(`   • Object Detection: ${metrics.objectDetection ? 'ENABLED' : 'DISABLED'}`);
      console.log(`   • Gesture Recognition: ${metrics.gestureRecognition ? 'ENABLED' : 'DISABLED'}`);
      console.log(`   • OCR: ${metrics.ocrEnabled ? 'ENABLED' : 'DISABLED'}`);
      console.log(`   • Face Recognition: ${metrics.faceRecognition ? 'ENABLED' : 'DISABLED'}`);

      // Test configuration update
      console.log('\n🔧 Testing configuration update...');
      (visualProcessor as VisualProcessor).updateConfig({ frameRate: 60 });
      console.log('   ✅ Configuration updated to 60 fps');

      // Simulate visual events
      console.log('\n🎯 Simulating visual events...');
      (visualProcessor as VisualProcessor).emit('objectDetected', {
        id: 'demo-object-1',
        label: 'person',
        confidence: 0.95,
        boundingBox: { x: 100, y: 100, width: 200, height: 300 },
        timestamp: new Date()
      });
      console.log('   ✅ Object detection event simulated');

      (visualProcessor as VisualProcessor).emit('gestureDetected', {
        type: 'wave',
        confidence: 0.87,
        coordinates: { x: 300, y: 200 },
        direction: 'right',
        timestamp: new Date()
      });
      console.log('   ✅ Gesture detection event simulated');

      (visualProcessor as VisualProcessor).emit('textDetected', [{
        text: 'Hello KNIRV',
        confidence: 0.92,
        boundingBox: { x: 50, y: 50, width: 150, height: 30 },
        language: 'en'
      }]);
      console.log('   ✅ OCR detection event simulated');

    } catch (error) {
      console.error(`   Error: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  async testLoRAAdapter(): Promise<void> {
    console.log('\n🧠 Testing LoRA Adapter System (Month 9)');
    console.log('='.repeat(50));

    try {
      const loraAdapter = this.engine?.getLoRAAdapter();
      if (!loraAdapter) {
        console.log('❌ LoRA adapter not available');
        return;
      }

      console.log('🔧 LoRA Adapter Status:');
      const config = (loraAdapter as LoRAAdapter).getConfig();
      const metrics = (loraAdapter as LoRAAdapter).getMetrics();
      console.log(`   • Task Type: ${config.taskType}`);
      console.log(`   • Rank: ${config.rank}`);
      console.log(`   • Alpha: ${config.alpha}`);
      console.log(`   • Dropout: ${config.dropout}`);
      console.log(`   • Target Modules: ${config.targetModules.join(', ')}`);
      console.log(`   • Training Data Size: ${(loraAdapter as LoRAAdapter).getTrainingDataSize()}`);
      console.log(`   • Current Epoch: ${metrics.epoch}`);
      console.log(`   • Loss: ${metrics.loss.toFixed(4)}`);
      console.log(`   • Accuracy: ${(metrics.accuracy * 100).toFixed(1)}%`);

      // Test training data addition
      console.log('\n📚 Testing training data addition...');
      (loraAdapter as LoRAAdapter).enableTraining();

      const trainingData: TrainingData = {
        input: { text: 'Test input for LoRA training', features: [0.1, 0.2, 0.3] },
        output: { text: 'Expected output', confidence: 0.9 },
        feedback: 0.8,
        timestamp: new Date()
      };

      await (loraAdapter as LoRAAdapter).addTrainingData(trainingData);
      console.log('   ✅ Training data added successfully');

      // Test batch training
      console.log('\n🎯 Testing batch training...');
      const batchData: TrainingData[] = Array.from({ length: 5 }, (_, i) => ({
        input: { text: `Batch input ${i}`, features: [Math.random(), Math.random(), Math.random()] },
        output: { text: `Batch output ${i}`, confidence: 0.8 + Math.random() * 0.2 },
        feedback: 0.7 + Math.random() * 0.3,
        timestamp: new Date()
      }));

      await (loraAdapter as LoRAAdapter).trainOnBatch(batchData);
      console.log('   ✅ Batch training completed');

      // Test adaptation
      console.log('\n🔄 Testing adaptation...');
      const testInput = { text: 'Test adaptation input', features: [0.5, 0.6, 0.7] };
      const adaptedOutput = await (loraAdapter as LoRAAdapter).adapt(testInput, { text: 'Expected adapted output' }, 0.9);
      console.log('   ✅ Adaptation applied successfully');
      console.log(`   📊 Adapted output confidence: ${adaptedOutput.confidence?.toFixed(3) || 'N/A'}`);

      // Test weight export/import
      console.log('\n💾 Testing weight export/import...');
      const exportedWeights = (loraAdapter as LoRAAdapter).exportWeights();
      console.log(`   ✅ Weights exported (${exportedWeights.size} modules)`);

      (loraAdapter as LoRAAdapter).importWeights(exportedWeights);
      console.log('   ✅ Weights imported successfully');

      // Display final metrics
      const finalMetrics = (loraAdapter as LoRAAdapter).getMetrics();
      console.log('\n📊 Final LoRA Metrics:');
      console.log(`   • Epoch: ${finalMetrics.epoch}`);
      console.log(`   • Loss: ${finalMetrics.loss.toFixed(4)}`);
      console.log(`   • Accuracy: ${(finalMetrics.accuracy * 100).toFixed(1)}%`);
      console.log(`   • Learning Rate: ${finalMetrics.learningRate.toFixed(6)}`);

    } catch (error) {
      console.error(`   Error: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }
}

// Export demo instance for use in browser console
export const cognitiveDemo = new CognitiveShellDemo();

// Auto-run demo if in development mode
if (process.env.NODE_ENV === 'development') {
  console.log('🎭 KNIRV Cognitive Shell Demo Available');
  console.log('Run: cognitiveDemo.initializeDemo() then cognitiveDemo.runDemoSequence()');
}
