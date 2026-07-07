/**
 * URI parser module for the KNIRV Client SDK.
 * 
 * This module provides functions for parsing knirv:// URIs.
 */

/**
 * Error class for invalid KNIRV URIs.
 */
export class KnirvURIError extends Error {
  /**
   * The URI that caused the error.
   */
  public readonly uri: string;

  /**
   * Create a new KnirvURIError.
   * 
   * @param message - The error message.
   * @param uri - The URI that caused the error.
   */
  constructor(message: string, uri: string) {
    super(`Invalid knirv URI (${uri}): ${message}`);
    this.name = 'KnirvURIError';
    this.uri = uri;
  }
}

/**
 * Represents a parsed knirv:// URI.
 */
export class KnirvURI {
  /**
   * The ID part of the URI.
   */
  public readonly id: string;

  /**
   * The resource type part of the URI.
   */
  public readonly resourceType: string;

  /**
   * The path part of the URI.
   */
  public readonly path: string;

  /**
   * The query parameters as a URLSearchParams object.
   */
  public readonly query: URLSearchParams;

  /**
   * The original URI string.
   */
  public readonly raw: string;

  /**
   * Create a new KnirvURI.
   * 
   * @param id - The ID part of the URI.
   * @param resourceType - The resource type part of the URI.
   * @param path - The path part of the URI.
   * @param query - The query parameters as a URLSearchParams object.
   * @param raw - The original URI string.
   */
  constructor(id: string, resourceType: string, path: string, query: URLSearchParams, raw: string) {
    this.id = id;
    this.resourceType = resourceType;
    this.path = path;
    this.query = query;
    this.raw = raw;
  }

  /**
   * Get the string representation of the URI.
   * 
   * @returns The original URI string.
   */
  toString(): string {
    return this.raw;
  }

  /**
   * Get the first value for a query parameter.
   * 
   * @param key - The query parameter key.
   * @returns The first value for the query parameter, or null if it doesn't exist.
   */
  getQueryParam(key: string): string | null {
    return this.query.get(key);
  }

  /**
   * Check if a query parameter exists.
   * 
   * @param key - The query parameter key.
   * @returns True if the query parameter exists, false otherwise.
   */
  hasQueryParam(key: string): boolean {
    return this.query.has(key);
  }

  /**
   * Get all values for a query parameter.
   * 
   * @param key - The query parameter key.
   * @returns An array of values for the query parameter.
   */
  getQueryParams(key: string): string[] {
    return this.query.getAll(key);
  }

  /**
   * Get all query parameters.
   * 
   * @returns An object containing all query parameters.
   */
  getAllQueryParams(): Record<string, string[]> {
    const result: Record<string, string[]> = {};
    for (const [key, value] of this.query.entries()) {
      if (!result[key]) {
        result[key] = [];
      }
      result[key].push(value);
    }
    return result;
  }
}

/**
 * Parse a knirv:// URI string into a KnirvURI object.
 * 
 * @param uriString - The URI string to parse.
 * @returns A KnirvURI object representing the parsed URI.
 * @throws {KnirvURIError} If the URI is invalid.
 */
export function parseKnirvURI(uriString: string): KnirvURI {
  let parsedUrl: URL;
  
  try {
    parsedUrl = new URL(uriString);
  } catch (e) {
    throw new KnirvURIError(`Failed to parse URI: ${e instanceof Error ? e.message : String(e)}`, uriString);
  }

  // Check scheme
  if (parsedUrl.protocol !== 'knirv:') {
    throw new KnirvURIError(`Invalid scheme: expected 'knirv:', got '${parsedUrl.protocol}'`, uriString);
  }

  // Extract ID and ResourceType from authority
  const host = parsedUrl.hostname;
  const hostParts = host.split('.');
  
  if (hostParts.length !== 2) {
    throw new KnirvURIError('Invalid authority format: expected <ID>.<ResourceType>', uriString);
  }

  const id = hostParts[0];
  const resourceType = hostParts[1];

  // Validate ID and ResourceType are non-empty
  if (!id || !resourceType) {
    throw new KnirvURIError('ID and ResourceType cannot be empty', uriString);
  }

  // Normalize path
  let path = parsedUrl.pathname;
  if (!path) {
    path = '/'; // Default path
  }

  // Parse query parameters
  const query = parsedUrl.searchParams;

  return new KnirvURI(
    id,
    resourceType,
    path,
    query,
    uriString
  );
}