import { LogLevel, LogFormat, LogEntry, LoggerOptions } from './types';

/**
 * Log level utilities
 */
export class LogLevelUtils {
  static parseLevel(level: string): LogLevel {
    const normalized = level.toLowerCase();
    switch (normalized) {
      case 'debug':
        return LogLevel.DEBUG;
      case 'info':
        return LogLevel.INFO;
      case 'warn':
      case 'warning':
        return LogLevel.WARN;
      case 'error':
        return LogLevel.ERROR;
      case 'fatal':
        return LogLevel.FATAL;
      default:
        throw new Error(`Invalid log level: ${level}`);
    }
  }

  static isValidLevel(level: string): boolean {
    try {
      LogLevelUtils.parseLevel(level);
      return true;
    } catch {
      return false;
    }
  }

  static getNumericLevel(level: LogLevel): number {
    switch (level) {
      case LogLevel.DEBUG:
        return 10;
      case LogLevel.INFO:
        return 20;
      case LogLevel.WARN:
        return 30;
      case LogLevel.ERROR:
        return 40;
      case LogLevel.FATAL:
        return 50;
      default:
        return 20;
    }
  }

  static shouldLog(currentLevel: LogLevel, messageLevel: LogLevel): boolean {
    return LogLevelUtils.getNumericLevel(messageLevel) >= LogLevelUtils.getNumericLevel(currentLevel);
  }
}

/**
 * Log formatting utilities
 */
export class LogFormatter {
  static formatTimestamp(): string {
    return new Date().toISOString();
  }

  static formatCaller(): string {
    const stack = new Error().stack;
    if (!stack) return 'unknown';
    
    const lines = stack.split('\n');
    // Skip current function and find caller
    for (let i = 3; i < lines.length; i++) {
      const line = lines[i];
      if (line && !line.includes('logging')) {
        const match = line.match(/at\s+(.+?)\s+\((.+?):(\d+):\d+\)/);
        if (match) {
          return `${match[2]}:${match[3]}`;
        }
      }
    }
    return 'unknown';
  }

  static formatConsole(entry: LogEntry): string {
    const timestamp = entry.timestamp;
    const level = entry.level.toUpperCase().padEnd(5);
    const caller = entry.caller ? ` [${entry.caller}]` : '';
    const context = LogFormatter.extractContextFields(entry);
    const contextStr = Object.keys(context).length > 0 ? ` ${JSON.stringify(context)}` : '';
    
    return `${timestamp} ${level}${caller} ${entry.message}${contextStr}`;
  }

  static formatJSON(entry: LogEntry): string {
    return JSON.stringify(entry);
  }

  private static extractContextFields(entry: LogEntry): Record<string, any> {
    const { timestamp, level, message, logger, caller, stacktrace, ...context } = entry;
    return context;
  }

  static createErrorStack(error: Error): string {
    return error.stack || error.message || String(error);
  }
}