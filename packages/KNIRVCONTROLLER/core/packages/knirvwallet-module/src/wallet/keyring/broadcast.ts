import type {
  BroadcastTxCommitResult,
  BroadcastTxSyncResult,
  Provider,
  Tx,
} from '../wallet';

function endpoint(provider: Provider): string {
  if (!provider?.rpcUrl) throw new Error('KNIRV gateway URL is required');
  return `${provider.rpcUrl.replace(/\/$/, '')}/api/transmission/broadcast`;
}

async function submit(provider: Provider, signedTx: Tx, mode: 'sync' | 'commit'): Promise<any> {
  const response = await fetch(endpoint(provider), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ chain_id: provider.chainId, mode, transaction: signedTx }),
  });
  const text = await response.text();
  let result: any = {};
  if (text) {
    try { result = JSON.parse(text); } catch { result = { log: text }; }
  }
  if (!response.ok) {
    throw new Error(result.error || result.message || result.log || `Broadcast failed with HTTP ${response.status}`);
  }
  const data = result.data ?? result.result ?? result;
  if (!data.hash && !data.tx_hash && !data.txHash) {
    throw new Error('Broadcast response did not include a transaction hash');
  }
  return data;
}

export async function broadcastSync(provider: Provider, signedTx: Tx): Promise<BroadcastTxSyncResult> {
  const data = await submit(provider, signedTx, 'sync');
  return {
    hash: data.hash ?? data.tx_hash ?? data.txHash,
    code: Number(data.code ?? 0),
    log: String(data.log ?? data.raw_log ?? ''),
  };
}

export async function broadcastCommit(provider: Provider, signedTx: Tx): Promise<BroadcastTxCommitResult> {
  const data = await submit(provider, signedTx, 'commit');
  return {
    hash: data.hash ?? data.tx_hash ?? data.txHash,
    height: Number(data.height ?? 0),
    code: Number(data.code ?? 0),
    log: String(data.log ?? data.raw_log ?? ''),
    gasUsed: Number(data.gasUsed ?? data.gas_used ?? 0),
    gasWanted: Number(data.gasWanted ?? data.gas_wanted ?? 0),
  };
}
