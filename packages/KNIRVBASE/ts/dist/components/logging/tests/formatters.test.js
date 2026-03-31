import { LogLevelUtils, LogFormatter } from '../formatters';
import { LogLevel } from '../types';
describe('LogLevelUtils', () => {
    describe('parseLevel', () => {
        it('should parse valid log levels', () => {
            expect(LogLevelUtils.parseLevel('debug')).toBe(LogLevel.DEBUG);
            expect(LogLevelUtils.parseLevel('info')).toBe(LogLevel.INFO);
            expect(LogLevelUtils.parseLevel('warn')).toBe(LogLevel.WARN);
            expect(LogLevelUtils.parseLevel('warning')).toBe(LogLevel.WARN);
            expect(LogLevelUtils.parseLevel('error')).toBe(LogLevel.ERROR);
            expect(LogLevelUtils.parseLevel('fatal')).toBe(LogLevel.FATAL);
        });
        it('should throw error for invalid log level', () => {
            expect(() => LogLevelUtils.parseLevel('invalid')).toThrow('Invalid log level: invalid');
        });
    });
    describe('isValidLevel', () => {
        it('should return true for valid levels', () => {
            expect(LogLevelUtils.isValidLevel('debug')).toBe(true);
            expect(LogLevelUtils.isValidLevel('info')).toBe(true);
            expect(LogLevelUtils.isValidLevel('warn')).toBe(true);
            expect(LogLevelUtils.isValidLevel('error')).toBe(true);
            expect(LogLevelUtils.isValidLevel('fatal')).toBe(true);
        });
        it('should return false for invalid levels', () => {
            expect(LogLevelUtils.isValidLevel('invalid')).toBe(false);
        });
    });
    describe('getNumericLevel', () => {
        it('should return correct numeric values', () => {
            expect(LogLevelUtils.getNumericLevel(LogLevel.DEBUG)).toBe(10);
            expect(LogLevelUtils.getNumericLevel(LogLevel.INFO)).toBe(20);
            expect(LogLevelUtils.getNumericLevel(LogLevel.WARN)).toBe(30);
            expect(LogLevelUtils.getNumericLevel(LogLevel.ERROR)).toBe(40);
            expect(LogLevelUtils.getNumericLevel(LogLevel.FATAL)).toBe(50);
        });
    });
    describe('shouldLog', () => {
        it('should determine when to log correctly', () => {
            expect(LogLevelUtils.shouldLog(LogLevel.DEBUG, LogLevel.DEBUG)).toBe(true);
            expect(LogLevelUtils.shouldLog(LogLevel.DEBUG, LogLevel.INFO)).toBe(true);
            expect(LogLevelUtils.shouldLog(LogLevel.INFO, LogLevel.DEBUG)).toBe(false);
            expect(LogLevelUtils.shouldLog(LogLevel.WARN, LogLevel.ERROR)).toBe(true);
            expect(LogLevelUtils.shouldLog(LogLevel.ERROR, LogLevel.WARN)).toBe(false);
        });
    });
});
describe('LogFormatter', () => {
    describe('formatTimestamp', () => {
        it('should return ISO timestamp', () => {
            const timestamp = LogFormatter.formatTimestamp();
            expect(timestamp).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/);
        });
    });
    describe('formatCaller', () => {
        it('should return caller information', () => {
            const caller = LogFormatter.formatCaller();
            expect(caller).toBeTruthy();
        });
    });
    describe('formatConsole', () => {
        it('should format log entry for console output', () => {
            const entry = {
                timestamp: '2023-01-01T00:00:00.000Z',
                level: 'info',
                message: 'Test message',
                caller: 'test.js:10',
                custom_field: 'custom_value'
            };
            const formatted = LogFormatter.formatConsole(entry);
            expect(formatted).toContain('INFO');
            expect(formatted).toContain('Test message');
            expect(formatted).toContain('[test.js:10]');
            expect(formatted).toContain('custom_value');
        });
    });
    describe('formatJSON', () => {
        it('should format log entry as JSON', () => {
            const entry = {
                timestamp: '2023-01-01T00:00:00.000Z',
                level: 'info',
                message: 'Test message',
                custom_field: 'custom_value'
            };
            const formatted = LogFormatter.formatJSON(entry);
            const parsed = JSON.parse(formatted);
            expect(parsed.level).toBe('info');
            expect(parsed.message).toBe('Test message');
            expect(parsed.custom_field).toBe('custom_value');
        });
    });
    describe('createErrorStack', () => {
        it('should create error stack from Error object', () => {
            const error = new Error('Test error');
            const stack = LogFormatter.createErrorStack(error);
            expect(stack).toContain('Test error');
        });
        it('should handle error without stack', () => {
            const error = { message: 'Test error' };
            const stack = LogFormatter.createErrorStack(error);
            expect(stack).toBe('Test error');
        });
        it('should handle string error', () => {
            const stack = LogFormatter.createErrorStack('String error');
            expect(stack).toBe('String error');
        });
    });
});
//# sourceMappingURL=formatters.test.js.map