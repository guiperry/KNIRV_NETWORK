/**
 * Sentry Error Monitoring Service
 * Production error tracking and performance monitoring for KNIRVCONTROLLER
 */

import * as Sentry from '@sentry/browser';

export class SentryMonitoringService {
  private initialized = false;

  async initialize(): Promise<void> {
    if (this.initialized) return;

    const dsn = import.meta.env.VITE_SENTRY_DSN;

    // Only initialize if DSN is provided
    if (!dsn) {
      console.warn('Sentry DSN not configured, error monitoring disabled');
      return;
    }

    Sentry.init({
      dsn,
      environment: import.meta.env.VITE_SENTRY_ENV || import.meta.env.VITE_APP_ENV || 'development',
      release: import.meta.env.VITE_APP_VERSION,
      // Basic configuration without complex integrations
      sampleRate: import.meta.env.PROD ? 0.1 : 1.0, // 10% in production, 100% in dev
    });

    // Set user context if available
    this.setUserContext();

    // Configure global error handlers
    this.configureErrorHandlers();

    console.log('Sentry monitoring initialized');
    this.initialized = true;
  }

  private setUserContext(): void {
    // Set user context from local storage or app state
    const userId = localStorage.getItem('knirv_user_id');
    const walletAddress = localStorage.getItem('knirv_wallet_address');

    if (userId || walletAddress) {
      Sentry.setUser({
        id: userId || undefined,
        username: walletAddress || undefined,
      });
    }

    // Set additional context
    Sentry.setTag('app_version', import.meta.env.VITE_APP_VERSION);
    Sentry.setTag('network_type', import.meta.env.VITE_KNIRV_NETWORK_TYPE);
  }

  private configureErrorHandlers(): void {
    // Global unhandled promise rejection handler
    window.addEventListener('unhandledrejection', (event) => {
      console.error('Unhandled promise rejection:', event.reason);
      Sentry.captureException(event.reason);
    });

    // Global error handler for additional context
    window.addEventListener('error', (event) => {
      Sentry.withScope((scope) => {
        scope.setTag('error_type', 'javascript_error');
        scope.setContext('error_details', {
          message: event.message,
          filename: event.filename,
          lineno: event.lineno,
          colno: event.colno,
        });
        Sentry.captureException(event.error);
      });
    });
  }

  // Public methods for manual error reporting
  captureException(error: Error, context?: Record<string, unknown>): void {
    if (!this.initialized) return;

    Sentry.withScope((scope) => {
      if (context) {
        Object.keys(context).forEach(key => {
          scope.setTag(key, String(context[key]));
        });
      }
      Sentry.captureException(error);
    });
  }

  captureMessage(message: string, level: 'fatal' | 'error' | 'warning' | 'info' | 'debug' = 'info', context?: Record<string, unknown>): void {
    if (!this.initialized) return;

    Sentry.withScope((scope) => {
      scope.setLevel(level as Sentry.SeverityLevel);
      if (context) {
        Object.keys(context).forEach(key => {
          scope.setTag(key, String(context[key]));
        });
      }
      Sentry.captureMessage(message, level as Sentry.SeverityLevel);
    });
  }

  // Wallet transaction monitoring
  trackWalletTransaction(transactionType: string, details: Record<string, unknown>): void {
    if (!this.initialized) return;

    Sentry.withScope((scope) => {
      scope.setTag('transaction_type', transactionType);
      scope.setTag('component', 'wallet');

      // Don't log sensitive wallet data
      const sanitizedDetails = { ...details };
      delete sanitizedDetails.privateKey;
      delete sanitizedDetails.seedPhrase;

      scope.setContext('transaction_details', sanitizedDetails);
      Sentry.captureMessage(`Wallet ${transactionType}`, 'info');
    });
  }

  // Game mechanics monitoring
  trackGameEvent(eventType: string, details: Record<string, unknown>): void {
    if (!this.initialized) return;

    Sentry.withScope((scope) => {
      scope.setTag('game_event', eventType);
      scope.setTag('component', 'knirvana_bridge');
      scope.setContext('game_details', details);
      Sentry.captureMessage(`Game ${eventType}`, 'info');
    });
  }

  // Network monitoring
  trackNetworkRequest(url: string, method: string, duration: number, success: boolean): void {
    if (!this.initialized) return;

    Sentry.withScope((scope) => {
      scope.setTag('network_request', 'true');
      scope.setTag('http_method', method);
      scope.setTag('success', success.toString());

      Sentry.captureMessage(`Network Request: ${method} ${url}`, success ? 'info' : 'warning');
    });
  }

  // Mobile app crash reporting
  trackMobileCrash(error: Error, context: Record<string, unknown>): void {
    if (!this.initialized) return;

    Sentry.withScope((scope) => {
      scope.setTag('platform', 'mobile');
      scope.setTag('crash', 'true');

      scope.setContext('mobile_context', {
        ...context,
        userAgent: navigator.userAgent,
        platform: navigator.platform,
      });

      Sentry.captureException(error);
    });
  }

  // Add breadcrumb for better debugging
  addBreadcrumb(message: string, category: string): void {
    if (!this.initialized) return;

    Sentry.addBreadcrumb({
      message,
      category,
      level: 'info',
    });
  }

  // Update user context
  setUser(userId?: string, walletAddress?: string): void {
    if (!this.initialized) return;

    const user = {
      id: userId,
      username: walletAddress,
    };

    Sentry.setUser(userId || walletAddress ? user : null);
  }

  // Flush pending events (useful before app shutdown)
  async flush(timeout = 2000): Promise<boolean> {
    if (!this.initialized) return true;
    return await Sentry.flush(timeout);
  }

  // Clean up
  destroy(): void {
    if (!this.initialized) return;
    Sentry.close();
    this.initialized = false;
  }
}

// Export singleton instance
export const sentryMonitoringService = new SentryMonitoringService();
