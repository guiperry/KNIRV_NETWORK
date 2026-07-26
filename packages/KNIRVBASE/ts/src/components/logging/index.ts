import { Logger } from './logging';
import { LogLevel, LogFormat, LoggerOptions } from './types';
import { LogLevelUtils } from './formatters';

/**
 * Factory function to create a logger
 */
export function createLogger(level: string | LogLevel, format: string | LogFormat): Logger {
  return new Logger({
    level: level as LogLevel,
    format: format as LogFormat
  });
}

/**
 * Create a production-ready logger
 */
export function createProductionLogger(level: string | LogLevel = LogLevel.INFO): Logger {
  return new Logger({
    level: level as LogLevel,
    format: LogFormat.JSON,
    development: false
  });
}

/**
 * Create a development logger
 */
export function createDevelopmentLogger(level: string | LogLevel = LogLevel.DEBUG): Logger {
  return new Logger({
    level: level as LogLevel,
    format: LogFormat.CONSOLE,
    development: true
  });
}

/**
 * Create a logger from environment variables
 */
export function createLoggerFromEnv(): Logger {
  const level = process.env.LOG_LEVEL || LogLevel.INFO;
  const format = (process.env.LOG_FORMAT as LogFormat) || LogFormat.JSON;
  const development = process.env.NODE_ENV === 'development';

  return new Logger({
    level,
    format,
    development
  });
}

/**
 * Default logger instance
 */
export const defaultLogger = createLoggerFromEnv();

// Export all types and utilities
export { Logger, LogLevel, LogFormat, LoggerOptions, LogLevelUtils };