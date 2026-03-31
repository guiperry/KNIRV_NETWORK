import { Logger } from './logging';
import { LogLevel, LogFormat } from './types';
import { LogLevelUtils } from './formatters';
/**
 * Factory function to create a logger
 */
export function createLogger(level, format) {
    return new Logger({
        level: level,
        format: format
    });
}
/**
 * Create a production-ready logger
 */
export function createProductionLogger(level = LogLevel.INFO) {
    return new Logger({
        level: level,
        format: LogFormat.JSON,
        development: false
    });
}
/**
 * Create a development logger
 */
export function createDevelopmentLogger(level = LogLevel.DEBUG) {
    return new Logger({
        level: level,
        format: LogFormat.CONSOLE,
        development: true
    });
}
/**
 * Create a logger from environment variables
 */
export function createLoggerFromEnv() {
    const level = process.env.LOG_LEVEL || LogLevel.INFO;
    const format = process.env.LOG_FORMAT || LogFormat.JSON;
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
export { Logger, LogLevel, LogFormat, LogLevelUtils };
//# sourceMappingURL=index.js.map