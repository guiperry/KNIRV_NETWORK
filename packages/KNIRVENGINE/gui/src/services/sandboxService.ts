/**
 * Sandbox Service
 *
 * REST + WebSocket client for the KNIRVENGINE SandboxManager backend
 * (`api/sandbox_manager.go`). Mirrors the shape of `terminalService.js`,
 * ported to TypeScript with the `SandboxSession` wire contract from
 * `sandbox_implementation.md`.
 */

import { getApiUrl, getWebSocketUrl } from '../utils/apiBase';

export type SandboxSessionStatus =
  | 'idle'
  | 'provisioning'
  | 'running'
  | 'stopping'
  | 'stopped'
  | 'error';

export interface SandboxBind {
  mode: 'ro-bind' | 'bind' | 'tmpfs';
  src: string;
  dst: string;
}

export interface SandboxLaunchConfig {
  targetLabel: string;
  targetCommand: string;
  targetArgs?: string[];
  display?: string;
  binds?: SandboxBind[];
  unshareAll?: boolean;
  shareNet?: boolean;
  dieWithParent?: boolean;
}

export interface SandboxSession {
  id: string;
  createdAt: string;
  lastActivity: string;
  userId: number;
  targetLabel: string;
  targetCommand: string;
  status: SandboxSessionStatus;
  error?: string;
  pid?: number;
  display: string;
  netnsId: string;
  vncPort: number;
  vncWsPath?: string;
  statusWsPath: string;
  frontendUrl?: string;
  proxyPort?: number;
  clientCount?: number;
}

export interface SandboxProxyFlow {
  id: number; method: string; host: string; path: string; status: number;
  contentType?: string; size: number; durationMs: number; error?: string;
}

const API_BASE_PATH = '/api/v1/sandboxes';
const apiUrl = (path = '') => getApiUrl(`${API_BASE_PATH}${path}`);

interface StandardResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

const parseError = async (response: Response): Promise<Error> => {
  try {
    const data = await response.json();
    return new Error(data.error || `Request failed with status ${response.status}`);
  } catch {
    return new Error(`Request failed with status ${response.status}`);
  }
};

const readData = async <T>(response: Response): Promise<T> => {
  const payload = (await response.json()) as StandardResponse<T>;
  if (!payload.success) throw new Error(payload.error || `Request failed with status ${response.status}`);
  return payload.data as T;
};

export const createSandboxSession = async (
  config: SandboxLaunchConfig
): Promise<SandboxSession> => {
  const response = await fetch(apiUrl(), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
  if (!response.ok) throw await parseError(response);
  return readData<SandboxSession>(response);
};

export const getSandboxSession = async (id: string): Promise<SandboxSession> => {
  const response = await fetch(apiUrl(`/${id}`));
  if (!response.ok) throw await parseError(response);
  return readData<SandboxSession>(response);
};

export const listSandboxSessions = async (): Promise<SandboxSession[]> => {
  const response = await fetch(apiUrl());
  if (!response.ok) throw await parseError(response);
  const data = await readData<{ sandboxes: SandboxSession[] }>(response);
  return data.sandboxes ?? [];
};

export const closeSandboxSession = async (id: string): Promise<void> => {
  const response = await fetch(apiUrl(`/${id}`), { method: 'DELETE' });
  if (!response.ok && response.status !== 204) throw await parseError(response);
};

export const getSandboxStatusWebSocketUrl = (id: string): string => {
  return getWebSocketUrl(`${API_BASE_PATH}/${id}/ws`);
};

export const getSandboxVncWebSocketUrl = (id: string): string => {
  return getWebSocketUrl(`${API_BASE_PATH}/${id}/vnc`);
};

export interface DependencyStatus {
  binary: string;
  present: boolean;
  package?: string;
  installCommand?: string;
  error?: string;
}

export interface SandboxDependencyReport {
  ok: boolean;
  error?: string;
  dependencies: DependencyStatus[];
}

export const getSandboxDependencies = async (): Promise<SandboxDependencyReport> => {
  const response = await fetch(apiUrl('/deps'));
  if (!response.ok) throw await parseError(response);
  return response.json();
};

export const installSandboxDependencies = async (): Promise<SandboxDependencyReport> => {
  const response = await fetch(apiUrl('/deps/install'), { method: 'POST' });
  if (!response.ok) throw await parseError(response);
  return response.json();
};

export interface SandboxWebSocketHandlers {
  onOpen?: (event: Event) => void;
  onMessage?: (event: MessageEvent) => void;
  onClose?: (event: CloseEvent) => void;
  onError?: (event: Event) => void;
}

export const createSandboxStatusWebSocket = (
  id: string,
  handlers: SandboxWebSocketHandlers = {}
): WebSocket => {
  const ws = new WebSocket(getSandboxStatusWebSocketUrl(id));
  if (handlers.onOpen) ws.onopen = handlers.onOpen;
  if (handlers.onMessage) ws.onmessage = handlers.onMessage;
  if (handlers.onClose) ws.onclose = handlers.onClose;
  if (handlers.onError) ws.onerror = handlers.onError;
  return ws;
};

const sandboxService = {
  createSandboxSession,
  getSandboxSession,
  listSandboxSessions,
  closeSandboxSession,
  getSandboxStatusWebSocketUrl,
  getSandboxVncWebSocketUrl,
  createSandboxStatusWebSocket,
  getSandboxDependencies,
  installSandboxDependencies,
};

export default sandboxService;
