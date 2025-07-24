import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import WebConnectionModal from '../../components/modals/WebConnectionModal';

describe('WebConnectionModal', () => {
  const mockOnClose = jest.fn();
  const mockOnConnectionSaved = jest.fn();
  
  const mockConnection = {
    id: 1,
    name: 'Test Connection',
    type: 'oauth',
    provider: 'Google',
    description: 'Test description',
    scopes: ['https://www.googleapis.com/auth/drive.readonly'],
    connectionDetails: {
      clientId: 'test-client-id',
      clientSecret: 'test-client-secret',
      redirectUri: 'https://app.picaos.com/authkit/callback',
      tokenEndpoint: 'https://oauth2.googleapis.com/token',
      authEndpoint: 'https://accounts.google.com/o/oauth2/auth'
    }
  };
  
  const renderModal = (isOpen = true, connection = null) => {
    return render(
      <WebConnectionModal 
        isOpen={isOpen} 
        onClose={mockOnClose} 
        connection={connection} 
        onConnectionSaved={mockOnConnectionSaved} 
      />
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should not render when isOpen is false', () => {
    renderModal(false);
    expect(screen.queryByText('Add Connection')).not.toBeInTheDocument();
    expect(screen.queryByText('Edit Connection')).not.toBeInTheDocument();
  });

  test('should render "Add Connection" when creating a new connection', () => {
    renderModal(true, null);
    expect(screen.getByText('Add Connection')).toBeInTheDocument();
  });

  test('should render "Edit Connection" when editing an existing connection', () => {
    renderModal(true, mockConnection);
    expect(screen.getByText('Edit Connection')).toBeInTheDocument();
  });

  test('should populate form fields when editing an existing connection', () => {
    renderModal(true, mockConnection);
    expect(screen.getByLabelText('Connection Name')).toHaveValue(mockConnection.name);
    expect(screen.getByLabelText('Description')).toHaveValue(mockConnection.description);
    expect(screen.getByLabelText('Client ID')).toHaveValue(mockConnection.connectionDetails.clientId);
    expect(screen.getByLabelText('Client Secret')).toHaveValue(mockConnection.connectionDetails.clientSecret);
    
    // Check if the correct connection type is selected
    const oauthTypeElement = screen.getByText('OAuth 2.0');
    // The gradient class is on the parent div that contains the connection type
    const connectionTypeContainer = oauthTypeElement.closest('div').parentElement;
    expect(connectionTypeContainer).toHaveClass('bg-gradient-to-r');
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
    fireEvent.change(screen.getByLabelText('Connection Name'), {
      target: { value: '' }
    });
    
    fireEvent.change(screen.getByLabelText('Provider'), {
      target: { value: '' }
    });
    
    fireEvent.change(screen.getByLabelText('Description'), {
      target: { value: '' }
    });
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Connection'));
    
    // Wait for validation errors
    await waitFor(() => {
      expect(screen.getByText('Connection name is required')).toBeInTheDocument();
      expect(screen.getByText('Provider is required')).toBeInTheDocument();
      expect(screen.getByText('Description is required')).toBeInTheDocument();
      expect(screen.getByText('At least one scope is required')).toBeInTheDocument();
    });
    
    // Ensure onConnectionSaved was not called
    expect(mockOnConnectionSaved).not.toHaveBeenCalled();
  });

  test('should successfully save connection when form is valid', async () => {
    renderModal();
    
    // Fill out the form
    fireEvent.change(screen.getByLabelText('Connection Name'), {
      target: { value: 'New Test Connection' }
    });
    
    fireEvent.change(screen.getByLabelText('Provider'), {
      target: { value: 'Test Provider' }
    });
    
    fireEvent.change(screen.getByLabelText('Description'), {
      target: { value: 'Test description' }
    });
    
    // Add a scope
    fireEvent.change(screen.getByLabelText('Add Scopes'), {
      target: { value: 'test:scope' }
    });
    fireEvent.click(screen.getByText('Add'));
    
    // Fill out OAuth details
    fireEvent.change(screen.getByLabelText('Client ID'), {
      target: { value: 'test-client-id' }
    });
    
    fireEvent.change(screen.getByLabelText('Client Secret'), {
      target: { value: 'test-client-secret' }
    });
    
    fireEvent.change(screen.getByLabelText('Authorization Endpoint'), {
      target: { value: 'https://example.com/auth' }
    });
    
    fireEvent.change(screen.getByLabelText('Token Endpoint'), {
      target: { value: 'https://example.com/token' }
    });
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Connection'));
    
    // Wait for success message and callback
    await waitFor(() => {
      expect(screen.getByText('Connection saved!')).toBeInTheDocument();
      expect(mockOnConnectionSaved).toHaveBeenCalledTimes(1);
      expect(mockOnConnectionSaved).toHaveBeenCalledWith(expect.objectContaining({
        name: 'New Test Connection',
        provider: 'Test Provider',
        type: 'oauth'
      }));
    }, { timeout: 3000 });
  });

  test('should toggle between OAuth and API Key types', () => {
    renderModal();
    
    // Default should be OAuth
    expect(screen.getByLabelText('Client ID')).toBeInTheDocument();
    
    // Switch to API Key
    const apiKeyTypeElement = screen.getByText('API Key');
    fireEvent.click(apiKeyTypeElement);
    
    // Should now show API Key fields
    expect(screen.getByLabelText('API Key')).toBeInTheDocument();
    expect(screen.queryByLabelText('Client ID')).not.toBeInTheDocument();
    
    // Switch back to OAuth
    const oauthTypeElement = screen.getByText('OAuth 2.0');
    fireEvent.click(oauthTypeElement);
    
    // Should now show OAuth fields again
    expect(screen.getByLabelText('Client ID')).toBeInTheDocument();
    expect(screen.queryByLabelText('API Key')).not.toBeInTheDocument();
  });

  test('should toggle secret visibility', () => {
    renderModal(true, mockConnection);
    
    // Client Secret should be hidden initially
    const clientSecretInput = screen.getByLabelText('Client Secret');
    expect(clientSecretInput).toHaveAttribute('type', 'password');
    
    // Click show button
    const showButtons = screen.getAllByRole('button', { name: '' });
    const showSecretButton = showButtons.find(button => 
      button.closest('div')?.previousElementSibling?.textContent === 'Client Secret'
    );
    fireEvent.click(showSecretButton);
    
    // Client Secret should now be visible
    expect(clientSecretInput).toHaveAttribute('type', 'text');
  });

  test('should add and remove scopes', () => {
    renderModal();
    
    // Add a scope
    fireEvent.change(screen.getByLabelText('Add Scopes'), {
      target: { value: 'test:scope' }
    });
    fireEvent.click(screen.getByText('Add'));
    
    // Scope should be added
    expect(screen.getByText('test:scope')).toBeInTheDocument();
    
    // Add another scope
    fireEvent.change(screen.getByLabelText('Add Scopes'), {
      target: { value: 'another:scope' }
    });
    fireEvent.click(screen.getByText('Add'));
    
    // Both scopes should be present
    expect(screen.getByText('test:scope')).toBeInTheDocument();
    expect(screen.getByText('another:scope')).toBeInTheDocument();
    
    // Remove a scope by finding the X button within the first scope element
    const testScopeElement = screen.getByText('test:scope');
    const removeButton = testScopeElement.parentElement.querySelector('button');
    fireEvent.click(removeButton);
    
    // Scope should be removed
    expect(screen.queryByText('test:scope')).not.toBeInTheDocument();
    expect(screen.getByText('another:scope')).toBeInTheDocument();
  });

  test('should pre-fill endpoints for known providers', () => {
    renderModal();
    
    // Click on Google provider
    const googleProvider = screen.getByAltText('Google');
    fireEvent.click(googleProvider);
    
    // Should pre-fill Google endpoints
    expect(screen.getByLabelText('Authorization Endpoint')).toHaveValue('https://accounts.google.com/o/oauth2/auth');
    expect(screen.getByLabelText('Token Endpoint')).toHaveValue('https://oauth2.googleapis.com/token');
    
    // Click on GitHub provider
    const githubProvider = screen.getByAltText('GitHub');
    fireEvent.click(githubProvider);
    
    // Should pre-fill GitHub endpoints
    expect(screen.getByLabelText('Authorization Endpoint')).toHaveValue('https://github.com/login/oauth/authorize');
    expect(screen.getByLabelText('Token Endpoint')).toHaveValue('https://github.com/login/oauth/access_token');
  });

  test('should show AWS-specific fields when AWS is selected with API Key type', () => {
    renderModal();
    
    // Switch to API Key
    const apiKeyTypeElement = screen.getByText('API Key');
    fireEvent.click(apiKeyTypeElement);
    
    // Select AWS provider
    const awsProvider = screen.getByAltText('AWS');
    fireEvent.click(awsProvider);
    
    // Should show AWS-specific fields
    expect(screen.getByLabelText('Access Key ID')).toBeInTheDocument();
    expect(screen.getByLabelText('Secret Access Key')).toBeInTheDocument();
    expect(screen.getByLabelText('Region')).toBeInTheDocument();
    expect(screen.getByLabelText('S3 Bucket (optional)')).toBeInTheDocument();
  });
});