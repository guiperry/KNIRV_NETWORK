import React from 'react';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import '@testing-library/jest-dom';
import { TargetManager } from '../components/TargetManager';

// Mock the TargetSystemModal component
jest.mock('../components/modals/TargetSystemModal', () => {
  return jest.fn(({ isOpen, onClose, target, onTargetSaved }) => {
    if (!isOpen) return null;
    return (
      <div data-testid="target-system-modal">
        <h2>{target ? 'Edit Target System' : 'Add Target System'}</h2>
        <div>Target: {target ? target.name : 'New Target'}</div>
        <button onClick={() => onClose()}>Close</button>
        <button onClick={() => onTargetSaved({
          id: target ? target.id : Date.now(),
          name: target ? `Updated ${target.name}` : 'New Test Target',
          type: target ? target.type : 'browser',
          status: 'connected',
          capabilities: ['Test Capability'],
          permissions: ['read'],
          security: 'medium',
          lastActivity: 'just now',
          icon: target ? target.icon : require('lucide-react').Globe
        })}>
          Save Target
        </button>
      </div>
    );
  });
});

describe('TargetManager', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should render the component with initial targets', () => {
    render(<TargetManager />);
    expect(screen.getByRole('heading', { name: /Target System Manager/i })).toBeInTheDocument();
    expect(screen.getByText('Chrome Browser')).toBeInTheDocument();
    expect(screen.getByText('Local File System')).toBeInTheDocument();
  });

  test('should filter targets based on search term', () => {
    render(<TargetManager />);
    
    // Initially all targets should be visible
    expect(screen.getByText('Chrome Browser')).toBeInTheDocument();
    expect(screen.getByText('Local File System')).toBeInTheDocument();
    
    // Search for "Chrome"
    fireEvent.change(screen.getByPlaceholderText('Search target systems...'), {
      target: { value: 'Chrome' }
    });
    
    // Only Chrome Browser should be visible
    expect(screen.getByText('Chrome Browser')).toBeInTheDocument();
    expect(screen.queryByText('Local File System')).not.toBeInTheDocument();
  });

  test('should filter targets based on category', () => {
    render(<TargetManager />);
    
    // Initially all targets should be visible
    expect(screen.getByText('Chrome Browser')).toBeInTheDocument();
    expect(screen.getByText('Local File System')).toBeInTheDocument();
    
    // Filter by "filesystem" category
    fireEvent.click(screen.getByText('File Systems'));
    
    // Only File System targets should be visible
    expect(screen.queryByText('Chrome Browser')).not.toBeInTheDocument();
    expect(screen.getByText('Local File System')).toBeInTheDocument();
  });

  test('should open modal when Add Target button is clicked', () => {
    render(<TargetManager />);
    
    // Click Add Target button
    fireEvent.click(screen.getByText('Add Target'));
    
    // Modal should be open
    expect(screen.getByTestId('target-system-modal')).toBeInTheDocument();
    expect(screen.getByText('Add Target System')).toBeInTheDocument();
    
    // Close the modal
    fireEvent.click(screen.getByText('Close'));
    
    // Modal should be closed
    expect(screen.queryByTestId('target-system-modal')).not.toBeInTheDocument();
  });

  test('should open modal with target data when Edit button is clicked', () => {
    render(<TargetManager />);

    // Find the Edit button for the first target and click it
    const editButtons = screen.getAllByTestId('edit-target-button');
    fireEvent.click(editButtons[0]); // First edit button should be for Chrome Browser

    // Modal should be open with target data
    expect(screen.getByTestId('target-system-modal')).toBeInTheDocument();
    expect(screen.getByText('Edit Target System')).toBeInTheDocument();
    expect(screen.getByText('Target: Chrome Browser')).toBeInTheDocument();

    // Close the modal
    fireEvent.click(screen.getByText('Close'));

    // Modal should be closed
    expect(screen.queryByTestId('target-system-modal')).not.toBeInTheDocument();
  });

  test('should add a new target when created through the modal', async () => {
    render(<TargetManager />);
    
    // Initial target count
    const initialTargetCount = screen.getAllByText(/Browser|File System|Editor|Terminal|Interface|Photoshop|iPhone|Database/i).length;
    
    // Open creation modal
    fireEvent.click(screen.getByText('Add Target'));
    
    // Create a new target
    fireEvent.click(screen.getByText('Save Target'));
    
    // Wait for the new target to be added
    await waitFor(() => {
      const newTargetCount = screen.getAllByText(/Browser|File System|Editor|Terminal|Interface|Photoshop|iPhone|Database|New Test Target/i).length;
      expect(newTargetCount).toBe(initialTargetCount + 1);
    });
    
    // Verify the new target is in the list
    expect(screen.getByText('New Test Target')).toBeInTheDocument();
  });

  test('should update an existing target when edited through the modal', async () => {
    render(<TargetManager />);

    // Find the Edit button for the first target and click it
    const editButtons = screen.getAllByTestId('edit-target-button');
    fireEvent.click(editButtons[0]); // First edit button should be for Chrome Browser

    // Edit the target
    fireEvent.click(screen.getByText('Save Target'));

    // Wait for the target to be updated
    await waitFor(() => {
      expect(screen.getByText('Updated Chrome Browser')).toBeInTheDocument();
    });
  });

  test('should toggle target connection status when Connect/Disconnect button is clicked', () => {
    render(<TargetManager />);
    
    // Find a connected target
    const chromeTarget = screen.getByText('Chrome Browser').closest('[role="article"]');
    const disconnectButton = within(chromeTarget).getByText('Disconnect');
    
    // Click Disconnect button
    fireEvent.click(disconnectButton);
    
    // Target should now be disconnected
    expect(within(chromeTarget).getByText('disconnected')).toBeInTheDocument();
    expect(within(chromeTarget).getByText('Connect')).toBeInTheDocument();
    
    // Click Connect button
    const connectButton = within(chromeTarget).getByText('Connect');
    fireEvent.click(connectButton);
    
    // Target should now be connected again
    expect(within(chromeTarget).getByText('connected')).toBeInTheDocument();
    expect(within(chromeTarget).getByText('Disconnect')).toBeInTheDocument();
  });
});