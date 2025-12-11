// Stub TypeScript definitions for @cloudflare/workers-types
// Used when building on Render where Cloudflare types are not available

// Global declarations for Cloudflare Workers types
declare global {
  interface ExecutionContext {
    waitUntil(promise: Promise<any>): void;
    passThroughOnException(): void;
  }

  interface RequestInitCfProperties {
    cacheEverything?: boolean;
    cacheKey?: string;
    cacheTtl?: number;
    cacheTtlByStatus?: { [key: string]: number };
  }

  interface RequestInit {
    cf?: RequestInitCfProperties;
  }

  interface Env {
    [key: string]: any;
  }
}

declare module '@cloudflare/workers-types' {
  // Minimal stub definitions to satisfy TypeScript compilation
  interface RequestInitCfProperties {
    cacheEverything?: boolean;
    cacheKey?: string;
    cacheTtl?: number;
    cacheTtlByStatus?: { [key: string]: number };
  }

  interface RequestInit {
    cf?: RequestInitCfProperties;
  }

  interface ExecutionContext {
    waitUntil(promise: Promise<any>): void;
    passThroughOnException(): void;
  }

  interface Env {
    [key: string]: any;
  }

  export interface WorkerEntrypoint {
    fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response>;
  }
}

// Export the module to satisfy TypeScript
declare const workersTypes: any;
export = workersTypes;