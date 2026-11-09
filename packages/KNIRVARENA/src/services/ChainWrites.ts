import { getKnirvGatewayUrl } from '../config/runtimeConfig';

/**
 * The SDK's action schema version (`ACTION_SCHEMA_VERSION` in @knirv/sdk/signing).
 * Kept as a literal so importing this module does not load the SDK's wasm core;
 * the test suite asserts it still matches the SDK.
 */
export const ACTION_SCHEMA_VERSION = 'knirv.action.v1';

/**
 * Signed KNIRV Chain writes.
 *
 * A write is a transaction the user's Controller signs. The Arena:
 *   1. reads the sender's account number and next sequence from the chain;
 *   2. asks the Controller signing broker to sign a `transaction` request (the
 *      Controller holds the key and signs with the KNIRV SDK, canonical secp256k1);
 *   3. waits for the approved, signed transaction;
 *   4. posts it to the chain's `/transaction` route through the gateway.
 *
 * The chain verifies the signature, the account number, the sequence, the
 * sender's funds and the registration content, and includes the transaction only
 * when all of them hold.
 */

export interface AccountInfo {
  address: string;
  chain_id: string;
  account_number: number;
  sequence: number;
  pending: number;
  next_sequence: number;
  sequence_enforced: boolean;
}

export interface ChainWrite {
  /** Chain transaction type. It is the signed action, e.g. `register_capability_txn`. */
  type: string;
  /** The signer's KNIRV address. */
  sender: string;
  recipient?: string;
  /** NRN transferred with the transaction. */
  value?: number;
  /** NRN network fee the sender pays. */
  fee?: number;
  /** Denomination the fee is stated in. */
  feeDenom?: string;
  /** Canonical payload: signed as the action payload and stored as the transaction data. */
  data: Uint8Array;
  /** Shown to the user to approve in the Controller. Returns a closer for the prompt. */
  onApprovalUri?: (uri: string) => (() => void) | void;
}

export interface ChainWriteOptions {
  gatewayUrl?: string;
  fetchImpl?: typeof fetch;
  pollIntervalMs?: number;
  maxPolls?: number;
}

/** The SDK's signed transaction, as the signing broker returns it. */
export interface WireTransaction {
  body_bytes: string;
  auth_info_bytes: string;
  signatures: string[];
  public_key: string;
  address: string;
  hash: string;
}

/** Denomination of the fee field. Set per network; the signer states it and the chain binds it. */
export const DEFAULT_FEE_DENOM = 'NRN';

export const toBase64 = (bytes: Uint8Array): string => {
  let binary = '';
  for (let i = 0; i < bytes.length; i += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  }
  return btoa(binary);
};

const gatewayOf = (options: ChainWriteOptions): string =>
  (options.gatewayUrl ?? getKnirvGatewayUrl()).replace(/\/$/, '');

/** The chain's account state for an address, through the gateway. */
export async function getAccountInfo(address: string, options: ChainWriteOptions = {}): Promise<AccountInfo> {
  const fetchImpl = options.fetchImpl ?? fetch;
  const response = await fetchImpl(`${gatewayOf(options)}/api/chain/accounts/info?address=${encodeURIComponent(address)}`, {
    headers: { Accept: 'application/json' },
  });
  if (!response.ok) {
    throw new Error(`KNIRV Chain account lookup failed: HTTP ${response.status}`);
  }
  return (await response.json()) as AccountInfo;
}

async function awaitApproval(
  requestId: string,
  options: ChainWriteOptions,
): Promise<WireTransaction> {
  const fetchImpl = options.fetchImpl ?? fetch;
  const interval = options.pollIntervalMs ?? 2000;
  const maxPolls = options.maxPolls ?? 150;
  for (let i = 0; i < maxPolls; i++) {
    await new Promise((resolve) => setTimeout(resolve, interval));
    const response = await fetchImpl(
      `${gatewayOf(options)}/api/controller/signing/requests/${encodeURIComponent(requestId)}`,
      { cache: 'no-store' } as RequestInit,
    );
    const status = (await response.json()) as { status: string; result?: WireTransaction; reason?: string };
    if (status.status === 'approved' && status.result) {
      return status.result;
    }
    if (status.status === 'rejected' || status.status === 'expired') {
      throw new Error(status.reason || `Controller signing request ${status.status}`);
    }
  }
  throw new Error('Controller signing approval timed out');
}

/** The chain's transaction JSON for a signed wire transaction. */
export function toChainTransaction(
  wire: WireTransaction,
  write: ChainWrite,
  account: AccountInfo,
  timestampUnix: number,
): Record<string, unknown> {
  if (wire.address !== write.sender) {
    throw new Error('the signed transaction is not from the requested sender');
  }
  return {
    from: wire.address,
    to: write.recipient ?? '',
    value: write.value ?? 0,
    data: toBase64(write.data),
    status: '',
    timestamp: timestampUnix,
    transaction_hash: wire.hash,
    public_key: wire.public_key,
    signature: wire.signatures[0],
    body_bytes: wire.body_bytes,
    auth_info_bytes: wire.auth_info_bytes,
    chain_id: account.chain_id,
    account_number: account.account_number,
    sequence: account.next_sequence,
    fee: write.fee ?? 0,
    fee_denom: write.feeDenom ?? DEFAULT_FEE_DENOM,
    type: write.type,
  };
}

/**
 * Signs and submits one chain transaction. Resolves with the transaction hash the
 * chain accepted into its pool.
 */
export async function submitChainWrite(write: ChainWrite, options: ChainWriteOptions = {}): Promise<string> {
  const fetchImpl = options.fetchImpl ?? fetch;
  const gateway = gatewayOf(options);
  const account = await getAccountInfo(write.sender, options);
  const timestampUnix = Math.floor(Date.now() / 1000);
  const fee = write.fee ?? 0;

  const created = await fetchImpl(`${gateway}/api/controller/signing/requests`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      kind: 'transaction',
      chain_id: account.chain_id,
      transaction: {
        action: {
          schemaVersion: ACTION_SCHEMA_VERSION,
          action: write.type,
          sender: write.sender,
          recipient: write.recipient ?? '',
          amount: write.value ?? 0,
          payload: toBase64(write.data),
          timestampUnix,
        },
        chainId: account.chain_id,
        accountNumber: account.account_number,
        sequence: account.next_sequence,
        fee: { denom: write.feeDenom ?? DEFAULT_FEE_DENOM, amount: String(fee), gasLimit: 0, payer: '', granter: '' },
      },
    }),
  });
  const createdBody = (await created.json()) as { request_id?: string; approval_uri?: string; error?: string };
  if (!created.ok || !createdBody.request_id) {
    throw new Error(createdBody.error || 'Unable to create a Controller signing request');
  }

  const close = write.onApprovalUri?.(createdBody.approval_uri ?? '');
  let wire: WireTransaction;
  try {
    wire = await awaitApproval(createdBody.request_id, options);
  } finally {
    if (typeof close === 'function') close();
  }

  const transaction = toChainTransaction(wire, write, account, timestampUnix);
  const submitted = await fetchImpl(`${gateway}/api/chain/transaction`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(transaction),
  });
  if (!submitted.ok) {
    throw new Error(`KNIRV Chain rejected the transaction: HTTP ${submitted.status} ${(await submitted.text()).trim()}`);
  }
  const accepted = (await submitted.json()) as { tx_hash?: string };
  return accepted.tx_hash ?? wire.hash;
}
