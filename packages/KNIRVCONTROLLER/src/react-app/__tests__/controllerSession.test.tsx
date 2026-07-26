import { describe, expect, it } from 'vitest';
import {
  controllerPairingClaim,
  parseDVESessionPairing,
} from '@/react-app/platform/controllerSession';

describe('controller session pairing', () => {
  it('parses the CLI QR payload', () => {
    const payload = parseDVESessionPairing(JSON.stringify({
      version: '1.0',
      type: 'dve_session_pairing',
      session_id: 'sess-1',
      environment_id: 'env-1',
      user_id: 'user-1',
      pairing_token: 'token-1',
      code: 'ABC12345',
      expires_at: new Date(Date.now() + 60_000).toISOString(),
      capabilities: ['chat.read', 'chat.write'],
    }));
    expect(payload.session_id).toBe('sess-1');
    expect(payload.capabilities).toEqual(['chat.read', 'chat.write']);
  });

  it('rejects expired pairing payloads', () => {
    expect(() => parseDVESessionPairing(JSON.stringify({
      version: '1.0',
      type: 'dve_session_pairing',
      session_id: 'sess-1',
      environment_id: 'env-1',
      pairing_token: 'token-1',
      expires_at: new Date(Date.now() - 1_000).toISOString(),
    }))).toThrow('expired');
  });

  it('uses the backend canonical vault claim', () => {
    expect(controllerPairingClaim(' sess-1 ', ' token-1 ', ' phone-1 ')).toBe(
      'knirv:dve-session-pair:v1\n' +
      'session_id=sess-1\n' +
      'pairing_token=token-1\n' +
      'device_id=phone-1',
    );
  });
});
