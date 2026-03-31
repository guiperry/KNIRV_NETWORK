import { Logger, createLogger, LogLevel, LogFormat } from '../logging';
describe('Logger', () => {
    let logger;
    beforeEach(() => {
        logger = createLogger(LogLevel.INFO, LogFormat.JSON);
        jest.spyOn(console, 'log').mockImplementation();
        jest.spyOn(console, 'error').mockImplementation();
    });
    afterEach(() => {
        jest.restoreAllMocks();
    });
    describe('constructor', () => {
        it('should create logger with valid options', () => {
            const testLogger = new Logger({
                level: LogLevel.DEBUG,
                format: LogFormat.CONSOLE
            });
            expect(testLogger).toBeInstanceOf(Logger);
        });
        it('should throw error for invalid log level', () => {
            expect(() => {
                new Logger({
                    level: 'invalid',
                    format: LogFormat.JSON
                });
            }).toThrow('Invalid log level: invalid');
        });
    });
    describe('log levels', () => {
        it('should log info messages', () => {
            logger.info('Test info message');
            expect(console.log).toHaveBeenCalledWith(expect.stringContaining('Test info message'));
        });
        it('should log warning messages', () => {
            logger.warn('Test warning message');
            expect(console.log).toHaveBeenCalledWith(expect.stringContaining('Test warning message'));
        });
        it('should log error messages', () => {
            logger.error('Test error message');
            expect(console.error).toHaveBeenCalledWith(expect.stringContaining('Test error message'));
        });
        it('should log fatal messages', () => {
            logger.fatal('Test fatal message');
            expect(console.error).toHaveBeenCalledWith(expect.stringContaining('Test fatal message'));
        });
        it('should not log debug messages when level is info', () => {
            logger.debug('Test debug message');
            expect(console.log).not.toHaveBeenCalled();
        });
    });
    describe('context methods', () => {
        it('should create logger with block ID context', () => {
            const blockLogger = logger.withBlockID('test-block-123');
            blockLogger.info('Test message');
            expect(console.log).toHaveBeenCalledWith(expect.stringMatching(/.*"block_id":"test-block-123".*/));
        });
        it('should create logger with user ID context', () => {
            const userLogger = logger.withUserID('user-456');
            userLogger.info('Test message');
            expect(console.log).toHaveBeenCalledWith(expect.stringMatching(/.*"user_id":"user-456".*/));
        });
        it('should create logger with error context', () => {
            const testError = new Error('Test error');
            const errorLogger = logger.withError(testError);
            errorLogger.info('Test message');
            expect(console.log).toHaveBeenCalledWith(expect.stringMatching(/.*"stacktrace".*/));
        });
        it('should create logger with custom context', () => {
            const contextLogger = logger.withContext({
                custom_field: 'custom_value',
                number_field: 42
            });
            contextLogger.info('Test message');
            expect(console.log).toHaveBeenCalledWith(expect.stringMatching(/.*"custom_field":"custom_value".*/));
            expect(console.log).toHaveBeenCalledWith(expect.stringMatching(/.*"number_field":42.*/));
        });
    });
    describe('formatting', () => {
        it('should format as JSON', () => {
            const jsonLogger = createLogger(LogLevel.INFO, LogFormat.JSON);
            jsonLogger.info('Test message');
            expect(console.log).toHaveBeenCalledWith(expect.stringMatching(/^\{.*\}$/));
        });
        it('should format as console', () => {
            const consoleLogger = createLogger(LogLevel.INFO, LogFormat.CONSOLE);
            consoleLogger.info('Test message');
            expect(console.log).toHaveBeenCalledWith(expect.stringMatching(/INFO.*Test message/));
        });
    });
    describe('getters', () => {
        it('should return current options', () => {
            const options = logger.getOptions();
            expect(options.level).toBe(LogLevel.INFO);
            expect(options.format).toBe(LogFormat.JSON);
        });
        it('should return current context', () => {
            const context = logger.getContext();
            expect(context).toEqual({});
        });
    });
});
//# sourceMappingURL=logging.test.js.map