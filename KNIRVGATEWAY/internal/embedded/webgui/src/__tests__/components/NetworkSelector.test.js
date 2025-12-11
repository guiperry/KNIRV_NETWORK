import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import NetworkSelector from '../../components/NetworkSelector';
import { NetworkProvider } from '../../contexts/NetworkContext';

// Mock fetch for network health checks
global.fetch = jest.fn();

// Mock WebSocket
global.WebSocket = jest.fn().mockImplementation(() => ({
  onopen: null,
  onerror: null,
  close: jest.fn()
}));

const MockNetworkProvider = ({ children }) => (
  <NetworkProvider>
    {children}
  </NetworkProvider>
);

describe('NetworkSelector', () => {
  beforeEach(() => {
    fetch.mockClear();
    localStorage.clear();
  });

  test('renders network selector with loading state initially', () => {
    render(
      <MockNetworkProvider>
        <NetworkSelector />
      </MockNetworkProvider>
    );

    expect(screen.getByText('Loading networks...')).toBeInTheDocument();
  });

  test('renders compact selector correctly', async () => {
    // Mock successful health check
    fetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ status: 'healthy' })
    });

    render(
      <MockNetworkProvider>
        <NetworkSelector compact={true} />
      </MockNetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('KNIRV Testnet')).toBeInTheDocument();
    });
  });

  test('displays available networks', async () => {
    fetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ status: 'healthy' })
    });

    render(
      <MockNetworkProvider>
        <NetworkSelector />
      </MockNetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('KNIRV Mainnet')).toBeInTheDocument();
      expect(screen.getByText('KNIRV Testnet')).toBeInTheDocument();
      expect(screen.getByText('Local Development')).toBeInTheDocument();
      expect(screen.getByText('Private Testnet')).toBeInTheDocument();
    });
  });

  test('switches networks when clicked', async () => {
    fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'healthy' })
    });

    render(
      <MockNetworkProvider>
        <NetworkSelector />
      </MockNetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('KNIRV Testnet')).toBeInTheDocument();
    });

    // Click on mainnet
    const mainnetButton = screen.getByText('KNIRV Mainnet');
    fireEvent.click(mainnetButton);

    await waitFor(() => {
      expect(localStorage.getItem('knirv-selected-network')).toBe('mainnet');
    });
  });

  test('displays network health status', async () => {
    fetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ status: 'healthy' })
    });

    render(
      <MockNetworkProvider>
        <NetworkSelector />
      </MockNetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Network Health')).toBeInTheDocument();
      expect(screen.getByText('API')).toBeInTheDocument();
      expect(screen.getByText('WebSocket')).toBeInTheDocument();
    });
  });

  test('handles network connection errors', async () => {
    fetch.mockRejectedValueOnce(new Error('Network error'));

    render(
      <MockNetworkProvider>
        <NetworkSelector />
      </MockNetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Connection Error')).toBeInTheDocument();
    });
  });

  test('refreshes network health when refresh button clicked', async () => {
    fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'healthy' })
    });

    render(
      <MockNetworkProvider>
        <NetworkSelector />
      </MockNetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByTitle('Refresh network status')).toBeInTheDocument();
    });

    const refreshButton = screen.getByTitle('Refresh network status');
    fireEvent.click(refreshButton);

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledTimes(2); // Initial load + refresh
    });
  });

  test('opens explorer link in new tab', async () => {
    fetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ status: 'healthy' })
    });

    render(
      <MockNetworkProvider>
        <NetworkSelector />
      </MockNetworkProvider>
    );

    await waitFor(() => {
      const explorerLink = screen.getByText('Open Explorer');
      expect(explorerLink).toHaveAttribute('target', '_blank');
      expect(explorerLink).toHaveAttribute('rel', 'noopener noreferrer');
    });
  });
});

describe('NetworkSelector Compact Mode', () => {
  test('toggles dropdown when clicked', async () => {
    fetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ status: 'healthy' })
    });

    render(
      <MockNetworkProvider>
        <NetworkSelector compact={true} />
      </MockNetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('KNIRV Testnet')).toBeInTheDocument();
    });

    // Click to open dropdown
    const compactButton = screen.getByRole('button');
    fireEvent.click(compactButton);

    await waitFor(() => {
      expect(screen.getByText('KNIRV Mainnet')).toBeInTheDocument();
    });
  });

  test('closes dropdown when network is selected', async () => {
    fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'healthy' })
    });

    render(
      <MockNetworkProvider>
        <NetworkSelector compact={true} />
      </MockNetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('KNIRV Testnet')).toBeInTheDocument();
    });

    // Open dropdown
    const compactButton = screen.getByRole('button');
    fireEvent.click(compactButton);

    // Select mainnet
    const mainnetOption = screen.getByText('KNIRV Mainnet');
    fireEvent.click(mainnetOption);

    await waitFor(() => {
      expect(screen.queryByText('Local Development')).not.toBeInTheDocument();
    });
  });
});
