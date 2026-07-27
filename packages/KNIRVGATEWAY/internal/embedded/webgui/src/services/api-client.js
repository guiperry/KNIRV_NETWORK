const DEFAULT_TIMEOUT_MS = 10_000;

export class ApiError extends Error {
  constructor(message, { status, url, cause } = {}) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.url = url;
    this.cause = cause;
  }
}

export function collectionFrom(payload, keys = []) {
  if (Array.isArray(payload)) return payload;
  if (!payload || typeof payload !== 'object') return [];

  for (const key of keys) {
    if (Array.isArray(payload[key])) return payload[key];
  }
  return [];
}

export async function requestJSON(path, options = {}) {
  const { timeoutMs = DEFAULT_TIMEOUT_MS, ...requestOptions } = options;
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(path, {
      ...requestOptions,
      headers: {
        Accept: 'application/json',
        ...requestOptions.headers,
      },
      signal: controller.signal,
    });

    if (!response.ok) {
      const detail = await response.text().catch(() => '');
      throw new ApiError(
        `Request failed (${response.status}${detail ? `: ${detail}` : ''})`,
        { status: response.status, url: path },
      );
    }

    return await response.json();
  } catch (error) {
    if (error instanceof ApiError) throw error;
    if (error?.name === 'AbortError') {
      throw new ApiError(`Request timed out after ${timeoutMs}ms`, { url: path, cause: error });
    }
    throw new ApiError(error?.message || 'Network request failed', { url: path, cause: error });
  } finally {
    clearTimeout(timeout);
  }
}

export const oracleApi = {
  getHealth: () => requestJSON('/api/oracle/v3/health'),
  getBlocks: async () => collectionFrom(await requestJSON('/api/blocks'), ['blocks', 'block_metas']),
  getTransactions: async () => collectionFrom(await requestJSON('/api/transactions'), ['transactions', 'tx_responses']),
};

export const chainApi = {
  getNFTs: async () => collectionFrom(await requestJSON('/api/objects'), ['nfts', 'objects', 'assets']),
};

export const graphApi = {
  getSkills: async () => collectionFrom(await requestJSON('/api/graph/nrv/skills'), ['skills']),
  getErrors: async () => collectionFrom(await requestJSON('/api/graph/nrv/errors'), ['errors']),
};

// These services are exposed by backend_server over its /api/v1 proxy.  They
// are intentionally separate from the Cosmos-based Oracle endpoints above.
export const transactionChainApi = {
  getHealth: () => requestJSON('/api/v1/transaction-chain/health'),
  getChain: () => requestJSON('/api/v1/transaction-chain/chain'),
  getBlocks: async () => collectionFrom(await requestJSON('/api/v1/transaction-chain/blocks'), ['blocks']),
  getPendingTransactions: async () => collectionFrom(await requestJSON('/api/v1/transaction-chain/txn_pool'), ['transactions']),
};

export const validationChainApi = {
  getHealth: () => requestJSON('/api/v1/validation-chain/health'),
  getHeight: () => requestJSON('/api/v1/validation-chain/chain/height'),
  getBlocks: async () => collectionFrom(await requestJSON('/api/v1/validation-chain/blocks'), ['blocks']),
};
