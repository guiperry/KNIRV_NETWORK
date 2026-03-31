/**
 * Advanced Error Handler
 * Comprehensive error handling and recovery system for KNIRV Controller
 */

export enum ErrorSeverity {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical'
}

export enum ErrorCategory {
  NETWORK = 'network',
  VALIDATION = 'validation',
  AUTHENTICATION = 'authentication',
  AUTHORIZATION = 'authorization',
  BUSINESS_LOGIC = 'business_logic',
  SYSTEM = 'system',
  USER_INPUT = 'user_input',
  EXTERNAL_SERVICE = 'external_service'
}

export interface ErrorContext {
  userId?: string;
  sessionId?: string;
  component?: string;
  action?: string;
  timestamp: Date;
  userAgent?: string;
  url?: string;
  additionalData?: Record<string, unknown>;
}

export interface ErrorDetails {
  id: string;
  message: string;
  code?: string;
  severity: ErrorSeverity;
  category: ErrorCategory;
  context: ErrorContext;
  stack?: string;
  originalError?: Error;
  retryCount?: number;
  isRecoverable?: boolean;
}

export interface RecoveryStrategy {
  name: string;
  execute: (error: ErrorDetails) => Promise<boolean>;
  maxRetries: number;
  retryDelay: number;
}

export interface ErrorHandlerConfig {
  enableLogging: boolean;
  enableReporting: boolean;
  enableRecovery: boolean;
  maxRetries: number;
  retryDelay: number;
  reportingEndpoint?: string;
  logLevel: 'debug' | 'info' | 'warn' | 'error';
}

class ErrorHandler {
  private config: ErrorHandlerConfig;
  private errorHistory: ErrorDetails[] = [];
  private recoveryStrategies: Map<ErrorCategory, RecoveryStrategy[]> = new Map();
  private errorListeners: ((error: ErrorDetails) => void)[] = [];

  constructor(config: Partial<ErrorHandlerConfig> = {}) {
    this.config = {
      enableLogging: true,
      enableReporting: true,
      enableRecovery: true,
      maxRetries: 3,
      retryDelay: 1000,
      logLevel: 'error',
      ...config
    };

    this.initializeRecoveryStrategies();
    this.setupGlobalErrorHandlers();
  }

  /**
   * Initialize default recovery strategies
   */
  private initializeRecoveryStrategies(): void {
    // Network error recovery
    this.addRecoveryStrategy(ErrorCategory.NETWORK, {
      name: 'retry_with_backoff',
      execute: async (error: ErrorDetails) => {
        const delay = this.config.retryDelay * Math.pow(2, error.retryCount || 0);
        await this.delay(delay);
        return true; // Indicate retry should be attempted
      },
      maxRetries: 3,
      retryDelay: 1000
    });

    // Authentication error recovery
    this.addRecoveryStrategy(ErrorCategory.AUTHENTICATION, {
      name: 'refresh_token',
      execute: async (error: ErrorDetails) => {
        try {
          // Log the authentication error for debugging
          console.warn('Authentication error occurred:', error.message, error.context);

          // Attempt to refresh authentication token
          const response = await fetch('/api/auth/refresh', {
            method: 'POST',
            credentials: 'include',
            headers: {
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              errorContext: error.context,
              timestamp: error.context.timestamp
            })
          });
          return response.ok;
        } catch {
          return false;
        }
      },
      maxRetries: 1,
      retryDelay: 0
    });

    // External service error recovery
    this.addRecoveryStrategy(ErrorCategory.EXTERNAL_SERVICE, {
      name: 'fallback_service',
      execute: async (error: ErrorDetails) => {
        // Implement fallback to alternative service
        console.warn(`Falling back for external service error: ${error.message}`);
        return false; // No automatic retry, use fallback
      },
      maxRetries: 0,
      retryDelay: 0
    });
  }

  /**
   * Setup global error handlers
   */
  private setupGlobalErrorHandlers(): void {
    if (typeof window !== 'undefined') {
      // Handle unhandled promise rejections
      window.addEventListener('unhandledrejection', (event) => {
        this.handleError(event.reason, {
          component: 'global',
          action: 'unhandled_promise_rejection',
          timestamp: new Date(),
          url: window.location.href,
          userAgent: navigator.userAgent
        });
      });

      // Handle JavaScript errors
      window.addEventListener('error', (event) => {
        this.handleError(new Error(event.message), {
          component: 'global',
          action: 'javascript_error',
          timestamp: new Date(),
          url: window.location.href,
          userAgent: navigator.userAgent,
          additionalData: {
            filename: event.filename,
            lineno: event.lineno,
            colno: event.colno
          }
        });
      });
    }
  }

  /**
   * Handle an error with full processing pipeline
   */
  public async handleError(
    error: Error | string,
    context: Partial<ErrorContext> = {},
    category: ErrorCategory = ErrorCategory.SYSTEM,
    severity: ErrorSeverity = ErrorSeverity.MEDIUM
  ): Promise<ErrorDetails> {
    const errorDetails: ErrorDetails = {
      id: this.generateErrorId(),
      message: typeof error === 'string' ? error : error.message,
      severity,
      category,
      context: {
        timestamp: new Date(),
        ...context
      },
      stack: typeof error === 'object' ? error.stack : undefined,
      originalError: typeof error === 'object' ? error : undefined,
      retryCount: 0,
      isRecoverable: this.isRecoverable(category, severity)
    };

    // Add to error history
    this.errorHistory.push(errorDetails);

    // Log the error
    if (this.config.enableLogging) {
      this.logError(errorDetails);
    }

    // Notify listeners
    this.notifyListeners(errorDetails);

    // Attempt recovery if enabled and error is recoverable
    if (this.config.enableRecovery && errorDetails.isRecoverable) {
      await this.attemptRecovery(errorDetails);
    }

    // Report the error if enabled
    if (this.config.enableReporting) {
      await this.reportError(errorDetails);
    }

    return errorDetails;
  }

  /**
   * Determine if an error is recoverable
   */
  private isRecoverable(category: ErrorCategory, severity: ErrorSeverity): boolean {
    if (severity === ErrorSeverity.CRITICAL) {
      return false;
    }

    const recoverableCategories = [
      ErrorCategory.NETWORK,
      ErrorCategory.AUTHENTICATION,
      ErrorCategory.EXTERNAL_SERVICE
    ];

    return recoverableCategories.includes(category);
  }

  /**
   * Attempt error recovery
   */
  private async attemptRecovery(error: ErrorDetails): Promise<boolean> {
    const strategies = this.recoveryStrategies.get(error.category) || [];

    for (const strategy of strategies) {
      if ((error.retryCount || 0) >= strategy.maxRetries) {
        continue;
      }

      try {
        const shouldRetry = await strategy.execute(error);
        if (shouldRetry) {
          error.retryCount = (error.retryCount || 0) + 1;
          return true;
        }
      } catch (recoveryError) {
        console.warn(`Recovery strategy ${strategy.name} failed:`, recoveryError);
      }
    }

    return false;
  }

  /**
   * Log error based on severity and configuration
   */
  private logError(error: ErrorDetails): void {
    const logMessage = `[${error.severity.toUpperCase()}] ${error.category}: ${error.message}`;
    const logData = {
      id: error.id,
      context: error.context,
      stack: error.stack
    };

    switch (error.severity) {
      case ErrorSeverity.CRITICAL:
        console.error(logMessage, logData);
        break;
      case ErrorSeverity.HIGH:
        console.error(logMessage, logData);
        break;
      case ErrorSeverity.MEDIUM:
        console.warn(logMessage, logData);
        break;
      case ErrorSeverity.LOW:
        if (this.config.logLevel === 'debug') {
          console.info(logMessage, logData);
        }
        break;
    }
  }

  /**
   * Report error to external service
   */
  private async reportError(error: ErrorDetails): Promise<void> {
    if (!this.config.reportingEndpoint) {
      return;
    }

    try {
      await fetch(this.config.reportingEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          error: {
            id: error.id,
            message: error.message,
            severity: error.severity,
            category: error.category,
            context: error.context,
            stack: error.stack
          },
          timestamp: new Date().toISOString(),
          userAgent: navigator?.userAgent,
          url: window?.location?.href
        })
      });
    } catch (reportingError) {
      console.warn('Failed to report error:', reportingError);
    }
  }

  /**
   * Add a recovery strategy for a specific error category
   */
  public addRecoveryStrategy(category: ErrorCategory, strategy: RecoveryStrategy): void {
    if (!this.recoveryStrategies.has(category)) {
      this.recoveryStrategies.set(category, []);
    }
    this.recoveryStrategies.get(category)!.push(strategy);
  }

  /**
   * Clear recovery strategies for a specific error category
   */
  public clearRecoveryStrategies(category: ErrorCategory): void {
    this.recoveryStrategies.set(category, []);
  }

  /**
   * Add error listener
   */
  public addErrorListener(listener: (error: ErrorDetails) => void): void {
    this.errorListeners.push(listener);
  }

  /**
   * Remove error listener
   */
  public removeErrorListener(listener: (error: ErrorDetails) => void): void {
    const index = this.errorListeners.indexOf(listener);
    if (index > -1) {
      this.errorListeners.splice(index, 1);
    }
  }

  /**
   * Notify all error listeners
   */
  private notifyListeners(error: ErrorDetails): void {
    this.errorListeners.forEach(listener => {
      try {
        listener(error);
      } catch (listenerError) {
        console.warn('Error listener failed:', listenerError);
      }
    });
  }

  /**
   * Create a safe wrapper for async functions
   */
  public wrapAsync<T extends (...args: unknown[]) => Promise<unknown>>(
    fn: T,
    context: Partial<ErrorContext> = {},
    category: ErrorCategory = ErrorCategory.BUSINESS_LOGIC
  ): T {
    return (async (...args: Parameters<T>) => {
      try {
        return await fn(...args);
      } catch (error) {
        await this.handleError(error as Error, context, category);
        throw error; // Re-throw after handling
      }
    }) as T;
  }

  /**
   * Create a safe wrapper for sync functions
   */
  public wrapSync<T extends (...args: unknown[]) => unknown>(
    fn: T,
    context: Partial<ErrorContext> = {},
    category: ErrorCategory = ErrorCategory.BUSINESS_LOGIC
  ): T {
    return ((...args: Parameters<T>) => {
      try {
        return fn(...args);
      } catch (error) {
        this.handleError(error as Error, context, category);
        throw error; // Re-throw after handling
      }
    }) as T;
  }

  /**
   * Get error statistics
   */
  public getErrorStats(): {
    total: number;
    bySeverity: Record<ErrorSeverity, number>;
    byCategory: Record<ErrorCategory, number>;
    recentErrors: ErrorDetails[];
  } {
    const bySeverity = {} as Record<ErrorSeverity, number>;
    const byCategory = {} as Record<ErrorCategory, number>;

    // Initialize counters
    Object.values(ErrorSeverity).forEach(severity => {
      bySeverity[severity] = 0;
    });
    Object.values(ErrorCategory).forEach(category => {
      byCategory[category] = 0;
    });

    // Count errors
    this.errorHistory.forEach(error => {
      bySeverity[error.severity]++;
      byCategory[error.category]++;
    });

    // Get recent errors (last 10)
    const recentErrors = this.errorHistory
      .slice(-10)
      .sort((a, b) => b.context.timestamp.getTime() - a.context.timestamp.getTime());

    return {
      total: this.errorHistory.length,
      bySeverity,
      byCategory,
      recentErrors
    };
  }

  /**
   * Clear error history
   */
  public clearHistory(): void {
    this.errorHistory = [];
  }

  /**
   * Generate unique error ID
   */
  private generateErrorId(): string {
    return `err_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Utility delay function
   */
  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Singleton instance
export const errorHandler = new ErrorHandler();
export default ErrorHandler;
