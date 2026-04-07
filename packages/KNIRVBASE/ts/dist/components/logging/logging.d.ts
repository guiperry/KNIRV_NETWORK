import { LogLevel, LogFormat, LoggerOptions, LoggerContext } from './types';
export { LogLevel, LogFormat };
/**
 * Create a logger with the specified level and format
 */
export declare function createLogger(level?: LogLevel, format?: LogFormat): Logger;
/**
 * Main Logger class providing structured logging capabilities
 */
export declare class Logger {
    private options;
    private context;
    constructor(options: LoggerOptions);
    /**
     * Log a debug message
     */
    debug(message: string, context?: LoggerContext): void;
    /**
     * Log an info message
     */
    info(message: string, context?: LoggerContext): void;
    /**
     * Log a warning message
     */
    warn(message: string, context?: LoggerContext): void;
    /**
     * Log an error message
     */
    error(message: string, error?: Error | LoggerContext): void;
    /**
     * Log a fatal message
     */
    fatal(message: string, error?: Error | LoggerContext): void;
    /**
     * Core logging method
     */
    private log;
    /**
     * Format log entry based on configured format
     */
    private formatEntry;
    /**
     * Output formatted log entry
     */
    private output;
    /**
     * Create a new logger with additional block ID context
     */
    withBlockID(blockID: string): Logger;
    /**
     * Create a new logger with additional user ID context
     */
    withUserID(userID: string): Logger;
    /**
     * Create a new logger with additional error context
     */
    withError(error: Error): Logger;
    /**
     * Create a new logger with additional context
     */
    withContext(context: LoggerContext): Logger;
    /**
     * Get current logger options
     */
    getOptions(): LoggerOptions;
    /**
     * Get current logger context
     */
    getContext(): LoggerContext;
}
//# sourceMappingURL=logging.d.ts.map