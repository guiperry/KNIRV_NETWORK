// Mock for pino logger module
const mockLogger = {
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
  debug: jest.fn(),
  trace: jest.fn(),
  fatal: jest.fn(),
  child: jest.fn().mockReturnThis(),
  level: 'info',
  silent: false
};

module.exports = jest.fn(() => mockLogger);
module.exports.default = jest.fn(() => mockLogger);
module.exports.pino = jest.fn(() => mockLogger);

// Export the mock logger for direct access in tests
module.exports.mockLogger = mockLogger;
