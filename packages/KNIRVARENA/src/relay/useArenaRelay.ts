/**
 * Arena relay hook (arena_fixes.md §2.2–2.4).
 *
 * - Requests (chat.invoke, arena.status, arena.agent.deploy/recall) are signed
 *   with the arena's own per-pairing secp256k1 key in the relay worker and
 *   POSTed to the CLISupervisorHub relay endpoint; the CLI's ed25519-signed
 *   response is verified and its key pinned for the pairing.
 * - Events from the CLI (`arena.agent.*`, `arena.test.result`,
 *   `arena.chat.reply`) arrive on `/ws/cli-supervisor/{pairingId}/arena` and are
 *   verified the same way, against the pinned CLI key, before dispatch. The
 *   socket is opened with a single-use ticket from
 *   `POST …/sessions/{pairingId}/arena-ticket`, so the JWT never goes in a URL.
 *
 * The pairing link is `{gateway}/arena/?pair={pairingId}&lease={leaseEpoch}`.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  ArenaRelayClient,
  ArenaRelayError,
  RELAY_CAPABILITY_ARENA_AGENT_DEPLOY,
  RELAY_CAPABILITY_ARENA_AGENT_RECALL,
  RELAY_CAPABILITY_ARENA_STATUS,
  RELAY_CAPABILITY_CHAT_INVOKE,
} from './arenaRelayClient';
import { arenaRelayWorkerClient } from './arenaRelayWorkerClient';
import { ARENA_RELAY_CHAIN_ID } from './arenaRelayKeys';
import { verifyCliResponse, type CliResponsePayload } from './verifyCliResponse';
import { getKnirvServerUrl } from '../config/runtimeConfig';

export interface ArenaEvent {
  type: string;
  data: unknown;
  timestamp: number;
  pairingId: string;
  requestId: string;
}

export interface ArenaRelayOptions {
  userSubject: string;
  deviceId: string;
  pairingId: string;
  /** From the pairing link (`?lease=`); otherwise read from pairing status. */
  leaseEpoch?: number;
  onEvent?: (event: ArenaEvent) => void;
  onError?: (error: Error) => void;
  onConnect?: () => void;
  onDisconnect?: (code: number, reason: string) => void;
}

export interface ArenaRelayState {
  connected: boolean;
  connecting: boolean;
  status: 'disconnected' | 'connecting' | 'connected' | 'error';
  /** Whether the paired CLI is currently connected to the hub. */
  cliConnected: boolean;
  lastError: string | null;
  events: ArenaEvent[];
}

const AUTH_TOKEN_KEY = 'knirv_auth_token';
const PIN_PREFIX = 'knirv.arena.relay.cliKey.';
const WS_RECONNECT_BASE_DELAY_MS = 1000;
const WS_MAX_RECONNECT_ATTEMPTS = 10;

const readStorage = (key: string): string | null => {
  try { return localStorage.getItem(key); } catch { return null; }
};
const writeStorage = (key: string, value: string) => {
  try { localStorage.setItem(key, value); } catch { /* storage unavailable */ }
};

export const pinnedCliKey = (pairingId: string) => readStorage(PIN_PREFIX + pairingId) ?? undefined;

/** Pins the first verified CLI key for a pairing; later responses must match it. */
export const pinCliKey = (pairingId: string, publicKey: string) => {
  if (!pinnedCliKey(pairingId)) writeStorage(PIN_PREFIX + pairingId, publicKey);
};

function authToken(): string {
  const token = readStorage(AUTH_TOKEN_KEY);
  if (!token) throw new ArenaRelayError('Sign in to KNIRV before using your CLI from the arena', 401);
  return token;
}

/** Lease epoch from pairing status (only answers before the pairing is claimed). */
async function fetchLeaseEpoch(pairingId: string): Promise<number | undefined> {
  try {
    const response = await fetch(`${getKnirvServerUrl()}/api/v1/cli-supervisor/pairings/${encodeURIComponent(pairingId)}`, {
      headers: { Authorization: `Bearer ${authToken()}` },
    });
    if (!response.ok) return undefined;
    const body = (await response.json()) as { lease_epoch?: number };
    return Number.isSafeInteger(body.lease_epoch) && (body.lease_epoch ?? 0) > 0 ? body.lease_epoch : undefined;
  } catch {
    return undefined;
  }
}

const decodeJSON = (bytes?: Uint8Array): unknown => {
  if (!bytes || bytes.length === 0) return undefined;
  const text = new TextDecoder().decode(bytes);
  try { return JSON.parse(text); } catch { return text; }
};

export function useArenaRelay(opts: ArenaRelayOptions) {
  const { userSubject, deviceId, pairingId, onEvent, onError, onConnect, onDisconnect } = opts;

  const clientRef = useRef<ArenaRelayClient | null>(null);
  const leaseRef = useRef<number | undefined>(opts.leaseEpoch);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closedByUserRef = useRef(false);

  const [state, setState] = useState<ArenaRelayState>({
    connected: false,
    connecting: false,
    status: 'disconnected',
    cliConnected: false,
    lastError: null,
    events: [],
  });

  useEffect(() => { leaseRef.current = opts.leaseEpoch; }, [opts.leaseEpoch]);

  const client = useCallback((): ArenaRelayClient => {
    clientRef.current ??= new ArenaRelayClient((relayEnvelope, fields) =>
      arenaRelayWorkerClient.signRelayRequest(pairingId, relayEnvelope, fields),
    );
    return clientRef.current;
  }, [pairingId]);

  useEffect(() => { clientRef.current = null; }, [pairingId]);

  /** Sends one capability request and returns the CLI's verified, decoded result. */
  const sendRequest = useCallback(async (capability: string, payload: Uint8Array): Promise<unknown> => {
    if (!pairingId) throw new ArenaRelayError('Pair the arena with your CLI first');
    if (typeof navigator !== 'undefined' && !navigator.onLine) throw new ArenaRelayError('Relay actions require an online connection');
    const leaseEpoch = leaseRef.current ?? (leaseRef.current = await fetchLeaseEpoch(pairingId)) ?? 1;
    const signed = await client().createSignedRequest({ userSubject, deviceId, pairingId, capability, payload, leaseEpoch });
    const result = await client().send(signed, {
      serverUrl: getKnirvServerUrl(),
      token: authToken(),
      chainId: ARENA_RELAY_CHAIN_ID,
      pinnedPublicKey: pinnedCliKey(pairingId),
    });
    pinCliKey(pairingId, result.cliPublicKey);
    return decodeJSON(result.payload);
  }, [pairingId, userSubject, deviceId, client]);

  const handleWsMessage = useCallback(async (messageEvent: MessageEvent) => {
    let frame: CliResponsePayload & { type?: string };
    try {
      frame = JSON.parse(messageEvent.data as string);
    } catch {
      return;
    }
    if (frame.type === 'cli_status' || frame.type === 'arena_ack') {
      const cliConnected = (frame as { cli_connected?: boolean }).cli_connected === true;
      setState((prev) => ({ ...prev, cliConnected }));
      return;
    }
    if (frame.type !== 'arena_event' || !frame.request_id) return;
    const verified = await verifyCliResponse(frame.request_id, frame, ARENA_RELAY_CHAIN_ID, pairingId, {
      pinnedPublicKey: pinnedCliKey(pairingId),
    });
    if (!verified.ok || !verified.publicKey) {
      console.warn('[arena relay] dropped unverifiable event:', verified.error);
      return;
    }
    pinCliKey(pairingId, verified.publicKey);
    const data = decodeJSON(verified.payload) as { type?: string } | undefined;
    const event: ArenaEvent = {
      type: (data && typeof data === 'object' && typeof data.type === 'string' ? data.type : frame.purpose) ?? 'arena.event',
      data,
      timestamp: Date.now(),
      pairingId,
      requestId: frame.request_id,
    };
    setState((prev) => ({ ...prev, events: [...prev.events.slice(-99), event] }));
    onEvent?.(event);
  }, [pairingId, onEvent]);

  const scheduleReconnect = useRef<() => void>(() => undefined);

  const connect = useCallback(async () => {
    if (!pairingId || wsRef.current) return;
    closedByUserRef.current = false;
    setState((prev) => ({ ...prev, connecting: true, status: 'connecting', lastError: null }));
    let ticket: string;
    try {
      const response = await fetch(`${getKnirvServerUrl()}/api/v1/cli-supervisor/sessions/${encodeURIComponent(pairingId)}/arena-ticket`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${authToken()}` },
      });
      if (response.status === 401 || response.status === 403 || response.status === 404 || response.status === 410) {
        // Not retryable: signed out, wrong account, or the pairing is gone.
        const error = new ArenaRelayError((await response.text()).trim() || 'This arena pairing is not available', response.status);
        setState((prev) => ({ ...prev, connecting: false, status: 'error', lastError: error.message }));
        onError?.(error);
        return;
      }
      if (!response.ok) throw new Error(`arena ticket failed with HTTP ${response.status}`);
      ticket = ((await response.json()) as { ticket: string }).ticket;
    } catch (err) {
      setState((prev) => ({ ...prev, connecting: false, status: 'disconnected', lastError: (err as Error).message }));
      if (err instanceof ArenaRelayError) {
        onError?.(err);
        return;
      }
      scheduleReconnect.current();
      return;
    }
    if (closedByUserRef.current) return;
    const base = getKnirvServerUrl().replace(/^http/, 'ws');
    const ws = new WebSocket(`${base}/ws/cli-supervisor/${encodeURIComponent(pairingId)}/arena?ticket=${encodeURIComponent(ticket)}`);
    wsRef.current = ws;
    ws.onopen = () => {
      reconnectAttemptsRef.current = 0;
      setState((prev) => ({ ...prev, connected: true, connecting: false, status: 'connected', lastError: null }));
      onConnect?.();
    };
    ws.onmessage = (event) => { void handleWsMessage(event); };
    ws.onclose = (event) => {
      wsRef.current = null;
      setState((prev) => ({ ...prev, connected: false, connecting: false, status: 'disconnected' }));
      onDisconnect?.(event.code, event.reason);
      if (!closedByUserRef.current) scheduleReconnect.current();
    };
    ws.onerror = () => setState((prev) => ({ ...prev, lastError: 'Relay connection error' }));
  }, [pairingId, handleWsMessage, onConnect, onDisconnect, onError]);

  scheduleReconnect.current = () => {
    if (reconnectAttemptsRef.current < WS_MAX_RECONNECT_ATTEMPTS) {
      const delay = WS_RECONNECT_BASE_DELAY_MS * 2 ** reconnectAttemptsRef.current;
      reconnectAttemptsRef.current += 1;
      reconnectTimeoutRef.current = setTimeout(() => { void connect(); }, delay);
    } else {
      const error = new Error('Could not keep a connection to your CLI; check that it is paired and running');
      setState((prev) => ({ ...prev, status: 'error', lastError: error.message }));
      onError?.(error);
    }
  };

  const disconnect = useCallback(() => {
    closedByUserRef.current = true;
    if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
    reconnectTimeoutRef.current = null;
    wsRef.current?.close(1000, 'Client disconnect');
    wsRef.current = null;
    setState((prev) => ({ ...prev, connected: false, connecting: false, status: 'disconnected' }));
  }, []);

  useEffect(() => {
    if (!pairingId) return undefined;
    void connect();
    return () => disconnect();
  }, [pairingId, connect, disconnect]);

  useEffect(() => () => arenaRelayWorkerClient.terminate(), []);

  const encode = (value: unknown) => new TextEncoder().encode(JSON.stringify(value));

  return {
    ...state,
    connect,
    disconnect,
    sendRequest,
    invokeChat: (text: string) => sendRequest(RELAY_CAPABILITY_CHAT_INVOKE, encode({ text })),
    requestArenaStatus: () => sendRequest(RELAY_CAPABILITY_ARENA_STATUS, encode({})),
    deployAgent: (errorNodeId: string, agent: string) => sendRequest(RELAY_CAPABILITY_ARENA_AGENT_DEPLOY, encode({ error_node_id: errorNodeId, agent })),
    recallAgent: (agentId: string) => sendRequest(RELAY_CAPABILITY_ARENA_AGENT_RECALL, encode({ agent_id: agentId })),
  };
}
