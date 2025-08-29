import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import MCPCapabilityModal from '../../components/modals/MCPCapabilityModal';

describe('MCPCapabilityModal', () => {
  const mockOnClose = jest.fn();
  const mockOnCapabilitySaved = jest.fn();
  
  const mockServer = {
    id: 1,
    name: 'Test Server',
    type: 'llm',
    status: 'connected'
  };
  
  const mockCapability = {
    id: 'test-capability',
    name: 'Test Capability',
    type: 'text',
    description: 'Test description',
    schema: '{"type": "object", "properties": {}}',
    parameters: '{"param1": "value1"}',
    version: '1.0.0'
  };
  
  const renderModal = (isOpen = true, server = mockServer, capability = null) => {
    return render(
      <MCPCapabilityModal 
        isOpen={isOpen} 
        onClose={mockOnClose} 
        server={server} 
        capability={capability} 
        onCapabilitySaved={mockOnCapabilitySaved} 
      />
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should not render when isOpen is false', () => {
    renderModal(false);
    expect(screen.queryByText('Add Capability')).not.toBeInTheDocument();
    expect(screen.queryByText('Edit Capability')).not.toBeInTheDocument();
  });

  test('should not render when server is null', () => {
    renderModal(true, null);
    expect(screen.queryByText('Add Capability')).not.toBeInTheDocument();
    expect(screen.queryByText('Edit Capability')).not.toBeInTheDocument();
  });

  test('should render "Add Capability" when creating a new capability', () => {
    renderModal(true, mockServer, null);
    expect(screen.getByText('Add Capability')).toBeInTheDocument();
    expect(screen.getByText(`Server: ${mockServer.name}`)).toBeInTheDocument();
  });

  test('should render "Edit Capability" when editing an existing capability', () => {
    renderModal(true, mockServer, mockCapability);
    expect(screen.getByText('Edit Capability')).toBeInTheDocument();
  });

  test('should populate form fields when editing an existing capability', () => {
    renderModal(true, mockServer, mockCapability);
    expect(screen.getByLabelText('Capability Name')).toHaveValue(mockCapability.name);
    expect(screen.getByLabelText('Description')).toHaveValue(mockCapability.description);
    expect(screen.getByLabelText('JSON Schema')).toHaveValue(mockCapability.schema);
    expect(screen.getByLabelText('Parameters (JSON)')).toHaveValue(mockCapability.parameters);
    expect(screen.getByLabelText('Version')).toHaveValue(mockCapability.version);
    
    // Check if the correct capability type is selected
    expect(screen.getByLabelText('Capability Type')).toHaveValue(mockCapability.type);
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
    fireEvent.change(screen.getByLabelText('Capability Name'), {
      target: { value: '' }
    });
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Capability'));
    
    // Wait for validation errors
    await waitFor(() => {
      expect(screen.getByText('Capability name is required')).toBeInTheDocument();
    });
    
    // Ensure onCapabilitySaved was not called
    expect(mockOnCapabilitySaved).not.toHaveBeenCalled();
  });

  test('should validate JSON schema', async () => {
    renderModal();
    
    // Enter invalid JSON
    fireEvent.change(screen.getByLabelText('JSON Schema'), {
      target: { value: '{invalid json}' }
    });
    
    // Click validate button
    fireEvent.click(screen.getByText('Validate'));
    
    // Wait for validation error
    await waitFor(() => {
      expect(screen.getByText(/Invalid JSON/)).toBeInTheDocument();
    });
    
    // Enter valid JSON
    fireEvent.change(screen.getByLabelText('JSON Schema'), {
      target: { value: '{"type": "object", "properties": {}}' }
    });
    
    // Click validate button
    fireEvent.click(screen.getByText('Validate'));
    
    // Wait for validation success
    await waitFor(() => {
      expect(screen.getByText('Schema is valid JSON')).toBeInTheDocument();
    });
  });

  test('should successfully save capability when form is valid', async () => {
    renderModal();
    
    // Fill out the form
    fireEvent.change(screen.getByLabelText('Capability Name'), {
      target: { value: 'New Test Capability' }
    });
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Capability'));
    
    // Wait for success message to appear
    await waitFor(() => {
      expect(screen.getByText('Capability saved!')).toBeInTheDocument();
    }, { timeout: 3000 });

    // Verify callback was called
    expect(mockOnCapabilitySaved).toHaveBeenCalledTimes(1);
    expect(mockOnCapabilitySaved).toHaveBeenCalledWith(expect.objectContaining({
      name: 'New Test Capability',
      type: 'text',
      status: 'active',
      serverId: mockServer.id
    }));
  });

  test('should handle file upload for JSON schema', async () => {
    renderModal();
    
    // Create a mock file with valid JSON content
    const validJsonContent = '{"type": "object", "properties": {"name": {"type": "string"}}}';
    const file = new File([validJsonContent], 'schema.json', { type: 'application/json' });
    
    // Mock FileReader
    const originalFileReader = global.FileReader;
    const mockFileReader = {
      readAsText: jest.fn(),
      onload: null
    };
    global.FileReader = jest.fn(() => mockFileReader);
    
    // Trigger file upload
    const fileInput = document.querySelector('input[type="file"]');
    fireEvent.change(fileInput, { target: { files: [file] } });
    
    // Simulate FileReader onload event
    act(() => {
      mockFileReader.onload({ target: { result: validJsonContent } });
    });
    
    // Wait for schema to be updated and validated
    await waitFor(() => {
      expect(screen.getByLabelText('JSON Schema')).toHaveValue(validJsonContent);
    });
    
    // Restore FileReader
    global.FileReader = originalFileReader;
  });
});