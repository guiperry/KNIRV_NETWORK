import { chromium, FullConfig } from '@playwright/test';

async function globalTeardown(config: FullConfig) {
  console.log('🧹 Global teardown: Cleaning up after KNIRV-NEXUS E2E tests...');
  
  const browser = await chromium.launch();
  const page = await browser.newPage();
  
  try {
    // Clean up all test data that might have been created
    console.log('🧹 Final cleanup of test data...');
    
    const testNodeIds = [
      'test-node-e2e',
      'validation-test-node',
      'bulk-test-node-1', 
      'bulk-test-node-2',
      'bulk-test-node-3',
      'compute-only-node',
      'storage-only-node', 
      'high-stake-node',
      'duplicate-test-node',
      'large-payload-node'
    ];
    
    let cleanedCount = 0;
    
    for (const nodeId of testNodeIds) {
      try {
        const response = await page.request.delete(`http://localhost:8080/api/dve-nodes/${nodeId}`);
        if (response.status() === 200) {
          cleanedCount++;
        }
      } catch (error) {
        // Node doesn't exist or API is down, that's fine
      }
    }
    
    console.log(`✅ Cleaned up ${cleanedCount} test DVE nodes`);
    
    // Generate test summary
    console.log('📊 Test execution summary:');
    console.log(`   • Test environment: ${process.env.NODE_ENV || 'development'}`);
    console.log(`   • Backend URL: http://localhost:8080`);
    console.log(`   • Frontend URL: http://localhost:8090`);
    console.log(`   • Test data cleaned: ${cleanedCount} nodes`);
    
    console.log('✅ Global teardown completed successfully!');
    
  } catch (error) {
    console.error('❌ Global teardown failed:', error);
    // Don't throw the error as teardown failures shouldn't fail the entire test suite
  } finally {
    await browser.close();
  }
}

export default globalTeardown;