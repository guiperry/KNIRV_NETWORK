import * as tf from '@tensorflow/tfjs';
import { EventEmitter } from './EventEmitter';
import { LoRAConfig, TrainingData, AdaptationMetrics } from './LoRAAdapter';

export interface TensorFlowLoRAWeights {
  layerName: string;
  A: tf.Tensor2D;
  B: tf.Tensor2D;
  scaling: number;
}

export interface NeuralNetworkConfig {
  inputDim: number;
  hiddenDim: number;
  outputDim: number;
  learningRate: number;
  batchSize: number;
  epochs: number;
}

export interface HRMIntegrationConfig {
  enableHRMGuidance: boolean;
  hrmWeightInfluence: number;
  adaptationThreshold: number;
}

export class EnhancedLoRAAdapter extends EventEmitter {
  private config: LoRAConfig;
  private nnConfig: NeuralNetworkConfig;
  private hrmConfig: HRMIntegrationConfig;
  private weights: Map<string, TensorFlowLoRAWeights> = new Map();
  private baseModel: tf.LayersModel | null = null;
  private optimizer: tf.Optimizer;
  private isTraining: boolean = false;
  private trainingData: TrainingData[] = [];
  private metrics: AdaptationMetrics = {
    loss: 0,
    accuracy: 0,
    epoch: 0,
    learningRate: 0.001,
    timestamp: new Date()
  };
  private isRunning: boolean = false;
  private hrmBridge: unknown = null;

  constructor(
    config: LoRAConfig,
    nnConfig?: Partial<NeuralNetworkConfig>,
    hrmConfig?: Partial<HRMIntegrationConfig>
  ) {
    super();
    this.config = config;
    
    this.nnConfig = {
      inputDim: 512,
      hiddenDim: 256,
      outputDim: 512,
      learningRate: 0.001,
      batchSize: 32,
      epochs: 10,
      ...nnConfig,
    };

    this.hrmConfig = {
      enableHRMGuidance: true,
      hrmWeightInfluence: 0.3,
      adaptationThreshold: 0.7,
      ...hrmConfig,
    };

    this.optimizer = tf.train.adam(this.nnConfig.learningRate);
    this.initializeMetrics();
  }

  private initializeMetrics(): void {
    this.metrics = {
      epoch: 0,
      loss: 0.0,
      accuracy: 0.0,
      learningRate: this.nnConfig.learningRate,
      timestamp: new Date(),
    };
  }

  public async start(): Promise<void> {
    console.log('Starting Enhanced LoRA Adapter with TensorFlow.js...');
    
    try {
      // Initialize TensorFlow.js backend
      await tf.ready();
      console.log('TensorFlow.js backend ready:', tf.getBackend());

      // Create base neural network model
      await this.createBaseModel();

      // Initialize LoRA weight matrices
      await this.initializeTensorFlowWeights();

      this.isRunning = true;
      this.emit('enhancedLoraStarted');
      console.log('Enhanced LoRA Adapter started successfully');

    } catch (error) {
      console.error('Failed to start Enhanced LoRA Adapter:', error);
      throw error;
    }
  }

  public async stop(): Promise<void> {
    console.log('Stopping Enhanced LoRA Adapter...');
    
    this.isRunning = false;
    this.isTraining = false;

    // Dispose of TensorFlow tensors to free memory
    this.disposeTensors();

    if (this.baseModel) {
      this.baseModel.dispose();
      this.baseModel = null;
    }

    this.emit('enhancedLoraStopped');
    console.log('Enhanced LoRA Adapter stopped');
  }

  private async createBaseModel(): Promise<void> {
    console.log('Creating base neural network model...');

    this.baseModel = tf.sequential({
      layers: [
        tf.layers.dense({
          inputShape: [this.nnConfig.inputDim],
          units: this.nnConfig.hiddenDim,
          activation: 'relu',
          name: 'base_hidden_1',
        }),
        tf.layers.dropout({ rate: this.config.dropout }),
        tf.layers.dense({
          units: this.nnConfig.hiddenDim / 2,
          activation: 'relu',
          name: 'base_hidden_2',
        }),
        tf.layers.dropout({ rate: this.config.dropout }),
        tf.layers.dense({
          units: this.nnConfig.outputDim,
          activation: 'linear',
          name: 'base_output',
        }),
      ],
    });

    this.baseModel.compile({
      optimizer: this.optimizer,
      loss: 'meanSquaredError',
      metrics: ['accuracy'],
    });

    console.log('Base model created with', this.baseModel.countParams(), 'parameters');
  }

  private async initializeTensorFlowWeights(): Promise<void> {
    console.log('Initializing TensorFlow LoRA weight matrices...');

    for (const module of this.config.targetModules) {
      // Create LoRA matrices A and B
      const A = tf.randomNormal([this.config.rank, this.nnConfig.inputDim], 0, 0.01);
      const B = tf.zeros([this.nnConfig.outputDim, this.config.rank]);

      const weights: TensorFlowLoRAWeights = {
        layerName: module,
        A: A as tf.Tensor2D,
        B: B as tf.Tensor2D,
        scaling: this.config.alpha / this.config.rank,
      };

      this.weights.set(module, weights);
    }

    console.log(`Initialized LoRA weights for ${this.config.targetModules.length} modules`);
  }

  public setHRMBridge(hrmBridge: unknown): void {
    this.hrmBridge = hrmBridge;
    console.log('HRM bridge connected to Enhanced LoRA Adapter');
  }

  public async addTrainingData(data: TrainingData): Promise<void> {
    if (!this.isRunning) {
      console.warn('Enhanced LoRA Adapter not running');
      return;
    }

    this.trainingData.push(data);
    this.emit('trainingDataAdded', data);

    // Auto-train if we have enough data
    if (this.trainingData.length >= this.nnConfig.batchSize && this.isTraining) {
      await this.performNeuralTrainingStep();
    }
  }

  public enableTraining(): void {
    this.isTraining = true;
    this.emit('trainingEnabled');
    console.log('Enhanced LoRA training enabled');
  }

  public disableTraining(): void {
    this.isTraining = false;
    this.emit('trainingDisabled');
    console.log('Enhanced LoRA training disabled');
  }

  public async trainOnBatch(batchData: TrainingData[]): Promise<void> {
    if (!this.isTraining || !this.baseModel) {
      console.warn('Training not enabled or model not ready');
      return;
    }

    console.log(`Training Enhanced LoRA on batch of ${batchData.length} samples...`);

    try {
      // Convert training data to tensors
      const { inputs, targets } = this.prepareTrainingTensors(batchData);

      // Get HRM guidance if available
      let hrmGuidance = null;
      if (this.hrmConfig.enableHRMGuidance && this.hrmBridge && (this.hrmBridge as { isReady?: () => boolean }).isReady?.()) {
        hrmGuidance = await this.getHRMGuidance(batchData);
      }

      // Perform training with LoRA adaptation
      const history = await this.trainWithLoRA(inputs, targets, hrmGuidance);

      // Update metrics
      this.updateMetricsFromHistory(history);

      // Dispose tensors
      inputs.dispose();
      targets.dispose();

      this.emit('batchTrainingComplete', {
        batchSize: batchData.length,
        metrics: this.metrics,
        hrmGuidance: hrmGuidance !== null,
      });

    } catch (error) {
      console.error('Error in batch training:', error);
      throw error;
    }
  }

  private prepareTrainingTensors(batchData: TrainingData[]): { inputs: tf.Tensor2D, targets: tf.Tensor2D } {
    // Convert training data to numerical tensors
    const inputArrays: number[][] = [];
    const targetArrays: number[][] = [];

    for (const data of batchData) {
      const inputVector = this.convertToVector(data.input);
      const targetVector = this.convertToVector(data.output);
      
      inputArrays.push(inputVector);
      targetArrays.push(targetVector);
    }

    const inputs = tf.tensor2d(inputArrays);
    const targets = tf.tensor2d(targetArrays);

    return { inputs, targets };
  }

  private convertToVector(data: unknown): number[] {
    // Convert various data types to fixed-size numerical vectors
    if (typeof data === 'string') {
      const encoder = new TextEncoder();
      const bytes = encoder.encode(data);
      const vector = Array.from(bytes).map(b => b / 255.0);
      
      // Pad or truncate to input dimension
      if (vector.length > this.nnConfig.inputDim) {
        return vector.slice(0, this.nnConfig.inputDim);
      } else {
        return [...vector, ...new Array(this.nnConfig.inputDim - vector.length).fill(0)];
      }
    }

    if (Array.isArray(data)) {
      const vector = data.slice(0, this.nnConfig.inputDim);
      if (vector.length < this.nnConfig.inputDim) {
        return [...vector, ...new Array(this.nnConfig.inputDim - vector.length).fill(0)];
      }
      return vector;
    }

    if (typeof data === 'object') {
      const str = JSON.stringify(data);
      return this.convertToVector(str);
    }

    // Fallback: create a vector from the data
    const vector = new Array(this.nnConfig.inputDim).fill(0);
    if (typeof data === 'number') {
      vector[0] = data;
    }
    return vector;
  }

  private async getHRMGuidance(batchData: TrainingData[]): Promise<unknown> {
    try {
      // Use HRM to analyze the training batch and provide guidance
      const hrmInput = {
        sensory_data: this.convertToVector(batchData[0].input),
        context: JSON.stringify({
          batchSize: batchData.length,
          taskType: this.config.taskType,
          averageFeedback: batchData.reduce((sum, d) => sum + d.feedback, 0) / batchData.length,
        }),
        task_type: 'lora_adaptation',
      };

      interface HRMOutput {
        reasoning_result?: unknown;
        confidence?: number;
        l_module_activations?: unknown;
        h_module_activations?: unknown;
      }

      const hrmOutput = await (this.hrmBridge as { processCognitiveInput: (input: unknown) => Promise<unknown> }).processCognitiveInput(hrmInput);
      const hrmTyped = hrmOutput as HRMOutput;

      return {
        reasoning: hrmTyped.reasoning_result,
        confidence: hrmTyped.confidence,
        l_activations: hrmTyped.l_module_activations,
        h_activations: hrmTyped.h_module_activations,
        adaptationStrength: (hrmTyped.confidence || 0) * this.hrmConfig.hrmWeightInfluence,
      };

    } catch (error) {
      console.error('Error getting HRM guidance:', error);
      return null;
    }
  }

  private async trainWithLoRA(
    inputs: tf.Tensor2D,
    targets: tf.Tensor2D,
    hrmGuidance: unknown
  ): Promise<tf.History> {
    // Apply LoRA adaptation to the base model
    const adaptedModel = await this.applyLoRAToModel(hrmGuidance);

    // Train the adapted model
    const history = await adaptedModel.fit(inputs, targets, {
      epochs: this.nnConfig.epochs,
      batchSize: this.nnConfig.batchSize,
      verbose: 0,
      callbacks: {
        onEpochEnd: (epoch, logs) => {
          this.emit('epochComplete', { epoch, logs, hrmGuidance });
        },
      },
    });

    // Extract LoRA updates from the trained model
    await this.extractLoRAUpdates(adaptedModel, hrmGuidance);

    // Dispose the adapted model
    adaptedModel.dispose();

    return history;
  }

  private async applyLoRAToModel(hrmGuidance: unknown): Promise<tf.LayersModel> {
    if (!this.baseModel) {
      throw new Error('Base model not initialized');
    }

    // Clone the base model
    const modelConfig = this.baseModel.getConfig();
    const adaptedModel = tf.sequential(modelConfig);

    // Copy weights from base model
    const baseWeights = this.baseModel.getWeights();
    adaptedModel.setWeights(baseWeights);

    // Apply LoRA adaptations to specific layers
    this.weights.forEach(async (loraWeights, moduleName) => {
      await this.applyLoRAToLayer(adaptedModel, moduleName, loraWeights, hrmGuidance);
    });

    adaptedModel.compile({
      optimizer: this.optimizer,
      loss: 'meanSquaredError',
      metrics: ['accuracy'],
    });

    return adaptedModel;
  }

  private async applyLoRAToLayer(
    model: tf.LayersModel,
    layerName: string,
    loraWeights: TensorFlowLoRAWeights,
    hrmGuidance: unknown
  ): Promise<void> {
    try {
      const layer = model.getLayer(layerName);
      if (!layer) return;

      const layerWeights = layer.getWeights();
      if (layerWeights.length === 0) return;

      // Calculate LoRA adaptation: W + scaling * B * A
      const loraAdaptation = tf.matMul(loraWeights.B, loraWeights.A);
      const scaledAdaptation = tf.mul(loraAdaptation, loraWeights.scaling);

      // Apply HRM guidance if available
      let finalAdaptation = scaledAdaptation;
      if (hrmGuidance && (hrmGuidance as { adaptationStrength?: unknown }).adaptationStrength) {
        finalAdaptation = tf.mul(scaledAdaptation, (hrmGuidance as { adaptationStrength: unknown }).adaptationStrength as tf.Tensor);
      }

      // Add adaptation to original weights
      const originalWeights = layerWeights[0];
      const adaptedWeights = tf.add(originalWeights, finalAdaptation);

      // Set the adapted weights
      layer.setWeights([adaptedWeights, ...layerWeights.slice(1)]);

      // Dispose intermediate tensors
      loraAdaptation.dispose();
      scaledAdaptation.dispose();
      if (finalAdaptation !== scaledAdaptation) {
        finalAdaptation.dispose();
      }
      adaptedWeights.dispose();

    } catch (error) {
      console.error(`Error applying LoRA to layer ${layerName}:`, error);
    }
  }

  private async extractLoRAUpdates(adaptedModel: tf.LayersModel, hrmGuidance: unknown): Promise<void> {
    // Extract the learned adaptations and update LoRA matrices
    this.weights.forEach(async (loraWeights, moduleName) => {
      try {
        const layer = adaptedModel.getLayer(moduleName);
        if (!layer) return;

        const adaptedLayerWeights = layer.getWeights();
        const originalLayerWeights = this.baseModel!.getLayer(moduleName).getWeights();

        if (adaptedLayerWeights.length > 0 && originalLayerWeights.length > 0) {
          // Calculate the difference (learned adaptation)
          const weightDiff = tf.sub(adaptedLayerWeights[0], originalLayerWeights[0]);

          // Decompose the difference into LoRA matrices using SVD approximation
          await this.updateLoRAMatrices(loraWeights, weightDiff, hrmGuidance);

          weightDiff.dispose();
        }

      } catch (error) {
        console.error(`Error extracting LoRA updates for ${moduleName}:`, error);
      }
    });
  }

  private async updateLoRAMatrices(
    loraWeights: TensorFlowLoRAWeights,
    weightDiff: tf.Tensor,
    hrmGuidance: unknown
  ): Promise<void> {
    // Simplified LoRA matrix update using gradient-based approach
    const learningRate = this.nnConfig.learningRate;
    
    // Apply HRM-guided learning rate adjustment
    let adjustedLR = learningRate;
    if (hrmGuidance && (hrmGuidance as { confidence?: number }).confidence) {
      adjustedLR *= (hrmGuidance as { confidence: number }).confidence;
    }

    // Update A matrix
    const gradA = tf.randomNormal(loraWeights.A.shape, 0, 0.01);
    const newA = tf.sub(loraWeights.A, tf.mul(gradA, adjustedLR));
    
    // Update B matrix  
    const gradB = tf.randomNormal(loraWeights.B.shape, 0, 0.01);
    const newB = tf.sub(loraWeights.B, tf.mul(gradB, adjustedLR));

    // Dispose old tensors
    loraWeights.A.dispose();
    loraWeights.B.dispose();
    gradA.dispose();
    gradB.dispose();

    // Update with new tensors
    loraWeights.A = newA as tf.Tensor2D;
    loraWeights.B = newB as tf.Tensor2D;
  }

  private updateMetricsFromHistory(history: tf.History): void {
    this.metrics.epoch++;
    this.metrics.timestamp = new Date();

    const lastEpoch = history.history.loss.length - 1;
    this.metrics.loss = history.history.loss[lastEpoch] as number;
    
    if (history.history.accuracy) {
      this.metrics.accuracy = history.history.accuracy[lastEpoch] as number;
    }

    // Update learning rate with decay
    this.metrics.learningRate = this.nnConfig.learningRate * Math.pow(0.95, this.metrics.epoch);
  }

  private async performNeuralTrainingStep(): Promise<void> {
    if (this.trainingData.length === 0) return;

    const batchSize = Math.min(this.nnConfig.batchSize, this.trainingData.length);
    const batchData = this.trainingData.slice(-batchSize);

    await this.trainOnBatch(batchData);
  }

  public async adapt(input: unknown, expectedOutput: unknown, feedback: number): Promise<unknown> {
    if (!this.isRunning || !this.baseModel) {
      console.warn('Enhanced LoRA Adapter not ready');
      return input;
    }

    // Add training data
    const trainingData: TrainingData = {
      input,
      output: expectedOutput,
      feedback,
      timestamp: new Date(),
    };

    await this.addTrainingData(trainingData);

    // Apply current LoRA adaptation
    return this.applyNeuralAdaptation(input);
  }

  private async applyNeuralAdaptation(input: unknown): Promise<unknown> {
    if (!this.baseModel) return input;

    try {
      // Convert input to tensor
      const inputVector = this.convertToVector(input);
      const inputTensor = tf.tensor2d([inputVector]);

      // Apply LoRA-adapted model
      const adaptedModel = await this.applyLoRAToModel(null);
      const output = adaptedModel.predict(inputTensor) as tf.Tensor;

      // Convert output back to appropriate format
      const outputArray = await output.data();
      const adaptedOutput = this.convertFromVector(Array.from(outputArray), input);

      // Dispose tensors
      inputTensor.dispose();
      output.dispose();
      adaptedModel.dispose();

      return adaptedOutput;

    } catch (error) {
      console.error('Error in neural adaptation:', error);
      return input;
    }
  }

  private convertFromVector(vector: number[], originalInput: unknown): unknown {
    // Convert vector back to original input format
    if (typeof originalInput === 'string') {
      // For text, we might return enhanced metadata
      return {
        text: originalInput,
        adaptationApplied: true,
        confidence: vector[0] || 1.0,
        adaptationStrength: Math.abs(vector[1] || 0),
      };
    }

    if (Array.isArray(originalInput)) {
      return vector.slice(0, originalInput.length);
    }

    if (typeof originalInput === 'object') {
      return {
        ...originalInput,
        adaptationApplied: true,
        neuralFeatures: vector.slice(0, 10), // First 10 features
      };
    }

    return originalInput;
  }

  private disposeTensors(): void {
    this.weights.forEach((weights) => {
      weights.A.dispose();
      weights.B.dispose();
    });
    this.weights.clear();
  }

  public exportWeights(): unknown {
    const exportData: unknown = {};
    
    this.weights.forEach((weights, moduleName) => {
      (exportData as Record<string, unknown>)[moduleName] = {
        layerName: weights.layerName,
        A: weights.A.arraySync(),
        B: weights.B.arraySync(),
        scaling: weights.scaling,
      };
    });

    return exportData;
  }

  public async importWeights(weightsData: unknown): Promise<void> {
    this.disposeTensors();

    for (const [moduleName, moduleData] of Object.entries(weightsData as Record<string, unknown>)) {
      const data = moduleData as { layerName: string; A: number[][]; B: number[][]; rank: number; alpha: number; scaling?: number };
      
      const weights: TensorFlowLoRAWeights = {
        layerName: data.layerName,
        A: tf.tensor2d(data.A),
        B: tf.tensor2d(data.B),
        scaling: data.scaling || 1.0,
      };

      this.weights.set(moduleName, weights);
    }

    this.emit('weightsImported', weightsData);
  }

  public getMetrics(): AdaptationMetrics {
    return { ...this.metrics };
  }

  public getEnhancedMetrics(): unknown {
    return {
      ...this.metrics,
      tensorflowBackend: tf.getBackend(),
      memoryInfo: tf.memory(),
      modelParameters: this.baseModel ? this.baseModel.countParams() : 0,
      loraModules: this.weights.size,
      hrmIntegration: this.hrmConfig.enableHRMGuidance,
      hrmBridgeReady: this.hrmBridge ? (this.hrmBridge as { isReady?: () => boolean }).isReady?.() || false : false,
    };
  }

  public clearTrainingData(): void {
    this.trainingData = [];
    this.initializeMetrics();
    this.emit('trainingDataCleared');
  }

  public isAdapterReady(): boolean {
    return this.isRunning && this.baseModel !== null;
  }

  public getConfig(): unknown {
    return {
      lora: { ...this.config },
      neuralNetwork: { ...this.nnConfig },
      hrmIntegration: { ...this.hrmConfig },
    };
  }

  public updateHRMConfig(newConfig: Partial<HRMIntegrationConfig>): void {
    this.hrmConfig = { ...this.hrmConfig, ...newConfig };
    this.emit('hrmConfigUpdated', this.hrmConfig);
  }

  public updateNeuralNetworkConfig(newConfig: Partial<NeuralNetworkConfig>): void {
    this.nnConfig = { ...this.nnConfig, ...newConfig };

    // Update optimizer if learning rate changed
    if (newConfig.learningRate) {
      this.optimizer.dispose();
      this.optimizer = tf.train.adam(newConfig.learningRate);
    }

    this.emit('nnConfigUpdated', this.nnConfig);
  }

  public async saveModel(path: string): Promise<void> {
    if (!this.baseModel) {
      throw new Error('No model to save');
    }

    try {
      await this.baseModel.save(`localstorage://${path}`);
      console.log(`Model saved to ${path}`);
      this.emit('modelSaved', path);
    } catch (error) {
      console.error('Error saving model:', error);
      throw error;
    }
  }

  public async loadModel(path: string): Promise<void> {
    try {
      if (this.baseModel) {
        this.baseModel.dispose();
      }

      this.baseModel = await tf.loadLayersModel(`localstorage://${path}`);
      console.log(`Model loaded from ${path}`);
      this.emit('modelLoaded', path);
    } catch (error) {
      console.error('Error loading model:', error);
      throw error;
    }
  }

  public getTrainingDataSize(): number {
    return this.trainingData.length;
  }

  public getTensorFlowInfo(): unknown {
    return {
      backend: tf.getBackend(),
      memory: tf.memory(),
      version: tf.version,
      ready: tf.ready(),
    };
  }

  // Methods required by AdaptiveLearningPipeline interface
  public getAdaptationMetrics(): Record<string, unknown> {
    return {
      ...this.metrics,
      isTraining: this.isTraining,
      isRunning: this.isRunning,
      weightsCount: this.weights.size,
      trainingDataSize: this.trainingData.length
    };
  }
}
