import { LogLevel, LogFormat } from './types';
import { LogLevelUtils, LogFormatter } from './formatters';
/**
 * Main Logger class providing structured logging capabilities
 */
export class Logger {
    constructor(options) {
        this.context = {};
        this.options = {
            output: 'stdout',
            errorOutput: 'stderr',
            development: false,
            ...options
        };
        // Validate and normalize options
        this.options.level = LogLevelUtils.parseLevel(this.options.level);
        this.options.format = this.options.format;
    }
    /**
     * Log a debug message
     */
    debug(message, context) {
        this.log(LogLevel.DEBUG, message, context);
    }
    /**
     * Log an info message
     */
    info(message, context) {
        this.log(LogLevel.INFO, message, context);
    }
    /**
     * Log a warning message
     */
    warn(message, context) {
        this.log(LogLevel.WARN, message, context);
    }
    /**
     * Log an error message
     */
    error(message, error) {
        let context = {};
        if (error instanceof Error) {
            context.error = error;
        }
        else if (error) {
            context = error;
        }
        this.log(LogLevel.ERROR, message, context);
    }
    /**
     * Log a fatal message
     */
    fatal(message, error) {
        let context = {};
        if (error instanceof Error) {
            context.error = error;
        }
        else if (error) {
            context = error;
        }
        this.log(LogLevel.FATAL, message, context);
    }
    /**
     * Core logging method
     */
    log(level, message, context) {
        if (!LogLevelUtils.shouldLog(this.options.level, level)) {
            return;
        }
        const entry = {
            timestamp: LogFormatter.formatTimestamp(),
            level: level,
            message: message,
            ...this.context,
            ...context
        };
        // Add caller information in development mode
        if (this.options.development) {
            entry.caller = LogFormatter.formatCaller();
        }
        // Add error details if present
        if (context?.error) {
            entry.stacktrace = LogFormatter.createErrorStack(context.error);
        }
        // Format and output the log entry
        const formatted = this.formatEntry(entry);
        this.output(formatted, level);
    }
    /**
     * Format log entry based on configured format
     */
    formatEntry(entry) {
        switch (this.options.format) {
            case LogFormat.JSON:
                return LogFormatter.formatJSON(entry);
            case LogFormat.CONSOLE:
                return LogFormatter.formatConsole(entry);
            default:
                return LogFormatter.formatConsole(entry);
        }
    }
    /**
     * Output formatted log entry
     */
    output(formatted, level) {
        const isError = level === LogLevel.ERROR || level === LogLevel.FATAL;
        const output = isError ? this.options.errorOutput : this.options.output;
        if (output === 'stderr') {
            console.error(formatted);
        }
        else {
            console.log(formatted);
        }
    }
    /**
     * Create a new logger with additional block ID context
     */
    withBlockID(blockID) {
        const newLogger = new Logger(this.options);
        newLogger.context = { ...this.context, block_id: blockID };
        return newLogger;
    }
    /**
     * Create a new logger with additional user ID context
     */
    withUserID(userID) {
        const newLogger = new Logger(this.options);
        newLogger.context = { ...this.context, user_id: userID };
        return newLogger;
    }
    /**
     * Create a new logger with additional error context
     */
    withError(error) {
        const newLogger = new Logger(this.options);
        newLogger.context = { ...this.context, error };
        return newLogger;
    }
    /**
     * Create a new logger with additional context
     */
    withContext(context) {
        const newLogger = new Logger(this.options);
        newLogger.context = { ...this.context, ...context };
        return newLogger;
    }
    /**
     * Get current logger options
     */
    getOptions() {
        return { ...this.options };
    }
    /**
     * Get current logger context
     */
    getContext() {
        return { ...this.context };
    }
}
//# sourceMappingURL=logging.js.map