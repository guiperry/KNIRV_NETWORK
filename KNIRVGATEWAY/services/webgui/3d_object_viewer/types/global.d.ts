// Global type declarations for the project

// Declare Node.js modules for client-side code
declare module 'crypto' {
  export * from 'crypto';
}

declare module 'fs' {
  export * from 'fs';
}

declare module 'http' {
  export * from 'http';
}

declare module 'path' {
  export * from 'path';
}

declare module 'url' {
  export * from 'url';
}

declare module 'os' {
  export * from 'os';
}

declare module 'child_process' {
  export * from 'child_process';
}

declare module 'util' {
  export * from 'util';
}

// Declare global process variable
declare const process: {
  env: {
    [key: string]: string | undefined;
  };
  browser: boolean;
  version: string;
  platform: string;
  cwd(): string;
  [key: string]: unknown;
};

// Declare Buffer global
declare const Buffer: {
  from(data: string | ArrayBuffer, encoding?: string): Buffer;
  alloc(size: number): Buffer;
  allocUnsafe(size: number): Buffer;
  allocUnsafeSlow(size: number): Buffer;
  byteLength(string: string | ArrayBuffer | SharedArrayBuffer, encoding?: string): number;
  compare(buf1: Buffer, buf2: Buffer): number;
  concat(list: Buffer[], totalLength?: number): Buffer;
  isBuffer(obj: unknown): boolean;
  isEncoding(encoding: string): boolean;
  poolSize: number;
};