import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import AgentDeploymentModal from '../components/modals/AgentDeploymentModal';

describe('AgentDeploymentModal', () => {
  const mockOnClose = jest.fn();
  const mockOnAgentDeployed = jest.fn();
  
  const mockAgent = {
    id: 1,
    name: 'Test Agent',
    collection: 'Test Collection',
    image: 'https://example.com/image.jpg',
    status: 'idle',
    capabilities: ['Web Analysis', 'Data Extraction'],
    targetTypes: ['Browser', 'File System']
  };
  
  const renderModal = (isOpen = true, agent = mockAgent) => {
    return render(
      <AgentDeploymentModal 
        isOpen={isOpen} 
        onClose={mockOnClose} 
        agent={agent} 
        onAgentDeployed={mockOnAgentDeployed} 
      />
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should not render when isOpen is false', () => {
    renderModal(false);
    expect(screen.queryByText('Deploy Agent')).not.toBeInTheDocument();
  });

  test('should not render when agent is null', () => {
    renderModal(true, null);
    expect(screen.queryByText('Deploy Agent')).not.toBeInTheDocument();
  });

  test('should render when isOpen is true and agent is provided', () => {
    renderModal();
    expect(screen.getByRole('heading', { name: /Deploy Agent/i })).toBeInTheDocument();
    expect(screen.getByText('Select Target System')).toBeInTheDocument();
    expect(screen.getByText(mockAgent.name)).toBeInTheDocument();
  });

  test('should call onClose when cancel button is clicked', () => {
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /Cancel/i }));
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  test('should call onClose when X button is clicked', () => {
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /close/i }));
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  test('should show validation errors when trying to deploy without selecting target and capability', async () => {
    renderModal();

    // Try to deploy without selecting target and capability
    fireEvent.click(screen.getByRole('button', { name: /Deploy Agent/i }));

    // Wait for target validation error
    await waitFor(() => {
      expect(screen.getByText('Please select a target system')).toBeInTheDocument();
    });

    // Select a target to reveal capability section
    const chromeTarget = screen.getByText('Chrome Browser');
    fireEvent.click(chromeTarget);

    // Wait for capabilities to load
    await waitFor(() => {
      expect(screen.getByText('Select Capability')).toBeInTheDocument();
    });

    // Try to deploy without selecting capability
    fireEvent.click(screen.getByRole('button', { name: /Deploy Agent/i }));

    // Wait for capability validation error
    await waitFor(() => {
      expect(screen.getByText('Please select a capability')).toBeInTheDocument();
    });

    // Ensure onAgentDeployed was not called
    expect(mockOnAgentDeployed).not.toHaveBeenCalled();
  });

  test('should successfully deploy agent when target and capability are selected', async () => {
    renderModal();

    // Select a target system (click on the Chrome Browser target div)
    const chromeTarget = screen.getByText('Chrome Browser');
    fireEvent.click(chromeTarget);

    // Wait for capabilities to load based on selected target
    await waitFor(() => {
      expect(screen.getByText('Select Capability')).toBeInTheDocument();
      expect(screen.getAllByText('Web Analysis')).toHaveLength(2); // One in agent info, one in capabilities
    });

    // Select a capability (click on the Web Analysis capability div in the capabilities section)
    const webAnalysisCapabilities = screen.getAllByText('Web Analysis');
    // The second one should be in the capabilities section (the first is in agent info)
    const capabilityDiv = webAnalysisCapabilities[1].closest('div[class*="cursor-pointer"]');
    fireEvent.click(capabilityDiv);

    // Verify deployment summary appears
    await waitFor(() => {
      expect(screen.getByText('Deployment Summary')).toBeInTheDocument();
    });

    // Deploy the agent
    fireEvent.click(screen.getByRole('button', { name: /Deploy Agent/i }));

    // Verify deployment status updates
    await waitFor(() => {
      expect(screen.getByText('Deploying Agent...')).toBeInTheDocument();
    });

    // Wait for success message and callback (deployment simulation takes ~4 seconds)
    await waitFor(() => {
      expect(screen.getByText('Deployment Successful!')).toBeInTheDocument();
      expect(mockOnAgentDeployed).toHaveBeenCalledTimes(1);
      expect(mockOnAgentDeployed).toHaveBeenCalledWith(expect.objectContaining({
        id: mockAgent.id,
        status: 'active'
      }));
    }, { timeout: 6000 }); // Increase timeout to 6 seconds to account for deployment simulation
  });

  test('should handle no compatible capabilities scenario', async () => {
    // Create an agent with capabilities that don't match any target
    const incompatibleAgent = {
      ...mockAgent,
      capabilities: ['Unique Capability That No Target Has']
    };
    
    renderModal(true, incompatibleAgent);
    
    // Select a target system
    const targets = screen.getAllByText(/Chrome Browser|Local File System/i);
    fireEvent.click(targets[0]);
    
    // Wait for no compatible capabilities message
    await waitFor(() => {
      expect(screen.getByText('No compatible capabilities')).toBeInTheDocument();
    });
  });
});