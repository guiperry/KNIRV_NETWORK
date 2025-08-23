// Comprehensive Unit Tests for KNIRVWALLET React Native - MetaAccountDashboard Component
import React from 'react';
import { render, fireEvent, waitFor, act } from '@testing-library/react-native';
import { Alert } from 'react-native';
import { MetaAccountDashboard } from '../../../../KNIRVENGINE/agentic-wallet/src/components/MetaAccountDashboard';
import { XionMetaAccount, WalletManager, MetaAccountConfig } from '../../../../KNIRVENGINE/agentic-wallet/src/xion-meta-accounts';
import { XionTestUtils } from '../../../test-utils/xion-test-utils';
import { TEST_ADDRESSES, TEST_MNEMONICS } from '../../../test-utils/test-data';

// Mock React Native components and modules
jest.mock('react-native', () => {
  const RN = jest.requireActual('react-native');
  return {
    ...RN,
    Alert: {
      alert: jest.fn()
    },
    Dimensions: {
      get: jest.fn().mockReturnValue({ width: 375, height: 812 })
    }
  };
});

jest.mock('react-native-safe-area-context', () => ({
  SafeAreaView: ({ children }: any) => children
}));

jest.mock('expo-linear-gradient', () => ({
  LinearGradient: ({ children }: any) => children
}));

jest.mock('lucide-react-native', () => ({
  Wallet: () => 'Wallet',
  Plus: () => 'Plus',
  RefreshCw: () => 'RefreshCw',
  Send: () => 'Send',
  Zap: () => 'Zap'
}));

jest.mock('../../../components/GlassCard', () => {
  return ({ children }: any) => children;
});

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
  createWallet: jest.fn().mockResolvedValue(mockMetaAccount),
  importWallet: jest.fn().mockResolvedValue(mockMetaAccount),
  getWallet: jest.fn().mockResolvedValue(mockMetaAccount),
  listWallets: jest.fn().mockResolvedValue(['wallet1', 'wallet2'])
};

jest.mock('../../../agentic-wallet/src/xion-meta-accounts', () => ({
  XionMetaAccount: jest.fn().mockImplementation(() => mockMetaAccount),
  WalletManager: jest.fn().mockImplementation(() => mockWalletManager)
}));

describe('MetaAccountDashboard Component', () => {
  let config: MetaAccountConfig;

  beforeEach(() => {
    config = XionTestUtils.createTestXionConfig('testnet');
    jest.clearAllMocks();
  });

  describe('Component Rendering', () => {
    it('should render dashboard with initial state', () => {
      const { getByText, getByTestId } = render(
        <MetaAccountDashboard config={config} />
      );

      expect(getByText('XION Meta Accounts')).toBeTruthy();
      expect(getByText('Create Wallet')).toBeTruthy();
      expect(getByText('Import Wallet')).toBeTruthy();
    });

    it('should render wallet list when wallets exist', async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      await waitFor(() => {
        expect(getByText('wallet1')).toBeTruthy();
        expect(getByText('wallet2')).toBeTruthy();
      });
    });

    it('should render wallet details when wallet is selected', async () => {
      const { getByText, getByTestId } = render(
        <MetaAccountDashboard config={config} />
      );

      // Wait for wallets to load and select first wallet
      await waitFor(() => {
        fireEvent.press(getByText('wallet1'));
      });

      await waitFor(() => {
        expect(getByText('1.000000 XION')).toBeTruthy();
        expect(getByText('0.500000 NRN')).toBeTruthy();
        expect(getByText('Transfer NRN')).toBeTruthy();
        expect(getByText('Invoke Skill')).toBeTruthy();
      });
    });

    it('should show loading state during operations', async () => {
      mockMetaAccount.refreshBalances.mockImplementation(() => 
        new Promise(resolve => setTimeout(resolve, 1000))
      );

      const { getByText, getByTestId } = render(
        <MetaAccountDashboard config={config} />
      );

      // Select wallet and trigger refresh
      await waitFor(() => {
        fireEvent.press(getByText('wallet1'));
      });

      await waitFor(() => {
        fireEvent.press(getByTestId('refresh-button'));
      });

      // Should show loading indicator
      expect(getByTestId('loading-indicator')).toBeTruthy();
    });
  });

  describe('Wallet Creation', () => {
    it('should create new wallet successfully', async () => {
      const { getByText, getByTestId, getByPlaceholderText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open create wallet modal
      fireEvent.press(getByText('Create Wallet'));

      // Enter wallet name
      const walletNameInput = getByPlaceholderText('Enter wallet name');
      fireEvent.changeText(walletNameInput, 'New Test Wallet');

      // Submit creation
      fireEvent.press(getByText('Create'));

      await waitFor(() => {
        expect(mockWalletManager.createWallet).toHaveBeenCalledWith('New Test Wallet');
      });
    });

    it('should validate wallet name input', async () => {
      const { getByText, getByTestId } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open create wallet modal
      fireEvent.press(getByText('Create Wallet'));

      // Try to create without name
      fireEvent.press(getByText('Create'));

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Please enter a wallet name'
        );
      });
    });

    it('should handle wallet creation errors', async () => {
      mockWalletManager.createWallet.mockRejectedValueOnce(new Error('Creation failed'));

      const { getByText, getByPlaceholderText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open create wallet modal
      fireEvent.press(getByText('Create Wallet'));

      // Enter wallet name and submit
      const walletNameInput = getByPlaceholderText('Enter wallet name');
      fireEvent.changeText(walletNameInput, 'Error Wallet');
      fireEvent.press(getByText('Create'));

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Failed to create wallet: Creation failed'
        );
      });
    });
  });

  describe('Wallet Import', () => {
    it('should import wallet from mnemonic successfully', async () => {
      const { getByText, getByPlaceholderText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open import wallet modal
      fireEvent.press(getByText('Import Wallet'));

      // Enter wallet details
      const walletNameInput = getByPlaceholderText('Enter wallet name');
      const mnemonicInput = getByPlaceholderText('Enter mnemonic phrase');

      fireEvent.changeText(walletNameInput, 'Imported Wallet');
      fireEvent.changeText(mnemonicInput, TEST_MNEMONICS.VALID_12_WORD);

      // Submit import
      fireEvent.press(getByText('Import'));

      await waitFor(() => {
        expect(mockWalletManager.importWallet).toHaveBeenCalledWith(
          'Imported Wallet',
          TEST_MNEMONICS.VALID_12_WORD
        );
      });
    });

    it('should validate import inputs', async () => {
      const { getByText, getByPlaceholderText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open import wallet modal
      fireEvent.press(getByText('Import Wallet'));

      // Try to import without inputs
      fireEvent.press(getByText('Import'));

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Please enter wallet name and mnemonic'
        );
      });
    });

    it('should handle invalid mnemonic', async () => {
      mockWalletManager.importWallet.mockRejectedValueOnce(new Error('Invalid mnemonic'));

      const { getByText, getByPlaceholderText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open import wallet modal
      fireEvent.press(getByText('Import Wallet'));

      // Enter invalid mnemonic
      const walletNameInput = getByPlaceholderText('Enter wallet name');
      const mnemonicInput = getByPlaceholderText('Enter mnemonic phrase');

      fireEvent.changeText(walletNameInput, 'Invalid Wallet');
      fireEvent.changeText(mnemonicInput, 'invalid mnemonic phrase');

      fireEvent.press(getByText('Import'));

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Failed to import wallet: Invalid mnemonic'
        );
      });
    });
  });

  describe('Balance Operations', () => {
    beforeEach(async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Select a wallet first
      await waitFor(() => {
        fireEvent.press(getByText('wallet1'));
      });
    });

    it('should refresh balances successfully', async () => {
      const { getByTestId } = render(
        <MetaAccountDashboard config={config} />
      );

      await waitFor(() => {
        fireEvent.press(getByTestId('refresh-button'));
      });

      expect(mockMetaAccount.refreshBalances).toHaveBeenCalled();
    });

    it('should handle balance refresh errors', async () => {
      mockMetaAccount.refreshBalances.mockRejectedValueOnce(new Error('Network error'));

      const { getByTestId } = render(
        <MetaAccountDashboard config={config} />
      );

      await waitFor(() => {
        fireEvent.press(getByTestId('refresh-button'));
      });

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Failed to refresh balances: Network error'
        );
      });
    });

    it('should display formatted balances correctly', async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      await waitFor(() => {
        expect(getByText('1.000000 XION')).toBeTruthy();
        expect(getByText('0.500000 NRN')).toBeTruthy();
      });
    });
  });

  describe('NRN Transfer Operations', () => {
    beforeEach(async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Select a wallet first
      await waitFor(() => {
        fireEvent.press(getByText('wallet1'));
      });
    });

    it('should transfer NRN successfully', async () => {
      const { getByText, getByPlaceholderText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open transfer modal
      fireEvent.press(getByText('Transfer NRN'));

      // Enter transfer details
      const recipientInput = getByPlaceholderText('Recipient address');
      const amountInput = getByPlaceholderText('Amount');

      fireEvent.changeText(recipientInput, TEST_ADDRESSES.XION);
      fireEvent.changeText(amountInput, '100000');

      // Submit transfer
      fireEvent.press(getByText('Send'));

      await waitFor(() => {
        expect(mockMetaAccount.transferNRN).toHaveBeenCalledWith(
          TEST_ADDRESSES.XION,
          '100000'
        );
      });

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Success',
          'Transfer successful! TX: 0x123...abc'
        );
      });
    });

    it('should validate transfer inputs', async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open transfer modal
      fireEvent.press(getByText('Transfer NRN'));

      // Try to transfer without inputs
      fireEvent.press(getByText('Send'));

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Please enter recipient address and amount'
        );
      });
    });

    it('should handle transfer errors', async () => {
      mockMetaAccount.transferNRN.mockRejectedValueOnce(new Error('Insufficient balance'));

      const { getByText, getByPlaceholderText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open transfer modal
      fireEvent.press(getByText('Transfer NRN'));

      // Enter transfer details
      const recipientInput = getByPlaceholderText('Recipient address');
      const amountInput = getByPlaceholderText('Amount');

      fireEvent.changeText(recipientInput, TEST_ADDRESSES.XION);
      fireEvent.changeText(amountInput, '999999999');

      fireEvent.press(getByText('Send'));

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Transfer failed: Insufficient balance'
        );
      });
    });
  });

  describe('Skill Invocation Operations', () => {
    beforeEach(async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Select a wallet first
      await waitFor(() => {
        fireEvent.press(getByText('wallet1'));
      });
    });

    it('should invoke skill successfully', async () => {
      const { getByText, getByPlaceholderText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open skill invocation modal
      fireEvent.press(getByText('Invoke Skill'));

      // Enter skill details
      const skillIdInput = getByPlaceholderText('Skill ID');
      const amountInput = getByPlaceholderText('NRN Amount');

      fireEvent.changeText(skillIdInput, 'skill-test-001');
      fireEvent.changeText(amountInput, '50000');

      // Submit skill invocation
      fireEvent.press(getByText('Invoke'));

      await waitFor(() => {
        expect(mockMetaAccount.burnNRNForSkill).toHaveBeenCalledWith(
          'skill-test-001',
          '50000'
        );
      });

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Success',
          'Skill invoked successfully! TX: 0x456...def'
        );
      });
    });

    it('should validate skill invocation inputs', async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open skill invocation modal
      fireEvent.press(getByText('Invoke Skill'));

      // Try to invoke without inputs
      fireEvent.press(getByText('Invoke'));

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Please enter skill ID and NRN amount'
        );
      });
    });

    it('should handle skill invocation errors', async () => {
      mockMetaAccount.burnNRNForSkill.mockRejectedValueOnce(new Error('Skill not found'));

      const { getByText, getByPlaceholderText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open skill invocation modal
      fireEvent.press(getByText('Invoke Skill'));

      // Enter skill details
      const skillIdInput = getByPlaceholderText('Skill ID');
      const amountInput = getByPlaceholderText('NRN Amount');

      fireEvent.changeText(skillIdInput, 'invalid-skill');
      fireEvent.changeText(amountInput, '50000');

      fireEvent.press(getByText('Invoke'));

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Skill invocation failed: Skill not found'
        );
      });
    });
  });

  describe('Faucet Operations', () => {
    beforeEach(async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Select a wallet first
      await waitFor(() => {
        fireEvent.press(getByText('wallet1'));
      });
    });

    it('should request from faucet successfully', async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      fireEvent.press(getByText('Request from Faucet'));

      await waitFor(() => {
        expect(mockMetaAccount.requestFromFaucet).toHaveBeenCalled();
      });

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Success',
          'Faucet request successful! TX: 0x789...ghi'
        );
      });
    });

    it('should handle faucet request errors', async () => {
      mockMetaAccount.requestFromFaucet.mockRejectedValueOnce(new Error('Faucet limit exceeded'));

      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      fireEvent.press(getByText('Request from Faucet'));

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Faucet request failed: Faucet limit exceeded'
        );
      });
    });
  });

  describe('Gasless Transaction Management', () => {
    beforeEach(async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Select a wallet first
      await waitFor(() => {
        fireEvent.press(getByText('wallet1'));
      });
    });

    it('should enable gasless transactions', async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      fireEvent.press(getByText('Enable Gasless'));

      await waitFor(() => {
        expect(mockMetaAccount.enableGaslessTransactions).toHaveBeenCalled();
      });

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Success',
          'Gasless transactions enabled!'
        );
      });
    });

    it('should handle gasless enablement errors', async () => {
      mockMetaAccount.enableGaslessTransactions.mockRejectedValueOnce(new Error('Setup failed'));

      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      fireEvent.press(getByText('Enable Gasless'));

      await waitFor(() => {
        expect(Alert.alert).toHaveBeenCalledWith(
          'Error',
          'Failed to enable gasless transactions: Setup failed'
        );
      });
    });
  });

  describe('Component State Management', () => {
    it('should manage modal visibility correctly', async () => {
      const { getByText, queryByText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open create wallet modal
      fireEvent.press(getByText('Create Wallet'));
      expect(getByText('Create New Wallet')).toBeTruthy();

      // Close modal
      fireEvent.press(getByText('Cancel'));
      await waitFor(() => {
        expect(queryByText('Create New Wallet')).toBeFalsy();
      });
    });

    it('should handle wallet switching correctly', async () => {
      const { getByText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Switch between wallets
      await waitFor(() => {
        fireEvent.press(getByText('wallet1'));
      });

      expect(mockWalletManager.getWallet).toHaveBeenCalledWith('wallet1');

      await waitFor(() => {
        fireEvent.press(getByText('wallet2'));
      });

      expect(mockWalletManager.getWallet).toHaveBeenCalledWith('wallet2');
    });

    it('should maintain form state correctly', async () => {
      const { getByText, getByPlaceholderText } = render(
        <MetaAccountDashboard config={config} />
      );

      // Open transfer modal and enter data
      fireEvent.press(getByText('Transfer NRN'));

      const recipientInput = getByPlaceholderText('Recipient address');
      fireEvent.changeText(recipientInput, 'partial-address');

      // Close and reopen modal
      fireEvent.press(getByText('Cancel'));
      fireEvent.press(getByText('Transfer NRN'));

      // Form should be reset
      const newRecipientInput = getByPlaceholderText('Recipient address');
      expect(newRecipientInput.props.value).toBe('');
    });
  });
});
