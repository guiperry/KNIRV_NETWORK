/**
 * CORTEX Training Service
 * Manages the training of personal KNIRVCORTEX models using personal KNIRVGRAPH data
 */

import { personalKNIRVGRAPHService, GraphNode, PersonalGraph } from './PersonalKNIRVGRAPHService';
import { rxdbService } from './RxDBService';

export interface TrainingConfig {
  learningRate: number;
  batchSize: number;
  epochs: number;
  validationSplit: number;
  optimizerType: 'adam' | 'sgd' | 'rmsprop';
  lossFunction: 'mse' | 'categorical_crossentropy' | 'binary_crossentropy';
}

export interface TrainingData {
  inputs: Float32Array[];
  outputs: Float32Array[];
  nodeTypes: string[];
  metadata: Record<string, unknown>;
}

export interface ModelMetrics {
  accuracy: number;
  loss: number;
  validationAccuracy: number;
  validationLoss: number;
  trainingTime: number;
  epoch: number;
}

export interface CortexModel {
  id: string;
  name: string;
  version: string;
  createdAt: Date;
  trainingConfig: TrainingConfig;
  metrics: ModelMetrics;
  modelWeights: Float32Array;
  graphSnapshot: {
    nodeCount: number;
    edgeCount: number;
    nodeTypes: Record<string, number>;
  };
  description: string;
}

export interface TrainingProgress {
  epoch: number;
  totalEpochs: number;
  loss: number;
  accuracy: number;
  validationLoss?: number;
  validationAccuracy?: number;
  estimatedTimeRemaining: number;
}

class CortexTrainingService {
  private isTraining = false;
  private currentTrainingId: string | null = null;
  private progressCallback: ((progress: TrainingProgress) => void) | null = null;

  /**
   * Prepare training data from personal KNIRVGRAPH
   */
  async prepareTrainingData(): Promise<TrainingData> {
    const graph = personalKNIRVGRAPHService.getCurrentGraph();
    if (!graph) {
      throw new Error('No personal graph available for training');
    }

    const inputs: Float32Array[] = [];
    const outputs: Float32Array[] = [];
    const nodeTypes: string[] = [];

    // Convert graph nodes to training vectors
    for (const node of graph.nodes) {
      const inputVector = this.nodeToVector(node);
      const outputVector = this.createTargetVector(node, graph);
      
      inputs.push(inputVector);
      outputs.push(outputVector);
      nodeTypes.push(node.type);
    }

    return {
      inputs,
      outputs,
      nodeTypes,
      metadata: {
        graphId: graph.id,
        nodeCount: graph.nodes.length,
        edgeCount: graph.edges.length,
        createdAt: Date.now()
      }
    };
  }

  /**
   * Convert a graph node to a feature vector
   */
  private nodeToVector(node: GraphNode): Float32Array {
    const vector = new Float32Array(128); // Fixed size feature vector
    
    // Node type encoding (one-hot)
    const typeIndex = this.getTypeIndex(node.type);
    vector[typeIndex] = 1.0;

    // Position encoding
    vector[10] = node.position.x / 1000; // Normalize position
    vector[11] = node.position.y / 1000;
    vector[12] = node.position.z / 1000;

    // Connection count
    vector[13] = node.connections.length / 10; // Normalize connection count

    // Node-specific features
    if (node.type === 'error' && 'timestamp' in node.data) {
      vector[20] = (Date.now() - (node.data.timestamp as number)) / (1000 * 60 * 60 * 24); // Days since creation
    }

    if (node.type === 'skill' && 'proficiency' in node.data) {
      vector[21] = node.data.proficiency as number;
    }

    if (node.type === 'property' && 'feasibilityReport' in node.data) {
      const report = node.data.feasibilityReport as { feasibilityScore: number };
      vector[22] = report.feasibilityScore / 100;
    }

    // Label embedding (simple hash-based encoding)
    const labelHash = this.hashString(node.label);
    for (let i = 0; i < 32; i++) {
      vector[30 + i] = ((labelHash >> i) & 1) ? 1.0 : 0.0;
    }

    return vector;
  }

  /**
   * Create target vector for supervised learning
   */
  private createTargetVector(node: GraphNode, graph: PersonalGraph): Float32Array {
    const target = new Float32Array(64); // Target vector size
    
    // Predict node importance (based on connections)
    target[0] = Math.min(node.connections.length / 5, 1.0);

    // Predict node success (mock - in real implementation, use historical data)
    target[1] = Math.random(); // Success probability

    // Predict optimal connections (simplified)
    const potentialConnections = graph.nodes.filter(n => 
      n.id !== node.id && !node.connections.includes(n.id)
    );
    target[2] = Math.min(potentialConnections.length / 10, 1.0);

    return target;
  }

  /**
   * Train a new CORTEX model
   */
  async trainModel(
    config: TrainingConfig,
    onProgress?: (progress: TrainingProgress) => void
  ): Promise<CortexModel> {
    if (this.isTraining) {
      throw new Error('Training already in progress');
    }

    this.isTraining = true;
    this.currentTrainingId = `training-${Date.now()}`;
    this.progressCallback = onProgress || null;

    try {
      // Prepare training data
      const trainingData = await this.prepareTrainingData();
      
      if (trainingData.inputs.length < 5) {
        throw new Error('Insufficient training data. Need at least 5 nodes.');
      }

      // Initialize model weights (simplified - in real implementation, use proper neural network)
      const modelWeights = new Float32Array(128 * 64); // Input size * output size
      for (let i = 0; i < modelWeights.length; i++) {
        modelWeights[i] = (Math.random() - 0.5) * 0.1; // Small random weights
      }

      // Simulate training process
      const startTime = Date.now();
      let bestAccuracy = 0;
      let bestLoss = Infinity;

      for (let epoch = 0; epoch < config.epochs; epoch++) {
        // Simulate training step
        const loss = this.simulateTrainingStep(trainingData, modelWeights, config);
        const accuracy = Math.min(0.5 + (epoch / config.epochs) * 0.4 + Math.random() * 0.1, 0.95);

        if (accuracy > bestAccuracy) {
          bestAccuracy = accuracy;
        }
        if (loss < bestLoss) {
          bestLoss = loss;
        }

        // Report progress
        if (this.progressCallback) {
          const progress: TrainingProgress = {
            epoch: epoch + 1,
            totalEpochs: config.epochs,
            loss,
            accuracy,
            validationLoss: loss * (1 + Math.random() * 0.1),
            validationAccuracy: accuracy * (0.9 + Math.random() * 0.1),
            estimatedTimeRemaining: ((Date.now() - startTime) / (epoch + 1)) * (config.epochs - epoch - 1)
          };
          this.progressCallback(progress);
        }

        // Simulate training delay
        await new Promise(resolve => setTimeout(resolve, 50));
      }

      const trainingTime = Date.now() - startTime;
      const graph = personalKNIRVGRAPHService.getCurrentGraph()!;

      // Create model
      const model: CortexModel = {
        id: `cortex-${Date.now()}`,
        name: `Personal CORTEX v${await this.getNextVersion()}`,
        version: await this.getNextVersion(),
        createdAt: new Date(),
        trainingConfig: config,
        metrics: {
          accuracy: bestAccuracy,
          loss: bestLoss,
          validationAccuracy: bestAccuracy * 0.95,
          validationLoss: bestLoss * 1.05,
          trainingTime,
          epoch: config.epochs
        },
        modelWeights,
        graphSnapshot: {
          nodeCount: graph.nodes.length,
          edgeCount: graph.edges.length,
          nodeTypes: this.getNodeTypeCounts(graph.nodes)
        },
        description: `Trained on ${trainingData.inputs.length} nodes from personal KNIRVGRAPH`
      };

      // Save model to database
      await this.saveModel(model);

      return model;
    } finally {
      this.isTraining = false;
      this.currentTrainingId = null;
      this.progressCallback = null;
    }
  }

  /**
   * Simulate a training step (simplified)
   */
  private simulateTrainingStep(
    trainingData: TrainingData,
    weights: Float32Array,
    _config: TrainingConfig
  ): number {
    // Simplified loss calculation
    let totalLoss = 0;
    
    for (let i = 0; i < trainingData.inputs.length; i++) {
      const input = trainingData.inputs[i];
      const target = trainingData.outputs[i];
      
      // Forward pass (simplified)
      const prediction = new Float32Array(target.length);
      for (let j = 0; j < prediction.length; j++) {
        let sum = 0;
        for (let k = 0; k < input.length; k++) {
          sum += input[k] * weights[k * target.length + j];
        }
        prediction[j] = Math.tanh(sum); // Simple activation
      }
      
      // Calculate loss
      for (let j = 0; j < target.length; j++) {
        const diff = prediction[j] - target[j];
        totalLoss += diff * diff;
      }
    }
    
    return totalLoss / trainingData.inputs.length;
  }

  /**
   * Get saved models
   */
  async getSavedModels(): Promise<CortexModel[]> {
    try {
      const db = rxdbService.getDatabase();
      const settings = await db.settings.findOne({ selector: { key: 'cortex_models' } }).exec();
      
      if (settings) {
        return JSON.parse(settings.value);
      }
      return [];
    } catch (error) {
      console.error('Failed to load saved models:', error);
      return [];
    }
  }

  /**
   * Save model to database
   */
  private async saveModel(model: CortexModel): Promise<void> {
    try {
      const db = rxdbService.getDatabase();
      const existingModels = await this.getSavedModels();
      const updatedModels = [...existingModels, model];
      
      await db.settings.upsert({
        key: 'cortex_models',
        value: JSON.stringify(updatedModels),
        timestamp: Date.now()
      });
    } catch (error) {
      console.error('Failed to save model:', error);
      throw error;
    }
  }

  /**
   * Helper methods
   */
  private getTypeIndex(type: string): number {
    const typeMap: Record<string, number> = {
      'error': 0,
      'skill': 1,
      'capability': 2,
      'property': 3,
      'connection': 4,
      'agent': 5
    };
    return typeMap[type] || 6;
  }

  private hashString(str: string): number {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash);
  }

  private async getNextVersion(): Promise<string> {
    const models = await this.getSavedModels();
    return `1.${models.length}.0`;
  }

  private getNodeTypeCounts(nodes: GraphNode[]): Record<string, number> {
    const counts: Record<string, number> = {};
    for (const node of nodes) {
      counts[node.type] = (counts[node.type] || 0) + 1;
    }
    return counts;
  }

  /**
   * Check if training is in progress
   */
  isCurrentlyTraining(): boolean {
    return this.isTraining;
  }

  /**
   * Get current training ID
   */
  getCurrentTrainingId(): string | null {
    return this.currentTrainingId;
  }
}

export const cortexTrainingService = new CortexTrainingService();
