# KNIRV Client SDK for JavaScript/TypeScript

The KNIRV Client SDK for JavaScript/TypeScript provides a high-level interface for interacting with the KNIRVCHAIN network. It simplifies the process of resolving `knirv://` URIs, discovering peers on the private DHT, connecting to them, and fetching the underlying resources.

## Installation

```bash
npm install knirv-uri-sdk
# or
yarn add knirv-uri-sdk
```

## Features

- Parse `knirv://` URIs into their components (ID, ResourceType, Path, Query)
- Discover peers on the KNIRVCHAIN DHT that provide specific resources
- Connect to peers and fetch resources using libp2p streams
- Handle errors and retries gracefully
- Promise-based API for easy integration with async/await
- Full TypeScript support with type definitions

## Usage

### Parsing URIs

```typescript
import { parseKnirvURI, KnirvURIError } from 'knirv-uri-sdk';

try {
  const uriString = "knirv://mychain-alpha.chain/block?number=123&validator=node1";
  const parsedUri = parseKnirvURI(uriString);
  
  console.log(`ID: ${parsedUri.id}`);                 // "mychain-alpha"
  console.log(`ResourceType: ${parsedUri.resourceType}`); // "chain"
  console.log(`Path: ${parsedUri.path}`);             // "/block"
  console.log(`Query parameter 'number': ${parsedUri.getQueryParam('number')}`); // "123"
} catch (e) {
  if (e instanceof KnirvURIError) {
    console.error(`Error parsing URI: ${e.message}`);
  }
}
```

### Fetching Resources

```typescript
import { KnirvClient, ResourceNotFoundError, KnirvClientError } from 'knirv-uri-sdk';

async function main() {
  // Create a client with bootstrap peers
  const client = new KnirvClient({
    bootstrapPeers: [
      '/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ',
      // Add your own bootstrap peers here
    ],
    logEnabled: true
  });
  
  try {
    // Bootstrap the client (connect to DHT network)
    await client.bootstrap();
    
    // Fetch a resource
    const resource = await client.fetchResource('knirv://mychain-alpha.chain/block?number=123');
    
    // Use the resource data
    console.log(resource.toString());
    
    // Or get the raw bytes
    const data = resource.bytes();
    // Process data...
  } catch (e) {
    if (e instanceof ResourceNotFoundError) {
      console.error(`Resource not found: ${e.message}`);
    } else if (e instanceof KnirvClientError) {
      console.error(`Client error: ${e.message}`);
    } else {
      console.error(`Unexpected error: ${e instanceof Error ? e.message : String(e)}`);
    }
  } finally {
    // Close the client
    await client.stop();
  }
}

main().catch(console.error);
```

### Custom Configuration

```typescript
import { KnirvClient, KnirvClientConfig } from 'knirv-uri-sdk';

const config: KnirvClientConfig = {
  bootstrapPeers: [
    '/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ',
    // Add your own bootstrap peers here
  ],
  p2pPort: 6000,  // Specify a port for the libp2p host
  logEnabled: true  // Enable logging
};

const client = new KnirvClient(config);
```

## Error Handling

The SDK provides specific error types to help with error handling:

- `KnirvURIError`: Base exception for URI parsing errors
- `KnirvClientError`: Base exception for client errors
- `ResourceNotFoundError`: Thrown when a resource cannot be found
- `ConnectionFailedError`: Thrown when connection to a peer fails
- `FetchFailedError`: Thrown when fetching a resource fails
- `ClientClosedError`: Thrown when operations are attempted on a closed client

Example:

```typescript
import {
  KnirvClient, KnirvClientError, ResourceNotFoundError,
  ConnectionFailedError, FetchFailedError
} from 'knirv-uri-sdk';

async function fetchWithErrorHandling(uriString: string) {
  const client = new KnirvClient({ bootstrapPeers: [...] });
  try {
    await client.bootstrap();
    return await client.fetchResource(uriString);
  } catch (e) {
    if (e instanceof ResourceNotFoundError) {
      console.error(`Resource not found: ${uriString}`);
    } else if (e instanceof ConnectionFailedError) {
      console.error(`Failed to connect to peer: ${e.message}`);
    } else if (e instanceof FetchFailedError) {
      console.error(`Failed to fetch resource: ${e.message}`);
    } else if (e instanceof KnirvClientError) {
      console.error(`Unexpected client error: ${e.message}`);
    } else {
      console.error(`Unknown error: ${e instanceof Error ? e.message : String(e)}`);
    }
    throw e;
  } finally {
    await client.stop();
  }
}
```

## Example

See the `examples` directory for complete examples of using the SDK.

## Browser Support

This SDK can be used in both Node.js and browser environments. When using in a browser, you'll need to use a bundler like webpack, Rollup, or Parcel that can handle Node.js modules.

## License

[MIT License](LICENSE)