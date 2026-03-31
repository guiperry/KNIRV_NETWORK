import { Logger } from './logging';
import { LogLevel, LogFormat, LoggerOptions } from './types';
import { LogLevelUtils } from './formatters';
/**
 * Factory function to create a logger
 */
export declare function createLogger(level: string | LogLevel, format: string | LogFormat): Logger;
/**
 * Create a production-ready logger
 */
export declare function createProductionLogger(level?: string | LogLevel): Logger;
/**
 * Create a development logger
 */
export declare function createDevelopmentLogger(level?: string | LogLevel): Logger;
/**
 * Create a logger from environment variables
 */
export declare function createLoggerFromEnv(): Logger;
/**
 * Default logger instance
 */
export declare const defaultLogger: Logger;
export { Logger, LogLevel, LogFormat, LoggerOptions, LogLevelUtils };
//# sourceMappingURL=index.d.ts.map