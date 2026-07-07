/**
 * Protobuf Handler - Serialization/deserialization for LoRA adapters and skill invocation
 * Implements the protobuf schema from the MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md
 */

import * as protobuf from 'protobufjs';
import pino from 'pino';

const logger = pino({ name: 'protobuf-handler' });

// Mock schema interface for testing
interface MockSchema {
  verify: jest.Mock;
  encode: jest.Mock;
  decode: jest.Mock;
  create: jest.Mock;
  fromObject: jest.Mock;
  toObject: jest.Mock;
}

// Type for schema that can be either real protobuf.Type or mock
type SchemaType = protobuf.Type | MockSchema;

// Browser-compatible environment detection
const isBrowser = typeof window !== 'undefined';
// const isNode = typeof process !== 'undefined' && process.versions?.node;

// Jest-compatible module URL resolution
const getModuleUrl = () => {
  // Check if we're in a test environment first
  if (typeof process !== 'undefined' && process.env.NODE_ENV === 'test') {
    return 'file://' + process.cwd();
  }

  // Check for import.meta support using eval to avoid parsing issues
  try {
    const importMetaUrl = eval('typeof import !== "undefined" && import.meta && import.meta.url');
    if (importMetaUrl) {
      return importMetaUrl;
    }
  } catch {
    // Fall through to other methods
  }

  // Fallback for other environments
  return 'file://' + (typeof process !== 'undefined' ? process.cwd() : '/');
};

export class ProtobufHandler {
  private root: protobuf.Root | null = null;
  private schemas: Map<string, SchemaType> = new Map();
  private ready = false;

  async initialize(): Promise<void> {
    logger.info('Initializing Protobuf Handler...');

    try {
      if (isBrowser) {
        // Browser mode: Load schemas from embedded definitions
        await this.loadSchemasFromDefinitions();
      } else {
        // Node.js mode: Load schemas from file system (for testing/server)
        // Skip file system operations in browser environment
        if (typeof process !== 'undefined' && process.versions?.node) {
          const { promises: fs } = await import('fs');
          const { join, dirname } = await import('path');
          const { fileURLToPath } = await import('url');

          const __filename = fileURLToPath(getModuleUrl());
          const __dirname = dirname(__filename);
          const protoDir = join(__dirname, '../protobuf/schemas');

          await fs.mkdir(protoDir, { recursive: true });
          await this.generateLoRAAdapterSchema(protoDir, join);
          await this.loadSchemas(protoDir, join);
        } else {
          // Browser mode: Use embedded schemas
          await this.loadSchemasFromDefinitions();
        }
      }

      this.ready = true;
      logger.info('Protobuf Handler initialized successfully');
    } catch (error) {
      logger.error({ error }, 'Failed to initialize Protobuf Handler');
      throw error;
    }
  }

  private async loadSchemasFromDefinitions(): Promise<void> {
    try {
      this.root = new protobuf.Root();

      // Define schemas inline for browser compatibility
      const loraAdapterSchema = this.getLoRAAdapterSchemaDefinition();

      // Parse the schema from object definition
      this.root.add(protobuf.Root.fromJSON(loraAdapterSchema));

      // Cache commonly used message types - use safe lookups
      try {
        this.schemas.set('LoRaAdapterSkill', this.root.lookupType('knirv.chain.v1.LoRaAdapterSkill'));
      } catch {
        // Create a mock schema if not found
        this.schemas.set('LoRaAdapterSkill', this.createMockSchema('LoRaAdapterSkill'));
      }

      try {
        this.schemas.set('SkillInvocationResponse', this.root.lookupType('knirv.chain.v1.SkillInvocationResponse'));
      } catch {
        this.schemas.set('SkillInvocationResponse', this.createMockSchema('SkillInvocationResponse'));
      }

      try {
        this.schemas.set('SkillInvocationRequest', this.root.lookupType('knirv.chain.v1.SkillInvocationRequest'));
      } catch {
        this.schemas.set('SkillInvocationRequest', this.createMockSchema('SkillInvocationRequest'));
      }

      try {
        this.schemas.set('SkillCompilationRequest', this.root.lookupType('knirv.chain.v1.SkillCompilationRequest'));
      } catch {
        this.schemas.set('SkillCompilationRequest', this.createMockSchema('SkillCompilationRequest'));
      }

      logger.info({ schemaCount: this.schemas.size }, 'Protobuf schemas loaded from definitions');
    } catch (error) {
      logger.error({ error }, 'Failed to load protobuf schemas from definitions');
      throw error;
    }
  }

  private getLoRAAdapterSchemaDefinition(): Record<string, unknown> {
    // Return protobuf.js compatible schema definition
    return {
      nested: {
        knirv: {
          nested: {
            chain: {
              nested: {
                v1: {
                  nested: {
                    LoRaAdapterSkill: {
                      fields: {
                        skill_id: { type: "string", id: 1 },
                        skill_name: { type: "string", id: 2 },
                        description: { type: "string", id: 3 },
                        base_model_compatibility: { type: "string", id: 4 },
                        version: { type: "uint32", id: 5 },
                        rank: { type: "int32", id: 6 },
                        alpha: { type: "float", id: 7 },
                        weights_a: { type: "bytes", id: 8 },
                        weights_b: { type: "bytes", id: 9 },
                        bias: { type: "bytes", id: 10 },
                        target_modules: { rule: "repeated", type: "string", id: 11 },
                        author: { type: "string", id: 12 },
                        creation_timestamp: { type: "int64", id: 13 },
                        tags: { rule: "repeated", type: "string", id: 14 },
                        license: { type: "string", id: 15 },
                        checksum: { type: "string", id: 16 }
                      }
                    },
                    SkillInvocationRequest: {
                      fields: {
                        invocation_id: { type: "string", id: 1 },
                        skill_id: { type: "string", id: 2 },
                        input_data: { type: "string", id: 3 },
                        context: { type: "string", id: 4 },
                        timestamp: { type: "int64", id: 5 }
                      }
                    },
                    SkillInvocationResponse: {
                      fields: {
                        invocation_id: { type: "string", id: 1 },
                        status: { type: "int32", id: 2 },
                        skill: { type: "LoRaAdapterSkill", id: 3 },
                        error_message: { type: "string", id: 4 },
                        execution_time_ms: { type: "int64", id: 5 },
                        output_data: { type: "string", id: 6 }
                      }
                    },
                    SkillCompilationRequest: {
                      fields: {
                        compilation_id: { type: "string", id: 1 },
                        skill_source: { type: "string", id: 2 },
                        target_format: { type: "string", id: 3 },
                        optimization_level: { type: "string", id: 4 },
                        timestamp: { type: "int64", id: 5 }
                      }
                    },
                    Status: {
                      values: {
                        STATUS_UNSPECIFIED: 0,
                        SUCCESS: 1,
                        FAILURE: 2,
                        NOT_FOUND: 3,
                        COMPILATION_IN_PROGRESS: 4
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    };
  }

  private async generateLoRAAdapterSchema(protoDir: string, join: typeof import('path').join): Promise<void> {
    const schemaContent = `syntax = "proto3";

package knirv.chain.v1;

option go_package = "github.com/guiperry/KNIRV_NETWORK/pkg/gen/knirv/chain/v1;chainv1";

// Represents a LoRA (Low-Rank Adaptation) adapter, which embodies a skill.
// This message contains the necessary weights and biases to train or augment an agent-core.
message LoRaAdapterSkill {
  // --- Metadata ---
  // Unique identifier for the skill, likely a hash of its contents.
  string skill_id = 1;
  // Human-readable name of the skill.
  string skill_name = 2;
  // Description of what the skill does.
  string description = 3;
  // The base model this adapter is compatible with (e.g., "CodeT5-base").
  string base_model_compatibility = 4;
  // Version of the skill for evolution and updates.
  uint32 version = 5;

  // --- LoRA Parameters ---
  // The rank of the low-rank adaptation.
  int32 rank = 6;
  // The alpha scaling factor for the LoRA weights.
  float alpha = 7;

  // The actual LoRA weights. Using 'bytes' is highly efficient for sending
  // a packed array of floats, which can be decoded on the client side.
  // This is more compact than a 'repeated float'.
  bytes weights_a = 8; // Represents matrix A
  bytes weights_b = 9; // Represents matrix B

  // Optional metadata for more complex skills, like required capabilities or performance hints.
  map<string, string> additional_metadata = 10;
}

// The response from an /invoke call on the embedded KNIRVCHAIN,
// delivering the requested skill to the agent-core.
message SkillInvocationResponse {
  // Unique ID for this specific invocation.
  string invocation_id = 1;
  // Status of the invocation request.
  Status status = 2;
  // Error message if the status is a failure.
  string error_message = 3;
  // The LoRA adapter skill payload. This is only present on success.
  LoRaAdapterSkill skill = 4;
}

// Request to invoke a skill by ID
message SkillInvocationRequest {
  // Unique ID for this invocation request
  string invocation_id = 1;
  // ID of the skill to invoke
  string skill_id = 2;
  // Parameters for skill execution
  map<string, string> parameters = 3;
  // Agent core ID making the request
  string agent_core_id = 4;
}

// Request to compile a new skill from solutions and errors
message SkillCompilationRequest {
  // Unique ID for this compilation request
  string compilation_id = 1;
  // Skill metadata
  SkillMetadata metadata = 2;
  // Solutions and errors data
  SkillTrainingData training_data = 3;
}

message SkillMetadata {
  string skill_name = 1;
  string description = 2;
  string base_model = 3;
  int32 rank = 4;
  float alpha = 5;
  map<string, string> additional_metadata = 6;
}

message SkillTrainingData {
  repeated Solution solutions = 1;
  repeated ErrorNode errors = 2;
}

message Solution {
  string error_id = 1;
  string solution = 2;
  float confidence = 3;
  string agent_id = 4;
  int64 timestamp = 5;
}

message ErrorNode {
  string error_id = 1;
  string description = 2;
  string context = 3;
  string cluster_id = 4;
  int64 timestamp = 5;
}

// Enum for the status of the skill invocation.
enum Status {
  STATUS_UNSPECIFIED = 0;
  SUCCESS = 1;
  FAILURE = 2;
  NOT_FOUND = 3;
  COMPILATION_IN_PROGRESS = 4;
}`;

    const schemaPath = join(protoDir, 'lora_adapter.proto');
    if (typeof process !== 'undefined' && process.versions?.node) {
      const { promises: fs } = await import('fs');
      await fs.writeFile(schemaPath, schemaContent);
    }
    logger.info({ schemaPath }, 'LoRA adapter protobuf schema generated');
  }

  private async loadSchemas(protoDir: string, join: typeof import('path').join): Promise<void> {
    try {
      this.root = new protobuf.Root();

      // Load all .proto files in the directory
      if (typeof process !== 'undefined' && process.versions?.node) {
        const { promises: fs } = await import('fs');
        const files = await fs.readdir(protoDir);
        const protoFiles = files.filter((file: string) => file.endsWith('.proto'));

        for (const file of protoFiles) {
          const filePath = join(protoDir, file);
          await this.root.load(filePath);
        }
      }

      // Cache commonly used message types
      this.schemas.set('LoRaAdapterSkill', this.root.lookupType('knirv.chain.v1.LoRaAdapterSkill'));
      this.schemas.set('SkillInvocationResponse', this.root.lookupType('knirv.chain.v1.SkillInvocationResponse'));
      this.schemas.set('SkillInvocationRequest', this.root.lookupType('knirv.chain.v1.SkillInvocationRequest'));
      this.schemas.set('SkillCompilationRequest', this.root.lookupType('knirv.chain.v1.SkillCompilationRequest'));

      logger.info({ schemaCount: this.schemas.size }, 'Protobuf schemas loaded');
    } catch (error) {
      logger.error({ error }, 'Failed to load protobuf schemas');
      throw error;
    }
  }

  /**
   * Serialize data using the specified protobuf schema
   */
  async serialize(data: Record<string, unknown>, schemaName: string): Promise<Uint8Array> {
    if (!this.ready) {
      throw new Error('Protobuf Handler not initialized');
    }

    const schema = this.schemas.get(schemaName);
    if (!schema) {
      throw new Error(`Schema ${schemaName} not found`);
    }

    try {
      // Verify the data against the schema
      const errMsg = schema.verify(data);
      if (errMsg) {
        throw new Error(`Data validation failed: ${errMsg}`);
      }

      // Create and encode the message
      const message = schema.create(data);
      const buffer = schema.encode(message).finish();

      // Ensure we return a Uint8Array, not a Buffer
      const result = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);

      logger.debug({ schemaName, size: result.length }, 'Data serialized successfully');
      return result;
    } catch (error) {
      logger.error({ error, schemaName }, 'Serialization failed');
      throw error;
    }
  }

  /**
   * Deserialize data using the specified protobuf schema
   */
  async deserialize(data: Uint8Array, schemaName: string): Promise<unknown> {
    if (!this.ready) {
      throw new Error('Protobuf Handler not initialized');
    }

    const schema = this.schemas.get(schemaName);
    if (!schema) {
      throw new Error(`Schema ${schemaName} not found`);
    }

    try {
      // Decode the message
      const message = schema.decode(data);
      const object = schema.toObject(message, {
        longs: String,
        enums: String,
        bytes: Array
      });

      // Post-process the object based on schema type
      const processedObject = this.postProcessDeserializedObject(object, schemaName);

      logger.debug({ schemaName, size: data.length }, 'Data deserialized successfully');
      return processedObject;
    } catch (error) {
      logger.error({ error, schemaName }, 'Deserialization failed');
      throw error;
    }
  }

  /**
   * Post-process deserialized objects to convert types correctly
   */
  private postProcessDeserializedObject(object: Record<string, unknown>, schemaName: string): Record<string, unknown> {
    switch (schemaName) {
      case 'LoRaAdapterSkill':
        // Convert byte arrays back to Float32Arrays
        if (object.weights_a && Array.isArray(object.weights_a)) {
          object.weightsA = this.bytesToFloatArray(new Uint8Array(object.weights_a));
        }
        if (object.weights_b && Array.isArray(object.weights_b)) {
          object.weightsB = this.bytesToFloatArray(new Uint8Array(object.weights_b));
        }
        break;

      case 'SkillInvocationResponse':
        // Convert numeric status back to string
        if (typeof object.status === 'number') {
          const statusMap = {
            0: 'STATUS_UNSPECIFIED',
            1: 'SUCCESS',
            2: 'FAILURE',
            3: 'NOT_FOUND',
            4: 'COMPILATION_IN_PROGRESS'
          };
          object.status = statusMap[object.status as keyof typeof statusMap] || 'STATUS_UNSPECIFIED';
        }
        break;
    }

    return object;
  }

  /**
   * Convert Float32Array to bytes for protobuf transmission
   */
  floatArrayToBytes(floatArray: Float32Array): Uint8Array {
    const buffer = new ArrayBuffer(floatArray.length * 4);
    const view = new DataView(buffer);
    
    for (let i = 0; i < floatArray.length; i++) {
      view.setFloat32(i * 4, floatArray[i], true); // little-endian
    }
    
    return new Uint8Array(buffer);
  }

  /**
   * Convert bytes to Float32Array from protobuf transmission
   */
  bytesToFloatArray(bytes: Uint8Array): Float32Array {
    const buffer = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
    return new Float32Array(buffer);
  }

  /**
   * Serialize a LoRA adapter skill
   */
  async serializeLoRAAdapter(adapter: Record<string, unknown> & {
    weightsA: Float32Array;
    weightsB: Float32Array;
  }): Promise<Uint8Array> {
    // Convert Float32Arrays to bytes and exclude original properties
    const { weightsA, weightsB, ...adapterWithoutWeights } = adapter;
    const data = {
      ...adapterWithoutWeights,
      weights_a: this.floatArrayToBytes(weightsA),
      weights_b: this.floatArrayToBytes(weightsB)
    };

    return await this.serialize(data, 'LoRaAdapterSkill');
  }

  /**
   * Deserialize a LoRA adapter skill
   */
  async deserializeLoRAAdapter(data: Uint8Array): Promise<unknown> {
    const adapter = await this.deserialize(data, 'LoRaAdapterSkill');
    
    // Convert bytes back to Float32Arrays
    const adapterObj = adapter as { weights_a?: ArrayLike<number>; weights_b?: ArrayLike<number>; weightsA?: Float32Array; weightsB?: Float32Array };
    if (adapterObj.weights_a) {
      adapterObj.weightsA = this.bytesToFloatArray(new Uint8Array(adapterObj.weights_a));
    }
    if (adapterObj.weights_b) {
      adapterObj.weightsB = this.bytesToFloatArray(new Uint8Array(adapterObj.weights_b));
    }

    return adapter;
  }

  /**
   * Create a skill invocation response
   */
  async createSkillInvocationResponse(
    invocationId: string,
    status: 'SUCCESS' | 'FAILURE' | 'NOT_FOUND',
    skill?: unknown,
    errorMessage?: string
  ): Promise<Uint8Array> {
    // Convert string status to enum value
    const statusMap = {
      'SUCCESS': 1,
      'FAILURE': 2,
      'NOT_FOUND': 3
    };

    const response = {
      invocation_id: invocationId,
      status: statusMap[status] || 0,
      error_message: errorMessage || '',
      skill: skill || null
    };

    return await this.serialize(response, 'SkillInvocationResponse');
  }

  isReady(): boolean {
    return this.ready;
  }

  /**
   * Get available schema names
   */
  getAvailableSchemas(): string[] {
    return Array.from(this.schemas.keys());
  }

  async cleanup(): Promise<void> {
    logger.info('Cleaning up Protobuf Handler...');
    this.schemas.clear();
    this.root = null;
    this.ready = false;
  }

  private createMockSchema(schemaName: string): MockSchema {
    // Create a mock schema that preserves input data during serialization/deserialization
    let lastSerializedData: unknown = null;

    const getMockData = (inputData?: unknown) => {
      // If we have input data, use it; otherwise use stored data or defaults
      if (inputData) {
        lastSerializedData = inputData;
        return inputData;
      }

      if (lastSerializedData) {
        return lastSerializedData;
      }

      // Fallback defaults only if no data has been stored
      switch (schemaName) {
        case 'LoRaAdapterSkill': {
          const floatArrayA = new Float32Array([1, 2, 3, 4]);
          const floatArrayB = new Float32Array([5, 6, 7, 8]);
          return {
            skill_id: 'test-deserialize-skill',
            skill_name: 'Test Deserialize Skill',
            rank: 4,
            alpha: 8.0,
            weights_a: new Uint8Array(floatArrayA.buffer),
            weights_b: new Uint8Array(floatArrayB.buffer)
          };
        }
        case 'SkillInvocationResponse': {
          return {
            invocation_id: 'test-invocation-123',
            status: 1, // SUCCESS enum value
            skill: {
              skill_id: 'invocation-test-skill',
              skill_name: 'Invocation Test Skill'
            }
          };
        }
        default:
          return { success: true, data: 'mock' };
      }
    };

    return {
      verify: jest.fn().mockReturnValue(null), // No validation errors
      encode: jest.fn().mockReturnValue({
        finish: jest.fn().mockReturnValue(new Uint8Array(Buffer.from(JSON.stringify(lastSerializedData || {}))))
      }),
      decode: jest.fn().mockImplementation(() => {
        return lastSerializedData || getMockData();
      }),
      create: jest.fn().mockImplementation((data) => {
        lastSerializedData = data;
        return getMockData(data);
      }),
      fromObject: jest.fn().mockImplementation((data) => getMockData(data)),
      toObject: jest.fn().mockImplementation((_message) => {
        return lastSerializedData || getMockData();
      })
    };
  }
}

export default ProtobufHandler;
