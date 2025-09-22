/**
 * Abstraxion Wallet Integration Service
 * Handles XION wallet connection and USDC-to-NRN conversions using Abstraxion Dave SDK
 * Implements Meta Accounts and Treasury Contracts for gasless transactions
 */

import { useEffect, useState } from "react";

// Types for the service
export interface XIONAccount {
  id: string;
  address: string;
  name: string;
  balance: string;
  usdcBalance: string;
  nrnBalance: string;
  isConnected: boolean;
  metaAccountType: 'email' | 'social' | 'wallet' | 'passkey';
  gasless: boolean;
}

export interface ConversionRequest {
  usdcAmount: string;
  nrnTargetAddress?: string;
  memo?: string;
  gasless?: boolean;
}

export interface ConversionResult {
  transactionId: string;
  usdcAmount: string;
  nrnAmount: string;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: Date;
  gasUsed?: string;
  fee?: string;
}

export interface TreasuryConfig {
  contractAddress: string;
  enabled: boolean;
  maxGasLimit: string;
  allowedOperations: string[];
}

export interface TransactionMessage {
  from: string;
  to: string;
  amount: string;
  memo?: string;
  gasLimit?: string;
  fee?: string;
}

export interface TransactionResult {
  transactionHash?: string;
  gasUsed?: string;
  fee?: string;
  success: boolean;
  error?: string;
}

export interface ContractMessage {
  contractAddress: string;
  msg: Record<string, unknown>;
  funds?: Array<{ denom: string; amount: string }>;
}

export interface ConnectOptions {
  authMethod?: 'email' | 'social' | 'wallet' | 'passkey';
  autoConnect?: boolean;
}

// Abstraxion Wallet Service Class
export class AbstraxionWalletService {
  private account: XIONAccount | null = null;
  private isInitialized = false;
  private treasuryConfig: TreasuryConfig;
  private abstraxionSDK: unknown = null; // Will be @burnt-labs/abstraxion-react-native

  // Configuration
  private config = {
    contracts: {
      tokenContract: "xion1usdc_contract_address", // USDC contract address on XION
      swapContract: "xion1nrn_contract_address", // NRN contract for conversion
      treasuryContract: "xion1treasury_address", // Treasury contract for gasless transactions
    },
    endpoints: {
      rpc: "https://rpc.xion-testnet-1.burnt.com:443",
      rest: "https://api.xion-testnet-1.burnt.com",
      knirvOracle: "http://localhost:8080", // KNIRVORACLE payment gateway endpoint
    },
    paymentGateway: {
      enabled: true,
      conversionRate: "10", // 1 USDC = 10 NRN
      maxTransactionAmount: "10000000000", // 10,000 USDC (6 decimals)
      minTransactionAmount: "1000000", // 1 USDC (6 decimals)
    }
  };

  constructor() {
    this.treasuryConfig = {
      contractAddress: 'xion1treasury...', // Treasury contract address
      enabled: true,
      maxGasLimit: '200000',
      allowedOperations: ['usdc_to_nrn', 'nrn_transfer', 'skill_invocation']
    };
    this.initialize();
  }

  private async initialize(): Promise<void> {
    if (this.isInitialized) return;

    try {
      // Initialize Abstraxion SDK with Dave Mobile Development Kit
      console.log('Initializing Abstraxion wallet service with Dave SDK...');

      // In a real implementation, this would:
      // 1. Import @burnt-labs/abstraxion-react-native
      // 2. Configure the SDK with proper chain settings
      // 3. Set up Meta Accounts with email/social/wallet/passkey auth
      // 4. Initialize Treasury Contracts for gasless transactions
      // 5. Set up event listeners for account changes

      // Mock SDK initialization
      this.abstraxionSDK = {
        connect: this.mockConnect.bind(this),
        disconnect: this.mockDisconnect.bind(this),
        signTransaction: this.mockSignTransaction.bind(this),
        executeContract: this.mockExecuteContract.bind(this)
      };

      this.isInitialized = true;
      console.log('Abstraxion wallet service initialized with Dave SDK');
    } catch (error) {
      console.error('Failed to initialize Abstraxion service:', error);
    }
  }

  // Connect to XION wallet using Meta Accounts
  async connectWallet(authMethod: 'email' | 'social' | 'wallet' | 'passkey' = 'email'): Promise<XIONAccount | null> {
    try {
      if (!this.isInitialized) {
        await this.initialize();
      }

      console.log(`Connecting to XION wallet using ${authMethod} authentication...`);

      // In a real implementation, this would call Abstraxion's connect method
      // with the specified authentication method
      const connectionResult = await (this.abstraxionSDK as any).connect({
        authMethod,
        enableGasless: this.treasuryConfig.enabled
      });

      // Log connection result for debugging
      console.log('Connection established:', connectionResult.success);

      // Simulate connection with Meta Account features
      this.account = {
        id: 'xion_meta_account_' + Date.now(),
        address: 'xion1abc123def456...',
        name: `XION Meta Account (${authMethod})`,
        balance: '1000',
        usdcBalance: '500',
        nrnBalance: '0',
        isConnected: true,
        metaAccountType: authMethod,
        gasless: this.treasuryConfig.enabled
      };

      console.log('Successfully connected to XION Meta Account');
      return this.account;
    } catch (error) {
      console.error('Failed to connect XION wallet:', error);
      throw error;
    }
  }

  // Disconnect wallet
  async disconnectWallet(): Promise<void> {
    if (this.abstraxionSDK) {
      await (this.abstraxionSDK as any).disconnect();
    }
    this.account = null;
    console.log('Disconnected from XION wallet');
  }

  // Get current account
  getCurrentAccount(): XIONAccount | null {
    return this.account;
  }

  // Get USDC balance
  async getUSDCBalance(): Promise<string> {
    if (!this.account) throw new Error('No wallet connected');

    try {
      // In a real implementation, this would query the USDC contract
      console.log('Fetching USDC balance...');
      return this.account.usdcBalance;
    } catch (error) {
      console.error('Failed to get USDC balance:', error);
      throw error;
    }
  }

  // Convert USDC to NRN tokens using KNIRVORACLE Payment Gateway
  async convertUSDCToNRN(request: ConversionRequest): Promise<ConversionResult> {
    if (!this.account) throw new Error('No wallet connected');

    try {
      console.log(`Converting ${request.usdcAmount} USDC to NRN via KNIRVORACLE Payment Gateway...`);

      // Use KNIRVORACLE payment gateway for conversion
      if (this.config.paymentGateway.enabled) {
        return await this.convertViaKNIRVORACLE(request);
      }

      // Fallback to direct XION transaction
      return await this.convertViaDirectTransaction(request);
    } catch (error) {
      console.error('USDC to NRN conversion failed:', error);
      throw error;
    }
  }

  // Convert via KNIRVORACLE Payment Gateway (preferred method)
  private async convertViaKNIRVORACLE(request: ConversionRequest): Promise<ConversionResult> {
    try {
      console.log('Using KNIRVORACLE Payment Gateway for conversion...');

      // Prepare payment request for KNIRVORACLE
      const paymentRequest = {
        user_address: this.account!.address,
        usdc_amount: request.usdcAmount,
        meta_account_type: this.account!.metaAccountType,
        gasless: request.gasless || this.treasuryConfig.enabled,
        memo: request.memo || 'USDC to NRN conversion via KNIRVCONTROLLER'
      };

      // Call KNIRVORACLE payment gateway API
      const response = await fetch(`${this.config.endpoints.knirvOracle}/api/payment/usdc-to-nrn`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(paymentRequest)
      });

      if (!response.ok) {
        throw new Error(`Payment gateway error: ${response.status} ${response.statusText}`);
      }

      const paymentResult = await response.json();

      if (!paymentResult.success) {
        throw new Error(`Payment failed: ${paymentResult.error}`);
      }

      const result: ConversionResult = {
        transactionId: paymentResult.data.payment_id,
        usdcAmount: paymentResult.data.usdc_amount,
        nrnAmount: paymentResult.data.nrn_amount,
        status: 'pending',
        timestamp: new Date(),
        gasUsed: '0', // Gasless transaction
        fee: '0' // No fee for gasless transactions
      };

      // Update account balances optimistically
      if (this.account) {
        this.account.usdcBalance = (parseFloat(this.account.usdcBalance) - parseFloat(request.usdcAmount)).toString();
        this.account.nrnBalance = (parseFloat(this.account.nrnBalance) + parseFloat(paymentResult.data.nrn_amount)).toString();
      }

      // Monitor payment status via KNIRVORACLE
      this.monitorKNIRVORACLEPayment(result.transactionId);

      console.log('KNIRVORACLE payment initiated successfully:', result.transactionId);
      return result;
    } catch (error) {
      console.error('KNIRVORACLE payment gateway error:', error);
      throw error;
    }
  }

  // Fallback: Convert via direct XION transaction
  private async convertViaDirectTransaction(request: ConversionRequest): Promise<ConversionResult> {
    try {
      console.log('Using direct XION transaction for conversion...');

      // Calculate conversion rate
      const conversionRate = parseFloat(this.config.paymentGateway.conversionRate);
      const nrnAmount = (parseFloat(request.usdcAmount) * conversionRate).toString();

      // Prepare transaction message
      const txMsg = {
        typeUrl: '/cosmwasm.wasm.v1.MsgExecuteContract',
        value: {
          sender: this.account!.address,
          contract: this.config.contracts.swapContract,
          msg: {
            swap_usdc_to_nrn: {
              usdc_amount: request.usdcAmount,
              min_nrn_amount: nrnAmount,
              recipient: request.nrnTargetAddress || this.account!.address
            }
          },
          funds: [{
            denom: 'ibc/USDC_DENOM', // USDC IBC denom on XION
            amount: request.usdcAmount
          }]
        }
      };

      // Execute transaction (gasless if treasury is enabled)
      let txResult;
      if (request.gasless && this.treasuryConfig.enabled) {
        console.log('Executing gasless transaction via Treasury Contract...');
        txResult = await this.executeGaslessTransaction(txMsg as any);
      } else {
        console.log('Executing standard transaction...');
        txResult = await (this.abstraxionSDK as any).signTransaction(txMsg);
      }

      const result: ConversionResult = {
        transactionId: txResult.transactionHash || 'tx_' + Date.now(),
        usdcAmount: request.usdcAmount,
        nrnAmount: nrnAmount,
        status: 'pending',
        timestamp: new Date(),
        gasUsed: txResult.gasUsed,
        fee: txResult.fee
      };

      // Update account balances
      if (this.account) {
        this.account.usdcBalance = (parseFloat(this.account.usdcBalance) - parseFloat(request.usdcAmount)).toString();
        this.account.nrnBalance = (parseFloat(this.account.nrnBalance) + parseFloat(nrnAmount)).toString();
      }

      // Monitor transaction status
      this.monitorTransactionStatus(result);

      return result;
    } catch (error) {
      console.error('Direct transaction conversion failed:', error);
      throw error;
    }
  }

  // Check transaction status (supports both KNIRVORACLE payments and direct transactions)
  async checkConversionStatus(transactionId: string): Promise<ConversionResult | null> {
    try {
      console.log('Checking conversion status for:', transactionId);

      // Check if this is a KNIRVORACLE payment (starts with 'pay_')
      if (transactionId.startsWith('pay_')) {
        return await this.checkKNIRVORACLEPaymentStatus(transactionId);
      }

      // Otherwise, check as direct blockchain transaction
      return await this.checkDirectTransactionStatus(transactionId);
    } catch (error) {
      console.error('Failed to check conversion status:', error);
      return null;
    }
  }

  // Check KNIRVORACLE payment status
  private async checkKNIRVORACLEPaymentStatus(paymentId: string): Promise<ConversionResult | null> {
    try {
      const response = await fetch(`${this.config.endpoints.knirvOracle}/api/payment/status/${paymentId}`);

      if (!response.ok) {
        throw new Error(`Failed to check payment status: ${response.status}`);
      }

      const statusResult = await response.json();

      if (!statusResult.success) {
        throw new Error(`Payment status check failed: ${statusResult.error}`);
      }

      const paymentData = statusResult.data;

      return {
        transactionId: paymentData.payment_id,
        usdcAmount: paymentData.usdc_amount,
        nrnAmount: paymentData.nrn_amount,
        status: this.mapKNIRVORACLEStatus(paymentData.status),
        timestamp: new Date(paymentData.created_at),
        gasUsed: '0', // Gasless
        fee: '0' // No fee
      };
    } catch (error) {
      console.error('Failed to check KNIRVORACLE payment status:', error);
      return null;
    }
  }

  // Check direct blockchain transaction status
  private async checkDirectTransactionStatus(transactionId: string): Promise<ConversionResult | null> {
    try {
      // In a real implementation, this would query the XION blockchain
      console.log('Checking direct transaction status:', transactionId);

      // Simulate blockchain query
      const result: ConversionResult = {
        transactionId,
        usdcAmount: '100',
        nrnAmount: '1000',
        status: 'confirmed',
        timestamp: new Date(Date.now() - 60000) // 1 minute ago
      };

      return result;
    } catch (error) {
      console.error('Failed to check direct transaction status:', error);
      return null;
    }
  }

  // Map KNIRVORACLE payment status to ConversionResult status
  private mapKNIRVORACLEStatus(knirvStatus: string): 'pending' | 'confirmed' | 'failed' {
    switch (knirvStatus) {
      case 'pending':
      case 'processing':
        return 'pending';
      case 'completed':
        return 'confirmed';
      case 'failed':
      case 'expired':
        return 'failed';
      default:
        return 'pending';
    }
  }

  // Execute gasless transaction via Treasury Contract
  private async executeGaslessTransaction(_txMsg: TransactionMessage): Promise<TransactionResult> {
    try {
      // In a real implementation, this would:
      // 1. Request permission from Treasury Contract
      // 2. Submit transaction to be sponsored
      // 3. Return transaction result

      console.log('Requesting gasless transaction execution...');

      // Mock gasless execution
      return {
        transactionHash: 'gasless_tx_' + Date.now(),
        gasUsed: '0', // No gas used by user
        fee: '0', // No fee paid by user
        success: true
      };
    } catch (error) {
      console.error('Gasless transaction failed:', error);
      throw error;
    }
  }

  // Monitor KNIRVORACLE payment status
  private async monitorKNIRVORACLEPayment(paymentId: string): Promise<void> {
    try {
      console.log(`Monitoring KNIRVORACLE payment: ${paymentId}`);

      const checkStatus = async () => {
        try {
          const response = await fetch(`${this.config.endpoints.knirvOracle}/api/payment/status/${paymentId}`);

          if (!response.ok) {
            console.error(`Failed to check payment status: ${response.status}`);
            return;
          }

          const statusResult = await response.json();

          if (statusResult.success) {
            const status = statusResult.data.status;
            console.log(`Payment ${paymentId} status: ${status}`);

            if (status === 'completed') {
              console.log(`Payment ${paymentId} completed successfully`);
              // Update UI or trigger callbacks here
              return;
            } else if (status === 'failed') {
              console.error(`Payment ${paymentId} failed: ${statusResult.data.error_message}`);
              return;
            }
          }

          // Continue monitoring if still pending
          if (statusResult.data.status === 'pending' || statusResult.data.status === 'processing') {
            setTimeout(checkStatus, 5000); // Check again in 5 seconds
          }
        } catch (error) {
          console.error('Error checking payment status:', error);
        }
      };

      // Start monitoring
      setTimeout(checkStatus, 2000); // Initial check after 2 seconds
    } catch (error) {
      console.error('Payment monitoring setup failed:', error);
    }
  }

  // Monitor transaction status (for direct transactions)
  private async monitorTransactionStatus(result: ConversionResult): Promise<void> {
    try {
      // Simulate transaction monitoring
      setTimeout(() => {
        result.status = 'confirmed';
        console.log(`Transaction ${result.transactionId} confirmed`);
      }, 3000);
    } catch (error) {
      console.error('Transaction monitoring failed:', error);
      result.status = 'failed';
    }
  }

  // Get payment gateway configuration
  async getPaymentGatewayConfig(): Promise<Record<string, unknown>> {
    try {
      const response = await fetch(`${this.config.endpoints.knirvOracle}/api/payment/config`);

      if (!response.ok) {
        throw new Error(`Failed to get payment gateway config: ${response.status}`);
      }

      const configResult = await response.json();

      if (!configResult.success) {
        throw new Error(`Config fetch failed: ${configResult.error}`);
      }

      return configResult.data;
    } catch (error) {
      console.error('Failed to get payment gateway config:', error);
      // Return default config
      return {
        chain_id: 'xion-testnet-1',
        conversion_rate: this.config.paymentGateway.conversionRate,
        gasless_enabled: true,
        min_transaction_amount: this.config.paymentGateway.minTransactionAmount,
        max_transaction_amount: this.config.paymentGateway.maxTransactionAmount,
        supported_meta_accounts: ['email', 'social', 'wallet', 'passkey']
      };
    }
  }

  // Get current conversion rates
  async getConversionRates(): Promise<Record<string, unknown>> {
    try {
      const response = await fetch(`${this.config.endpoints.knirvOracle}/api/payment/rates`);

      if (!response.ok) {
        throw new Error(`Failed to get conversion rates: ${response.status}`);
      }

      const ratesResult = await response.json();

      if (!ratesResult.success) {
        throw new Error(`Rates fetch failed: ${ratesResult.error}`);
      }

      return ratesResult.data;
    } catch (error) {
      console.error('Failed to get conversion rates:', error);
      // Return default rates
      return {
        usdc_to_nrn: this.config.paymentGateway.conversionRate,
        last_updated: new Date().toISOString(),
        rate_type: 'fixed',
        base_currency: 'USDC',
        quote_currency: 'NRN'
      };
    }
  }

  // Get payment history for current user
  async getPaymentHistory(): Promise<ConversionResult[]> {
    if (!this.account) {
      throw new Error('No wallet connected');
    }

    try {
      const response = await fetch(`${this.config.endpoints.knirvOracle}/api/payment/history/${this.account.address}`);

      if (!response.ok) {
        throw new Error(`Failed to get payment history: ${response.status}`);
      }

      const historyResult = await response.json();

      if (!historyResult.success) {
        throw new Error(`History fetch failed: ${historyResult.error}`);
      }

      // Convert KNIRVORACLE payment records to ConversionResult format
      return historyResult.data.payments.map((payment: {
        payment_id: string;
        usdc_amount: string;
        nrn_amount: string;
        completed_at: string;
        gas_fee?: string;
      }) => ({
        transactionId: payment.payment_id,
        usdcAmount: payment.usdc_amount,
        nrnAmount: payment.nrn_amount,
        status: this.mapKNIRVORACLEStatus('completed'), // History only shows completed payments
        timestamp: new Date(payment.completed_at),
        gasUsed: payment.gas_fee || '0',
        fee: payment.gas_fee || '0'
      }));
    } catch (error) {
      console.error('Failed to get payment history:', error);
      return [];
    }
  }

  // Connect Meta Account to KNIRVORACLE
  async connectMetaAccount(authMethod: 'email' | 'social' | 'wallet' | 'passkey', identifier: string): Promise<Record<string, unknown>> {
    try {
      const response = await fetch(`${this.config.endpoints.knirvOracle}/api/payment/meta-account/connect`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          auth_method: authMethod,
          identifier: identifier
        })
      });

      if (!response.ok) {
        throw new Error(`Failed to connect meta account: ${response.status}`);
      }

      const connectResult = await response.json();

      if (!connectResult.success) {
        throw new Error(`Meta account connection failed: ${connectResult.error}`);
      }

      return connectResult.data;
    } catch (error) {
      console.error('Failed to connect meta account:', error);
      throw error;
    }
  }

  // Get account balance from KNIRVORACLE
  async getAccountBalance(address?: string): Promise<{
    address: string;
    usdc_balance: string;
    nrn_balance: string;
    last_updated: string;
  }> {
    const targetAddress = address || this.account?.address;

    if (!targetAddress) {
      throw new Error('No address provided and no wallet connected');
    }

    try {
      const response = await fetch(`${this.config.endpoints.knirvOracle}/api/payment/meta-account/balance/${targetAddress}`);

      if (!response.ok) {
        throw new Error(`Failed to get account balance: ${response.status}`);
      }

      const balanceResult = await response.json();

      if (!balanceResult.success) {
        throw new Error(`Balance fetch failed: ${balanceResult.error}`);
      }

      return balanceResult.data;
    } catch (error) {
      console.error('Failed to get account balance:', error);
      // Return mock balance
      return {
        address: targetAddress,
        usdc_balance: '1000000000', // 1000 USDC
        nrn_balance: '5000000000000000000000', // 5000 NRN
        last_updated: new Date().toISOString()
      };
    }
  }

  // Mock SDK methods (in real implementation, these would be from @burnt-labs/abstraxion-react-native)
  private async mockConnect(options: ConnectOptions): Promise<{ success: boolean; account: string }> {
    console.log('Mock connect with options:', options);
    return { success: true, account: 'xion1...' };
  }

  private async mockDisconnect(): Promise<void> {
    console.log('Mock disconnect');
  }

  private async mockSignTransaction(txMsg: TransactionMessage): Promise<TransactionResult> {
    console.log('Mock sign transaction:', txMsg);
    return {
      transactionHash: 'tx_' + Date.now(),
      gasUsed: '150000',
      fee: '0.001',
      success: true
    };
  }

  private async mockExecuteContract(contractMsg: ContractMessage): Promise<TransactionResult> {
    console.log('Mock execute contract:', contractMsg);
    return {
      transactionHash: 'contract_tx_' + Date.now(),
      success: true
    };
  }

  // Get conversion history
  async getConversionHistory(): Promise<ConversionResult[]> {
    try {
      // In a real implementation, this would fetch from blockchain or local storage
      console.log('Fetching conversion history...');

      // Simulate history with gasless and regular transactions
      const history: ConversionResult[] = [
        {
          transactionId: 'gasless_tx_1234567890',
          usdcAmount: '50',
          nrnAmount: '500',
          status: 'confirmed',
          timestamp: new Date(Date.now() - 86400000), // 1 day ago
          gasUsed: '0',
          fee: '0'
        },
        {
          transactionId: 'tx_0987654321',
          usdcAmount: '25',
          nrnAmount: '250',
          status: 'confirmed',
          timestamp: new Date(Date.now() - 172800000), // 2 days ago
          gasUsed: '150000',
          fee: '0.001'
        }
      ];

      return history;
    } catch (error) {
      console.error('Failed to get conversion history:', error);
      return [];
    }
  }
}

// React Hook for Abstraxion wallet
export const useAbstraxionWallet = () => {
  const [walletService] = useState(() => new AbstraxionWalletService());
  const [account, setAccount] = useState<XIONAccount | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    // Check if already connected
    const currentAccount = walletService.getCurrentAccount();
    if (currentAccount) {
      setAccount(currentAccount);
      setIsConnected(currentAccount.isConnected);
    }
  }, [walletService]);

  const connect = async (authMethod: 'email' | 'social' | 'wallet' | 'passkey' = 'email') => {
    setIsLoading(true);
    try {
      const connectedAccount = await walletService.connectWallet(authMethod);
      if (connectedAccount) {
        setAccount(connectedAccount);
        setIsConnected(true);
      }
    } catch (error) {
      console.error('Failed to connect wallet:', error);
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  const disconnect = async () => {
    setIsLoading(true);
    try {
      await walletService.disconnectWallet();
      setAccount(null);
      setIsConnected(false);
    } catch (error) {
      console.error('Failed to disconnect wallet:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const convertUSDCToNRN = async (request: ConversionRequest) => {
    if (!account) throw new Error('No wallet connected');

    setIsLoading(true);
    try {
      // Enable gasless by default if treasury is available
      const gaslessRequest = { ...request, gasless: true };
      const result = await walletService.convertUSDCToNRN(gaslessRequest);

      // Update local account balance with both USDC and NRN changes
      if (account) {
        const conversionRate = 10; // Same rate as in service
        const nrnAmount = parseFloat(request.usdcAmount) * conversionRate;

        setAccount({
          ...account,
          usdcBalance: (parseFloat(account.usdcBalance) - parseFloat(request.usdcAmount)).toString(),
          nrnBalance: (parseFloat(account.nrnBalance || '0') + nrnAmount).toString()
        });
      }

      return result;
    } catch (error) {
      console.error('Conversion failed:', error);
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  const getUSDCBalance = async () => {
    if (!account) return '0';
    return await walletService.getUSDCBalance();
  };

  return {
    account,
    isConnected,
    isLoading,
    connect,
    disconnect,
    convertUSDCToNRN,
    getUSDCBalance
  };
};



// Export singleton instance
export const abstraxionWalletService = new AbstraxionWalletService();
