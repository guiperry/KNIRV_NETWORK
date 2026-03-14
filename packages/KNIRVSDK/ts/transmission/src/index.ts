/**
 * KNIRV Client SDK for JavaScript/TypeScript.
 * 
 * This package provides a high-level interface for interacting with the KNIRVCHAIN network.
 * It simplifies the process of resolving `knirv://` URIs, discovering peers on the private DHT,
 * connecting to them, and fetching the underlying resources.
 */

export { KnirvURI, parseKnirvURI, KnirvURIError } from './parser';
export {
  KnirvClient,
  KnirvClientConfig,
  defaultConfig,
  ResourceData,
  KnirvClientError,
  ResourceNotFoundError,
  ConnectionFailedError,
  FetchFailedError,
  ClientClosedError,
} from './client';