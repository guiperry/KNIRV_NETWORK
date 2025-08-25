/**
 * Protobuf Handler - Serialization/deserialization for LoRA adapters and skill invocation
 * Implements the protobuf schema from the MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md
 */

import protobuf from 'protobufjs';
import { promises as fs } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import pino from 'pino';

const logger = pino({ name: 'protobuf-handler' });

// Jest compatibility: handle import.meta.url fallback
let __filename: string;
let __dirname: string;

try {
  // ES module environment - use eval to avoid Jest parsing issues
  const importMeta = eval('import.meta');
  if (importMeta && importMeta.url) {
    __filename = fileURLToPath(importMeta.url);
    __dirname = dirname(__filename);
  } else {
    throw new Error('import.meta.url not available');
  }
} catch (error) {
  // CommonJS/Jest environment fallback
  __filename = require.resolve('./ProtobufHandler.ts');
  __dirname = dirname(__filename);
}

export class ProtobufHandler {
  private root: protobuf.Root | null = null;
  private schemas: Map<string, protobuf.Type> = new Map();
  private ready = false;

  async initialize(): Promise<void> {
    logger.info('Initializing Protobuf Handler...');

    try {
      // Create protobuf schema directory if it doesn't exist
      const protoDir = join(__dirname, '../protobuf/schemas');
      await fs.mkdir(protoDir, { recursive: true });

      // Generate the LoRA adapter protobuf schema
      await this.generateLoRAAdapterSchema(protoDir);

      // Load the protobuf schemas
      await this.loadSchemas(protoDir);

      this.ready = true;
      logger.info('Protobuf Handler initialized successfully');
    } catch (error) {
      logger.error({ error }, 'Failed to initialize Protobuf Handler');
      throw error;
    }
  }

  private async generateLoRAAdapterSchema(protoDir: string): Promise<void> {
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
    await fs.writeFile(schemaPath, schemaContent);
    logger.info({ schemaPath }, 'LoRA adapter protobuf schema generated');
  }

  private async loadSchemas(protoDir: string): Promise<void> {
    try {
      this.root = new protobuf.Root();
      
      // Load all .proto files in the directory
      const files = await fs.readdir(protoDir);
      const protoFiles = files.filter(file => file.endsWith('.proto'));

      for (const file of protoFiles) {
        const filePath = join(protoDir, file);
        await this.root.load(filePath);
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
  async serialize(data: any, schemaName: string): Promise<Uint8Array> {
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

      logger.debug({ schemaName, size: buffer.length }, 'Data serialized successfully');
      return buffer;
    } catch (error) {
      logger.error({ error, schemaName }, 'Serialization failed');
      throw error;
    }
  }

  /**
   * Deserialize data using the specified protobuf schema
   */
  async deserialize(data: Uint8Array, schemaName: string): Promise<any> {
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
        bytes: String
      });

      logger.debug({ schemaName, size: data.length }, 'Data deserialized successfully');
      return object;
    } catch (error) {
      logger.error({ error, schemaName }, 'Deserialization failed');
      throw error;
    }
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
  async serializeLoRAAdapter(adapter: any): Promise<Uint8Array> {
    // Convert Float32Arrays to bytes
    const data = {
      ...adapter,
      weights_a: this.floatArrayToBytes(adapter.weightsA),
      weights_b: this.floatArrayToBytes(adapter.weightsB)
    };

    // Remove the original Float32Array properties
    delete data.weightsA;
    delete data.weightsB;

    return await this.serialize(data, 'LoRaAdapterSkill');
  }

  /**
   * Deserialize a LoRA adapter skill
   */
  async deserializeLoRAAdapter(data: Uint8Array): Promise<any> {
    const adapter = await this.deserialize(data, 'LoRaAdapterSkill');
    
    // Convert bytes back to Float32Arrays
    adapter.weightsA = this.bytesToFloatArray(new Uint8Array(adapter.weights_a));
    adapter.weightsB = this.bytesToFloatArray(new Uint8Array(adapter.weights_b));

    return adapter;
  }

  /**
   * Create a skill invocation response
   */
  async createSkillInvocationResponse(
    invocationId: string,
    status: 'SUCCESS' | 'FAILURE' | 'NOT_FOUND',
    skill?: any,
    errorMessage?: string
  ): Promise<Uint8Array> {
    const response = {
      invocation_id: invocationId,
      status: status,
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
}

export default ProtobufHandler;
