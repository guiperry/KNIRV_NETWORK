export enum LogLevel {
  DEBUG = 'debug',
  INFO = 'info',
  WARN = 'warn',
  ERROR = 'error',
  FATAL = 'fatal'
}

export enum LogFormat {
  JSON = 'json',
  CONSOLE = 'console'
}

export interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
  logger?: string;
  caller?: string;
  stacktrace?: string;
  [key: string]: any; // Additional context fields
}

export interface LoggerOptions {
  level: LogLevel | string;
  format: LogFormat | string;
  output?: 'stdout' | 'stderr';
  errorOutput?: 'stderr' | 'stdout';
  development?: boolean;
}

export interface LoggerContext {
  block_id?: string;
  user_id?: string;
  error?: Error;
  [key: string]: any;
}