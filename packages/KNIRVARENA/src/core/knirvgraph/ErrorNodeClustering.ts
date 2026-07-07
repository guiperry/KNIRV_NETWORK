/**
 * ErrorNode Clustering Algorithm for KNIRVGRAPH LoRA Adapter Creation
 * 
 * Groups similar ErrorNodes into clusters for focused problem domains
 * Enables agent assignment and competitive solution development
 */

import pino from 'pino';

const logger = pino({ name: 'error-node-clustering' });

export interface ErrorNode {
  id: string;
  errorType: string;
  errorMessage: string;
  stackTrace?: string;
  context: Record<string, unknown>;
  severity: 'low' | 'medium' | 'high' | 'critical';
  timestamp: Date;
  bountyAmount: number;
  tags: string[];
  metadata: Record<string, unknown>;
}

export interface ErrorCluster {
  clusterId: string;
  clusterName: string;
  description: string;
  errorNodes: ErrorNode[];
  centroid: ErrorFeatureVector;
  similarity: number;
  totalBounty: number;
  assignedAgents: string[];
  solutions: Solution[];
  ownerAgent?: string;
  createdAt: Date;
  lastUpdated: Date;
}

export interface ErrorFeatureVector {
  errorTypeVector: number[];
  contextVector: number[];
  severityScore: number;
  tagVector: number[];
  semanticVector: number[];
}

export interface Solution {
  solutionId: string;
  agentId: string;
  errorNodeId: string;
  clusterId: string;
  solutionCode: string;
  description: string;
  validationStatus: 'pending' | 'validated' | 'rejected';
  dveValidationScore?: number;
  submittedAt: Date;
  validatedAt?: Date;
  bountyAwarded?: number;
}

export interface ClusteringConfig {
  maxClustersPerRun: number;
  similarityThreshold: number;
  minClusterSize: number;
  maxClusterSize: number;
  featureWeights: {
    errorType: number;
    context: number;
    severity: number;
    tags: number;
    semantic: number;
  };
}

export class ErrorNodeClustering {
  private config: ClusteringConfig;
  private clusters: Map<string, ErrorCluster> = new Map();
  private errorNodes: Map<string, ErrorNode> = new Map();
  private isInitialized: boolean = false;

  constructor(config: Partial<ClusteringConfig> = {}) {
    this.config = {
      maxClustersPerRun: 50,
      similarityThreshold: 0.7,
      minClusterSize: 3,
      maxClusterSize: 20,
      featureWeights: {
        errorType: 0.3,
        context: 0.2,
        severity: 0.1,
        tags: 0.2,
        semantic: 0.2
      },
      ...config
    };
  }

  /**
   * Initialize the clustering system
   */
  async initialize(): Promise<void> {
    logger.info('Initializing ErrorNode Clustering system...');
    
    this.isInitialized = true;
    logger.info('ErrorNode Clustering system initialized successfully');
  }

  /**
   * Add error node to the system
   */
  async addErrorNode(errorNode: ErrorNode): Promise<void> {
    if (!this.isInitialized) {
      throw new Error('ErrorNode Clustering system not initialized');
    }

    this.errorNodes.set(errorNode.id, errorNode);
    logger.info({ errorNodeId: errorNode.id }, 'ErrorNode added to clustering system');

    // Trigger clustering if we have enough nodes
    if (this.errorNodes.size % 10 === 0) {
      await this.performClustering();
    }
  }

  /**
   * Perform clustering algorithm on error nodes
   */
  async performClustering(): Promise<ErrorCluster[]> {
    logger.info('Performing ErrorNode clustering...');

    const errorNodes = Array.from(this.errorNodes.values());
    if (errorNodes.length < this.config.minClusterSize) {
      logger.info('Not enough error nodes for clustering');
      return [];
    }

    // Extract features from error nodes
    const featureVectors = errorNodes.map(node => this.extractFeatures(node));

    // Perform K-means clustering
    const clusters = await this.kMeansClustering(errorNodes, featureVectors);

    // Update cluster storage
    for (const cluster of clusters) {
      this.clusters.set(cluster.clusterId, cluster);
    }

    logger.info({ clusterCount: clusters.length }, 'ErrorNode clustering completed');
    return clusters;
  }

  /**
   * Extract feature vector from error node
   */
  private extractFeatures(errorNode: ErrorNode): ErrorFeatureVector {
    // Error type vector (one-hot encoding of common error types)
    const errorTypes = ['TypeError', 'ReferenceError', 'SyntaxError', 'RangeError', 'NetworkError', 'ValidationError'];
    const errorTypeVector = errorTypes.map(type => 
      errorNode.errorType.includes(type) ? 1 : 0
    );

    // Context vector (simplified representation of context keys)
    const contextKeys = Object.keys(errorNode.context);
    const contextVector = [
      contextKeys.length,
      contextKeys.includes('function') ? 1 : 0,
      contextKeys.includes('line') ? 1 : 0,
      contextKeys.includes('file') ? 1 : 0,
      contextKeys.includes('variable') ? 1 : 0
    ];

    // Severity score
    const severityScores = { low: 0.25, medium: 0.5, high: 0.75, critical: 1.0 };
    const severityScore = severityScores[errorNode.severity];

    // Tag vector (presence of common tags)
    const commonTags = ['frontend', 'backend', 'database', 'api', 'ui', 'performance', 'security'];
    const tagVector = commonTags.map(tag => 
      errorNode.tags.includes(tag) ? 1 : 0
    );

    // Semantic vector (simplified text analysis of error message)
    const semanticVector = this.extractSemanticFeatures(errorNode.errorMessage);

    return {
      errorTypeVector,
      contextVector,
      severityScore,
      tagVector,
      semanticVector
    };
  }

  /**
   * Extract semantic features from error message
   */
  private extractSemanticFeatures(errorMessage: string): number[] {
    const keywords = ['undefined', 'null', 'function', 'object', 'array', 'string', 'number', 'boolean'];
    const words = errorMessage.toLowerCase().split(/\s+/);
    
    return keywords.map(keyword => {
      const count = words.filter(word => word.includes(keyword)).length;
      return Math.min(count / words.length, 1.0); // Normalize
    });
  }

  /**
   * Calculate similarity between two feature vectors
   */
  private calculateSimilarity(vector1: ErrorFeatureVector, vector2: ErrorFeatureVector): number {
    const weights = this.config.featureWeights;
    
    // Cosine similarity for each feature type
    const errorTypeSim = this.cosineSimilarity(vector1.errorTypeVector, vector2.errorTypeVector);
    const contextSim = this.cosineSimilarity(vector1.contextVector, vector2.contextVector);
    const severitySim = 1 - Math.abs(vector1.severityScore - vector2.severityScore);
    const tagSim = this.cosineSimilarity(vector1.tagVector, vector2.tagVector);
    const semanticSim = this.cosineSimilarity(vector1.semanticVector, vector2.semanticVector);

    // Weighted combination
    return (
      errorTypeSim * weights.errorType +
      contextSim * weights.context +
      severitySim * weights.severity +
      tagSim * weights.tags +
      semanticSim * weights.semantic
    );
  }

  /**
   * Calculate cosine similarity between two vectors
   */
  private cosineSimilarity(vector1: number[], vector2: number[]): number {
    const minLength = Math.min(vector1.length, vector2.length);
    let dotProduct = 0;
    let norm1 = 0;
    let norm2 = 0;

    for (let i = 0; i < minLength; i++) {
      dotProduct += vector1[i] * vector2[i];
      norm1 += vector1[i] * vector1[i];
      norm2 += vector2[i] * vector2[i];
    }

    if (norm1 === 0 || norm2 === 0) {
      return 0;
    }

    return dotProduct / (Math.sqrt(norm1) * Math.sqrt(norm2));
  }

  /**
   * K-means clustering implementation
   */
  private async kMeansClustering(errorNodes: ErrorNode[], featureVectors: ErrorFeatureVector[]): Promise<ErrorCluster[]> {
    const k = Math.min(this.config.maxClustersPerRun, Math.floor(errorNodes.length / this.config.minClusterSize));
    
    if (k <= 0) {
      return [];
    }

    // Initialize centroids randomly
    let centroids = this.initializeCentroids(featureVectors, k);
    let clusters: ErrorCluster[] = [];
    let iterations = 0;
    const maxIterations = 100;

    while (iterations < maxIterations) {
      // Assign nodes to clusters
      const assignments = errorNodes.map((_node, index) => {
        const vector = featureVectors[index];
        let bestCluster = 0;
        let bestSimilarity = this.calculateSimilarity(vector, centroids[0]);

        for (let i = 1; i < centroids.length; i++) {
          const similarity = this.calculateSimilarity(vector, centroids[i]);
          if (similarity > bestSimilarity) {
            bestSimilarity = similarity;
            bestCluster = i;
          }
        }

        return { clusterIndex: bestCluster, similarity: bestSimilarity };
      });

      // Create clusters
      clusters = [];
      for (let i = 0; i < k; i++) {
        const clusterNodes = errorNodes.filter((_, nodeIndex) => assignments[nodeIndex].clusterIndex === i);
        const clusterVectors = featureVectors.filter((_, vectorIndex) => assignments[vectorIndex].clusterIndex === i);

        if (clusterNodes.length >= this.config.minClusterSize) {
          const clusterId = `cluster_${Date.now()}_${i}`;
          const totalBounty = clusterNodes.reduce((sum, node) => sum + node.bountyAmount, 0);
          const avgSimilarity = assignments
            .filter(a => a.clusterIndex === i)
            .reduce((sum, a) => sum + a.similarity, 0) / clusterNodes.length;

          clusters.push({
            clusterId,
            clusterName: `Error Cluster ${i + 1}`,
            description: this.generateClusterDescription(clusterNodes),
            errorNodes: clusterNodes,
            centroid: this.calculateCentroid(clusterVectors),
            similarity: avgSimilarity,
            totalBounty,
            assignedAgents: [],
            solutions: [],
            createdAt: new Date(),
            lastUpdated: new Date()
          });
        }
      }

      // Update centroids
      const newCentroids = clusters.map(cluster => cluster.centroid);
      
      // Check for convergence
      if (this.centroidsConverged(centroids, newCentroids)) {
        break;
      }

      centroids = newCentroids;
      iterations++;
    }

    return clusters.filter(cluster => cluster.errorNodes.length >= this.config.minClusterSize);
  }

  /**
   * Initialize centroids randomly
   */
  private initializeCentroids(featureVectors: ErrorFeatureVector[], k: number): ErrorFeatureVector[] {
    const centroids: ErrorFeatureVector[] = [];
    
    for (let i = 0; i < k; i++) {
      const randomIndex = Math.floor(Math.random() * featureVectors.length);
      centroids.push({ ...featureVectors[randomIndex] });
    }

    return centroids;
  }

  /**
   * Calculate centroid from feature vectors
   */
  private calculateCentroid(vectors: ErrorFeatureVector[]): ErrorFeatureVector {
    if (vectors.length === 0) {
      throw new Error('Cannot calculate centroid of empty vector set');
    }

    const centroid: ErrorFeatureVector = {
      errorTypeVector: new Array(vectors[0].errorTypeVector.length).fill(0),
      contextVector: new Array(vectors[0].contextVector.length).fill(0),
      severityScore: 0,
      tagVector: new Array(vectors[0].tagVector.length).fill(0),
      semanticVector: new Array(vectors[0].semanticVector.length).fill(0)
    };

    // Sum all vectors
    for (const vector of vectors) {
      for (let i = 0; i < centroid.errorTypeVector.length; i++) {
        centroid.errorTypeVector[i] += vector.errorTypeVector[i];
      }
      for (let i = 0; i < centroid.contextVector.length; i++) {
        centroid.contextVector[i] += vector.contextVector[i];
      }
      centroid.severityScore += vector.severityScore;
      for (let i = 0; i < centroid.tagVector.length; i++) {
        centroid.tagVector[i] += vector.tagVector[i];
      }
      for (let i = 0; i < centroid.semanticVector.length; i++) {
        centroid.semanticVector[i] += vector.semanticVector[i];
      }
    }

    // Average
    const count = vectors.length;
    centroid.errorTypeVector = centroid.errorTypeVector.map(v => v / count);
    centroid.contextVector = centroid.contextVector.map(v => v / count);
    centroid.severityScore = centroid.severityScore / count;
    centroid.tagVector = centroid.tagVector.map(v => v / count);
    centroid.semanticVector = centroid.semanticVector.map(v => v / count);

    return centroid;
  }

  /**
   * Check if centroids have converged
   */
  private centroidsConverged(oldCentroids: ErrorFeatureVector[], newCentroids: ErrorFeatureVector[]): boolean {
    if (oldCentroids.length !== newCentroids.length) {
      return false;
    }

    const threshold = 0.001;
    for (let i = 0; i < oldCentroids.length; i++) {
      const similarity = this.calculateSimilarity(oldCentroids[i], newCentroids[i]);
      if (similarity < 1 - threshold) {
        return false;
      }
    }

    return true;
  }

  /**
   * Generate cluster description
   */
  private generateClusterDescription(errorNodes: ErrorNode[]): string {
    const errorTypes = [...new Set(errorNodes.map(node => node.errorType))];
    const commonTags = [...new Set(errorNodes.flatMap(node => node.tags))];
    
    return `Cluster of ${errorNodes.length} errors: ${errorTypes.slice(0, 3).join(', ')}. Tags: ${commonTags.slice(0, 5).join(', ')}`;
  }

  /**
   * Get all clusters
   */
  getClusters(): ErrorCluster[] {
    return Array.from(this.clusters.values());
  }

  /**
   * Get cluster by ID
   */
  getCluster(clusterId: string): ErrorCluster | undefined {
    return this.clusters.get(clusterId);
  }

  /**
   * Get error nodes
   */
  getErrorNodes(): ErrorNode[] {
    return Array.from(this.errorNodes.values());
  }

  isReady(): boolean {
    return this.isInitialized;
  }
}
