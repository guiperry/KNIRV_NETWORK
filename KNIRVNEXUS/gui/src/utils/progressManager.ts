// Progress Manager for Long-Running Operations
// Provides centralized progress tracking with real-time updates

export interface ProgressState {
  id: string;
  type: 'agent_build' | 'agent_inference' | 'mcp_install' | 'workflow_execution' | 'file_upload';
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  progress: number; // 0-100
  message: string;
  details?: string;
  startTime: Date;
  endTime?: Date;
  estimatedDuration?: number; // in milliseconds
  metadata?: Record<string, any>;
}

export interface ProgressUpdate {
  id: string;
  progress?: number;
  message?: string;
  details?: string;
  status?: ProgressState['status'];
  metadata?: Record<string, any>;
}

export type ProgressCallback = (state: ProgressState) => void;

class ProgressManager {
  private operations: Map<string, ProgressState> = new Map();
  private callbacks: Map<string, Set<ProgressCallback>> = new Map();
  private globalCallbacks: Set<ProgressCallback> = new Set();

  // Start a new long-running operation
  startOperation(
    id: string,
    type: ProgressState['type'],
    message: string,
    estimatedDuration?: number
  ): ProgressState {
    const state: ProgressState = {
      id,
      type,
      status: 'pending',
      progress: 0,
      message,
      startTime: new Date(),
      estimatedDuration,
    };

    this.operations.set(id, state);
    this.notifyCallbacks(id, state);
    return state;
  }

  // Update an existing operation
  updateOperation(id: string, update: ProgressUpdate): ProgressState | null {
    const state = this.operations.get(id);
    if (!state) {
      console.warn(`Progress operation not found: ${id}`);
      return null;
    }

    // Update state
    if (update.progress !== undefined) state.progress = Math.max(0, Math.min(100, update.progress));
    if (update.message !== undefined) state.message = update.message;
    if (update.details !== undefined) state.details = update.details;
    if (update.status !== undefined) {
      state.status = update.status;
      if (update.status === 'completed' || update.status === 'failed' || update.status === 'cancelled') {
        state.endTime = new Date();
      }
    }
    if (update.metadata !== undefined) {
      state.metadata = { ...state.metadata, ...update.metadata };
    }

    this.operations.set(id, state);
    this.notifyCallbacks(id, state);
    return state;
  }

  // Complete an operation
  completeOperation(id: string, message?: string): ProgressState | null {
    return this.updateOperation(id, {
      id,
      status: 'completed',
      progress: 100,
      message: message || 'Operation completed successfully',
    });
  }

  // Fail an operation
  failOperation(id: string, message?: string, details?: string): ProgressState | null {
    return this.updateOperation(id, {
      id,
      status: 'failed',
      message: message || 'Operation failed',
      details,
    });
  }

  // Cancel an operation
  cancelOperation(id: string, message?: string): ProgressState | null {
    return this.updateOperation(id, {
      id,
      status: 'cancelled',
      message: message || 'Operation cancelled',
    });
  }

  // Get operation state
  getOperation(id: string): ProgressState | null {
    return this.operations.get(id) || null;
  }

  // Get all operations
  getAllOperations(): ProgressState[] {
    return Array.from(this.operations.values());
  }

  // Get operations by type
  getOperationsByType(type: ProgressState['type']): ProgressState[] {
    return Array.from(this.operations.values()).filter(op => op.type === type);
  }

  // Get active operations
  getActiveOperations(): ProgressState[] {
    return Array.from(this.operations.values()).filter(
      op => op.status === 'pending' || op.status === 'running'
    );
  }

  // Subscribe to operation updates
  subscribe(id: string, callback: ProgressCallback): () => void {
    if (!this.callbacks.has(id)) {
      this.callbacks.set(id, new Set());
    }
    this.callbacks.get(id)!.add(callback);

    // Return unsubscribe function
    return () => {
      const callbacks = this.callbacks.get(id);
      if (callbacks) {
        callbacks.delete(callback);
        if (callbacks.size === 0) {
          this.callbacks.delete(id);
        }
      }
    };
  }

  // Subscribe to all operation updates
  subscribeGlobal(callback: ProgressCallback): () => void {
    this.globalCallbacks.add(callback);
    return () => {
      this.globalCallbacks.delete(callback);
    };
  }

  // Clean up completed operations older than specified time
  cleanup(maxAge: number = 5 * 60 * 1000): void { // Default 5 minutes
    const now = new Date();
    const toDelete: string[] = [];

    for (const [id, state] of this.operations) {
      if (state.endTime && (now.getTime() - state.endTime.getTime()) > maxAge) {
        toDelete.push(id);
      }
    }

    toDelete.forEach(id => {
      this.operations.delete(id);
      this.callbacks.delete(id);
    });
  }

  // Calculate estimated time remaining
  getEstimatedTimeRemaining(id: string): number | null {
    const state = this.operations.get(id);
    if (!state || state.status !== 'running' || !state.estimatedDuration) {
      return null;
    }

    const elapsed = new Date().getTime() - state.startTime.getTime();
    const progressRatio = state.progress / 100;
    
    if (progressRatio <= 0) {
      return state.estimatedDuration;
    }

    const estimatedTotal = elapsed / progressRatio;
    return Math.max(0, estimatedTotal - elapsed);
  }

  // Get operation duration
  getDuration(id: string): number | null {
    const state = this.operations.get(id);
    if (!state) return null;

    const endTime = state.endTime || new Date();
    return endTime.getTime() - state.startTime.getTime();
  }

  private notifyCallbacks(id: string, state: ProgressState): void {
    // Notify operation-specific callbacks
    const callbacks = this.callbacks.get(id);
    if (callbacks) {
      callbacks.forEach(callback => {
        try {
          callback(state);
        } catch (error) {
          console.error('Error in progress callback:', error);
        }
      });
    }

    // Notify global callbacks
    this.globalCallbacks.forEach(callback => {
      try {
        callback(state);
      } catch (error) {
        console.error('Error in global progress callback:', error);
      }
    });
  }
}

// Global progress manager instance
export const progressManager = new ProgressManager();

// Convenience functions for common operations
export const startAgentBuild = (agentId: string, message: string = 'Building agent...') => {
  return progressManager.startOperation(`agent_build_${agentId}`, 'agent_build', message, 30000);
};

export const startAgentInference = (sessionId: string, message: string = 'Running inference...') => {
  return progressManager.startOperation(`agent_inference_${sessionId}`, 'agent_inference', message, 10000);
};

export const startMCPInstall = (serverId: string, message: string = 'Installing MCP server...') => {
  return progressManager.startOperation(`mcp_install_${serverId}`, 'mcp_install', message, 60000);
};

export const startWorkflowExecution = (workflowId: string, message: string = 'Executing workflow...') => {
  return progressManager.startOperation(`workflow_${workflowId}`, 'workflow_execution', message, 20000);
};

// Auto-cleanup every 5 minutes
setInterval(() => {
  progressManager.cleanup();
}, 5 * 60 * 1000);
