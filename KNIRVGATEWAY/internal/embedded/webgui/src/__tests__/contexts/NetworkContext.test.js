import React from 'react';
import { render, screen, act, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { NetworkProvider, useNetwork } from '../../contexts/NetworkContext';

// Mock fetch
global.fetch = jest.fn();

// Mock WebSocket
global.WebSocket = jest.fn().mockImplementation(() => ({
  onopen: null,
  onerror: null,
  close: jest.fn()
}));

// Test component to access network context
const TestComponent = () => {
  const {
    currentNetwork,
    isConnecting,
    connectionStatus,
    networks,
    switchNetwork,
    getApiClient
  } = useNetwork();

  return (
    <div>
      <div data-testid="current-network">
        {currentNetwork ? currentNetwork.name : 'No network'}
      </div>
      <div data-testid="connection-status">{connectionStatus}</div>
      <div data-testid="is-connecting">{isConnecting.toString()}</div>
      <div data-testid="networks-count">{Object.keys(networks).length}</div>
      <button 
        onClick={() => switchNetwork('mainnet')}
        data-testid="switch-to-mainnet"
      >
        Switch to Mainnet
      </button>
      <button 
        onClick={() => {
          const api = getApiClient();
          api.get('/test');
        }}
        data-testid="test-api"
      >
        Test API
      </button>
    </div>
  );
};

describe('NetworkContext', () => {
  beforeEach(() => {
    fetch.mockClear();
    localStorage.clear();
    // Mock successful health check by default
    fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'healthy' })
    });
  });

  test('provides network context to children', async () => {
    render(
      <NetworkProvider>
        <TestComponent />
      </NetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('current-network')).toHaveTextContent('KNIRV Testnet');
      expect(screen.getByTestId('networks-count')).toHaveTextContent('4');
    });
  });

  test('initializes with testnet by default', async () => {
    render(
      <NetworkProvider>
        <TestComponent />
      </NetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('current-network')).toHaveTextContent('KNIRV Testnet');
    });
  });

  test('loads saved network from localStorage', async () => {
    localStorage.setItem('knirv-selected-network', 'mainnet');

    render(
      <NetworkProvider>
        <TestComponent />
      </NetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('current-network')).toHaveTextContent('KNIRV Mainnet');
    });
  });

  test('switches networks correctly', async () => {
    render(
      <NetworkProvider>
        <TestComponent />
      </NetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('current-network')).toHaveTextContent('KNIRV Testnet');
    });

    const switchButton = screen.getByTestId('switch-to-mainnet');
    
    await act(async () => {
      switchButton.click();
    });

    await waitFor(() => {
      expect(screen.getByTestId('current-network')).toHaveTextContent('KNIRV Mainnet');
      expect(localStorage.getItem('knirv-selected-network')).toBe('mainnet');
    });
  });

  test('updates connection status based on health checks', async () => {
    render(
      <NetworkProvider>
        <TestComponent />
      </NetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('connection-status')).toHaveTextContent('connected');
    });
  });

  test('handles network health check failures', async () => {
    fetch.mockRejectedValueOnce(new Error('Network error'));

    render(
      <NetworkProvider>
        <TestComponent />
      </NetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('connection-status')).toHaveTextContent('error');
    });
  });

  test('provides API client with correct configuration', async () => {
    fetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: 'test' })
    });

    render(
      <NetworkProvider>
        <TestComponent />
      </NetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('current-network')).toHaveTextContent('KNIRV Testnet');
    });

    const testApiButton = screen.getByTestId('test-api');
    
    await act(async () => {
      testApiButton.click();
    });

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        'https://testnet-api.knirv.network/test',
        expect.objectContaining({
          method: 'GET',
          headers: expect.objectContaining({
            'Content-Type': 'application/json'
          })
        })
      );
    });
  });

  test('throws error when useNetwork is used outside provider', () => {
    // Suppress console.error for this test
    const originalError = console.error;
    console.error = jest.fn();

    expect(() => {
      render(<TestComponent />);
    }).toThrow('useNetwork must be used within a NetworkProvider');

    console.error = originalError;
  });

  test('dispatches networkChanged event when switching networks', async () => {
    const eventListener = jest.fn();
    window.addEventListener('networkChanged', eventListener);

    render(
      <NetworkProvider>
        <TestComponent />
      </NetworkProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('current-network')).toHaveTextContent('KNIRV Testnet');
    });

    const switchButton = screen.getByTestId('switch-to-mainnet');
    
    await act(async () => {
      switchButton.click();
    });

    await waitFor(() => {
      expect(eventListener).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'networkChanged',
          detail: expect.objectContaining({
            network: expect.objectContaining({
              id: 'mainnet',
              name: 'KNIRV Mainnet'
            })
          })
        })
      );
    });

    window.removeEventListener('networkChanged', eventListener);
  });

  test('API client handles POST requests correctly', async () => {
    fetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ success: true })
    });

    let apiClient;

    const TestApiComponent = () => {
      const { getApiClient } = useNetwork();
      
      React.useEffect(() => {
        apiClient = getApiClient();
      }, [getApiClient]);

      return <div>API Test</div>;
    };

    render(
      <NetworkProvider>
        <TestApiComponent />
      </NetworkProvider>
    );

    await waitFor(() => {
      expect(apiClient).toBeDefined();
    });

    await act(async () => {
      await apiClient.post('/test', { data: 'test' });
    });

    expect(fetch).toHaveBeenCalledWith(
      'https://testnet-api.knirv.network/test',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({
          'Content-Type': 'application/json'
        }),
        body: JSON.stringify({ data: 'test' })
      })
    );
  });
});
