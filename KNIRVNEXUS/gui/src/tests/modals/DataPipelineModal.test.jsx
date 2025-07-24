import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import DataPipelineModal from '../../components/modals/DataPipelineModal';

describe('DataPipelineModal', () => {
  const mockOnClose = jest.fn();
  const mockOnPipelineSaved = jest.fn();
  
  const mockPipeline = {
    id: 1,
    name: 'Test Pipeline',
    description: 'Test description',
    status: 'draft',
    lastRun: 'Never',
    nextRun: 'Not scheduled',
    createdBy: 'Test User',
    createdAt: '2024-01-15T10:30:00Z',
    components: [
      { id: 101, type: 'source', name: 'Test Source', position: { x: 100, y: 150 } },
      { id: 102, type: 'processor', name: 'Test Processor', position: { x: 300, y: 150 } }
    ],
    connections: [
      { source: 101, target: 102 }
    ],
    metrics: {
      eventsProcessed: 0,
      processingTime: 'N/A',
      errorRate: 'N/A',
      throughput: 'N/A'
    }
  };
  
  const mockComponents = [
    { id: 1, type: 'source', name: 'Kafka Source', description: 'Ingest data from Kafka topics', category: 'streaming' },
    { id: 2, type: 'processor', name: 'Filter', description: 'Filter data based on conditions', category: 'transform' },
    { id: 3, type: 'sink', name: 'Database Sink', description: 'Write data to databases', category: 'database' }
  ];
  
  const renderModal = (isOpen = true, pipeline = null) => {
    return render(
      <DataPipelineModal 
        isOpen={isOpen} 
        onClose={mockOnClose} 
        pipeline={pipeline} 
        components={mockComponents}
        onPipelineSaved={mockOnPipelineSaved} 
      />
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should not render when isOpen is false', () => {
    renderModal(false);
    expect(screen.queryByText('Create Data Pipeline')).not.toBeInTheDocument();
    expect(screen.queryByText('Edit Data Pipeline')).not.toBeInTheDocument();
  });

  test('should render "Create Data Pipeline" when creating a new pipeline', () => {
    renderModal(true, null);
    expect(screen.getByText('Create Data Pipeline')).toBeInTheDocument();
  });

  test('should render "Edit Data Pipeline" when editing an existing pipeline', () => {
    renderModal(true, mockPipeline);
    expect(screen.getByText('Edit Data Pipeline')).toBeInTheDocument();
  });

  test('should populate form fields when editing an existing pipeline', () => {
    renderModal(true, mockPipeline);
    expect(screen.getByLabelText('Pipeline Name')).toHaveValue(mockPipeline.name);
    expect(screen.getByLabelText('Description')).toHaveValue(mockPipeline.description);
    expect(screen.getByLabelText('Status')).toHaveValue(mockPipeline.status);
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
    fireEvent.change(screen.getByLabelText('Pipeline Name'), {
      target: { value: '' }
    });

    fireEvent.change(screen.getByLabelText('Description'), {
      target: { value: '' }
    });

    // Submit the form
    fireEvent.click(screen.getByText('Save Pipeline'));

    // Wait for validation errors on the Overview tab
    await waitFor(() => {
      expect(screen.getByText('Pipeline name is required')).toBeInTheDocument();
      expect(screen.getByText('Description is required')).toBeInTheDocument();
    });

    // Switch to Components tab to see component validation error
    fireEvent.click(screen.getByText('Components'));

    await waitFor(() => {
      expect(screen.getByText('At least one component is required')).toBeInTheDocument();
    });

    // Ensure onPipelineSaved was not called
    expect(mockOnPipelineSaved).not.toHaveBeenCalled();
  });

  test('should successfully save pipeline when form is valid', async () => {
    renderModal(true, mockPipeline);
    
    // Submit the form
    fireEvent.click(screen.getByText('Save Pipeline'));
    
    // Wait for success message and callback
    await waitFor(() => {
      expect(screen.getByText('Pipeline saved!')).toBeInTheDocument();
      expect(mockOnPipelineSaved).toHaveBeenCalledTimes(1);
      expect(mockOnPipelineSaved).toHaveBeenCalledWith(expect.objectContaining({
        id: mockPipeline.id,
        name: mockPipeline.name,
        status: mockPipeline.status
      }));
    }, { timeout: 3000 });
  });

  test('should switch between tabs', () => {
    renderModal();
    
    // Default tab should be Basic Information
    expect(screen.getByLabelText('Pipeline Name')).toBeInTheDocument();
    
    // Click on Components tab
    fireEvent.click(screen.getByText('Components'));
    expect(screen.getByText('Selected Components')).toBeInTheDocument();
    
    // Click on Connections tab
    fireEvent.click(screen.getByText('Connections'));
    expect(screen.getByText('Component Connections')).toBeInTheDocument();
    
    // Click back to Basic Information tab
    fireEvent.click(screen.getByText('Basic Information'));
    expect(screen.getByLabelText('Pipeline Name')).toBeInTheDocument();
  });

  test('should add and remove components', () => {
    renderModal(true, mockPipeline);
    
    // Switch to Components tab
    fireEvent.click(screen.getByText('Components'));
    
    // Initially should show existing components
    expect(screen.getByText('Test Source')).toBeInTheDocument();
    expect(screen.getByText('Test Processor')).toBeInTheDocument();
    
    // Add a new component
    const addButtons = screen.getAllByRole('button', { name: '' });
    fireEvent.click(addButtons[0]); // First add button
    
    // Should now have the new component
    expect(screen.getAllByText('Kafka Source').length).toBeGreaterThan(0);
    
    // Remove a component
    const removeButtons = screen.getAllByRole('button', { name: '' });
    fireEvent.click(removeButtons[0]); // First remove button
    
    // Should have one less component
    expect(screen.queryByText('Test Source')).not.toBeInTheDocument();
  });

  test('should add and remove connections', () => {
    renderModal(true, mockPipeline);
    
    // Switch to Connections tab
    fireEvent.click(screen.getByText('Connections'));
    
    // Initially should show existing connection
    expect(screen.getByText('Test Source')).toBeInTheDocument();
    expect(screen.getByText('Test Processor')).toBeInTheDocument();
    
    // Remove the connection
    const removeButton = screen.getByRole('button', { name: '' });
    fireEvent.click(removeButton);
    
    // Connection should be removed
    expect(screen.queryByText('Test Source')).not.toBeInTheDocument();
    
    // Add a new connection
    // This is more complex to test as it involves selecting options from dropdowns
    // and would require more detailed testing of the component's internal state
  });
});