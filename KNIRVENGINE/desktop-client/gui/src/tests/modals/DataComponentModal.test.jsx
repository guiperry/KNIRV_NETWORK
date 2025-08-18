import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import DataComponentModal from '../../components/modals/DataComponentModal';

describe('DataComponentModal', () => {
  const mockOnClose = jest.fn();
  const mockOnComponentSaved = jest.fn();
  
  const mockComponent = {
    id: 1,
    name: 'Test Component',
    type: 'processor',
    description: 'Test description',
    category: 'transform',
    schema: '{"type": "object", "properties": {}}',
    configuration: '{"param1": "value1"}'
  };
  
  const renderModal = (isOpen = true, component = null) => {
    return render(
      <DataComponentModal 
        isOpen={isOpen} 
        onClose={mockOnClose} 
        component={component} 
        onComponentSaved={mockOnComponentSaved} 
      />
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should not render when isOpen is false', () => {
    renderModal(false);
    expect(screen.queryByText('Add Component')).not.toBeInTheDocument();
    expect(screen.queryByText('Edit Component')).not.toBeInTheDocument();
  });

  test('should render "Add Component" when creating a new component', () => {
    renderModal(true, null);
    expect(screen.getByText('Add Component')).toBeInTheDocument();
  });

  test('should render "Edit Component" when editing an existing component', () => {
    renderModal(true, mockComponent);
    expect(screen.getByText('Edit Component')).toBeInTheDocument();
  });

  test('should populate form fields when editing an existing component', () => {
    renderModal(true, mockComponent);
    expect(screen.getByLabelText('Component Name')).toHaveValue(mockComponent.name);
    expect(screen.getByLabelText('Description')).toHaveValue(mockComponent.description);
    expect(screen.getByLabelText('JSON Schema')).toHaveValue(mockComponent.schema);
    expect(screen.getByLabelText('Default Configuration (JSON)')).toHaveValue(mockComponent.configuration);
    
    // Check if the correct component type is selected
    const processorTypeElement = screen.getByText('Processor');
    const processorContainer = processorTypeElement.closest('.cursor-pointer');
    expect(processorContainer).toHaveClass('bg-gradient-to-r');
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
    fireEvent.change(screen.getByLabelText('Component Name'), {
      target: { value: '' }
    });
    
    fireEvent.change(screen.getByLabelText('Description'), {
      target: { value: '' }
    });
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Component'));
    
    // Wait for validation errors
    await waitFor(() => {
      expect(screen.getByText('Component name is required')).toBeInTheDocument();
      expect(screen.getByText('Description is required')).toBeInTheDocument();
      expect(screen.getByText('Category is required')).toBeInTheDocument();
    });
    
    // Ensure onComponentSaved was not called
    expect(mockOnComponentSaved).not.toHaveBeenCalled();
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

  test('should successfully save component when form is valid', async () => {
    renderModal();

    // Fill out the form with all required fields
    fireEvent.change(screen.getByLabelText('Component Name'), {
      target: { value: 'New Test Component' }
    });

    fireEvent.change(screen.getByLabelText('Description'), {
      target: { value: 'Test description for the component' }
    });

    // Select processor type (already selected by default)

    // Select a category
    fireEvent.change(screen.getByLabelText('Category'), {
      target: { value: 'transform' }
    });

    // Submit the form
    fireEvent.click(screen.getByText('Save Component'));

    // Wait for success message and callback
    await waitFor(() => {
      expect(screen.getByText('Component saved!')).toBeInTheDocument();
      expect(mockOnComponentSaved).toHaveBeenCalledTimes(1);
      expect(mockOnComponentSaved).toHaveBeenCalledWith(expect.objectContaining({
        name: 'New Test Component',
        type: 'processor',
        category: 'transform',
        description: 'Test description for the component'
      }));
    }, { timeout: 3000 });
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
    const fileInput = screen.getByRole('textbox', { name: /json schema/i }).parentElement.querySelector('input[type="file"]');
    fireEvent.change(fileInput, { target: { files: [file] } });
    
    // Simulate FileReader onload event
    mockFileReader.onload({ target: { result: validJsonContent } });
    
    // Wait for schema to be updated and validated
    await waitFor(() => {
      expect(screen.getByLabelText('JSON Schema')).toHaveValue(validJsonContent);
    });
    
    // Restore FileReader
    global.FileReader = originalFileReader;
  });

  test('should update available categories when component type changes', () => {
    renderModal();
    
    // Initially should show processor categories
    const categorySelect = screen.getByLabelText('Category');
    expect(categorySelect).toBeInTheDocument();
    
    // Change to source type
    const sourceTypeElement = screen.getByText('Source');
    fireEvent.click(sourceTypeElement);
    
    // Should reset category
    expect(categorySelect).toHaveValue('');
    
    // Should have source categories available
    fireEvent.click(categorySelect);
    expect(screen.getByText('Streaming')).toBeInTheDocument();
  });
});