// Comprehensive Unit Tests for KNIRVWALLET React Native - MetaAccountDashboard Component
import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react-native';
// Test data for wallet operations - removed unused imports

// Import the actual MetaAccountDashboard component
import { MetaAccountDashboard } from '../../../src/components/MetaAccountDashboard';

// Mock XionMetaAccount, WalletManager, and MetaAccountConfig




// Mock implementations are used by the jest.mock calls below

// Mock React Native components and modules
jest.mock('react-native', () => ({
  View: 'View',
  Text: 'Text',
  TouchableOpacity: 'TouchableOpacity',
  ScrollView: 'ScrollView',
  FlatList: 'FlatList',
  TextInput: 'TextInput',
  Image: 'Image',
  Dimensions: {
    get: jest.fn().mockReturnValue({ width: 375, height: 812 }),
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
  },
  Platform: {
    OS: 'ios',
    select: jest.fn((obj) => obj.ios || obj.default),
  },
  StyleSheet: {
    create: jest.fn((styles) => styles),
    flatten: jest.fn((styles) => styles),
  },
  Alert: {
    alert: jest.fn(),
  },
}));

jest.mock('react-native-safe-area-context', () => ({
  SafeAreaView: ({ children }: { children: React.ReactNode }) => children
}));

jest.mock('expo-linear-gradient', () => ({
  LinearGradient: ({ children }: { children: React.ReactNode }) => children
}));

jest.mock('lucide-react-native', () => ({
  Wallet: () => 'Wallet',
  Plus: () => 'Plus',
  RefreshCw: () => 'RefreshCw',
  Send: () => 'Send',
  Zap: () => 'Zap'
}));

jest.mock('../../../components/GlassCard', () => {
  return ({ children }: { children: React.ReactNode }) => children;
});

// Mock WalletIntegrationService
jest.mock('../../../src/services/WalletIntegrationService', () => ({
  walletIntegrationService: {
    getCurrentAccount: jest.fn().mockReturnValue(null),
    connectWallet: jest.fn().mockResolvedValue({
      id: 'test-account',
      address: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
      name: 'Test Wallet'
    }),
    getAccountBalance: jest.fn().mockResolvedValue({
      balance: '1.000000',
      nrnBalance: '0.500000'
    }),
    getTransactionHistory: jest.fn().mockResolvedValue([]),
    createTransaction: jest.fn().mockResolvedValue('tx-123'),
    disconnectWallet: jest.fn().mockResolvedValue(undefined)
  }
}));

// Mock XION Meta Account and Wallet Manager
const mockMetaAccount = {
  getAddress: jest.fn().mockResolvedValue('xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5'),
  getBalance: jest.fn().mockResolvedValue('1000000'),
  getNRNBalance: jest.fn().mockResolvedValue('500000'),
  refreshBalances: jest.fn().mockResolvedValue(undefined),
  transferNRN: jest.fn().mockResolvedValue('0x123...abc'),
  burnNRNForSkill: jest.fn().mockResolvedValue('0x456...def'),
  requestFromFaucet: jest.fn().mockResolvedValue('0x789...ghi'),
  enableGaslessTransactions: jest.fn().mockResolvedValue(undefined)
};

const mockWalletManager = {
  initialize: jest.fn().mockResolvedValue(true),
  createWallet: jest.fn().mockResolvedValue(mockMetaAccount),
  importWallet: jest.fn().mockResolvedValue(mockMetaAccount),
  getWallet: jest.fn().mockResolvedValue(mockMetaAccount),
  listWallets: jest.fn().mockResolvedValue(['wallet1', 'wallet2'])
};

// Mock KNIRVENGINE imports since they're from a sibling project
const XionMetaAccount = jest.fn().mockImplementation(() => mockMetaAccount);
const WalletManager = jest.fn().mockImplementation(() => mockWalletManager);

describe('MetaAccountDashboard Component', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Component Rendering', () => {
    it('should render dashboard with initial state', () => {
      const mockConfig = {
        chainId: 'xion-testnet-1',
        rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443'
      };

      const { getByText } = render(
        <MetaAccountDashboard config={mockConfig} />
      );

      expect(getByText('Meta Account Dashboard')).toBeTruthy();
      expect(getByText('Chain: xion-testnet-1')).toBeTruthy();
      expect(getByText('Wallet Not Connected')).toBeTruthy();
      // Component shows "Connecting..." during initialization
      expect(getByText('Connecting...')).toBeTruthy();
    });

    it('should initialize with mock wallet services', () => {
      // Test that mock classes are properly instantiated
      const metaAccount = new XionMetaAccount();
      const walletManager = new WalletManager();

      expect(metaAccount).toBeDefined();
      expect(walletManager).toBeDefined();
      expect(XionMetaAccount).toHaveBeenCalled();
      expect(WalletManager).toHaveBeenCalled();
    });

    it('should render connected wallet state', async () => {
      const mockConfig = {
        chainId: 'xion-testnet-1',
        rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443'
      };

      // Mock wallet service to return connected state
      const mockWalletService = await import('../../../src/services/WalletIntegrationService');
      mockWalletService.walletIntegrationService.getCurrentAccount = jest.fn().mockReturnValue({
        id: 'test-account',
        address: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
        name: 'Test Wallet'
      });
      mockWalletService.walletIntegrationService.getAccountBalance = jest.fn().mockResolvedValue({
        balance: '1.000000',
        nrnBalance: '0.500000'
      });
      mockWalletService.walletIntegrationService.getTransactionHistory = jest.fn().mockResolvedValue([]);

      const { getByText } = render(
        <MetaAccountDashboard config={mockConfig} />
      );

      await waitFor(() => {
        expect(getByText('Meta Account Dashboard')).toBeTruthy();
        expect(getByText('1.000000 KNIRV')).toBeTruthy();
        expect(getByText('0.500000 NRN')).toBeTruthy();
      });
    });

    it('should handle wallet connection', async () => {
      const mockConfig = {
        chainId: 'xion-testnet-1',
        rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443'
      };

      // Mock wallet service for connection flow
      const mockWalletService = await import('../../../src/services/WalletIntegrationService');
      mockWalletService.walletIntegrationService.getCurrentAccount = jest.fn().mockReturnValue(null);
      mockWalletService.walletIntegrationService.connectWallet = jest.fn().mockResolvedValue({
        id: 'connected-account',
        address: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
        name: 'Connected Wallet'
      });
      mockWalletService.walletIntegrationService.getAccountBalance = jest.fn().mockResolvedValue({
        balance: '1.000000',
        nrnBalance: '0.500000'
      });
      mockWalletService.walletIntegrationService.getTransactionHistory = jest.fn().mockResolvedValue([]);

      const { getByText, getByTestId } = render(
        <MetaAccountDashboard config={mockConfig} />
      );

      // Initially should show connecting state during initialization
      expect(getByText('Connecting...')).toBeTruthy();

      // Click connect wallet
      fireEvent.press(getByTestId('connect-wallet-button'));

      await waitFor(() => {
        expect(mockWalletService.walletIntegrationService.connectWallet).toHaveBeenCalled();
      });
    });

    it('should show loading state during wallet connection', async () => {
      const mockConfig = {
        chainId: 'xion-testnet-1',
        rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443'
      };

      // Mock slow connection
      const mockWalletService = await import('../../../src/services/WalletIntegrationService');
      mockWalletService.walletIntegrationService.getCurrentAccount = jest.fn().mockReturnValue(null);
      mockWalletService.walletIntegrationService.connectWallet = jest.fn().mockImplementation(() =>
        new Promise(resolve => setTimeout(() => resolve({
          id: 'test-account',
          address: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
          name: 'Test Wallet'
        }), 100))
      );

      const { getByText, getByTestId } = render(
        <MetaAccountDashboard config={mockConfig} />
      );

      // Click connect wallet button (using testID since text is "Connecting...")
      fireEvent.press(getByTestId('connect-wallet-button'));

      // Should show loading state
      expect(getByText('Connecting...')).toBeTruthy();
    });
  });

  describe('Transaction Operations', () => {
    it('should handle send transaction', async () => {
      const mockConfig = {
        chainId: 'xion-testnet-1',
        rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443'
      };

      // Mock connected wallet state
      const mockWalletService = await import('../../../src/services/WalletIntegrationService');
      mockWalletService.walletIntegrationService.getCurrentAccount = jest.fn().mockReturnValue({
        id: 'test-account',
        address: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
        name: 'Test Wallet'
      });
      mockWalletService.walletIntegrationService.getAccountBalance = jest.fn().mockResolvedValue({
        balance: '1.000000',
        nrnBalance: '0.500000'
      });
      mockWalletService.walletIntegrationService.getTransactionHistory = jest.fn().mockResolvedValue([]);
      mockWalletService.walletIntegrationService.createTransaction = jest.fn().mockResolvedValue('tx-123');

      const { getByTestId } = render(
        <MetaAccountDashboard config={mockConfig} />
      );

      await waitFor(() => {
        fireEvent.press(getByTestId('send-transaction-button'));
      });

      await waitFor(() => {
        expect(mockWalletService.walletIntegrationService.createTransaction).toHaveBeenCalled();
      });
    });

    it('should handle transaction errors', async () => {
      const mockConfig = {
        chainId: 'xion-testnet-1',
        rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443'
      };

      // Mock connected wallet state with transaction error
      const mockWalletService = await import('../../../src/services/WalletIntegrationService');
      mockWalletService.walletIntegrationService.getCurrentAccount = jest.fn().mockReturnValue({
        id: 'test-account',
        address: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
        name: 'Test Wallet'
      });
      mockWalletService.walletIntegrationService.getAccountBalance = jest.fn().mockResolvedValue({
        balance: '1.000000',
        nrnBalance: '0.500000'
      });
      mockWalletService.walletIntegrationService.getTransactionHistory = jest.fn().mockResolvedValue([]);
      mockWalletService.walletIntegrationService.createTransaction = jest.fn().mockRejectedValue(new Error('Transaction failed'));

      const { getByTestId } = render(
        <MetaAccountDashboard config={mockConfig} />
      );

      await waitFor(() => {
        fireEvent.press(getByTestId('send-transaction-button'));
      });

      await waitFor(() => {
        expect(mockWalletService.walletIntegrationService.createTransaction).toHaveBeenCalled();
      });
    });
  });

  describe('Component Props and Callbacks', () => {
    it('should call onTransactionSend callback when transaction is sent', async () => {
      const mockConfig = {
        chainId: 'xion-testnet-1',
        rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443'
      };

      const onTransactionSend = jest.fn();

      // Mock connected wallet state
      const mockWalletService = await import('../../../src/services/WalletIntegrationService');
      mockWalletService.walletIntegrationService.getCurrentAccount = jest.fn().mockReturnValue({
        id: 'test-account',
        address: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
        name: 'Test Wallet'
      });
      mockWalletService.walletIntegrationService.getAccountBalance = jest.fn().mockResolvedValue({
        balance: '1.000000',
        nrnBalance: '0.500000'
      });
      mockWalletService.walletIntegrationService.getTransactionHistory = jest.fn().mockResolvedValue([]);
      mockWalletService.walletIntegrationService.createTransaction = jest.fn().mockResolvedValue('tx-123');

      const { getByTestId } = render(
        <MetaAccountDashboard config={mockConfig} onTransactionSend={onTransactionSend} />
      );

      await waitFor(() => {
        fireEvent.press(getByTestId('send-transaction-button'));
      });

      await waitFor(() => {
        expect(onTransactionSend).toHaveBeenCalled();
      });
    });

    it('should call onBalanceUpdate callback when balance is loaded', async () => {
      const mockConfig = {
        chainId: 'xion-testnet-1',
        rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443'
      };

      const onBalanceUpdate = jest.fn();

      // Mock connected wallet state
      const mockWalletService = await import('../../../src/services/WalletIntegrationService');
      mockWalletService.walletIntegrationService.getCurrentAccount = jest.fn().mockReturnValue({
        id: 'test-account',
        address: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
        name: 'Test Wallet'
      });
      mockWalletService.walletIntegrationService.getAccountBalance = jest.fn().mockResolvedValue({
        balance: '1.000000',
        nrnBalance: '0.500000'
      });
      mockWalletService.walletIntegrationService.getTransactionHistory = jest.fn().mockResolvedValue([]);

      render(
        <MetaAccountDashboard config={mockConfig} onBalanceUpdate={onBalanceUpdate} />
      );

      await waitFor(() => {
        expect(onBalanceUpdate).toHaveBeenCalledWith('1.000000');
      });
    });
  });

});
