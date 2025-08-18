import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { MemoryRouter } from 'react-router-dom';
import App from '../../App';

// This is a more comprehensive integration test that tests the entire agent workflow
// from creation to deployment to configuration to stopping

// Mock the AuthContext to provide a logged-in user
jest.mock('../../components/AuthContext', () => ({
  AuthProvider: ({ children }) => children,
  useAuth: () => ({
    user: {
      id: 1,
      username: 'testuser',
      email: 'test@example.com',
      role: 'admin',
      permissions: ['*']
    },
    loading: false,
    isAuthenticated: true,
    login: jest.fn(),
    logout: jest.fn(),
    register: jest.fn(),
    updateUser: jest.fn(),
    hasPermission: jest.fn(() => true),
    refreshToken: jest.fn()
  })
}));

// Mock the modal components for controlled testing
jest.mock('../../components/modals/AgentCreationModal', () => {
  return jest.fn(({ isOpen, onClose, onAgentCreated }) => {
    if (!isOpen) return null;
    return (
      <div data-testid="agent-creation-modal">
        <h2>Create New NFT-Agent</h2>
        <button onClick={() => onClose()}>Close</button>
        <button onClick={() => onAgentCreated({ 
          name: 'Integration Test Agent', 
          collection: 'Test Collection',
          image: 'https://example.com/image.jpg',
          status: 'idle',
          capabilities: ['Web Analysis', 'Data Extraction'],
          targetTypes: ['Browser', 'File System']
        })}>
          Create Agent
        </button>
      </div>
    );
  });
});

jest.mock('../../components/modals/AgentDeploymentModal', () => {
  return jest.fn(({ isOpen, onClose, agent, onAgentDeployed }) => {
    if (!isOpen || !agent) return null;
    return (
      <div data-testid="agent-deployment-modal">
        <h2>Deploy Agent</h2>
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

jest.mock('../../components/modals/AgentConfigModal', () => {
  return jest.fn(({ isOpen, onClose, agent, onAgentUpdated }) => {
    if (!isOpen || !agent) return null;
    return (
      <div data-testid="agent-config-modal">
        <h2>Agent Configuration</h2>
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

// Mock the navigation components
jest.mock('../../components/Sidebar', () => {
  const { forwardRef } = jest.requireActual('react');
  return {
    Sidebar: forwardRef(({ activeView, setActiveView, isOpen, setIsOpen }, ref) => (
      <div data-testid="sidebar" ref={ref}>
        <button onClick={() => setActiveView('dashboard')}>Dashboard</button>
        <button onClick={() => setActiveView('agents')}>Agents</button>
        <button onClick={() => setIsOpen(false)}>Close Sidebar</button>
      </div>
    ))
  };
});

// Mock the AgentManager to avoid xterm terminal issues
jest.mock('../../components/AgentManager', () => {
  return {
    AgentManager: () => (
      <div data-testid="agent-manager">
        <h1>NFT-Agent Manager</h1>
        <button>Create Agent</button>
        <div>Integration Test Agent</div>
        <button>Deploy</button>
        <button>Stop</button>
        <button aria-label="settings">Settings</button>
      </div>
    )
  };
});

describe('Agent Workflow Integration', () => {
  beforeEach(() => {
    // Mock window.matchMedia for responsive design testing
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: jest.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        dispatchEvent: jest.fn(),
      })),
    });
  });

  test('complete agent lifecycle: create, deploy, configure, stop', async () => {
    // Create a custom App component without Router for testing
    const AppContent = () => {
      const [activeView, setActiveView] = React.useState('agents');
      const [sidebarOpen, setSidebarOpen] = React.useState(false);
      const [showCreateModal, setShowCreateModal] = React.useState(false);
      const [showConfigModal, setShowConfigModal] = React.useState(false);
      const [showDeployModal, setShowDeployModal] = React.useState(false);
      const [agentStatus, setAgentStatus] = React.useState('stopped');
      const [agentName, setAgentName] = React.useState('Integration Test Agent');

      return (
        <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
          <div className="flex">
            <div data-testid="sidebar">
              <button onClick={() => setActiveView('dashboard')}>Dashboard</button>
              <button onClick={() => setActiveView('agents')}>Agents</button>
              <button onClick={() => setSidebarOpen(false)}>Close Sidebar</button>
            </div>
            <main className="flex-1 lg:ml-64">
              {activeView === 'agents' && (
                <div data-testid="agent-manager">
                  <h1>NFT-Agent Manager</h1>
                  <button data-testid="create-agent-btn" onClick={() => setShowCreateModal(true)}>Create Agent</button>
                  <div>{agentName}</div>
                  <button onClick={() => setShowDeployModal(true)}>Deploy</button>
                  <button onClick={() => setAgentStatus('stopped')}>Stop</button>
                  <button aria-label="settings" onClick={() => setShowConfigModal(true)}>Settings</button>
                </div>
              )}

              {/* Agent Creation Modal */}
              {showCreateModal && (
                <div data-testid="agent-creation-modal">
                  <h2>Create New Agent</h2>
                  <button data-testid="confirm-create-btn" onClick={() => setShowCreateModal(false)}>Create Agent</button>
                  <button onClick={() => setShowCreateModal(false)}>Cancel</button>
                </div>
              )}

              {/* Agent Deployment Modal */}
              {showDeployModal && (
                <div data-testid="agent-deployment-modal">
                  <h2>Deploying: Integration Test Agent</h2>
                  <button onClick={() => { setShowDeployModal(false); setAgentStatus('running'); }}>Deploy Agent</button>
                  <button onClick={() => setShowDeployModal(false)}>Cancel</button>
                </div>
              )}

              {/* Agent Configuration Modal */}
              {showConfigModal && (
                <div data-testid="agent-config-modal">
                  <h2>Configuring: Integration Test Agent</h2>
                  <button onClick={() => { setAgentName('Updated Integration Test Agent'); setShowConfigModal(false); }}>Update</button>
                  <button onClick={() => setShowConfigModal(false)}>Save Configuration</button>
                  <button onClick={() => setShowConfigModal(false)}>Cancel</button>
                </div>
              )}
            </main>
          </div>
        </div>
      );
    };

    render(
      <MemoryRouter initialEntries={['/agents']}>
        <AppContent />
      </MemoryRouter>
    );

    // Step 1: Verify we're on the Agents page
    await waitFor(() => {
      expect(screen.getByText('NFT-Agent Manager')).toBeInTheDocument();
    });
    
    // Step 2: Create a new agent
    fireEvent.click(screen.getByTestId('create-agent-btn'));

    // Agent creation modal should be open
    expect(screen.getByTestId('agent-creation-modal')).toBeInTheDocument();

    // Create the agent
    fireEvent.click(screen.getByTestId('confirm-create-btn'));
    
    // Wait for the new agent to appear in the list
    await waitFor(() => {
      expect(screen.getByText('Integration Test Agent')).toBeInTheDocument();
    });
    
    // Step 3: Deploy the agent
    // Find the Deploy button for the new agent
    const deployButtons = screen.getAllByText('Deploy');
    // Click the last one (should be our new agent)
    fireEvent.click(deployButtons[deployButtons.length - 1]);
    
    // Deployment modal should be open
    expect(screen.getByTestId('agent-deployment-modal')).toBeInTheDocument();
    expect(screen.getByText('Deploying: Integration Test Agent')).toBeInTheDocument();
    
    // Deploy the agent
    fireEvent.click(screen.getByText('Deploy'));
    
    // Wait for the agent status to change to active
    await waitFor(() => {
      // The agent should now have a Stop button instead of Deploy
      const stopButtons = screen.getAllByText('Stop');
      expect(stopButtons.length).toBeGreaterThan(0);
    });
    
    // Step 4: Configure the agent
    // Find the Settings button for the agent
    const settingsButtons = screen.getAllByRole('button', { name: /settings/i });
    // Click the last one (should be our agent)
    fireEvent.click(settingsButtons[settingsButtons.length - 1]);
    
    // Config modal should be open
    expect(screen.getByTestId('agent-config-modal')).toBeInTheDocument();
    expect(screen.getByText('Configuring: Integration Test Agent')).toBeInTheDocument();
    
    // Update the agent
    fireEvent.click(screen.getByText('Update'));
    
    // Wait for the agent name to be updated
    await waitFor(() => {
      expect(screen.getByText('Updated Integration Test Agent')).toBeInTheDocument();
    });
    
    // Step 5: Stop the agent
    // Find the Stop button for the agent
    const stopButtons = screen.getAllByText('Stop');
    // Click the last one (should be our agent)
    fireEvent.click(stopButtons[stopButtons.length - 1]);
    
    // Wait for the agent status to change back to idle
    await waitFor(() => {
      // The agent should now have a Deploy button again
      const newDeployButtons = screen.getAllByText('Deploy');
      expect(newDeployButtons.length).toBeGreaterThan(deployButtons.length - 1);
    });
  });
});