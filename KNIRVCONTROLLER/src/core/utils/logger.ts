/**
 * Logger utility for KNIRVCONTROLLER
 * Provides structured logging with different levels and contexts
 */

export interface LogContext {
  [key: string]: unknown;
}

export interface LogEntry {
  level: LogLevel;
  message: string;
  context?: LogContext;
  timestamp: Date;
  component?: string;
}

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

class Logger {
  private component: string;
  private logLevel: LogLevel;

  constructor(component: string = 'KNIRV', logLevel: LogLevel = 'info') {
    this.component = component;
    this.logLevel = logLevel;
  }

  private shouldLog(level: LogLevel): boolean {
    const levels: Record<LogLevel, number> = {
      debug: 0,
      info: 1,
      warn: 2,
      error: 3
    };
    
    return levels[level] >= levels[this.logLevel];
  }

  private formatMessage(level: LogLevel, message: string, context?: LogContext): string {
    const timestamp = new Date().toISOString();
    const levelStr = level.toUpperCase().padEnd(5);
    const componentStr = this.component.padEnd(12);
    
    let formatted = `[${timestamp}] ${levelStr} [${componentStr}] ${message}`;
    
    if (context && Object.keys(context).length > 0) {
      formatted += ` ${JSON.stringify(context)}`;
    }
    
    return formatted;
  }

  private log(level: LogLevel, message: string, context?: LogContext): void {
    if (!this.shouldLog(level)) {
      return;
    }

    const formatted = this.formatMessage(level, message, context);
    
    switch (level) {
      case 'debug':
        console.debug(formatted);
        break;
      case 'info':
        console.info(formatted);
        break;
      case 'warn':
        console.warn(formatted);
        break;
      case 'error':
        console.error(formatted);
        break;
    }
  }

  debug(message: string, context?: LogContext): void;
  debug(context: LogContext, message: string): void;
  debug(messageOrContext: string | LogContext, contextOrMessage?: LogContext | string): void {
    if (typeof messageOrContext === 'string') {
      this.log('debug', messageOrContext, contextOrMessage as LogContext);
    } else {
      this.log('debug', contextOrMessage as string, messageOrContext);
    }
  }

  info(message: string, context?: LogContext): void;
  info(context: LogContext, message: string): void;
  info(messageOrContext: string | LogContext, contextOrMessage?: LogContext | string): void {
    if (typeof messageOrContext === 'string') {
      this.log('info', messageOrContext, contextOrMessage as LogContext);
    } else {
      this.log('info', contextOrMessage as string, messageOrContext);
    }
  }

  warn(message: string, context?: LogContext): void;
  warn(context: LogContext, message: string): void;
  warn(messageOrContext: string | LogContext, contextOrMessage?: LogContext | string): void {
    if (typeof messageOrContext === 'string') {
      this.log('warn', messageOrContext, contextOrMessage as LogContext);
    } else {
      this.log('warn', contextOrMessage as string, messageOrContext);
    }
  }

  error(message: string, context?: LogContext): void;
  error(context: LogContext, message: string): void;
  error(messageOrContext: string | LogContext, contextOrMessage?: LogContext | string): void {
    if (typeof messageOrContext === 'string') {
      this.log('error', messageOrContext, contextOrMessage as LogContext);
    } else {
      this.log('error', contextOrMessage as string, messageOrContext);
    }
  }

  child(component: string): Logger {
    return new Logger(`${this.component}:${component}`, this.logLevel);
  }

  setLevel(level: LogLevel): void {
    this.logLevel = level;
  }

  getLevel(): LogLevel {
    return this.logLevel;
  }
}

// Create default logger instance
export const logger = new Logger('KNIRV', process.env.NODE_ENV === 'development' ? 'debug' : 'info');

// Create component-specific loggers
export const createLogger = (component: string, level?: LogLevel): Logger => {
  return new Logger(component, level || logger.getLevel());
};

// Export logger class for custom instances
export { Logger };

// Convenience functions for quick logging
export const log = {
  debug: (message: string, context?: LogContext) => logger.debug(message, context),
  info: (message: string, context?: LogContext) => logger.info(message, context),
  warn: (message: string, context?: LogContext) => logger.warn(message, context),
  error: (message: string, context?: LogContext) => logger.error(message, context),
};

export default logger;
