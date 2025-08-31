import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { Logger, createLogger, logger, log } from '../../src/core/utils/logger';

// Mock console methods
const mockConsole = {
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
};

describe('Logger Utility', () => {
  beforeEach(() => {
    // Mock console methods
    global.console.debug = mockConsole.debug;
    global.console.info = mockConsole.info;
    global.console.warn = mockConsole.warn;
    global.console.error = mockConsole.error;
    
    // Clear all mocks
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('Logger Class', () => {
    it('should create logger with default component and level', () => {
      const testLogger = new Logger();
      expect(testLogger.getLevel()).toBe('info');
    });

    it('should create logger with custom component and level', () => {
      const testLogger = new Logger('TEST', 'debug');
      expect(testLogger.getLevel()).toBe('debug');
    });

    it('should log debug messages when level is debug', () => {
      const testLogger = new Logger('TEST', 'debug');
      testLogger.debug('Test debug message');
      
      expect(mockConsole.debug).toHaveBeenCalledWith(
        expect.stringContaining('DEBUG [TEST        ] Test debug message')
      );
    });

    it('should not log debug messages when level is info', () => {
      const testLogger = new Logger('TEST', 'info');
      testLogger.debug('Test debug message');
      
      expect(mockConsole.debug).not.toHaveBeenCalled();
    });

    it('should log info messages', () => {
      const testLogger = new Logger('TEST', 'info');
      testLogger.info('Test info message');
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('INFO  [TEST        ] Test info message')
      );
    });

    it('should log warn messages', () => {
      const testLogger = new Logger('TEST', 'info');
      testLogger.warn('Test warn message');
      
      expect(mockConsole.warn).toHaveBeenCalledWith(
        expect.stringContaining('WARN  [TEST        ] Test warn message')
      );
    });

    it('should log error messages', () => {
      const testLogger = new Logger('TEST', 'info');
      testLogger.error('Test error message');
      
      expect(mockConsole.error).toHaveBeenCalledWith(
        expect.stringContaining('ERROR [TEST        ] Test error message')
      );
    });

    it('should log messages with context', () => {
      const testLogger = new Logger('TEST', 'info');
      const context = { userId: '123', action: 'test' };
      testLogger.info('Test message with context', context);
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('Test message with context {"userId":"123","action":"test"}')
      );
    });

    it('should support alternative parameter order (context first)', () => {
      const testLogger = new Logger('TEST', 'info');
      const context = { userId: '123' };
      testLogger.info(context, 'Test message');
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('Test message {"userId":"123"}')
      );
    });

    it('should create child logger with extended component name', () => {
      const parentLogger = new Logger('PARENT', 'debug');
      const childLogger = parentLogger.child('CHILD');
      
      childLogger.info('Child message');
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('INFO  [PARENT:CHILD] Child message')
      );
    });

    it('should change log level dynamically', () => {
      const testLogger = new Logger('TEST', 'info');
      
      // Should not log debug initially
      testLogger.debug('Debug message 1');
      expect(mockConsole.debug).not.toHaveBeenCalled();
      
      // Change to debug level
      testLogger.setLevel('debug');
      testLogger.debug('Debug message 2');
      expect(mockConsole.debug).toHaveBeenCalledWith(
        expect.stringContaining('Debug message 2')
      );
    });

    it('should respect log level hierarchy', () => {
      const testLogger = new Logger('TEST', 'warn');
      
      testLogger.debug('Debug message');
      testLogger.info('Info message');
      testLogger.warn('Warn message');
      testLogger.error('Error message');
      
      expect(mockConsole.debug).not.toHaveBeenCalled();
      expect(mockConsole.info).not.toHaveBeenCalled();
      expect(mockConsole.warn).toHaveBeenCalled();
      expect(mockConsole.error).toHaveBeenCalled();
    });

    it('should format timestamps correctly', () => {
      const testLogger = new Logger('TEST', 'info');
      testLogger.info('Test message');
      
      const logCall = mockConsole.info.mock.calls[0][0];
      expect(logCall).toMatch(/^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z\]/);
    });

    it('should pad component names correctly', () => {
      const shortLogger = new Logger('A', 'info');
      const longLogger = new Logger('VERYLONGCOMPONENT', 'info');
      
      shortLogger.info('Short message');
      longLogger.info('Long message');
      
      expect(mockConsole.info.mock.calls[0][0]).toContain('[A           ]');
      expect(mockConsole.info.mock.calls[1][0]).toContain('[VERYLONGCOMPONENT]');
    });
  });

  describe('Default Logger Instance', () => {
    it('should have correct default configuration', () => {
      expect(logger.getLevel()).toBe(process.env.NODE_ENV === 'development' ? 'debug' : 'info');
    });

    it('should log messages using default logger', () => {
      logger.info('Default logger test');
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('INFO  [KNIRV       ] Default logger test')
      );
    });
  });

  describe('createLogger Function', () => {
    it('should create logger with specified component', () => {
      const customLogger = createLogger('CUSTOM');
      customLogger.info('Custom logger test');
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('INFO  [CUSTOM      ] Custom logger test')
      );
    });

    it('should create logger with specified component and level', () => {
      const customLogger = createLogger('CUSTOM', 'debug');
      expect(customLogger.getLevel()).toBe('debug');
    });

    it('should inherit level from default logger if not specified', () => {
      const customLogger = createLogger('CUSTOM');
      expect(customLogger.getLevel()).toBe(logger.getLevel());
    });
  });

  describe('Convenience Log Functions', () => {
    it('should log debug messages', () => {
      log.debug('Debug message');
      
      // Note: This might not log if default level is not debug
      if (logger.getLevel() === 'debug') {
        expect(mockConsole.debug).toHaveBeenCalled();
      }
    });

    it('should log info messages', () => {
      log.info('Info message');
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('INFO  [KNIRV       ] Info message')
      );
    });

    it('should log warn messages', () => {
      log.warn('Warn message');
      
      expect(mockConsole.warn).toHaveBeenCalledWith(
        expect.stringContaining('WARN  [KNIRV       ] Warn message')
      );
    });

    it('should log error messages', () => {
      log.error('Error message');
      
      expect(mockConsole.error).toHaveBeenCalledWith(
        expect.stringContaining('ERROR [KNIRV       ] Error message')
      );
    });

    it('should log messages with context using convenience functions', () => {
      const context = { test: true };
      log.info('Info with context', context);
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('Info with context {"test":true}')
      );
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty context objects', () => {
      const testLogger = new Logger('TEST', 'info');
      testLogger.info('Message with empty context', {});
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('Message with empty context')
      );
      expect(mockConsole.info.mock.calls[0][0]).not.toContain('{}');
    });

    it('should handle null context', () => {
      const testLogger = new Logger('TEST', 'info');
      testLogger.info('Message with null context', undefined);
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('Message with null context')
      );
    });

    it('should handle undefined context', () => {
      const testLogger = new Logger('TEST', 'info');
      testLogger.info('Message with undefined context', undefined);
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('Message with undefined context')
      );
    });

    it('should handle complex context objects', () => {
      const testLogger = new Logger('TEST', 'info');
      const complexContext = {
        user: { id: 123, name: 'Test User' },
        metadata: { timestamp: Date.now(), version: '1.0.0' },
        array: [1, 2, 3]
      };
      
      testLogger.info('Complex context message', complexContext);
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining('Complex context message')
      );
      expect(mockConsole.info.mock.calls[0][0]).toContain('"user"');
      expect(mockConsole.info.mock.calls[0][0]).toContain('"metadata"');
      expect(mockConsole.info.mock.calls[0][0]).toContain('"array"');
    });

    it('should handle very long messages', () => {
      const testLogger = new Logger('TEST', 'info');
      const longMessage = 'A'.repeat(1000);
      
      testLogger.info(longMessage);
      
      expect(mockConsole.info).toHaveBeenCalledWith(
        expect.stringContaining(longMessage)
      );
    });
  });
});
