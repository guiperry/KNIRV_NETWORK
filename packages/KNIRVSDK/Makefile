# KNIRVSDK build and release.
#
#   make build-sdk      build, stamp and test everything; no upload
#   make release-sdk    build-sdk, then publish the core to R2
#
# The upload follows the KNIRV CLI convention (KNIRV_CORP/packages/cli/Makefile):
# rclone, remote `incline`, bucket path `knirv/...`, served from
# https://releases.knirv.com/...  (knirv/sdk -> https://releases.knirv.com/sdk).
# All three are overridable on the command line.

R2_CMD        ?= rclone
R2_REMOTE     ?= incline
R2_SDK_PREFIX ?= knirv/sdk

SDK_VERSION := $(shell sed -n 's/^version *= *"\(.*\)"/\1/p' core/Cargo.toml | head -1)
DIST        := dist/$(SDK_VERSION)
WASI_OUT    := bindings/wasi/target/wasm32-wasip1/release/knirv_core.wasm

.PHONY: help build-wasm stamp check-stamp check-divergence repro sync-modules build-npm build-go test-go parity build-sdk upload-wasm release-sdk clean

help: ## list targets
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*## /\t/'

build-wasm: ## build the WASM modules, then both cores (WASI for Go, wasm-bindgen for npm); write dist/<version>/ with digests and manifests
	node scripts/sdk-dist.mjs modules
	node scripts/sdk-dist.mjs stamp-modules
	cargo build --release --locked --target wasm32-wasip1 --manifest-path bindings/wasi/Cargo.toml
	mkdir -p $(DIST)
	cp $(WASI_OUT) $(DIST)/knirv_core.wasm
	cd npm-package && node scripts/build.mjs
	node scripts/sdk-dist.mjs manifest

stamp: build-wasm ## write version + digests into the Go constants and npm core.json
	node scripts/sdk-dist.mjs stamp

check-stamp: ## fail unless Cargo.toml, manifest, Go and npm agree
	node scripts/sdk-dist.mjs check

build-npm: stamp ## build and test @knirv/sdk
	cd npm-package && npm test

test-go: ## go test, with the dist core seeded into a throwaway data directory
	@seed=$$(mktemp -d) && mkdir -p $$seed/$(SDK_VERSION) && cp $(DIST)/knirv_core.wasm $$seed/$(SDK_VERSION)/ && \
	(cd go-package && KNIRV_SDK_DATA_DIR=$$seed KNIRV_SDK_TEST_DIST=$(CURDIR)/dist go test ./...) && \
	(cd go-package/models && KNIRV_SDK_DATA_DIR=$$seed go test ./...); \
	rc=$$?; rm -rf $$seed; exit $$rc

parity: ## run the shared requests on the Go (WASI) and TypeScript (wasm-bindgen) hosts; fail on any difference
	@seed=$$(mktemp -d) && mkdir -p $$seed/$(SDK_VERSION) && cp $(DIST)/knirv_core.wasm $$seed/$(SDK_VERSION)/ && \
	(cd go-package && KNIRV_SDK_DATA_DIR=$$seed KNIRV_SDK_TEST_DIST=$(CURDIR)/dist KNIRV_PARITY_OUT=$$seed/go.json go test ./internal/core -run ParityDump >/dev/null) && \
	node scripts/parity.mjs $$seed/go.json; rc=$$?; rm -rf $$seed; exit $$rc

build-go: stamp test-go ## stamp, then test the Go package against the same digest

check-divergence: ## fail unless versions agree and no wrapper re-implements core logic (scripts/check-sdk-divergence.mjs)
	node scripts/check-sdk-divergence.mjs

repro: ## clean-target rebuild of modules + both cores into a temp dir; verify the released digests (path-dependent: run from the release checkout)
	node scripts/sdk-dist.mjs repro

# check-divergence runs first: a stale Go stamp fails it with the fix (make stamp).
build-sdk: check-divergence build-npm build-go parity check-stamp ## everything, no upload

upload-wasm: check-stamp ## publish dist/ to R2 (identical files are skipped by checksum; changed files for the same version are overwritten)
	@command -v "$(R2_CMD)" >/dev/null 2>&1 || { echo "ERROR: $(R2_CMD) not found. Install/configure rclone or set R2_CMD."; exit 1; }
	@test -f $(DIST)/knirv_core.wasm && test -f $(DIST)/manifest.json || { echo "ERROR: $(DIST) is incomplete; run make build-wasm"; exit 1; }
	@# --checksum: compare content, not modtime. Unchanged files are skipped; changed files are overwritten.
	$(R2_CMD) copy --checksum -v $(DIST) "$(R2_REMOTE):$(R2_SDK_PREFIX)/$(SDK_VERSION)/"
	$(R2_CMD) copy --checksum dist/latest "$(R2_REMOTE):$(R2_SDK_PREFIX)/latest/"
	@echo "Published $(SDK_VERSION) to $(R2_REMOTE):$(R2_SDK_PREFIX)/ (https://releases.knirv.com/sdk/$(SDK_VERSION)/)"

# sync-modules: rebuild the WASM modules and replace the tracked bytes when the
# module sources changed since the last stamp. Refuses when the sources are
# unchanged but the bytes differ (see sync-modules in scripts/sdk-dist.mjs).
sync-modules: ## rebuild modules and replace tracked bytes if their sources changed
	node scripts/sdk-dist.mjs sync-modules

# If build-sdk fails, try a module sync once and rebuild. A sync that finds no
# source change exits non-zero, so an unrelated failure still stops the release.
release-sdk: ## build, stamp, test and publish
	$(MAKE) build-sdk || { $(MAKE) sync-modules && $(MAKE) build-sdk; }
	$(MAKE) upload-wasm

clean: ## remove dist/
	rm -rf dist
