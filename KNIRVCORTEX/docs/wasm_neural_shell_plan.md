# WASM Neural Shell Implementation Plan
## TinyLlama KNIRV Architecture

### Executive Summary

This plan implements a WebAssembly-based neural shell (`cortex.wasm`) that separates model logic from weights, following the KNIRV (Knowledge-Network-Intelligence-Retrieval-Virtualization) pattern. The shell orchestrates inference while all parameters remain on a secure Weight Server.

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  cortex.wasm    │◄───┤  Weight Server   │◄───┤  model.safetensors│
│  (cognitive     │    │  (ProtoBuf API)  │    │  (pure weights)   │
│   shell)        │    │                  │    │                   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

**Timeline:** 10-12 weeks | **Team Size:** 3-4 engineers | **Shell Size:** <2MB compressed

---
This comprehensive implementation plan provides a complete roadmap for building a WASM-based neural shell following the KNIRV pattern. Here are the key highlights:

## Architecture Benefits

The **KNIRV pattern** (Knowledge-Network-Intelligence-Retrieval-Virtualization) provides several crucial advantages:

1. **IP Protection**: Model weights never leave your servers - only the "cognitive shell" is distributed
2. **Tiny Footprint**: <2MB WASM shell vs. gigabytes of model weights  
3. **Dynamic Loading**: Fetch only needed parameters on-demand
4. **Version Control**: Update models without redistributing client code
5. **Compliance**: Easier to meet data sovereignty and licensing requirements

## Technical Innovations

**High-Performance Kernels**: Custom SIMD-optimized matrix multiplication, flash attention, and fused operators designed specifically for WebAssembly.

**Smart Memory Management**: Arena-based allocator with KV-cache growing from high memory and activations from low memory - no malloc during inference.

**Streaming Architecture**: Protobuf-based streaming of weight chunks prevents large tensor copies and enables progressive loading.

**Security by Design**: Ed25519 signed requests, time-limited tokens, CSP compliance, and zero model data logging.

## Production Readiness

The plan includes comprehensive testing, security auditing, CI/CD pipelines, monitoring, and deployment strategies. Performance targets ensure real-world usability:

- **8+ tokens/sec** on desktop browsers
- **4+ tokens/sec** on mobile devices  
- **<300ms** time-to-first-token
- **<256MB** total memory usage

## Implementation Path

The 12-week timeline with 8 parallel tracks allows a 3-4 person team to deliver a production-ready system. Each track has clear deliverables and can proceed independently while maintaining integration points.

This approach enables deploying powerful language models in browsers while maintaining complete control over your model assets - exactly what the KNIRV pattern was designed to achieve.

---

## Track 1: Architecture Specification & Parameter Mapping
**Duration:** 1 day | **Owner:** ML Engineer

### 1.1 Model Architecture Analysis
Based on TinyLlama specifications:
- **Architecture:** Llama-2 with optimizations
- **Parameters:** 1.1B total (compressed to ~38M effective)
- **Context Length:** 4,096 tokens
- **Vocabulary:** 32,000 tokens
- **Layers:** 22 transformer blocks
- **Model Dimension:** 2,048
- **Attention Heads:** 32 (4 KV heads for GQA)
- **FFN Multiplier:** 2.66 (SwiGLU activation)
- **Normalization:** RMSNorm (no bias except final classifier)

### 1.2 Parameter Manifest Generation
Create comprehensive parameter inventory:

```yaml
# parameter_manifest.yaml
model_config:
  vocab_size: 32000
  hidden_size: 2048
  num_layers: 22
  num_attention_heads: 32
  num_key_value_heads: 4
  intermediate_size: 5461  # int(2048 * 2.66)
  max_position_embeddings: 4096

parameter_map:
  embeddings:
    - name: "model.embed_tokens.weight"
      shape: [32000, 2048]
      dtype: "float32"
      size_mb: 250.0
      shard: "model-00001-of-00003.safetensors"
  
  layers:
    - layer_idx: 0-21
      patterns:
        - name: "model.layers.{i}.input_layernorm.weight"
          shape: [2048]
          dtype: "float32"
        - name: "model.layers.{i}.self_attn.q_proj.weight"
          shape: [2048, 2048]
          dtype: "float32"
        - name: "model.layers.{i}.self_attn.k_proj.weight"
          shape: [256, 2048]  # GQA: 4 heads * 64 dim
          dtype: "float32"
        - name: "model.layers.{i}.self_attn.v_proj.weight"
          shape: [256, 2048]
          dtype: "float32"
        - name: "model.layers.{i}.self_attn.o_proj.weight"
          shape: [2048, 2048]
          dtype: "float32"
        - name: "model.layers.{i}.post_attention_layernorm.weight"
          shape: [2048]
          dtype: "float32"
        - name: "model.layers.{i}.mlp.gate_proj.weight"
          shape: [5461, 2048]
          dtype: "float32"
        - name: "model.layers.{i}.mlp.up_proj.weight"
          shape: [5461, 2048]
          dtype: "float32"
        - name: "model.layers.{i}.mlp.down_proj.weight"
          shape: [2048, 5461]
          dtype: "float32"

  output:
    - name: "model.norm.weight"
      shape: [2048]
      dtype: "float32"
    - name: "lm_head.weight"
      shape: [32000, 2048]
      dtype: "float32"
      size_mb: 250.0
      shard: "model-00003-of-00003.safetensors"
```

---

## Track 2: Weight Server Implementation
**Duration:** 3 weeks | **Owner:** Backend Engineer | **Language:** Rust

### 2.1 Project Structure
```
weight-server/
├── Cargo.toml
├── proto/
│   └── weight_service.proto
├── src/
│   ├── main.rs
│   ├── server.rs
│   ├── storage.rs
│   ├── auth.rs
│   ├── compression.rs
│   └── proto.rs (generated)
├── docker/
│   └── Dockerfile
└── tests/
    ├── integration_tests.rs
    └── load_tests.rs
```

### 2.2 Protocol Buffer Definitions
```protobuf
// proto/weight_service.proto
syntax = "proto3";
package weight_service;

service WeightService {
  rpc GetTensor(TensorRequest) returns (stream TensorChunk);
  rpc GetModelInfo(ModelInfoRequest) returns (ModelInfoResponse);
  rpc GetTensorBatch(TensorBatchRequest) returns (stream TensorBatch);
}

message TensorRequest {
  string model_id = 1;        // e.g., "tinyllama-1.1b-v1"
  string shard_name = 2;      // e.g., "model-00001-of-00003.safetensors"
  string tensor_name = 3;     // e.g., "model.layers.0.self_attn.q_proj.weight"
  CompressionType compression = 4;
  QuantizationType quantization = 5;
}

message TensorChunk {
  bytes data = 1;
  int64 offset = 2;
  int64 total_size = 3;
  TensorMetadata metadata = 4;
}

message TensorMetadata {
  repeated int64 shape = 1;
  DataType dtype = 2;
  string name = 3;
  int64 total_elements = 4;
}

message TensorBatchRequest {
  string model_id = 1;
  repeated string tensor_names = 2;
  CompressionType compression = 3;
  QuantizationType quantization = 4;
}

message TensorBatch {
  string tensor_name = 1;
  bytes data = 2;
  TensorMetadata metadata = 3;
}

message ModelInfoRequest {
  string model_id = 1;
}

message ModelInfoResponse {
  string model_id = 1;
  repeated ShardInfo shards = 2;
  ModelConfig config = 3;
}

message ShardInfo {
  string name = 1;
  int64 size_bytes = 2;
  repeated string tensor_names = 3;
}

message ModelConfig {
  int32 vocab_size = 1;
  int32 hidden_size = 2;
  int32 num_layers = 3;
  int32 num_heads = 4;
  int32 num_kv_heads = 5;
  int32 max_seq_len = 6;
}

enum DataType {
  FLOAT32 = 0;
  FLOAT16 = 1;
  INT8 = 2;
  INT32 = 3;
}

enum CompressionType {
  NONE = 0;
  LZ4 = 1;
  ZSTD = 2;
}

enum QuantizationType {
  NO_QUANT = 0;
  Q4_0 = 1;
  Q8_0 = 2;
}
```

### 2.3 Core Server Implementation

**Cargo.toml:**
```toml
[package]
name = "weight-server"
version = "0.1.0"
edition = "2021"

[dependencies]
tokio = { version = "1.0", features = ["full"] }
tonic = "0.10"
prost = "0.12"
safetensors = "0.4"
memmap2 = "0.9"
lz4_flex = "0.11"
zstd = "0.13"
ed25519-dalek = "2.0"
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
clap = { version = "4.0", features = ["derive"] }
tracing = "0.1"
tracing-subscriber = "0.3"
bytes = "1.0"
futures = "0.3"

[build-dependencies]
tonic-build = "0.10"
```

**src/server.rs:**
```rust
use tonic::{transport::Server, Request, Response, Status, Streaming};
use crate::proto::weight_service_server::{WeightService, WeightServiceServer};
use crate::proto::*;
use crate::storage::SafeTensorStorage;
use crate::auth::AuthValidator;
use crate::compression::{compress_data, decompress_data};

pub struct WeightServer {
    storage: SafeTensorStorage,
    auth: AuthValidator,
}

impl WeightServer {
    pub fn new(model_path: String, auth_pubkey: String) -> Self {
        Self {
            storage: SafeTensorStorage::new(model_path),
            auth: AuthValidator::new(auth_pubkey),
        }
    }
}

#[tonic::async_trait]
impl WeightService for WeightServer {
    type GetTensorStream = tokio_stream::wrappers::ReceiverStream<
        Result<TensorChunk, Status>
    >;

    async fn get_tensor(
        &self,
        request: Request<TensorRequest>,
    ) -> Result<Response<Self::GetTensorStream>, Status> {
        // Validate authentication
        self.auth.validate_request(&request)?;
        
        let req = request.into_inner();
        let (tx, rx) = tokio::sync::mpsc::channel(128);

        let storage = self.storage.clone();
        tokio::spawn(async move {
            match storage.stream_tensor(&req.shard_name, &req.tensor_name).await {
                Ok(mut stream) => {
                    let mut offset = 0;
                    let total_size = stream.total_size();
                    
                    while let Some(chunk_result) = stream.next().await {
                        match chunk_result {
                            Ok(data) => {
                                let compressed_data = if req.compression != CompressionType::None as i32 {
                                    compress_data(&data, req.compression.into())?
                                } else {
                                    data
                                };

                                let chunk = TensorChunk {
                                    data: compressed_data,
                                    offset,
                                    total_size,
                                    metadata: Some(stream.metadata()),
                                };
                                
                                if tx.send(Ok(chunk)).await.is_err() {
                                    break;
                                }
                                offset += data.len() as i64;
                            }
                            Err(e) => {
                                let _ = tx.send(Err(Status::internal(e.to_string()))).await;
                                break;
                            }
                        }
                    }
                }
                Err(e) => {
                    let _ = tx.send(Err(Status::not_found(e.to_string()))).await;
                }
            }
        });

        Ok(Response::new(tokio_stream::wrappers::ReceiverStream::new(rx)))
    }

    async fn get_model_info(
        &self,
        request: Request<ModelInfoRequest>,
    ) -> Result<Response<ModelInfoResponse>, Status> {
        self.auth.validate_request(&request)?;
        
        let req = request.into_inner();
        let model_info = self.storage.get_model_info(&req.model_id).await
            .map_err(|e| Status::not_found(e.to_string()))?;

        Ok(Response::new(model_info))
    }
}
```

### 2.4 Storage Layer
**src/storage.rs:**
```rust
use memmap2::Mmap;
use safetensors::SafeTensors;
use std::collections::HashMap;
use std::path::PathBuf;
use tokio::sync::RwLock;

#[derive(Clone)]
pub struct SafeTensorStorage {
    model_path: PathBuf,
    shard_cache: Arc<RwLock<HashMap<String, Arc<SafeTensors<Mmap>>>>>,
}

impl SafeTensorStorage {
    pub fn new(model_path: String) -> Self {
        Self {
            model_path: PathBuf::from(model_path),
            shard_cache: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    pub async fn stream_tensor(
        &self,
        shard_name: &str,
        tensor_name: &str,
    ) -> Result<TensorStream, StorageError> {
        let safetensors = self.load_shard(shard_name).await?;
        let tensor_data = safetensors.tensor(tensor_name)
            .map_err(|_| StorageError::TensorNotFound(tensor_name.to_string()))?;

        Ok(TensorStream::new(tensor_data))
    }

    async fn load_shard(&self, shard_name: &str) -> Result<Arc<SafeTensors<Mmap>>, StorageError> {
        // Check cache first
        {
            let cache = self.shard_cache.read().await;
            if let Some(shard) = cache.get(shard_name) {
                return Ok(Arc::clone(shard));
            }
        }

        // Load shard from disk
        let shard_path = self.model_path.join(shard_name);
        let file = std::fs::File::open(&shard_path)
            .map_err(|e| StorageError::IoError(e))?;
        
        let mmap = unsafe { Mmap::map(&file) }
            .map_err(|e| StorageError::IoError(e))?;
        
        let safetensors = SafeTensors::deserialize(&mmap)
            .map_err(|e| StorageError::ParseError(e.to_string()))?;
        
        let shard = Arc::new(safetensors);

        // Cache the loaded shard
        {
            let mut cache = self.shard_cache.write().await;
            cache.insert(shard_name.to_string(), Arc::clone(&shard));
        }

        Ok(shard)
    }
}

pub struct TensorStream {
    data: Vec<u8>,
    chunk_size: usize,
    position: usize,
    metadata: TensorMetadata,
}

impl TensorStream {
    pub fn new(tensor_view: safetensors::tensor::TensorView) -> Self {
        let shape = tensor_view.shape().to_vec();
        let data = tensor_view.data().to_vec();
        
        Self {
            data,
            chunk_size: 64 * 1024, // 64KB chunks
            position: 0,
            metadata: TensorMetadata {
                shape: shape.iter().map(|&x| x as i64).collect(),
                dtype: DataType::Float32 as i32, // Assume f32 for now
                name: "".to_string(),
                total_elements: shape.iter().product::<usize>() as i64,
            },
        }
    }

    pub fn total_size(&self) -> i64 {
        self.data.len() as i64
    }

    pub fn metadata(&self) -> TensorMetadata {
        self.metadata.clone()
    }

    pub async fn next(&mut self) -> Option<Result<Vec<u8>, StorageError>> {
        if self.position >= self.data.len() {
            return None;
        }

        let end = std::cmp::min(self.position + self.chunk_size, self.data.len());
        let chunk = self.data[self.position..end].to_vec();
        self.position = end;

        Some(Ok(chunk))
    }
}
```

### 2.5 Authentication & Security
**src/auth.rs:**
```rust
use ed25519_dalek::{PublicKey, Signature, Verifier};
use tonic::{Request, Status};

pub struct AuthValidator {
    public_key: PublicKey,
}

impl AuthValidator {
    pub fn new(pubkey_hex: String) -> Self {
        let pubkey_bytes = hex::decode(pubkey_hex)
            .expect("Invalid public key format");
        let public_key = PublicKey::from_bytes(&pubkey_bytes)
            .expect("Invalid public key");
        
        Self { public_key }
    }

    pub fn validate_request<T>(&self, request: &Request<T>) -> Result<(), Status> {
        let signature_header = request
            .metadata()
            .get("x-ws-signature")
            .ok_or_else(|| Status::unauthenticated("Missing signature"))?
            .to_str()
            .map_err(|_| Status::unauthenticated("Invalid signature format"))?;

        let timestamp_header = request
            .metadata()
            .get("x-ws-timestamp")
            .ok_or_else(|| Status::unauthenticated("Missing timestamp"))?
            .to_str()
            .map_err(|_| Status::unauthenticated("Invalid timestamp format"))?;

        // Verify timestamp is within 15 minutes
        let timestamp: u64 = timestamp_header.parse()
            .map_err(|_| Status::unauthenticated("Invalid timestamp"))?;
        
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        if now.abs_diff(timestamp) > 900 { // 15 minutes
            return Err(Status::unauthenticated("Timestamp expired"));
        }

        // Verify signature
        let message = format!("{}:{}", request.uri().path(), timestamp);
        let signature_bytes = base64::decode(signature_header)
            .map_err(|_| Status::unauthenticated("Invalid signature encoding"))?;
        
        let signature = Signature::from_bytes(&signature_bytes)
            .map_err(|_| Status::unauthenticated("Invalid signature"))?;

        self.public_key
            .verify(message.as_bytes(), &signature)
            .map_err(|_| Status::unauthenticated("Signature verification failed"))?;

        Ok(())
    }
}
```

### 2.6 Deployment Configuration
**docker/Dockerfile:**
```dockerfile
FROM rust:1.75-slim as builder

WORKDIR /app
COPY . .
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/target/release/weight-server /usr/local/bin/weight-server

EXPOSE 7676
CMD ["weight-server", "--host", "0.0.0.0", "--port", "7676"]
```

**Configuration:**
```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 7676
  max_concurrent_streams: 100
  
models:
  - id: "tinyllama-1.1b-v1"
    path: "/models/tinyllama-1.1b"
    shards:
      - "model-00001-of-00003.safetensors"
      - "model-00002-of-00003.safetensors"  
      - "model-00003-of-00003.safetensors"

auth:
  public_key: "${WS_PUBLIC_KEY}"
  max_request_age_seconds: 900

compression:
  default: "zstd"
  level: 3

logging:
  level: "info"
  format: "json"
```

---

## Track 3: WASM Neural Shell (cortex.wasm)
**Duration:** 4 weeks | **Owner:** WebAssembly Engineer | **Language:** Rust

### 3.1 Project Structure
```
cortex/
├── Cargo.toml
├── build.rs
├── src/
│   ├── lib.rs              # WASM entry points
│   ├── model/
│   │   ├── mod.rs
│   │   ├── config.rs       # Model configuration
│   │   ├── layers.rs       # Transformer layers
│   │   └── attention.rs    # Attention implementation
│   ├── kernels/
│   │   ├── mod.rs
│   │   ├── matmul.rs       # Optimized matrix multiplication
│   │   ├── attention.rs    # Attention kernels
│   │   ├── activation.rs   # SwiGLU, RMSNorm
│   │   └── simd.rs         # SIMD utilities
│   ├── memory/
│   │   ├── mod.rs
│   │   ├── arena.rs        # Memory arena allocator
│   │   └── cache.rs        # KV cache management
│   ├── io/
│   │   ├── mod.rs
│   │   ├── fetcher.rs      # Weight fetching
│   │   └── proto.rs        # Generated protobuf code
│   ├── tensor/
│   │   ├── mod.rs
│   │   ├── tensor.rs       # Tensor abstraction
│   │   └── quantization.rs # Quantization support
│   └── utils/
│       ├── mod.rs
│       └── rope.rs         # RoPE position encoding
├── proto/
│   └── weight_service.proto
├── tests/
│   ├── integration/
│   └── unit/
└── benches/
    └── inference.rs
```

### 3.2 Cargo Configuration
**Cargo.toml:**
```toml
[package]
name = "cortex"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
wasm-bindgen = "0.2"
wasm-bindgen-futures = "0.4"
js-sys = "0.3"
web-sys = { version = "0.3", features = [
  "console",
  "Response",
  "Request",
  "Headers",
  "RequestInit",
  "ReadableStream",
  "ReadableStreamByobReader",
  "Uint8Array",
  "ArrayBuffer",
  "Worker",
  "MessageEvent",
  "WorkerGlobalScope"
]}
serde = { version = "1.0", features = ["derive"] }
serde-wasm-bindgen = "0.6"
prost = "0.12"
tokio = { version = "1.0", features = ["sync"] }
futures = "0.3"
half = "2.3"
bytemuck = { version = "1.14", features = ["derive"] }

[build-dependencies]
prost-build = "0.12"

[profile.release]
opt-level = "s"          # Optimize for size
lto = true               # Link time optimization
codegen-units = 1        # Better optimization
panic = "abort"          # Smaller binary
overflow-checks = false  # Performance

[profile.release.package."*"]
opt-level = "s"
```

### 3.3 Core WASM Interface
**src/lib.rs:**
```rust
use wasm_bindgen::prelude::*;
use crate::model::ModelConfig;
use crate::memory::MemoryArena;
use crate::io::WeightFetcher;

// Import console.log for debugging
#[wasm_bindgen]
extern "C" {
    #[wasm_bindgen(js_namespace = console)]
    fn log(s: &str);
}

macro_rules! console_log {
    ($($t:tt)*) => (log(&format_args!($($t)*).to_string()))
}

#[wasm_bindgen]
pub struct Cortex {
    config: ModelConfig,
    memory_arena: MemoryArena,
    weight_fetcher: WeightFetcher,
    kv_cache: KVCache,
    current_position: usize,
}

#[wasm_bindgen]
impl Cortex {
    #[wasm_bindgen(constructor)]
    pub fn new(
        weight_server_url: &str,
        auth_token: &str,
        cache_size_mb: u32,
    ) -> Result<Cortex, JsValue> {
        console_log!("Initializing Cortex with cache size: {}MB", cache_size_mb);

        let config = ModelConfig::tinyllama_1_1b();
        let memory_arena = MemoryArena::new((cache_size_mb as usize) * 1024 * 1024)?;
        let weight_fetcher = WeightFetcher::new(weight_server_url, auth_token)?;
        let kv_cache = KVCache::new(&config, &memory_arena)?;

        Ok(Cortex {
            config,
            memory_arena,
            weight_fetcher,
            kv_cache,
            current_position: 0,
        })
    }

    #[wasm_bindgen]
    pub async fn warmup(&mut self) -> Result<(), JsValue> {
        console_log!("Starting warmup - fetching small tensors...");
        
        // Fetch and cache small tensors (embeddings, norms, etc.)
        let critical_tensors = vec![
            "model.embed_tokens.weight",
            "model.norm.weight",
            "lm_head.weight",
        ];

        for tensor_name in critical_tensors {
            self.weight_fetcher.fetch_and_cache(tensor_name).await
                .map_err(|e| JsValue::from_str(&e.to_string()))?;
        }

        console_log!("Warmup complete");
        Ok(())
    }

    #[wasm_bindgen]
    pub async fn prefill(&mut self, prompt_tokens: &[u32]) -> Result<*const f32, JsValue> {
        console_log!("Starting prefill with {} tokens", prompt_tokens.len());
        
        let seq_len = prompt_tokens.len();
        self.current_position = seq_len;

        // Clear KV cache
        self.kv_cache.clear();

        // Run forward pass for all prompt tokens
        let logits = self.forward_batch(prompt_tokens, 0).await?;

        // Return pointer to the last token's logits
        let last_logits = &logits[(seq_len - 1) * self.config.vocab_size..];
        Ok(last_logits.as_ptr())
    }

    #[wasm_bindgen]
    pub async fn decode(&mut self, token: u32) -> Result<*const f32, JsValue> {
        let tokens = [token];
        let logits = self.forward_batch(&tokens, self.current_position).await?;
        self.current_position += 1;
        Ok(logits.as_ptr())
    }

    #[wasm_bindgen]
    pub fn get_vocab_size(&self) -> u32 {
        self.config.vocab_size as u32
    }

    #[wasm_bindgen]
    pub fn reset(&mut self) {
        self.current_position = 0;
        self.kv_cache.clear();
    }
}

impl Cortex {
    async fn forward_batch(
        &mut self,
        tokens: &[u32],
        start_pos: usize,
    ) -> Result<Vec<f32>, Box<dyn std::error::Error>> {
        let seq_len = tokens.len();
        let batch_size = 1;

        // Allocate activation tensors in arena
        let hidden_size = self.config.hidden_size;
        let mut hidden_states = self.memory_arena.allocate_tensor(&[batch_size, seq_len, hidden_size])?;

        // Embedding lookup
        self.embedding_forward(tokens, &mut hidden_states).await?;

        // Transformer layers
        for layer_idx in 0..self.config.num_layers {
            self.layer_forward(layer_idx, &mut hidden_states, start_pos).await?;
        }

        // Final norm and output projection
        self.output_forward(&hidden_states).await
    }

    async fn embedding_forward(
        &mut self,
        tokens: &[u32],
        hidden_states: &mut [f32],
    ) -> Result<(), Box<dyn std::error::Error>> {
        let embed_weight = self.weight_fetcher
            .get_tensor("model.embed_tokens.weight").await?;

        // Embedding lookup: hidden_states[i, :] = embed_weight[tokens[i], :]
        for (i, &token_id) in tokens.iter().enumerate() {
            let token_embed = &embed_weight.data()
                [(token_id as usize) * self.config.hidden_size..
                 (token_id as usize + 1) * self.config.hidden_size];
            
            let out_slice = &mut hidden_states
                [i * self.config.hidden_size..(i + 1) * self.config.hidden_size];
            
            out_slice.copy_from_slice(token_embed);
        }

        Ok(())
    }

    async fn layer_forward(
        &mut self,
        layer_idx: usize,
        hidden_states: &mut [f32],
        start_pos: usize,
    ) -> Result<(), Box<dyn std::error::Error>> {
        let seq_len = hidden_states.len() / self.config.hidden_size;
        
        // Pre-attention LayerNorm
        let norm_weight = self.weight_fetcher
            .get_tensor(&format!("model.layers.{}.input_layernorm.weight", layer_idx)).await?;
        
        let mut normed_hidden = self.memory_arena.allocate_tensor(&[1, seq_len, self.config.hidden_size])?;
        kernels::rms_norm(hidden_states, norm_weight.data(), &mut normed_hidden, 1e-6);

        // Self-attention
        let attention_output = self.attention_forward(layer_idx, &normed_hidden, start_pos).await?;
        
        // Residual connection
        for i in 0..hidden_states.len() {
            hidden_states[i] += attention_output[i];
        }

        // Post-attention LayerNorm
        let post_norm_weight = self.weight_fetcher
            .get_tensor(&format!("model.layers.{}.post_attention_layernorm.weight", layer_idx)).await?;
        
        kernels::rms_norm(hidden_states, post_norm_weight.data(), &mut normed_hidden, 1e-6);

        // MLP
        let mlp_output = self.mlp_forward(layer_idx, &normed_hidden).await?;
        
        // Residual connection
        for i in 0..hidden_states.len() {
            hidden_states[i] += mlp_output[i];
        }

        Ok(())
    }

    async fn attention_forward(
        &mut self,
        layer_idx: usize,
        hidden_states: &[f32],
        start_pos: usize,
    ) -> Result<Vec<f32>, Box<dyn std::error::Error>> {
        let seq_len = hidden_states.len() / self.config.hidden_size;
        let head_dim = self.config.hidden_size / self.config.num_heads;
        let kv_head_dim = self.config.hidden_size / self.config.num_kv_heads;

        // Load attention weights
        let q_weight = self.weight_fetcher
            .get_tensor(&format!("model.layers.{}.self_attn.q_proj.weight", layer_idx)).await?;
        let k_weight = self.weight_fetcher
            .get_tensor(&format!("model.layers.{}.self_attn.k_proj.weight", layer_idx)).await?;
        let v_weight = self.weight_fetcher
            .get_tensor(&format!("model.layers.{}.self_attn.v_proj.weight", layer_idx)).await?;
        let o_weight = self.weight_fetcher
            .get_tensor(&format!("model.layers.{}.self_attn.o_proj.weight", layer_idx)).await?;

        // Q, K, V projections
        let mut q_states = self.memory_arena.allocate_tensor(&[seq_len, self.config.hidden_size])?;
        let mut k_states = self.memory_arena.allocate_tensor(&[seq_len, kv_head_dim * self.config.num_kv_heads])?;
        let mut v_states = self.memory_arena.allocate_tensor(&[seq_len, kv_head_dim * self.config.num_kv_heads])?;

        kernels::matmul(hidden_states, q_weight.data(), &mut q_states, seq_len, self.config.hidden_size, self.config.hidden_size);
        kernels::matmul(hidden_states, k_weight.data(), &mut k_states, seq_len, self.config.hidden_size, kv_head_dim * self.config.num_kv_heads);
        kernels::matmul(hidden_states, v_weight.data(), &mut v_states, seq_len, self.config.hidden_size, kv_head_dim * self.config.num_kv_heads);

        // Apply RoPE to Q and K
        kernels::apply_rope(&mut q_states, &mut k_states, start_pos, head_dim, seq_len, self.config.num_heads, self.config.num_kv_heads);

        // Update KV cache
        self.kv_cache.update(layer_idx, &k_states, &v_states, start_pos, seq_len)?;

        // Get full K, V from cache
        let (cached_k, cached_v) = self.kv_cache.get(layer_idx, start_pos + seq_len)?;

        // Multi-head attention with GQA
        let mut attn_output = self.memory_arena.allocate_tensor(&[seq_len, self.config.hidden_size])?;
        kernels::grouped_query_attention(
            &q_states,
            &cached_k,
            &cached_v,
            &mut attn_output,
            seq_len,
            start_pos + seq_len,
            self.config.num_heads,
            self.config.num_kv_heads,
            head_dim,
        );

        // Output projection
        let mut output = vec![0.0f32; seq_len * self.config.hidden_size];
        kernels::matmul(&attn_output, o_weight.data(), &mut output, seq_len, self.config.hidden_size, self.config.hidden_size);

        Ok(output)
    }

    async fn mlp_forward(
        &mut self,
        layer_idx: usize,
        hidden_states: &[f32],
    ) -> Result<Vec<f32>, Box<dyn std::error::Error>> {
        let seq_len = hidden_states.len() / self.config.hidden_size;
        let intermediate_size = self.config.intermediate_size;

        // Load MLP weights
        let gate_weight = self.weight_fetcher
            .get_tensor(&format!("model.layers.{}.mlp.gate_proj.weight", layer_idx)).await?;
        let up_weight = self.weight_fetcher
            .get_tensor(&format!("model.layers.{}.mlp.up_proj.weight", layer_idx)).await?;
        let down_weight = self.weight_fetcher
            .get_tensor(&format!("model.layers.{}.mlp.down_proj.weight", layer_idx)).await?;

        // Gate and Up projections
        let mut gate_output = self.memory_arena.allocate_tensor(&[seq_len, intermediate_size])?;
        let mut up_output = self.memory_arena.allocate_tensor(&[seq_len, intermediate_size])?;

        kernels::matmul(hidden_states, gate_weight.data(), &mut gate_output, seq_len, self.config.hidden_size, intermediate_size);
        kernels::matmul(hidden_states, up_weight.data(), &mut up_output, seq_len, self.config.hidden_size, intermediate_size);

        // SwiGLU activation
        kernels::swiglu_inplace(&mut gate_output, &up_output);

        // Down projection
        let mut output = vec![0.0f32; seq_len * self.config.hidden_size];
        kernels::matmul(&gate_output, down_weight.data(), &mut output, seq_len, intermediate_size, self.config.hidden_size);

        Ok(output)
    }

    async fn output_forward(&mut self, hidden_states: &[f32]) -> Result<Vec<f32>, Box<dyn std::error::Error>> {
        let seq_len = hidden_states.len() / self.config.hidden_size;

        // Final LayerNorm
        let norm_weight = self.weight_fetcher.get_tensor("model.norm.weight").await?;
        let mut normed_hidden = self.memory_arena.allocate_tensor(&[seq_len, self.config.hidden_size])?;
        kernels::rms_norm(hidden_states, norm_weight.data(), &mut normed_hidden, 1e-6);

        // Output projection (language model head)
        let lm_head_weight = self.weight_fetcher.get_tensor("lm_head.weight").await?;
        let mut logits = vec![0.0f32; seq_len * self.config.vocab_size];
        kernels::matmul(&normed_hidden, lm_head_weight.data(), &mut logits, seq_len, self.config.hidden_size, self.config.vocab_size);

        Ok(logits)
    }
}
```

### 3.4 High-Performance Kernels
**src/kernels/matmul.rs:**
```rust
use std::arch::wasm32::*;

pub fn matmul(
    a: &[f32],    // [M, K]
    b: &[f32],    // [K, N] 
    c: &mut [f32], // [M, N]
    m: usize,
    k: usize, 
    n: usize,
) {
    // Tiled matrix multiplication with SIMD
    const TILE_SIZE: usize = 8;
    
    for i in (0..m).step_by(TILE_SIZE) {
        for j in (0..n).step_by(TILE_SIZE) {
            for kk in (0..k).step_by(TILE_SIZE) {
                let i_max = (i + TILE_SIZE).min(m);
                let j_max = (j + TILE_SIZE).min(n);
                let kk_max = (kk + TILE_SIZE).min(k);
                
                matmul_tile(a, b, c, i, j, kk, i_max, j_max, kk_max, k, n);
            }
        }
    }
}

#[inline(always)]
fn matmul_tile(
    a: &[f32],
    b: &[f32], 
    c: &mut [f32],
    i_start: usize,
    j_start: usize,
    k_start: usize,
    i_end: usize,
    j_end: usize,
    k_end: usize,
    k_stride: usize,
    n_stride: usize,
) {
    for i in i_start..i_end {
        for j in (j_start..j_end).step_by(4) {
            let mut sum = unsafe { f32x4_splat(0.0) };
            
            for kk in k_start..k_end {
                let a_val = unsafe { f32x4_splat(a[i * k_stride + kk]) };
                let b_vals = unsafe {
                    v128_load(&b[(kk * n_stride + j) * 4] as *const f32 as *const v128)
                };
                sum = unsafe { f32x4_add(sum, f32x4_mul(a_val, b_vals)) };
            }
            
            let c_idx = i * n_stride + j;
            let existing = unsafe {
                v128_load(&c[c_idx] as *const f32 as *const v128)
            };
            let result = unsafe { f32x4_add(existing, sum) };
            unsafe {
                v128_store(&mut c[c_idx] as *mut f32 as *mut v128, result);
            }
        }
    }
}

// Fallback scalar implementation
fn matmul_scalar(a: &[f32], b: &[f32], c: &mut [f32], m: usize, k: usize, n: usize) {
    for i in 0..m {
        for j in 0..n {
            let mut sum = 0.0f32;
            for kk in 0..k {
                sum += a[i * k + kk] * b[kk * n + j];
            }
            c[i * n + j] += sum;
        }
    }
}
```

**src/kernels/attention.rs:**
```rust
use crate::kernels::matmul::matmul;

pub fn grouped_query_attention(
    q: &[f32],        // [seq_len, num_heads * head_dim]
    k: &[f32],        // [cache_len, num_kv_heads * head_dim] 
    v: &[f32],        // [cache_len, num_kv_heads * head_dim]
    output: &mut [f32], // [seq_len, num_heads * head_dim]
    seq_len: usize,
    cache_len: usize,
    num_heads: usize,
    num_kv_heads: usize,
    head_dim: usize,
) {
    let group_size = num_heads / num_kv_heads;
    let scale = 1.0 / (head_dim as f32).sqrt();
    
    for h in 0..num_heads {
        let kv_h = h / group_size; // Which KV head this query head uses
        
        for seq_i in 0..seq_len {
            let q_offset = seq_i * num_heads * head_dim + h * head_dim;
            let q_head = &q[q_offset..q_offset + head_dim];
            
            // Compute attention scores
            let mut scores = vec![0.0f32; cache_len];
            for cache_i in 0..cache_len {
                let k_offset = cache_i * num_kv_heads * head_dim + kv_h * head_dim;
                let k_head = &k[k_offset..k_offset + head_dim];
                
                let mut score = 0.0f32;
                for d in 0..head_dim {
                    score += q_head[d] * k_head[d];
                }
                scores[cache_i] = score * scale;
            }
            
            // Apply causal mask (can only attend to previous positions)
            let current_pos = cache_len - seq_len + seq_i;
            for i in (current_pos + 1)..cache_len {
                scores[i] = f32::NEG_INFINITY;
            }
            
            // Softmax
            let max_score = scores.iter().fold(f32::NEG_INFINITY, |a, &b| a.max(b));
            let mut exp_sum = 0.0f32;
            for score in &mut scores {
                *score = (*score - max_score).exp();
                exp_sum += *score;
            }
            for score in &mut scores {
                *score /= exp_sum;
            }
            
            // Weighted sum of values
            let out_offset = seq_i * num_heads * head_dim + h * head_dim;
            let out_head = &mut output[out_offset..out_offset + head_dim];
            
            for d in 0..head_dim {
                let mut weighted_sum = 0.0f32;
                for cache_i in 0..cache_len {
                    let v_offset = cache_i * num_kv_heads * head_dim + kv_h * head_dim;
                    weighted_sum += scores[cache_i] * v[v_offset + d];
                }
                out_head[d] = weighted_sum;
            }
        }
    }
}

pub fn flash_attention(
    q: &[f32],
    k: &[f32], 
    v: &[f32],
    output: &mut [f32],
    seq_len: usize,
    num_heads: usize,
    head_dim: usize,
) {
    // Flash attention implementation for memory efficiency
    const BLOCK_SIZE: usize = 64;
    let scale = 1.0 / (head_dim as f32).sqrt();
    
    for h in 0..num_heads {
        for i in (0..seq_len).step_by(BLOCK_SIZE) {
            let block_end = (i + BLOCK_SIZE).min(seq_len);
            
            // Process block of queries
            for qi in i..block_end {
                let q_offset = qi * num_heads * head_dim + h * head_dim;
                let q_head = &q[q_offset..q_offset + head_dim];
                
                let mut row_max = f32::NEG_INFINITY;
                let mut row_sum = 0.0f32;
                let mut output_acc = vec![0.0f32; head_dim];
                
                // Process in blocks of keys/values
                for j in (0..=qi).step_by(BLOCK_SIZE) {
                    let kv_block_end = (j + BLOCK_SIZE).min(qi + 1);
                    
                    // Compute scores for this block
                    let mut block_scores = vec![0.0f32; kv_block_end - j];
                    let mut block_max = f32::NEG_INFINITY;
                    
                    for (idx, kj) in (j..kv_block_end).enumerate() {
                        let k_offset = kj * num_heads * head_dim + h * head_dim;
                        let k_head = &k[k_offset..k_offset + head_dim];
                        
                        let mut score = 0.0f32;
                        for d in 0..head_dim {
                            score += q_head[d] * k_head[d];
                        }
                        score *= scale;
                        block_scores[idx] = score;
                        block_max = block_max.max(score);
                    }
                    
                    // Online softmax update
                    let new_max = row_max.max(block_max);
                    let exp_diff = (row_max - new_max).exp();
                    
                    // Update previous accumulator
                    for d in 0..head_dim {
                        output_acc[d] *= exp_diff;
                    }
                    row_sum *= exp_diff;
                    
                    // Process this block
                    for (idx, kj) in (j..kv_block_end).enumerate() {
                        let prob = (block_scores[idx] - new_max).exp();
                        row_sum += prob;
                        
                        let v_offset = kj * num_heads * head_dim + h * head_dim;
                        let v_head = &v[v_offset..v_offset + head_dim];
                        
                        for d in 0..head_dim {
                            output_acc[d] += prob * v_head[d];
                        }
                    }
                    
                    row_max = new_max;
                }
                
                // Normalize and write output
                let out_offset = qi * num_heads * head_dim + h * head_dim;
                let out_head = &mut output[out_offset..out_offset + head_dim];
                for d in 0..head_dim {
                    out_head[d] = output_acc[d] / row_sum;
                }
            }
        }
    }
}
```

**src/kernels/activation.rs:**
```rust
pub fn rms_norm(
    input: &[f32],
    weight: &[f32],
    output: &mut [f32],
    eps: f32,
) {
    let hidden_size = weight.len();
    let seq_len = input.len() / hidden_size;
    
    for seq_i in 0..seq_len {
        let input_slice = &input[seq_i * hidden_size..(seq_i + 1) * hidden_size];
        let output_slice = &mut output[seq_i * hidden_size..(seq_i + 1) * hidden_size];
        
        // Compute RMS
        let mut sum_sq = 0.0f32;
        for &x in input_slice {
            sum_sq += x * x;
        }
        let rms = (sum_sq / hidden_size as f32 + eps).sqrt();
        
        // Apply normalization and scale
        for (i, (&input_val, &weight_val)) in input_slice.iter().zip(weight.iter()).enumerate() {
            output_slice[i] = (input_val / rms) * weight_val;
        }
    }
}

pub fn swiglu_inplace(gate: &mut [f32], up: &[f32]) {
    assert_eq!(gate.len(), up.len());
    
    for (g, &u) in gate.iter_mut().zip(up.iter()) {
        // SwiGLU: gate(x) * SiLU(up(x))
        // SiLU(x) = x * sigmoid(x) = x / (1 + exp(-x))
        let silu = u / (1.0 + (-u).exp());
        *g *= silu;
    }
}

pub fn gelu(x: f32) -> f32 {
    const SQRT_2_PI: f32 = 0.7978845608; // sqrt(2/π)
    0.5 * x * (1.0 + libm::tanhf(SQRT_2_PI * (x + 0.044715 * x * x * x)))
}

pub fn silu(x: f32) -> f32 {
    x / (1.0 + (-x).exp())
}
```

### 3.5 Memory Management
**src/memory/arena.rs:**
```rust
use std::ptr::NonNull;
use std::alloc::{self, Layout};

pub struct MemoryArena {
    memory: NonNull<u8>,
    size: usize,
    offset: usize,
    high_water_mark: usize,
}

impl MemoryArena {
    pub fn new(size: usize) -> Result<Self, &'static str> {
        let layout = Layout::from_size_align(size, 64)
            .map_err(|_| "Invalid memory layout")?;
        
        let memory = unsafe {
            let ptr = alloc::alloc_zeroed(layout);
            if ptr.is_null() {
                return Err("Failed to allocate memory");
            }
            NonNull::new_unchecked(ptr)
        };

        Ok(Self {
            memory,
            size,
            offset: 0,
            high_water_mark: size, // KV cache grows from high end
        })
    }

    pub fn allocate_tensor(&mut self, shape: &[usize]) -> Result<&mut [f32], &'static str> {
        let elements: usize = shape.iter().product();
        let bytes_needed = elements * std::mem::size_of::<f32>();
        let aligned_bytes = (bytes_needed + 63) & !63; // 64-byte alignment

        if self.offset + aligned_bytes > self.high_water_mark {
            return Err("Out of memory");
        }

        let ptr = unsafe {
            self.memory.as_ptr().add(self.offset) as *mut f32
        };
        
        self.offset += aligned_bytes;
        
        unsafe {
            Ok(std::slice::from_raw_parts_mut(ptr, elements))
        }
    }

    pub fn allocate_kv_cache(&mut self, size_bytes: usize) -> Result<&mut [u8], &'static str> {
        let aligned_size = (size_bytes + 63) & !63;
        
        if self.high_water_mark < aligned_size || self.high_water_mark - aligned_size < self.offset {
            return Err("Out of memory for KV cache");
        }

        self.high_water_mark -= aligned_size;
        
        let ptr = unsafe {
            self.memory.as_ptr().add(self.high_water_mark)
        };
        
        unsafe {
            Ok(std::slice::from_raw_parts_mut(ptr, size_bytes))
        }
    }

    pub fn reset_activations(&mut self) {
        // Reset activation memory (keep KV cache intact)
        self.offset = 0;
    }

    pub fn memory_usage(&self) -> (usize, usize, usize) {
        // Returns (activations_used, kv_cache_used, total_available)
        let activations_used = self.offset;
        let kv_cache_used = self.size - self.high_water_mark;
        (activations_used, kv_cache_used, self.size)
    }
}

impl Drop for MemoryArena {
    fn drop(&mut self) {
        let layout = Layout::from_size_align(self.size, 64).unwrap();
        unsafe {
            alloc::dealloc(self.memory.as_ptr(), layout);
        }
    }
}

unsafe impl Send for MemoryArena {}
unsafe impl Sync for MemoryArena {}
```

**src/memory/cache.rs:**
```rust
use crate::model::ModelConfig;
use crate::memory::MemoryArena;

pub struct KVCache {
    k_cache: Vec<&'static mut [f32]>, // Per-layer key cache
    v_cache: Vec<&'static mut [f32]>, // Per-layer value cache
    max_seq_len: usize,
    num_layers: usize,
    num_kv_heads: usize,
    head_dim: usize,
    current_len: usize,
}

impl KVCache {
    pub fn new(
        config: &ModelConfig,
        arena: &mut MemoryArena,
    ) -> Result<Self, &'static str> {
        let max_seq_len = config.max_seq_len;
        let num_layers = config.num_layers;
        let num_kv_heads = config.num_kv_heads;
        let head_dim = config.hidden_size / config.num_heads;
        
        let cache_size_per_layer = max_seq_len * num_kv_heads * head_dim;
        let total_cache_size = num_layers * 2 * cache_size_per_layer * std::mem::size_of::<f32>();
        
        let cache_memory = arena.allocate_kv_cache(total_cache_size)?;
        let cache_ptr = cache_memory.as_mut_ptr() as *mut f32;
        
        let mut k_cache = Vec::with_capacity(num_layers);
        let mut v_cache = Vec::with_capacity(num_layers);
        
        for layer in 0..num_layers {
            let k_offset = layer * 2 * cache_size_per_layer;
            let v_offset = layer * 2 * cache_size_per_layer + cache_size_per_layer;
            
            let k_slice = unsafe {
                std::slice::from_raw_parts_mut(
                    cache_ptr.add(k_offset),
                    cache_size_per_layer,
                )
            };
            let v_slice = unsafe {
                std::slice::from_raw_parts_mut(
                    cache_ptr.add(v_offset),
                    cache_size_per_layer,
                )
            };
            
            k_cache.push(unsafe { std::mem::transmute(k_slice) });
            v_cache.push(unsafe { std::mem::transmute(v_slice) });
        }
        
        Ok(Self {
            k_cache,
            v_cache,
            max_seq_len,
            num_layers,
            num_kv_heads,
            head_dim,
            current_len: 0,
        })
    }

    pub fn update(
        &mut self,
        layer: usize,
        new_k: &[f32],
        new_v: &[f32],
        start_pos: usize,
        seq_len: usize,
    ) -> Result<(), &'static str> {
        if layer >= self.num_layers {
            return Err("Layer index out of bounds");
        }
        
        if start_pos + seq_len > self.max_seq_len {
            return Err("Sequence too long for cache");
        }
        
        let kv_dim = self.num_kv_heads * self.head_dim;
        
        for i in 0..seq_len {
            let src_offset = i * kv_dim;
            let dst_offset = (start_pos + i) * kv_dim;
            
            let k_src = &new_k[src_offset..src_offset + kv_dim];
            let v_src = &new_v[src_offset..src_offset + kv_dim];
            
            let k_dst = &mut self.k_cache[layer][dst_offset..dst_offset + kv_dim];
            let v_dst = &mut self.v_cache[layer][dst_offset..dst_offset + kv_dim];
            
            k_dst.copy_from_slice(k_src);
            v_dst.copy_from_slice(v_src);
        }
        
        self.current_len = self.current_len.max(start_pos + seq_len);
        Ok(())
    }

    pub fn get(&self, layer: usize, seq_len: usize) -> Result<(&[f32], &[f32]), &'static str> {
        if layer >= self.num_layers {
            return Err("Layer index out of bounds");
        }
        
        let total_elements = seq_len * self.num_kv_heads * self.head_dim;
        
        let k_slice = &self.k_cache[layer][..total_elements];
        let v_slice = &self.v_cache[layer][..total_elements];
        
        Ok((k_slice, v_slice))
    }

    pub fn clear(&mut self) {
        self.current_len = 0;
        // Optionally zero out memory for security
        for layer in 0..self.num_layers {
            self.k_cache[layer].fill(0.0);
            self.v_cache[layer].fill(0.0);
        }
    }
}
```

### 3.6 Weight Fetching & Caching
**src/io/fetcher.rs:**
```rust
use wasm_bindgen::prelude::*;
use wasm_bindgen_futures::JsFuture;
use web_sys::{Request, RequestInit, Headers, Response};
use std::collections::HashMap;
use crate::tensor::Tensor;

pub struct WeightFetcher {
    base_url: String,
    auth_token: String,
    tensor_cache: HashMap<String, Tensor>,
    cache_size_limit: usize,
    cache_size_used: usize,
}

impl WeightFetcher {
    pub fn new(base_url: &str, auth_token: &str) -> Result<Self, JsValue> {
        Ok(Self {
            base_url: base_url.to_string(),
            auth_token: auth_token.to_string(),
            tensor_cache: HashMap::new(),
            cache_size_limit: 8 * 1024 * 1024, // 8MB cache
            cache_size_used: 0,
        })
    }

    pub async fn get_tensor(&mut self, tensor_name: &str) -> Result<&Tensor, Box<dyn std::error::Error>> {
        // Check cache first
        if let Some(tensor) = self.tensor_cache.get(tensor_name) {
            return Ok(tensor);
        }

        // Fetch from server
        let tensor = self.fetch_tensor(tensor_name).await?;
        
        // Cache small tensors (layernorms, embeddings, etc.)
        if tensor.size_bytes() < self.cache_size_limit {
            self.evict_if_needed(tensor.size_bytes());
            self.cache_size_used += tensor.size_bytes();
            self.tensor_cache.insert(tensor_name.to_string(), tensor);
            Ok(self.tensor_cache.get(tensor_name).unwrap())
        } else {
            // For large tensors, return without caching
            // Note: This is a simplified approach - in practice you'd want
            // to store the tensor somewhere and return a reference
            Err("Large tensor handling not implemented".into())
        }
    }

    pub async fn fetch_and_cache(&mut self, tensor_name: &str) -> Result<(), Box<dyn std::error::Error>> {
        if self.tensor_cache.contains_key(tensor_name) {
            return Ok(());
        }

        let tensor = self.fetch_tensor(tensor_name).await?;
        self.evict_if_needed(tensor.size_bytes());
        self.cache_size_used += tensor.size_bytes();
        self.tensor_cache.insert(tensor_name.to_string(), tensor);
        Ok(())
    }

    async fn fetch_tensor(&self, tensor_name: &str) -> Result<Tensor, Box<dyn std::error::Error>> {
        // Determine which shard contains this tensor
        let shard_name = self.get_shard_for_tensor(tensor_name);
        
        // Create request
        let mut request_init = RequestInit::new();
        request_init.method("POST");
        
        // Set headers
        let headers = Headers::new()?;
        headers.set("Content-Type", "application/x-protobuf")?;
        headers.set("Authorization", &format!("Bearer {}", self.auth_token))?;
        headers.set("X-WS-Timestamp", &self.get_timestamp())?;
        headers.set("X-WS-Signature", &self.sign_request(tensor_name)?)?;
        request_init.headers(&headers);

        // Create protobuf request
        let tensor_request = crate::proto::TensorRequest {
            model_id: "tinyllama-1.1b-v1".to_string(),
            shard_name: shard_name.to_string(),
            tensor_name: tensor_name.to_string(),
            compression: 0, // No compression
            quantization: 0, // No quantization
        };

        let request_bytes = self.encode_protobuf_request(&tensor_request)?;
        request_init.body(Some(&js_sys::Uint8Array::from(request_bytes.as_slice()).into()));

        // Make request
        let request = Request::new_with_str_and_init(&self.base_url, &request_init)?;
        let window = web_sys::window().unwrap();
        let resp_value = JsFuture::from(window.fetch_with_request(&request)).await?;
        let response: Response = resp_value.dyn_into().unwrap();

        if !response.ok() {
            return Err(format!("HTTP error: {}", response.status()).into());
        }

        // Stream response body
        let reader = response.body()
            .unwrap()
            .get_reader()
            .unchecked_into::<web_sys::ReadableStreamDefaultReader>();

        let mut tensor_data = Vec::new();
        let mut metadata = None;

        loop {
            let chunk_promise = reader.read();
            let chunk_result = JsFuture::from(chunk_promise).await?;
            let chunk = js_sys::Reflect::get(&chunk_result, &"value".into())?;
            let done = js_sys::Reflect::get(&chunk_result, &"done".into())?.as_bool().unwrap();

            if done {
                break;
            }

            let chunk_bytes = js_sys::Uint8Array::new(&chunk).to_vec();
            let tensor_chunk = self.decode_protobuf_chunk(&chunk_bytes)?;
            
            if metadata.is_none() {
                metadata = tensor_chunk.metadata.clone();
            }
            
            tensor_data.extend_from_slice(&tensor_chunk.data);
        }

        let metadata = metadata.ok_or("No metadata received")?;
        Ok(Tensor::new(tensor_data, metadata.shape, tensor_name))
    }

    fn get_shard_for_tensor(&self, tensor_name: &str) -> &str {
        // Simple shard mapping - in practice this would come from model info
        if tensor_name.contains("embed_tokens") || tensor_name.contains("lm_head") {
            if tensor_name.contains("embed_tokens") {
                "model-00001-of-00003.safetensors"
            } else {
                "model-00003-of-00003.safetensors"
            }
        } else {
            "model-00002-of-00003.safetensors" // Most layer weights
        }
    }

    fn get_timestamp(&self) -> String {
        let now = js_sys::Date::now() as u64 / 1000;
        now.to_string()
    }

    fn sign_request(&self, tensor_name: &str) -> Result<String, Box<dyn std::error::Error>> {
        // In a real implementation, you'd use a proper signing library
        // For now, return a mock signature
        Ok(base64::encode(&format!("{}:{}", self.auth_token, tensor_name)))
    }

    fn encode_protobuf_request(&self, request: &crate::proto::TensorRequest) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
        // Encode using prost
        let mut buf = Vec::new();
        prost::Message::encode(request, &mut buf)?;
        Ok(buf)
    }

    fn decode_protobuf_chunk(&self, data: &[u8]) -> Result<crate::proto::TensorChunk, Box<dyn std::error::Error>> {
        Ok(prost::Message::decode(data)?)
    }

    fn evict_if_needed(&mut self, new_size: usize) {
        while self.cache_size_used + new_size > self.cache_size_limit && !self.tensor_cache.is_empty() {
            // Simple FIFO eviction - remove first cached tensor
            if let Some((key, tensor)) = self.tensor_cache.iter().next() {
                let size = tensor.size_bytes();
                let key = key.clone();
                self.tensor_cache.remove(&key);
                self.cache_size_used -= size;
            }
        }
    }
}
```

### 3.7 Tensor Abstraction
**src/tensor/tensor.rs:**
```rust
#[derive(Clone, Debug)]
pub struct Tensor {
    data: Vec<f32>,
    shape: Vec<usize>,
    name: String,
    strides: Vec<usize>,
}

impl Tensor {
    pub fn new(raw_data: Vec<u8>, shape: Vec<i64>, name: &str) -> Self {
        // Convert raw bytes to f32 array
        let data = raw_data
            .chunks_exact(4)
            .map(|chunk| f32::from_le_bytes([chunk[0], chunk[1], chunk[2], chunk[3]]))
            .collect();
        
        let shape_usize: Vec<usize> = shape.iter().map(|&x| x as usize).collect();
        let strides = Self::compute_strides(&shape_usize);
        
        Self {
            data,
            shape: shape_usize,
            name: name.to_string(),
            strides,
        }
    }

    pub fn from_f32(data: Vec<f32>, shape: Vec<usize>, name: &str) -> Self {
        let strides = Self::compute_strides(&shape);
        Self {
            data,
            shape,
            name: name.to_string(),
            strides,
        }
    }

    fn compute_strides(shape: &[usize]) -> Vec<usize> {
        let mut strides = vec![1; shape.len()];
        for i in (0..shape.len().saturating_sub(1)).rev() {
            strides[i] = strides[i + 1] * shape[i + 1];
        }
        strides
    }

    pub fn data(&self) -> &[f32] {
        &self.data
    }

    pub fn shape(&self) -> &[usize] {
        &self.shape
    }

    pub fn ndim(&self) -> usize {
        self.shape.len()
    }

    pub fn size(&self) -> usize {
        self.data.len()
    }

    pub fn size_bytes(&self) -> usize {
        self.data.len() * std::mem::size_of::<f32>()
    }

    pub fn name(&self) -> &str {
        &self.name
    }

    // Index into tensor with multi-dimensional indices
    pub fn get(&self, indices: &[usize]) -> Option<f32> {
        if indices.len() != self.shape.len() {
            return None;
        }

        let mut flat_index = 0;
        for (i, (&idx, &dim_size)) in indices.iter().zip(&self.shape).enumerate() {
            if idx >= dim_size {
                return None;
            }
            flat_index += idx * self.strides[i];
        }

        self.data.get(flat_index).copied()
    }

    // Get a slice of the tensor data
    pub fn slice(&self, start: usize, end: usize) -> &[f32] {
        &self.data[start..end.min(self.data.len())]
    }

    // Reshape tensor (data must be compatible)
    pub fn reshape(&mut self, new_shape: Vec<usize>) -> Result<(), &'static str> {
        let new_size: usize = new_shape.iter().product();
        if new_size != self.data.len() {
            return Err("Incompatible reshape dimensions");
        }

        self.shape = new_shape;
        self.strides = Self::compute_strides(&self.shape);
        Ok(())
    }

    // Create a view into a specific row (for matrix operations)
    pub fn row(&self, row_idx: usize) -> Option<&[f32]> {
        if self.ndim() != 2 || row_idx >= self.shape[0] {
            return None;
        }

        let start = row_idx * self.shape[1];
        let end = start + self.shape[1];
        Some(&self.data[start..end])
    }

    // Matrix-vector multiplication optimized for transformer weights
    pub fn matmul_vec(&self, input: &[f32], output: &mut [f32]) -> Result<(), &'static str> {
        if self.ndim() != 2 {
            return Err("Matrix must be 2D");
        }
        
        let [rows, cols] = [self.shape[0], self.shape[1]];
        
        if input.len() != cols || output.len() != rows {
            return Err("Dimension mismatch");
        }

        crate::kernels::matmul::matmul(&self.data, input, output, rows, cols, 1);
        Ok(())
    }
}

// Quantized tensor support
#[derive(Clone, Debug)]
pub struct QuantizedTensor {
    data: Vec<u8>,
    scales: Vec<f32>,
    shape: Vec<usize>,
    quantization_type: QuantizationType,
}

#[derive(Clone, Debug)]
pub enum QuantizationType {
    Q4_0,
    Q8_0,
}

impl QuantizedTensor {
    pub fn dequantize(&self) -> Tensor {
        match self.quantization_type {
            QuantizationType::Q4_0 => self.dequantize_q4(),
            QuantizationType::Q8_0 => self.dequantize_q8(),
        }
    }

    fn dequantize_q4(&self) -> Tensor {
        let mut dequantized = Vec::with_capacity(self.data.len() * 2);
        
        for (chunk, &scale) in self.data.chunks(16).zip(&self.scales) {
            for &byte in chunk {
                // Each byte contains two 4-bit values
                let val1 = ((byte & 0xF) as i8 - 8) as f32 * scale;
                let val2 = (((byte >> 4) & 0xF) as i8 - 8) as f32 * scale;
                dequantized.push(val1);
                dequantized.push(val2);
            }
        }

        Tensor::from_f32(dequantized, self.shape.clone(), "dequantized")
    }

    fn dequantize_q8(&self) -> Tensor {
        let mut dequantized = Vec::with_capacity(self.data.len());
        
        for (&byte, &scale) in self.data.iter().zip(&self.scales) {
            let val = (byte as i8) as f32 * scale;
            dequantized.push(val);
        }

        Tensor::from_f32(dequantized, self.shape.clone(), "dequantized")
    }
}
```

### 3.8 RoPE Implementation
**src/utils/rope.rs:**
```rust
// Pre-computed RoPE tables embedded at compile time
static ROPE_FREQS: &[f32] = include!("../../build/rope_freqs.rs");

pub fn apply_rope(
    q: &mut [f32],
    k: &mut [f32],
    start_pos: usize,
    head_dim: usize,
    seq_len: usize,
    num_q_heads: usize,
    num_kv_heads: usize,
) {
    let half_head_dim = head_dim / 2;
    
    // Apply RoPE to queries
    for seq_i in 0..seq_len {
        let pos = start_pos + seq_i;
        
        for head in 0..num_q_heads {
            let q_offset = seq_i * num_q_heads * head_dim + head * head_dim;
            apply_rope_to_head(&mut q[q_offset..q_offset + head_dim], pos, half_head_dim);
        }
    }
    
    // Apply RoPE to keys
    for seq_i in 0..seq_len {
        let pos = start_pos + seq_i;
        
        for head in 0..num_kv_heads {
            let k_offset = seq_i * num_kv_heads * head_dim + head * head_dim;
            apply_rope_to_head(&mut k[k_offset..k_offset + head_dim], pos, half_head_dim);
        }
    }
}

fn apply_rope_to_head(head: &mut [f32], pos: usize, half_head_dim: usize) {
    for i in 0..half_head_dim {
        let freq = ROPE_FREQS[i];
        let angle = pos as f32 * freq;
        let cos_val = angle.cos();
        let sin_val = angle.sin();
        
        let x = head[i];
        let y = head[i + half_head_dim];
        
        head[i] = x * cos_val - y * sin_val;
        head[i + half_head_dim] = x * sin_val + y * cos_val;
    }
}

// Build-time RoPE frequency computation
pub fn compute_rope_freqs(head_dim: usize, max_seq_len: usize, theta: f32) -> Vec<f32> {
    let half_dim = head_dim / 2;
    let mut freqs = Vec::with_capacity(half_dim);
    
    for i in 0..half_dim {
        let freq = 1.0 / theta.powf((i * 2) as f32 / head_dim as f32);
        freqs.push(freq);
    }
    
    freqs
}
```

**build.rs:**
```rust
use std::env;
use std::fs;
use std::path::Path;

fn main() {
    // Generate protobuf bindings
    prost_build::Config::new()
        .out_dir("src/io")
        .compile_protos(&["proto/weight_service.proto"], &["proto/"])
        .unwrap();

    // Pre-compute RoPE frequencies
    let out_dir = env::var("OUT_DIR").unwrap();
    let dest_path = Path::new(&out_dir).join("rope_freqs.rs");
    
    // TinyLlama configuration
    let head_dim = 64; // 2048 / 32 heads
    let max_seq_len = 4096;
    let theta = 10000.0;
    
    let freqs = compute_rope_freqs(head_dim, max_seq_len, theta);
    let freqs_code = format!(
        "&[{}]",
        freqs.iter()
            .map(|f| f.to_string())
            .collect::<Vec<_>>()
            .join(", ")
    );
    
    fs::write(&dest_path, freqs_code).unwrap();
    
    // Copy to source for static inclusion
    let src_dest = Path::new("build").join("rope_freqs.rs");
    fs::create_dir_all("build").unwrap();
    fs::write(&src_dest, freqs_code).unwrap();

    println!("cargo:rerun-if-changed=proto/weight_service.proto");
    println!("cargo:rerun-if-changed=build.rs");
}

fn compute_rope_freqs(head_dim: usize, max_seq_len: usize, theta: f32) -> Vec<f32> {
    let half_dim = head_dim / 2;
    let mut freqs = Vec::with_capacity(half_dim);
    
    for i in 0..half_dim {
        let freq = 1.0 / theta.powf((i * 2) as f32 / head_dim as f32);
        freqs.push(freq);
    }
    
    freqs
}
```

---

## Track 4: Tokenizer Integration
**Duration:** 2 weeks | **Owner:** NLP Engineer

### 4.1 TikToken WASM Wrapper
```rust
// tokenizer/src/lib.rs
use wasm_bindgen::prelude::*;
use tiktoken_rs::{CoreBPE, get_bpe_from_model};

#[wasm_bindgen]
pub struct Tokenizer {
    bpe: CoreBPE,
}

#[wasm_bindgen]
impl Tokenizer {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Result<Tokenizer, JsValue> {
        let bpe = get_bpe_from_model("gpt-3.5-turbo")
            .map_err(|e| JsValue::from_str(&e.to_string()))?;
        
        Ok(Tokenizer { bpe })
    }

    #[wasm_bindgen]
    pub fn encode(&self, text: &str) -> Vec<u32> {
        self.bpe.encode_with_special_tokens(text)
            .into_iter()
            .map(|x| x as u32)
            .collect()
    }

    #[wasm_bindgen]
    pub fn decode(&self, tokens: &[u32]) -> String {
        let tokens: Vec<usize> = tokens.iter().map(|&x| x as usize).collect();
        self.bpe.decode(tokens).unwrap_or_default()
    }

    #[wasm_bindgen]
    pub fn get_vocab_size(&self) -> u32 {
        32000 // TinyLlama vocab size
    }
}
```

---

## Track 5: Sampling & Generation
**Duration:** 1 week | **Owner:** ML Engineer

### 5.1 JavaScript Sampling Interface
```javascript
// sampler.js
export class Sampler {
    constructor(config = {}) {
        this.temperature = config.temperature || 0.7;
        this.topP = config.topP || 0.9;
        this.topK = config.topK || 50;
        this.repetitionPenalty = config.repetitionPenalty || 1.1;
        this.eosTokenId = config.eosTokenId || 2;
    }

    sample(logitsPtr, vocabSize, generatedTokens = []) {
        // Copy logits from WASM memory
        const logits = new Float32Array(
            wasmModule.memory.buffer,
            logitsPtr,
            vocabSize
        );

        // Apply repetition penalty
        if (this.repetitionPenalty !== 1.0 && generatedTokens.length > 0) {
            this.applyRepetitionPenalty(logits, generatedTokens);
        }

        // Apply temperature scaling
        if (this.temperature !== 1.0) {
            for (let i = 0; i < logits.length; i++) {
                logits[i] /= this.temperature;
            }
        }

        // Convert to probabilities
        const probs = this.softmax(logits);

        // Apply top-k filtering
        if (this.topK > 0) {
            this.applyTopK(probs, this.topK);
        }

        // Apply nucleus (top-p) sampling
        if (this.topP < 1.0) {
            return this.nucleusSample(probs);
        }

        // Multinomial sampling
        return this.multinomialSample(probs);
    }

    softmax(logits) {
        const maxLogit = Math.max(...logits);
        const expLogits = logits.map(x => Math.exp(x - maxLogit));
        const sumExp = expLogits.reduce((a, b) => a + b, 0);
        return expLogits.map(x => x / sumExp);
    }

    applyRepetitionPenalty(logits, tokens) {
        const seenTokens = new Set(tokens.slice(-64)); // Last 64 tokens
        for (const token of seenTokens) {
            if (logits[token] > 0) {
                logits[token] /= this.repetitionPenalty;
            } else {
                logits[token] *= this.repetitionPenalty;
            }
        }
    }

    applyTopK(probs, k) {
        // Get top-k indices
        const indices = Array.from({length: probs.length}, (_, i) => i);
        indices.sort((a, b) => probs[b] - probs[a]);
        
        // Zero out all but top-k
        for (let i = k; i < indices.length; i++) {
            probs[indices[i]] = 0;
        }

        // Renormalize
        const sum = probs.reduce((a, b) => a + b, 0);
        if (sum > 0) {
            for (let i = 0; i < probs.length; i++) {
                probs[i] /= sum;
            }
        }
    }

    nucleusSample(probs) {
        // Sort probabilities in descending order
        const sorted = probs.map((p, i) => [p, i]).sort((a, b) => b[0] - a[0]);
        
        // Find nucleus cutoff
        let cumulativeP = 0;
        let cutoff = 0;
        for (let i = 0; i < sorted.length; i++) {
            cumulativeP += sorted[i][0];
            if (cumulativeP >= this.topP) {
                cutoff = i + 1;
                break;
            }
        }

        // Sample from nucleus
        const nucleus = sorted.slice(0, cutoff);
        const nucleusSum = nucleus.reduce((sum, [p]) => sum + p, 0);
        
        const r = Math.random() * nucleusSum;
        let cumulative = 0;
        for (const [prob, index] of nucleus) {
            cumulative += prob;
            if (r <= cumulative) {
                return index;
            }
        }
        
        return nucleus[0][1]; // Fallback
    }

    multinomialSample(probs) {
        const r = Math.random();
        let cumulative = 0;
        for (let i = 0; i < probs.length; i++) {
            cumulative += probs[i];
            if (r <= cumulative) {
                return i;
            }
        }
        return probs.length - 1; // Fallback
    }
}

// Constraint-based sampling for structured output
export class ConstrainedSampler extends Sampler {
    constructor(config = {}) {
        super(config);
        this.grammar = config.grammar;
        this.state = 'start';
    }

    sample(logitsPtr, vocabSize, generatedTokens = []) {
        const logits = new Float32Array(
            wasmModule.memory.buffer,
            logitsPtr,
            vocabSize
        );

        // Apply grammar constraints
        if (this.grammar) {
            this.applyGrammarMask(logits, this.state);
        }

        const token = super.sample(logitsPtr, vocabSize, generatedTokens);
        
        // Update grammar state
        if (this.grammar) {
            this.state = this.updateGrammarState(this.state, token);
        }
        
        return token;
    }

    applyGrammarMask(logits, state) {
        const allowedTokens = this.grammar.getAllowedTokens(state);
        for (let i = 0; i < logits.length; i++) {
            if (!allowedTokens.has(i)) {
                logits[i] = -Infinity;
            }
        }
    }
}
```

---

## Track 6: WebWorker Integration
**Duration:** 1 week | **Owner:** Frontend Engineer

### 6.1 Worker Implementation
```javascript
// cortex-worker.js
import init, { Cortex } from './pkg/cortex.js';
import { Tokenizer } from './tokenizer.js';
import { Sampler } from './sampler.js';

class CortexWorker {
    constructor() {
        this.cortex = null;
        this.tokenizer = null;
        this.sampler = null;
        this.isInitialized = false;
    }

    async initialize(config) {
        try {
            // Initialize WASM
            await init();
            
            // Initialize components
            this.cortex = new Cortex(
                config.weightServerUrl,
                config.authToken,
                config.cacheSizeMB || 256
            );
            
            this.tokenizer = new Tokenizer();
            this.sampler = new Sampler(config.sampling || {});
            
            // Warmup
            await this.cortex.warmup();
            
            this.isInitialized = true;
            return { success: true };
        } catch (error) {
            return { success: false, error: error.message };
        }
    }

    async generateText(prompt, config = {}) {
        if (!this.isInitialized) {
            throw new Error('Worker not initialized');
        }

        const maxTokens = config.maxTokens || 100;
        const stopTokens = config.stopTokens || [];
        
        // Encode prompt
        const promptTokens = this.tokenizer.encode(prompt);
        
        // Prefill
        const logitsPtr = await this.cortex.prefill(promptTokens);
        
        // Generate
        const generated = [];
        let currentText = prompt;
        
        for (let i = 0; i < maxTokens; i++) {
            // Sample next token
            const token = this.sampler.sample(
                logitsPtr,
                this.cortex.get_vocab_size(),
                generated
            );
            
            // Check for stop conditions
            if (stopTokens.includes(token) || token === this.sampler.eosTokenId) {
                break;
            }
            
            generated.push(token);
            
            // Decode and send progress
            const newText = this.tokenizer.decode([token]);
            currentText += newText;
            
            self.postMessage({
                type: 'progress',
                data: {
                    token,
                    text: newText,
                    fullText: currentText,
                    tokenCount: generated.length
                }
            });
            
            // Continue generation
            if (i < maxTokens - 1) {
                const nextLogitsPtr = await this.cortex.decode(token);
                // logitsPtr remains valid for next iteration
            }
        }
        
        return {
            text: currentText,
            tokens: [...promptTokens, ...generated],
            tokenCount: generated.length
        };
    }

    async handleMessage(event) {
        const { id, type, data } = event.data;
        
        try {
            let result;
            
            switch (type) {
                case 'initialize':
                    result = await this.initialize(data);
                    break;
                    
                case 'generate':
                    result = await this.generateText(data.prompt, data.config);
                    break;
                    
                case 'reset':
                    if (this.cortex) {
                        this.cortex.reset();
                    }
                    result = { success: true };
                    break;
                    
                default:
                    throw new Error(`Unknown message type: ${type}`);
            }
            
            self.postMessage({
                id,
                type: 'response',
                data: result
            });
        } catch (error) {
            self.postMessage({
                id,
                type: 'error',
                data: { error: error.message, stack: error.stack }
            });
        }
    }
}

//
        