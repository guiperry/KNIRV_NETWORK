import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { BrowserRouter } from 'react-router-dom';
import Dashboard from '../../pages/dashboard';
import { NetworkProvider } from '../../contexts/NetworkContext';
import { RoleProvider } from '../../contexts/RoleContext';

// Mock fetch
global.fetch = jest.fn();

// Mock WebSocket
global.WebSocket = jest.fn().mockImplementation(() => ({
  onopen: null,
  onerror: null,
  close: jest.fn()
}));

const MockProviders = ({ children }) => (
  <BrowserRouter>
    <NetworkProvider>
      <RoleProvider>
        {children}
      </RoleProvider>
    </NetworkProvider>
  </BrowserRouter>
);

describe('Dashboard', () => {
  beforeEach(() => {
    fetch.mockClear();
    localStorage.clear();
    
    // Mock successful network health check
    fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'healthy' })
    });
  });

  test('renders dashboard with main sections', async () => {
    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getByText('KNIRV Network Oracle Dashboard')).toBeInTheDocument();
      expect(screen.getByText('Welcome to the KNIRV Network')).toBeInTheDocument();
    });
  });

  test('displays network status cards', async () => {
    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getByText('Network Status')).toBeInTheDocument();
      expect(screen.getByText('Active Nodes')).toBeInTheDocument();
      expect(screen.getByText('Total Models')).toBeInTheDocument();
      expect(screen.getByText('Active Skills')).toBeInTheDocument();
    });
  });

  test('shows KNIRVCONTROLLER connection section', async () => {
    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getByText('KNIRVCONTROLLER Connection')).toBeInTheDocument();
      expect(screen.getByText('Connect Mobile App')).toBeInTheDocument();
    });
  });

  test('displays personal API endpoints', async () => {
    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getByText('Personal API Endpoints')).toBeInTheDocument();
      expect(screen.getByText('Model Management API')).toBeInTheDocument();
      expect(screen.getByText('Skill Execution API')).toBeInTheDocument();
    });
  });

  test('shows quick actions section', async () => {
    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getByText('Quick Actions')).toBeInTheDocument();
      expect(screen.getByText('Deploy Model')).toBeInTheDocument();
      expect(screen.getByText('Create Skill')).toBeInTheDocument();
      expect(screen.getByText('Open Marketplace')).toBeInTheDocument();
    });
  });

  test('handles QR code modal opening and closing', async () => {
    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getByText('Show QR Code')).toBeInTheDocument();
    });

    // Open QR modal
    const qrButton = screen.getByText('Show QR Code');
    fireEvent.click(qrButton);

    await waitFor(() => {
      expect(screen.getByText('KNIRVCONTROLLER Connection QR')).toBeInTheDocument();
    });

    // Close QR modal
    const closeButton = screen.getByText('×');
    fireEvent.click(closeButton);

    await waitFor(() => {
      expect(screen.queryByText('KNIRVCONTROLLER Connection QR')).not.toBeInTheDocument();
    });
  });

  test('displays recent activity section', async () => {
    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getByText('Recent Activity')).toBeInTheDocument();
      expect(screen.getByText('Model Training Completed')).toBeInTheDocument();
      expect(screen.getByText('New Skill Published')).toBeInTheDocument();
    });
  });

  test('shows network statistics with correct values', async () => {
    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getByText('1,247')).toBeInTheDocument(); // Active Nodes
      expect(screen.getByText('3,456')).toBeInTheDocument(); // Total Models
      expect(screen.getByText('892')).toBeInTheDocument(); // Active Skills
      expect(screen.getByText('15,432')).toBeInTheDocument(); // Transactions
    });
  });

  test('handles copy to clipboard for API endpoints', async () => {
    // Mock clipboard API
    Object.assign(navigator, {
      clipboard: {
        writeText: jest.fn().mockResolvedValue()
      }
    });

    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getAllByTitle('Copy to clipboard')).toHaveLength(3);
    });

    const copyButtons = screen.getAllByTitle('Copy to clipboard');
    fireEvent.click(copyButtons[0]);

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        'https://api.knirv.network/user/models'
      );
    });
  });

  test('displays connection status indicators', async () => {
    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getByText('Connected')).toBeInTheDocument();
      expect(screen.getByText('Not Connected')).toBeInTheDocument();
    });
  });

  test('handles network status refresh', async () => {
    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    await waitFor(() => {
      expect(screen.getByTitle('Refresh Status')).toBeInTheDocument();
    });

    const refreshButton = screen.getByTitle('Refresh Status');
    fireEvent.click(refreshButton);

    // Should trigger additional fetch calls
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledTimes(2); // Initial + refresh
    });
  });

  test('shows appropriate loading states', async () => {
    // Mock slow network response
    fetch.mockImplementationOnce(() => 
      new Promise(resolve => setTimeout(() => resolve({
        ok: true,
        json: () => Promise.resolve({ status: 'healthy' })
      }), 100))
    );

    render(
      <MockProviders>
        <Dashboard />
      </MockProviders>
    );

    // Should show loading state initially
    expect(screen.getByText('Loading...')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.queryByText('Loading...')).not.toBeInTheDocument();
    });
  });
});
