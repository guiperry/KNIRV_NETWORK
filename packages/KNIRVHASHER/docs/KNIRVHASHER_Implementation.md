# KNIRVHASHER × KNIRV Network Implementation Plan

## Goal
Ingest user ontology data and server activity from KNIRVSERVER via the DVE, train user-centric logic gate hash networks, and create custom security rules for KNIRV agents.

---

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Data Transfer | **gRPC** (encrypted) | Direct to local hasher instance; PQC-encrypted payloads |
| gRPC Transport | **Unix socket** (`/var/run/hasher.sock`) | Better for containerized environments |
| Training Triggers | **On-demand + Scheduled + Event-driven** | Demos + production + guardrail violations |
| Seed Storage | **KNIRVBASE** | PQC-encrypted, distributed, existing infrastructure |
| Data Format | **.arrow + .json** | Both formats for compatibility |
| Connector Language | **Go** | Native to hasher, no TS overhead |
| Connector Location | **0_DATA_CONNECTOR** | Pre-pipeline data preparation |
| Integration Point | **DVE** | Orchestrates validation, guardrails, ontology |
| Fallback | **Existing guardrails only** | Hasher is advisory; fail gracefully |

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              KNIRVSERVER                                   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        DVE (Integration Point)                        │   │
│  │                                                                      │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │   │
│  │  │ DVEManager │  │Validation  │  │ Guardrail   │  │ DVEOntology │  │   │
│  │  │            │  │ Core       │  │ Engine      │  │ Manager     │  │   │
│  │  └──────┬──────┘  └──────┬─────┘  └──────┬──────┘  └──────┬──────┘  │   │
│  │         │                │                │                │         │   │
│  │         └────────────────┴────────────────┴────────────────┘         │   │
│  │                                │                                      │   │
│  │                    ┌───────────┴───────────┐                          │   │
│  │                    │  HasherIntegration   │                          │   │
│  │                    │  Hook                │                          │   │
│  │                    └───────────┬───────────┘                          │   │
│  └────────────────────────────────┼────────────────────────────────────┘   │
│                                   │                                          │
│  ┌────────────────────────────────┼────────────────────────────────────┐   │
│  │           DVE Data Sources      │                                    │   │
│  │  ┌──────────┐  ┌────────────┐  │  ┌────────────┐  ┌─────────────┐  │   │
│  │  │ BuntDB   │  │ Markdown   │  │  │ SSE Events │  │ Guardrail   │  │   │
│  │  │ (Users)  │  │ Storage    │  │  │            │  │ Manager     │  │   │
│  │  └──────────┘  └────────────┘  │  └────────────┘  └─────────────┘  │   │
│  └────────────────────────────────┼────────────────────────────────────┘   │
└──────────────────────────────────┼────────────────────────────────────────┘
                                   │
                    ┌──────────────┴──────────────┐
                    │      gRPC (PQC Encrypted)   │
                    │   Encrypted Payloads Only    │
                    └──────────────┬──────────────┘
                                   │
┌──────────────────────────────────┼────────────────────────────────────────┐
│                           KNIRVHASHER                                    │
│                                                                      │
│  ┌─────────────────┐     ┌─────────────────┐                         │
│  │   0_DATA_       │     │   Hasher gRPC   │                         │
│  │   CONNECTOR     │────▶│   Service       │                         │
│  │   (Go)         │     │   (Server)      │                         │
│  └────────┬────────┘     └────────┬────────┘                         │
│           │                       │                                   │
│           ▼                       ▼                                   │
│  ┌─────────────────────────────────────────────────────────────┐     │
│  │                    HASHER PIPELINE                           │     │
│  │                                                               │     │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │     │
│  │  │ 1_DATA_MINER │  │ 2_DATA_     │  │ 3_DATA_     │      │     │
│  │  │              │─▶│ ENCODER     │─▶│ TRAINER     │      │     │
│  │  │ SpaCy NLP    │  │ BGE Embed   │  │ Evo-GRPO    │      │     │
│  │  │ Output: .arrow│  │ Output: .arrow│  │ Seeds       │      │     │
│  │  └──────────────┘  └──────────────┘  └──────┬───────┘      │     │
│  │                                              │              │     │
│  │                                              ▼              │     │
│  │  ┌─────────────────────────────────────────────────────┐   │     │
│  │  │           USER LOGIC GATE TRAINING                   │   │     │
│  │  │  • Security constraints → Slot 4/10 tokens          │   │     │
│  │  │  • Violations → Negative reinforcement            │   │     │
│  │  │  • Trained seeds → UserSecurityGates             │   │     │
│  │  └─────────────────────────────────────────────────────┘   │     │
│  └──────────────────────────────────────────────────────────────┘     │
│                                   │                                      │
│                                   ▼                                      │
│  ┌───────────────────────────────────────────────────────────────┐     │
│  │                      KNIRVBASE                                 │     │
│  │  • PQC-encrypted seed storage                                 │     │
│  │  • Collection: hasher_seeds                                    │     │
│  │  • Collection: hasher_rules                                    │     │
│  └───────────────────────────────────────────────────────────────┘     │
│                                   │                                      │
│                                   ▼                                      │
│  ┌───────────────────────────────────────────────────────────────┐     │
│  │                 SECURITY ENFORCER                               │     │
│  │  • ValidateAction(userID, action) → SecurityDecision            │     │
│  │  • Integration with DVE GuardrailEngine                         │     │
│  └───────────────────────────────────────────────────────────────┘     │
└───────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: KNIRVSERVER Hasher gRPC Service

### 1.1 Proto Definition

**File:** `backend/internal/proto/hasher.proto`

```protobuf
syntax = "proto3";

package hasher;

service HasherService {
    rpc ExportSecurityData(ExportRequest) returns (stream EncryptedChunk);
    rpc TriggerTraining(TrainingRequest) returns (TrainingResponse);
    rpc GetTrainingStatus(TrainingStatusRequest) returns (TrainingStatusResponse);
    rpc GetUserRules(RulesRequest) returns (RulesResponse);
    rpc ValidateAction(ActionRequest) returns (ActionResponse);
}

message ExportRequest {
    string org_id = 1;
    string user_id = 2;
    DataType data_type = 3;
    bool encrypted = 4;
}

enum DataType {
    ALL = 0;
    ONTOLOGY = 1;
    GUARDRAILS = 2;
    ACTIVITY = 3;
    MARKDOWN = 4;
}

message EncryptedChunk {
    bytes data = 1;
    string chunk_id = 2;
    bool is_last = 3;
}

message TrainingRequest {
    string org_id = 1;
    string user_id = 2;
    TrainingTrigger trigger = 3;
    map<string, string> options = 4;
}

enum TrainingTrigger {
    ON_DEMAND = 0;
    SCHEDULED = 1;
    GUARDRAIL_VIOLATION = 2;
}

message TrainingResponse {
    string training_id = 1;
    string status = 2;
}

message ActionRequest {
    string user_id = 1;
    string action = 2;
    map<string, string> context = 3;
}

message ActionResponse {
    bool allowed = 1;
    float confidence = 2;
    repeated string violations = 3;
    repeated string applied_rules = 4;
}
```

### 1.2 Hasher Integration Hook

**File:** `backend/internal/services/dve/hasher_integration.go`

```go
type HasherIntegration struct {
    grpcClient  hasherpb.HasherServiceClient
    dveManager  *DVEManager
    guardrailMgr *guardrails.DynamicGuardrailManager
    ontologyMgr  *DVEOntologyManager
    kvbase      knirvbase.Collection
}

func (hi *HasherIntegration) OnGuardrailViolation(violation *GuardrailViolation) error {
    return hi.TriggerTraining(violation.NodeID, GUARDRAIL_VIOLATION)
}

func (hi *HasherIntegration) OnValidationComplete(result *TaskResult) error {
    // Analyze patterns, trigger training if needed
}

func (hi *HasherIntegration) ExportUserData(orgID, userID string) (<-chan *EncryptedChunk, error)
func (hi *HasherIntegration) TriggerTraining(orgID, userID string, trigger TrainingTrigger) error
func (hi *HasherIntegration) ValidateAction(userID, action string, ctx map[string]string) (*ActionResponse, error)
```

### 1.3 Guardrail Extension

**File:** `backend/internal/services/guardrails/guardrail_manager.go` (extend)

```go
const (
    // ... existing types ...
    GuardrailTypeHasher GuardrailType = "hasher"
)

func (gm *DynamicGuardrailManager) ValidateWithHasher(nodeID string, action string) (*SecurityDecision, error) {
    if gm.hasherIntegration == nil || !gm.hasherIntegration.IsAvailable() {
        // Fallback: Use existing guardrails only (hasher is advisory)
        return &SecurityDecision{Allowed: true, Confidence: 0, Note: "hasher_unavailable"}, nil
    }
    return gm.hasherIntegration.ValidateAction(nodeID, action, nil)
}
```

---

## Phase 2: 0_DATA_CONNECTOR (Go)

### 2.1 Directory Structure

```
pipeline/0_DATA_CONNECTOR/
├── cmd/
│   └── connector/
│       └── main.go
├── internal/
│   ├── grpc/
│   │   └── client.go
│   ├── normalizer/
│   │   ├── normalizer.go
│   │   └── security_mapper.go
│   ├── cleaner/
│   │   └── cleaner.go
│   ├── encoder/
│   │   └── arrow_encoder.go
│   └── writer/
│       └── writer.go
├── config/
│   └── connector.yaml
├── go.mod
└── Makefile
```

### 2.2 Main Entry Point

**File:** `cmd/connector/main.go`

```go
func main() {
    config := LoadConfig()
    
    client := grpc.NewClient(config.HasherAddr)
    defer client.Close()
    
    stream, err := client.ExportSecurityData(&ExportRequest{
        OrgId: config.OrgID,
        UserId: config.UserID,
        DataType: DataType_ALL,
        Encrypted: true,
    })
    
    normalizer := normalizer.NewSecurityNormalizer()
    encoder := encoder.NewArrowEncoder()
    writer := writer.NewFileWriter(config.OutputDir)
    
    for chunk := range stream {
        records := normalizer.Process(chunk.Data)
        frames := encoder.Encode(records)
        writer.Write(frames)
    }
}
```

### 2.3 Security Normalizer

**File:** `internal/normalizer/security_mapper.go`

```go
type SecurityNormalizer struct {
    schema *SecuritySchema
}

type SecurityRecord struct {
    FileName   string    `arrow:"file_name"`
    ChunkID    int32     `arrow:"chunk_id"`
    Content    string    `arrow:"content"`
    Embedding  []float32 `arrow:"embedding"`
    Tokens     []string  `arrow:"tokens"`
    POSTags    []int     `arrow:"pos_tags"`
    DepHashes  []uint32  `arrow:"dep_hashes"`
    SecurityTags []string `arrow:"security_tags"`
}

var SECURITY_TAG_MAPPINGS = map[string]SecurityMapping{
    "guardrail_block": {
        Slot10: 0x2400, // Logic/Set domain
        Slot4:  0x07,  // PREP (constraint marker)
        Weight: -1.0,   // Negative reinforcement
    },
    "guardrail_warn": {
        Slot10: 0x2401, // Guardrail subdomain
        Slot4:  0x04,  // ADV
        Weight: -0.5,
    },
    "security_constraint": {
        Slot10: 0x2400,
        Slot4:  0x02,  // VERB (action)
        Weight: 1.0,
    },
    "violation": {
        Slot10: 0x2402, // Violation subdomain
        Slot4:  0x01,  // NOUN (subject)
        Weight: -2.0,   // Strong negative
    },
}
```

### 2.4 Arrow Encoder

**File:** `internal/encoder/arrow_encoder.go`

```go
type ArrowEncoder struct {
    schema *arrow.Schema
    pool   memory.GoAllocator
}

func NewArrowEncoder() *ArrowEncoder {
    schema := arrow.NewSchema([]arrow.Field{
        {Name: "file_name", Type: arrow.PrimitiveTypes.UTF8},
        {Name: "chunk_id", Type: arrow.PrimitiveTypes.Int32},
        {Name: "content", Type: arrow.PrimitiveTypes.UTF8},
        {Name: "embedding", Type: arrow.ListOf(arrow.PrimitiveTypes.Float32)},
        {Name: "tokens", Type: arrow.ListOf(arrow.PrimitiveTypes.UTF8)},
        {Name: "pos_tags", Type: arrow.ListOf(arrow.PrimitiveTypes.Int32)},
        {Name: "dep_hashes", Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32)},
        {Name: "security_tags", Type: arrow.ListOf(arrow.PrimitiveTypes.UTF8)},
    })
    return &ArrowEncoder{schema: schema, pool: memory.NewGoAllocator()}
}

func (e *ArrowEncoder) Encode(records []*SecurityRecord) (*arrow.Buffer, error) {
    // Build arrow record batch from records
    // Return IPC-encoded buffer
}
```

### 2.5 Config

**File:** `config/connector.yaml`

```yaml
hasher:
  socket: "/var/run/hasher.sock"  # Unix socket for container compatibility
  timeout: 300

source:
  org_id: "org_default"
  user_id: "all"

output:
  arrow_dir: "/tmp/hasher/frames/arrow"
  json_dir: "/tmp/hasher/frames/json"
  batch_size: 100

processing:
  max_concurrent: 4
  pii_scrub: true
  deduplicate: true
```

---

## Phase 3: Pipeline Updates

### 3.1 Data Miner Updates

**File:** `pipeline/1_DATA_MINER/internal/app/knirv.go` (new)

```go
func (c *Config) LoadKnirvInput() ([]*app.DocumentRecord, error) {
    arrowFiles, err := filepath.Glob(c.InputDir + "/*.arrow")
    jsonFiles, err := filepath.Glob(c.InputDir + "/*.json")
    
    var records []*app.DocumentRecord
    for _, f := range append(arrowFiles, jsonFiles...) {
        recs, err := loadRecords(f)
        records = append(records, recs...)
    }
    return records, nil
}

func loadRecords(path string) ([]*app.DocumentRecord, error) {
    if strings.HasSuffix(path, ".arrow") {
        return loadArrowRecords(path)
    }
    return loadJSONRecords(path)
}
```

### 3.2 Data Encoder: Arrow Output

**File:** `pipeline/2_DATA_ENCODER/internal/tensor_packer.go` (modify)

```go
type TensorPacker struct {
    outputFormat string // "arrow", "json", or "both"
}

func (tp *TensorPacker) SaveTrainingFrames(frames []*NeuralFrame, outputPath string) error {
    switch tp.outputFormat {
    case "arrow":
        return tp.saveArrow(frames, outputPath)
    case "json":
        return tp.saveJSON(frames, outputPath)
    case "both":
        if err := tp.saveArrow(frames, outputPath+".arrow"); err != nil {
            return err
        }
        return tp.saveJSON(frames, outputPath+".json")
    }
}

func (tp *TensorPacker) saveArrow(frames []*NeuralFrame, path string) error {
    schema := arrow.NewSchema([]arrow.Field{
        {Name: "frame_id", Type: arrow.PrimitiveTypes.Int32},
        {Name: "slots", Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32)},
        {Name: "embedding", Type: arrow.ListOf(arrow.PrimitiveTypes.Float32)},
        {Name: "target_token", Type: arrow.PrimitiveTypes.Int32},
    })
    
    // Build record batch and write IPC format
    writer, err := ipc.NewFileWriter(os.Create(path), ipc.WithSchema(schema))
    // ...
}
```

### 3.3 Security Schema

**File:** `pipeline/2_DATA_ENCODER/config/security_schema.yaml`

```yaml
domain:
  name: "SECURITY"
  slot10_base: 0x2400

subdomains:
  - id: 0x00
    name: "Constraint"
    slot10: 0x2400
  - id: 0x01
    name: "Guardrail"
    slot10: 0x2401
  - id: 0x02
    name: "Violation"
    slot10: 0x2402

pos_mappings:
  - id: 0x01
    name: "SUBJECT"
    spaCy_tags: ["NN", "NNS", "NNP", "NNPS"]
  - id: 0x02
    name: "ACTION"
    spaCy_tags: ["VB", "VBD", "VBG", "VBN", "VBP", "VBZ"]
  - id: 0x07
    name: "PREP"
    spaCy_tags: ["IN"]
```

---

## Phase 4: User Logic Gate Training

### 4.1 Training Module

**File:** `pipeline/3_DATA_TRAINER/pkg/training/user_security_gates.go`

```go
type UserSecurityGates struct {
    OrgID       string
    UserID      string
    Constraints []SecurityConstraint
    Patterns    []BehaviorPattern
    Seeds       []Seed
    Rules       []LogicalRule
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type SecurityConstraint struct {
    RuleID   string
    Text     string
    Type     string  // "allow", "deny", "flag"
    Severity string  // "low", "medium", "high", "critical"
    Tags     []string
}

type BehaviorPattern struct {
    TaskType string
    Success  bool
    Metrics  map[string]float64
    ActionHash uint64
}

type LogicalRule struct {
    RuleType   string   // "subsumption", "disjoint", "constraint"
    Premises   []string
    Conclusion string
    Source     string   // "guardrail", "ontology", "user"
}

type SecurityFitness struct {
    Alignment        float64
    Stability        float64
    Format           float64
    ConstraintScore  float64  // Does seed match constraints?
    ViolationPenalty float64  // Does seed avoid violations?
}

func (f *SecurityFitness) Total() float64 {
    return f.Alignment * 0.25 +
           f.Stability * 0.20 +
           f.Format * 0.15 +
           f.ConstraintScore * 0.25 +
           f.ViolationPenalty * 0.15
}

func (tg *UserTrainer) TrainUserGates(ctx context.Context, gates *UserSecurityGates) (*TrainedGates, error) {
    // 1. Extract constraint tokens from SecurityConstraints
    constraintTokens := tg.extractConstraintTokens(gates.Constraints)
    
    // 2. Extract violation patterns from BehaviorPatterns
    violationPatterns := tg.extractViolationPatterns(gates.Patterns)
    
    // 3. Run GRPO with security-aware fitness
    population := tg.initPopulation(len(constraintTokens))
    
    for gen := 0; gen < gates.MaxGenerations; gen++ {
        results := tg.evaluateWithSecurity(population, constraintTokens, violationPatterns)
        population = tg.selectAndMutate(results)
        
        if gates.CheckConvergence(results) {
            break
        }
    }
    
    // 4. Build logical rules from constraints
    rules := tg.buildLogicalRules(gates.Constraints, gates.Patterns)
    
    return &TrainedGates{
        Seeds:      population.BestSeeds(),
        Rules:      rules,
        Fitness:    population.BestFitness(),
        UserID:     gates.UserID,
        TrainedAt:  time.Now(),
    }, nil
}
```

### 4.2 Training Triggers

```go
type TrainingScheduler struct {
    onDemand   chan *TrainingRequest
    scheduled  *cron.Cron
    violation  chan *GuardrailViolation
    client     *grpc.Client
}

func (ts *TrainingScheduler) Start() {
    // On-demand handler
    go ts.handleOnDemand()
    
    // Scheduled (e.g., every 24 hours)
    ts.scheduled.AddFunc("0 0 * * *", ts.runScheduledTraining)
    
    // Event-driven from guardrails
    go ts.handleViolations()
}
```

---

## Phase 5: KNIRVBASE Storage

### 5.1 Collection Schema

**Collection:** `hasher_seeds`

```json
{
  "id": "user_{userID}_seed_{index}",
  "entryType": "MEMORY",
  "payload": {
    "org_id": "org_xxx",
    "user_id": "user_xxx",
    "seed_data": [0.123, 0.456, ...],
    "seed_hash": "sha256:abc123...",
    "fitness": 0.95,
    "training_id": "train_xxx",
    "trigger": "guardrail_violation",
    "created_at": "2026-04-01T00:00:00Z"
  }
}
```

**Collection:** `hasher_rules`

```json
{
  "id": "user_{userID}_rule_{index}",
  "entryType": "MEMORY",
  "payload": {
    "org_id": "org_xxx",
    "user_id": "user_xxx",
    "rule_type": "constraint",
    "premises": ["subject=exec", "domain=finance"],
    "conclusion": "deny",
    "confidence": 0.92,
    "source": "guardrail",
    "created_at": "2026-04-01T00:00:00Z"
  }
}
```

### 5.2 Storage Client

**File:** `pkg/storage/knirvbase_client.go`

```go
type KNIRVBASEClient struct {
    db knirvbase.DB
}

func (c *KNIRVBASEClient) SaveSeeds(seeds *TrainedGates) error {
    collection := c.db.Collection("hasher_seeds")
    
    for i, seed := range seeds.Seeds {
        _, err := collection.Insert(map[string]interface{}{
            "id": fmt.Sprintf("user_%s_seed_%d", seeds.UserID, i),
            "entryType": "MEMORY",
            "payload": map[string]interface{}{
                "org_id": seeds.OrgID,
                "user_id": seeds.UserID,
                "seed_data": seed.Data,
                "fitness": seed.Fitness,
                "created_at": time.Now(),
            },
        })
        if err != nil {
            return err
        }
    }
    return nil
}

func (c *KNIRVBASEClient) GetUserSeeds(userID string) ([]Seed, error) {
    collection := c.db.Collection("hasher_seeds")
    docs, err := collection.FindAll()
    // Filter by user_id and return seeds
}
```

---

## Phase 6: Security Enforcer

### 6.1 Enforcer Interface

**File:** `pkg/hashing/agent/security_enforcer.go`

```go
type SecurityEnforcer struct {
    network       *neural.HashNetwork
    userGates     map[string]*UserSecurityGates
    logicEngine   *validation.LogicalValidator
    kvbaseClient  *KNIRVBASEClient
}

type SecurityDecision struct {
    Allowed      bool
    Confidence   float64
    Violations   []string
    AppliedRules []string
    SeedID       string
}

func (se *SecurityEnforcer) ValidateAction(userID, action string, ctx map[string]string) (*SecurityDecision, error) {
    gates, err := se.loadUserGates(userID)
    if err != nil {
        return &SecurityDecision{Allowed: false, Confidence: 0, Violations: []string{"Failed to load user gates"}}, err
    }
    
    // 1. Encode action as hash network input
    input := se.encodeAction(action, ctx)
    
    // 2. Run inference with user-specific seeds
    prediction, confidence := se.network.ForwardWithSeeds(input, gates.Seeds)
    
    // 3. Apply logical rule validation
    ruleViolations := se.logicEngine.Validate(prediction, gates.Rules)
    
    // 4. Build decision
    decision := &SecurityDecision{
        Allowed:      len(ruleViolations) == 0,
        Confidence:   confidence,
        Violations:   ruleViolations,
        AppliedRules: se.getAppliedRules(gates.Rules),
        SeedID:       gates.BestSeedID(),
    }
    
    return decision, nil
}
```

### 6.2 DVE Integration

**File:** `backend/internal/services/dve/hasher_integration.go`

```go
func (hi *HasherIntegration) RegisterWithDVEManager(dve *DVEManager) {
    // Register hasher as a guardrail type
    dve.GuardrailEngine.RegisterType("hasher", hi.validateWithHasher)
    
    // Register event handlers
    dve.EventBus.Subscribe("guardrail.violation", hi.OnGuardrailViolation)
    dve.EventBus.Subscribe("validation.complete", hi.OnValidationComplete)
    
    // Register pre-action hook
    dve.PolicyEngine.RegisterPreActionHook(hi.ValidateAction)
}
```

---

## File Structure Summary

```
KNIRVHASHER/
├── pipeline/
│   ├── 0_DATA_CONNECTOR/          # NEW: Go connector
│   │   ├── cmd/connector/main.go
│   │   ├── internal/
│   │   │   ├── grpc/client.go
│   │   │   ├── normalizer/
│   │   │   ├── encoder/arrow_encoder.go
│   │   │   └── writer/
│   │   ├── config/connector.yaml
│   │   └── go.mod
│   ├── 1_DATA_MINER/
│   │   └── internal/app/knirv.go  # NEW: Load .arrow/.json
│   ├── 2_DATA_ENCODER/
│   │   └── internal/tensor_packer.go  # MODIFY: Arrow output
│   └── 3_DATA_TRAINER/
│       └── pkg/training/user_security_gates.go  # NEW
├── pkg/
│   ├── hashing/agent/security_enforcer.go  # NEW
│   └── storage/knirvbase_client.go         # NEW
└── internal/proto/hasher.proto             # NEW

KNIRVSERVER/
├── backend/
│   └── internal/
│       ├── proto/hasher.proto              # NEW
│       ├── services/
│       │   ├── dve/
│       │   │   └── hasher_integration.go   # NEW
│       │   └── guardrails/
│       │       └── guardrail_manager.go     # MODIFY
│       └── web/
│           └── routes.go                    # MODIFY
└── Makefile                                # MODIFY
```

---

## Implementation Order

| Phase | Task | Effort | Priority |
|-------|------|--------|----------|
| 1 | Define `proto/hasher.proto` | Low | P0 |
| 1 | Implement KNIRVSERVER gRPC server | Medium | P0 |
| 1 | Create `hasher_integration.go` in DVE | Medium | P0 |
| 2 | Build `0_DATA_CONNECTOR` in Go | High | P0 |
| 3 | Update pipeline for .arrow/.json | Medium | P1 |
| 4 | Implement `user_security_gates.go` | High | P1 |
| 5 | Create `knirvbase_client.go` | Medium | P1 |
| 6 | Implement `security_enforcer.go` | High | P1 |
| 6 | Wire enforcer to DVE guardrails | Medium | P1 |
| 7 | Add training scheduler (cron) | Low | P2 |
| 8 | End-to-end testing | High | P2 |

---

## Testing Strategy

### Unit Tests
- `0_DATA_CONNECTOR`: Normalizer, Arrow encoder
- `user_security_gates.go`: Fitness calculation, rule extraction
- `security_enforcer.go`: Action validation

### Integration Tests
```
KNIRVSERVER DVE → gRPC → 0_DATA_CONNECTOR → Pipeline → KNIRVBASE
```

### Validation Tests
| Test | Expected |
|------|----------|
| Guardrail violation → Training triggered | TrainingResponse received |
| Constraint "no exec" → Action denied | `allowed: false` |
| Unknown user → Flagged | `confidence: < 0.5` |
| Mixed training data → Balanced seeds | ConstraintScore ≈ ViolationPenalty |

---

## Open Questions

1. **gRPC Port**: Should the hasher gRPC service run on a fixed port (e.g., `:50051`) or dynamic?

2. **Training Concurrency**: Should multiple training jobs run in parallel, or queue sequentially?

3. **Seed Versioning**: Should we keep history of all trained seeds, or only latest per user?

4. **Fallback Behavior**: If hasher is unavailable, should DVE fall back to existing guardrails only, or block all actions?
