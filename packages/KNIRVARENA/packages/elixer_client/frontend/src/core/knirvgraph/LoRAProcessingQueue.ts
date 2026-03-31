/**
 * LoRA Adapter Processing Queue for KNIRVGRAPH
 * 
 * Manages pending LoRA adapters awaiting core model training and validation
 * Provides queue management, priority scheduling, and batch processing
 */

import pino from 'pino';
import { EventEmitter } from 'events';
import { LoRAAdapterSkill } from '../lora/LoRAAdapterEngine';
import { TrainingDataset } from './LoRAAdapterTrainingPipeline';
import { HRMCoreModel, SkillDiscoveryResult } from './HRMCoreModel';

const logger = pino({ name: 'lora-processing-queue' });

export interface QueuedLoRAAdapter {
  queueId: string;
  loraAdapter: LoRAAdapterSkill;
  trainingDataset: TrainingDataset;
  priority: number;
  status: QueueStatus;
  submittedAt: Date;
  processingStartedAt?: Date;
  completedAt?: Date;
  retryCount: number;
  maxRetries: number;
  errorMessage?: string;
  discoveryResult?: SkillDiscoveryResult;
}

export enum QueueStatus {
  PENDING = 'pending',
  PROCESSING = 'processing',
  COMPLETED = 'completed',
  FAILED = 'failed',
  RETRYING = 'retrying'
}

export interface QueueConfig {
  maxConcurrentProcessing: number;
  maxQueueSize: number;
  defaultPriority: number;
  maxRetries: number;
  processingTimeout: number;
  batchSize: number;
  retryDelay: number;
}

export interface QueueMetrics {
  totalQueued: number;
  totalProcessing: number;
  totalCompleted: number;
  totalFailed: number;
  averageProcessingTime: number;
  successRate: number;
  queueThroughput: number;
}

export class LoRAProcessingQueue extends EventEmitter {
  private queue: Map<string, QueuedLoRAAdapter> = new Map();
  private processingQueue: Map<string, QueuedLoRAAdapter> = new Map();
  private completedQueue: Map<string, QueuedLoRAAdapter> = new Map();
  private failedQueue: Map<string, QueuedLoRAAdapter> = new Map();
  
  private hrmCoreModel: HRMCoreModel;
  private config: QueueConfig;
  private isProcessing: boolean = false;
  private isInitialized: boolean = false;
  private processingInterval?: NodeJS.Timeout;
  private metrics: QueueMetrics;

  constructor(hrmCoreModel: HRMCoreModel, config: Partial<QueueConfig> = {}) {
    super();
    
    this.hrmCoreModel = hrmCoreModel;
    this.config = {
      maxConcurrentProcessing: 3,
      maxQueueSize: 100,
      defaultPriority: 5,
      maxRetries: 3,
      processingTimeout: 300000, // 5 minutes
      batchSize: 5,
      retryDelay: 30000, // 30 seconds
      ...config
    };

    this.metrics = {
      totalQueued: 0,
      totalProcessing: 0,
      totalCompleted: 0,
      totalFailed: 0,
      averageProcessingTime: 0,
      successRate: 0,
      queueThroughput: 0
    };
  }

  /**
   * Initialize the processing queue
   */
  async initialize(): Promise<void> {
    logger.info('Initializing LoRA Processing Queue...');

    try {
      // Ensure HRM Core Model is ready
      if (!this.hrmCoreModel.isReady()) {
        await this.hrmCoreModel.initialize();
      }

      // Start processing loop
      this.startProcessingLoop();

      this.isInitialized = true;
      logger.info('LoRA Processing Queue initialized successfully');

    } catch (error) {
      logger.error({ error }, 'Failed to initialize LoRA Processing Queue');
      throw error;
    }
  }

  /**
   * Add LoRA adapter to processing queue
   */
  async enqueue(
    loraAdapter: LoRAAdapterSkill,
    trainingDataset: TrainingDataset,
    priority: number = this.config.defaultPriority
  ): Promise<string> {
    if (!this.isInitialized) {
      throw new Error('LoRA Processing Queue not initialized');
    }

    if (this.queue.size >= this.config.maxQueueSize) {
      throw new Error('Processing queue is full');
    }

    const queueId = `queue_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

    const queuedAdapter: QueuedLoRAAdapter = {
      queueId,
      loraAdapter,
      trainingDataset,
      priority,
      status: QueueStatus.PENDING,
      submittedAt: new Date(),
      retryCount: 0,
      maxRetries: this.config.maxRetries
    };

    this.queue.set(queueId, queuedAdapter);
    this.metrics.totalQueued++;

    logger.info({
      queueId,
      skillId: loraAdapter.skillId,
      priority,
      queueSize: this.queue.size
    }, 'LoRA adapter added to processing queue');

    this.emit('enqueued', queuedAdapter);
    return queueId;
  }

  /**
   * Get queue status
   */
  getQueueStatus(queueId: string): QueuedLoRAAdapter | undefined {
    return this.queue.get(queueId) || 
           this.processingQueue.get(queueId) || 
           this.completedQueue.get(queueId) || 
           this.failedQueue.get(queueId);
  }

  /**
   * Get all pending items
   */
  getPendingItems(): QueuedLoRAAdapter[] {
    return Array.from(this.queue.values())
      .filter(item => item.status === QueueStatus.PENDING)
      .sort((a, b) => b.priority - a.priority); // Higher priority first
  }

  /**
   * Get processing items
   */
  getProcessingItems(): QueuedLoRAAdapter[] {
    return Array.from(this.processingQueue.values());
  }

  /**
   * Get completed items
   */
  getCompletedItems(): QueuedLoRAAdapter[] {
    return Array.from(this.completedQueue.values());
  }

  /**
   * Get failed items
   */
  getFailedItems(): QueuedLoRAAdapter[] {
    return Array.from(this.failedQueue.values());
  }

  /**
   * Start processing loop
   */
  private startProcessingLoop(): void {
    this.processingInterval = setInterval(async () => {
      if (!this.isProcessing) {
        await this.processQueue();
      }
    }, 5000) as unknown as NodeJS.Timeout; // Check every 5 seconds
  }

  /**
   * Process queue items
   */
  private async processQueue(): Promise<void> {
    if (this.processingQueue.size >= this.config.maxConcurrentProcessing) {
      return; // Already at max concurrent processing
    }

    const pendingItems = this.getPendingItems();
    const availableSlots = this.config.maxConcurrentProcessing - this.processingQueue.size;
    const itemsToProcess = pendingItems.slice(0, Math.min(availableSlots, this.config.batchSize));

    if (itemsToProcess.length === 0) {
      return; // No items to process
    }

    logger.info({
      itemsToProcess: itemsToProcess.length,
      availableSlots,
      queueSize: this.queue.size
    }, 'Processing queue batch');

    // Process items concurrently
    const processingPromises = itemsToProcess.map(item => this.processItem(item));
    await Promise.allSettled(processingPromises);
  }

  /**
   * Process individual queue item
   */
  private async processItem(queuedAdapter: QueuedLoRAAdapter): Promise<void> {
    const { queueId, loraAdapter, trainingDataset } = queuedAdapter;

    try {
      // Move to processing queue
      this.queue.delete(queueId);
      queuedAdapter.status = QueueStatus.PROCESSING;
      queuedAdapter.processingStartedAt = new Date();
      this.processingQueue.set(queueId, queuedAdapter);
      this.metrics.totalProcessing++;

      logger.info({
        queueId,
        skillId: loraAdapter.skillId
      }, 'Starting LoRA adapter processing');

      this.emit('processingStarted', queuedAdapter);

      // Set processing timeout
      const timeoutPromise = new Promise<never>((_, reject) => {
        setTimeout(() => reject(new Error('Processing timeout')), this.config.processingTimeout);
      });

      // Process with HRM Core Model
      const processingPromise = this.hrmCoreModel.discoverSkill(loraAdapter, trainingDataset);

      // Race between processing and timeout
      const discoveryResult = await Promise.race([processingPromise, timeoutPromise]);

      // Processing completed successfully
      queuedAdapter.status = QueueStatus.COMPLETED;
      queuedAdapter.completedAt = new Date();
      queuedAdapter.discoveryResult = discoveryResult;

      // Move to completed queue
      this.processingQueue.delete(queueId);
      this.completedQueue.set(queueId, queuedAdapter);
      this.metrics.totalCompleted++;

      // Update metrics
      this.updateMetrics(queuedAdapter);

      logger.info({
        queueId,
        skillId: loraAdapter.skillId,
        discoveredName: discoveryResult.discoveredName,
        confidence: discoveryResult.confidence
      }, 'LoRA adapter processing completed');

      this.emit('processingCompleted', queuedAdapter);

    } catch (error) {
      await this.handleProcessingError(queuedAdapter, error as Error);
    }
  }

  /**
   * Handle processing error
   */
  private async handleProcessingError(queuedAdapter: QueuedLoRAAdapter, error: Error): Promise<void> {
    const { queueId, loraAdapter } = queuedAdapter;

    logger.error({
      queueId,
      skillId: loraAdapter.skillId,
      error: error.message,
      retryCount: queuedAdapter.retryCount
    }, 'LoRA adapter processing failed');

    queuedAdapter.errorMessage = error.message;
    queuedAdapter.retryCount++;

    // Check if we should retry
    if (queuedAdapter.retryCount <= queuedAdapter.maxRetries) {
      // Schedule retry
      queuedAdapter.status = QueueStatus.RETRYING;
      
      setTimeout(() => {
        // Move back to pending queue for retry
        this.processingQueue.delete(queueId);
        queuedAdapter.status = QueueStatus.PENDING;
        this.queue.set(queueId, queuedAdapter);
        
        logger.info({
          queueId,
          skillId: loraAdapter.skillId,
          retryCount: queuedAdapter.retryCount
        }, 'Retrying LoRA adapter processing');

        this.emit('retrying', queuedAdapter);
      }, this.config.retryDelay);

    } else {
      // Max retries exceeded, mark as failed
      queuedAdapter.status = QueueStatus.FAILED;
      queuedAdapter.completedAt = new Date();

      this.processingQueue.delete(queueId);
      this.failedQueue.set(queueId, queuedAdapter);
      this.metrics.totalFailed++;

      this.emit('processingFailed', queuedAdapter);
    }
  }

  /**
   * Update processing metrics
   */
  private updateMetrics(queuedAdapter: QueuedLoRAAdapter): void {
    if (queuedAdapter.processingStartedAt && queuedAdapter.completedAt) {
      const processingTime = queuedAdapter.completedAt.getTime() - queuedAdapter.processingStartedAt.getTime();
      
      // Update average processing time
      const totalProcessed = this.metrics.totalCompleted + this.metrics.totalFailed;
      this.metrics.averageProcessingTime = 
        (this.metrics.averageProcessingTime * (totalProcessed - 1) + processingTime) / totalProcessed;
    }

    // Update success rate
    const totalProcessed = this.metrics.totalCompleted + this.metrics.totalFailed;
    if (totalProcessed > 0) {
      this.metrics.successRate = this.metrics.totalCompleted / totalProcessed;
    }

    // Update throughput (items per hour)
    const hoursSinceStart = (Date.now() - ((this.metrics as { startTime?: number }).startTime || Date.now())) / (1000 * 60 * 60);
    this.metrics.queueThroughput = totalProcessed / Math.max(hoursSinceStart, 0.1);
  }

  /**
   * Get queue metrics
   */
  getMetrics(): QueueMetrics {
    return { ...this.metrics };
  }

  /**
   * Clear completed items
   */
  clearCompleted(): number {
    const count = this.completedQueue.size;
    this.completedQueue.clear();
    logger.info({ clearedCount: count }, 'Cleared completed queue items');
    return count;
  }

  /**
   * Clear failed items
   */
  clearFailed(): number {
    const count = this.failedQueue.size;
    this.failedQueue.clear();
    logger.info({ clearedCount: count }, 'Cleared failed queue items');
    return count;
  }

  /**
   * Stop processing
   */
  stop(): void {
    if (this.processingInterval) {
      clearInterval(this.processingInterval);
      this.processingInterval = undefined;
    }
    
    this.isProcessing = false;
    logger.info('LoRA Processing Queue stopped');
  }

  /**
   * Get queue size
   */
  getQueueSize(): number {
    return this.queue.size;
  }

  /**
   * Check if ready
   */
  isReady(): boolean {
    return this.isInitialized && this.hrmCoreModel.isReady();
  }
}
