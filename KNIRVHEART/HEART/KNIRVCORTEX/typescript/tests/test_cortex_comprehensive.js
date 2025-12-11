#!/usr/bin/env node

/**
 * Comprehensive KNIRV-CORTEX Testing Suite
 * Tests the actual capabilities of cortex.wasm
 *
 * ARCHITECTURE:
 * - cortex.wasm: Orchestrator (loads stem.wasm, applies LoRAs, handles ALL errors)
 * - stem.wasm: Compiled SLM for pure inference (NO error handling)
 *
 * TESTING APPROACH:
 * - Test cortex.wasm orchestration capabilities
 * - Verify cortex.wasm handles stem.wasm errors properly
 * - Test HEART integration in cortex.wasm
 * - Ensure stem.wasm is used only for inference
 */

const fs = require('fs');
const path = require('path');

const TEST_RESULTS = {
    passed: 0,
    failed: 0,
    tests: []
};

function logTest(name, passed, details = '') {
    const status = passed ? '✅ PASS' : '❌ FAIL';
    console.log(`${status}: ${name}`);
    if (details) console.log(`   ${details}`);
    
    TEST_RESULTS.tests.push({ name, passed, details });
    if (passed) TEST_RESULTS.passed++;
    else TEST_RESULTS.failed++;
}

async function testWasmLoading() {
    console.log('\n🔧 Testing WASM Loading and Initialization...');

    try {
        // Test if WASM files exist
        const wasmPath = '../pkg/knirv_cortex_wasm_bg.wasm';
        const jsPath = '../pkg/knirv_cortex_wasm.js';

        if (!fs.existsSync(wasmPath)) {
            logTest('WASM file existence', false, `WASM file not found: ${wasmPath}`);
            return false;
        }

        if (!fs.existsSync(jsPath)) {
            logTest('JS bindings existence', false, `JS bindings not found: ${jsPath}`);
            return false;
        }

        logTest('WASM files existence', true, 'WASM and JS binding files found');

        // For now, we'll test the raw WASM file properties
        const wasmBuffer = fs.readFileSync(wasmPath);
        logTest('WASM file loading', true, `Loaded ${wasmBuffer.length} bytes`);

        return true;
    } catch (error) {
        logTest('WASM module initialization', false, `Error: ${error.message}`);
        return false;
    }
}

async function testCortexInstantiation() {
    console.log('\n🧠 Testing Cortex Instantiation...');

    try {
        // Since we can't easily import the WASM module in CommonJS,
        // we'll test the WASM structure instead
        logTest('Cortex instantiation', false, 'Cannot test instantiation without proper ES module setup');
        return null;
    } catch (error) {
        logTest('Cortex instantiation', false, `Error: ${error.message}`);
        return null;
    }
}

async function testBasicInference(cortex) {
    console.log('\n💭 Testing Basic Inference Capabilities...');
    
    if (!cortex) {
        logTest('Basic inference test', false, 'No cortex instance available');
        return;
    }

    try {
        // Test simple prompt
        const simplePrompt = "Hello, can you help me?";
        console.log(`   Input: "${simplePrompt}"`);
        
        // Note: We need to check what methods are actually available
        // The current implementation might not have direct inference methods exposed
        logTest('Basic inference test', false, 'Need to implement proper inference method exposure');
        
    } catch (error) {
        logTest('Basic inference test', false, `Error: ${error.message}`);
    }
}

async function testMemoryManagement(cortex) {
    console.log('\n🧮 Testing Memory Management...');
    
    if (!cortex) {
        logTest('Memory management test', false, 'No cortex instance available');
        return;
    }

    try {
        // Test memory allocation patterns
        const largePrompt = "A".repeat(10000); // 10KB prompt
        console.log(`   Testing with ${largePrompt.length} character prompt`);
        
        logTest('Memory management test', false, 'Need to implement memory testing methods');
        
    } catch (error) {
        logTest('Memory management test', false, `Error: ${error.message}`);
    }
}

async function testProtobufSerialization() {
    console.log('\n📦 Testing ProtoBuf Serialization...');
    
    try {
        // Test if we can create and serialize protobuf messages
        // This would require importing the protobuf definitions
        logTest('ProtoBuf serialization test', false, 'Need to implement protobuf testing');
        
    } catch (error) {
        logTest('ProtoBuf serialization test', false, `Error: ${error.message}`);
    }
}

async function testChatCapabilities(cortex) {
    console.log('\n💬 Testing Chat Capabilities...');
    
    if (!cortex) {
        logTest('Chat capabilities test', false, 'No cortex instance available');
        return;
    }

    const testConversation = [
        "Hello, what's your name?",
        "Can you help me with a math problem?",
        "What is 2 + 2?",
        "Thank you for your help!"
    ];

    try {
        for (let i = 0; i < testConversation.length; i++) {
            const prompt = testConversation[i];
            console.log(`   Turn ${i + 1}: "${prompt}"`);
            
            // Test conversation flow
            // Current implementation likely just simulates responses
        }
        
        logTest('Chat capabilities test', false, 'Chat is simulated, not real inference');
        
    } catch (error) {
        logTest('Chat capabilities test', false, `Error: ${error.message}`);
    }
}

async function testModelWeights() {
    console.log('\n⚖️ Testing Model Weights and Bias...');
    
    try {
        // Check if the WASM file contains actual neural network weights
        const wasmPath = './pkg/knirv_cortex_wasm_bg.wasm';
        const wasmBuffer = fs.readFileSync(wasmPath);
        
        console.log(`   WASM file size: ${wasmBuffer.length} bytes (${(wasmBuffer.length / 1024).toFixed(2)} KB)`);
        
        // Analyze the WASM content for patterns that might indicate weights
        const hasFloatPatterns = analyzeForFloatPatterns(wasmBuffer);
        const hasLargeDataSections = analyzeForLargeDataSections(wasmBuffer);
        
        logTest('Model weights analysis', true, 
            `Size: ${(wasmBuffer.length / 1024).toFixed(2)}KB, Float patterns: ${hasFloatPatterns}, Large data: ${hasLargeDataSections}`);
        
        // Conclusion: This size suggests no real neural network weights
        if (wasmBuffer.length < 1024 * 1024) { // Less than 1MB
            logTest('Real neural network weights', false, 
                'File too small to contain meaningful neural network weights');
        }
        
    } catch (error) {
        logTest('Model weights analysis', false, `Error: ${error.message}`);
    }
}

function analyzeForFloatPatterns(buffer) {
    // Look for patterns that might indicate floating point weight arrays
    let floatPatternCount = 0;

    // Convert Buffer to ArrayBuffer for DataView
    const arrayBuffer = buffer.buffer.slice(buffer.byteOffset, buffer.byteOffset + buffer.byteLength);
    const view = new DataView(arrayBuffer);

    // Sample every 1000 bytes to look for float-like patterns
    for (let i = 0; i < buffer.length - 4; i += 1000) {
        try {
            const float = view.getFloat32(i, true); // little endian
            if (!isNaN(float) && isFinite(float) && Math.abs(float) < 10) {
                floatPatternCount++;
            }
        } catch (e) {
            // Ignore out of bounds
        }
    }

    return floatPatternCount > 10;
}

function analyzeForLargeDataSections(buffer) {
    // Look for large consecutive data sections that might be weight matrices
    let maxConsecutiveNonZero = 0;
    let currentConsecutive = 0;
    
    for (let i = 0; i < Math.min(buffer.length, 50000); i++) {
        if (buffer[i] !== 0) {
            currentConsecutive++;
        } else {
            maxConsecutiveNonZero = Math.max(maxConsecutiveNonZero, currentConsecutive);
            currentConsecutive = 0;
        }
    }
    
    return maxConsecutiveNonZero > 1000;
}

async function testPerformanceBenchmarks(cortex) {
    console.log('\n⚡ Testing Performance Benchmarks...');
    
    if (!cortex) {
        logTest('Performance benchmarks', false, 'No cortex instance available');
        return;
    }

    try {
        const iterations = 100;
        const testPrompt = "Benchmark test prompt for performance measurement";
        
        console.log(`   Running ${iterations} iterations...`);
        
        const startTime = performance.now();
        
        for (let i = 0; i < iterations; i++) {
            // Would test actual inference here
            // Currently just measuring instantiation overhead
        }
        
        const endTime = performance.now();
        const avgTime = (endTime - startTime) / iterations;
        
        logTest('Performance benchmarks', true, 
            `Average time per operation: ${avgTime.toFixed(2)}ms`);
        
    } catch (error) {
        logTest('Performance benchmarks', false, `Error: ${error.message}`);
    }
}

function printTestSummary() {
    console.log('\n📊 Test Summary');
    console.log('================');
    console.log(`Total Tests: ${TEST_RESULTS.tests.length}`);
    console.log(`Passed: ${TEST_RESULTS.passed}`);
    console.log(`Failed: ${TEST_RESULTS.failed}`);
    console.log(`Success Rate: ${((TEST_RESULTS.passed / TEST_RESULTS.tests.length) * 100).toFixed(1)}%`);
    
    if (TEST_RESULTS.failed > 0) {
        console.log('\n❌ Failed Tests:');
        TEST_RESULTS.tests
            .filter(t => !t.passed)
            .forEach(t => console.log(`   - ${t.name}: ${t.details}`));
    }
}

async function main() {
    console.log('🧠 KNIRV-CORTEX Comprehensive Testing Suite');
    console.log('===========================================');
    
    const wasmLoaded = await testWasmLoading();
    if (!wasmLoaded) {
        console.log('❌ Cannot proceed without WASM loading');
        return;
    }
    
    const cortex = await testCortexInstantiation();
    
    await testBasicInference(cortex);
    await testMemoryManagement(cortex);
    await testProtobufSerialization();
    await testChatCapabilities(cortex);
    await testModelWeights();
    await testPerformanceBenchmarks(cortex);
    
    printTestSummary();
    
    console.log('\n🔍 Analysis Complete - See cortexAnalysis.md for detailed findings');
}

// Handle CommonJS execution
if (require.main === module) {
    main().catch(console.error);
}

module.exports = { runTests: main };
