/**
 * URI parser module for the KNIRV Client SDK.
 *
 * This module provides functions for parsing knirv:// URIs.
 */
/**
 * Error class for invalid KNIRV URIs.
 */
export declare class KnirvURIError extends Error {
    /**
     * The URI that caused the error.
     */
    readonly uri: string;
    /**
     * Create a new KnirvURIError.
     *
     * @param message - The error message.
     * @param uri - The URI that caused the error.
     */
    constructor(message: string, uri: string);
}
/**
 * Represents a parsed knirv:// URI.
 */
export declare class KnirvURI {
    /**
     * The ID part of the URI.
     */
    readonly id: string;
    /**
     * The resource type part of the URI.
     */
    readonly resourceType: string;
    /**
     * The path part of the URI.
     */
    readonly path: string;
    /**
     * The query parameters as a URLSearchParams object.
     */
    readonly query: URLSearchParams;
    /**
     * The original URI string.
     */
    readonly raw: string;
    /**
     * Create a new KnirvURI.
     *
     * @param id - The ID part of the URI.
     * @param resourceType - The resource type part of the URI.
     * @param path - The path part of the URI.
     * @param query - The query parameters as a URLSearchParams object.
     * @param raw - The original URI string.
     */
    constructor(id: string, resourceType: string, path: string, query: URLSearchParams, raw: string);
    /**
     * Get the string representation of the URI.
     *
     * @returns The original URI string.
     */
    toString(): string;
    /**
     * Get the first value for a query parameter.
     *
     * @param key - The query parameter key.
     * @returns The first value for the query parameter, or null if it doesn't exist.
     */
    getQueryParam(key: string): string | null;
    /**
     * Check if a query parameter exists.
     *
     * @param key - The query parameter key.
     * @returns True if the query parameter exists, false otherwise.
     */
    hasQueryParam(key: string): boolean;
    /**
     * Get all values for a query parameter.
     *
     * @param key - The query parameter key.
     * @returns An array of values for the query parameter.
     */
    getQueryParams(key: string): string[];
    /**
     * Get all query parameters.
     *
     * @returns An object containing all query parameters.
     */
    getAllQueryParams(): Record<string, string[]>;
}
/**
 * Parse a knirv:// URI string into a KnirvURI object.
 *
 * @param uriString - The URI string to parse.
 * @returns A KnirvURI object representing the parsed URI.
 * @throws {KnirvURIError} If the URI is invalid.
 */
export declare function parseKnirvURI(uriString: string): KnirvURI;
//# sourceMappingURL=parser.d.ts.map