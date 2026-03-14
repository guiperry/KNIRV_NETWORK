/**
 * TypeScript Usage Examples for KNIRV-CORTEX
 *
 * ARCHITECTURE OVERVIEW:
 * =====================
 *
 * stem.wasm (SLM Runtime):
 * - RESERVED for compiled Small Language Model
 * - Pure inference execution ONLY
 * - Model weight loading
 * - NO error handling
 * - NO recovery logic
 * - NO HEART integration
 *
 * cortex.wasm (Cognitive Shell Orchestrator):
 * - Loads and orchestrates stem.wasm
 * - Loads and applies LoRA adapters
 * - Handles ALL errors from stem.wasm
 * - HEART integration for error analysis
 * - Memory policy management
 * - External inference fallbacks
 *
 * DATA FLOW:
 * ==========
 *
 * User Request
 *     ↓
 * cortex.wasm (Orchestrator)
 *     ├─→ Load stem.wasm (if not loaded)
 *     ├─→ Apply LoRA adapters
 *     ├─→ Call stem.wasm.run_inference()
 *     │       ↓
 *     │   stem.wasm returns result or error
 *     │       ↓
 *     ├─→ If error: Query HEART for analysis
 *     ├─→ Apply HEART recommendations
 *     └─→ Return final result to user
 */

import { createCortexOrchestrator, CortexOrchestrator, InferenceInput } from '../wrapper/CortexWrapper';

// ============================================================================
// Example 1: Basic Initialization
// ============================================================================

async function example1_BasicInitialization() {
  console.log('\n📘 Example 1: Basic Initialization\n');

  // Create cortex orchestrator
  // This loads cortex.wasm which will internally manage stem.wasm
  const cortex = await createCortexOrchestrator(
    './dist/cortex.wasm',
    {
      version: '1.0',
      temperature: 0.7,
      max_tokens: 2048,
      deterministic: true
    }
  );

  console.log('✅ cortex.wasm orchestrator initialized');
  console.log('   - Ready to load stem.wasm');
  console.log('   - Ready to handle errors via HEART');
  console.log('   - Ready to apply LoRA adapters');

  return cortex;
}

// ============================================================================
// Example 2: Loading Model Weights into stem.wasm
// ============================================================================

async function example2_LoadingWeights(cortex: CortexOrchestrator) {
  console.log('\n📘 Example 2: Loading Model Weights\n');

  // Create dummy model weights (in production, load from file)
  const modelWeights = new Uint8Array(2048);
  for (let i = 0; i < modelWeights.length; i++) {
    modelWeights[i] = Math.floor(Math.random() * 256);
  }

  console.log('📦 Loading weights into stem.wasm (SLM runtime)...');
  console.log('   - cortex.wasm orchestrates this operation');
  console.log('   - stem.wasm receives and stores weights');
  console.log('   - stem.wasm does NOT handle loading errors');
  console.log('   - cortex.wasm catches and handles any errors');

  await cortex.loadWeights(modelWeights);

  console.log('✅ Weights loaded successfully');
  console.log(cortex.getInfo());
}

// ============================================================================
// Example 3: Initializing HEART for Error Handling
// ============================================================================

async function example3_InitializeHEART(cortex: CortexOrchestrator) {
  console.log('\n📘 Example 3: HEART Error Analysis Integration\n');

  console.log('🧠 Initializing HEART client in cortex.wasm...');
  console.log('   CRITICAL: stem.wasm has NO error handling');
  console.log('   CRITICAL: cortex.wasm uses HEART for ALL error handling');

  await cortex.initializeHEART({
    endpoint: 'http://localhost:8080/heart/analyze',
    timeout_ms: 5000,
    min_confidence_threshold: 0.7,
    enable_pattern_recognition: true,
    enable_similarity_search: true,
    fallback_to_local: true
  });

  console.log('✅ HEART initialized');
  console.log('   - cortex.wasm will query HEART when stem.wasm errors occur');
  console.log('   - HEART provides heuristic analysis and recovery strategies');
  console.log('   - cortex.wasm applies HEART recommendations');
}

// ============================================================================
// Example 4: Running Inference (Normal Flow)
// ============================================================================

async function example4_RunInference(cortex: CortexOrchestrator) {
  console.log('\n📘 Example 4: Running Inference (Success Case)\n');

  const input: InferenceInput = {
    prompt: 'Analyze the following network traffic pattern',
    context: 'Security monitoring system',
  };

  console.log('🔄 Inference Flow:');
  console.log('   1. User sends request to cortex.wasm');
  console.log('   2. cortex.wasm prepares input for stem.wasm');
  console.log('   3. cortex.wasm calls stem.wasm.run_inference()');
  console.log('   4. stem.wasm runs pure inference (NO error handling)');
  console.log('   5. stem.wasm returns result to cortex.wasm');
  console.log('   6. cortex.wasm returns result to user');

  const result = await cortex.runInference(input);

  console.log('\n✅ Inference completed successfully');
  console.log('Response:', result.response);
  console.log('Confidence:', result.confidence);
  console.log('Processing time:', result.processing_time_ms, 'ms');
}

// ============================================================================
// Example 5: Error Handling Flow
// ============================================================================

async function example5_ErrorHandling(cortex: CortexOrchestrator) {
  console.log('\n📘 Example 5: Error Handling Flow\n');

  console.log('🔄 Error Handling Flow (when stem.wasm fails):');
  console.log('   1. User sends request to cortex.wasm');
  console.log('   2. cortex.wasm calls stem.wasm.run_inference()');
  console.log('   3. stem.wasm encounters error (e.g., invalid input)');
  console.log('   4. stem.wasm returns error (NO error handling)');
  console.log('   5. cortex.wasm detects error from stem.wasm');
  console.log('   6. cortex.wasm queries HEART with error details');
  console.log('   7. HEART analyzes error and returns recommendations');
  console.log('   8. cortex.wasm applies HEART recommendations');
  console.log('   9. cortex.wasm may retry or return enhanced error');

  // Simulate an error scenario
  try {
    const input: InferenceInput = {
      prompt: '', // Empty prompt might cause error in stem.wasm
      context: 'Error test',
    };

    await cortex.runInference(input);
  } catch (error) {
    console.log('\n⚠️  Error caught by cortex.wasm');
    console.log('   - stem.wasm returned error (as it should)');
    console.log('   - cortex.wasm handled it via HEART');
    console.log('   - User receives enhanced error information');
  }
}

// ============================================================================
// Example 6: LoRA Adapter Management
// ============================================================================

async function example6_LoRAAdapters(cortex: CortexOrchestrator) {
  console.log('\n📘 Example 6: LoRA Adapter Management\n');

  console.log('🔧 LoRA Adapter Flow:');
  console.log('   1. cortex.wasm compiles LoRA adapter from skill data');
  console.log('   2. cortex.wasm applies LoRA to stem.wasm');
  console.log('   3. stem.wasm now has enhanced capabilities');
  console.log('   4. cortex.wasm manages multiple LoRA adapters');

  // Compile a LoRA adapter
  const skillData = {
    skill_id: 'skill_001',
    skill_name: 'Network Analysis',
    description: 'Enhanced network traffic analysis'
  };

  console.log('\n📦 Compiling LoRA adapter...');
  await cortex.compileLoRAAdapter(skillData);

  // Get available adapters
  const adapters = cortex.getLoRAAdapters();
  console.log('\n✅ Available LoRA Adapters:', adapters.length);
  console.log('   - cortex.wasm manages all LoRA adapters');
  console.log('   - stem.wasm uses LoRAs during inference');
  console.log('   - LoRA errors handled by cortex.wasm, not stem.wasm');
}

// ============================================================================
// Example 7: Full Production Workflow
// ============================================================================

async function example7_ProductionWorkflow() {
  console.log('\n📘 Example 7: Full Production Workflow\n');
  console.log('=' .repeat(70));

  try {
    // 1. Initialize cortex orchestrator
    console.log('\n[1/6] Initializing cortex.wasm orchestrator...');
    const cortex = await example1_BasicInitialization();

    // 2. Load model weights into stem.wasm
    console.log('\n[2/6] Loading model weights into stem.wasm...');
    await example2_LoadingWeights(cortex);

    // 3. Initialize HEART for error handling
    console.log('\n[3/6] Initializing HEART error analysis...');
    await example3_InitializeHEART(cortex);

    // 4. Compile LoRA adapters
    console.log('\n[4/6] Compiling LoRA adapters...');
    await example6_LoRAAdapters(cortex);

    // 5. Run successful inference
    console.log('\n[5/6] Running inference (success case)...');
    await example4_RunInference(cortex);

    // 6. Test error handling
    console.log('\n[6/6] Testing error handling...');
    await example5_ErrorHandling(cortex);

    console.log('\n' + '='.repeat(70));
    console.log('✅ All examples completed successfully!');
    console.log('\nKEY ARCHITECTURE POINTS:');
    console.log('  ✓ stem.wasm: Pure SLM inference, NO error handling');
    console.log('  ✓ cortex.wasm: Orchestrates stem.wasm, handles ALL errors');
    console.log('  ✓ HEART: Error analysis system used by cortex.wasm');
    console.log('  ✓ LoRA: Managed by cortex.wasm, applied to stem.wasm');

  } catch (error) {
    console.error('\n❌ Error in production workflow:', error);
  }
}

// ============================================================================
// Main Entry Point
// ============================================================================

if (require.main === module) {
  console.log('🚀 KNIRV-CORTEX TypeScript Usage Examples');
  console.log('==========================================\n');
  console.log('ARCHITECTURE:');
  console.log('  stem.wasm: SLM runtime (pure inference, NO error handling)');
  console.log('  cortex.wasm: Orchestrator (loads stem.wasm, handles ALL errors)');

  example7_ProductionWorkflow()
    .then(() => {
      console.log('\n✅ Examples completed successfully');
      process.exit(0);
    })
    .catch((error) => {
      console.error('\n❌ Fatal error:', error);
      process.exit(1);
    });
}

export {
  example1_BasicInitialization,
  example2_LoadingWeights,
  example3_InitializeHEART,
  example4_RunInference,
  example5_ErrorHandling,
  example6_LoRAAdapters,
  example7_ProductionWorkflow
};
