/**
 * Minimal ambient declarations for the Node `fs` surface used by the file-backed
 * NonceStore. The SDK is compiled without @types/node (browser-compatible target),
 * so only the few calls we use are typed here.
 */
declare module "node:fs" {
  export const constants: {
    F_OK: number;
    LOCK_EX: number;
    LOCK_UN: number;
    LOCK_SH: number;
  };
  export function openSync(path: string, flags: string | number, mode?: number): number;
  export function closeSync(fd: number): void;
  export function flockSync(fd: number, mode: number): void;
  export function readFileSync(path: string, encoding?: string): string;
  export function writeFileSync(path: string, data: string | Uint8Array): void;
  export function renameSync(oldPath: string, newPath: string): void;
  export function mkdirSync(path: string, options?: { recursive?: boolean }): void;
  export function existsSync(path: string): boolean;
  export function mkdtempSync(prefix: string): string;
  export function rmSync(path: string, options?: { recursive?: boolean; force?: boolean }): void;
  export function statSync(path: string): { mtimeMs: number };
  export function unlinkSync(path: string): void;
}

declare module "node:child_process" {
  export interface ChildProcess {
    stdout: { on(event: string, cb: (data: Uint8Array | string) => void): void };
    on(event: "close", cb: (code?: number) => void): void;
    on(event: "error", cb: (err: unknown) => void): void;
  }
  export function spawn(command: string, args?: string[], options?: { cwd?: string }): ChildProcess;
}

declare module "node:url" {
  export function fileURLToPath(url: string | URL): string;
  export function pathToFileURL(path: string): URL;
}

declare const process: {
  cwd(): string;
  argv: string[];
  env: Record<string, string | undefined>;
};
