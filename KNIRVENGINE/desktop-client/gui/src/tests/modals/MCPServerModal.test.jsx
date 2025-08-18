import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import MCPServerModal from '../../components/modals/MCPServerModal';

describe('MCPServerModal', () => {
  const mockOnClose = jest.fn();
  const mockOnServerSaved = jest.fn();
  
  const mockServer = {
    id: 1,
    name: 'Test Server',
    type: 'llm',
    description: 'Test description',
    endpoint: 'https://test.example.com',
    apiKey: 'test-api-key',
    version: '1.0.0'
  };
  
  const renderModal = (isOpen = true, server = null) => {
    return render(
      <MCPServerModal 
        isOpen={isOpen} 
        onClose={mockOnClose} 
        server={server} 
        onServerSaved={mockOnServerSaved} 
      />
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should not render when isOpen is false', () => {
    renderModal(false);
    expect(screen.queryByText('Add MCP Server')).not.toBeInTheDocument();
    expect(screen.queryByText('Edit MCP Server')).not.toBeInTheDocument();
  });

  test('should render "Add MCP Server" when creating a new server', () => {
    renderModal(true, null);
    expect(screen.getByText('Add MCP Server')).toBeInTheDocument();
  });

  test('should render "Edit MCP Server" when editing an existing server', () => {
    renderModal(true, mockServer);
    expect(screen.getByText('Edit MCP Server')).toBeInTheDocument();
  });

  test('should populate form fields when editing an existing server', () => {
    renderModal(true, mockServer);
    expect(screen.getByLabelText('Server Name')).toHaveValue(mockServer.name);
    expect(screen.getByLabelText('Description')).toHaveValue(mockServer.description);
    expect(screen.getByLabelText('Endpoint URL')).toHaveValue(mockServer.endpoint);
    expect(screen.getByLabelText('API Key (if required)')).toHaveValue(mockServer.apiKey);
    expect(screen.getByLabelText('Version')).toHaveValue(mockServer.version);

    // Check if the correct server type is selected
    const llmTypeElement = screen.getByText('LLM Provider');
    // The gradient class is on the parent div that contains the server type
    const serverTypeContainer = llmTypeElement.closest('div').parentElement;
    expect(serverTypeContainer).toHaveClass('bg-gradient-to-r');
  });

  test('should call onClose when cancel button is clicked', () => {
    renderModal();
    fireEvent.click(screen.getByText('Cancel'));
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  test('should call onClose when X button is clicked', () => {
    renderModal();
    fireEvent.click(screen.getByLabelText('Close'));
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  test('should show validation errors when form is submitted with invalid data', async () => {
    renderModal();
    
    // Clear required fields
    fireEvent.change(screen.getByLabelText('Server Name'), {
      target: { value: '' }
    });
    
    fireEvent.change(screen.getByLabelText('Endpoint URL'), {
      target: { value: '' }
    });
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Server'));
    
    // Wait for validation errors
    await waitFor(() => {
      expect(screen.getByText('Server name is required')).toBeInTheDocument();
      expect(screen.getByText('Endpoint URL is required')).toBeInTheDocument();
    });
    
    // Ensure onServerSaved was not called
    expect(mockOnServerSaved).not.toHaveBeenCalled();
  });

  test('should validate endpoint URL format', async () => {
    renderModal();
    
    // Fill out the form with invalid URL
    fireEvent.change(screen.getByLabelText('Server Name'), {
      target: { value: 'Test Server' }
    });
    
    fireEvent.change(screen.getByLabelText('Endpoint URL'), {
      target: { value: 'invalid-url' }
    });
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Server'));
    
    // Wait for validation errors
    await waitFor(() => {
      expect(screen.getByText('Please enter a valid URL')).toBeInTheDocument();
    });
    
    // Ensure onServerSaved was not called
    expect(mockOnServerSaved).not.toHaveBeenCalled();
  });

  test('should successfully save server when form is valid', async () => {
    renderModal();
    
    // Fill out the form
    fireEvent.change(screen.getByLabelText('Server Name'), {
      target: { value: 'New Test Server' }
    });
    
    fireEvent.change(screen.getByLabelText('Endpoint URL'), {
      target: { value: 'https://test.example.com' }
    });
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Server'));
    
    // Wait for success message and callback
    await waitFor(() => {
      expect(screen.getByText('Server saved!')).toBeInTheDocument();
      expect(mockOnServerSaved).toHaveBeenCalledTimes(1);
      expect(mockOnServerSaved).toHaveBeenCalledWith(expect.objectContaining({
        name: 'New Test Server',
        endpoint: 'https://test.example.com',
        status: 'connected'
      }));
    }, { timeout: 3000 });
  });

  test('should test connection successfully', async () => {
    // Mock Math.random to always return 0.5 (success case)
    const originalRandom = Math.random;
    Math.random = jest.fn(() => 0.5);
    
    renderModal();
    
    // Fill required fields
    fireEvent.change(screen.getByLabelText('Server Name'), {
      target: { value: 'Test Server' }
    });
    
    fireEvent.change(screen.getByLabelText('Endpoint URL'), {
      target: { value: 'https://test.example.com' }
    });
    
    // Click test connection button
    fireEvent.click(screen.getByText('Test Connection'));
    
    // Wait for testing state
    expect(screen.getByText('Testing Connection...')).toBeInTheDocument();
    
    // Wait for success message
    await waitFor(() => {
      expect(screen.getByText('Connection successful!')).toBeInTheDocument();
    }, { timeout: 3000 });
    
    // Restore Math.random
    Math.random = originalRandom;
  });

  test('should handle connection test failure', async () => {
    // Mock Math.random to always return 0.9 (failure case)
    const originalRandom = Math.random;
    Math.random = jest.fn(() => 0.9);
    
    renderModal();
    
    // Fill required fields
    fireEvent.change(screen.getByLabelText('Server Name'), {
      target: { value: 'Test Server' }
    });
    
    fireEvent.change(screen.getByLabelText('Endpoint URL'), {
      target: { value: 'https://test.example.com' }
    });
    
    // Click test connection button
    fireEvent.click(screen.getByText('Test Connection'));
    
    // Wait for testing state
    expect(screen.getByText('Testing Connection...')).toBeInTheDocument();
    
    // Wait for error message
    await waitFor(() => {
      expect(screen.getByText('Connection failed!')).toBeInTheDocument();
    }, { timeout: 5000 });
    
    // Restore Math.random
    Math.random = originalRandom;
  });

  test('should toggle API key visibility', () => {
    renderModal(true, mockServer);
    
    // API key should be hidden initially
    const apiKeyInput = screen.getByLabelText('API Key (if required)');
    expect(apiKeyInput).toHaveAttribute('type', 'password');
    
    // Click show button
    fireEvent.click(screen.getByText('Show'));
    
    // API key should be visible
    expect(apiKeyInput).toHaveAttribute('type', 'text');
    
    // Click hide button
    fireEvent.click(screen.getByText('Hide'));
    
    // API key should be hidden again
    expect(apiKeyInput).toHaveAttribute('type', 'password');
  });
});