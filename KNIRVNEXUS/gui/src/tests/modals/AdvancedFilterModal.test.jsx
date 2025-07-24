import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import AdvancedFilterModal from '../../components/modals/AdvancedFilterModal';

describe('AdvancedFilterModal', () => {
  const mockOnClose = jest.fn();
  const mockOnApplyFilters = jest.fn();
  
  const mockFilterOptions = {
    fields: [
      { id: 'name', name: 'Name' },
      { id: 'status', name: 'Status' },
      { id: 'type', name: 'Type' }
    ],
    operators: [
      { id: 'equals', name: 'Equals' },
      { id: 'contains', name: 'Contains' },
      { id: 'greater_than', name: 'Greater Than' }
    ]
  };
  
  const mockInitialFilters = [
    { field: 'status', operator: 'equals', value: 'active', id: 101 }
  ];
  
  const renderModal = (isOpen = true, initialFilters = []) => {
    return render(
      <AdvancedFilterModal 
        isOpen={isOpen} 
        onClose={mockOnClose} 
        onApplyFilters={mockOnApplyFilters} 
        initialFilters={initialFilters}
        filterOptions={mockFilterOptions}
        entityType="test"
      />
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();

    // Mock localStorage
    const localStorageMock = {
      getItem: jest.fn(),
      setItem: jest.fn(),
      clear: jest.fn()
    };
    Object.defineProperty(window, 'localStorage', {
      value: localStorageMock,
      writable: true
    });
  });

  test('should not render when isOpen is false', () => {
    renderModal(false);
    expect(screen.queryByText('Advanced Filters')).not.toBeInTheDocument();
  });

  test('should render with initial empty filter when no initialFilters provided', () => {
    renderModal(true, []);
    expect(screen.getByText('Advanced Filters')).toBeInTheDocument();
    expect(screen.getByText('Build Filters')).toBeInTheDocument();
    
    // Should have one empty filter row
    expect(screen.getByText('Select Field')).toBeInTheDocument();
    expect(screen.getByText('Equals')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Value')).toBeInTheDocument();
  });

  test('should render with initialFilters when provided', () => {
    renderModal(true, mockInitialFilters);
    
    // Should show the initial filter
    const fieldSelects = screen.getAllByRole('combobox');
    expect(fieldSelects[0]).toHaveValue('status');
    
    const valueInputs = screen.getAllByPlaceholderText('Value');
    expect(valueInputs[0]).toHaveValue('active');
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

  test('should add a new filter when Add Filter button is clicked', () => {
    renderModal();
    
    // Initially one filter row
    expect(screen.getAllByText('Select Field').length).toBe(1);
    
    // Click Add Filter button
    fireEvent.click(screen.getByText('Add Filter'));
    
    // Should now have two filter rows
    expect(screen.getAllByText('Select Field').length).toBe(2);
  });

  test('should remove a filter when remove button is clicked', () => {
    renderModal();
    
    // Add a second filter
    fireEvent.click(screen.getByText('Add Filter'));
    
    // Should have two filter rows
    expect(screen.getAllByText('Select Field').length).toBe(2);
    
    // Click the remove button on the second filter
    const removeButtons = screen.getAllByRole('button', { name: '' });
    fireEvent.click(removeButtons[1]); // Second remove button
    
    // Should now have one filter row
    expect(screen.getAllByText('Select Field').length).toBe(1);
  });

  test('should update filter when field, operator, or value changes', () => {
    renderModal();

    // Select a field (first combobox)
    const comboboxes = screen.getAllByRole('combobox');
    const fieldSelect = comboboxes[0];
    fireEvent.change(fieldSelect, { target: { value: 'name' } });
    expect(fieldSelect).toHaveValue('name');

    // Select an operator (second combobox)
    const operatorSelect = comboboxes[1];
    fireEvent.change(operatorSelect, { target: { value: 'contains' } });
    expect(operatorSelect).toHaveValue('contains');

    // Enter a value
    const valueInput = screen.getByPlaceholderText('Value');
    fireEvent.change(valueInput, { target: { value: 'test' } });
    expect(valueInput).toHaveValue('test');
  });

  test('should apply filters when Apply Filters button is clicked', async () => {
    renderModal();

    // Set up a filter
    const comboboxes = screen.getAllByRole('combobox');
    const fieldSelect = comboboxes[0];
    fireEvent.change(fieldSelect, { target: { value: 'name' } });

    const valueInput = screen.getByPlaceholderText('Value');
    fireEvent.change(valueInput, { target: { value: 'test' } });

    // Click Apply Filters button
    fireEvent.click(screen.getByText('Apply Filters'));

    // Should show loading state
    expect(screen.getByText('Applying...')).toBeInTheDocument();

    // Wait for success message
    await waitFor(() => {
      expect(screen.getByText('Filters applied!')).toBeInTheDocument();
    });

    // Should call onApplyFilters with the filters
    expect(mockOnApplyFilters).toHaveBeenCalledTimes(1);
    expect(mockOnApplyFilters).toHaveBeenCalledWith(expect.arrayContaining([
      expect.objectContaining({
        field: 'name',
        operator: 'equals',
        value: 'test'
      })
    ]));

    // Should close the modal after success
    await waitFor(() => {
      expect(mockOnClose).toHaveBeenCalledTimes(1);
    });
  });

  test('should show validation error when trying to apply with incomplete filters', async () => {
    renderModal();
    
    // Leave the filter incomplete (no field selected)
    
    // Click Apply Filters button
    fireEvent.click(screen.getByText('Apply Filters'));
    
    // Should show validation error
    await waitFor(() => {
      expect(screen.getByText('At least one complete filter is required')).toBeInTheDocument();
    });
    
    // Should not call onApplyFilters
    expect(mockOnApplyFilters).not.toHaveBeenCalled();
  });

  test('should save a preset when Save Preset button is clicked', async () => {
    renderModal();

    // Set up a filter
    const comboboxes = screen.getAllByRole('combobox');
    const fieldSelect = comboboxes[0];
    fireEvent.change(fieldSelect, { target: { value: 'name' } });

    const valueInput = screen.getByPlaceholderText('Value');
    fireEvent.change(valueInput, { target: { value: 'test' } });

    // Enter a preset name
    const presetNameInput = screen.getByLabelText('Preset Name');
    fireEvent.change(presetNameInput, { target: { value: 'My Test Preset' } });

    // Click Save Preset button
    fireEvent.click(screen.getByText('Save Preset'));

    // Should save to localStorage
    await waitFor(() => {
      expect(localStorage.setItem).toHaveBeenCalledWith(
        'test-filter-presets',
        expect.stringContaining('My Test Preset')
      );
    });
  });

  test('should show validation error when trying to save preset without a name', () => {
    renderModal();

    // Set up a filter
    const comboboxes = screen.getAllByRole('combobox');
    const fieldSelect = comboboxes[0];
    fireEvent.change(fieldSelect, { target: { value: 'name' } });

    // Leave preset name empty

    // Click Save Preset button
    fireEvent.click(screen.getByText('Save Preset'));

    // Should show validation error
    expect(screen.getByText('Preset name is required')).toBeInTheDocument();
  });

  test('should reset filters when Reset button is clicked', () => {
    renderModal(true, mockInitialFilters);
    
    // Should have initial filter
    const fieldSelects = screen.getAllByRole('combobox');
    expect(fieldSelects[0]).toHaveValue('status');
    
    // Click Reset button
    fireEvent.click(screen.getByText('Reset'));
    
    // Should reset to empty filter
    const newFieldSelects = screen.getAllByRole('combobox');
    expect(newFieldSelects[0]).toHaveValue('');
  });
});