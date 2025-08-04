/**
 * Client module for the KNIRV Client SDK.
 * 
 * This module provides the KnirvClient class for interacting with the KNIRVCHAIN network.
 */

import { createLibp2p, Libp2p } from 'libp2p';
import { noise } from '@chainsafe/libp2p-noise';
import { yamux } from '@chainsafe/libp2p-yamux';
import { tcp } from '@libp2p/tcp';
import { mplex } from '@libp2p/mplex';
import { bootstrap } from '@libp2p/bootstrap';
import { kadDHT } from '@libp2p/kad-dht';
import { multiaddr } from '@multiformats/multiaddr';
import { peerIdFromString } from '@libp2p/peer-id';
import { sha256 } from 'multiformats/hashes/sha2';
import { CID } from 'multiformats/cid';
import { base58btc } from 'multiformats/bases/base58';
import { toString as uint8ArrayToString } from 'uint8arrays/to-string';
import { fromString as uint8ArrayFromString } from 'uint8arrays/from-string';
import { KnirvURI, parseKnirvURI } from './parser';

/**
 * Base error class for KNIRV client errors.
 */
export class KnirvClientError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'KnirvClientError';
  }
}

/**
 * Error thrown when a resource is not found.
 */
export class ResourceNotFoundError extends KnirvClientError {
  constructor(message: string) {
    super(message);
    this.name = 'ResourceNotFoundError';
  }
}

/**
 * Error thrown when connection to a peer fails.
 */
export class ConnectionFailedError extends KnirvClientError {
  constructor(message: string) {
    super(message);
    this.name = 'ConnectionFailedError';
  }
}

/**
 * Error thrown when fetching a resource fails.
 */
export class FetchFailedError extends KnirvClientError {
  constructor(message: string) {
    super(message);
    this.name = 'FetchFailedError';
  }
}

/**
 * Error thrown when operations are attempted on a closed client.
 */
export class ClientClosedError extends KnirvClientError {
  constructor(message: string = 'Client is closed') {
    super(message);
    this.name = 'ClientClosedError';
  }
}

/**
 * Configuration for the KNIRV client.
 */
export interface KnirvClientConfig {
  /**
   * List of bootstrap peer multiaddresses.
   */
  bootstrapPeers: string[];

  /**
   * Port to use for the libp2p host (0 for random).
   */
  p2pPort?: number;

  /**
   * Whether to enable logging.
   */
  logEnabled?: boolean;
}

/**
 * Default configuration for the KNIRV client.
 */
export function defaultConfig(): KnirvClientConfig {
  return {
    bootstrapPeers: [
      '/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ',
      '/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM',
      '/ip4/128.199.219.111/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu',
      '/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64',
    ],
    p2pPort: 0,
    logEnabled: false,
  };
}

/**
 * Represents data fetched from a KNIRV resource.
 */
export class ResourceData {
  private readonly _data: Uint8Array;

  /**
   * Create a new ResourceData.
   * 
   * @param data - The raw byte data.
   */
  constructor(data: Uint8Array) {
    this._data = data;
  }

  /**
   * Get the raw byte data.
   * 
   * @returns The raw byte data as a Uint8Array.
   */
  bytes(): Uint8Array {
    return this._data;
  }

  /**
   * Get the data as a string.
   * 
   * @param encoding - The encoding to use (default: 'utf-8').
   * @returns The data as a string.
   */
  toString(encoding: string = 'utf-8'): string {
    return uint8ArrayToString(this._data, encoding);
  }
}

/**
 * Client for interacting with the KNIRVCHAIN network.
 */
export class KnirvClient {
  private readonly config: KnirvClientConfig;
  private libp2p: Libp2p | null = null;
  private closed: boolean = false;
  private bootstrapped: boolean = false;

  /**
   * Create a new KnirvClient.
   * 
   * @param config - The client configuration.
   */
  constructor(config: KnirvClientConfig) {
    this.config = {
      ...defaultConfig(),
      ...config,
    };
  }

  /**
   * Initialize the libp2p node.
   * 
   * @returns A promise that resolves when the node is initialized.
   * @throws {KnirvClientError} If initialization fails.
   */
  private async initLibp2p(): Promise<void> {
    try {
      this.libp2p = await createLibp2p({
        addresses: {
          listen: [`/ip4/0.0.0.0/tcp/${this.config.p2pPort || 0}`],
        },
        transports: [tcp()],
        streamMuxers: [yamux(), mplex()],
        connectionEncryption: [noise()],
        dht: kadDHT({
          clientMode: true,
        }),
        peerDiscovery: [
          bootstrap({
            list: this.config.bootstrapPeers,
          }),
        ],
      });

      // Register protocol handlers
      this.registerProtocolHandlers();

      if (this.config.logEnabled) {
        console.log(`KNIRV Client started with PeerID: ${this.libp2p.peerId.toString()}`);
        for (const addr of this.libp2p.getMultiaddrs()) {
          console.log(`Listening on: ${addr.toString()}`);
        }
      }
    } catch (e) {
      throw new KnirvClientError(`Failed to initialize libp2p: ${e instanceof Error ? e.message : String(e)}`);
    }
  }

  /**
   * Register protocol handlers.
   */
  private registerProtocolHandlers(): void {
    if (!this.libp2p) return;

    // Register handler for the resource fetch protocol
    this.libp2p.handle('/knirv/resource-fetch/1.0.0', async ({ stream }) => {
      // This is a client, so we don't expect incoming resource fetch requests
      // But we need to handle them gracefully
      try {
        await stream.sink([uint8ArrayFromString('ERROR: This is a client node and does not serve resources')]);
      } catch (e) {
        if (this.config.logEnabled) {
          console.error('Error handling incoming stream:', e);
        }
      } finally {
        await stream.close();
      }
    });
  }

  /**
   * Bootstrap the client by connecting to bootstrap peers and joining the DHT network.
   * 
   * @returns A promise that resolves when bootstrapping is complete.
   * @throws {ClientClosedError} If the client is closed.
   * @throws {KnirvClientError} If bootstrapping fails.
   */
  async bootstrap(): Promise<void> {
    if (this.closed) {
      throw new ClientClosedError();
    }

    if (this.bootstrapped) {
      return;
    }

    // Initialize libp2p if not already initialized
    if (!this.libp2p) {
      await this.initLibp2p();
    }

    if (!this.libp2p) {
      throw new KnirvClientError('Failed to initialize libp2p');
    }

    if (this.config.logEnabled) {
      console.log('Connecting to bootstrap nodes...');
    }

    // Start the libp2p node
    await this.libp2p.start();

    // Wait for some connections to bootstrap peers
    try {
      await new Promise<void>((resolve, reject) => {
        const timeout = setTimeout(() => {
          resolve(); // Resolve anyway after timeout
        }, 30000);

        // Listen for peer connection events
        const onConnect = () => {
          if (this.libp2p?.getConnections().length) {
            clearTimeout(timeout);
            this.libp2p.removeEventListener('peer:connect', onConnect);
            resolve();
          }
        };

        this.libp2p?.addEventListener('peer:connect', onConnect);

        // Check if we already have connections
        if (this.libp2p?.getConnections().length) {
          clearTimeout(timeout);
          this.libp2p.removeEventListener('peer:connect', onConnect);
          resolve();
        }
      });
    } catch (e) {
      if (this.config.logEnabled) {
        console.warn('Bootstrap connection timeout, continuing with available connections');
      }
    }

    this.bootstrapped = true;
    if (this.config.logEnabled) {
      console.log('DHT bootstrapped successfully');
    }
  }

  /**
   * Close the client and release resources.
   * 
   * @returns A promise that resolves when the client is closed.
   */
  async stop(): Promise<void> {
    if (this.closed) {
      return;
    }

    this.closed = true;

    if (this.libp2p) {
      await this.libp2p.stop();
      this.libp2p = null;
    }

    if (this.config.logEnabled) {
      console.log('Client closed');
    }
  }

  /**
   * Fetch a resource from the KNIRVCHAIN network.
   * 
   * @param uriString - The URI string of the resource to fetch.
   * @returns A promise that resolves to a ResourceData object containing the fetched data.
   * @throws {KnirvURIError} If the URI is invalid.
   * @throws {ResourceNotFoundError} If the resource is not found.
   * @throws {ConnectionFailedError} If connection to a peer fails.
   * @throws {FetchFailedError} If fetching the resource fails.
   * @throws {ClientClosedError} If the client is closed.
   */
  async fetchResource(uriString: string): Promise<ResourceData> {
    if (this.closed) {
      throw new ClientClosedError();
    }

    // Ensure client is bootstrapped
    if (!this.bootstrapped) {
      await this.bootstrap();
    }

    // Parse the URI
    const uri = parseKnirvURI(uriString);

    // Find providers for the resource
    const providers = await this.findResourceProviders(uri.id, uri.resourceType);

    // Try to connect to providers and fetch the resource
    for (const provider of providers) {
      try {
        const data = await this.fetchFromProvider(provider, uri);
        return new ResourceData(data);
      } catch (e) {
        if (this.config.logEnabled) {
          console.warn(`Failed to fetch from provider ${provider.toString()}: ${e instanceof Error ? e.message : String(e)}`);
        }
        continue;
      }
    }

    throw new ResourceNotFoundError(`No providers found for resource: ${uriString}`);
  }

  /**
   * Find providers for a resource in the DHT.
   * 
   * @param id - The ID part of the resource.
   * @param resourceType - The resource type part of the resource.
   * @returns A promise that resolves to an array of peer IDs for providers of the resource.
   * @throws {ResourceNotFoundError} If no providers are found.
   */
  private async findResourceProviders(id: string, resourceType: string): Promise<string[]> {
    if (!this.libp2p) {
      throw new KnirvClientError('libp2p not initialized');
    }

    // Create a CID from the resource identifier
    const resourceKey = `${id}.${resourceType}`;
    const hash = await sha256.digest(uint8ArrayFromString(resourceKey));
    const cid = CID.create(1, 0x55, hash);

    if (this.config.logEnabled) {
      console.log(`Looking for providers of resource: ${resourceKey} (CID: ${cid.toString()})`);
    }

    // Find providers for this resource
    try {
      const providers = await this.libp2p.contentRouting.findProviders(cid, { timeout: 30000 });
      
      if (!providers.length) {
        throw new ResourceNotFoundError(`No providers found for resource: ${resourceKey}`);
      }

      return providers.map(provider => provider.id.toString());
    } catch (e) {
      throw new ResourceNotFoundError(`Failed to find providers: ${e instanceof Error ? e.message : String(e)}`);
    }
  }

  /**
   * Fetch a resource from a specific provider.
   * 
   * @param providerId - The peer ID of the provider.
   * @param uri - The parsed URI.
   * @returns A promise that resolves to the fetched data as a Uint8Array.
   * @throws {ConnectionFailedError} If connection to the peer fails.
   * @throws {FetchFailedError} If fetching the resource fails.
   */
  private async fetchFromProvider(providerId: string, uri: KnirvURI): Promise<Uint8Array> {
    if (!this.libp2p) {
      throw new KnirvClientError('libp2p not initialized');
    }

    // Connect to the provider
    try {
      const peerId = peerIdFromString(providerId);
      await this.libp2p.dial(peerId);
    } catch (e) {
      throw new ConnectionFailedError(`Failed to connect to peer: ${e instanceof Error ? e.message : String(e)}`);
    }

    // Open a stream to the provider
    let stream;
    try {
      const peerId = peerIdFromString(providerId);
      stream = await this.libp2p.dialProtocol(peerId, '/knirv/resource-fetch/1.0.0');
    } catch (e) {
      throw new ConnectionFailedError(`Failed to open stream: ${e instanceof Error ? e.message : String(e)}`);
    }

    // Prepare the request
    let request = `GET ${uri.path}`;
    if (uri.query.toString()) {
      request += `?${uri.query.toString()}`;
    }
    request += '\n';

    // Send the request
    if (this.config.logEnabled) {
      console.log(`Sending request to ${providerId}: ${request}`);
    }

    try {
      await stream.sink([uint8ArrayFromString(request)]);
    } catch (e) {
      throw new FetchFailedError(`Failed to send request: ${e instanceof Error ? e.message : String(e)}`);
    }

    // Read the response
    let responseData: Uint8Array;
    try {
      const chunks = [];
      for await (const chunk of stream.source) {
        chunks.push(chunk);
      }
      responseData = new Uint8Array(chunks.reduce((acc, chunk) => acc + chunk.length, 0));
      let offset = 0;
      for (const chunk of chunks) {
        responseData.set(chunk, offset);
        offset += chunk.length;
      }
    } catch (e) {
      throw new FetchFailedError(`Failed to read response: ${e instanceof Error ? e.message : String(e)}`);
    }

    if (!responseData.length) {
      throw new FetchFailedError('Empty response from provider');
    }

    // Check if the response indicates an error
    const responsePrefix = uint8ArrayToString(responseData.slice(0, 6));
    if (responsePrefix === 'ERROR:') {
      throw new FetchFailedError(uint8ArrayToString(responseData));
    }

    return responseData;
  }

  /**
   * Get the peer ID of this client.
   * 
   * @returns The peer ID as a string.
   * @throws {KnirvClientError} If the client is not initialized.
   */
  getPeerId(): string {
    if (!this.libp2p) {
      throw new KnirvClientError('libp2p not initialized');
    }
    return this.libp2p.peerId.toString();
  }

  /**
   * Get the multiaddresses of this client.
   * 
   * @returns An array of multiaddress strings.
   * @throws {KnirvClientError} If the client is not initialized.
   */
  getMultiaddrs(): string[] {
    if (!this.libp2p) {
      throw new KnirvClientError('libp2p not initialized');
    }
    return this.libp2p.getMultiaddrs().map(addr => addr.toString());
  }
}