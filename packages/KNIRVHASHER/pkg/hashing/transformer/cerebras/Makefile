# HEART Transformer Makefile for Cerebras WSE
# Provides targets for compilation, weight export, and execution

SHELL := /bin/bash

# Directories
CEREBRAS_DIR := .
BUILD_DIR := build
WEIGHTS_DIR := weights
DATA_DIR := data

# Cerebras SDK commands
CSLC := cslc
CS_PYTHON := cs_python

# Configuration
LAYOUT_FILE := layout.csl
PARAMS_FILE := params.json
PROGRAM_NAME := heart_program

# Default parameters (can be overridden)
GRID_WIDTH ?= 8
GRID_HEIGHT ?= 8
FABRIC_DIMS := $(GRID_WIDTH),$(GRID_HEIGHT)
FABRIC_OFFSETS := 4,1

# Targets
.PHONY: all clean compile weights test run help

all: compile ## Build everything

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(WEIGHTS_DIR):
	mkdir -p $(WEIGHTS_DIR)

$(DATA_DIR):
	mkdir -p $(DATA_DIR)

compile: $(BUILD_DIR) ## Compile CSL program for Cerebras WSE
	@echo "Compiling HEART transformer for Cerebras WSE..."
	$(CSLC) $(LAYOUT_FILE) \
		--fabric-dims=$(FABRIC_DIMS) \
		--fabric-offsets=$(FABRIC_OFFSETS) \
		--params=$(PARAMS_FILE) \
		-o $(BUILD_DIR)/$(PROGRAM_NAME) \
		--memcpy --channels=1 --width-west-buf=0 --width-east-buf=0
	@echo "Compilation complete! Output: $(BUILD_DIR)/$(PROGRAM_NAME)"

compile-debug: $(BUILD_DIR) ## Compile with debug symbols
	@echo "Compiling HEART transformer (debug mode)..."
	$(CSLC) $(LAYOUT_FILE) \
		--fabric-dims=$(FABRIC_DIMS) \
		--fabric-offsets=$(FABRIC_OFFSETS) \
		--params=$(PARAMS_FILE) \
		-o $(BUILD_DIR)/$(PROGRAM_NAME) \
		--memcpy --channels=1 \
		--compile-only --debug
	@echo "Debug compilation complete!"

weights: $(WEIGHTS_DIR) ## Export Gorgonia weights to Cerebras format
	@echo "Exporting weights from Gorgonia model..."
	cd .. && go run . -export-weights cerebras/$(WEIGHTS_DIR)/model_weights.npz
	@echo "Weights exported to $(WEIGHTS_DIR)/model_weights.npz"

init-weights: $(WEIGHTS_DIR) ## Initialize random weights for testing
	@echo "Initializing random weights..."
	$(CS_PYTHON) -c "import numpy as np; \
		np.savez('$(WEIGHTS_DIR)/model_weights.npz', \
		embedding_weights=np.random.randn(16, 256).astype(np.float32), \
		positional_encoding=np.random.randn(64, 256).astype(np.float32), \
		layer_weights=np.random.randn(10000).astype(np.float32), \
		output_weights=np.random.randn(256, 8).astype(np.float32))"
	@echo "Random weights initialized"

test-data: $(DATA_DIR) ## Generate synthetic test data
	@echo "Generating synthetic test data..."
	$(CS_PYTHON) -c "import numpy as np; \
		np.savez('$(DATA_DIR)/test_input.npz', \
		measurements=np.random.randn(16384, 16).astype(np.float32), \
		node_ids=np.tile(np.arange(256, dtype=np.uint32), 64), \
		time_slices=np.repeat(np.arange(64, dtype=np.uint32), 256))"
	@echo "Test data generated at $(DATA_DIR)/test_input.npz"

run-simulator: compile init-weights test-data ## Run on Cerebras fabric simulator
	@echo "Running HEART on Cerebras fabric simulator..."
	$(CS_PYTHON) run_heart.py \
		--program $(BUILD_DIR)/$(PROGRAM_NAME) \
		--weights $(WEIGHTS_DIR)/model_weights.npz \
		--input $(DATA_DIR)/test_input.npz \
		--output $(DATA_DIR)/test_output.npz
	@echo "Simulation complete! Output: $(DATA_DIR)/test_output.npz"

run-device: compile weights ## Run on actual Cerebras WSE device
	@echo "Running HEART on Cerebras WSE device..."
	@if [ -z "$$CEREBRAS_CMADDR" ]; then \
		echo "Error: CEREBRAS_CMADDR environment variable not set"; \
		exit 1; \
	fi
	$(CS_PYTHON) run_heart.py \
		--program $(BUILD_DIR)/$(PROGRAM_NAME) \
		--weights $(WEIGHTS_DIR)/model_weights.npz \
		--input $(DATA_DIR)/input.npz \
		--output $(DATA_DIR)/output.npz \
		--cmaddr $$CEREBRAS_CMADDR
	@echo "Execution complete!"

test: compile init-weights ## Run quick test on simulator
	@echo "Running quick test..."
	$(CS_PYTHON) run_heart.py \
		--program $(BUILD_DIR)/$(PROGRAM_NAME) \
		--weights $(WEIGHTS_DIR)/model_weights.npz \
		--test \
		--output $(DATA_DIR)/test_output.npz
	@echo "Test passed!"

verify-output: ## Verify output data integrity
	@echo "Verifying output data..."
	$(CS_PYTHON) -c "import numpy as np; \
		data = np.load('$(DATA_DIR)/test_output.npz'); \
		print('Output shapes:'); \
		print('  Commands:', data['heuristic_commands'].shape); \
		print('  Node IDs:', data['node_ids'].shape); \
		print('  Time Slices:', data['time_slices'].shape); \
		print('Sample output:', data['heuristic_commands'][0])"

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -rf $(DATA_DIR)/*.npz
	@echo "Build artifacts cleaned"

clean-all: clean ## Clean everything including weights
	rm -rf $(WEIGHTS_DIR)
	@echo "All artifacts cleaned"

info: ## Show configuration info
	@echo "HEART Transformer Configuration:"
	@echo "  Grid: $(GRID_WIDTH) x $(GRID_HEIGHT)"
	@echo "  Sequence Length: $$(jq -r .SEQUENCE_LENGTH $(PARAMS_FILE))"
	@echo "  Embedding Dim: $$(jq -r .EMBEDDING_DIM $(PARAMS_FILE))"
	@echo "  Num Layers: $$(jq -r .NUM_LAYERS $(PARAMS_FILE))"
	@echo "  Build Dir: $(BUILD_DIR)"
	@echo "  Weights Dir: $(WEIGHTS_DIR)"
