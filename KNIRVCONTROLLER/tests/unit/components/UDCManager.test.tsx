/**
 * UDCManager Component Tests
 * Comprehensive test suite for UDC management functionality
 */

import * as React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
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
    jest.useFakeTimers();
    
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

  afterEach(() => {
    jest.useRealTimers();
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
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('UDC-udc-1')).toBeInTheDocument();
        expect(screen.getByText('UDC-udc-2')).toBeInTheDocument();
        expect(screen.getByText('Agent: agent-123')).toBeInTheDocument();
        expect(screen.getByText('Agent: agent-456')).toBeInTheDocument();
      });
    });

    it('should display UDC status icons correctly', async () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        // Should show active and expired status icons
        const activeIcons = screen.getAllByTestId('check-circle-icon');
        const expiredIcons = screen.getAllByTestId('x-circle-icon');
        
        expect(activeIcons.length).toBeGreaterThan(0);
        expect(expiredIcons.length).toBeGreaterThan(0);
      });
    });

    it('should display authority level badges correctly', async () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('read')).toBeInTheDocument();
        expect(screen.getByText('write')).toBeInTheDocument();
      });
    });

    it('should display UDC metadata', async () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('basic')).toBeInTheDocument();
        expect(screen.getByText('advanced')).toBeInTheDocument();
        expect(screen.getByText('25')).toBeInTheDocument(); // Usage count for udc-1
        expect(screen.getByText('150')).toBeInTheDocument(); // Usage count for udc-2
      });
    });

    it('should show validation results', async () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('Valid')).toBeInTheDocument();
        expect(screen.getByText('Invalid')).toBeInTheDocument();
        expect(screen.getByText('17 days remaining')).toBeInTheDocument();
        expect(screen.getByText('Quota: 975/1000')).toBeInTheDocument();
      });
    });

    it('should show empty state when no UDCs exist', async () => {
      mockUDCManagementService.getAllUDCs.mockReturnValue([]);
      
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('No UDCs found')).toBeInTheDocument();
        expect(screen.getByText('Create First UDC')).toBeInTheDocument();
      });
    });
  });

  describe('UDC Actions', () => {
    it('should renew UDC when renew button is clicked', async () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const renewButtons = screen.getAllByTestId('refresh-cw-icon');
        fireEvent.click(renewButtons[0]);
      });
      
      expect(mockUDCManagementService.renewUDC).toHaveBeenCalledWith('udc-1', 30);
    });

    it('should revoke UDC when revoke button is clicked', async () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const revokeButtons = screen.getAllByTestId('trash-2-icon');
        fireEvent.click(revokeButtons[0]);
      });
      
      expect(mockUDCManagementService.revokeUDC).toHaveBeenCalledWith('udc-1', 'Manual revocation');
    });

    it('should open UDC details when view button is clicked', async () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const viewButtons = screen.getAllByTestId('eye-icon');
        fireEvent.click(viewButtons[0]);
      });
      
      expect(screen.getByText('UDC Details')).toBeInTheDocument();
      expect(screen.getByText('udc-1')).toBeInTheDocument();
    });

    it('should refresh UDC list after actions', async () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(1);
      });
      
      const renewButtons = screen.getAllByTestId('refresh-cw-icon');
      fireEvent.click(renewButtons[0]);
      
      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(2);
      });
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
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      fireEvent.click(screen.getByText('Expiring Soon'));
      
      await waitFor(() => {
        expect(mockUDCManagementService.getExpiringUDCs).toHaveBeenCalledWith(7);
        expect(screen.getByText('UDC-udc-1')).toBeInTheDocument();
      });
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
    beforeEach(() => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      fireEvent.click(screen.getByText('Create UDC'));
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
      fireEvent.change(screen.getByLabelText('Agent ID'), { 
        target: { value: 'test-agent-123' } 
      });
      fireEvent.change(screen.getByLabelText('Scope Description'), { 
        target: { value: 'Test UDC scope' } 
      });
      
      const submitButton = screen.getByText('Create UDC');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockUDCManagementService.createUDC).toHaveBeenCalledWith(
          expect.objectContaining({
            agentId: 'test-agent-123',
            scope: 'Test UDC scope'
          })
        );
      });
    });

    it('should cancel form and return to list tab', () => {
      const cancelButton = screen.getByText('Cancel');
      fireEvent.click(cancelButton);
      
      // Should return to list tab
      expect(screen.getByText('UDC-udc-1')).toBeInTheDocument();
    });

    it('should validate required fields', () => {
      const submitButton = screen.getByText('Create UDC');
      fireEvent.click(submitButton);
      
      // Form should not submit without required fields
      expect(mockUDCManagementService.createUDC).not.toHaveBeenCalled();
    });

    it('should update form fields correctly', () => {
      const agentIdInput = screen.getByLabelText('Agent ID');
      const typeSelect = screen.getByLabelText('UDC Type');
      const authoritySelect = screen.getByLabelText('Authority Level');
      
      fireEvent.change(agentIdInput, { target: { value: 'new-agent' } });
      fireEvent.change(typeSelect, { target: { value: 'advanced' } });
      fireEvent.change(authoritySelect, { target: { value: 'write' } });
      
      expect(agentIdInput).toHaveValue('new-agent');
      expect(typeSelect).toHaveValue('advanced');
      expect(authoritySelect).toHaveValue('write');
    });
  });

  describe('Expiring UDCs Display', () => {
    beforeEach(async () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      fireEvent.click(screen.getByText('Expiring Soon'));
      
      await waitFor(() => {
        expect(mockUDCManagementService.getExpiringUDCs).toHaveBeenCalled();
      });
    });

    it('should display expiring UDCs with warning styling', () => {
      expect(screen.getByText('UDC-udc-1')).toBeInTheDocument();
      
      // Should have warning styling
      const expiringCard = screen.getByText('UDC-udc-1').closest('div');
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
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const viewButtons = screen.getAllByTestId('eye-icon');
        fireEvent.click(viewButtons[0]);
      });
    });

    it('should display UDC details', () => {
      expect(screen.getByText('UDC Details')).toBeInTheDocument();
      expect(screen.getByText('udc-1')).toBeInTheDocument();
      expect(screen.getByText('agent-123')).toBeInTheDocument();
      expect(screen.getByText('basic')).toBeInTheDocument();
      expect(screen.getByText('read')).toBeInTheDocument();
    });

    it('should display scope and permissions', () => {
      expect(screen.getByText('Scope')).toBeInTheDocument();
      expect(screen.getByText('Read access to data endpoints')).toBeInTheDocument();
      expect(screen.getByText('Permissions')).toBeInTheDocument();
      expect(screen.getByText('read')).toBeInTheDocument();
      expect(screen.getByText('list')).toBeInTheDocument();
    });

    it('should display signature', () => {
      expect(screen.getByText('Signature')).toBeInTheDocument();
      expect(screen.getByText('abc123def456')).toBeInTheDocument();
    });

    it('should close modal when close button is clicked', () => {
      const closeButton = screen.getByText('×');
      fireEvent.click(closeButton);
      
      expect(screen.queryByText('UDC Details')).not.toBeInTheDocument();
    });
  });

  describe('Auto-refresh', () => {
    it('should auto-refresh every 30 seconds', async () => {
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      // Initial load
      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(1);
      });
      
      // Fast-forward 30 seconds
      jest.advanceTimersByTime(30000);
      
      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(2);
      });
    });

    it('should stop auto-refresh when component is closed', async () => {
      const { rerender } = render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(1);
      });
      
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
      render(<UDCManager isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(1);
      });
      
      const refreshButton = screen.getByRole('button', { name: /refresh/i });
      fireEvent.click(refreshButton);
      
      await waitFor(() => {
        expect(mockUDCManagementService.getAllUDCs).toHaveBeenCalledTimes(2);
      });
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
      
      expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
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
