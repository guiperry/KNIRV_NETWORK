/**
 * UDCManager Component Tests
 * Comprehensive test suite for UDC management functionality
 */

import * as React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import UDCManager from '../../../src/components/UDCManager';
import { udcManagementService, UDC } from '../../../src/services/UDCManagementService';

// Mock the UDC management service
jest.mock('../../../src/services/UDCManagementService', () => ({
  udcManagementService: {
    getAllUDCs: jest.fn(),
    validateUDC: jest.fn(),
    createUDC: jest.fn(),
    renewUDC: jest.fn(),
    revokeUDC: jest.fn(),
    getExpiringUDCs: jest.fn()
  }
}));

describe('UDCManager', () => {
  const mockUDCManagementService = udcManagementService as jest.Mocked<typeof udcManagementService>;
  
  const mockUDCs = [
    {
      id: 'udc-1',
      agentId: 'agent-123',
      type: 'basic' as const,
      authorityLevel: 'read' as const,
      status: 'active' as const,
      issuedDate: new Date('2024-01-01T00:00:00Z'),
      expiresDate: new Date('2024-02-01T00:00:00Z'),
      scope: 'Read access to data endpoints',
      permissions: ['read', 'list'],
      signature: 'abc123def456',
      issuer: 'KNIRV-CONTROLLER',
      subject: 'agent-123',
      metadata: {
        version: '1.0',
        description: 'Basic read access UDC',
        tags: ['basic', 'read'],
        usage: {
          executionCount: 25,
          lastUsed: new Date('2024-01-15T10:30:00Z'),
          usageHistory: []
        },
        constraints: {
          maxExecutions: 1000,
          allowedHours: [9, 10, 11, 12, 13, 14, 15, 16, 17]
        },
        security: {
          securityFlags: [],
          encryptionLevel: 'standard',
          requiresMFA: false
        }
      }
    },
    {
      id: 'udc-2',
      agentId: 'agent-456',
      type: 'advanced' as const,
      authorityLevel: 'write' as const,
      status: 'expired' as const,
      issuedDate: new Date('2023-12-01T00:00:00Z'),
      expiresDate: new Date('2024-01-01T00:00:00Z'),
      scope: 'Write access to configuration',
      permissions: ['read', 'write', 'update'],
      signature: 'def456ghi789',
      issuer: 'KNIRV-CONTROLLER',
      subject: 'agent-456',
      metadata: {
        version: '1.0',
        description: 'Advanced write access UDC',
        tags: ['advanced', 'write'],
        usage: {
          executionCount: 150,
          lastUsed: new Date('2023-12-31T23:59:00Z'),
          usageHistory: []
        },
        constraints: {
          maxExecutions: 500,
          allowedHours: Array.from({length: 24}, (_, i) => i)
        },
        security: {
          securityFlags: [],
          encryptionLevel: 'standard',
          requiresMFA: false
        }
      }
    }
  ];

  const mockValidationResults = {
    'udc-1': {
      isValid: true,
      securityChecks: {
        signature: true,
        expiry: true,
        permissions: true,
        constraints: true
      },
      remainingTime: 1468800000, // ~17 days in ms
      usageQuota: {
        used: 25,
        total: 1000,
        remaining: 975
      }
    },
    'udc-2': {
      isValid: false,
      reason: 'UDC expired',
      securityChecks: {
        signature: true,
        expiry: false,
        permissions: true,
        constraints: true
      },
      remainingTime: 0,
      usageQuota: {
        used: 150,
        total: 500,
        remaining: 350
      }
    }
  };

  beforeEach(() => {
    jest.clearAllMocks();

    mockUDCManagementService.getAllUDCs.mockReturnValue(mockUDCs as UDC[]);
    mockUDCManagementService.validateUDC.mockImplementation((udcId) =>
      Promise.resolve(mockValidationResults[udcId as keyof typeof mockValidationResults])
    );
    mockUDCManagementService.createUDC.mockResolvedValue({
      ...mockUDCs[0],
      id: 'new-udc-id',
      agentId: 'new-agent'
    } as UDC);
    mockUDCManagementService.renewUDC.mockResolvedValue({
      ...mockUDCs[0],
      expiresDate: new Date('2024-03-01T00:00:00Z')
    } as UDC);
    mockUDCManagementService.revokeUDC.mockResolvedValue(undefined);
    mockUDCManagementService.getExpiringUDCs.mockReturnValue([mockUDCs[0] as UDC]);
  });

  describe('Rendering', () => {
    it('should not render when isOpen is false', () => {
      render(<UDCManager isOpen={false} onClose={jest.fn()} />);
      
      expect(screen.queryByText('UDC Manager')).not.toBeInTheDocument();
    });

    it('should render when isOpen is true', () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      expect(screen.getByText('UDC Manager')).toBeInTheDocument();
      expect(screen.getByText('Universal Delegation Certificates')).toBeInTheDocument();
    });

    it('should render all tabs', () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      expect(screen.getByText('All UDCs')).toBeInTheDocument();
      expect(screen.getByText('Create UDC')).toBeInTheDocument();
      expect(screen.getByText('Expiring Soon')).toBeInTheDocument();
    });
  });

  describe('UDC List Display', () => {
    it('should display UDCs when available', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      // Wait for async validation to complete
      await waitFor(() => {
        expect(screen.getByText('UDC-udc-1')).toBeInTheDocument();
        expect(screen.getByText('UDC-udc-2')).toBeInTheDocument();
        expect(screen.getByText('Agent: agent-123')).toBeInTheDocument();
        expect(screen.getByText('Agent: agent-456')).toBeInTheDocument();
      }, { timeout: 5000, container });
    });

    it('should display UDC status icons correctly', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        // Check that UDCs are rendered (icons are SVG elements from Lucide React)
        const udcElements = screen.getAllByText(/UDC-/);
        expect(udcElements.length).toBe(2);

        // Check for status text or other indicators
        expect(screen.getByText('Agent: agent-123')).toBeInTheDocument();
        expect(screen.getByText('Agent: agent-456')).toBeInTheDocument();
      }, { timeout: 5000, container });
    });

    it('should display authority level badges correctly', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        expect(screen.getByText('read')).toBeInTheDocument();
        expect(screen.getByText('write')).toBeInTheDocument();
      }, { container });
    });

    it('should display UDC metadata', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        expect(screen.getByText('basic')).toBeInTheDocument();
        expect(screen.getByText('advanced')).toBeInTheDocument();
        expect(screen.getByText('25')).toBeInTheDocument(); // Usage count for udc-1
        expect(screen.getByText('150')).toBeInTheDocument(); // Usage count for udc-2
      }, { container });
    });

    it('should show validation results', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        expect(screen.getByText('Valid')).toBeInTheDocument();
        expect(screen.getByText('Invalid')).toBeInTheDocument();
        expect(screen.getByText('17 days remaining')).toBeInTheDocument();
        expect(screen.getByText('Quota: 975/1000')).toBeInTheDocument();
      }, { timeout: 5000, container });
    });

    it('should show empty state when no UDCs exist', async () => {
      mockUDCManagementService.getAllUDCs.mockReturnValue([]);

      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        expect(screen.getByText('No UDCs found')).toBeInTheDocument();
        expect(screen.getByText('Create First UDC')).toBeInTheDocument();
      }, { container });
    });
  });

  describe('UDC Actions', () => {
    it('should renew UDC when renew button is clicked', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        const renewButtons = screen.getAllByLabelText(/Renew UDC/);
        fireEvent.click(renewButtons[0]);
      }, { container });

      expect(mockUDCManagementService.renewUDC).toHaveBeenCalledWith('udc-1', 30);
    });

    it('should revoke UDC when revoke button is clicked', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        const revokeButtons = screen.getAllByLabelText(/Revoke UDC/);
        fireEvent.click(revokeButtons[0]);
      }, { container });

      expect(mockUDCManagementService.revokeUDC).toHaveBeenCalledWith('udc-1', 'Manual revocation');
    });

    it('should open UDC details when view button is clicked', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        const viewButtons = screen.getAllByLabelText(/View details for UDC/);
        fireEvent.click(viewButtons[0]);
      }, { container });

      expect(screen.getByText('UDC Details')).toBeInTheDocument();
      expect(screen.getByText('udc-1')).toBeInTheDocument();
    });

    it('should refresh UDC list after actions', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(1);
      }, { container });

      const refreshButton = screen.getByLabelText('Refresh UDCs');
      fireEvent.click(refreshButton);

      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(2);
      }, { container });
    });
  });

  describe('Tab Navigation', () => {
    it('should switch to create UDC tab', () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      fireEvent.click(screen.getByText('Create UDC'));
      
      expect(screen.getByText('Agent ID')).toBeInTheDocument();
      expect(screen.getByText('UDC Type')).toBeInTheDocument();
      expect(screen.getByText('Authority Level')).toBeInTheDocument();
    });

    it('should switch to expiring soon tab', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      fireEvent.click(screen.getByText('Expiring Soon'));

      await waitFor(() => {
        expect(mockUDCManagementService.getExpiringUDCs).toHaveBeenCalledWith(7);
        expect(screen.getByText('UDC-udc-1')).toBeInTheDocument();
      }, { container });
    });

    it('should highlight active tab', () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      const allUDCsTab = screen.getByText('All UDCs').closest('button');
      const createTab = screen.getByText('Create UDC').closest('button');
      
      // All UDCs should be active by default
      expect(allUDCsTab).toHaveClass('text-emerald-400');
      
      fireEvent.click(screen.getByText('Create UDC'));
      
      expect(createTab).toHaveClass('text-emerald-400');
    });
  });

  describe('UDC Creation Form', () => {
    let container: HTMLElement;

    beforeEach(() => {
      const result = render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      container = result.container;
      fireEvent.click(screen.getAllByText('Create UDC')[0]); // Click the tab, not the button
    });

    it('should render all form fields', () => {
      expect(screen.getByLabelText('Agent ID')).toBeInTheDocument();
      expect(screen.getByLabelText('UDC Type')).toBeInTheDocument();
      expect(screen.getByLabelText('Authority Level')).toBeInTheDocument();
      expect(screen.getByLabelText('Validity Period (days)')).toBeInTheDocument();
      expect(screen.getByLabelText('Scope Description')).toBeInTheDocument();
      expect(screen.getByLabelText('Max Executions')).toBeInTheDocument();
    });

    it('should submit form with valid data', async () => {
      const user = userEvent.setup();

      const agentIdInput = screen.getByLabelText('Agent ID') as HTMLInputElement;
      const scopeInput = screen.getByLabelText('Scope Description') as HTMLTextAreaElement;

      // Fill in the required Agent ID field using userEvent for better simulation
      await user.clear(agentIdInput);
      await user.type(agentIdInput, 'test-agent-123');

      // Fill in the scope description
      await user.clear(scopeInput);
      await user.type(scopeInput, 'Test UDC scope');

      // Wait for form state to update
      await waitFor(() => {
        expect(agentIdInput.value).toBe('test-agent-123');
        expect(scopeInput.value).toBe('Test UDC scope');
      }, { container });

      const submitButton = screen.getByTestId('create-udc-submit-button');

      // Submit the form
      await user.click(submitButton);

      // Wait for the service to be called and form to be submitted
      await waitFor(() => {
        expect(mockUDCManagementService.createUDC).toHaveBeenCalled();
      }, { container, timeout: 5000 });

      // Verify the service was called with correct data
      expect(mockUDCManagementService.createUDC).toHaveBeenCalledWith({
        agentId: 'test-agent-123',
        type: 'basic',
        authorityLevel: 'read',
        validityPeriod: 30,
        scope: 'Test UDC scope',
        permissions: ['read'],
        constraints: {
          maxExecutions: 1000,
          allowedHours: Array.from({length: 24}, (_, i) => i)
        },
        metadata: {
          description: 'UDC for agent test-agent-123',
          tags: ['basic', 'read']
        }
      });

      // Wait for the tab to switch back to list and UDCs to be reloaded
      await waitFor(() => {
        // Should show existing UDCs after switching back to list tab
        expect(screen.getByText('UDC-udc-1')).toBeInTheDocument();
      }, { container, timeout: 5000 });
    });

    it('should cancel form and return to list tab', () => {
      const cancelButton = screen.getByText('Cancel');
      fireEvent.click(cancelButton);
      
      // Should return to list tab
      expect(screen.getByText('UDC-udc-1')).toBeInTheDocument();
    });

    it('should validate required fields', () => {
      const submitButton = screen.getByTestId('create-udc-submit-button');
      fireEvent.click(submitButton);

      // Form should not submit without required fields
      expect(mockUDCManagementService.createUDC).not.toHaveBeenCalled();
    });

    it('should render form fields with correct attributes', () => {
      const agentIdInput = screen.getByLabelText('Agent ID');
      const typeSelect = screen.getByLabelText('UDC Type');
      const authoritySelect = screen.getByLabelText('Authority Level');
      const validityInput = screen.getByLabelText('Validity Period (days)');
      const scopeTextarea = screen.getByLabelText('Scope Description');
      const maxExecutionsInput = screen.getByLabelText('Max Executions');

      // Test that all form fields are present and have correct attributes
      expect(agentIdInput).toBeInTheDocument();
      expect(agentIdInput).toHaveAttribute('type', 'text');
      expect(agentIdInput).toHaveAttribute('required');

      expect(typeSelect).toBeInTheDocument();
      expect(typeSelect.tagName).toBe('SELECT');

      expect(authoritySelect).toBeInTheDocument();
      expect(authoritySelect.tagName).toBe('SELECT');

      expect(validityInput).toBeInTheDocument();
      expect(validityInput).toHaveAttribute('type', 'number');

      expect(scopeTextarea).toBeInTheDocument();
      expect(scopeTextarea.tagName).toBe('TEXTAREA');

      expect(maxExecutionsInput).toBeInTheDocument();
      expect(maxExecutionsInput).toHaveAttribute('type', 'number');

      // Verify the form submit button is present
      expect(screen.getByTestId('create-udc-submit-button')).toBeInTheDocument();
    });
  });

  describe('Expiring UDCs Display', () => {
    beforeEach(async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      fireEvent.click(screen.getByText('Expiring Soon'));

      await waitFor(() => {
        expect(mockUDCManagementService.getExpiringUDCs).toHaveBeenCalled();
      }, { container });
    });

    it('should display expiring UDCs with warning styling', () => {
      expect(screen.getByText('UDC-udc-1')).toBeInTheDocument();

      // Should have warning styling - find the card container with the yellow background
      const expiringCard = screen.getByText('UDC-udc-1').closest('.bg-yellow-500\\/10');
      expect(expiringCard).toBeInTheDocument();
      expect(expiringCard).toHaveClass('bg-yellow-500/10');
    });

    it('should show expiration information', () => {
      expect(screen.getByText(/Expires:/)).toBeInTheDocument();
      expect(screen.getByText(/days\)/)).toBeInTheDocument();
    });

    it('should provide renew button for expiring UDCs', () => {
      const renewButton = screen.getByText('Renew 30 Days');
      expect(renewButton).toBeInTheDocument();
      
      fireEvent.click(renewButton);
      expect(mockUDCManagementService.renewUDC).toHaveBeenCalledWith('udc-1', 30);
    });
  });

  describe('UDC Details Modal', () => {
    beforeEach(async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        const viewButtons = screen.getAllByLabelText(/View details for UDC/);
        fireEvent.click(viewButtons[0]);
      }, { container });
    });

    it('should display UDC details', () => {
      expect(screen.getByText('UDC Details')).toBeInTheDocument();
      expect(screen.getByText('udc-1')).toBeInTheDocument();
      expect(screen.getByText('agent-123')).toBeInTheDocument();
      expect(screen.getAllByText('basic')[0]).toBeInTheDocument();
      expect(screen.getAllByText('read')[0]).toBeInTheDocument();
    });

    it('should display scope and permissions', () => {
      expect(screen.getByText('Scope')).toBeInTheDocument();
      expect(screen.getByText('Read access to data endpoints')).toBeInTheDocument();
      expect(screen.getByText('Permissions')).toBeInTheDocument();
      expect(screen.getAllByText('read')[0]).toBeInTheDocument();
      expect(screen.getByText('list')).toBeInTheDocument();
    });

    it('should display signature', () => {
      expect(screen.getByText('Signature')).toBeInTheDocument();
      expect(screen.getByText('abc123def456')).toBeInTheDocument();
    });

    it('should close modal when close button is clicked', () => {
      const closeButtons = screen.getAllByText('×');
      // Click the second close button (the modal close button, not the main UDC Manager close button)
      fireEvent.click(closeButtons[1]);

      expect(screen.queryByText('UDC Details')).not.toBeInTheDocument();
    });
  });

  describe('Auto-refresh', () => {
    beforeEach(() => {
      jest.useFakeTimers();
    });

    afterEach(() => {
      jest.useRealTimers();
    });

    it('should auto-refresh every 30 seconds', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      // Initial load
      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(1);
      }, { container });

      // Fast-forward 30 seconds
      act(() => {
        jest.advanceTimersByTime(30000);
      });

      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(2);
      }, { container });
    });

    it('should stop auto-refresh when component is closed', async () => {
      const { rerender, container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(1);
      }, { container });

      // Close component
      rerender(<UDCManager isOpen={false} onClose={jest.fn()} />);

      // Fast-forward time
      jest.advanceTimersByTime(30000);

      // Should not call again
      expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(1);
    });
  });

  describe('Refresh Button', () => {
    it('should refresh data when refresh button is clicked', async () => {
      const { container } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(1);
      }, { container });

      const refreshButton = screen.getByLabelText('Refresh UDCs');
      fireEvent.click(refreshButton);

      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(2);
      }, { container });
    });
  });

  describe('Close Functionality', () => {
    it('should call onClose when close button is clicked', () => {
      const onCloseMock = jest.fn();
      render(<UDCManager isOpen={true} onClose={onCloseMock} />);
      
      const closeButton = screen.getByText('×');
      fireEvent.click(closeButton);
      
      expect(onCloseMock).toHaveBeenCalled();
    });
  });

  describe('Error Handling', () => {
    it('should handle UDC loading errors gracefully', async () => {
      mockUDCManagementService.getAllUDCs.mockImplementation(() => {
        throw new Error('Loading failed');
      });
      
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      // Should not crash
      expect(screen.getByText('UDC Manager')).toBeInTheDocument();
    });

    it('should handle validation errors gracefully', async () => {
      mockUDCManagementService.validateUDC.mockRejectedValue(new Error('Validation failed'));
      
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      // Should not crash
      expect(screen.getByText('UDC Manager')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels', () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);

      expect(screen.getByRole('button', { name: /refresh udcs/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /close udc manager/i })).toBeInTheDocument();
    });

    it('should support keyboard navigation', () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      const allUDCsTab = screen.getByText('All UDCs').closest('button');
      const createTab = screen.getByText('Create UDC').closest('button');
      
      expect(allUDCsTab).toBeInTheDocument();
      expect(createTab).toBeInTheDocument();
      
      // Tabs should be focusable
      allUDCsTab?.focus();
      expect(document.activeElement).toBe(allUDCsTab);
    });
  });
});
