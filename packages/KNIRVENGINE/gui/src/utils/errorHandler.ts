// Enhanced Error Handling System
// Provides centralized error management with categorization and recovery strategies

declare global {
  interface Window {
    electronAPI?: {
      isElectron: boolean;
    };
  }
}

// Utility function to get the correct API base URL
function getApiBaseUrl(): string {
  // Check if we're running in Electron
  if (typeof window !== 'undefined' && window.electronAPI?.isElectron) {
    return 'http://localhost:8081';
  }
  // In web mode, use relative paths (handled by Vite proxy)
  return '';
}

export interface ErrorInfo {
  id: string;
  type: 'network' | 'validation' | 'authentication' | 'authorization' | 'server' | 'client' | 'unknown';
  severity: 'low' | 'medium' | 'high' | 'critical';
  message: string;
  details?: string;
  code?: string | number;
  timestamp: Date;
  context?: Record<string, unknown>;
  recoverable: boolean;
  retryable: boolean;
  userMessage: string;
  technicalMessage: string;
  // Enhanced fields for LLM inference
  stackTrace?: string;
  systemInfo?: {
    userAgent?: string;
    url?: string;
    timestamp?: string;
    userId?: string;
    sessionId?: string;
    buildVersion?: string;
  };
  inferenceRequested?: boolean;
  inferenceResult?: {
    analysis: string;
    suggestedFixes: string[];
    confidence: number;
    category: string;
    estimatedResolutionTime?: string;
    requiresUserAction: boolean;
    automatedFixAvailable: boolean;
  };
}

// Define a type for possible error inputs
export type ErrorSource = 
  | Error 
  | { status?: number; response?: { status?: number; data?: { message?: string; error?: string; details?: string } } }
  | { name?: string; message?: string; type?: string; stack?: string }
  | string
  | unknown;

// Type guard functions to check error types
function isErrorWithResponse(error: unknown): error is { response?: { status?: number; data?: unknown } } {
  return typeof error === 'object' && error !== null && 'response' in error;
}

function isErrorWithStatus(error: unknown): error is { status?: number } {
  return typeof error === 'object' && error !== null && 'status' in error;
}

function isErrorWithName(error: unknown): error is { name?: string; message?: string; type?: string } {
  return typeof error === 'object' && error !== null && ('name' in error || 'type' in error);
}

export interface ErrorRecoveryStrategy {
  type: 'retry' | 'fallback' | 'redirect' | 'refresh' | 'manual';
  action?: () => Promise<void> | void;
  message?: string;
  delay?: number;
}

export type ErrorCallback = (error: ErrorInfo) => void;
export type InferenceCallback = (error: ErrorInfo) => void;

class ErrorHandler {
  private errors: Map<string, ErrorInfo> = new Map();
  private callbacks: Set<ErrorCallback> = new Set();
  private inferenceCallbacks: Set<InferenceCallback> = new Set();
  private retryAttempts: Map<string, number> = new Map();
  private maxRetries = 3;
  private inferenceEnabled = true;
  private autoInferenceThreshold: ErrorInfo['severity'][] = ['high', 'critical'];

  // Process and categorize an error
  handleError(error: ErrorSource, context?: Record<string, unknown>): ErrorInfo {
    const errorInfo = this.categorizeError(error, context);

    // Add system information
    errorInfo.systemInfo = this.collectSystemInfo();

    // Add stack trace if available
    if (error instanceof Error && error.stack) {
      errorInfo.stackTrace = error.stack;
    }

    // Store error
    this.errors.set(errorInfo.id, errorInfo);

    // Notify callbacks
    this.notifyCallbacks(errorInfo);

    // Log error
    this.logError(errorInfo);

    // Trigger automatic inference for critical errors
    if (this.inferenceEnabled && this.autoInferenceThreshold.includes(errorInfo.severity)) {
      this.requestInference(errorInfo.id);
    }

    return errorInfo;
  }

  // Categorize error based on type and content
  private categorizeError(error: ErrorSource, context?: Record<string, unknown>): ErrorInfo {
    const id = `error_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    const timestamp = new Date();
    
    let type: ErrorInfo['type'] = 'unknown';
    let severity: ErrorInfo['severity'] = 'medium';
    let message = 'An unexpected error occurred';
    let details = '';
    let code: string | number | undefined;
    let recoverable = false;
    let retryable = false;
    let userMessage = 'Something went wrong. Please try again.';
    let technicalMessage = '';
    
    // Set technical message based on error type
    if (error instanceof Error) {
      technicalMessage = error.message;
    } else if (typeof error === 'string') {
      technicalMessage = error;
    } else if (error !== null && typeof error === 'object') {
      technicalMessage = 'message' in error && typeof error.message === 'string' 
        ? error.message 
        : String(error);
    } else {
      technicalMessage = String(error);
    }

    // Network errors
    if (error instanceof TypeError && error.message.includes('fetch')) {
      type = 'network';
      severity = 'high';
      message = 'Network connection failed';
      userMessage = 'Unable to connect to the server. Please check your internet connection.';
      recoverable = true;
      retryable = true;
    }
    // HTTP errors
    else if (isErrorWithStatus(error) || (isErrorWithResponse(error) && error.response?.status)) {
      const status = isErrorWithStatus(error) ? error.status : 
                    isErrorWithResponse(error) && error.response ? error.response.status : undefined;
      
      if (status) {
        code = status;
        
        if (status >= 400 && status < 500) {
          // Client errors
          type = 'client';
          severity = status === 401 || status === 403 ? 'high' : 'medium';
          
          switch (status) {
            case 400:
              message = 'Invalid request';
              userMessage = 'The request was invalid. Please check your input and try again.';
              break;
            case 401:
              type = 'authentication';
              message = 'Authentication required';
              userMessage = 'Please log in to continue.';
              recoverable = true;
              break;
            case 403:
              type = 'authorization';
              message = 'Access denied';
              userMessage = 'You do not have permission to perform this action.';
              break;
            case 404:
              message = 'Resource not found';
              userMessage = 'The requested resource was not found.';
              break;
            case 422:
              type = 'validation';
              message = 'Validation failed';
              userMessage = 'Please check your input and try again.';
              recoverable = true;
              break;
            case 429:
              message = 'Rate limit exceeded';
              userMessage = 'Too many requests. Please wait a moment and try again.';
              retryable = true;
              recoverable = true;
              break;
            default:
              message = `Client error (${status})`;
              userMessage = 'There was a problem with your request. Please try again.';
          }
        } else if (status >= 500) {
          // Server errors
          type = 'server';
          severity = 'high';
          message = `Server error (${status})`;
          userMessage = 'The server encountered an error. Please try again later.';
          retryable = true;
          recoverable = true;
        }
      }
    }
    // Validation errors
    else if (isErrorWithName(error) && (error.name === 'ValidationError' || error.type === 'validation')) {
      type = 'validation';
      severity = 'medium';
      message = 'Validation error';
      userMessage = 'Please check your input and try again.';
      recoverable = true;
    }
    // JavaScript errors
    else if (error instanceof Error) {
      type = 'client';
      severity = 'medium';
      message = error.message;
      technicalMessage = error.stack || error.message;
      
      if (error.name === 'TypeError') {
        userMessage = 'A technical error occurred. Please refresh the page and try again.';
      } else if (error.name === 'ReferenceError') {
        severity = 'high';
        userMessage = 'A technical error occurred. Please refresh the page and try again.';
      }
    }

    // Extract additional details from error response
    if (isErrorWithResponse(error) && error.response?.data) {
      const data = error.response.data as Record<string, unknown>;
      if ('message' in data && typeof data.message === 'string') details = data.message;
      if ('error' in data && typeof data.error === 'string') details = data.error;
      if ('details' in data && typeof data.details === 'string') details = data.details;
    }

    return {
      id,
      type,
      severity,
      message,
      details,
      code,
      timestamp,
      context,
      recoverable,
      retryable,
      userMessage,
      technicalMessage,
    };
  }

  // Get recovery strategy for an error
  getRecoveryStrategy(errorInfo: ErrorInfo): ErrorRecoveryStrategy | null {
    if (!errorInfo.recoverable) {
      return null;
    }

    switch (errorInfo.type) {
      case 'network':
        return {
          type: 'retry',
          message: 'Retry connection',
          delay: 2000,
        };
      
      case 'authentication':
        return {
          type: 'redirect',
          message: 'Please log in',
          action: () => {
            // Redirect to login
            window.location.href = '/login';
          },
        };
      
      case 'server':
        if (errorInfo.retryable) {
          const attempts = this.retryAttempts.get(errorInfo.id) || 0;
          if (attempts < this.maxRetries) {
            return {
              type: 'retry',
              message: `Retry (${attempts + 1}/${this.maxRetries})`,
              delay: Math.pow(2, attempts) * 1000, // Exponential backoff
            };
          }
        }
        return {
          type: 'refresh',
          message: 'Refresh page',
          action: () => window.location.reload(),
        };
      
      case 'validation':
        return {
          type: 'manual',
          message: 'Please correct the errors and try again',
        };
      
      default:
        if (errorInfo.retryable) {
          return {
            type: 'retry',
            message: 'Try again',
            delay: 1000,
          };
        }
        return {
          type: 'refresh',
          message: 'Refresh page',
          action: () => window.location.reload(),
        };
    }
  }

  // Attempt to recover from an error
  async recover(errorId: string, strategy?: ErrorRecoveryStrategy): Promise<boolean> {
    const errorInfo = this.errors.get(errorId);
    if (!errorInfo) {
      return false;
    }

    const recoveryStrategy = strategy || this.getRecoveryStrategy(errorInfo);
    if (!recoveryStrategy) {
      return false;
    }

    try {
      if (recoveryStrategy.type === 'retry') {
        // Increment retry count
        const attempts = this.retryAttempts.get(errorId) || 0;
        this.retryAttempts.set(errorId, attempts + 1);
        
        // Wait for delay if specified
        if (recoveryStrategy.delay) {
          await new Promise(resolve => setTimeout(resolve, recoveryStrategy.delay));
        }
      }

      // Execute recovery action
      if (recoveryStrategy.action) {
        await recoveryStrategy.action();
      }

      return true;
    } catch (recoveryError) {
      console.error('Error recovery failed:', recoveryError);
      return false;
    }
  }

  // Subscribe to error notifications
  subscribe(callback: ErrorCallback): () => void {
    this.callbacks.add(callback);
    return () => {
      this.callbacks.delete(callback);
    };
  }

  // Get all errors
  getAllErrors(): ErrorInfo[] {
    return Array.from(this.errors.values()).sort(
      (a, b) => b.timestamp.getTime() - a.timestamp.getTime()
    );
  }

  // Get errors by type
  getErrorsByType(type: ErrorInfo['type']): ErrorInfo[] {
    return this.getAllErrors().filter(error => error.type === type);
  }

  // Get errors by severity
  getErrorsBySeverity(severity: ErrorInfo['severity']): ErrorInfo[] {
    return this.getAllErrors().filter(error => error.severity === severity);
  }

  // Clear an error
  clearError(errorId: string): void {
    this.errors.delete(errorId);
    this.retryAttempts.delete(errorId);
  }

  // Clear all errors
  clearAllErrors(): void {
    this.errors.clear();
    this.retryAttempts.clear();
  }

  // Clear old errors
  clearOldErrors(maxAge: number = 5 * 60 * 1000): void {
    const now = new Date();
    const toDelete: string[] = [];

    for (const [id, error] of this.errors) {
      if (now.getTime() - error.timestamp.getTime() > maxAge) {
        toDelete.push(id);
      }
    }

    toDelete.forEach(id => {
      this.errors.delete(id);
      this.retryAttempts.delete(id);
    });
  }

  // Request LLM inference for error analysis
  async requestInference(errorId: string): Promise<void> {
    const errorInfo = this.errors.get(errorId);
    if (!errorInfo || errorInfo.inferenceRequested) {
      return;
    }

    // Mark as inference requested
    errorInfo.inferenceRequested = true;
    this.errors.set(errorId, errorInfo);

    try {
      console.log('Requesting inference for error:', errorId);
      const inferenceResult = await this.performInference(errorInfo);
      errorInfo.inferenceResult = inferenceResult;
      this.errors.set(errorId, errorInfo);

      // Notify inference callbacks
      this.notifyInferenceCallbacks(errorInfo);
      console.log('Inference completed successfully for error:', errorId);
    } catch (inferenceError) {
      console.error('Error inference failed:', inferenceError);
      errorInfo.inferenceRequested = false;
      this.errors.set(errorId, errorInfo);
    }
  }

  // Perform LLM inference to analyze error and suggest fixes
  private async performInference(errorInfo: ErrorInfo): Promise<ErrorInfo['inferenceResult']> {
    const prompt = this.buildInferencePrompt(errorInfo);

    try {
      console.log('Calling inference API with embedded keys...');
      
      // Extract symptoms from error message and details for better analysis
      const symptoms = [
        errorInfo.message,
        errorInfo.details,
        errorInfo.technicalMessage
      ].filter(Boolean);
      
      // Call the enhanced inference API with troubleshooting knowledge
      const apiBaseUrl = getApiBaseUrl();
      const response = await fetch(`${apiBaseUrl}/api/v1/inference/analyze-error`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          prompt,
          error_context: {
            type: errorInfo.type,
            severity: errorInfo.severity,
            code: errorInfo.code,
            message: errorInfo.message,
            details: errorInfo.details,
            stackTrace: errorInfo.stackTrace,
            systemInfo: {
              ...errorInfo.systemInfo,
              // Add additional context that might help with troubleshooting
              url: window.location.pathname,
              userAgent: navigator.userAgent,
              timestamp: new Date().toISOString(),
              buildVersion: import.meta.env.VITE_APP_VERSION || 'unknown',
            },
            context: errorInfo.context,
            symptoms: symptoms
          },
        }),
      });

      if (!response.ok) {
        throw new Error(`Inference API failed: ${response.statusText}`);
      }

      const result = await response.json();
      console.log('Inference API response:', result);

      return {
        analysis: result.analysis || 'Unable to analyze error',
        suggestedFixes: result.suggested_fixes || [],
        confidence: result.confidence || 0.5,
        category: result.category || 'unknown',
        estimatedResolutionTime: result.estimated_resolution_time,
        requiresUserAction: result.requires_user_action || true,
        automatedFixAvailable: result.automated_fix_available || false,
      };
    } catch (apiError) {
      console.error('Inference API error:', apiError);

      // Fallback to rule-based analysis
      return this.performRuleBasedAnalysis(errorInfo);
    }
  }

  // Build inference prompt for LLM analysis
  private buildInferencePrompt(errorInfo: ErrorInfo): string {
    return `
You are an expert system administrator and developer assistant for the KNIRVENGINE application. Analyze the following error and provide actionable solutions.

ERROR DETAILS:
- Type: ${errorInfo.type}
- Severity: ${errorInfo.severity}
- Message: ${errorInfo.message}
- Details: ${errorInfo.details || 'None'}
- Code: ${errorInfo.code || 'None'}
- Technical Message: ${errorInfo.technicalMessage}

SYSTEM CONTEXT:
- URL: ${errorInfo.systemInfo?.url || 'Unknown'}
- User Agent: ${errorInfo.systemInfo?.userAgent || 'Unknown'}
- Build Version: ${errorInfo.systemInfo?.buildVersion || 'Unknown'}
- Timestamp: ${errorInfo.timestamp.toISOString()}

STACK TRACE:
${errorInfo.stackTrace || 'Not available'}

ADDITIONAL CONTEXT:
${JSON.stringify(errorInfo.context || {}, null, 2)}

INSTRUCTIONS:
1. The server will provide you with RELEVANT TROUBLESHOOTING INFORMATION from our knowledge base
2. Use this information to identify if this is a known issue with documented solutions
3. If it matches a known issue, adapt the troubleshooting steps to this specific error
4. If it doesn't match any known issue, provide your best analysis based on the error details
5. Be specific and actionable in your recommendations

Please provide:
1. A clear analysis of what went wrong
2. 3-5 specific, actionable steps to fix this error
3. Your confidence level (0-1) in the solution
4. Whether this requires user action or can be automated
5. Estimated time to resolve
6. Error category classification

Format your response as JSON with the following structure:
{
  "analysis": "Clear explanation of the error",
  "suggested_fixes": ["Step 1", "Step 2", "Step 3"],
  "confidence": 0.8,
  "category": "configuration|code|network|permissions|data",
  "estimated_resolution_time": "5 minutes",
  "requires_user_action": true,
  "automated_fix_available": false
}
`;
  }

  // Fallback rule-based analysis when LLM is unavailable
  private performRuleBasedAnalysis(errorInfo: ErrorInfo): ErrorInfo['inferenceResult'] {
    const fixes: string[] = [];
    let analysis = 'Error analysis based on known patterns.';
    let confidence = 0.6;
    let category = 'unknown';
    const requiresUserAction = true;
    const automatedFixAvailable = false;

    switch (errorInfo.type) {
      case 'network':
        analysis = 'Network connectivity issue detected.';
        fixes.push('Check your internet connection');
        fixes.push('Verify the server is accessible');
        fixes.push('Try refreshing the page');
        fixes.push('Check for firewall or proxy issues');
        category = 'network';
        confidence = 0.8;
        break;

      case 'authentication':
        analysis = 'Authentication failure - credentials may be invalid or expired.';
        fixes.push('Log out and log back in');
        fixes.push('Clear browser cookies and cache');
        fixes.push('Check if your session has expired');
        fixes.push('Verify your credentials are correct');
        category = 'permissions';
        confidence = 0.9;
        break;

      case 'validation':
        analysis = 'Input validation failed - data format or content is invalid.';
        fixes.push('Check all required fields are filled');
        fixes.push('Verify data formats (dates, emails, etc.)');
        fixes.push('Remove any special characters that might be invalid');
        fixes.push('Check field length limits');
        category = 'data';
        confidence = 0.7;
        break;

      case 'server':
        analysis = 'Server-side error occurred during request processing.';
        fixes.push('Wait a few minutes and try again');
        fixes.push('Check server status page if available');
        fixes.push('Contact support if the issue persists');
        fixes.push('Try using a different browser or device');
        category = 'configuration';
        confidence = 0.6;
        break;

      default:
        analysis = 'Unknown error type - general troubleshooting recommended.';
        fixes.push('Refresh the page and try again');
        fixes.push('Clear browser cache and cookies');
        fixes.push('Try using a different browser');
        fixes.push('Contact support with error details');
        confidence = 0.4;
    }

    return {
      analysis,
      suggestedFixes: fixes,
      confidence,
      category,
      estimatedResolutionTime: '5-10 minutes',
      requiresUserAction,
      automatedFixAvailable,
    };
  }

  // Collect system information for error context
  private collectSystemInfo(): ErrorInfo['systemInfo'] {
    return {
      userAgent: navigator.userAgent,
      url: window.location.href,
      timestamp: new Date().toISOString(),
      userId: localStorage.getItem('userId') || 'anonymous',
      sessionId: localStorage.getItem('sessionId') || 'unknown',
      buildVersion: import.meta.env.VITE_APP_VERSION || 'unknown',
    };
  }

  // Subscribe to inference notifications
  subscribeToInference(callback: InferenceCallback): () => void {
    this.inferenceCallbacks.add(callback);
    return () => {
      this.inferenceCallbacks.delete(callback);
    };
  }

  // Get errors that have inference results
  getErrorsWithInference(): ErrorInfo[] {
    return this.getAllErrors().filter(error => error.inferenceResult);
  }

  // Get errors that need inference
  getErrorsNeedingInference(): ErrorInfo[] {
    return this.getAllErrors().filter(
      error => !error.inferenceRequested &&
      (error.severity === 'high' || error.severity === 'critical')
    );
  }

  // Enable/disable automatic inference
  setInferenceEnabled(enabled: boolean): void {
    this.inferenceEnabled = enabled;
  }

  // Set severity threshold for automatic inference
  setAutoInferenceThreshold(severities: ErrorInfo['severity'][]): void {
    this.autoInferenceThreshold = severities;
  }

  private notifyInferenceCallbacks(error: ErrorInfo): void {
    this.inferenceCallbacks.forEach(callback => {
      try {
        callback(error);
      } catch (callbackError) {
        console.error('Error in inference callback:', callbackError);
      }
    });
  }

  private notifyCallbacks(error: ErrorInfo): void {
    this.callbacks.forEach(callback => {
      try {
        callback(error);
      } catch (callbackError) {
        console.error('Error in error callback:', callbackError);
      }
    });
  }

  private logError(error: ErrorInfo): void {
    const logLevel = error.severity === 'critical' || error.severity === 'high' ? 'error' : 'warn';
    console[logLevel]('Error handled:', {
      id: error.id,
      type: error.type,
      severity: error.severity,
      message: error.message,
      details: error.details,
      code: error.code,
      context: error.context,
      technicalMessage: error.technicalMessage,
    });
  }
}

// Global error handler instance
export const errorHandler = new ErrorHandler();

// Convenience functions
export const handleApiError = (error: ErrorSource, context?: Record<string, unknown>) => {
  return errorHandler.handleError(error, context);
};

export const recoverFromError = (errorId: string, strategy?: ErrorRecoveryStrategy) => {
  return errorHandler.recover(errorId, strategy);
};

export const requestErrorAnalysis = (errorId: string) => {
  return errorHandler.requestInference(errorId);
};

export const subscribeToErrors = (callback: ErrorCallback) => {
  return errorHandler.subscribe(callback);
};

export const subscribeToInference = (callback: InferenceCallback) => {
  return errorHandler.subscribeToInference(callback);
};

export const getErrorsNeedingHelp = () => {
  return errorHandler.getErrorsNeedingInference();
};

export const getAllErrorsWithAnalysis = () => {
  return errorHandler.getErrorsWithInference();
};

export const enableIntelligentErrorAnalysis = (enabled: boolean = true) => {
  errorHandler.setInferenceEnabled(enabled);
};

export const setErrorAnalysisThreshold = (severities: ErrorInfo['severity'][]) => {
  errorHandler.setAutoInferenceThreshold(severities);
};

// Auto-cleanup old errors every 5 minutes
setInterval(() => {
  errorHandler.clearOldErrors();
}, 5 * 60 * 1000);
