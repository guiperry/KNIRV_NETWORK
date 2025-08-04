/**
 * PoAu-D SDK Usage Example for TypeScript/JavaScript
 * This example demonstrates how to use the KNIRV TypeScript SDK to interact with the PoAu-D consensus mechanism
 */

import { PoAuDClient, validateNetworkAuthor, getDefaultPoAuDConfig } from '../poaud';
import { KNIRVGatewayClient } from '../client';

async function main() {
  // Create a new PoAu-D client
  const client = new PoAuDClient({
    baseURL: 'http://localhost:8000', // Gateway URL
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer your-api-key-here', // Optional API key
    },
  });

  try {
    // Example 1: Check current PoAu-D status
    console.log('=== PoAu-D Status Check ===');
    const status = await client.getConsensusStatus();
    console.log(`PoAu-D Enabled: ${status.enabled}`);
    console.log(`Network Authors Count: ${status.network_authors_count || 0}`);
    
    if (status.enabled) {
      console.log(`Main Pool Size: ${status.main_pool_size || 0}`);
      console.log(`PAS Pool Size: ${status.pas_pool_size || 0}`);
      console.log(`Delegated Transactions: ${status.delegated_transactions || 0}`);
    }

    // Example 2: Enable PoAu-D consensus
    console.log('\n=== Enabling PoAu-D Consensus ===');
    const enableResp = await client.enableConsensus();
    console.log(`Enable Response: ${enableResp.message}`);

    // Example 3: Add Network Authors
    console.log('\n=== Adding Network Authors ===');
    const networkAuthors = [
      'knirv1abc123def456ghi789',
      'knirv1xyz789uvw456rst123',
      'knirv1mno345pqr678stu901',
    ];

    for (const author of networkAuthors) {
      try {
        validateNetworkAuthor(author); // Validate before adding
        const addResp = await client.addNetworkAuthor(author);
        console.log(`Added Network Author: ${author} - ${addResp.message}`);
      } catch (error) {
        console.error(`Error adding network author ${author}:`, error);
      }
    }

    // Example 4: List all Network Authors
    console.log('\n=== Listing Network Authors ===');
    const authors = await client.listNetworkAuthors();
    console.log(`Total Network Authors: ${authors.count}`);
    authors.network_authors.forEach((author, index) => {
      console.log(`  ${index + 1}. ${author}`);
    });

    // Example 5: Check if specific address is a Network Author
    console.log('\n=== Checking Network Author Status ===');
    const testAddress = 'knirv1abc123def456ghi789';
    const isAuthor = await client.isNetworkAuthor(testAddress);
    console.log(`Address ${testAddress} is Network Author: ${isAuthor}`);

    // Example 6: Get delegation statistics
    console.log('\n=== Delegation Statistics ===');
    const stats = await client.getDelegationStatistics();
    console.log('Delegation Statistics:');
    Object.entries(stats).forEach(([key, value]) => {
      console.log(`  ${key}: ${value}`);
    });

    // Example 7: Monitor PoAu-D status over time
    console.log('\n=== Monitoring PoAu-D Status ===');
    await monitorPoAuDStatus(client, 3); // Monitor for 3 iterations

    // Example 8: Remove a Network Author
    console.log('\n=== Removing Network Author ===');
    const removeAddress = 'knirv1xyz789uvw456rst123';
    const removeResp = await client.removeNetworkAuthor(removeAddress);
    console.log(`Removed Network Author: ${removeAddress} - ${removeResp.message}`);

    // Example 9: Disable PoAu-D consensus
    console.log('\n=== Disabling PoAu-D Consensus ===');
    const disableResp = await client.disableConsensus();
    console.log(`Disable Response: ${disableResp.message}`);

    // Example 10: Final status check
    console.log('\n=== Final Status Check ===');
    const finalStatus = await client.getConsensusStatus();
    console.log(`Final PoAu-D Enabled: ${finalStatus.enabled}`);

    console.log('\n=== PoAu-D SDK Example Complete ===');

  } catch (error) {
    console.error('Error in PoAu-D example:', error);
  }
}

/**
 * Monitor PoAu-D status over time
 */
async function monitorPoAuDStatus(client: PoAuDClient, iterations: number): Promise<void> {
  for (let i = 0; i < iterations; i++) {
    try {
      const status = await client.getConsensusStatus();
      console.log(`Monitor ${i + 1} - Enabled: ${status.enabled}, NAPs: ${status.network_authors_count || 0}, Delegated: ${status.delegated_transactions || 0}`);
    } catch (error) {
      console.error(`Monitor iteration ${i + 1} - Error:`, error);
    }
    
    if (i < iterations - 1) {
      await new Promise(resolve => setTimeout(resolve, 2000)); // Wait 2 seconds
    }
  }
}

/**
 * Demonstrate error handling
 */
async function demonstrateErrorHandling(): Promise<void> {
  const client = new PoAuDClient();

  // Example of handling validation errors
  try {
    validateNetworkAuthor('');
  } catch (error) {
    console.log('Validation error:', error);
  }

  try {
    validateNetworkAuthor('short');
  } catch (error) {
    console.log('Validation error:', error);
  }

  // Example of handling API errors
  try {
    await client.addNetworkAuthor('invalid-address');
  } catch (error) {
    console.log('API error:', error);
  }
}

/**
 * Demonstrate advanced usage
 */
async function demonstrateAdvancedUsage(): Promise<void> {
  // Get default configuration
  const config = getDefaultPoAuDConfig();
  console.log('Default PoAu-D Config:', config);

  // Create client with custom options
  const client = new PoAuDClient({
    baseURL: 'https://api.knirv.network',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer production-api-key',
    },
  });

  try {
    // Check if PoAu-D is enabled before performing operations
    const enabled = await client.isPoAuDEnabled();
    
    if (!enabled) {
      console.log('PoAu-D is not enabled, enabling it first...');
      await client.enableConsensus();
    }

    // Get network author count
    const count = await client.getNetworkAuthorCount();
    console.log(`Current Network Author Count: ${count}`);

    // Perform operations based on current state
    if (count === 0) {
      console.log('No Network Authors found, adding initial authors...');
      // Add initial network authors
    } else {
      console.log(`Found ${count} Network Authors, system is ready`);
    }
  } catch (error) {
    console.error('Error in advanced usage:', error);
  }
}

/**
 * Example of batch operations
 */
async function demonstrateBatchOperations(): Promise<void> {
  const client = new PoAuDClient();

  // Batch add multiple network authors
  const authors = [
    'knirv1batch001',
    'knirv1batch002',
    'knirv1batch003',
  ];

  console.log('Adding multiple Network Authors...');
  
  const addPromises = authors.map(async (author, index) => {
    try {
      const resp = await client.addNetworkAuthor(author);
      console.log(`Added author ${index + 1}: ${resp.message}`);
      return { success: true, author, response: resp };
    } catch (error) {
      console.error(`Failed to add author ${index + 1} (${author}):`, error);
      return { success: false, author, error };
    }
  });

  const results = await Promise.allSettled(addPromises);
  
  // Verify all were added
  try {
    const authorsList = await client.listNetworkAuthors();
    console.log(`Total authors after batch add: ${authorsList.count}`);
  } catch (error) {
    console.error('Error verifying authors:', error);
  }
}

/**
 * Example using the full gateway client
 */
async function demonstrateGatewayClientUsage(): Promise<void> {
  // Create full gateway client
  const gatewayClient = new KNIRVGatewayClient({
    baseURL: 'http://localhost:8000',
    headers: {
      'Authorization': 'Bearer your-api-key',
    },
  });

  try {
    // Access PoAu-D service through the gateway client
    const status = await gatewayClient.poaud.getStatus();
    console.log('PoAu-D Status via Gateway Client:', status);

    // Add network author via gateway client
    const addResp = await gatewayClient.poaud.networkAuthors.add('knirv1gateway001');
    console.log('Added via Gateway Client:', addResp);

    // List authors via gateway client
    const authors = await gatewayClient.poaud.networkAuthors.list();
    console.log('Authors via Gateway Client:', authors);

  } catch (error) {
    console.error('Error using gateway client:', error);
  }
}

// Run the main example
if (require.main === module) {
  main().catch(console.error);
}

// Export functions for use in other modules
export {
  main,
  monitorPoAuDStatus,
  demonstrateErrorHandling,
  demonstrateAdvancedUsage,
  demonstrateBatchOperations,
  demonstrateGatewayClientUsage,
};
