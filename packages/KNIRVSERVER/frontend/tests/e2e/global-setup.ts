import { chromium, FullConfig } from '@playwright/test';

async function globalSetup(config: FullConfig) {
  console.log('🚀 Global setup: Preparing KNIRV-SERVER E2E tests...');
  
  const { baseURL } = config.projects[0].use;
  const browser = await chromium.launch();
  const page = await browser.newPage();
  
  try {
    // Wait for the backend API to be ready
    console.log('⏳ Waiting for KNIRV-SERVER backend API to be ready...');
    
    let apiReady = false;
    let attempts = 0;
    const maxAttempts = 30; // 30 seconds
    
    while (!apiReady && attempts < maxAttempts) {
      try {
        const response = await page.request.get('http://localhost:8090/api/dve-nodes');
        if (response.status() === 200) {
          apiReady = true;
          console.log('✅ Backend API is ready!');
        }
      } catch (error) {
        // API not ready yet
        await page.waitForTimeout(1000);
        attempts++;
        console.log(`⏳ Attempt ${attempts}/${maxAttempts}: Waiting for backend API...`);
      }
    }
    
    if (!apiReady) {
      throw new Error('❌ Backend API did not become ready in time');
    }
    
    // Wait for frontend to be ready
    console.log('⏳ Waiting for KNIRV-SERVER frontend to be ready...');
    
    let frontendReady = false;
    attempts = 0;
    
    while (!frontendReady && attempts < maxAttempts) {
      try {
        await page.goto(baseURL || 'http://localhost:8090', { 
          waitUntil: 'domcontentloaded',
          timeout: 5000 
        });
        
        // Check if main application elements are present
        const titleElement = await page.locator('text=KNIRV SERVER').first();
        if (await titleElement.isVisible()) {
          frontendReady = true;
          console.log('✅ Frontend is ready!');
        }
      } catch (error) {
        await page.waitForTimeout(1000);
        attempts++;
        console.log(`⏳ Attempt ${attempts}/${maxAttempts}: Waiting for frontend...`);
      }
    }
    
    if (!frontendReady) {
      console.warn('⚠️  Frontend may not be fully ready, but continuing with tests...');
    }
    
    // Clean up any existing test data
    console.log('🧹 Cleaning up any existing test data...');
    
    try {
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
      
      for (const nodeId of testNodeIds) {
        try {
          await page.request.delete(`http://localhost:8090/api/dve-nodes/${nodeId}`);
        } catch (error) {
          // Node doesn't exist, that's fine
        }
      }
      
      console.log('✅ Test data cleanup completed');
    } catch (error) {
      console.warn('⚠️  Test data cleanup failed, but continuing...', error);
    }
    
    console.log('✅ Global setup completed successfully!');
    
  } catch (error) {
    console.error('❌ Global setup failed:', error);
    throw error;
  } finally {
    await browser.close();
  }
}

export default globalSetup;