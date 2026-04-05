import { LogLevel, LogFormat, LogEntry, LoggerOptions, LoggerContext } from './types';
import { LogLevelUtils, LogFormatter } from './formatters';

export { LogLevel, LogFormat };

/**
 * Create a logger with the specified level and format
 */
export function createLogger(level: LogLevel = LogLevel.INFO, format: LogFormat = LogFormat.CONSOLE): Logger {
  return new Logger({
    level,
    format,
    output: 'stdout',
    errorOutput: 'stderr',
    development: false
  });
}

/**
 * Main Logger class providing structured logging capabilities
 */
export class Logger {
  private options: LoggerOptions;
  private context: LoggerContext = {};

  constructor(options: LoggerOptions) {
    this.options = {
      output: 'stdout',
      errorOutput: 'stderr',
      development: false,
      ...options
    };

    // Validate and normalize options
    this.options.level = LogLevelUtils.parseLevel(this.options.level as string);
    this.options.format = this.options.format as LogFormat;
  }

  /**
   * Log a debug message
   */
  debug(message: string, context?: LoggerContext): void {
    this.log(LogLevel.DEBUG, message, context);
  }

  /**
   * Log an info message
   */
  info(message: string, context?: LoggerContext): void {
    this.log(LogLevel.INFO, message, context);
  }

  /**
   * Log a warning message
   */
  warn(message: string, context?: LoggerContext): void {
    this.log(LogLevel.WARN, message, context);
  }

  /**
   * Log an error message
   */
  error(message: string, error?: Error | LoggerContext): void {
    let context: LoggerContext = {};
    if (error instanceof Error) {
      context.error = error;
    } else if (error) {
      context = error;
    }
    this.log(LogLevel.ERROR, message, context);
  }

  /**
   * Log a fatal message
   */
  fatal(message: string, error?: Error | LoggerContext): void {
    let context: LoggerContext = {};
    if (error instanceof Error) {
      context.error = error;
    } else if (error) {
      context = error;
    }
    this.log(LogLevel.FATAL, message, context);
  }

  /**
   * Core logging method
   */
  private log(level: LogLevel, message: string, context?: LoggerContext): void {
    if (!LogLevelUtils.shouldLog(this.options.level as LogLevel, level)) {
      return;
    }

    const entry: LogEntry = {
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
    const error = context?.error || this.context.error;
    if (error) {
      entry.stacktrace = LogFormatter.createErrorStack(error);
    }

    // Format and output the log entry
    const formatted = this.formatEntry(entry);
    this.output(formatted, level);
  }

  /**
   * Format log entry based on configured format
   */
  private formatEntry(entry: LogEntry): string {
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
  private output(formatted: string, level: LogLevel): void {
    const isError = level === LogLevel.ERROR || level === LogLevel.FATAL;
    const output = isError ? this.options.errorOutput : this.options.output;
    
    if (output === 'stderr') {
      console.error(formatted);
    } else {
      console.log(formatted);
    }
  }

  /**
   * Create a new logger with additional block ID context
   */
  withBlockID(blockID: string): Logger {
    const newLogger = new Logger(this.options);
    newLogger.context = { ...this.context, block_id: blockID };
    return newLogger;
  }

  /**
   * Create a new logger with additional user ID context
   */
  withUserID(userID: string): Logger {
    const newLogger = new Logger(this.options);
    newLogger.context = { ...this.context, user_id: userID };
    return newLogger;
  }

  /**
   * Create a new logger with additional error context
   */
  withError(error: Error): Logger {
    const newLogger = new Logger(this.options);
    newLogger.context = { ...this.context, error };
    return newLogger;
  }

  /**
   * Create a new logger with additional context
   */
  withContext(context: LoggerContext): Logger {
    const newLogger = new Logger(this.options);
    newLogger.context = { ...this.context, ...context };
    return newLogger;
  }

  /**
   * Get current logger options
   */
  getOptions(): LoggerOptions {
    return { ...this.options };
  }

  /**
   * Get current logger context
   */
  getContext(): LoggerContext {
    return { ...this.context };
  }
}