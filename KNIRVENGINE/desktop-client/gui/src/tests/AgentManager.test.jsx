import React from 'react';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import '@testing-library/jest-dom';
import { BrowserRouter } from 'react-router-dom';
import { AgentManager } from '../components/AgentManager';

// Mock the API module first
jest.mock('../utils/api', () => ({
  fetchAgents: jest.fn(() => Promise.resolve([
    {
      id: '1',
      name: 'CyberPunk Agent #7804',
      type: 'standard',
      config: JSON.stringify({
        collection: 'CyberPunk Collection',
        status: 'active',
        capabilities: ['Data Mining', 'Pattern Recognition'],
        target_types: ['Database', 'API'],
        current_target: 'Production DB',
        capability: 'Data Mining',
        deployed_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(), // 2 hours ago
        image_url: 'https://api.dicebear.com/7.x/bottts/svg?seed=cyberpunk7804'
      }),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    },
    {
      id: '2',
      name: 'Data Miner #3749',
      type: 'standard',
      config: JSON.stringify({
        collection: 'Mining Collection',
        status: 'idle',
        capabilities: ['Data Extraction', 'Analysis'],
        target_types: ['File System', 'Database'],
        current_target: null,
        capability: null,
        deployed_at: null,
        image_url: 'https://api.dicebear.com/7.x/bottts/svg?seed=dataminer3749'
      }),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    },
    {
      id: '3',
      name: 'System Monitor #4523',
      type: 'standard',
      config: JSON.stringify({
        collection: 'Monitoring Collection',
        status: 'maintenance',
        capabilities: ['System Monitoring', 'Alert Management'],
        target_types: ['Server', 'Network'],
        current_target: 'Maintenance Mode',
        capability: 'System Monitoring',
        deployed_at: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(), // 1 day ago
        image_url: 'https://api.dicebear.com/7.x/bottts/svg?seed=monitor4523'
      }),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    },
    {
      id: '4',
      name: 'Code Reviewer #7894',
      type: 'standard',
      config: JSON.stringify({
        collection: 'Development Collection',
        status: 'idle',
        capabilities: ['Code Analysis', 'Quality Assurance'],
        target_types: ['Git Repository', 'CI/CD'],
        current_target: null,
        capability: null,
        deployed_at: null,
        image_url: 'https://api.dicebear.com/7.x/bottts/svg?seed=reviewer7894'
      }),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    }
  ])),
  deployAgent: jest.fn(() => Promise.resolve({ success: true })),
  stopAgent: jest.fn(() => Promise.resolve({ success: true })),
  updateAgent: jest.fn(() => Promise.resolve({ success: true })),
  deleteAgent: jest.fn(() => Promise.resolve({ success: true })),
  createAgent: jest.fn(() => Promise.resolve({
    id: 'new-agent-id',
    name: 'New Test Agent',
    type: 'standard',
    config: JSON.stringify({
      collection: 'Test Collection',
      status: 'idle',
      capabilities: ['Test Capability'],
      target_types: ['Test Target']
    }),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString()
  }))
}));

// Mock the WebSocket module
jest.mock('../utils/websocket', () => ({
  wsManager: {
    connect: jest.fn(() => Promise.resolve()),
    disconnect: jest.fn(),
    subscribe: jest.fn(),
    unsubscribe: jest.fn(),
    send: jest.fn()
  },
  subscribeToAgentStatus: jest.fn()
}));

// Mock the asset path hooks
jest.mock('../hooks/useAssetPath', () => ({
  useDefaultAgentImage: () => '/test-agent-image.png',
  useAppLogo: () => '/test-logo.png',
}));

// Mock the AgentImage component
jest.mock('../components/common/AgentImage', () => {
  return function AgentImage({ src, alt, className, ...props }) {
    return <img src={src || '/test-agent-image.png'} alt={alt} className={className} {...props} />;
  };
});

// Mock the modal components
jest.mock('../components/modals/AgentCreationModal', () => {
  return jest.fn(({ isOpen, onClose, onAgentCreated }) => {
    if (!isOpen) return null;
    return (
      <div data-testid="agent-creation-modal">
        <button onClick={() => onClose()}>Close</button>
        <button onClick={() => onAgentCreated({ 
          name: 'New Test Agent', 
          collection: 'Test Collection',
          status: 'idle',
          capabilities: ['Test Capability'],
          targetTypes: ['Test Target']
        })}>
          Create Agent
        </button>
      </div>
    );
  });
});

jest.mock('../components/modals/AgentDeploymentModal', () => {
  return jest.fn(({ isOpen, onClose, agent, onAgentDeployed }) => {
    if (!isOpen || !agent) return null;
    return (
      <div data-testid="agent-deployment-modal">
        <div>Deploying: {agent.name}</div>
        <button onClick={() => onClose()}>Close</button>
        <button onClick={() => onAgentDeployed({
          ...agent,
          status: 'active',
          currentTarget: 'Test Target',
          capability: 'Test Capability',
          deployedSince: 'just now'
        })}>
          Deploy
        </button>
      </div>
    );
  });
});

jest.mock('../components/modals/AgentConfigModal', () => {
  return jest.fn(({ isOpen, onClose, agent, onAgentUpdated }) => {
    if (!isOpen || !agent) return null;
    return (
      <div data-testid="agent-config-modal">
        <div>Configuring: {agent.name}</div>
        <button onClick={() => onClose()}>Close</button>
        <button onClick={() => onAgentUpdated({
          ...agent,
          name: 'Updated ' + agent.name
        })}>
          Update
        </button>
      </div>
    );
  });
});

jest.mock('../components/modals/AdvancedFilterModal', () => {
  return jest.fn(({ isOpen, onClose, onApplyFilters, initialFilters, filterOptions }) => {
    if (!isOpen) return null;
    return (
      <div data-testid="advanced-filter-modal">
        <div>Advanced Filters</div>
        <button onClick={() => onClose()}>Close</button>
        <button onClick={() => onApplyFilters([
          { id: 1, field: 'status', operator: 'equals', value: 'active' }
        ])}>
          Apply Filters
        </button>
      </div>
    );
  });
});

jest.mock('../components/modals/AgentDetailModal', () => {
  return jest.fn(({ isOpen, onClose, agent, onDeploy, onStop, onConfigure }) => {
    if (!isOpen || !agent) return null;
    return (
      <div data-testid="agent-detail-modal">
        <div>Agent Details: {agent.name}</div>
        <button onClick={() => onClose()}>Close</button>
        <button onClick={() => onDeploy(agent)}>Deploy</button>
        <button onClick={() => onStop(agent)}>Stop</button>
        <button onClick={() => onConfigure(agent)}>Configure</button>
      </div>
    );
  });
});

describe('AgentManager', () => {
  const renderComponent = () => {
    return render(
      <BrowserRouter>
        <AgentManager />
      </BrowserRouter>
    );
  };

  test('should render the component with initial agents', async () => {
    renderComponent();
    expect(screen.getByRole('heading', { name: /NFT-Agent Manager/i })).toBeInTheDocument();

    // Initially should show loading
    expect(screen.getByText('Loading agents...')).toBeInTheDocument();

    // Wait for agents to load
    await waitFor(() => {
      expect(screen.queryByText('Loading agents...')).not.toBeInTheDocument();
      expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
      expect(screen.getByText('Data Miner #3749')).toBeInTheDocument();
    }, { timeout: 5000 });
  });

  test('should filter agents based on search term', async () => {
    renderComponent();

    // Wait for agents to load
    await waitFor(() => {
      expect(screen.queryByText('Loading agents...')).not.toBeInTheDocument();
      expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
      expect(screen.getByText('Data Miner #3749')).toBeInTheDocument();
    }, { timeout: 5000 });

    // Search for "CyberPunk"
    fireEvent.change(screen.getByPlaceholderText('Search agents...'), {
      target: { value: 'CyberPunk' }
    });

    // Only CyberPunk agent should be visible
    expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
    expect(screen.queryByText('Data Miner #3749')).not.toBeInTheDocument();
  });

  test('should filter agents based on category', async () => {
    renderComponent();

    // Wait for agents to load
    await waitFor(() => {
      expect(screen.queryByText('Loading agents...')).not.toBeInTheDocument();
      expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
      expect(screen.getByText('System Monitor #4523')).toBeInTheDocument();
    }, { timeout: 5000 });

    // Filter by "maintenance" category
    fireEvent.click(screen.getByText('Maintenance'));

    // Only maintenance agents should be visible
    expect(screen.queryByText('CyberPunk Agent #7804')).not.toBeInTheDocument();
    expect(screen.getByText('System Monitor #4523')).toBeInTheDocument();
  });

  test('should toggle between grid and list view', async () => {
    renderComponent();

    // Wait for agents to load first
    await waitFor(() => {
      expect(screen.queryByText('Loading agents...')).not.toBeInTheDocument();
      expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
    }, { timeout: 5000 });

    // Default view should be grid - look for Deploy buttons
    await waitFor(() => {
      expect(screen.getAllByText('Deploy').length).toBeGreaterThan(0);
    });

    // Switch to list view
    fireEvent.click(screen.getByRole('button', { name: /list/i }));

    // List view should show table headers
    expect(screen.getByText('Agent')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getByText('Current Target')).toBeInTheDocument();

    // Switch back to grid view
    fireEvent.click(screen.getByRole('button', { name: /grid/i }));

    // Grid view should show cards with Deploy buttons
    await waitFor(() => {
      expect(screen.getAllByText('Deploy').length).toBeGreaterThan(0);
    });
  });

  test('should open agent creation modal when Create Agent button is clicked', async () => {
    renderComponent();

    // Wait for loading to complete so button is enabled
    await waitFor(() => {
      expect(screen.queryByText('Loading agents...')).not.toBeInTheDocument();
    }, { timeout: 5000 });

    // Click Create Agent button
    fireEvent.click(screen.getByRole('button', { name: /Create Agent/i }));

    // Modal should be open
    expect(screen.getByTestId('agent-creation-modal')).toBeInTheDocument();

    // Close the modal
    fireEvent.click(screen.getByRole('button', { name: /Close/i }));

    // Modal should be closed
    expect(screen.queryByTestId('agent-creation-modal')).not.toBeInTheDocument();
  });

  test('should verify createAgent function exists and is mockable', async () => {
    const { createAgent } = require('../utils/api');

    // Verify the mock function exists
    expect(createAgent).toBeDefined();
    expect(typeof createAgent).toBe('function');

    // Test that the mock can be called
    await createAgent({
      name: 'Test Agent',
      collection: 'Test Collection',
      imageURL: 'test-image.jpg',
      capabilities: ['test'],
      targetTypes: ['test']
    });

    expect(createAgent).toHaveBeenCalled();
  });

  test('should open deployment modal when Deploy button is clicked', async () => {
    renderComponent();

    // Wait for agents to load first
    await waitFor(() => {
      expect(screen.queryByText('Loading agents...')).not.toBeInTheDocument();
      expect(screen.getAllByText('Deploy').length).toBeGreaterThan(0);
    }, { timeout: 5000 });

    // Find an idle agent and click its Deploy button
    const deployButtons = screen.getAllByText('Deploy');
    fireEvent.click(deployButtons[0]);

    // Deployment modal should be open
    expect(screen.getByTestId('agent-deployment-modal')).toBeInTheDocument();

    // Close the modal - use a more specific selector
    const modal = screen.getByTestId('agent-deployment-modal');
    const closeButton = within(modal).getByRole('button', { name: /Close/i });
    fireEvent.click(closeButton);

    // Modal should be closed
    expect(screen.queryByTestId('agent-deployment-modal')).not.toBeInTheDocument();
  });

  test('should update agent status when deployed through the modal', async () => {
    renderComponent();

    // Wait for agents to load first
    await waitFor(() => {
      expect(screen.queryByText('Loading agents...')).not.toBeInTheDocument();
      expect(screen.getAllByText('Deploy').length).toBeGreaterThan(0);
    }, { timeout: 5000 });

    // Find an idle agent and click its Deploy button
    const deployButtons = screen.getAllByText('Deploy');
    fireEvent.click(deployButtons[0]);

    // Deploy the agent - use a more specific selector within the modal
    const modal = screen.getByTestId('agent-deployment-modal');
    const deployButton = within(modal).getByRole('button', { name: /Deploy/i });
    fireEvent.click(deployButton);

    // Wait for the agent to be updated
    await waitFor(() => {
      // The Stop button should now be visible for this agent
      expect(screen.getAllByText('Stop').length).toBeGreaterThan(0);
    });
  });

  test('should open configuration modal when Settings button is clicked', async () => {
    renderComponent();

    // Wait for agents to load first
    await waitFor(() => {
      expect(screen.queryByText('Loading agents...')).not.toBeInTheDocument();
      expect(screen.getAllByRole('button', { name: /settings/i }).length).toBeGreaterThan(0);
    }, { timeout: 5000 });

    // Find a Settings button and click it
    const settingsButtons = screen.getAllByRole('button', { name: /settings/i });
    fireEvent.click(settingsButtons[0]);

    // Config modal should be open
    expect(screen.getByTestId('agent-config-modal')).toBeInTheDocument();

    // Close the modal - use a more specific selector
    const modal = screen.getByTestId('agent-config-modal');
    const closeButton = within(modal).getByRole('button', { name: /Close/i });
    fireEvent.click(closeButton);

    // Modal should be closed
    expect(screen.queryByTestId('agent-config-modal')).not.toBeInTheDocument();
  });

  test('should update agent when configured through the modal', async () => {
    renderComponent();

    // Wait for agents to load first
    await waitFor(() => {
      expect(screen.queryByText('Loading agents...')).not.toBeInTheDocument();
      expect(screen.getAllByRole('button', { name: /settings/i }).length).toBeGreaterThan(0);
    }, { timeout: 5000 });

    // Find a Settings button and click it
    const settingsButtons = screen.getAllByRole('button', { name: /settings/i });
    fireEvent.click(settingsButtons[0]);

    // Update the agent
    fireEvent.click(screen.getByRole('button', { name: /Update/i }));

    // Wait for the agent to be updated
    await waitFor(() => {
      expect(screen.getByText(/Updated CyberPunk Agent #7804|Updated Data Miner #3749/)).toBeInTheDocument();
    });
  });

  test('should call stopAgent API when Stop button is clicked', async () => {
    const { stopAgent } = require('../utils/api');
    renderComponent();

    // Wait for agents to load first
    await waitFor(() => {
      expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
    }, { timeout: 5000 });

    // Find an active agent and click its Stop button
    const stopButtons = screen.getAllByText('Stop');
    fireEvent.click(stopButtons[0]);

    // Wait for the API call to be made
    await waitFor(() => {
      expect(stopAgent).toHaveBeenCalledWith('1');
    });
  });

  test('should open advanced filter modal when Filter button is clicked', () => {
    renderComponent();
    
    // Click Filter button
    fireEvent.click(screen.getByRole('button', { name: /Filter/i }));
    
    // Modal should be open
    expect(screen.getByTestId('advanced-filter-modal')).toBeInTheDocument();
    
    // Close the modal
    fireEvent.click(screen.getByRole('button', { name: /Close/i }));
    
    // Modal should be closed
    expect(screen.queryByTestId('advanced-filter-modal')).not.toBeInTheDocument();
  });

  test('should apply filters when applied through the modal', async () => {
    renderComponent();
    
    // Wait for agents to load first
    await waitFor(() => {
      expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
    }, { timeout: 5000 });

    // Initially all agents should be visible
    expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
    expect(screen.getByText('Code Reviewer #7894')).toBeInTheDocument();
    
    // Open filter modal
    fireEvent.click(screen.getByRole('button', { name: /Filter/i }));
    
    // Apply a filter for active agents
    fireEvent.click(screen.getByRole('button', { name: /Apply Filters/i }));
    
    // Wait for filters to be applied
    await waitFor(() => {
      // Should show active agents and hide idle agents
      expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
      expect(screen.queryByText('Code Reviewer #7894')).not.toBeInTheDocument();
    });
    
    // Should show active filters display
    expect(screen.getByText('Active filters:')).toBeInTheDocument();
    expect(screen.getByText('Status Equals active')).toBeInTheDocument();
  });

  test('should clear filters when Clear button is clicked', async () => {
    renderComponent();
    
    // Open filter modal
    fireEvent.click(screen.getByRole('button', { name: /Filter/i }));
    
    // Apply a filter
    fireEvent.click(screen.getByRole('button', { name: /Apply Filters/i }));
    
    // Wait for filters to be applied
    await waitFor(() => {
      expect(screen.getByText('Active filters:')).toBeInTheDocument();
    });
    
    // Clear filters
    fireEvent.click(screen.getByText('Clear'));
    
    // All agents should be visible again
    expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
    expect(screen.getByText('Code Reviewer #7894')).toBeInTheDocument();
    
    // Active filters display should be gone
    expect(screen.queryByText('Active filters:')).not.toBeInTheDocument();
  });

  test('should open agent detail modal when View button is clicked', async () => {
    renderComponent();

    // Wait for agents to load
    await waitFor(() => {
      expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
    });

    // Find a View button and click it
    const viewButtons = screen.getAllByRole('button', { name: /view details for/i });
    fireEvent.click(viewButtons[0]);

    // Detail modal should be open
    expect(screen.getByTestId('agent-detail-modal')).toBeInTheDocument();

    // Close the modal
    fireEvent.click(screen.getByRole('button', { name: /Close/i }));

    // Modal should be closed
    expect(screen.queryByTestId('agent-detail-modal')).not.toBeInTheDocument();
  });

  test('should handle actions from the detail modal', async () => {
    renderComponent();

    // Wait for agents to load
    await waitFor(() => {
      expect(screen.getByText('CyberPunk Agent #7804')).toBeInTheDocument();
    });

    // Find a View button and click it
    const viewButtons = screen.getAllByRole('button', { name: /view details for/i });
    fireEvent.click(viewButtons[0]);

    // Detail modal should be open
    expect(screen.getByTestId('agent-detail-modal')).toBeInTheDocument();
    
    // Click Deploy in the detail modal
    const deployButtons = screen.getAllByText('Deploy');
    fireEvent.click(deployButtons[0]); // Click the first Deploy button
    
    // Deployment modal should open
    expect(screen.getByTestId('agent-deployment-modal')).toBeInTheDocument();
    
    // Close the deployment modal
    fireEvent.click(screen.getAllByRole('button', { name: /Close/i })[0]);
    
    // Click Stop in the detail modal (use getAllByText to handle multiple Stop buttons)
    const stopButtons = screen.getAllByText('Stop');
    fireEvent.click(stopButtons[0]); // Click the first Stop button
    
    // Agent should be stopped
    await waitFor(() => {
      // Detail modal should still be open
      expect(screen.getByTestId('agent-detail-modal')).toBeInTheDocument();
    });
    
    // Click Configure in the detail modal
    fireEvent.click(screen.getByText('Configure'));
    
    // Config modal should open
    expect(screen.getByTestId('agent-config-modal')).toBeInTheDocument();
  });
});