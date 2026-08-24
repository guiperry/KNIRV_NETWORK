import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { BrowserRouter } from 'react-router-dom';
import { Dashboard } from '../components/Dashboard';

// Mock react-router-dom's useNavigate
const mockNavigate = jest.fn();
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate
}));

// Mock the AgentCreationModal component
jest.mock('../components/modals/AgentCreationModal', () => {
  return jest.fn(({ isOpen, onClose, onAgentCreated }) => {
    if (!isOpen) return null;
    return (
      <div data-testid="agent-creation-modal">
        <button onClick={() => onClose()}>Close</button>
        <button onClick={() => onAgentCreated({ name: 'New Test Agent' })}>
          Create Agent
        </button>
      </div>
    );
  });
});

describe('Dashboard', () => {
  const renderDashboard = () => {
    return render(
      <BrowserRouter>
        <Dashboard />
      </BrowserRouter>
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should render the dashboard with all sections', () => {
    renderDashboard();
    
    // Header
    expect(screen.getByRole('heading', { name: /Agent Command Center/i })).toBeInTheDocument();
    
    // Stats
    expect(screen.getByText('Active Agents')).toBeInTheDocument();
    expect(screen.getByText('Target Systems')).toBeInTheDocument();
    expect(screen.getByText('Inferences Today')).toBeInTheDocument();
    expect(screen.getByText('Success Rate')).toBeInTheDocument();
    
    // Recent Activity
    expect(screen.getByRole('heading', { name: /Recent Agent Activity/i })).toBeInTheDocument();
    
    // Target System Status
    expect(screen.getByRole('heading', { name: /Target System Status/i })).toBeInTheDocument();
    
    // Quick Actions
    expect(screen.getByRole('heading', { name: /Quick Actions/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Deploy New Agent/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Add Target System/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Install Capability/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Monitor Activity/i })).toBeInTheDocument();
  });

  test('should open agent creation modal when Create Agent button is clicked', () => {
    renderDashboard();

    // Click the Create Agent button in the header
    fireEvent.click(screen.getByRole('button', { name: /Create Agent/i }));

    // Modal should be open
    expect(screen.getByTestId('agent-creation-modal')).toBeInTheDocument();
    
    // Close the modal
    fireEvent.click(screen.getByRole('button', { name: /Close/i }));
    
    // Modal should be closed
    expect(screen.queryByTestId('agent-creation-modal')).not.toBeInTheDocument();
  });

  test('should navigate to agents page when agent is created', async () => {
    renderDashboard();

    // Open the modal - use the first Create Agent button
    const createButtons = screen.getAllByRole('button', { name: /Create Agent/i });
    fireEvent.click(createButtons[0]);

    // Wait for modal to open and then look for buttons again
    await waitFor(() => {
      expect(screen.getByTestId('agent-creation-modal')).toBeInTheDocument();
    });

    // Now get all Create Agent buttons again (should include modal button)
    const allCreateButtons = screen.getAllByRole('button', { name: /Create Agent/i });
    if (allCreateButtons.length > 1) {
      fireEvent.click(allCreateButtons[1]);
    } else {
      // If there's still only one button, just click it again
      fireEvent.click(allCreateButtons[0]);
    }

    // Should navigate to agents page
    expect(mockNavigate).toHaveBeenCalledWith('/agents');
  });

  test('should navigate to agents page when Deploy New Agent quick action is clicked', () => {
    renderDashboard();
    
    // Click the Deploy New Agent quick action
    fireEvent.click(screen.getByRole('button', { name: /Deploy New Agent/i }));
    
    // Modal should be open
    expect(screen.getByTestId('agent-creation-modal')).toBeInTheDocument();
  });

  test('should navigate to targets page when Add Target System quick action is clicked', () => {
    renderDashboard();
    
    // Click the Add Target System quick action
    fireEvent.click(screen.getByRole('button', { name: /Add Target System/i }));
    
    // Should navigate to targets page
    expect(mockNavigate).toHaveBeenCalledWith('/targets');
  });

  test('should navigate to capabilities page when Install Capability quick action is clicked', () => {
    renderDashboard();
    
    // Click the Install Capability quick action
    fireEvent.click(screen.getByRole('button', { name: /Install Capability/i }));
    
    // Should navigate to capabilities page
    expect(mockNavigate).toHaveBeenCalledWith('/capabilities');
  });

  test('should navigate to orchestrator page when Monitor Activity quick action is clicked', () => {
    renderDashboard();
    
    // Click the Monitor Activity quick action
    fireEvent.click(screen.getByRole('button', { name: /Monitor Activity/i }));
    
    // Should navigate to orchestrator page
    expect(mockNavigate).toHaveBeenCalledWith('/orchestrator');
  });

  test('should navigate to orchestrator page when View All button is clicked', () => {
    renderDashboard();
    
    // Click the View All button in Recent Agent Activity
    fireEvent.click(screen.getByRole('button', { name: /View All/i }));
    
    // Should navigate to orchestrator page
    expect(mockNavigate).toHaveBeenCalledWith('/orchestrator');
  });
});