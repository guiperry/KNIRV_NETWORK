/**
 * Mock WASM module for Jest testing
 */

// Mock WASM exports
const mockWasmExports = {
  agentCoreExecute: jest.fn().mockResolvedValue('{"success": true, "result": "mock response"}'),
  agentCoreInitialize: jest.fn().mockResolvedValue(true),
  agentCoreDispose: jest.fn().mockResolvedValue(true),
  memory: {
    buffer: new ArrayBuffer(1024 * 1024), // 1MB mock memory
  }
};

// Mock WASM instance
const mockWasmInstance = {
  exports: mockWasmExports
};

// Mock WASM module
const mockWasmModule = {
  instantiate: jest.fn().mockResolvedValue(mockWasmInstance),
  compile: jest.fn().mockResolvedValue({}),
};

// Mock the default export (what would be imported from the WASM module)
module.exports = {
  default: mockWasmModule,
  ...mockWasmExports,
  __esModule: true
};
