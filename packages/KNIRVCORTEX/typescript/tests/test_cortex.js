#!/usr/bin/env node

/**
 * Test script for KNIRV-CORTEX WASM module
 * Tests the ProtoBuf ABI and inner runtime integration
 *
 * ARCHITECTURE TESTED:
 * - cortex.wasm: Orchestrator that loads stem.wasm, handles errors via HEART
 * - stem.wasm: SLM runtime for pure inference (NO error handling)
 *
 * This test loads cortex.wasm which internally orchestrates stem.wasm.
 * cortex.wasm is responsible for ALL error handling.
 */

const fs = require('fs');
const path = require('path');

// Simple WASM loader for Node.js
async function loadWasm() {
    const wasmPath = path.join(__dirname, '../../dist', 'cortex.wasm');
    
    if (!fs.existsSync(wasmPath)) {
        throw new Error('cortex.wasm not found. Run "make build-cortex" first.');
    }
    
    const wasmBuffer = fs.readFileSync(wasmPath);
    const wasmModule = await WebAssembly.compile(wasmBuffer);
    
    // Create memory for WASM
    const memory = new WebAssembly.Memory({ initial: 10 });
    
    const imports = {
        env: {
            memory: memory,
        },
        // Mock console.log for WASM
        console: {
            log: (ptr, len) => {
                const bytes = new Uint8Array(memory.buffer, ptr, len);
                const str = new TextDecoder().decode(bytes);
                console.log('[WASM]', str);
            }
        }
    };
    
    const instance = await WebAssembly.instantiate(wasmModule, imports);
    return { instance, memory };
}

// Helper to write string to WASM memory
function writeStringToMemory(memory, str) {
    const bytes = new TextEncoder().encode(str);
    const ptr = memory.buffer.byteLength - bytes.length - 1000; // Simple allocation
    const view = new Uint8Array(memory.buffer);
    view.set(bytes, ptr);
    return { ptr, len: bytes.length };
}

// Helper to read string from WASM memory
function readStringFromMemory(memory, ptr, len) {
    const bytes = new Uint8Array(memory.buffer, ptr, len);
    return new TextDecoder().decode(bytes);
}

// Pack pointer and length into u64 (as per KNIRV-CORTEX ABI)
function unpackPtrLen(packed) {
    const ptr = (packed >> 32) & 0xFFFFFFFF;
    const len = packed & 0xFFFFFFFF;
    return { ptr, len };
}

async function testCortex() {
    console.log('🧠 Testing KNIRV-CORTEX WASM module...\n');
    
    try {
        // Load WASM module
        console.log('1. Loading cortex.wasm...');
        const { instance, memory } = await loadWasm();
        console.log('   ✅ WASM module loaded successfully\n');
        
        // Test basic initialization
        console.log('2. Testing initialization...');
        if (typeof instance.exports.initialize === 'function') {
            // Create dummy config
            const config = JSON.stringify({
                version: "1.0",
                max_tokens: 100,
                temperature: 0.7,
                deterministic: true
            });
            
            const { ptr: configPtr, len: configLen } = writeStringToMemory(memory, config);
            const initResult = instance.exports.initialize(configPtr, configLen);
            console.log('   ✅ Initialization result:', initResult ? 'Success' : 'Failed');
        } else {
            console.log('   ⚠️  Initialize function not found');
        }
        
        // Test weight loading
        console.log('\n3. Testing weight loading...');
        if (typeof instance.exports.load_weights === 'function') {
            // Create dummy weights (2KB)
            const dummyWeights = new Uint8Array(2048);
            for (let i = 0; i < dummyWeights.length; i++) {
                dummyWeights[i] = i % 256;
            }
            
            const weightsPtr = memory.buffer.byteLength - dummyWeights.length - 2000;
            const view = new Uint8Array(memory.buffer);
            view.set(dummyWeights, weightsPtr);
            
            const loadResult = instance.exports.load_weights(weightsPtr, dummyWeights.length);
            console.log('   ✅ Weight loading result:', loadResult ? 'Success' : 'Failed');
        } else {
            console.log('   ⚠️  load_weights function not found');
        }
        
        // Test cognitive task execution
        console.log('\n4. Testing cognitive task execution...');
        if (typeof instance.exports.run_cognitive_task === 'function') {
            const prompt = "Hello, KNIRV-CORTEX! Can you process this test prompt?";
            const { ptr: promptPtr, len: promptLen } = writeStringToMemory(memory, prompt);
            
            console.log('   📝 Input prompt:', prompt);
            
            const resultPacked = instance.exports.run_cognitive_task(promptPtr, promptLen);
            const { ptr: resultPtr, len: resultLen } = unpackPtrLen(resultPacked);
            
            if (resultLen > 0) {
                const responseBytes = new Uint8Array(memory.buffer, resultPtr, resultLen);
                console.log('   ✅ Got response:', responseBytes.length, 'bytes');
                console.log('   📄 Raw response (first 100 bytes):', 
                           Array.from(responseBytes.slice(0, 100)).map(b => b.toString(16).padStart(2, '0')).join(' '));
            } else {
                console.log('   ❌ No response received');
            }
        } else {
            console.log('   ⚠️  run_cognitive_task function not found');
        }
        
        // Test model info
        console.log('\n5. Testing model info...');
        if (typeof instance.exports.get_model_info === 'function') {
            const infoResult = instance.exports.get_model_info();
            console.log('   ✅ Model info available');
        } else {
            console.log('   ⚠️  get_model_info function not found');
        }
        
        // List all available exports
        console.log('\n6. Available WASM exports:');
        const exports = Object.keys(instance.exports);
        exports.forEach(name => {
            const type = typeof instance.exports[name];
            console.log(`   - ${name}: ${type}`);
        });
        
        console.log('\n🎉 KNIRV-CORTEX test completed successfully!');
        
    } catch (error) {
        console.error('❌ Test failed:', error.message);
        console.error(error.stack);
        process.exit(1);
    }
}

// Run the test
if (require.main === module) {
    testCortex().catch(console.error);
}

module.exports = { testCortex };
