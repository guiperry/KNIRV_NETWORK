import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import TargetSystemModal from '../components/modals/TargetSystemModal';

describe('TargetSystemModal', () => {
  const mockOnClose = jest.fn();
  const mockOnTargetSaved = jest.fn();
  
  const mockTarget = {
    id: 1,
    name: 'Test Target',
    type: 'browser',
    description: 'Test description',
    capabilities: ['Web Scraping', 'DOM Analysis'],
    permissions: ['read', 'write'],
    security: 'medium',
    connectionMethod: 'Test Connection Method',
    dataAccess: ['Test Data Access']
  };
  
  const renderModal = (isOpen = true, target = null) => {
    return render(
      <TargetSystemModal 
        isOpen={isOpen} 
        onClose={mockOnClose} 
        target={target} 
        onTargetSaved={mockOnTargetSaved} 
      />
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should not render when isOpen is false', () => {
    renderModal(false);
    expect(screen.queryByText('Add Target System')).not.toBeInTheDocument();
    expect(screen.queryByText('Edit Target System')).not.toBeInTheDocument();
  });

  test('should render "Add Target System" when creating a new target', () => {
    renderModal(true, null);
    expect(screen.getByText('Add Target System')).toBeInTheDocument();
  });

  test('should render "Edit Target System" when editing an existing target', () => {
    renderModal(true, mockTarget);
    expect(screen.getByText('Edit Target System')).toBeInTheDocument();
  });

  test('should populate form fields when editing an existing target', () => {
    renderModal(true, mockTarget);
    expect(screen.getByLabelText('Target Name')).toHaveValue(mockTarget.name);
    expect(screen.getByLabelText('Description')).toHaveValue(mockTarget.description);
    expect(screen.getByLabelText('Connection Method')).toHaveValue(mockTarget.connectionMethod);
    
    // Check if the correct target type is selected
    const browserTypeElement = screen.getByText('Web Browser');
    const browserContainer = browserTypeElement.closest('div[class*="bg-gradient-to-r"]');
    expect(browserContainer).toHaveClass('bg-gradient-to-r');

    // Check if capabilities are selected
    mockTarget.capabilities.forEach(capability => {
      const capabilityElement = screen.getByText(capability);
      const capabilityContainer = capabilityElement.closest('div[class*="bg-gradient-to-r"]');
      expect(capabilityContainer).toHaveClass('bg-gradient-to-r');
    });

    // Check if permissions are selected
    mockTarget.permissions.forEach(permission => {
      const permissionElement = screen.getByText(permission);
      const permissionContainer = permissionElement.closest('div[class*="bg-gradient-to-r"]');
      expect(permissionContainer).toHaveClass('bg-gradient-to-r');
    });
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
    fireEvent.change(screen.getByLabelText('Target Name'), {
      target: { value: '' }
    });
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Target System'));
    
    // Wait for validation errors
    await waitFor(() => {
      expect(screen.getByText('Target name is required')).toBeInTheDocument();
      expect(screen.getByText('Select at least one capability')).toBeInTheDocument();
    });
    
    // Ensure onTargetSaved was not called
    expect(mockOnTargetSaved).not.toHaveBeenCalled();
  });

  test('should successfully save target when form is valid', async () => {
    renderModal();
    
    // Fill out the form
    fireEvent.change(screen.getByLabelText('Target Name'), {
      target: { value: 'New Test Target' }
    });
    
    // Select a target type (browser is already selected by default)
    
    // Select a capability
    const capabilities = screen.getAllByText(/Web Scraping|DOM Analysis/);
    fireEvent.click(capabilities[0]);
    
    // Select a permission (read is already selected by default)
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Target System'));
    
    // Wait for success message and callback
    await waitFor(() => {
      expect(screen.getByText('Target system saved!')).toBeInTheDocument();
    }, { timeout: 3000 });

    expect(mockOnTargetSaved).toHaveBeenCalledTimes(1);
    expect(mockOnTargetSaved).toHaveBeenCalledWith(expect.objectContaining({
      name: 'New Test Target',
      type: 'browser',
      status: 'connected'
    }));
  });

  test('should test connection successfully', async () => {
    // Mock Math.random to always return 0.5 (success case)
    const originalRandom = Math.random;
    Math.random = jest.fn(() => 0.5);
    
    renderModal();
    
    // Fill required fields
    fireEvent.change(screen.getByLabelText('Target Name'), {
      target: { value: 'Test Target' }
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
    fireEvent.change(screen.getByLabelText('Target Name'), {
      target: { value: 'Test Target' }
    });
    
    // Click test connection button
    fireEvent.click(screen.getByText('Test Connection'));
    
    // Wait for testing state
    expect(screen.getByText('Testing Connection...')).toBeInTheDocument();
    
    // Wait for error message
    await waitFor(() => {
      expect(screen.getByText('Connection failed!')).toBeInTheDocument();
    }, { timeout: 3000 });
    
    // Restore Math.random
    Math.random = originalRandom;
  });

  test('should update capabilities when target type changes', () => {
    renderModal();
    
    // Initially browser type is selected, check for browser capabilities
    expect(screen.getByText('Web Scraping')).toBeInTheDocument();
    
    // Change target type to filesystem
    const filesystemType = screen.getByText('File System');
    fireEvent.click(filesystemType);
    
    // Check that browser capabilities are no longer shown and filesystem capabilities are shown
    expect(screen.queryByText('Web Scraping')).not.toBeInTheDocument();
    expect(screen.getByText('File Analysis')).toBeInTheDocument();
  });
});