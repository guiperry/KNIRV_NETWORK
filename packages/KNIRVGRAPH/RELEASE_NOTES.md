# Release notes

## GraphRAG consolidation

- Added persistent HNSW search with cosine, dot-product, and Euclidean metrics,
  deterministic optimization, incremental upsert/delete, and Badger recovery.
- Added out-of-process Candle and GLiNER providers without CGo linkage.
- Persisted document status, chunks, entities, relationships, semantic graph
  artifacts, and delete/overwrite manifests.
- Migrated DRQ embeddings to the shared KNIRVGRAPH embedding service and added
  bounded host-side retrieval context for DVE skill execution.
- Added Prometheus RAG metrics, W3C trace propagation, structured request logs,
  detailed readiness, streaming backup/restore hooks, and scheduled compaction.
- Migrated backend knowledge retrieval to `graph.sock` and removed the embedded
  KNIRVSERVER GraphRAG package, startup flag, and static-library build target.

See [docs/GRAPHRAG_MIGRATION.md](docs/GRAPHRAG_MIGRATION.md) for configuration,
validation, backup, and rollback instructions.
