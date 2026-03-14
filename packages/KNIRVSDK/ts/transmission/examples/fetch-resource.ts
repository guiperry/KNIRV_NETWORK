/**
 * Example script for fetching a resource using the KNIRV Client SDK.
 */

import { KnirvClient, parseKnirvURI, KnirvURIError, KnirvClientError } from '../src';

async function main() {
  // Parse command line arguments
  if (process.argv.length < 3) {
    console.log('Usage: ts-node fetch-resource.ts <knirv-uri>');
    console.log('Example: ts-node fetch-resource.ts knirv://mychain-alpha.chain/block?number=123');
    process.exit(1);
  }
  
  const uriString = process.argv[2];
  
  // Parse the URI to validate it
  try {
    const parsedUri = parseKnirvURI(uriString);
    
    console.log('Parsed URI:');
    console.log(`  ID:           ${parsedUri.id}`);
    console.log(`  ResourceType: ${parsedUri.resourceType}`);
    console.log(`  Path:         ${parsedUri.path}`);
    console.log(`  Raw URI:      ${parsedUri.raw}`);
    console.log('  Query Parameters:');
    for (const [key, value] of parsedUri.query.entries()) {
      console.log(`    ${key}: ${value}`);
    }
  } catch (e) {
    if (e instanceof KnirvURIError) {
      console.error(`Failed to parse URI: ${e.message}`);
      process.exit(1);
    }
    throw e;
  }
  
  // Create a client with default configuration
  const client = new KnirvClient({
    bootstrapPeers: [
      '/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ',
      '/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM',
      '/ip4/128.199.219.111/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu',
      '/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64',
    ],
    logEnabled: true
  });
  
  try {
    // Bootstrap the client
    console.log('Bootstrapping client...');
    await client.bootstrap();
    
    // Fetch the resource
    console.log(`Fetching resource: ${uriString}`);
    const resource = await client.fetchResource(uriString);
    
    // Print the resource data
    console.log('Resource data:');
    console.log(resource.toString());
    
    process.exit(0);
  } catch (e) {
    if (e instanceof KnirvClientError) {
      console.error(`Failed to fetch resource: ${e.message}`);
      process.exit(1);
    }
    throw e;
  } finally {
    // Close the client
    await client.stop();
  }
}

// Handle unhandled promise rejections
process.on('unhandledRejection', (reason, promise) => {
  console.error('Unhandled Rejection at:', promise, 'reason:', reason);
  process.exit(1);
});

// Run the main function
main().catch(e => {
  console.error('Unexpected error:', e);
  process.exit(1);
});