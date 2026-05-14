/**
 * WASM Test Utilities
 * Provides utilities for loading real WASM files in tests
 */

import * as fs from 'fs';
import * as path from 'path';

export interface TestWasmFile {
  name: string;
  path: string;
  size: number;
  type: string;
  arrayBuffer(): Promise<ArrayBuffer>;
  text(): Promise<string>;
}

/**
 * Available WASM files for testing
 */
export const AVAILABLE_WASM_FILES = {
  KNIRV_CONTROLLER: 'build/knirv-controller.wasm',
  KNIRV_CONTROLLER_DEBUG: 'build/knirv-controller-debug.wasm',
  KNIRV_CORTEX: 'rust-wasm/target/wasm32-unknown-unknown/release/knirv_cortex_wasm.wasm',
  KNIRV_CORTEX_PKG: 'src/wasm-pkg/knirv_cortex_wasm_bg.wasm'
} as const;

/**
 * Creates a File-like object from a real WASM file
 */
export async function loadWasmAsFile(wasmPath: keyof typeof AVAILABLE_WASM_FILES, customName?: string): Promise<TestWasmFile> {
  const filePath = path.join(process.cwd(), AVAILABLE_WASM_FILES[wasmPath]);
  
  if (!fs.existsSync(filePath)) {
    throw new Error(`WASM file not found: ${filePath}`);
  }

  const stats = fs.statSync(filePath);
  const buffer = fs.readFileSync(filePath);
  const fileName = customName || path.basename(filePath);

  return {
    name: fileName,
    path: filePath,
    size: stats.size,
    type: 'application/wasm',
    
    async arrayBuffer(): Promise<ArrayBuffer> {
      return buffer.buffer.slice(buffer.byteOffset, buffer.byteOffset + buffer.byteLength);
    },
    
    async text(): Promise<string> {
      // WASM files are binary, but we can return a base64 representation
      return buffer.toString('base64');
    }
  };
}

/**
 * Creates a mock LoRA file for testing
 */
export function createMockLoraFile(name: string = 'test-lora.json'): TestWasmFile {
  const loraData = {
    model: 'test-model',
    adapter: {
      type: 'lora',
      rank: 16,
      alpha: 32,
      dropout: 0.1
    },
    training: {
      epochs: 10,
      learning_rate: 0.001,
      batch_size: 4
    },
    metadata: {
      created: new Date().toISOString(),
      version: '1.0.0'
    }
  };

  const content = JSON.stringify(loraData, null, 2);
  const buffer = Buffer.from(content, 'utf-8');

  return {
    name,
    path: '',
    size: buffer.length,
    type: 'application/json',
    
    async arrayBuffer(): Promise<ArrayBuffer> {
      return buffer.buffer.slice(buffer.byteOffset, buffer.byteOffset + buffer.byteLength);
    },
    
    async text(): Promise<string> {
      return content;
    }
  };
}

/**
 * Creates a test agent file with specific requirements
 */
export async function createTestAgentFile(
  type: 'wasm' | 'lora',
  requirements?: {
    memory?: number;
    cpu?: number;
    storage?: number;
  },
  customName?: string
): Promise<TestWasmFile> {
  if (type === 'wasm') {
    // Use a real WASM file
    return loadWasmAsFile('KNIRV_CONTROLLER_DEBUG', customName);
  } else {
    // Create a LoRA file with specific requirements
    const loraData = {
      model: 'test-model',
      adapter: {
        type: 'lora',
        rank: 16,
        alpha: 32
      },
      requirements: requirements || {
        memory: 64,
        cpu: 1,
        storage: 10
      },
      metadata: {
        created: new Date().toISOString(),
        version: '1.0.0'
      }
    };

    const content = JSON.stringify(loraData, null, 2);
    const buffer = Buffer.from(content, 'utf-8');

    return {
      name: customName || 'test-lora.json',
      path: '',
      size: buffer.length,
      type: 'application/json',
      
      async arrayBuffer(): Promise<ArrayBuffer> {
        return buffer.buffer.slice(buffer.byteOffset, buffer.byteOffset + buffer.byteLength);
      },
      
      async text(): Promise<string> {
        return content;
      }
    };
  }
}

/**
 * Validates that a WASM file is properly formatted
 */
export async function validateWasmFile(file: TestWasmFile): Promise<boolean> {
  try {
    const buffer = await file.arrayBuffer();
    const view = new Uint8Array(buffer);
    
    // Check WASM magic number (0x00 0x61 0x73 0x6D)
    return (
      view[0] === 0x00 &&
      view[1] === 0x61 &&
      view[2] === 0x73 &&
      view[3] === 0x6D
    );
  } catch (error) {
    console.error('WASM module validation failed:', error);
    return false;
  }
}

/**
 * Gets file size in a human-readable format
 */
export function formatFileSize(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB'];
  let size = bytes;
  let unitIndex = 0;
  
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  
  return `${size.toFixed(1)} ${units[unitIndex]}`;
}

/**
 * Lists all available WASM files with their info
 */
export async function listAvailableWasmFiles(): Promise<Array<{
  key: string;
  path: string;
  exists: boolean;
  size?: number;
  formattedSize?: string;
}>> {
  const results = [];
  
  for (const [key, relativePath] of Object.entries(AVAILABLE_WASM_FILES)) {
    const fullPath = path.join(process.cwd(), relativePath);
    const exists = fs.existsSync(fullPath);
    
    let size: number | undefined;
    let formattedSize: string | undefined;
    
    if (exists) {
      const stats = fs.statSync(fullPath);
      size = stats.size;
      formattedSize = formatFileSize(size);
    }
    
    results.push({
      key,
      path: relativePath,
      exists,
      size,
      formattedSize
    });
  }
  
  return results;
}

/**
 * Creates a File object compatible with browser File API from TestWasmFile
 */
export function createFileFromTestWasm(testFile: TestWasmFile): File {
  // Create a mock File that's compatible with the browser File API
  const file = {
    name: testFile.name,
    size: testFile.size,
    type: testFile.type,
    lastModified: Date.now(),
    
    arrayBuffer: testFile.arrayBuffer.bind(testFile),
    text: testFile.text.bind(testFile),
    
    stream() {
      return new ReadableStream({
        async start(controller) {
          const buffer = await testFile.arrayBuffer();
          controller.enqueue(new Uint8Array(buffer));
          controller.close();
        }
      });
    },
    
    slice() {
      return this;
    }
  };
  
  return file as unknown as File;
}
