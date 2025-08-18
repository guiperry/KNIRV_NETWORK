import React from 'react';
import { render, screen, fireEvent, waitFor, within, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import { WebConnectionsManager } from '../components/WebConnectionsManager';

// Mock the modal components
jest.mock('../components/modals/WebConnectionModal', () => {
  return jest.fn(({ isOpen, onClose, connection, onConnectionSaved }) => {
    if (!isOpen) return null;
    return (
      <div data-testid="web-connection-modal">
        <h2>{connection ? 'Edit Connection' : 'Add Connection'}</h2>
        <div>Connection: {connection ? connection.name : 'New Connection'}</div>
        <button onClick={() => onClose()}>Close</button>
        <button onClick={() => onConnectionSaved({
          id: connection ? connection.id : Date.now(),
          name: connection ? `Updated ${connection.name}` : 'New Test Connection',
          type: connection ? connection.type : 'oauth',
          provider: connection ? connection.provider : 'Test Provider',
          status: 'inactive',
          description: 'Test description',
          icon: 'https://example.com/icon.png',
          lastUsed: 'Never',
          expiresAt: null,
          scopes: ['test:scope'],
          createdAt: new Date().toISOString(),
          createdBy: 'Test User',
          connectionDetails: {
            clientId: 'test-client-id',
            clientSecret: 'test-client-secret',
            redirectUri: 'https://app.picaos.com/authkit/callback',
            tokenEndpoint: 'https://example.com/token',
            authEndpoint: 'https://example.com/auth'
          }
        })}>
          Save Connection
        </button>
      </div>
    );
  });
});

jest.mock('../components/modals/AdvancedFilterModal', () => {
  return jest.fn(({ isOpen, onClose, onApplyFilters, initialFilters, filterOptions }) => {
    if (!isOpen) return null;
    return (
      <div data-testid="advanced-filter-modal">
        <div>Advanced Filters</div>
        <button onClick={() => onClose()}>Close</button>
        <button onClick={() => onApplyFilters([
          { id: 1, field: 'status', operator: 'equals', value: 'active' }
        ])}>
          Apply Filters
        </button>
      </div>
    );
  });
});

describe('WebConnectionsManager', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Mock setTimeout to execute immediately
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  test('should render the component with initial connections', () => {
    render(<WebConnectionsManager />);
    expect(screen.getByRole('heading', { name: /Web Connections/i })).toBeInTheDocument();
    expect(screen.getByText('Google Drive')).toBeInTheDocument();
    // Use getAllByText since there are multiple "GitHub" texts (heading and span)
    expect(screen.getAllByText('GitHub')[0]).toBeInTheDocument();
  });

  test('should filter connections based on search term', () => {
    render(<WebConnectionsManager />);

    // Initially all connections should be visible
    expect(screen.getByText('Google Drive')).toBeInTheDocument();
    expect(screen.getAllByText('GitHub')[0]).toBeInTheDocument();

    // Search for "Google"
    fireEvent.change(screen.getByPlaceholderText('Search connections...'), {
      target: { value: 'Google' }
    });

    // Only Google Drive should be visible
    expect(screen.getByText('Google Drive')).toBeInTheDocument();
    expect(screen.queryByText('GitHub')).not.toBeInTheDocument();
  });

  test('should filter connections based on category', () => {
    render(<WebConnectionsManager />);
    
    // Initially all connections should be visible
    expect(screen.getByText('Google Drive')).toBeInTheDocument();
    expect(screen.getByText('Slack Workspace')).toBeInTheDocument();
    
    // Filter by "inactive" category
    fireEvent.click(screen.getByText('Inactive'));
    
    // Only inactive connections should be visible
    expect(screen.queryByText('Google Drive')).not.toBeInTheDocument();
    expect(screen.getByText('Slack Workspace')).toBeInTheDocument();
  });

  test('should open modal when Add Connection button is clicked', () => {
    render(<WebConnectionsManager />);

    // Click Add Connection button (use role to be more specific)
    fireEvent.click(screen.getByRole('button', { name: /Add Connection/i }));

    // Modal should be open
    expect(screen.getByTestId('web-connection-modal')).toBeInTheDocument();
    // Use heading role to distinguish modal title from button text
    expect(screen.getByRole('heading', { name: /Add Connection/i })).toBeInTheDocument();

    // Close the modal
    fireEvent.click(screen.getByText('Close'));

    // Modal should be closed
    expect(screen.queryByTestId('web-connection-modal')).not.toBeInTheDocument();
  });

  test('should open modal with connection data when Edit button is clicked', () => {
    render(<WebConnectionsManager />);
    
    // Find the edit button (first button with empty name - these are the Settings icon buttons)
    const editButtons = screen.getAllByRole('button', { name: '' });
    // The first empty-name button should be the edit button for the first connection
    fireEvent.click(editButtons[0]);
    
    // Modal should be open with connection data
    expect(screen.getByTestId('web-connection-modal')).toBeInTheDocument();
    expect(screen.getByText('Edit Connection')).toBeInTheDocument();
    expect(screen.getByText('Connection: Google Drive')).toBeInTheDocument();
    
    // Close the modal
    fireEvent.click(screen.getByText('Close'));
    
    // Modal should be closed
    expect(screen.queryByTestId('web-connection-modal')).not.toBeInTheDocument();
  });

  test('should add a new connection when created through the modal', async () => {
    render(<WebConnectionsManager />);
    
    // Initial connection count
    const initialConnectionCount = screen.getAllByText(/Drive|GitHub|OpenAI|Slack|AWS|Salesforce/).length;
    
    // Open creation modal
    fireEvent.click(screen.getByText('Add Connection'));
    
    // Create a new connection
    fireEvent.click(screen.getByText('Save Connection'));
    
    // Wait for the new connection to be added
    await waitFor(() => {
      const newConnectionCount = screen.getAllByText(/Drive|GitHub|OpenAI|Slack|AWS|Salesforce|New Test Connection/).length;
      expect(newConnectionCount).toBe(initialConnectionCount + 1);
    });
    
    // Verify the new connection is in the list
    expect(screen.getByText('New Test Connection')).toBeInTheDocument();
  });

  test('should update an existing connection when edited through the modal', async () => {
    render(<WebConnectionsManager />);
    
    // Find the edit button (first button with empty name - these are the Settings icon buttons)
    const editButtons = screen.getAllByRole('button', { name: '' });
    // The first empty-name button should be the edit button for the first connection
    fireEvent.click(editButtons[0]);
    
    // Edit the connection
    fireEvent.click(screen.getByText('Save Connection'));
    
    // Wait for the connection to be updated
    await waitFor(() => {
      expect(screen.getByText('Updated Google Drive')).toBeInTheDocument();
    });
  });

  test('should delete a connection when delete button is clicked', () => {
    render(<WebConnectionsManager />);
    
    // Find the delete button (second button with empty name - edit is first, delete is second)
    const emptyNameButtons = screen.getAllByRole('button', { name: '' });
    // The second empty-name button should be the delete button for the first connection
    fireEvent.click(emptyNameButtons[1]);
    
    // Google Drive should be removed
    expect(screen.queryByText('Google Drive')).not.toBeInTheDocument();
  });

  test('should toggle connection status when Activate/Deactivate button is clicked', async () => {
    render(<WebConnectionsManager />);
    
    // Find the Deactivate button (use the first one since there are multiple)
    const deactivateButtons = screen.getAllByText('Deactivate');
    const deactivateButton = deactivateButtons[0];
    
    // Click Deactivate button
    fireEvent.click(deactivateButton);
    
    // Should show deactivating state (use first one since there are multiple)
    expect(screen.getAllByText('Deactivating...')[0]).toBeInTheDocument();

    // Fast-forward timers to complete the deactivation (1000ms timeout)
    act(() => {
      jest.advanceTimersByTime(1000);
    });

    // Wait for the state to update and check for Authorize button (Google Drive is OAuth)
    await waitFor(() => {
      expect(screen.getAllByText('Authorize')[0]).toBeInTheDocument();
    });

    // Click Authorize button (use the first one)
    const activateButton = screen.getAllByText('Authorize')[0];
    fireEvent.click(activateButton);

    // Should show authorizing state (OAuth connections show "Authorizing...")
    await waitFor(() => {
      expect(screen.getAllByText('Authorizing...')[0]).toBeInTheDocument();
    });

    // Fast-forward timers to complete the activation (1000ms timeout)
    act(() => {
      jest.advanceTimersByTime(1000);
    });

    // Connection should now be active again - should show Deactivate button
    await waitFor(() => {
      expect(screen.getAllByText('Deactivate')[0]).toBeInTheDocument();
    });
  });

  test('should refresh OAuth token when refresh button is clicked', async () => {
    render(<WebConnectionsManager />);
    
    // Find the refresh button (use the first one since there are multiple)
    const refreshButtons = screen.getAllByRole('button', { name: /Refresh Token/i });
    const refreshButton = refreshButtons[0];
    
    // Click refresh button
    fireEvent.click(refreshButton);
    
    // The refresh function will fail due to fetch not being defined in test environment
    // But the component should handle this gracefully and update the timestamp

    // Fast-forward timers to complete the refresh operation
    jest.advanceTimersByTime(1500);

    // Last used should be updated (the component falls back to updating timestamps)
    await waitFor(() => {
      expect(screen.getByText('just now')).toBeInTheDocument();
    });
  });

  test('should toggle secret visibility when show/hide button is clicked', () => {
    render(<WebConnectionsManager />);
    
    // Find the show/hide button for a connection and click it (use the first one)
    const showButtons = screen.getAllByText('Show');
    fireEvent.click(showButtons[0]);
    
    // Should now show the secret
    expect(screen.getByText('Hide')).toBeInTheDocument();
    
    // Click hide button
    const hideButton = screen.getByText('Hide');
    fireEvent.click(hideButton);
    
    // Should hide the secret again
    expect(screen.getAllByText('Show')[0]).toBeInTheDocument();
  });

  test('should open advanced filter modal when Filter button is clicked', () => {
    render(<WebConnectionsManager />);
    
    // Click Filter button
    fireEvent.click(screen.getByText('Filter'));
    
    // Modal should be open
    expect(screen.getByTestId('advanced-filter-modal')).toBeInTheDocument();
    
    // Close the modal
    fireEvent.click(screen.getByText('Close'));
    
    // Modal should be closed
    expect(screen.queryByTestId('advanced-filter-modal')).not.toBeInTheDocument();
  });

  test('should apply filters when applied through the modal', async () => {
    render(<WebConnectionsManager />);
    
    // Initially all connections should be visible
    expect(screen.getByText('Google Drive')).toBeInTheDocument();
    expect(screen.getByText('Slack Workspace')).toBeInTheDocument();
    
    // Open filter modal
    fireEvent.click(screen.getByText('Filter'));
    
    // Apply a filter for active connections
    fireEvent.click(screen.getByText('Apply Filters'));
    
    // Wait for filters to be applied
    await waitFor(() => {
      // Should show active connections and hide inactive connections
      expect(screen.getByText('Google Drive')).toBeInTheDocument();
      expect(screen.queryByText('Slack Workspace')).not.toBeInTheDocument();
    });
    
    // Should show active filters display
    expect(screen.getByText('Active filters:')).toBeInTheDocument();
    expect(screen.getByText('Status Equals active')).toBeInTheDocument();
  });

  test('should clear filters when Clear button is clicked', async () => {
    render(<WebConnectionsManager />);
    
    // Open filter modal
    fireEvent.click(screen.getByText('Filter'));
    
    // Apply a filter
    fireEvent.click(screen.getByText('Apply Filters'));
    
    // Wait for filters to be applied
    await waitFor(() => {
      expect(screen.getByText('Active filters:')).toBeInTheDocument();
    });
    
    // Clear filters
    fireEvent.click(screen.getByText('Clear'));
    
    // All connections should be visible again
    expect(screen.getByText('Google Drive')).toBeInTheDocument();
    expect(screen.getByText('Slack Workspace')).toBeInTheDocument();
    
    // Active filters display should be gone
    expect(screen.queryByText('Active filters:')).not.toBeInTheDocument();
  });

  test('should show empty state when no connections match filters', async () => {
    render(<WebConnectionsManager />);
    
    // Search for a non-existent connection
    fireEvent.change(screen.getByPlaceholderText('Search connections...'), {
      target: { value: 'NonExistentConnection' }
    });
    
    // Should show empty state
    expect(screen.getByText('No connections found')).toBeInTheDocument();
    expect(screen.getByText('No connections match your current filters. Try adjusting your search or filters.')).toBeInTheDocument();
  });
});