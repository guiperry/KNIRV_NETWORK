# Cognitive Engine Frontend Exposure Analysis

## Overview

This document maps the KNIRVSERVER cognitive engine backend functionality to frontend components, identifying gaps and implementation requirements. The analysis distinguishes between:
- **Primary Dashboard** (`dashboard-wrapper`): Network-wide view across all DVEs
- **Inner DVE Dashboard** (`dve_dual_dashboard`): Per-DVE instance view

---

## Part 1: Backend Functionality Inventory

### Core Cognitive Engine (`cognitive_engine.go`)

| Function | Line | Public API | Description |
|----------|------|------------|-------------|
| `GetLearningState()` | 384-390 | Direct call | Returns LearningState snapshot |
| `GetLearningStateRaw()` | 394-401 | API endpoint | Returns primitives (tasks, success rate, progress, confidence) |
| `GetCognitiveMetrics()` | 365-381 | API endpoint | Per-node cognitive metrics |
| `GetAdaptationHistory()` | 425-434 | API endpoint | Recent adaptation events |
| `GetGuardrailViolations()` | 437-439 | Via guardrailEngine | Policy violations |
| `OnAgentTaskComplexity()` | 450-490 | Callback | Records DVE agent complexity |
| `OnAgentResourceUsage()` | 496-545 | Callback | Records CPU/memory usage |
| `GetOntologyStats()` | 548-550 | API endpoint | Entity and relation counts |
| `GetLatestTelemetry()` | 559-561 | API endpoint | eBPF resource snapshot |

### Guardrail Engine (`guardrail_engine.go`)

| Function | Line | Public API | Description |
|----------|------|------------|-------------|
| `GetViolations()` | 414-427 | API endpoint | Recent policy violations |
| `ViolationCountForNode()` | 430-440 | Internal | Unremediated violation count |
| `GetStatistics()` | 488-518 | API endpoint | Policy/violation statistics |
| `GetActiveViolations()` | 520-531 | API endpoint | Unremediated violations |
| `AddPolicy()` | 266-271 | API endpoint | Register policy rule |
| `GetPolicy()` | 274-279 | API endpoint | Get policy by ID |
| `EnablePolicy()` / `DisablePolicy()` | 558-572 | API endpoint | Toggle policies |

### eBPF Bridge (`ebpf_bridge.go`)

| Function | Line | Public API | Description |
|----------|------|------------|-------------|
| `LatestTelemetry()` | 194-196 | API endpoint | SystemResourceSnapshot |
| `IsNodeIsolated()` | 186-191 | API endpoint | Panic isolation status |
| `TriggerPanicIsolation()` | 162-183 | Internal | Trigger kernel isolation |
| `XDPOffloadManager` | 198-319 | API endpoint | XDP filter management |
| `ResourceQuotaManager` | 321-423 | API endpoint | Resource quota management |
| `ControlPlane` | 425-539 | API endpoint | Control commands |

### Predictive Analytics (`predictive_analytics.go`)

| Function | Line | Public API | Description |
|----------|------|------------|-------------|
| `PredictLoad()` | 70-104 | API endpoint | Load prediction with anomaly detection |
| `GetRecommendedAction()` | 208-225 | API endpoint | Scaling recommendation (scale_up/down/maintain) |
| `GetTrendDirection()` | 227-239 | API endpoint | Metric trend (increasing/decreasing/stable) |
| `GetCapacityForecast()` | 241-260 | API endpoint | Current and projected utilization |
| `RecordMetric()` | 47-68 | Internal | Record metric timeseries |
| `ShouldTriggerProactiveScaling()` | 194-206 | Internal | Scaling trigger decision |

### Ontology Manager (`ontology.go`)

| Function | Line | Public API | Description |
|----------|------|------------|-------------|
| `Stats()` | 279-283 | API endpoint | Entity and relation counts |
| `GetEntity()` | 155-160 | API endpoint | Get entity by ID |
| `QueryByType()` | 163-173 | API endpoint | Query entities by type |
| `FindRelations()` | 176-186 | API endpoint | Find relations for entity |

### Kubernetes Scaler (`kubernetes_scaler.go`)

| Function | Line | Public API | Description |
|----------|------|------------|-------------|
| `GetMetrics()` | 446-455 | API endpoint | K8s replica and utilization metrics |
| `GetStats()` | 623-641 | API endpoint | Full scaler statistics |
| `ScaleReplicas()` | 387-436 | API endpoint | Manual scaling |
| `GetHPA()` | 567-603 | API endpoint | Horizontal Pod Autoscaler status |

---

## Part 2: Frontend Component Analysis

### Primary Dashboard Components (`packages/KNIRVSERVER/frontend/src/components/dashboard/`)

| Component | File | Status | Purpose |
|-----------|------|--------|---------|
| `dashboard-wrapper.tsx` | Main entry | ✅ Existing | Network-wide dashboard shell |
| `cognitive-engine-panel.tsx` | Lines 1-496 | ✅ Existing | Cognitive engine status and controls |
| `dve-nodes-panel.tsx` | - | ✅ Existing | DVE node listing |
| `badge-lab-panel.tsx` | - | ✅ Existing | Badge creation lab |
| `policy-editor.tsx` | - | ✅ Existing | Guardrail policy editor |
| `dve_dual_dashboard.tsx` | - | ✅ Existing | Per-DVE dashboard |
| `neural-desktop-panel.tsx` | - | ✅ Existing | Neural desktop interface |

### Existing Hooks (`packages/KNIRVSERVER/frontend/src/hooks/`)

| Hook | File | Data Provided |
|------|------|---------------|
| `use-cognitive-engine.ts` | 125 lines | Engine status, controls, metrics, tasks |

---

## Part 3: Gap Analysis - Primary Dashboard

### ✅ Already Implemented

| Backend Data | Frontend Location | Component File |
|--------------|-------------------|----------------|
| Cognitive Engine Status | `dashboard-wrapper` → `cognitive` tab | `cognitive-engine-panel.tsx` |
| Engine Controls (Start/Stop/Health) | cognitive-engine-panel | `use-cognitive-engine.ts` |
| Performance Metrics | cognitive-engine-panel | Lines 361-401 |
| Learning Metrics | cognitive-engine-panel | Lines 403-443 |
| Background Tasks | cognitive-engine-panel | Lines 445-487 |
| Guardrail Policies (config) | Onboarding Guide | `policy-editor.tsx`, `onboarding-guide.tsx` |
| Guardrail Config | Onboarding modal | `PreferencesModal.tsx` |

### ❌ Missing - Add to Cognitive Tab

| Backend Functionality | Suggested Component | Data Source Endpoint |
|----------------------|---------------------|---------------------|
| Guardrail Violations List | `guardrail-violations-panel.tsx` | `GET /api/guardrail/violations` |
| Guardrail Statistics | Integrate into panel header | `GET /api/guardrail/statistics` |
| Predictive Analytics | `predictive-analytics-panel.tsx` | `GET /api/analytics/predict` |
| Scaling Recommendations | Add to analytics panel | `GET /api/analytics/recommendations` |
| Capacity Forecast | Add to analytics panel | `GET /api/analytics/forecast` |
| Live Activity Feed | `live-activity-feed.tsx` | `SSE /api/cognitive/events` |

### ❌ Missing - Add to Network Overview

| Backend Functionality | Suggested Component | Data Source Endpoint |
|----------------------|---------------------|---------------------|
| eBPF Telemetry | `system-telemetry-card.tsx` | `GET /api/cognitive/telemetry` |
| Ontology Stats | `knowledge-graph-overview.tsx` | `GET /api/ontology/stats` |

---

## Part 4: Gap Analysis - Inner DVE Dashboard

### Existing Components to Enhance

| Component | File | Enhancement Required |
|-----------|------|---------------------|
| CVEPanel | `dve_dual_dashboard.tsx:537-561` | Replace placeholder with real `AgentMetrics` |
| Security Policy Editor | `dve_dual_dashboard.tsx:317-359` | Connect to guardrail policy API |
| Monitor Panel | `dve_dual_dashboard.tsx:456-535` | Connect to DVE-specific telemetry |

### New Components Needed

| Backend Functionality | Suggested Component | Data Source |
|----------------------|---------------------|-------------|
| DVE Agent Metrics | `dve-agent-metrics-card.tsx` | `AgentDVEMetrics` via hook |
| DVE Resource Usage | `dve-resource-usage.tsx` | CPU%, MemoryMB per DVE |
| DVE Guardrail Status | `dve-guardrail-badge.tsx` | Per-DVE policy status |

---

## Part 5: Required API Endpoints

### Existing Endpoints (Verify)

| Method | Endpoint | Handler |
|--------|----------|---------|
| GET | `/api/cognitive-engine` | `cognitive_engine.go` via web |
| POST | `/api/cognitive-engine` | Actions (start/stop/etc.) |
| GET | `/api/guardrails/policies` | `guardrail_handlers.go` |
| POST | `/api/guardrails/policies` | Create policy |
| PUT | `/api/guardrails/policies/:id` | Update policy |
| DELETE | `/api/guardrails/policies/:id` | Delete policy |

### New Endpoints Required

| Method | Endpoint | Handler File | Purpose |
|--------|----------|--------------|---------|
| GET | `/api/guardrail/violations` | `guardrail_handlers.go:177` | List violations |
| GET | `/api/guardrail/statistics` | `guardrail_handlers.go:204` | Stats summary |
| GET | `/api/cognitive/telemetry` | New: `cognitive_telemetry_handlers.go` | eBPF snapshot |
| GET | `/api/analytics/predict` | New: `analytics_handlers.go` | Load prediction |
| GET | `/api/analytics/recommendations` | New: `analytics_handlers.go` | Scaling action |
| GET | `/api/analytics/forecast` | New: `analytics_handlers.go` | Capacity forecast |
| GET | `/api/ontology/stats` | New: `ontology_handlers.go` | Entity counts |
| GET | `/api/ontology/entities` | New: `ontology_handlers.go` | Entity list |
| GET | `/api/ontology/relations` | New: `ontology_handlers.go` | Relation list |
| GET | `/api/ebpf/isolation-status` | New: `ebpf_handlers.go` | Isolated nodes |
| GET | `/api/ebpf/filters` | `ebpf_bridge.go` | XDP filters |
| GET | `/api/ebpf/quotas` | `ebpf_bridge.go` | Resource quotas |
| SSE | `/api/cognitive/events` | Reuse `event_bus.go` | Live event stream |

---

## Part 6: Implementation Roadmap

### Phase 1: Backend API (Priority: HIGH)
1. Implement `/api/guardrail/violations` endpoint
2. Implement `/api/guardrail/statistics` endpoint
3. Implement `/api/cognitive/telemetry` endpoint
4. Implement analytics endpoints (`/predict`, `/recommendations`, `/forecast`)
5. Implement ontology endpoints (`/stats`, `/entities`, `/relations`)
6. Implement eBPF endpoints (`/isolation-status`, `/filters`, `/quotas`)

### Phase 2: Frontend Hooks (Priority: HIGH)
1. Create `use-guardrail.ts` hook
2. Create `use-analytics.ts` hook
3. Create `use-telemetry.ts` hook
4. Create `use-ontology.ts` hook
5. Create `use-ebpf.ts` hook

### Phase 3: Primary Dashboard Extensions (Priority: HIGH)
1. Extend `cognitive-engine-panel.tsx`:
   - Add Guardrail Statistics card to header
   - Add Guardrail Violations panel below controls
   - Add Predictive Analytics section
   - Add Capacity Forecast card
   - Add Live Activity Feed (SSE)

2. Extend Network Overview cards:
   - Replace static eBPF values with real telemetry
   - Replace static ontology counts with real stats

### Phase 4: Inner DVE Dashboard Enhancements (Priority: MEDIUM)
1. Replace CVEPanel placeholder with real agent metrics
2. Connect Security Policy Editor to real guardrail API
3. Connect Monitor Panel to DVE-specific telemetry
4. Add DVE Resource Usage card

---

## Part 7: Data Flow Diagrams

### Current State (Partial)

```
Backend                    Frontend
─────────────────────────────────────────────────
cognitive_engine.go  ──►  useCognitiveEngine ──► CognitiveEnginePanel
      │                         │
      │                         ▼
      │                  dashboard-wrapper
      │                  (cognitive tab)
      │
guardrail_engine.go
      │                         (gap: no violations UI)
      ▼
   policy-editor.tsx ◄─── (config only, not status)
```

### Target State (Full Exposure)

```
Backend                    Frontend
─────────────────────────────────────────────────
cognitive_engine.go  ──►  useCognitiveEngine ──► CognitiveEnginePanel
      │                         │
      │                         ▼
      │                  dashboard-wrapper
      │                  ├── cognitive tab ──► CognitiveEnginePanel
      │                  │    ├── Engine Controls
      │                  │    ├── Performance Metrics
      │                  │    ├── Learning Metrics
      │                  │    ├── ⚠ Guardrail Violations (NEW)
      │                  │    ├── ⚠ Predictive Analytics (NEW)
      │                  │    ├── ⚠ Capacity Forecast (NEW)
      │                  │    └── ⚠ Live Activity Feed (NEW)
      │                  │
      │                  └── network overview ──► ⚠ Telemetry Card (NEW)
      │                                             ⚠ Ontology Stats (NEW)
      │
guardrail_engine.go  ──►  useGuardrail ──► GuardrailPanel
      │
ebpf_bridge.go      ──►  useTelemetry ──► TelemetryCard
      │
ontology.go         ──►  useOntology ──► KnowledgeGraphCard
      │
predictive_analytics.go ──► useAnalytics ──► PredictionPanel
```

---

## Part 8: Quick Reference

### Frontend File Locations

```
packages/KNIRVSERVER/frontend/src/
├── components/dashboard/
│   ├── dashboard-wrapper.tsx     # Primary entry
│   ├── cognitive-engine-panel.tsx # Cognitive tab content
│   ├── dve_dual_dashboard.tsx     # Inner DVE dashboard
│   ├── dve-nodes-panel.tsx       # Node listing
│   ├── policy-editor.tsx         # Guardrail config
│   └── badge-lab-panel.tsx       # Badge creation
├── hooks/
│   ├── use-cognitive-engine.ts    # Existing cognitive hook
│   ├── use-guardrail.ts          # NEW
│   ├── use-analytics.ts          # NEW
│   ├── use-telemetry.ts          # NEW
│   ├── use-ontology.ts           # NEW
│   └── use-ebpf.ts               # NEW
└── lib/
    └── api.ts                    # API utilities
```

### Backend File Locations

```
packages/KNIRVSERVER/backend/
├── internal/services/cognitiveengine/
│   ├── cognitive_engine.go       # Main engine
│   ├── guardrail_engine.go       # Guardrail system
│   ├── ebpf_bridge.go            # eBPF telemetry
│   ├── ontology.go               # Knowledge graph
│   ├── predictive_analytics.go   # Predictions
│   └── kubernetes_scaler.go     # K8s scaling
└── internal/web/
    ├── cognitive_handlers.go     # Existing handlers
    └── guardrail_handlers.go    # Guardrail handlers
```

---

*Generated: April 2026*
*Analysis Scope: KNIRVSERVER cognitive engine backend → KNIRVGATEWAY frontend mapping*
