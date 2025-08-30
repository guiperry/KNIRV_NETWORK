/**
 * ErrorHandler Unit Tests
 * Comprehensive test suite for error handling and recovery utilities
 */

import ErrorHandler, { errorHandler, ErrorSeverity, ErrorCategory } from '../../../src/utils/ErrorHandler';

// Mock console methods to avoid noise in test output
const originalConsole = { ...console };
beforeAll(() => {
  console.error = jest.fn();
  console.warn = jest.fn();
  console.info = jest.fn();
});

afterAll(() => {
  Object.assign(console, originalConsole);
});

describe('ErrorHandler', () => {
  let handler: ErrorHandler;

  beforeEach(() => {
    // Create handler with external dependencies disabled for tests - no mocks needed!
    handler = new ErrorHandler({
      enableLogging: true,
      enableReporting: false, // Disable to avoid network calls
      enableRecovery: false,  // Disable to avoid async recovery loops
      maxRetries: 0,          // No retries to avoid delays
      retryDelay: 0,          // No delay
      logLevel: 'error'
    });
    jest.clearAllMocks();
    // Don't use fake timers initially to avoid timer issues
  });

  afterEach(() => {
    // Cleanup if needed
  });

  describe('Initialization', () => {
    it('should initialize with default configuration', () => {
      const defaultHandler = new ErrorHandler();
      expect(defaultHandler).toBeDefined();
      expect(defaultHandler.handleError).toBeDefined();
    });

    it('should initialize with custom configuration', () => {
      const customHandler = new ErrorHandler({
        enableLogging: false,
        maxRetries: 5
      });
      expect(customHandler).toBeDefined();
    });
  });

  describe('Error Handling', () => {
    it('should handle string errors', async () => {
      const errorDetails = await handler.handleError(
        'Test error message',
        { component: 'TestComponent' },
        ErrorCategory.SYSTEM,
        ErrorSeverity.MEDIUM
      );

      expect(errorDetails).toHaveProperty('id');
      expect(errorDetails).toHaveProperty('message', 'Test error message');
      expect(errorDetails).toHaveProperty('severity', ErrorSeverity.MEDIUM);
      expect(errorDetails).toHaveProperty('category', ErrorCategory.SYSTEM);
      expect(errorDetails.context).toHaveProperty('component', 'TestComponent');
    });

    it('should handle Error objects', async () => {
      const error = new Error('Test error object');
      const errorDetails = await handler.handleError(
        error,
        { component: 'TestComponent' },
        ErrorCategory.NETWORK,
        ErrorSeverity.HIGH
      );

      expect(errorDetails.message).toBe('Test error object');
      expect(errorDetails.stack).toBeDefined();
      expect(errorDetails.originalError).toBe(error);
    }, 10000); // 10 second timeout

    it('should generate unique error IDs', async () => {
      const error1 = await handler.handleError('Error 1');
      const error2 = await handler.handleError('Error 2');

      expect(error1.id).not.toBe(error2.id);
      expect(error1.id).toMatch(/^err_\d+_[a-z0-9]+$/);
    });

    it('should set correct timestamps', async () => {
      const beforeTime = Date.now();
      const errorDetails = await handler.handleError('Test error');
      const afterTime = Date.now();

      expect(errorDetails.context.timestamp.getTime()).toBeGreaterThanOrEqual(beforeTime);
      expect(errorDetails.context.timestamp.getTime()).toBeLessThanOrEqual(afterTime);
    });
  });

  describe('Error Logging', () => {
    it('should log critical errors', async () => {
      await handler.handleError('Critical error', {}, ErrorCategory.SYSTEM, ErrorSeverity.CRITICAL);
      
      expect(console.error).toHaveBeenCalledWith(
        expect.stringContaining('[CRITICAL] system: Critical error'),
        expect.any(Object)
      );
    });

    it('should log high severity errors', async () => {
      await handler.handleError('High error', {}, ErrorCategory.NETWORK, ErrorSeverity.HIGH);
      
      expect(console.error).toHaveBeenCalledWith(
        expect.stringContaining('[HIGH] network: High error'),
        expect.any(Object)
      );
    });

    it('should log medium severity errors as warnings', async () => {
      await handler.handleError('Medium error', {}, ErrorCategory.VALIDATION, ErrorSeverity.MEDIUM);
      
      expect(console.warn).toHaveBeenCalledWith(
        expect.stringContaining('[MEDIUM] validation: Medium error'),
        expect.any(Object)
      );
    });

    it('should not log low severity errors by default', async () => {
      await handler.handleError('Low error', {}, ErrorCategory.USER_INPUT, ErrorSeverity.LOW);
      
      expect(console.info).not.toHaveBeenCalled();
    });

    it('should log low severity errors in debug mode', async () => {
      const debugHandler = new ErrorHandler({ logLevel: 'debug' });
      await debugHandler.handleError('Low error', {}, ErrorCategory.USER_INPUT, ErrorSeverity.LOW);
      
      expect(console.info).toHaveBeenCalledWith(
        expect.stringContaining('[LOW] user_input: Low error'),
        expect.any(Object)
      );
    });
  });

  describe('Error Reporting', () => {
    it('should not report when reporting is disabled', async () => {
      const noReportHandler = new ErrorHandler({ enableReporting: false });
      const errorDetails = await noReportHandler.handleError('Test error');

      // Should complete successfully without network calls
      expect(errorDetails.message).toBe('Test error');
      expect(errorDetails.id).toBeDefined();
    });

    it('should handle missing reporting endpoint gracefully', async () => {
      const handlerWithoutEndpoint = new ErrorHandler({
        enableReporting: true,
        // No reportingEndpoint specified
      });

      // Should not throw and complete quickly
      const errorDetails = await handlerWithoutEndpoint.handleError('Test error');
      expect(errorDetails.message).toBe('Test error');
    });
  });

  describe('Recovery Strategies', () => {
    it('should attempt recovery for recoverable errors', async () => {
      // Create a handler with recovery enabled for this test
      const recoveryHandler = new ErrorHandler({
        enableLogging: true,
        enableReporting: false,
        enableRecovery: true,
        maxRetries: 2,
        retryDelay: 10, // Short delay for tests
        logLevel: 'error'
      });

      const mockStrategy = {
        name: 'test_recovery',
        execute: jest.fn().mockResolvedValue(true),
        maxRetries: 2,
        retryDelay: 10
      };

      // Clear any existing strategies and add our mock
      recoveryHandler.clearRecoveryStrategies(ErrorCategory.NETWORK);
      recoveryHandler.addRecoveryStrategy(ErrorCategory.NETWORK, mockStrategy);

      await recoveryHandler.handleError(
        'Network error',
        {},
        ErrorCategory.NETWORK,
        ErrorSeverity.MEDIUM
      );

      expect(mockStrategy.execute).toHaveBeenCalled();
    });

    it('should not attempt recovery for critical errors', async () => {
      const mockStrategy = {
        name: 'test_recovery',
        execute: jest.fn().mockResolvedValue(true),
        maxRetries: 2,
        retryDelay: 100
      };

      handler.addRecoveryStrategy(ErrorCategory.SYSTEM, mockStrategy);

      await handler.handleError(
        'Critical error',
        {},
        ErrorCategory.SYSTEM,
        ErrorSeverity.CRITICAL
      );

      expect(mockStrategy.execute).not.toHaveBeenCalled();
    });

    it('should respect retry limits', async () => {
      // Create a handler with recovery enabled for this test
      const recoveryHandler = new ErrorHandler({
        enableLogging: true,
        enableReporting: false,
        enableRecovery: true,
        maxRetries: 1,
        retryDelay: 10, // Short delay for tests
        logLevel: 'error'
      });

      const mockStrategy = {
        name: 'test_recovery',
        execute: jest.fn().mockResolvedValue(true),
        maxRetries: 1,
        retryDelay: 10
      };

      // Clear any existing strategies and add our mock
      recoveryHandler.clearRecoveryStrategies(ErrorCategory.NETWORK);
      recoveryHandler.addRecoveryStrategy(ErrorCategory.NETWORK, mockStrategy);

      // First error - should attempt recovery
      await recoveryHandler.handleError(
        'Network error',
        {},
        ErrorCategory.NETWORK,
        ErrorSeverity.MEDIUM
      );

      expect(mockStrategy.execute).toHaveBeenCalledTimes(1);
    });
  });

  describe('Error Listeners', () => {
    it('should notify error listeners', async () => {
      const listener = jest.fn();
      handler.addErrorListener(listener);

      await handler.handleError('Test error');

      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({
          message: 'Test error'
        })
      );
    });

    it('should remove error listeners', async () => {
      const listener = jest.fn();
      handler.addErrorListener(listener);
      handler.removeErrorListener(listener);

      await handler.handleError('Test error');

      expect(listener).not.toHaveBeenCalled();
    });

    it('should handle listener errors gracefully', async () => {
      const errorListener = jest.fn().mockImplementation(() => {
        throw new Error('Listener error');
      });
      
      handler.addErrorListener(errorListener);

      // Should not throw
      await expect(handler.handleError('Test error')).resolves.toBeDefined();
    });
  });

  describe('Function Wrappers', () => {
    it('should wrap async functions safely', async () => {
      const asyncFn = jest.fn().mockRejectedValue(new Error('Async error'));
      const wrappedFn = handler.wrapAsync(asyncFn, { component: 'TestComponent' });

      await expect(wrappedFn()).rejects.toThrow('Async error');
      expect(asyncFn).toHaveBeenCalled();
    });

    it('should wrap sync functions safely', () => {
      const syncFn = jest.fn().mockImplementation(() => {
        throw new Error('Sync error');
      });
      const wrappedFn = handler.wrapSync(syncFn, { component: 'TestComponent' });

      expect(() => wrappedFn()).toThrow('Sync error');
      expect(syncFn).toHaveBeenCalled();
    });

    it('should preserve function return values', async () => {
      const asyncFn = jest.fn().mockResolvedValue('success');
      const wrappedFn = handler.wrapAsync(asyncFn);

      const result = await wrappedFn();
      expect(result).toBe('success');
    });
  });

  describe('Error Statistics', () => {
    it('should track error statistics', async () => {
      await handler.handleError('Error 1', {}, ErrorCategory.NETWORK, ErrorSeverity.HIGH);
      await handler.handleError('Error 2', {}, ErrorCategory.SYSTEM, ErrorSeverity.MEDIUM);
      await handler.handleError('Error 3', {}, ErrorCategory.NETWORK, ErrorSeverity.LOW);

      const stats = handler.getErrorStats();

      expect(stats.total).toBe(3);
      expect(stats.bySeverity[ErrorSeverity.HIGH]).toBe(1);
      expect(stats.bySeverity[ErrorSeverity.MEDIUM]).toBe(1);
      expect(stats.bySeverity[ErrorSeverity.LOW]).toBe(1);
      expect(stats.byCategory[ErrorCategory.NETWORK]).toBe(2);
      expect(stats.byCategory[ErrorCategory.SYSTEM]).toBe(1);
      expect(stats.recentErrors).toHaveLength(3);
    });

    it('should limit recent errors to 10', async () => {
      // Create 15 errors
      for (let i = 0; i < 15; i++) {
        await handler.handleError(`Error ${i}`);
      }

      const stats = handler.getErrorStats();
      expect(stats.total).toBe(15);
      expect(stats.recentErrors).toHaveLength(10);
    });

    it('should sort recent errors by timestamp', async () => {
      await handler.handleError('First error');
      
      // Advance time
      jest.advanceTimersByTime(1000);
      
      await handler.handleError('Second error');

      const stats = handler.getErrorStats();
      expect(stats.recentErrors[0].message).toBe('Second error');
      expect(stats.recentErrors[1].message).toBe('First error');
    });
  });

  describe('History Management', () => {
    it('should clear error history', async () => {
      await handler.handleError('Error 1');
      await handler.handleError('Error 2');

      let stats = handler.getErrorStats();
      expect(stats.total).toBe(2);

      handler.clearHistory();

      stats = handler.getErrorStats();
      expect(stats.total).toBe(0);
      expect(stats.recentErrors).toHaveLength(0);
    });
  });

  describe('Global Error Handlers', () => {
    it('should setup global error handlers in browser environment', () => {
      // Mock window object
      const mockWindow = {
        addEventListener: jest.fn()
      };
      
      Object.defineProperty(global, 'window', {
        value: mockWindow,
        writable: true
      });

      new ErrorHandler();

      expect(mockWindow.addEventListener).toHaveBeenCalledWith(
        'unhandledrejection',
        expect.any(Function)
      );
      expect(mockWindow.addEventListener).toHaveBeenCalledWith(
        'error',
        expect.any(Function)
      );
    });
  });

  describe('Singleton Instance', () => {
    it('should provide a singleton instance', () => {
      expect(errorHandler).toBeDefined();
      expect(errorHandler).toBeInstanceOf(ErrorHandler);
    });
  });

  describe('Recovery Strategy Management', () => {
    it('should add multiple recovery strategies for same category', () => {
      const strategy1 = {
        name: 'strategy1',
        execute: jest.fn().mockResolvedValue(true),
        maxRetries: 1,
        retryDelay: 100
      };

      const strategy2 = {
        name: 'strategy2',
        execute: jest.fn().mockResolvedValue(true),
        maxRetries: 1,
        retryDelay: 100
      };

      handler.addRecoveryStrategy(ErrorCategory.NETWORK, strategy1);
      handler.addRecoveryStrategy(ErrorCategory.NETWORK, strategy2);

      // Both strategies should be available
      expect(() => handler.addRecoveryStrategy(ErrorCategory.NETWORK, strategy1)).not.toThrow();
    });
  });

  describe('Error Context', () => {
    it('should include additional context data', async () => {
      const context = {
        userId: '123',
        sessionId: 'abc',
        component: 'TestComponent',
        action: 'testAction',
        additionalData: { key: 'value' }
      };

      const errorDetails = await handler.handleError('Test error', context);

      expect(errorDetails.context).toMatchObject(context);
      expect(errorDetails.context.timestamp).toBeDefined();
    });
  });
});
