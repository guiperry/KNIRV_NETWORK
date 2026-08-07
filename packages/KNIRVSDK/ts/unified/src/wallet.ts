/** Controller-custodied wallet client. Private user keys never enter the SDK process. */
import type { DirectSignRequest, KNIRVFee } from './signing';

export interface WalletResponse<T = unknown> {
  code: number;
  status: WalletResponseStatus;
  type: WalletResponseType;
  message?: string;
  data?: T;
}

export enum WalletResponseStatus { SUCCESS = 'success', FAILURE = 'failure', REJECT = 'reject' }
export enum WalletResponseType { ESTABLISH = 'establish', ACCOUNT = 'account', NETWORK = 'network', SIGN = 'sign', TRANSACTION = 'transaction' }
export enum WalletResponseExecuteType { ADD_ESTABLISH = 'ADD_ESTABLISH', GET_ACCOUNT = 'GET_ACCOUNT', ADD_NETWORK = 'ADD_NETWORK', SWITCH_NETWORK = 'SWITCH_NETWORK', DO_CONTRACT = 'DO_CONTRACT', SIGN_TX = 'SIGN_TX' }
export enum WalletResponseFailureType { NETWORK_TIMEOUT = 'NETWORK_TIMEOUT', UNAPPROVED_CHAIN = 'UNAPPROVED_CHAIN', UNAPPROVED_HOST = 'UNAPPROVED_HOST', LOCKED_ACCOUNT = 'LOCKED_ACCOUNT', INVALID_FORMAT = 'INVALID_FORMAT', INVALID_TRANSACTION = 'INVALID_TRANSACTION', UNEXPECTED_ERROR = 'UNEXPECTED_ERROR' }
export enum WalletResponseRejectType { ESTABLISH_REJECTED = 'ESTABLISH_REJECTED', SIGN_REJECTED = 'SIGN_REJECTED', TRANSACTION_REJECTED = 'TRANSACTION_REJECTED' }
export enum WalletResponseSuccessType { ESTABLISH_SUCCESS = 'ESTABLISH_SUCCESS', SIGN_SUCCESS = 'SIGN_SUCCESS', TRANSACTION_SUCCESS = 'TRANSACTION_SUCCESS' }

export interface SkillInvocationParams {
  user_id: string;
  skill_id: string;
  sender: string;
  amount: string;
  account_number: string | number | bigint;
  sequence: string | number | bigint;
  fee?: KNIRVFee;
  metadata?: Record<string, unknown>;
}

export interface KNIRVWalletConfig {
  provider?: string;
  chainId?: string;
  enableGasless?: boolean;
  apiKey?: string;
  approvalTimeoutMs?: number;
}

const DEFAULT_GATEWAYS = [
  'https://gateway.knirv.network',
  'https://testnet-gateway.knirv.network',
  'http://localhost:8080',
];

interface ApprovalRequest {
  request_id: string;
  approval_uri: string;
  expires_at: string;
}

interface ApprovalStatus {
  status: 'pending' | 'approved' | 'rejected' | 'expired';
  result?: unknown;
  reason?: string;
}

function approvalJSON(value: unknown): string {
  return JSON.stringify(value, (_key, item) => {
    if (typeof item === 'bigint') return item.toString();
    if (!(item instanceof Uint8Array)) return item;
    let binary = '';
    for (let start = 0; start < item.length; start += 0x8000) {
      binary += String.fromCharCode(...item.subarray(start, start + 0x8000));
    }
    return btoa(binary);
  });
}

export class KNIRVWallet {
  private readonly gateways: string[];
  private readonly chainId: string;
  private readonly apiKey?: string;
  private readonly approvalTimeoutMs: number;

  constructor(config: KNIRVWalletConfig = {}) {
    this.gateways = config.provider
      ? [config.provider.replace(/\/$/, ''), ...DEFAULT_GATEWAYS.filter((url) => url !== config.provider)]
      : DEFAULT_GATEWAYS;
    this.chainId = config.chainId || 'knirv-1';
    this.apiKey = config.apiKey;
    this.approvalTimeoutMs = config.approvalTimeoutMs || 5 * 60_000;
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    let lastError: unknown;
    for (const gateway of this.gateways) {
      try {
        const response = await fetch(`${gateway}${path}`, {
          ...init,
          headers: {
            'Content-Type': 'application/json',
            ...(this.apiKey ? { Authorization: `Bearer ${this.apiKey}` } : {}),
            ...init.headers,
          },
        });
        if (response.status >= 500) throw new Error(`${gateway} returned HTTP ${response.status}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}: ${await response.text()}`);
        return await response.json() as T;
      } catch (error) {
        lastError = error;
      }
    }
    throw lastError instanceof Error ? lastError : new Error('No KNIRV gateway is available');
  }

  private async approvedOperation(path: string, payload: unknown, type: WalletResponseType): Promise<WalletResponse> {
    const approval = await this.request<ApprovalRequest>(path, { method: 'POST', body: approvalJSON(payload) });
    const deadline = Date.now() + this.approvalTimeoutMs;
    while (Date.now() < deadline) {
      const state = await this.request<ApprovalStatus>(`/api/controller/signing/requests/${encodeURIComponent(approval.request_id)}`);
      if (state.status === 'approved') return { code: 0, status: WalletResponseStatus.SUCCESS, type, data: state.result };
      if (state.status === 'rejected') return { code: 1, status: WalletResponseStatus.REJECT, type, message: state.reason || 'Rejected in KNIRVCONTROLLER' };
      if (state.status === 'expired') return { code: 1, status: WalletResponseStatus.FAILURE, type, message: 'KNIRVCONTROLLER approval expired' };
      await new Promise((resolve) => setTimeout(resolve, 1500));
    }
    return { code: 1, status: WalletResponseStatus.FAILURE, type, message: 'KNIRVCONTROLLER approval timed out' };
  }

  async getAccount(): Promise<WalletResponse> {
    const data = await this.request('/api/controller/account');
    return { code: 0, status: WalletResponseStatus.SUCCESS, type: WalletResponseType.ACCOUNT, data };
  }

  async addNetwork(params: unknown): Promise<WalletResponse> {
    return this.approvedOperation('/api/controller/networks', params, WalletResponseType.NETWORK);
  }

  async switchNetwork(chainId: string): Promise<WalletResponse> {
    return this.approvedOperation('/api/controller/networks/switch', { chain_id: chainId }, WalletResponseType.NETWORK);
  }

  async signTransaction(transaction: DirectSignRequest): Promise<WalletResponse> {
    return this.approvedOperation('/api/controller/signing/requests', { kind: 'transaction', chain_id: this.chainId, transaction }, WalletResponseType.SIGN);
  }

  async signMessage(envelope: unknown): Promise<WalletResponse> {
    return this.approvedOperation('/api/controller/signing/requests', { kind: 'message', chain_id: this.chainId, envelope }, WalletResponseType.SIGN);
  }

  async doContract(transaction: DirectSignRequest): Promise<WalletResponse> {
    return this.approvedOperation('/api/controller/signing/requests', { kind: 'transaction', chain_id: transaction.chainId, transaction }, WalletResponseType.TRANSACTION);
  }

  async invokeSkill(params: SkillInvocationParams): Promise<WalletResponse> {
    const transaction: DirectSignRequest = {
      action: {
        action: 'knirv.skill.invoke',
        sender: params.sender,
        recipient: params.skill_id,
        amount: params.amount,
        payload: new TextEncoder().encode(JSON.stringify({ user_id: params.user_id, metadata: params.metadata ?? {} })),
        timestampUnix: Math.floor(Date.now() / 1000),
      },
      chainId: this.chainId,
      accountNumber: params.account_number,
      sequence: params.sequence,
      fee: params.fee,
    };
    return this.approvedOperation('/api/controller/signing/requests', { kind: 'transaction', chain_id: this.chainId, transaction }, WalletResponseType.TRANSACTION);
  }

  async resolveKnirvURI(uri: string): Promise<WalletResponse> {
    const data = await this.request(`/api/transmission/resolve?uri=${encodeURIComponent(uri)}`);
    return { code: 0, status: WalletResponseStatus.SUCCESS, type: WalletResponseType.TRANSACTION, data };
  }

  async broadcastToNetwork(data: unknown): Promise<WalletResponse> {
    const result = await this.request('/api/transmission/broadcast', { method: 'POST', body: JSON.stringify(data) });
    return { code: 0, status: WalletResponseStatus.SUCCESS, type: WalletResponseType.TRANSACTION, data: result };
  }
}

export { KNIRVWallet as AdenaWallet };

export {
  generateHDPath, Provider, TransactionEndpoint, Tx, Wallet as Tm2Wallet,
  BroadcastTxCommitResult, BroadcastTxSyncResult, TxSignature,
} from '@gnolang/tm2-js-client';
export { LedgerConnector } from '@cosmjs/ledger-amino';
