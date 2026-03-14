# KNIRV Network Package Refactoring Plan

## Overview

Following the migration of all KNIRV submodules into the `packages/` directory, the root-level project tools and resources require updates to reference the new module locations. Each submodule remains fully independent and can be developed/deployed separately.

---

## Module Name Changes

| Old Name | New Name | Notes |
|----------|----------|-------|
| KNIRVSERVER | KNIRVSERVER | DVE/Validation Environment |
| KNIRVCONTROLLER | KNIRVWALLET | AI Controller PWA (future rename to KNIRVCONTROLLER planned) |

---

## Updated Package Structure

```
KNIRV_NETWORK/
├── packages/
│   ├── KNIRVARENA      (Node.js - WASM Controller)
│   ├── KNIRVBASE       (Go - Base Node)
│   ├── KNIRVCHAIN      (Rust - Blockchain)
│   ├── KNIRVCLI        (Go - CLI Tool)
│   ├── KNIRVCORE       (Go - Core)
│   ├── KNIRVCORTEX     (Go - Cortex)
│   ├── KNIRVGATEWAY    (Node.js - API Gateway)
│   ├── KNIRVGRAPH      (Go - Knowledge Graph)
│   ├── KNIRVHEART      (Go - Heart)
│   ├── KNIRVHUB        (Go - Hub)
│   ├── KNIRVRAMP       (Go - Ramp)
│   ├── KNIRVROUTER     (Go - P2P Router)
│   ├── KNIRVSDK        (Go - SDK)
│   ├── KNIRVSERVER     (Go - DVE/Nexus replacement)
│   ├── KNIRVSYNC       (Go - Sync)
│   ├── KNIRVTESTNET    (Go - Testnet)
│   └── KNIRVWALLET     (Node.js - Controller PWA)
├── scripts/             (Root-level utility scripts)
├── deployment/          (Ansible & Docker configs)
├── integration-tests/   (Cross-component tests)
├── Makefile            (Root build system)
└── shared-proto/       (Protobuf definitions)
```

---

## Phase 1: Root Makefile

**File:** `Makefile`

**Changes:** ~60+ targets using directory variables need `packages/` prefix.

| Variable | Current | Update To |
|----------|---------|-----------|
| `KNIRVGATEWAY_DIR` | `$(PROJECT_ROOT)/KNIRVGATEWAY` | `$(PROJECT_ROOT)/packages/KNIRVGATEWAY` |
| `KNIRVSDK_DIR` | `$(PROJECT_ROOT)/KNIRVSDK` | `$(PROJECT_ROOT)/packages/KNIRVSDK` |
| `KNIRVGRAPH_DIR` | `$(PROJECT_ROOT)/KNIRVGRAPH` | `$(PROJECT_ROOT)/packages/KNIRVGRAPH` |
| `KNIRVWALLET_DIR` | `$(PROJECT_ROOT)/KNIRVWALLET` | `$(PROJECT_ROOT)/packages/KNIRVWALLET` |
| `KNIRVROUTER_DIR` | `$(PROJECT_ROOT)/KNIRVROUTER` | `$(PROJECT_ROOT)/packages/KNIRVROUTER` |
| `KNIRVTESTNET_DIR` | `$(PROJECT_ROOT)/KNIRVTESTNET` | `$(PROJECT_ROOT)/packages/KNIRVTESTNET` |
| `KNIRVCHAIN_DIR` | `$(PROJECT_ROOT)/KNIRVCHAIN` | `$(PROJECT_ROOT)/packages/KNIRVCHAIN` |

---

## Phase 2: Root Scripts (High Priority)

### Name Change Mappings
- All `KNIRVCONTROLLER` references → `KNIRVWALLET` (keep future rename capability via alias)
- All `KNIRVSERVER` references → `KNIRVSERVER`

### Critical Files
| File | Changes |
|------|---------|
| `scripts/deploy-testnet-services.sh` | TESTNET_DIR, docker contexts, service starts |
| `scripts/kill_knirv.sh` | Service detection, KNIRVSERVER→KNIRVSERVER |
| `scripts/manage-knirv.sh` | Module paths |
| `scripts/deploy-controller-pwa.sh` | CONTROLLER_DIR path |
| `scripts/run-full-demo.sh` | CONTROLLER_DIR path |
| `scripts/validate-demo-setup.sh` | KNIRVCONTROLLER directory check |
| `scripts/sync_controller_to_public.sh` | Keep sync path flexible for future rename |
| `scripts/sync-portal-versions.sh` | KNIRVSERVER→KNIRVSERVER |

---

## Phase 3: Submodule Internal Scripts

### packages/KNIRVTESTNET/scripts/
| Script | Changes |
|--------|---------|
| `build-knirvchain.sh` | `../KNIRVCHAIN` → `../packages/KNIRVCHAIN` |
| `build-knirvgraph.sh` | `../KNIRVGRAPH` → `../packages/KNIRVGRAPH` |
| `build-knirvrouter.sh` | `../KNIRVROUTER` → `../packages/KNIRVROUTER` |
| `build-local-release.sh` | Update all `../KNIRV*` paths to `../packages/KNIRV*` |
| `start-testnet.sh` | Service startup paths |
| `stop-testnet.sh` | Service paths |
| `test-build.sh` | Dockerfile contexts |

---

## Phase 4: Ansible Playbooks

| File | Changes |
|------|---------|
| `deployment/ansible/deploy-knirvcontroller-pwa.yml` | KNIRVCONTROLLER→KNIRVWALLET path |
| `deployment/ansible/update-gateway-endpoints.yml` | gateway_root path |
| `deployment/ansible/environments/*.yml` | frontend_build_dir, CDN paths |
| `deployment/ansible/infrastructure-playbook.yml` | Service paths |
| `deployment/ansible/testnet-infrastructure-playbook.yml` | Testnet paths |

---

## Phase 5: Docker Compose

| File | Changes |
|------|---------|
| `packages/KNIRVTESTNET/docker-compose.yml` | Context: `../KNIRVCHAIN` → `../packages/KNIRVCHAIN` |
| `deployment/docker-compose.knirv-production.yml` | Build contexts |

---

## Phase 6: Integration Tests

| File | Changes |
|------|---------|
| `integration-tests/run_all_integration_tests.sh` | Docker build paths, KNIRVSERVER→KNIRVSERVER |
| `integration-tests/config/run-tests.sh` | Service startup paths |
| `integration-tests/config/setup.sh` | Build directory references |

---

## Phase 7: Keep Future Rename Flexibility

For KNIRVCONTROLLER → KNIRVWALLET (potential future rename):

1. Create an alias mechanism in scripts that check both:
   ```bash
   CONTROLLER_DIR="${PROJECT_ROOT}/packages/KNIRVWALLET"
   if [ -d "${PROJECT_ROOT}/packages/KNIRVCONTROLLER" ]; then
       CONTROLLER_DIR="${PROJECT_ROOT}/packages/KNIRVCONTROLLER"  # Legacy
   fi
   ```

2. Or use environment variable override:
   ```bash
   CONTROLLER_DIR="${KNIRV_CONTROLLER_DIR:-${PROJECT_ROOT}/packages/KNIRVWALLET}"
   ```

---

## Summary Statistics

| Category | Approximate Files |
|----------|-------------------|
| Root Makefile | 1 |
| Root Scripts | ~20 critical |
| Submodule Scripts (packages/) | ~15 |
| Ansible Playbooks | ~8 |
| Docker Compose | ~3 |
| Integration Tests | ~5 |
| **Total** | **~52 files** |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Missing path references | Use `grep` to search for remaining root-level module references |
| Breaking individual builds | Test each module independently after changes |
| Service port conflicts | Verify testnet port configurations |

---

## Validation Commands

After updates, verify with:

```bash
# Verify Makefile loads correctly
make help

# Test testnet functionality
make testnet-tests

# Verify individual modules
cd packages/KNIRVCHAIN && cargo build
cd packages/KNIRVGATEWAY && npm run build
```
