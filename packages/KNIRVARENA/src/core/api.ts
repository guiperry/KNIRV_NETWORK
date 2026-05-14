// Unified Backend API - Frontend Module
import { loraEngine } from './loraEngine';
import { wasmCompiler } from './wasmCompiler';
import { protobufHandler } from './protobufHandler';

export class BackendAPI {
  constructor() {
    this.initialize();
  }
  
  private async initialize() {
    console.log('Backend API initialized (frontend mode)');
  }
  
  // LoRA endpoints
  async compileLora(config: unknown): Promise<{ adapterId: string }> {
    const adapterId = await loraEngine.compileAdapter(config);
    return { adapterId };
  }
  
  async invokeLora(adapterId: string, input: unknown): Promise<unknown> {
    return await loraEngine.invokeAdapter(adapterId, input);
  }
  
  async getLoraAdapters(): Promise<{ adapters: string[] }> {
    return { adapters: loraEngine.getAdapters() };
  }
  
  // WASM endpoints
  async compileWasm(sourceCode: string): Promise<{ success: boolean; wasmBytes?: Uint8Array }> {
    try {
      const wasmBytes = await wasmCompiler.compileRust(sourceCode);
      return { success: true, wasmBytes };
    } catch {
      return { success: false };
    }
  }
  
  async getWasmStatus(): Promise<{ available: boolean }> {
    return { available: wasmCompiler.isAvailable() };
  }
  
  // Protobuf endpoints
  async serializeProtobuf(schema: string, data: unknown): Promise<{ serialized: Uint8Array }> {
    const serialized = protobufHandler.serialize(schema, data);
    return { serialized };
  }
  
  async deserializeProtobuf(schema: string, data: Uint8Array): Promise<{ deserialized: unknown }> {
    const deserialized = protobufHandler.deserialize(schema, data);
    return { deserialized };
  }
  
  async getProtobufSchemas(): Promise<{ schemas: string[] }> {
    return { schemas: protobufHandler.getSchemas() };
  }
  
  // Health check
  async getHealth(): Promise<{ status: string; timestamp: string; components: unknown }> {
    return {
      status: 'healthy',
      timestamp: new Date().toISOString(),
      components: {
        loraEngine: 'healthy',
        wasmCompiler: 'healthy',
        protobufHandler: 'healthy'
      }
    };
  }
}

export const backendAPI = new BackendAPI();
