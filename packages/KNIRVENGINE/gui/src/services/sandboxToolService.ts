/**
 * Sandbox Tool Service
 *
 * REST + WebSocket client for the tool execution API surface
 * (`/api/v1/sandboxes/{id}/tools/*`). Kept separate from sandboxService.ts
 * which stays session-lifecycle-only.
 */

import { getApiUrl, getWebSocketUrl } from '../utils/apiBase';

const API_BASE_PATH = '/api/v1/sandboxes';
const apiUrl = (sessionID: string, path: string) =>
  getApiUrl(`${API_BASE_PATH}/${sessionID}/tools${path}`);

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

// Types matching the Go backend responses.

export interface ToolScanResult {
  tool: string;
  rawOutput?: string;
  structured?: unknown;
  startedAt: string;
  durationMs: number;
}

export interface ToolEvent {
  tool: string;
  timestamp: string;
  type: string;
  payload?: unknown;
  rawLine?: string;
}

export interface ToolAttachState {
  tool: string;
  attached: boolean;
  pid?: number;
  log: string[];
}

export interface ToolWebSocketHandlers {
  onOpen?: (event: Event) => void;
  onMessage?: (event: MessageEvent) => void;
  onClose?: (event: CloseEvent) => void;
  onError?: (event: Event) => void;
}

// Lane 1 & Lane 6: Batch scan / native execution.

export const runToolScan = async (
  sessionID: string,
  tool: string,
  args: Record<string, unknown> = {}
): Promise<ToolScanResult> => {
  const response = await fetch(apiUrl(sessionID, `/${tool}/run`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(args),
  });
  if (!response.ok) throw await parseError(response);
  return readData<ToolScanResult>(response);
};

export const runToolNative = async (
  sessionID: string,
  tool: string,
  args: Record<string, unknown> = {}
): Promise<ToolScanResult> => {
  const response = await fetch(apiUrl(sessionID, `/${tool}/native`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(args),
  });
  if (!response.ok) throw await parseError(response);
  return readData<ToolScanResult>(response);
};

// Lane 2: Streaming daemon.

export const startToolStream = async (
  sessionID: string,
  tool: string,
  args: Record<string, unknown> = {}
): Promise<{ status: string; tool: string }> => {
  const response = await fetch(apiUrl(sessionID, `/${tool}/start`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(args),
  });
  if (!response.ok) throw await parseError(response);
  return readData<{ status: string; tool: string }>(response);
};

export const stopToolStream = async (
  sessionID: string,
  tool: string
): Promise<{ status: string; tool: string }> => {
  const response = await fetch(apiUrl(sessionID, `/${tool}/stop`), {
    method: 'POST',
  });
  if (!response.ok) throw await parseError(response);
  return readData<{ status: string; tool: string }>(response);
};

export const getToolStreamWebSocketUrl = (sessionID: string, tool: string): string =>
  getWebSocketUrl(`${API_BASE_PATH}/${sessionID}/tools/${tool}/ws`);

export const createToolStreamWebSocket = (
  sessionID: string,
  tool: string,
  handlers: ToolWebSocketHandlers = {}
): WebSocket => {
  const ws = new WebSocket(getToolStreamWebSocketUrl(sessionID, tool));
  if (handlers.onOpen) ws.onopen = handlers.onOpen;
  if (handlers.onMessage) ws.onmessage = handlers.onMessage;
  if (handlers.onClose) ws.onclose = handlers.onClose;
  if (handlers.onError) ws.onerror = handlers.onError;
  return ws;
};

// Lane 3: RPC attach.

export const attachTool = async (
  sessionID: string,
  tool: string,
  pid: number,
  args: Record<string, unknown> = {}
): Promise<ToolAttachState> => {
  const response = await fetch(apiUrl(sessionID, `/${tool}/attach`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pid, args }),
  });
  if (!response.ok) throw await parseError(response);
  return readData<ToolAttachState>(response);
};

export const detachTool = async (
  sessionID: string,
  tool: string
): Promise<{ status: string; tool: string }> => {
  const response = await fetch(apiUrl(sessionID, `/${tool}/detach`), {
    method: 'POST',
  });
  if (!response.ok) throw await parseError(response);
  return readData<{ status: string; tool: string }>(response);
};

export const getToolAttachWebSocketUrl = (sessionID: string, tool: string): string =>
  getWebSocketUrl(`${API_BASE_PATH}/${sessionID}/tools/${tool}/attach/ws`);

// Lane 4: Launch modifier (proxychains).

export const configureProxychains = async (
  sessionID: string,
  config: Record<string, unknown>
): Promise<{ configPath: string; status: string }> => {
  const response = await fetch(apiUrl(sessionID, '/proxychains/configure'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
  if (!response.ok) throw await parseError(response);
  return readData<{ configPath: string; status: string }>(response);
};

// Lane 5: Headless with native UI.

export const runToolAnalysis = async (
  sessionID: string,
  tool: string,
  args: Record<string, unknown> = {}
): Promise<ToolScanResult> => {
  const response = await fetch(apiUrl(sessionID, `/${tool}/analyze`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(args),
  });
  if (!response.ok) throw await parseError(response);
  return readData<ToolScanResult>(response);
};

const sandboxToolService = {
  runToolScan,
  runToolNative,
  startToolStream,
  stopToolStream,
  getToolStreamWebSocketUrl,
  createToolStreamWebSocket,
  attachTool,
  detachTool,
  getToolAttachWebSocketUrl,
  configureProxychains,
  runToolAnalysis,
};

export default sandboxToolService;
