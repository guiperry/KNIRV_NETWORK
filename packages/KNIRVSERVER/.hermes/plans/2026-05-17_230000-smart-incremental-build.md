# Smart Incremental Build — Proposal

## Goal

`make relink` (or a new `make build-changed`) should detect which submodules have source changes and only rebuild those, then relink the final unified binary. A no-op run (everything up to date) should be instant.

## Current Architecture

```
binary: proto ebpf-generate frontend backend
  └── go build . → dist/knirv-server (embeds bin/*, frontend/out/*, pkg/*/bin/*)

backend: deps-go gateway-build graph-build chain-build oracle-build hasher-build shell-build agent-build embedded-build
  └── go build ./cmd/backend_server → bin/backend_server (gzipped)

Each submodule target (gateway-build, etc.) is .PHONY → always runs its recipe.
```

Every submodule target runs its recipe even when nothing changed. While `go build` uses its cache (fast for unchanged code), the shell overhead of `mkdir`, `cp`, `gzip` still fires, and embedded Rust/cargo builds (graphrag, validation-chain) always re-run.

## Proposed Approach: File-Target Shift

**Core idea:** Remove submodule build targets from `.PHONY` and make them into real file targets. Make will then use its built-in timestamp comparison to skip unchanged modules.

### Dependency Graph (Proposed)

```
dist/knirv-server  ──depends-on──►  bin/backend_server  (gzipped)
                                    frontend/out/*       (via frontend target)
                                    config/*
                                    bin/root.key

bin/backend_server  ──depends-on──►  backend/cmd/backend_server/**/*.go
                                     pkg/*/bin/*.gz       (all submodule binaries)

pkg/knirvgateway/bin/knirvgateway  ──depends-on──►  ../KNIRVGATEWAY/**/*.go
                                                     internal/embedded/webgui/out/*

pkg/knirvgraph/bin/knirvgraph  ──depends-on──►  ../KNIRVGRAPH/**/*.go
... (same pattern for chain, oracle, hasher, shell, agent)
```

### How It Works

Make's timestamp rule: if any prerequisite is newer than the target file, the recipe runs. Otherwise, the target is considered up-to-date and the recipe is skipped entirely (no shell execution, no `go build`, no nothing).

## Step-by-Step Plan

### Phase 1: Source File Helper Macro

Add a Makefile macro that collects source files for a given directory. This avoids hardcoding file lists.

```makefile
# Collect all Go source files under a directory (recursive, 2 levels for go.mod root)
GO_SRCS = $(shell find $(1) -maxdepth 4 -name '*.go' -not -path '*/vendor/*' -not -path '*/.git/*' 2>/dev/null | sort)
```

Also add macros for specific source types:

```makefile
WEBGUI_OUT = $(WEBGUI_SRC_DIR)/out
FRONTEND_OUT = $(FRONTEND_DIR)/out
```

### Phase 2: Convert Submodule Targets to File Targets

Each submodule binary becomes a real file target. For example:

```makefile
# Old .PHONY target:
# gateway-build: webgui-build gateway-deps
#   ... always runs

# New file target (remove from .PHONY):
$(GATEWAY_PKG_BIN): $(call GO_SRCS,$(GATEWAY_SRC_DIR)) $(WEBGUI_OUT) $(GATEWAY_SRC_DIR)/go.mod $(GATEWAY_SRC_DIR)/go.sum
	@mkdir -p $(dir $@)
	@rm -f $@
	cd $(GATEWAY_SRC_DIR) && go build -ldflags "-s -w" -o $@ ./cmd/gateway
	@rm -f $@.tmp
	@gzip -cf $@ > $@.tmp && mv $@.tmp $@

# Keep .PHONY variant for explicit invocation:
.PHONY: gateway-build-force
gateway-build-force:
	@rm -f $(GATEWAY_PKG_BIN)
	$(MAKE) $(GATEWAY_PKG_BIN)
```

Same pattern for graph, chain, oracle, hasher, shell, agent.

### Phase 3: Handle go.mod/go.sum as Dependencies

Adding `go.mod` and `go.sum` as prerequisites ensures that dependency changes (adding/removing packages) trigger a rebuild:

```makefile
$(GATEWAY_PKG_BIN): $(call GO_SRCS,$(GATEWAY_SRC_DIR)) $(WEBGUI_OUT) \
                     $(GATEWAY_SRC_DIR)/go.mod $(GATEWAY_SRC_DIR)/go.sum
```

### Phase 4: Handle the WebGUI Special Case

`gateway-build` currently depends on `webgui-build` (npm). The npm build is always slow. Make it a file target too:

```makefile
$(WEBGUI_OUT)/index.html: $(shell find $(WEBGUI_SRC_DIR)/src -type f 2>/dev/null) \
                           $(WEBGUI_SRC_DIR)/package.json $(WEBGUI_SRC_DIR)/package-lock.json
	cd $(WEBGUI_SRC_DIR) && npm run build
```

However, `npm run build` writes to `out/` and Next.js handles its own incremental compilation. Since tracking every file in `src/` is fragile, a practical approach is:

```makefile
# Stamp-based: only rebuild WebGUI if package.json or key config changed
$(WEBGUI_OUT)/index.html: $(WEBGUI_SRC_DIR)/package.json $(WEBGUI_SRC_DIR)/next.config.js
	cd $(WEBGUI_SRC_DIR) && npm run build
```

Or skip smart detection for WebGUI and accept it always runs (it's already cached by Next.js).

### Phase 5: Convert the `backend` Target

The `backend_server` binary (gzipped in `bin/`) depends on the backend Go source AND all submodule binaries:

```makefile
bin/backend_server: $(call GO_SRCS,$(BACKEND_DIR)/cmd/backend_server) \
                    $(call GO_SRCS,$(BACKEND_DIR)/internal) \
                    $(GATEWAY_PKG_BIN) $(GRAPH_PKG_BIN) $(CHAIN_PKG_BIN) \
                    $(ORACLE_PKG_BIN) $(HASHER_PKG_BIN) $(SHELL_PKG_BIN) $(AGENT_PKG_BIN) \
                    $(BACKEND_DIR)/go.mod $(BACKEND_DIR)/go.sum
	@rm -f $(BACKEND_DIR)/backend_server $@
	cd $(BACKEND_DIR) && go build -ldflags "$(LDFLAGS)" -o backend_server ./cmd/backend_server
	cp $(BACKEND_DIR)/backend_server $@
	@rm -f $@.tmp
	@gzip -cf $@ > $@.tmp && mv $@.tmp $@
	rm $(BACKEND_DIR)/backend_server
```

Note: `$(call GO_SRCS,$(BACKEND_DIR)/cmd/backend_server)` captures just the backend_server source, not the entire backend tree. The `internal/` packages are covered by the second `GO_SRCS` call.

### Phase 6: Convert the `binary` Target

The final `dist/knirv-server` depends on the embedded files:

```makefile
dist/knirv-server: bin/backend_server $(FRONTEND_OUT) main.go go.mod go.sum
	@mkdir -p $(DIST_DIR)
	@rm -f $@
	go build -ldflags "$(LDFLAGS)" -o $@ .
	chmod 755 $@
	# Copy to deployer locations
	@mkdir -p $(APPDATA_DIR)/container_deployer/resources/golang-app-source
	@rm -f $(APPDATA_DIR)/container_deployer/resources/golang-app-source/knirv-server
	cp $@ $(APPDATA_DIR)/container_deployer/resources/golang-app-source
	chmod +x $(APPDATA_DIR)/container_deployer/resources/golang-app-source/knirv-server
	@mkdir -p $(BACKEND_DIR)/cmd/container_deployer/golang-app-source
	@rm -f $(BACKEND_DIR)/cmd/container_deployer/golang-app-source/knirv-server
	cp $@ $(BACKEND_DIR)/cmd/container_deployer/golang-app-source
	chmod +x $(BACKEND_DIR)/cmd/container_deployer/golang-app-source/knirv-server
```

### Phase 7: Update `relink`

`relink` becomes an alias for building `dist/knirv-server` — make naturally skips everything that's already fresh:

```makefile
relink: dist/knirv-server
```

That's it. No separate "fast path" needed — the file targets handle it.

### Phase 8: Keep `.PHONY` Wrappers

Provide `.PHONY` wrappers for explicit full rebuilds:

```makefile
# Force full rebuild of everything
build-full: clean binary

# Force a specific submodule
gateway-build: clean-gateway $(GATEWAY_PKG_BIN)
gateway-build-force:
	@rm -f $(GATEWAY_PKG_BIN)
	$(MAKE) $(GATEWAY_PKG_BIN)
```

## Files That Change

| File | Change |
|------|--------|
| `Makefile` | Remove submodule targets from `.PHONY`; add file-target versions; add `GO_SRCS` macro; update `backend` and `binary` targets; simplify `relink` |

## Edge Cases / Risks

### 1. go.mod changes in root
If `main.go`'s dependencies change (go.mod/go.sum in root), the `binary` target triggers a rebuild because `go.mod`/`go.sum` are prerequisites. ✅

### 2. Submodule dependency chain (e.g., backend imports gateway pkg)
The backend's `go build` handles this via Go's import graph and cache. If you change the gateway's Go source, it triggers gateway binary rebuild (via file deps), which changes `pkg/knirvgateway/bin/knirvgateway`'s timestamp, which triggers backend binary rebuild (via file deps). ✅

### 3. Dirty git working tree
`find` looks at actual filesystem timestamps, not git — works regardless of git state. `git ls-files` is not used to avoid this issue. ✅

### 4. First build (no binary exists yet)
The file target doesn't exist, so make runs the recipe. ✅

### 5. Typo: missing optional packages
The `find` command for `GO_SRCS` uses `2>/dev/null` — if a submodule dir is missing, it returns empty list, and the build skips that module. An explicit error check can be added:

```makefile
$(GATEWAY_PKG_BIN): $(call GO_SRCS,$(GATEWAY_SRC_DIR)) ...
	@test -d "$(GATEWAY_SRC_DIR)" || { echo "ERROR: KNIRVGATEWAY source not found"; exit 1; }
	...
```

### 6. `go build` has its own cache
Even when make decides to run `go build` (because a dependency is newer), Go's build cache still prevents recompiling individual packages that haven't changed. The `-o` flag forces linking but compilation is cached. So rebuild time is bounded by link time, not compile time. ✅

## Verification / Testing

1. `make dist/knirv-server` with no changes → should be instant ("Nothing to be done")
2. Touch a single `.go` file in `../KNIRVCHAIN/` → only chain-build + backend + binary should re-run
3. Touch `main.go` → only the final `go build` should re-run
4. `make clean && make dist/knirv-server` → full clean build
5. Run `touch` on each submodule individually and verify the right targets fire

## Open Questions

1. **WebGUI npm build**: Should we stamp-track it (package.json timestamp) or accept it always runs via `.PHONY`? The npm build is slowest part. Next.js has its own cache (`out/` dir), but the initial npm build takes ~30s. Stamping with `package.json` and `next.config.js` is the best tradeoff.

2. **Frontend npm build**: Same question for `frontend/out/*`. Currently `frontend` is a `.PHONY` target that always runs `npm run build`. If we make the frontend out dir a file target, we need its deps. Next.js tracks its own incremental compilation, so stamp-based tracking works.

3. **Graphrag + validation-chain (Rust cargo builds)**: These are the slowest. They should be stamp-tracked with their source dirs. But `cargo build` has its own incremental compilation, so the Rust crate itself handles caching. We just need to avoid calling `cargo build` when nothing changed.

4. **Make variable expansion**: `$(call GO_SRCS,$(DIR))` expands eagerly at parse time. For 7 submodules × recursive find, this could add ~100ms to make startup. Acceptable for a build tool.

5. **Backward compat**: Existing workflows (`make binary`, `make backend`, `make gateway-build`) must keep working. The plan keeps `binary` and `backend` as `.PHONY` targets that delegate to the file targets, preserving muscle memory.
