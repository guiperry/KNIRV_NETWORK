import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router';
import VaultPage from '@/react-app/pages/VaultPage';

// Mock Layout
vi.mock('@/react-app/components/Layout', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div data-testid="layout-wrapper">{children}</div>,
}));

// Mock the useVault hook
const mockUnlockVault = vi.fn();
const mockLockVault = vi.fn();

vi.mock('@/react-app/hooks/useVault', () => ({
  useVault: () => ({
    status: 'unlocked',
    currentAccount: {
      getAddress: (_prefix: string) => 'knirv1abc123def456',
    },
    unlockVault: mockUnlockVault,
    lockVault: mockLockVault,
  }),
}));

// Mock the useBackend hook
const mockRefresh = vi.fn();
const mockSendNRN = vi.fn();

vi.mock('@/react-app/hooks/useBackend', () => ({
  useBackend: () => ({
    walletData: {
      nrnBalance: 18247,
      usdValue: 4567.89,
      change24h: 2.4,
    },
    transactions: [
      {
        id: 'tx-1',
        type: 'consumption',
        amount: 500,
        description: 'DVE Staking: DVE-Alpha',
        timestamp: new Date().toISOString(),
      },
      {
        id: 'tx-2',
        type: 'reward',
        amount: 120,
        description: 'Workflow reward: Data Verification',
        timestamp: new Date().toISOString(),
      },
    ],
    isLoading: false,
    refresh: mockRefresh,
    sendNRN: mockSendNRN,
  }),
}));

// Mock lucide-react
vi.mock('lucide-react', () => ({
  Wallet: () => <div data-testid="icon-wallet" />,
  ArrowUpRight: () => <div data-testid="icon-arrow-up-right" />,
  ArrowDownLeft: () => <div data-testid="icon-arrow-down-left" />,
  Zap: () => <div data-testid="icon-zap" />,
  TrendingUp: () => <div data-testid="icon-trending-up" />,
  Copy: () => <div data-testid="icon-copy" />,
  ExternalLink: () => <div data-testid="icon-external-link" />,
  Lock: () => <div data-testid="icon-lock" />,
  Loader: () => <div data-testid="icon-loader" />,
  RefreshCw: () => <div data-testid="icon-refresh" />,
}));

const renderWithRouter = (ui: React.ReactElement) => {
  return render(<BrowserRouter>{ui}</BrowserRouter>);
};

describe('VaultPage', () => {
  it('renders with Layout wrapper', () => {
    renderWithRouter(<VaultPage />);
    expect(screen.getByTestId('layout-wrapper')).toBeInTheDocument();
  });

  it('renders "Vault" heading text', () => {
    renderWithRouter(<VaultPage />);
    expect(screen.getByText('Vault')).toBeInTheDocument();
  });

  it('renders "Manage your Vault and DVE wallet assets" description', () => {
    renderWithRouter(<VaultPage />);
    expect(screen.getByText('Manage your Vault and DVE wallet assets')).toBeInTheDocument();
  });

  it('renders the NRN Balance card', () => {
    renderWithRouter(<VaultPage />);
    expect(screen.getByText('NRN Balance')).toBeInTheDocument();
  });

  it('displays the NRN balance amount', () => {
    renderWithRouter(<VaultPage />);
    expect(screen.getByText('18,247 NRN')).toBeInTheDocument();
  });

  it('displays the USD value', () => {
    renderWithRouter(<VaultPage />);
    expect(screen.getByText('$4,567.89 USD')).toBeInTheDocument();
  });

  it('renders the wallet address section', () => {
    renderWithRouter(<VaultPage />);
    expect(screen.getByText('Wallet Address')).toBeInTheDocument();
  });

  it('renders the "Send NRN" button to trigger send modal', () => {
    renderWithRouter(<VaultPage />);
    const sendButton = screen.getByText('Send NRN');
    expect(sendButton).toBeInTheDocument();
  });

  it('opens the send modal when click "Send NRN" button', () => {
    renderWithRouter(<VaultPage />);
    const sendButton = screen.getByText('Send NRN');
    fireEvent.click(sendButton);
    expect(screen.getByText('Send NRN')).toBeInTheDocument();
    expect(screen.getByText('Transfer tokens to another address')).toBeInTheDocument();
  });

  it('renders "Recent Transactions" section', () => {
    renderWithRouter(<VaultPage />);
    expect(screen.getByText('Recent Transactions')).toBeInTheDocument();
  });

  it('renders "Add Funds" action button', () => {
    renderWithRouter(<VaultPage />);
    expect(screen.getByText('Add Funds')).toBeInTheDocument();
  });

  it('renders the Refresh button', () => {
    renderWithRouter(<VaultPage />);
    expect(screen.getByText('Refresh')).toBeInTheDocument();
  });

  it('renders a lock button for locking vault', () => {
    renderWithRouter(<VaultPage />);
    // The lock button is an SVG icon button; ensure Lock icon testid exists
    expect(screen.getByTestId('icon-lock')).toBeInTheDocument();
  });
});
