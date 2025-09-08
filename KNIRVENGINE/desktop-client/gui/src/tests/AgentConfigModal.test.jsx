import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import AgentConfigModal from '../components/modals/AgentConfigModal';

describe('AgentConfigModal', () => {
  const mockOnClose = jest.fn();
  const mockOnAgentUpdated = jest.fn();
  
  const mockAgent = {
    id: 1,
    name: 'Test Agent',
    collection: 'Test Collection',
    status: 'active',
    capabilities: ['Web Analysis', 'Data Extraction'],
    targetTypes: ['Browser', 'File System'],
    settings: {
      responseLength: 'medium',
      responseStyle: 'balanced',
      maxInferences: 100,
      securityLevel: 'standard',
      timeout: 30,
      autoRestart: true,
      logLevel: 'info'
    }
  };
  
  const renderModal = (isOpen = true, agent = mockAgent) => {
    return render(
      <AgentConfigModal 
        isOpen={isOpen} 
        onClose={mockOnClose} 
        agent={agent} 
        onAgentUpdated={mockOnAgentUpdated} 
      />
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should not render when isOpen is false', () => {
    renderModal(false);
    expect(screen.queryByText('Agent Configuration')).not.toBeInTheDocument();
  });

  test('should not render when agent is null', () => {
    renderModal(true, null);
    expect(screen.queryByText('Agent Configuration')).not.toBeInTheDocument();
  });

  test('should render when isOpen is true and agent is provided', () => {
    renderModal();
    expect(screen.getByRole('heading', { name: /Agent Configuration/i })).toBeInTheDocument();
    expect(screen.getByText(mockAgent.name)).toBeInTheDocument();
  });

  test('should display all tabs', () => {
    renderModal();
    expect(screen.getByRole('tab', { name: /General/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Capabilities/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Target Types/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Performance/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Security/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Advanced/i })).toBeInTheDocument();
  });

  test('should switch tabs when clicked', () => {
    renderModal();
    
    // Default tab should be General
    expect(screen.getByLabelText('Agent Name')).toBeInTheDocument();
    
    // Click on Capabilities tab
    fireEvent.click(screen.getByRole('tab', { name: /Capabilities/i }));
    expect(screen.getByText('Agent Capabilities')).toBeInTheDocument();
    
    // Click on Target Types tab
    fireEvent.click(screen.getByRole('tab', { name: /Target Types/i }));
    expect(screen.getAllByText('Target Types')[0]).toBeInTheDocument();
    
    // Click on Performance tab
    fireEvent.click(screen.getByRole('tab', { name: /Performance/i }));
    expect(screen.getByText('Performance Settings')).toBeInTheDocument();
    
    // Click on Security tab
    fireEvent.click(screen.getByRole('tab', { name: /Security/i }));
    expect(screen.getByText('Security Settings')).toBeInTheDocument();
    
    // Click on Advanced tab
    fireEvent.click(screen.getByRole('tab', { name: /Advanced/i }));
    expect(screen.getByText('Advanced Settings')).toBeInTheDocument();
    
    // Go back to General tab
    fireEvent.click(screen.getByRole('tab', { name: /General/i }));
    expect(screen.getByLabelText('Agent Name')).toBeInTheDocument();
  });

  test('should call onClose when cancel button is clicked', () => {
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /Cancel/i }));
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  test('should call onClose when X button is clicked', () => {
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /close/i }));
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  test('should show validation errors when form is submitted with invalid data', async () => {
    renderModal();
    
    // Clear required fields
    fireEvent.change(screen.getByLabelText(/Agent Name/i), {
      target: { value: '' }
    });
    
    fireEvent.change(screen.getByLabelText(/Collection/i), {
      target: { value: '' }
    });
    
    // Submit the form
    fireEvent.click(screen.getByRole('button', { name: /Save Configuration/i }));
    
    // Wait for validation errors
    await waitFor(() => {
      expect(screen.getByText('Agent name is required')).toBeInTheDocument();
      expect(screen.getByText('Collection name is required')).toBeInTheDocument();
    });
    
    // Ensure onAgentUpdated was not called
    expect(mockOnAgentUpdated).not.toHaveBeenCalled();
  });

  test('should successfully update agent when form is valid', async () => {
    renderModal();
    
    // Update agent name
    fireEvent.change(screen.getByLabelText(/Agent Name/i), {
      target: { value: 'Updated Agent Name' }
    });
    
    // Go to Performance tab and update settings
    fireEvent.click(screen.getByText('Performance'));
    
    fireEvent.change(screen.getByLabelText(/Maximum Inferences Per Day/i), {
      target: { value: '200' }
    });
    
    // Submit the form
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /Save Configuration/i }));
    });

    // Wait for the form to be submitting (loading state)
    await waitFor(() => {
      expect(screen.getByText('Saving...')).toBeInTheDocument();
    }, { timeout: 1000 });

    // Wait for success message (component has 1.5s delay)
    await waitFor(() => {
      expect(screen.getByText('Configuration saved!')).toBeInTheDocument();
    }, { timeout: 3000 });

    // Verify the callback was called
    await waitFor(() => {
      expect(mockOnAgentUpdated).toHaveBeenCalledTimes(1);
    }, { timeout: 1000 });

    // Verify the callback was called with correct data
    expect(mockOnAgentUpdated).toHaveBeenCalledWith(expect.objectContaining({
      id: mockAgent.id,
      name: 'Updated Agent Name',
      settings: expect.objectContaining({
        maxInferences: '200' // HTML input values are strings
      })
    }));
  }, 10000); // Increase test timeout to 10 seconds
});