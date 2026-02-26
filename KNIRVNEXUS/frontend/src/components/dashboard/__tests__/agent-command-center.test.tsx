import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

// ---------------------------------------------------------------------------
// Mock API module before importing the component
// ---------------------------------------------------------------------------
jest.mock('@/lib/api', () => ({
  apiRequest: jest.fn(),
  API_BASE_URL: 'http://localhost:8082',
}));

// ---------------------------------------------------------------------------
// Mock UI components (same pattern as login-form.test.tsx)
// ---------------------------------------------------------------------------
jest.mock('@/components/ui/card', () => ({
  Card: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardContent: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardDescription: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardHeader: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardTitle: ({ children, ...props }: any) => <div {...props}>{children}</div>,
}));

jest.mock('@/components/ui/button', () => ({
  Button: ({ children, onClick, disabled, ...props }: any) => (
    <button onClick={onClick} disabled={disabled} {...props}>
      {children}
    </button>
  ),
}));

jest.mock('@/components/ui/badge', () => ({
  Badge: ({ children, ...props }: any) => <span {...props}>{children}</span>,
}));

jest.mock('@/components/ui/alert', () => ({
  Alert: ({ children, ...props }: any) => <div role="alert" {...props}>{children}</div>,
  AlertDescription: ({ children, ...props }: any) => <div {...props}>{children}</div>,
}));

jest.mock('@/components/ui/progress', () => ({
  Progress: ({ value, ...props }: any) => (
    <div role="progressbar" aria-valuenow={value} {...props} />
  ),
}));

// ---------------------------------------------------------------------------
// Mock lucide-react icons (same pattern as login-form.test.tsx)
// ---------------------------------------------------------------------------
jest.mock('lucide-react', () => ({
  Bot: (props: any) => <span data-testid="Bot-icon" {...props} />,
  Play: (props: any) => <span data-testid="Play-icon" {...props} />,
  Square: (props: any) => <span data-testid="Square-icon" {...props} />,
  Plus: (props: any) => <span data-testid="Plus-icon" {...props} />,
  RefreshCw: (props: any) => <span data-testid="RefreshCw-icon" {...props} />,
  CheckCircle: (props: any) => <span data-testid="CheckCircle-icon" {...props} />,
  AlertCircle: (props: any) => <span data-testid="AlertCircle-icon" {...props} />,
  Clock: (props: any) => <span data-testid="Clock-icon" {...props} />,
  Activity: (props: any) => <span data-testid="Activity-icon" {...props} />,
  Loader2: (props: any) => <span data-testid="Loader2-icon" {...props} />,
  ChevronDown: (props: any) => <span data-testid="ChevronDown-icon" {...props} />,
  ChevronUp: (props: any) => <span data-testid="ChevronUp-icon" {...props} />,
  Monitor: (props: any) => <span data-testid="Monitor-icon" {...props} />,
  ListTodo: (props: any) => <span data-testid="ListTodo-icon" {...props} />,
}));

// ---------------------------------------------------------------------------
// Mock useToast
// ---------------------------------------------------------------------------
const mockToast = jest.fn();
jest.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: mockToast }),
}));

// ---------------------------------------------------------------------------
// Import component + API after mocks are set up
// ---------------------------------------------------------------------------
import { AgentCommandCenter } from '../agent-command-center';
import type { AgentStatusResponse, AgentTask } from '../agent-command-center';
import { apiRequest } from '@/lib/api';

const mockApiRequest = apiRequest as jest.MockedFunction<typeof apiRequest>;

// ---------------------------------------------------------------------------
// Shared test fixtures
// ---------------------------------------------------------------------------

const mockStatusStopped: AgentStatusResponse = {
  dve_id: 'dve-abc-123',
  status: 'stopped',
};

const mockStatusRunning: AgentStatusResponse = {
  dve_id: 'dve-abc-123',
  status: 'running',
  viewport_url: 'http://localhost:9000/viewport',
  started_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(), // 5 minutes ago
  task_count: 2,
};

const makeTask = (overrides: Partial<AgentTask> = {}): AgentTask => ({
  id: 'task-001',
  dve_id: 'dve-abc-123',
  title: 'Test Research Task',
  description: 'A test task description.',
  type: 'research',
  priority: 'medium',
  status: 'pending',
  created_at: new Date(Date.now() - 60 * 1000).toISOString(),
  updated_at: new Date(Date.now() - 60 * 1000).toISOString(),
  ...overrides,
});

// Helper: configure apiRequest to return status + empty tasks by default
const setupDefaultMocks = (
  statusOverride?: AgentStatusResponse,
  tasksOverride?: AgentTask[]
) => {
  mockApiRequest.mockImplementation(async (url: string) => {
    if (url.includes('/agent/status')) {
      return {
        success: true,
        data: statusOverride ?? mockStatusStopped,
        timestamp: new Date().toISOString(),
      };
    }
    if (url.includes('/agent/tasks') && !url.includes('/tasks/')) {
      return {
        success: true,
        data: tasksOverride ?? [],
        timestamp: new Date().toISOString(),
      };
    }
    return { success: true, data: null, timestamp: new Date().toISOString() };
  });
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('AgentCommandCenter', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    setupDefaultMocks();
  });

  // -------------------------------------------------------------------------
  // Rendering
  // -------------------------------------------------------------------------

  describe('Initial render', () => {
    it('renders the root container with the correct test id', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.getByTestId('agent-command-center')).toBeInTheDocument();
      });
    });

    it('displays the DVE id in the card description', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.getByText(/dve-abc-123/)).toBeInTheDocument();
      });
    });

    it('shows the Launch Agent button when agent is stopped', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.getByTestId('launch-agent-button')).toBeInTheDocument();
      });
    });

    it('does not show the Stop Agent button when agent is stopped', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.queryByTestId('stop-agent-button')).not.toBeInTheDocument();
      });
    });

    it('shows task stats cards', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.getByText('Total')).toBeInTheDocument();
        expect(screen.getByText('Pending')).toBeInTheDocument();
        expect(screen.getByText('Completed')).toBeInTheDocument();
        expect(screen.getByText('Failed')).toBeInTheDocument();
      });
    });

    it('shows empty task list message when no tasks exist', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.getByTestId('tasks-empty')).toBeInTheDocument();
      });
    });
  });

  // -------------------------------------------------------------------------
  // Agent running state
  // -------------------------------------------------------------------------

  describe('Running agent state', () => {
    beforeEach(() => {
      setupDefaultMocks(mockStatusRunning);
    });

    it('shows Stop Agent button when agent is running', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.getByTestId('stop-agent-button')).toBeInTheDocument();
      });
    });

    it('does not show Launch Agent button when agent is running', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.queryByTestId('launch-agent-button')).not.toBeInTheDocument();
      });
    });

    it('shows the viewport toggle button when agent has a viewport URL', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.getByTestId('toggle-viewport-button')).toBeInTheDocument();
      });
    });

    it('does not show the iframe by default (viewport collapsed)', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.queryByTestId('agent-viewport-iframe')).not.toBeInTheDocument();
      });
    });

    it('shows the iframe after clicking the expand viewport button', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('toggle-viewport-button')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('toggle-viewport-button'));

      expect(screen.getByTestId('agent-viewport-iframe')).toBeInTheDocument();
      expect(screen.getByTestId('agent-viewport-iframe')).toHaveAttribute(
        'src',
        'http://localhost:9000/viewport'
      );
    });

    it('collapses the viewport after clicking the collapse button', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('toggle-viewport-button')).toBeInTheDocument();
      });

      // expand
      await user.click(screen.getByTestId('toggle-viewport-button'));
      expect(screen.getByTestId('agent-viewport-iframe')).toBeInTheDocument();

      // collapse
      await user.click(screen.getByTestId('toggle-viewport-button'));
      expect(screen.queryByTestId('agent-viewport-iframe')).not.toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  // API calls on mount
  // -------------------------------------------------------------------------

  describe('Data fetching on mount', () => {
    it('calls /agent/status on mount', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(mockApiRequest).toHaveBeenCalledWith(
          expect.stringContaining('/api/dve/dve-abc-123/agent/status'),
          expect.objectContaining({ method: 'GET' })
        );
      });
    });

    it('calls /agent/tasks on mount', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(mockApiRequest).toHaveBeenCalledWith(
          expect.stringContaining('/api/dve/dve-abc-123/agent/tasks'),
          expect.objectContaining({ method: 'GET' })
        );
      });
    });

    it('renders task items when tasks are returned', async () => {
      setupDefaultMocks(mockStatusStopped, [
        makeTask({ id: 'task-001', title: 'Alpha Task' }),
        makeTask({ id: 'task-002', title: 'Beta Task' }),
      ]);

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('task-list')).toBeInTheDocument();
        expect(screen.getByText('Alpha Task')).toBeInTheDocument();
        expect(screen.getByText('Beta Task')).toBeInTheDocument();
      });
    });

    it('shows correct task stats counts', async () => {
      setupDefaultMocks(mockStatusStopped, [
        makeTask({ id: 't1', status: 'pending' }),
        makeTask({ id: 't2', status: 'running' }),
        makeTask({ id: 't3', status: 'completed' }),
        makeTask({ id: 't4', status: 'failed' }),
      ]);

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        // Total = 4
        const totals = screen.getAllByText('4');
        expect(totals.length).toBeGreaterThanOrEqual(1);
      });
    });
  });

  // -------------------------------------------------------------------------
  // Refresh button
  // -------------------------------------------------------------------------

  describe('Refresh button', () => {
    it('triggers re-fetch on refresh button click', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      // Wait for initial mount calls (2: status + tasks)
      await waitFor(() => {
        expect(mockApiRequest).toHaveBeenCalledTimes(2);
      });

      await user.click(screen.getByTestId('refresh-button'));

      await waitFor(() => {
        // Should have been called again (another 2 calls: status + tasks)
        expect(mockApiRequest).toHaveBeenCalledTimes(4);
      });
    });
  });

  // -------------------------------------------------------------------------
  // Agent Launch
  // -------------------------------------------------------------------------

  describe('Launch Agent', () => {
    it('calls POST /agent/launch when Launch button is clicked', async () => {
      const user = userEvent.setup();

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusStopped,
          timestamp: new Date().toISOString(),
        }) // status
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        }) // tasks
        .mockResolvedValueOnce({
          success: true,
          data: { ...mockStatusRunning },
          timestamp: new Date().toISOString(),
        }); // launch

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('launch-agent-button')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('launch-agent-button'));

      await waitFor(() => {
        expect(mockApiRequest).toHaveBeenCalledWith(
          expect.stringContaining('/api/dve/dve-abc-123/agent/launch'),
          expect.objectContaining({ method: 'POST' })
        );
      });
    });

    it('shows a success toast after launching the agent', async () => {
      const user = userEvent.setup();

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusStopped,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusRunning,
          timestamp: new Date().toISOString(),
        });

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('launch-agent-button')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('launch-agent-button'));

      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith(
          expect.objectContaining({ title: 'Agent Launched' })
        );
      });
    });

    it('shows an error toast when launch API call fails', async () => {
      const user = userEvent.setup();

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusStopped,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockRejectedValueOnce(new Error('Network timeout'));

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('launch-agent-button')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('launch-agent-button'));

      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith(
          expect.objectContaining({
            title: 'Launch Failed',
            variant: 'destructive',
          })
        );
      });
    });
  });

  // -------------------------------------------------------------------------
  // Agent Stop
  // -------------------------------------------------------------------------

  describe('Stop Agent', () => {
    beforeEach(() => {
      setupDefaultMocks(mockStatusRunning);
    });

    it('calls DELETE /agent when Stop button is clicked', async () => {
      const user = userEvent.setup();

      // Stop call returns success
      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusRunning,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: null,
          timestamp: new Date().toISOString(),
        });

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('stop-agent-button')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('stop-agent-button'));

      await waitFor(() => {
        expect(mockApiRequest).toHaveBeenCalledWith(
          expect.stringContaining('/api/dve/dve-abc-123/agent'),
          expect.objectContaining({ method: 'DELETE' })
        );
      });
    });

    it('shows a success toast after stopping the agent', async () => {
      const user = userEvent.setup();

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusRunning,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: null,
          timestamp: new Date().toISOString(),
        });

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('stop-agent-button')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('stop-agent-button'));

      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith(
          expect.objectContaining({ title: 'Agent Stopped' })
        );
      });
    });

    it('shows an error toast when stop API call fails', async () => {
      const user = userEvent.setup();

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusRunning,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockRejectedValueOnce(new Error('Server error'));

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('stop-agent-button')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('stop-agent-button'));

      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith(
          expect.objectContaining({
            title: 'Stop Failed',
            variant: 'destructive',
          })
        );
      });
    });
  });

  // -------------------------------------------------------------------------
  // Task form visibility
  // -------------------------------------------------------------------------

  describe('Task form toggle', () => {
    it('task form is hidden by default', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await waitFor(() => {
        expect(screen.queryByTestId('task-form')).not.toBeInTheDocument();
      });
    });

    it('shows task form when New Task button is clicked', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('new-task-toggle-button')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('new-task-toggle-button'));

      expect(screen.getByTestId('task-form')).toBeInTheDocument();
    });

    it('hides task form when Cancel button is clicked', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('new-task-toggle-button')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('new-task-toggle-button'));
      expect(screen.getByTestId('task-form')).toBeInTheDocument();

      await user.click(screen.getByTestId('cancel-task-button'));
      expect(screen.queryByTestId('task-form')).not.toBeInTheDocument();
    });

    it('hides task form when New Task toggle button is clicked again', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('new-task-toggle-button')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('new-task-toggle-button'));
      expect(screen.getByTestId('task-form')).toBeInTheDocument();

      await user.click(screen.getByTestId('new-task-toggle-button'));
      expect(screen.queryByTestId('task-form')).not.toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  // Task form fields
  // -------------------------------------------------------------------------

  describe('Task form fields', () => {
    const openForm = async (user: ReturnType<typeof userEvent.setup>) => {
      await waitFor(() => {
        expect(screen.getByTestId('new-task-toggle-button')).toBeInTheDocument();
      });
      await user.click(screen.getByTestId('new-task-toggle-button'));
      await waitFor(() => {
        expect(screen.getByTestId('task-form')).toBeInTheDocument();
      });
    };

    it('renders title, description, type and priority fields', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openForm(user);

      expect(screen.getByTestId('task-title-input')).toBeInTheDocument();
      expect(screen.getByTestId('task-description-input')).toBeInTheDocument();
      expect(screen.getByTestId('task-type-select')).toBeInTheDocument();
      expect(screen.getByTestId('task-priority-select')).toBeInTheDocument();
    });

    it('type select contains all expected options', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openForm(user);

      const typeSelect = screen.getByTestId('task-type-select');
      const options = Array.from(typeSelect.querySelectorAll('option')).map(o => o.value);
      expect(options).toEqual(['research', 'coding', 'validation', 'analysis']);
    });

    it('priority select contains all expected options', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openForm(user);

      const prioritySelect = screen.getByTestId('task-priority-select');
      const options = Array.from(prioritySelect.querySelectorAll('option')).map(o => o.value);
      expect(options).toEqual(['low', 'medium', 'high', 'critical']);
    });

    it('updates title field on user input', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openForm(user);

      const titleInput = screen.getByTestId('task-title-input');
      await user.type(titleInput, 'My new task');
      expect(titleInput).toHaveValue('My new task');
    });

    it('updates description field on user input', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openForm(user);

      const descInput = screen.getByTestId('task-description-input');
      await user.type(descInput, 'Task description text');
      expect(descInput).toHaveValue('Task description text');
    });
  });

  // -------------------------------------------------------------------------
  // Task form validation
  // -------------------------------------------------------------------------

  describe('Task form validation', () => {
    const openForm = async (user: ReturnType<typeof userEvent.setup>) => {
      await waitFor(() => {
        expect(screen.getByTestId('new-task-toggle-button')).toBeInTheDocument();
      });
      await user.click(screen.getByTestId('new-task-toggle-button'));
    };

    it('shows validation error when submitting with empty title', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openForm(user);

      await user.click(screen.getByTestId('submit-task-button'));

      await waitFor(() => {
        expect(screen.getByText(/task title is required/i)).toBeInTheDocument();
      });
    });

    it('does not call API when title is empty', async () => {
      const user = userEvent.setup();
      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openForm(user);

      // Wait for initial mount calls to settle
      await waitFor(() => {
        expect(mockApiRequest).toHaveBeenCalledTimes(2);
      });

      await user.click(screen.getByTestId('submit-task-button'));

      // Still only 2 calls (the initial mount calls)
      expect(mockApiRequest).toHaveBeenCalledTimes(2);
    });
  });

  // -------------------------------------------------------------------------
  // Task form submission
  // -------------------------------------------------------------------------

  describe('Task form submission', () => {
    const openFormAndFill = async (
      user: ReturnType<typeof userEvent.setup>,
      title = 'Research quantum algorithms',
      type = 'research',
      priority = 'high'
    ) => {
      await waitFor(() => {
        expect(screen.getByTestId('new-task-toggle-button')).toBeInTheDocument();
      });
      await user.click(screen.getByTestId('new-task-toggle-button'));

      await user.type(screen.getByTestId('task-title-input'), title);
      await user.selectOptions(screen.getByTestId('task-type-select'), type);
      await user.selectOptions(screen.getByTestId('task-priority-select'), priority);
    };

    it('calls POST /agent/tasks with correct payload', async () => {
      const user = userEvent.setup();

      const createdTask = makeTask({
        id: 'new-task-001',
        title: 'Research quantum algorithms',
        type: 'research',
        priority: 'high',
        status: 'pending',
      });

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusStopped,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: createdTask,
          timestamp: new Date().toISOString(),
        });

      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openFormAndFill(user, 'Research quantum algorithms', 'research', 'high');

      await user.click(screen.getByTestId('submit-task-button'));

      await waitFor(() => {
        expect(mockApiRequest).toHaveBeenCalledWith(
          expect.stringContaining('/api/dve/dve-abc-123/agent/tasks'),
          expect.objectContaining({
            method: 'POST',
            body: expect.stringContaining('Research quantum algorithms'),
          })
        );
      });
    });

    it('adds newly created task to the task list', async () => {
      const user = userEvent.setup();

      const createdTask = makeTask({
        id: 'new-task-001',
        title: 'Research quantum algorithms',
      });

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusStopped,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: createdTask,
          timestamp: new Date().toISOString(),
        });

      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openFormAndFill(user, 'Research quantum algorithms');

      await user.click(screen.getByTestId('submit-task-button'));

      await waitFor(() => {
        expect(screen.getByText('Research quantum algorithms')).toBeInTheDocument();
      });
    });

    it('closes the form after successful task submission', async () => {
      const user = userEvent.setup();

      const createdTask = makeTask({ id: 'new-task-002', title: 'Coding challenge' });

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusStopped,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: createdTask,
          timestamp: new Date().toISOString(),
        });

      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openFormAndFill(user, 'Coding challenge', 'coding', 'medium');

      await user.click(screen.getByTestId('submit-task-button'));

      await waitFor(() => {
        expect(screen.queryByTestId('task-form')).not.toBeInTheDocument();
      });
    });

    it('shows success toast after task creation', async () => {
      const user = userEvent.setup();

      const createdTask = makeTask({ id: 'new-task-003', title: 'Analysis run' });

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusStopped,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: createdTask,
          timestamp: new Date().toISOString(),
        });

      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openFormAndFill(user, 'Analysis run');

      await user.click(screen.getByTestId('submit-task-button'));

      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith(
          expect.objectContaining({ title: 'Task Created' })
        );
      });
    });

    it('shows error in form when task creation API fails', async () => {
      const user = userEvent.setup();

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusStopped,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockRejectedValueOnce(new Error('Backend error'));

      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openFormAndFill(user, 'Failing task');

      await user.click(screen.getByTestId('submit-task-button'));

      await waitFor(() => {
        expect(screen.getByText('Backend error')).toBeInTheDocument();
      });
    });

    it('shows destructive toast when task creation API fails', async () => {
      const user = userEvent.setup();

      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusStopped,
          timestamp: new Date().toISOString(),
        })
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        })
        .mockRejectedValueOnce(new Error('Backend error'));

      render(<AgentCommandCenter dveId="dve-abc-123" />);
      await openFormAndFill(user, 'Failing task');

      await user.click(screen.getByTestId('submit-task-button'));

      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith(
          expect.objectContaining({
            title: 'Task Creation Failed',
            variant: 'destructive',
          })
        );
      });
    });
  });

  // -------------------------------------------------------------------------
  // Task list items
  // -------------------------------------------------------------------------

  describe('Task list items', () => {
    it('renders task item with correct test id', async () => {
      setupDefaultMocks(mockStatusStopped, [makeTask({ id: 'task-xyz' })]);
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('task-item-task-xyz')).toBeInTheDocument();
      });
    });

    it('shows task status badge', async () => {
      setupDefaultMocks(mockStatusStopped, [makeTask({ id: 'task-xyz', status: 'running' })]);
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('task-status-task-xyz')).toBeInTheDocument();
        expect(screen.getByTestId('task-status-task-xyz')).toHaveTextContent('running');
      });
    });

    it('expands task detail on click', async () => {
      const user = userEvent.setup();
      const task = makeTask({ id: 'task-xyz', description: 'Detailed info here' });
      setupDefaultMocks(mockStatusStopped, [task]);

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('task-item-task-xyz')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('task-item-task-xyz'));

      await waitFor(() => {
        expect(screen.getByTestId('task-detail-task-xyz')).toBeInTheDocument();
        expect(screen.getByText('Detailed info here')).toBeInTheDocument();
      });
    });

    it('collapses task detail on second click', async () => {
      const user = userEvent.setup();
      const task = makeTask({ id: 'task-xyz', description: 'Detailed info here' });
      setupDefaultMocks(mockStatusStopped, [task]);

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('task-item-task-xyz')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('task-item-task-xyz'));
      expect(screen.getByTestId('task-detail-task-xyz')).toBeInTheDocument();

      await user.click(screen.getByTestId('task-item-task-xyz'));
      expect(screen.queryByTestId('task-detail-task-xyz')).not.toBeInTheDocument();
    });

    it('shows error_message in expanded task detail when present', async () => {
      const user = userEvent.setup();
      const task = makeTask({
        id: 'task-err',
        status: 'failed',
        error_message: 'Execution timeout exceeded',
      });
      setupDefaultMocks(mockStatusStopped, [task]);

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('task-item-task-err')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('task-item-task-err'));

      await waitFor(() => {
        expect(screen.getByText('Execution timeout exceeded')).toBeInTheDocument();
      });
    });

    it('shows all four task type badges correctly', async () => {
      setupDefaultMocks(mockStatusStopped, [
        makeTask({ id: 't-r', type: 'research' }),
        makeTask({ id: 't-c', type: 'coding' }),
        makeTask({ id: 't-v', type: 'validation' }),
        makeTask({ id: 't-a', type: 'analysis' }),
      ]);

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByText('research')).toBeInTheDocument();
        expect(screen.getByText('coding')).toBeInTheDocument();
        expect(screen.getByText('validation')).toBeInTheDocument();
        expect(screen.getByText('analysis')).toBeInTheDocument();
      });
    });
  });

  // -------------------------------------------------------------------------
  // Agent status badge rendering
  // -------------------------------------------------------------------------

  describe('Agent status badge', () => {
    it.each<[AgentStatusResponse['status']]>([
      ['stopped'],
      ['running'],
      ['starting'],
      ['stopping'],
    ])('renders status badge text for "%s"', async (status) => {
      setupDefaultMocks({ dve_id: 'dve-abc-123', status });
      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByTestId('agent-status-badge')).toHaveTextContent(status);
      });
    });
  });

  // -------------------------------------------------------------------------
  // className prop
  // -------------------------------------------------------------------------

  describe('className prop', () => {
    it('applies custom className to the root element', async () => {
      render(<AgentCommandCenter dveId="dve-abc-123" className="my-custom-class" />);
      await waitFor(() => {
        const root = screen.getByTestId('agent-command-center');
        expect(root.className).toContain('my-custom-class');
      });
    });
  });

  // -------------------------------------------------------------------------
  // Error handling: status fetch failure
  // -------------------------------------------------------------------------

  describe('Error handling', () => {
    it('shows an error toast when status fetch fails on mount', async () => {
      mockApiRequest
        .mockRejectedValueOnce(new Error('Status fetch failed'))
        .mockResolvedValueOnce({
          success: true,
          data: [],
          timestamp: new Date().toISOString(),
        });

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith(
          expect.objectContaining({
            title: 'Status Error',
            variant: 'destructive',
          })
        );
      });
    });

    it('shows tasks error alert when tasks fetch fails', async () => {
      mockApiRequest
        .mockResolvedValueOnce({
          success: true,
          data: mockStatusStopped,
          timestamp: new Date().toISOString(),
        })
        .mockRejectedValueOnce(new Error('Tasks fetch failed'));

      render(<AgentCommandCenter dveId="dve-abc-123" />);

      await waitFor(() => {
        expect(screen.getByText('Tasks fetch failed')).toBeInTheDocument();
      });
    });
  });
});
