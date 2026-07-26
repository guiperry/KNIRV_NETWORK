import { nativeStorage } from '@/react-app/platform/nativeStorage';

const CONTROLLER_DEVICE_KEY = 'knirv.controller.device-id';
const JOINED_SESSION_KEY = 'knirv.controller.joined-session';

export interface DVESessionPairingPayload {
  version: string;
  type: 'dve_session_pairing';
  session_id: string;
  environment_id: string;
  user_id: string;
  pairing_token: string;
  code: string;
  expires_at: string;
  capabilities: string[];
}

export interface JoinedControllerSession {
  session_id: string;
  environment_id: string;
  device_id: string;
  access_token: string;
  expires_at: string;
  trust_level: string;
}

export function parseDVESessionPairing(value: string): DVESessionPairingPayload {
  let decoded: unknown;
  try {
    decoded = JSON.parse(value);
  } catch {
    throw new Error('The QR code is not a KNIRV DVE session pairing.');
  }
  if (!decoded || typeof decoded !== 'object') {
    throw new Error('The QR code is not a KNIRV DVE session pairing.');
  }
  const payload = decoded as Partial<DVESessionPairingPayload>;
  if (
    payload.type !== 'dve_session_pairing' ||
    payload.version !== '1.0' ||
    !payload.session_id ||
    !payload.environment_id ||
    !payload.pairing_token ||
    !payload.expires_at
  ) {
    throw new Error('The DVE session pairing payload is incomplete.');
  }
  const expiresAt = Date.parse(payload.expires_at);
  if (!Number.isFinite(expiresAt) || expiresAt <= Date.now()) {
    throw new Error('The DVE session pairing has expired.');
  }
  return {
    version: payload.version,
    type: payload.type,
    session_id: payload.session_id,
    environment_id: payload.environment_id,
    user_id: payload.user_id || '',
    pairing_token: payload.pairing_token,
    code: payload.code || '',
    expires_at: payload.expires_at,
    capabilities: payload.capabilities || [],
  };
}

export function controllerPairingClaim(
  sessionID: string,
  pairingToken: string,
  deviceID: string,
): string {
  return (
    'knirv:dve-session-pair:v1\n' +
    `session_id=${sessionID.trim()}\n` +
    `pairing_token=${pairingToken.trim()}\n` +
    `device_id=${deviceID.trim()}`
  );
}

export async function getControllerDeviceID(): Promise<string> {
  const existing = await nativeStorage.get(CONTROLLER_DEVICE_KEY);
  if (existing) {
    return existing;
  }
  const id = `controller-${crypto.randomUUID()}`;
  await nativeStorage.set(CONTROLLER_DEVICE_KEY, id);
  return id;
}

export async function saveJoinedControllerSession(session: JoinedControllerSession): Promise<void> {
  await nativeStorage.set(JOINED_SESSION_KEY, JSON.stringify(session));
}

export async function getJoinedControllerSession(): Promise<JoinedControllerSession | null> {
  const stored = await nativeStorage.get(JOINED_SESSION_KEY);
  if (!stored) {
    return null;
  }
  try {
    const session = JSON.parse(stored) as JoinedControllerSession;
    const expiresAt = Date.parse(session.expires_at);
    if (!session.session_id || !session.access_token || !Number.isFinite(expiresAt) || expiresAt <= Date.now()) {
      await nativeStorage.remove(JOINED_SESSION_KEY);
      return null;
    }
    return session;
  } catch {
    await nativeStorage.remove(JOINED_SESSION_KEY);
    return null;
  }
}

export function bytesToBase64(value: Uint8Array): string {
  let binary = '';
  for (const byte of value) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
}

export function workerWebSocketURL(sessionID: string, backendWebSocketURL: string): string {
  const backend = new URL(backendWebSocketURL);
  const worker = new URL(
    `/worker/dve/sessions/${encodeURIComponent(sessionID)}/socket`,
    window.location.origin,
  );
  worker.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  worker.search = backend.search;
  return worker.toString();
}
