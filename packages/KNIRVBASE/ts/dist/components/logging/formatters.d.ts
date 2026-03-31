import { LogLevel, LogEntry } from './types';
/**
 * Log level utilities
 */
export declare class LogLevelUtils {
    static parseLevel(level: string): LogLevel;
    static isValidLevel(level: string): boolean;
    static getNumericLevel(level: LogLevel): number;
    static shouldLog(currentLevel: LogLevel, messageLevel: LogLevel): boolean;
}
/**
 * Log formatting utilities
 */
export declare class LogFormatter {
    static formatTimestamp(): string;
    static formatCaller(): string;
    static formatConsole(entry: LogEntry): string;
    static formatJSON(entry: LogEntry): string;
    private static extractContextFields;
    static createErrorStack(error: Error): string;
}
//# sourceMappingURL=formatters.d.ts.map