// Payment system types and interfaces
export interface PaymentRequest {
  id: string;
  type: 'skill_invocation' | 'agent_deployment' | 'data_access' | 'compute_time';
  amount: number;
  currency: 'NRN' | 'XION' | 'USD';
  recipient: string;
  sender?: string;
  description: string;
  metadata?: Record<string, unknown>;
  timestamp: Date;
  expiresAt?: Date;
  priority?: 'low' | 'medium' | 'high' | 'urgent';
}

export interface PaymentResult {
  id: string;
  requestId: string;
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled';
  transactionHash?: string;
  blockHeight?: number;
  gasUsed?: number;
  gasPrice?: string;
  fee?: number;
  actualAmount?: number;
  timestamp: Date;
  completedAt?: Date;
  error?: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
  receipt?: {
    id: string;
    amount: number;
    currency: string;
    sender: string;
    recipient: string;
    timestamp: Date;
    signature?: string;
  };
}

export interface PaymentMethod {
  id: string;
  type: 'wallet' | 'credit_card' | 'bank_transfer' | 'crypto';
  name: string;
  isDefault: boolean;
  isActive: boolean;
  metadata: Record<string, unknown>;
  lastUsed?: Date;
  createdAt: Date;
}

export interface PaymentHistory {
  id: string;
  type: 'incoming' | 'outgoing';
  amount: number;
  currency: string;
  counterparty: string;
  description: string;
  status: 'completed' | 'pending' | 'failed';
  timestamp: Date;
  transactionHash?: string;
  fee?: number;
  metadata?: Record<string, unknown>;
}

export interface PaymentConfig {
  defaultCurrency: 'NRN' | 'XION' | 'USD';
  autoApproveLimit: number;
  requireConfirmation: boolean;
  enableNotifications: boolean;
  gasSettings: {
    gasPrice: string;
    gasLimit: number;
    maxFeePerGas?: string;
    maxPriorityFeePerGas?: string;
  };
  retrySettings: {
    maxRetries: number;
    retryDelay: number;
    backoffMultiplier: number;
  };
}

export interface PaymentProvider {
  id: string;
  name: string;
  type: 'blockchain' | 'traditional' | 'hybrid';
  supportedCurrencies: string[];
  fees: {
    fixed?: number;
    percentage?: number;
    minimum?: number;
    maximum?: number;
  };
  limits: {
    daily?: number;
    monthly?: number;
    perTransaction?: number;
  };
  isActive: boolean;
  config: Record<string, unknown>;
}

export interface QRPaymentData {
  type: 'payment' | 'skill_invocation' | 'wallet_connect' | 'agent_deploy';
  amount?: number;
  currency?: string;
  recipient?: string;
  description?: string;
  metadata?: Record<string, unknown>;
  expiresAt?: Date;
  signature?: string;
}

export interface PaymentNotification {
  id: string;
  type: 'payment_received' | 'payment_sent' | 'payment_failed' | 'payment_pending';
  title: string;
  message: string;
  amount?: number;
  currency?: string;
  timestamp: Date;
  isRead: boolean;
  metadata?: Record<string, unknown>;
}
