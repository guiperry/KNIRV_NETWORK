# Makefile for KNIRV Network Deployment and Releases

# Default environment
ENV ?= production

# Get current git tag or commit hash
GIT_TAG := $(shell git describe --tags --always --abbrev=0)
VERSION ?= $(GIT_TAG)

.PHONY: help releases build-all upload-assets update-portal-links

help:
	@echo "KNIRV Network Makefile"
	@echo "----------------------"
	@echo "Available targets:"
	@echo "  make releases         - Build all binaries, create a GitHub release, and upload assets."
	@echo "  make build-all        - Build all application binaries for all platforms."
	@echo "  make upload-assets    - Upload built assets to the GitHub release (requires VERSION)."
	@echo "  make update-portal-links - Update portal-links.yaml with new release asset URLs (requires VERSION)."

# =============================================================================
# RELEASE AUTOMATION
# =============================================================================

releases:
	@echo "🚀 Starting full release process for version $(VERSION)..."
	@./scripts/release.sh all $(VERSION)

build-all:
	@echo "🏗️ Building all binaries for version $(VERSION)..."
	@./scripts/release.sh build $(VERSION)

upload-assets:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is not set. Use: make upload-assets VERSION=<tag>"; exit 1; fi
	@echo "⬆️ Uploading assets for version $(VERSION)..."
	@./scripts/release.sh upload $(VERSION)

update-portal-links:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is not set. Use: make update-portal-links VERSION=<tag>"; exit 1; fi
	@echo "🔗 Updating portal links for version $(VERSION)..."
	@./scripts/release.sh update-links $(VERSION)