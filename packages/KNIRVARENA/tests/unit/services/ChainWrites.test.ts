import '../../utils/realWebAssembly';
import { ACTION_SCHEMA_VERSION as SDK_ACTION_SCHEMA_VERSION } from '@knirv/sdk/signing';
import { submitChainWrite, ACTION_SCHEMA_VERSION, type ChainWrite, type WireTransaction } from '../../../src/services/ChainWrites';

const GATEWAY = 'https://testnet-gateway.knirv.com';
const SENDER = 'knirv1sender';
const REQUEST_ID = 'req-123';

const wire: WireTransaction = {
  body_bytes: 'Ym9keQ==',
  auth_info_bytes: 'YXV0aA==',
  signatures: ['c2ln'],
  public_key: 'cHVi',
  address: SENDER,
  hash: 'deadbeef',
};

const json = (status: number, body: unknown): Response =>
  ({ ok: status >= 200 && status < 300, status, json: async () => body, text: async () => JSON.stringify(body) }) as unknown as Response;

type Route = (url: string, init: RequestInit | undefined) => Response;

/** Fake gateway: each route is matched by URL substring and records what it was sent. */
function gateway(routes: Record<string, Route>) {
  const calls: { url: string; init: RequestInit | undefined }[] = [];
  const fetchImpl = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    calls.push({ url, init });
    const match = Object.keys(routes).find((key) => url.includes(key));
    if (!match) return json(404, { error: `no route for ${url}` });
    return routes[match](url, init);
  }) as unknown as typeof fetch;
  return { fetchImpl, calls };
}

const accountRoute: Route = () => json(200, {
  address: SENDER, chain_id: 'knirv-1', account_number: 42, sequence: 3, pending: 0, next_sequence: 3, sequence_enforced: true,
});

const writeRequest: ChainWrite = {
  type: 'register_capability_txn',
  sender: SENDER,
  data: new TextEncoder().encode('{"capabilityID":"skill-1"}'),
  fee: 2,
};

describe('submitChainWrite', () => {
  it('keeps the action schema version equal to the SDK', () => {
    expect(ACTION_SCHEMA_VERSION).toBe(SDK_ACTION_SCHEMA_VERSION);
  });

  it('signs through the Controller broker and posts the signed transaction to the chain', async () => {
    const shown: string[] = [];
    const { fetchImpl, calls } = gateway({
      '/api/chain/accounts/info': accountRoute,
      '/api/controller/signing/requests/': () => json(200, { status: 'approved', result: wire }),
      '/api/controller/signing/requests': () => json(201, { request_id: REQUEST_ID, approval_uri: 'knirv://sign?x=1' }),
      '/api/chain/transaction': () => json(201, { status: 'success', tx_hash: 'deadbeef' }),
    });

    const hash = await submitChainWrite(
      { ...writeRequest, onApprovalUri: (uri) => { shown.push(uri); return () => shown.push('closed'); } },
      { gatewayUrl: GATEWAY, fetchImpl, pollIntervalMs: 0 },
    );

    expect(hash).toBe('deadbeef');
    expect(shown).toEqual(['knirv://sign?x=1', 'closed']);

    const signingCall = calls.find((c) => c.url.endsWith('/api/controller/signing/requests'));
    const body = JSON.parse(String(signingCall?.init?.body));
    expect(body.kind).toBe('transaction');
    expect(body.chain_id).toBe('knirv-1');
    expect(body.transaction.chainId).toBe('knirv-1');
    expect(body.transaction.accountNumber).toBe(42);
    expect(body.transaction.sequence).toBe(3);
    expect(body.transaction.action).toEqual(expect.objectContaining({
      schemaVersion: 'knirv.action.v1',
      action: 'register_capability_txn',
      sender: SENDER,
      recipient: '',
      amount: 0,
    }));
    expect(Buffer.from(body.transaction.action.payload, 'base64').toString()).toBe('{"capabilityID":"skill-1"}');
    expect(body.transaction.fee).toEqual(expect.objectContaining({ denom: 'NRN', amount: '2' }));

    const submitted = JSON.parse(String(calls.find((c) => c.url.endsWith('/api/chain/transaction'))?.init?.body));
    expect(submitted).toEqual(expect.objectContaining({
      from: SENDER,
      transaction_hash: 'deadbeef',
      public_key: 'cHVi',
      signature: 'c2ln',
      body_bytes: 'Ym9keQ==',
      auth_info_bytes: 'YXV0aA==',
      chain_id: 'knirv-1',
      account_number: 42,
      sequence: 3,
      type: 'register_capability_txn',
      fee: 2,
    }));
  });

  it('fails when the Controller rejects the request', async () => {
    const { fetchImpl, calls } = gateway({
      '/api/chain/accounts/info': accountRoute,
      '/api/controller/signing/requests/': () => json(200, { status: 'rejected', reason: 'user declined' }),
      '/api/controller/signing/requests': () => json(201, { request_id: REQUEST_ID, approval_uri: 'knirv://sign' }),
    });

    await expect(submitChainWrite(writeRequest, { gatewayUrl: GATEWAY, fetchImpl, pollIntervalMs: 0 })).rejects.toThrow('user declined');
    expect(calls.some((c) => c.url.endsWith('/api/chain/transaction'))).toBe(false);
  });

  it('refuses a signed transaction from a different address', async () => {
    const { fetchImpl, calls } = gateway({
      '/api/chain/accounts/info': accountRoute,
      '/api/controller/signing/requests/': () => json(200, { status: 'approved', result: { ...wire, address: 'knirv1other' } }),
      '/api/controller/signing/requests': () => json(201, { request_id: REQUEST_ID, approval_uri: 'knirv://sign' }),
    });

    await expect(submitChainWrite(writeRequest, { gatewayUrl: GATEWAY, fetchImpl, pollIntervalMs: 0 })).rejects.toThrow('not from the requested sender');
    expect(calls.some((c) => c.url.endsWith('/api/chain/transaction'))).toBe(false);
  });

  it('surfaces the chain rejection, such as a stale sequence', async () => {
    const { fetchImpl } = gateway({
      '/api/chain/accounts/info': accountRoute,
      '/api/controller/signing/requests/': () => json(200, { status: 'approved', result: wire }),
      '/api/controller/signing/requests': () => json(201, { request_id: REQUEST_ID, approval_uri: 'knirv://sign' }),
      '/api/chain/transaction': () => json(409, 'sequence 3 does not match the next sequence 4'),
    });

    await expect(submitChainWrite(writeRequest, { gatewayUrl: GATEWAY, fetchImpl, pollIntervalMs: 0 })).rejects.toThrow(
      'KNIRV Chain rejected the transaction: HTTP 409',
    );
  });

  it('fails when the account state cannot be read', async () => {
    const { fetchImpl } = gateway({ '/api/chain/accounts/info': () => json(503, {}) });
    await expect(submitChainWrite(writeRequest, { gatewayUrl: GATEWAY, fetchImpl, pollIntervalMs: 0 })).rejects.toThrow('account lookup failed: HTTP 503');
  });
});
