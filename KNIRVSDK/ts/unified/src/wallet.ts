/**
 * Wallet compatibility layer for KNIRV SDK
 * 
 * This module provides wallet functionality and maintains compatibility
 * with existing wallet implementations while integrating KNIRV services.
 */

// Re-export essential wallet types and classes for compatibility
// These maintain compatibility with existing @adena-wallet/sdk usage

export interface WalletResponse<T = any> {
  code: number;
  status: WalletResponseStatus;
  type: WalletResponseType;
  message?: string;
  data?: T;
}

export enum WalletResponseStatus {
  SUCCESS = 'success',
  FAILURE = 'failure',
  REJECT = 'reject',
}

export enum WalletResponseType {
  ESTABLISH = 'establish',
  ACCOUNT = 'account',
  NETWORK = 'network',
  SIGN = 'sign',
  TRANSACTION = 'transaction',
}

export enum WalletResponseExecuteType {
  ADD_ESTABLISH = 'ADD_ESTABLISH',
  GET_ACCOUNT = 'GET_ACCOUNT',
  ADD_NETWORK = 'ADD_NETWORK',
  SWITCH_NETWORK = 'SWITCH_NETWORK',
  DO_CONTRACT = 'DO_CONTRACT',
  SIGN_TX = 'SIGN_TX',
}

export enum WalletResponseFailureType {
  NETWORK_TIMEOUT = 'NETWORK_TIMEOUT',
  UNAPPROVED_CHAIN = 'UNAPPROVED_CHAIN',
  UNAPPROVED_HOST = 'UNAPPROVED_HOST',
  LOCKED_ACCOUNT = 'LOCKED_ACCOUNT',
  INVALID_FORMAT = 'INVALID_FORMAT',
  INVALID_TRANSACTION = 'INVALID_TRANSACTION',
  UNEXPECTED_ERROR = 'UNEXPECTED_ERROR',
}

export enum WalletResponseRejectType {
  ESTABLISH_REJECTED = 'ESTABLISH_REJECTED',
  SIGN_REJECTED = 'SIGN_REJECTED',
  TRANSACTION_REJECTED = 'TRANSACTION_REJECTED',
}

export enum WalletResponseSuccessType {
  ESTABLISH_SUCCESS = 'ESTABLISH_SUCCESS',
  SIGN_SUCCESS = 'SIGN_SUCCESS',
  TRANSACTION_SUCCESS = 'TRANSACTION_SUCCESS',
}

// Wallet interface for KNIRV integration
export interface KNIRVWallet {
  // Account management
  getAccount(): Promise<WalletResponse>;
  
  // Network management
  addNetwork(params: any): Promise<WalletResponse>;
  switchNetwork(chainId: string): Promise<WalletResponse>;
  
  // Transaction signing
  signTransaction(transaction: any): Promise<WalletResponse>;
  
  // Contract interaction
  doContract(params: any): Promise<WalletResponse>;
  
  // KNIRV-specific functionality
  invokeSkill?(params: SkillInvocationParams): Promise<WalletResponse>;
  resolveKnirvURI?(uri: string): Promise<WalletResponse>;
  broadcastToNetwork?(data: any): Promise<WalletResponse>;
}

export interface SkillInvocationParams {
  user_id: string;
  skill_id: string;
  amount: string;
  metadata?: Record<string, any>;
}

// Legacy AdenaWallet class for backward compatibility
export class AdenaWallet implements KNIRVWallet {
  constructor(private config?: any) {}
  
  async getAccount(): Promise<WalletResponse> {
    // Implementation would integrate with KNIRV transaction API
    throw new Error('Not implemented - integrate with KNIRV transaction API');
  }
  
  async addNetwork(params: any): Promise<WalletResponse> {
    // Implementation would integrate with KNIRV gateway API
    throw new Error('Not implemented - integrate with KNIRV gateway API');
  }
  
  async switchNetwork(chainId: string): Promise<WalletResponse> {
    // Implementation would integrate with KNIRV gateway API
    throw new Error('Not implemented - integrate with KNIRV gateway API');
  }
  
  async signTransaction(transaction: any): Promise<WalletResponse> {
    // Implementation would integrate with KNIRV transaction API
    throw new Error('Not implemented - integrate with KNIRV transaction API');
  }
  
  async doContract(params: any): Promise<WalletResponse> {
    // Implementation would integrate with KNIRV transaction API
    throw new Error('Not implemented - integrate with KNIRV transaction API');
  }
  
  async invokeSkill(params: SkillInvocationParams): Promise<WalletResponse> {
    // Implementation would integrate with KNIRV gateway economics API
    throw new Error('Not implemented - integrate with KNIRV gateway economics API');
  }
  
  async resolveKnirvURI(uri: string): Promise<WalletResponse> {
    // Implementation would integrate with KNIRV transmission API
    throw new Error('Not implemented - integrate with KNIRV transmission API');
  }
  
  async broadcastToNetwork(data: any): Promise<WalletResponse> {
    // Implementation would integrate with KNIRV transmission API
    throw new Error('Not implemented - integrate with KNIRV transmission API');
  }
}

// Re-export blockchain functionality from @gnolang/tm2-js-client
export {
  generateHDPath,
  Provider,
  TransactionEndpoint,
  Tx,
  Wallet as Tm2Wallet,
  BroadcastTxCommitResult,
  BroadcastTxSyncResult,
  TxSignature,
} from '@gnolang/tm2-js-client';

// Re-export ledger functionality
export { LedgerConnector } from '@cosmjs/ledger-amino';
