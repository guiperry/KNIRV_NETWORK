In general, the memory for an agent is something that we provide via context in the prompt passed to LLM that helps the agent to better plan and react given past interactions or data not immediately available.

It is useful to group the memory into four types:

𝟭. 𝗘𝗽𝗶𝘀𝗼𝗱𝗶𝗰 - This type of memory contains past interactions and actions performed by the agent. After an action is taken, the application controlling the agent would store the action in some kind of persistent storage so that it can be retrieved later if needed. A good example would be using a vector Database to store semantic meaning of the interactions.

𝟮. 𝗦𝗲𝗺𝗮𝗻𝘁𝗶𝗰 - Any external information that is available to the agent and any knowledge the agent should have about itself. You can 
think of this as a context similar to one used in RAG applications. It can be internal knowledge only available to the agent or a grounding context to isolate part of the internet scale data for more accurate answers.

𝟯. 𝗣𝗿𝗼𝗰𝗲𝗱𝘂𝗿𝗮𝗹 - This is systemic information like the structure of the System Prompt, available tools, guardrails etc. It will usually be stored in Git, Prompt and Tool Registries.

𝟰. Occasionally, the agent application would pull information from long-term memory and store it locally if it is needed for the task at hand.

𝟱. All of the information pulled together from the long-term or stored in local memory is called short-term or working memory. Compiling all of it into a prompt will produce the prompt to be passed to the LLM and it will provide further actions to be taken by the system.

We usually label 1. - 3. as Long-Term memory and 5. as Short-Term memory.

And that is it! The rest is all about how you architect the topology of your Agentic Systems.


Below is a **complete, build-ready implementation plan** for building a complete Agentic Memory Platform.

---

# 🧠 Agentic Memory – Full Implementation Plan

## 0. Guiding Principles
- **Separation of concerns**: Core orchestrator never stores state; all memory is a service.  
- **Vector-first**: Anything that can be embedded lives in the same latent space → one ANN lookup.  
- **Privacy-by-design**: PII is encrypted at rest, anonymised in vectors, user-deletable (GDPR).  
- **Versioned memory**: Every write produces an immutable revision → reproducible reasoning.  
- **Tool-centric**: “If it isn’t in the Tool Registry the agent can’t do it.”

---

## 1. High-Level Architecture (deliverable: C4 diagram + this doc)

| Component | Responsibility | Tech Candidate |
|-----------|----------------|----------------|
| 1. Core Orchestrator | ReAct loop, prompt assembly, tool dispatch | Python (FastAPI + async) |
| 2. LLM Gateway | Unified OAI / Azure / Bedrock / OSS router | LiteLLM or LangChain-OpenAPI |
| 3. Memory Manager | CRUD façade over all memory types | Same service as #1 |
| 4. Vector Service | Embedding, ANN, metadata filtering | pgvector / Qdrant / Pinecone |
| 5. Episodic Store | Time-series JSON/BLOB | PostgreSQL (partitioned) |
| 6. Semantic Store | Concept triples + vectors | Neo4j + pgvector |
| 7. Procedural Store | Skills, checklists, SOPs | PostgreSQL (or Git) |
| 8. Tool Registry | OpenAPI specs + auth + rate-limit | FastAPI + Keycloak |
| 9. Privacy Layer | Encryption, anonymiser, purge jobs | AES-256 + Presidio |
|10. Observability | Tracing, metrics, evaluator | OpenTelemetry + Prometheus + Grafana |

---

## 2. Data Schema & Contracts

### 2.1 Unified Memory Record (UMR) – every memory object inherits this
```json
{
  "id": "uuid7",
  "user_id": "hash",
  "session_id": "uuid",
  "memory_type": "episodic|semantic|procedural",
  "created_at": "2025-09-11T14:23:45Z",
  "revision": 3,
  "visibility": "private|shared|public",
  "retention_policy": "90_days|forever|task_end",
  "payload": { ...type-specific blob... },
  "embedding": [0.01, ..., 0.43],
  "metadata": { "tool": "EmailSender", "outcome": "success", ... }
}
```

### 2.2 Embedding contract
- **Dimensions**: 1536 (OpenAI v3 small) – configurable.  
- **Normalisation**: L2 → cosine distance.  
- **Sparse-dense hybrid**: optional SPLADE for keyword boost.

### 2.3 ANN search request/response
```
POST /memory/search
body: { "query": "user likes jazz", "filter": { "memory_type": "semantic" }, "top_k": 5, "user_id": "hash" }
→ [{UMR+score}, ...]
```

---

## 3. Memory Types – Detailed Build Tasks

### ✅ 3.1 Short-Term (Working) Memory
- **Scope**: last 5–7 turns or 4k tokens.  
- **Storage**: in-process LRU deque.  
- **Eviction policy**: FIFO after token threshold; checkpoint to Episodic every turn.  
- **Task**: implement `WorkingBuffer` class with `checkpoint()` method.

### ✅ 3.2 Episodic Memory
- **Schema**: turn-by-turn JSON with action, observation, reward, timestamp.  
- **Partitioning**: by `user_id` + monthly partitions.  
- **Retention**: auto-purge after `retention_policy`.  
- **Task**:
  1. Create `episodic` table.  
  2. Implement `write_episode(turn)` DAO.  
  3. Background job to compress old episodes into vector summaries (see 3.5).

### ✅ 3.3 Semantic Memory
- **Schema**: subject-predicate-object triples + vector.  
- **Storage**: Neo4j for edges, pgvector for vectors.  
- **Merge strategy**: UPSERT on `(subj, pred)`; vector = mean(old, new).  
- **Task**:
  1. Define core ontology (Person, Topic, Preference, Constraint).  
  2. Implement `upsert_triple()` and `triple_to_vector()` service.  
  3.Expose `/memory/semantic` REST endpoints.

### ✅ 3.4 Procedural Memory
- **Schema**: `{name, description, param_schema, steps[], vector}`.  
- **Storage**: PostgreSQL JSONB; versioned via `revision`.  
- **Task**:
  1. Create `procedural` table.  
  2. Build `SkillCompiler` that turns YAML → JSONB + embedding.  
  3. Integrate with Tool Registry (procedural skills appear as tools).

### ✅ 3.5 Approximate Reasoning History (compressed episodes)
- **Idea**: nightly job summarises 100-turn episode → 1 paragraph → embedding.  
- **Task**:
  1. Prompt LLM with “Summarise the key events and user preferences”.  
  2. Store paragraph in Semantic store with type=“compressed_episode”.  
  3. Delete raw episodes older than 90 days (GDPR).

---

## 4. Vector Layer Implementation

| Sub-Task | Details |
|----------|---------|
| 4.1 Embedding micro-service | Wrap OpenAI, Cohere, OSS (BAAI/bge). Route per model alias. |
| 4.2 Vector DB choice matrix | pgvector (self-host), Qdrant (k8s), Pinecone (SaaS) – pick one per env. |
| 4.3 Index tuning | IVFFlat or HNSW; `m=16`, `ef_construct=64`; benchmark 1M @ <50 ms. |
| 4.4 Metadata filtering | Add composite index on `(user_id, memory_type, created_at)`. |
| 4.5 Embedding cache | Redis 7 with vector module → 30 % cost saving. |

---

## 5. Tool Registry & Grounding

- **Spec**: OpenAPI 3.1 + x-memory hooks.  
- **Registration flow**:  
  1. Developer pushes YAML to Git.  
  2. CI validates + embeds description → registers in DB.  
  3. Memory Manager links tool_id to UMRs (grounding).  
- **Runtime**: Orchestrator loads `AvailableTools` list into prompt; top-k selected by vector similarity to current intent.  
- **Task list**:
  - ✅ Design JSON schema for `x-memory`.  
  - ✅ Build CLI `swirl tool register <file>`.  
  - ✅ Add RBAC: user can only see tools they own or public.

---

## 6. Privacy, Security, Compliance

| Control | Implementation |
|---------|----------------|
| Encryption | AES-256-GCM at rest (PG), TLS 1.3 in transit. |
| Anonymisation | Presidio SDK in ingestion pipeline; mask emails, phones. |
| Vector anonymisation | Hash user_id → uint64 before storing in vector DB. |
| Right-to-be-forgotten | Async job deletes all UMR for `user_id` + re-indexes. |
| Audit log | Immutable append-only log (Loki) for every CRUD. |

---

## 7. Orchestrator Prompt Structure (v1)

```
You are SwirlAI agent.
Short-term memory:
{working_buffer}

Relevant past episodes:
{episodic_top_3}

Relevant facts:
{semantic_top_5}

Available tools:
{tools_top_10}

User prompt: {user_input}

Thought 1: ...
Action 1: ...
```

- **Task**: implement `PromptAssembler` class; unit test with 100 synthetic dialogs.

---

## 8. Evaluation & Metrics

| Metric | Target | Tool |
|--------|--------|------|
| Recall@5 episodic | ≥0.85 | Human-labelled 500 dialogs |
| Recall@5 semantic | ≥0.90 | Same |
| ANN latency p95 | <80 ms | K6 load test |
| Memory cost / user / month | <$0.05 | AWS Cost Explorer |
| GDPR delete <72 h | 100 % | Periodic audit |

---

## 9. CI/CD & Release Flow

1. **PR opened** → unit tests (pytest) + lint (ruff).  
2. **Merge to main** → build Docker images (core, vector, db-migrator).  
3. **Deploy to staging** (helm chart).  
4. **Evaluation pipeline** runs nightly → posts metrics to Grafana.  
5. **Manual approval** → prod k8s; blue-green with 5 % canary.

---

## 10. Delivery Milestones (12-week plan)

| Week | Deliverable |
|------|-------------|
| 1 | Finalised schema, repo scaffold, docker-compose dev env. |
| 2 | Episodic store + basic REST CRUD. |
| 3 | Embedding service + pgvector integration. |
| 4 | Semantic triple store + search endpoint. |
| 5 | Working memory buffer + orchestrator loop. |
| 6 | Tool Registry + first 5 internal tools. |
| 7 | Privacy layer + anonymiser. |
| 8 | Compression job + procedural store. |
| 9 | End-to-end ReAct demo (internal). |
|10 | Evaluation suite + latency optimisation. |
|11 | Security audit + GDPR delete workflow. |
|12 | Prod deploy + public beta invite. |

---

## 11. Run-Books (snippet)

**Hot-fix: ANN latency spike**  
1. Check `pgvector` index with `EXPLAIN ANALYZE`.  
2. If rows >1 M and recall <0.8 → increase `lists` or migrate to Qdrant HNSW.  
3. Scale read replicas.

**Incident: memory leak in working buffer**  
1. Set env `WORKING_BUFFER_MAX_TOKENS=2048`.  
2. Restart pod; buffer truncates automatically.

---

## 12. Future Extensions (post v1)

- Multi-modal embeddings (image, audio).  
- Federated memory (share anonymised vectors across tenants).  
- Reinforcement learning layer on top of episodic rewards.  
- On-device vector DB for edge agents (SQLite-vss).

---

Copy this markdown into your internal wiki, create Jira tickets from every ✅ item, and start shipping the **Agentic Memory** platform.


Build credit:
x.com/@Aurimas_Gr
linkedin.com/in/aurimas-griciunas