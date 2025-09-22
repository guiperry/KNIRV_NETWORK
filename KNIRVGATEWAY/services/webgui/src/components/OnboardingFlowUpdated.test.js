/**
 * @jest-environment jsdom
 */

import React from 'react'; // Import React here
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import OnboardingFlowUpdated from './OnboardingFlowUpdated';

// Mock the useRouter hook from next/router
jest.mock('next/router', () => ({
  useRouter: jest.fn(),
}));

describe('OnboardingFlowUpdated', () => {
  beforeEach(() => {
    // Reset the mock before each test
    jest.clearAllMocks();
  });

  it('should start at step 1', () => {
    render(<OnboardingFlowUpdated />);
    const step1Indicator = screen.getByText('1');
    expect(step1Indicator).toHaveClass('active');
  });

  it('should move to step 2 when clicking "Continue to Minting"', async () => {
    render(<OnboardingFlowUpdated />);
    // Add items to the mint pool
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    fireEvent.click(screen.getByText('Continue to Minting'));

    await waitFor(() => {
      const step2Indicator = screen.getByText('2');
      expect(step2Indicator).toHaveClass('active');
    });
  });

  it('should move to step 3 after minting', async () => {
    render(<OnboardingFlowUpdated />);
    // Add items to the mint pool
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    fireEvent.click(screen.getByText('Continue to Minting'));
    fireEvent.click(screen.getByText('Mint Inventory Contract'));

    await waitFor(() => {
      const step3Indicator = screen.getByText('✓');
      expect(step3Indicator).toBeInTheDocument();
    });
  });

  it('should change input method to upload when clicking "Upload Spreadsheet"', () => {
    render(<OnboardingFlowUpdated />);
    fireEvent.click(screen.getByText('Upload Spreadsheet'));
    expect(screen.getByText('Drag and drop your inventory spreadsheet here')).toBeInTheDocument();
  });

  it('should change input method to form when clicking "Manual Entry Form"', () => {
    render(<OnboardingFlowUpdated />);
    fireEvent.click(screen.getByText('Manual Entry Form'));
    expect(screen.getByLabelText('Part Number*')).toBeInTheDocument();
  });

  it('should add a new item to formItems when clicking "Add Item to Inventory"', () => {
    render(<OnboardingFlowUpdated />);
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    expect(screen.getByText('ABC-123')).toBeInTheDocument();
  });

  it('should update totalValue when adding items via form', () => {
    render(<OnboardingFlowUpdated />);
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    expect(screen.getByText('Total Value: $150000.00')).toBeInTheDocument();
  });

  it('should update totalValue when adding items via spreadsheet', async () => {
    render(<OnboardingFlowUpdated />);
    const file = new File(['(⌐□_□)'], 'inventory.xlsx', { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
    const input = screen.getByLabelText('Supported formats: .xlsx, .csv');
    fireEvent.change(input, { target: { files: [file] } });
    await waitFor(() => {
      expect(screen.getByText('Total Value: $1060.00')).toBeInTheDocument();
    });
  });

  it('should add items to mintPool when clicking "Add to Mint Pool"', async () => {
    render(<OnboardingFlowUpdated />);
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    await waitFor(() => {
      expect(screen.getByText('Items ready to be minted: 1')).toBeInTheDocument();
    });
  });

  it('should display the completion message after minting', async () => {
    render(<OnboardingFlowUpdated />);
    // Add items to the mint pool
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    fireEvent.click(screen.getByText('Continue to Minting'));
    fireEvent.click(screen.getByText('Mint Inventory Contract'));

    await waitFor(() => {
      expect(screen.getByText('Inventory Contract Minted!')).toBeInTheDocument();
    });
  });

  it('should display the transaction details after minting', async () => {
    render(<OnboardingFlowUpdated />);
    // Add items to the mint pool
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    fireEvent.click(screen.getByText('Continue to Minting'));
    fireEvent.click(screen.getByText('Mint Inventory Contract'));

    await waitFor(() => {
      expect(screen.getByText('Transaction Hash:')).toBeInTheDocument();
      expect(screen.getByText('Contract Address:')).toBeInTheDocument();
      expect(screen.getByText('Gas Used:')).toBeInTheDocument();
    });
  });

  it('should set the token name based on the first item', async () => {
    render(<OnboardingFlowUpdated />);
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    fireEvent.click(screen.getByText('Continue to Minting'));
    await waitFor(() => {
      expect(screen.getByText('Token Name:')).toBeInTheDocument();
      expect(screen.getByText('Sac ABC-123 #1')).toBeInTheDocument();
    });
  });

  it('should update the token name when the first item is removed', async () => {
    render(<OnboardingFlowUpdated />);
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'DEF-456' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '200' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '2000' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getAllByText('Remove')[0]);
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    fireEvent.click(screen.getByText('Continue to Minting'));
    await waitFor(() => {
      expect(screen.getByText('Token Name:')).toBeInTheDocument();
      expect(screen.getByText('Sac DEF-456 #1')).toBeInTheDocument();
    });
  });

  it('should display the empty state when the mint pool is empty', async () => {
    render(<OnboardingFlowUpdated />);
    await waitFor(() => {
      expect(screen.getByText('Your mint pool is empty')).toBeInTheDocument();
    });
  });

  it('should display a warning when there is current inventory not added to the mint pool', async () => {
    render(<OnboardingFlowUpdated />);
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'DEF-456' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '200' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '2000' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Continue to Minting'));
    await waitFor(() => {
      expect(screen.getByText('You have unsaved inventory items that are not in the mint pool.')).toBeInTheDocument();
    });
  });

  it('should add current items to the mint pool when clicking the warning button', async () => {
    render(<OnboardingFlowUpdated />);
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'DEF-456' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '200' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '2000' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Continue to Minting'));
    fireEvent.click(screen.getByText('Add Current Items to Mint Pool'));
    await waitFor(() => {
      expect(screen.getByText('Items ready to be minted: 2')).toBeInTheDocument();
    });
  });

  it('should disable the mint button when minting', async () => {
    render(<OnboardingFlowUpdated />);
    // Add items to the mint pool
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    fireEvent.click(screen.getByText('Continue to Minting'));
    fireEvent.click(screen.getByText('Mint Inventory Contract'));
    await waitFor(() => {
      expect(screen.getByText('Minting...')).toBeInTheDocument();
    });
  });

  it('should disable the mint button when there is no inventory', async () => {
    render(<OnboardingFlowUpdated />);
    fireEvent.click(screen.getByText('Continue to Minting'));
    await waitFor(() => {
      expect(screen.getByText('Mint Inventory Contract')).toBeInTheDocument();
      expect(screen.getByText('Mint Inventory Contract')).toHaveClass('disabledButton');
    });
  });

  it('should call onComplete when clicking "Continue to Dashboard"', async () => {
    const onComplete = jest.fn();
    render(<OnboardingFlowUpdated onComplete={onComplete} />);
    // Add items to the mint pool
    fireEvent.click(screen.getByText('Manual Entry Form'));
    fireEvent.change(screen.getByLabelText('Part Number*'), { target: { value: 'ABC-123' } });
    fireEvent.change(screen.getByLabelText('Quantity*'), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText('Value*'), { target: { value: '1500' } });
    fireEvent.click(screen.getByText('Add Item to Inventory'));
    fireEvent.click(screen.getByText('Add to Mint Pool'));
    fireEvent.click(screen.getByText('Continue to Minting'));
    fireEvent.click(screen.getByText('Mint Inventory Contract'));
    await waitFor(() => {
      fireEvent.click(screen.getByText('Continue to Dashboard'));
      expect(onComplete).toHaveBeenCalled();
    });
  });

  it('should close the modal when clicking the close button', async () => {
    const onComplete = jest.fn();
    render(<OnboardingFlowUpdated onComplete={onComplete} />);
    fireEvent.click(screen.getByText('×'));
    expect(onComplete).toHaveBeenCalled();
  });
});
