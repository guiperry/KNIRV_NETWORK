/**
 * TaskScheduler Component Tests
 * Comprehensive test suite for task scheduler functionality
 */

import * as React from 'react';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import '@testing-library/jest-dom';
import TaskScheduler from '../../../src/components/TaskScheduler';
import { taskSchedulingService } from '../../../src/services/TaskSchedulingService';

// Mock the task scheduling service
jest.mock('../../../src/services/TaskSchedulingService', () => ({
  taskSchedulingService: {
    getAllTasks: jest.fn(),
    getTaskExecutions: jest.fn(),
    createTask: jest.fn(),
    executeTask: jest.fn(),
    deleteTask: jest.fn(),
    createWorkflow: jest.fn(),
    executeWorkflow: jest.fn()
  }
}));

describe.skip('TaskScheduler', () => {
  const mockTaskSchedulingService = taskSchedulingService as jest.Mocked<typeof taskSchedulingService>;

  // Clean up after each test to prevent DOM pollution
  afterEach(() => {
    cleanup();
    // Clear all timers to prevent hanging tests
    jest.clearAllTimers();
  });
  
  const mockTasks = [
    {
      id: 'task-1',
      name: 'Daily Backup',
      description: 'Automated daily backup task',
      type: 'maintenance' as const,
      status: 'completed' as const,
      priority: 'high' as const,
      schedule: {
        type: 'recurring' as const,
        startTime: new Date('2024-01-15T02:00:00Z'),
        interval: 86400000 // 24 hours
      },
      action: {
        type: 'system_command' as const,
        target: 'backup.sh',
        parameters: { destination: '/backup' }
      },
      runCount: 15,
      successCount: 14,
      failureCount: 1,
      nextRun: new Date('2024-01-16T02:00:00Z'),
      createdAt: new Date('2024-01-01T00:00:00Z'),
      metadata: {}
    },
    {
      id: 'task-2',
      name: 'Agent Health Check',
      description: 'Check agent status and performance',
      type: 'analysis' as const,
      status: 'running' as const,
      priority: 'medium' as const,
      schedule: {
        type: 'recurring' as const,
        startTime: new Date('2024-01-15T00:00:00Z'),
        interval: 300000 // 5 minutes
      },
      action: {
        type: 'api_call' as const,
        target: 'http://localhost:3001/api/agents/health',
        parameters: { timeout: 30000 }
      },
      runCount: 288,
      successCount: 285,
      failureCount: 3,
      nextRun: new Date('2024-01-15T10:35:00Z'),
      createdAt: new Date('2024-01-01T00:00:00Z'),
      metadata: {}
    }
  ];

  const mockExecutions = {
    'task-1': [
      {
        id: 'exec-1',
        taskId: 'task-1',
        status: 'completed' as const,
        startTime: new Date('2024-01-15T02:00:00Z'),
        endTime: new Date('2024-01-15T02:05:00Z'),
        duration: 300000,
        result: 'Backup completed successfully',
        logs: ['Starting backup...', 'Backup completed']
      }
    ],
    'task-2': [
      {
        id: 'exec-2',
        taskId: 'task-2',
        status: 'completed' as const,
        startTime: new Date('2024-01-15T10:30:00Z'),
        endTime: new Date('2024-01-15T10:30:30Z'),
        duration: 30000,
        result: 'All agents healthy',
        logs: ['Checking agents...', 'Health check completed']
      }
    ]
  };

  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
    
    mockTaskSchedulingService.getAllTasks.mockReturnValue(mockTasks);
    mockTaskSchedulingService.getTaskExecutions.mockImplementation((taskId) => 
      mockExecutions[taskId as keyof typeof mockExecutions] || []
    );
    mockTaskSchedulingService.createTask.mockResolvedValue({
      ...mockTasks[0],
      id: 'new-task-id',
      name: 'New Task'
    });
    mockTaskSchedulingService.executeTask.mockResolvedValue(mockExecutions['task-1'][0]);
    mockTaskSchedulingService.deleteTask.mockResolvedValue(undefined);
  });

  afterEach(() => {
    jest.useRealTimers();
    jest.clearAllTimers();
  });

  describe('Rendering', () => {
    it('should not render when isOpen is false', () => {
      render(<TaskScheduler isOpen={false} onClose={jest.fn()} />);
      
      expect(screen.queryByText('Task Scheduler')).not.toBeInTheDocument();
    });

    it('should render when isOpen is true', () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      expect(screen.getByText('Task Scheduler')).toBeInTheDocument();
      expect(screen.getByText('Automated workflow management')).toBeInTheDocument();
    });

    it('should render all tabs', () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      expect(screen.getByText('Tasks')).toBeInTheDocument();
      expect(screen.getByText('Create Task')).toBeInTheDocument();
      expect(screen.getByText('Executions')).toBeInTheDocument();
    });
  });

  describe('Task List Display', () => {
    it('should display tasks when available', async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('Daily Backup')).toBeInTheDocument();
        expect(screen.getByText('Agent Health Check')).toBeInTheDocument();
        expect(screen.getByText('Automated daily backup task')).toBeInTheDocument();
        expect(screen.getByText('Check agent status and performance')).toBeInTheDocument();
      });
    });

    it('should display task status icons correctly', async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        // Should show completed and running status icons
        const completedIcons = screen.getAllByTestId('check-circle-icon');
        const runningIcons = screen.getAllByTestId('refresh-cw-icon');
        
        expect(completedIcons.length).toBeGreaterThan(0);
        expect(runningIcons.length).toBeGreaterThan(0);
      });
    });

    it('should display priority badges correctly', async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('high')).toBeInTheDocument();
        expect(screen.getByText('medium')).toBeInTheDocument();
      });
    });

    it('should display task statistics', async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('15')).toBeInTheDocument(); // Run count for task-1
        expect(screen.getByText('288')).toBeInTheDocument(); // Run count for task-2
        expect(screen.getByText('93%')).toBeInTheDocument(); // Success rate for task-1 (14/15)
        expect(screen.getByText('99%')).toBeInTheDocument(); // Success rate for task-2 (285/288)
      });
    });

    it('should show empty state when no tasks exist', async () => {
      mockTaskSchedulingService.getAllTasks.mockReturnValue([]);
      
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('No scheduled tasks found')).toBeInTheDocument();
        expect(screen.getByText('Create First Task')).toBeInTheDocument();
      });
    });
  });

  describe('Task Actions', () => {
    it('should execute task when play button is clicked', async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const playButtons = screen.getAllByTestId('play-icon');
        fireEvent.click(playButtons[0]);
      });
      
      expect(mockTaskSchedulingService.executeTask).toHaveBeenCalledWith('task-1');
    });

    it('should delete task when delete button is clicked', async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const deleteButtons = screen.getAllByTestId('trash-2-icon');
        fireEvent.click(deleteButtons[0]);
      });
      
      expect(mockTaskSchedulingService.deleteTask).toHaveBeenCalledWith('task-1');
    });

    it('should refresh task list after actions', async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(1);
      });
      
      const playButtons = screen.getAllByTestId('play-icon');
      fireEvent.click(playButtons[0]);
      
      await waitFor(() => {
        expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(2);
      });
    });
  });

  describe('Tab Navigation', () => {
    it('should switch to create task tab', () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      fireEvent.click(screen.getByText('Create Task'));
      
      expect(screen.getByText('Task Name')).toBeInTheDocument();
      expect(screen.getByText('Priority')).toBeInTheDocument();
      expect(screen.getByText('Schedule Type')).toBeInTheDocument();
    });

    it('should switch to executions tab', async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      fireEvent.click(screen.getByText('Executions'));
      
      await waitFor(() => {
        expect(screen.getByText('Daily Backup')).toBeInTheDocument();
        expect(screen.getByText('Agent Health Check')).toBeInTheDocument();
      });
    });

    it('should highlight active tab', () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      const tasksTab = screen.getByText('Tasks').closest('button');
      const createTab = screen.getByText('Create Task').closest('button');
      
      // Tasks should be active by default
      expect(tasksTab).toHaveClass('text-purple-400');
      
      fireEvent.click(screen.getByText('Create Task'));
      
      expect(createTab).toHaveClass('text-purple-400');
    });
  });

  describe('Task Creation Form', () => {
    beforeEach(() => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      fireEvent.click(screen.getByText('Create Task'));
    });

    it('should render all form fields', () => {
      expect(screen.getByLabelText('Task Name')).toBeInTheDocument();
      expect(screen.getByLabelText('Priority')).toBeInTheDocument();
      expect(screen.getByLabelText('Description')).toBeInTheDocument();
      expect(screen.getByLabelText('Schedule Type')).toBeInTheDocument();
      expect(screen.getByLabelText('Start Time')).toBeInTheDocument();
      expect(screen.getByLabelText('Action Type')).toBeInTheDocument();
      expect(screen.getByLabelText('Target')).toBeInTheDocument();
      expect(screen.getByLabelText('Parameters (JSON)')).toBeInTheDocument();
    });

    it('should show interval field for recurring tasks', () => {
      const scheduleSelect = screen.getByLabelText('Schedule Type');
      fireEvent.change(scheduleSelect, { target: { value: 'recurring' } });
      
      expect(screen.getByLabelText('Interval (minutes)')).toBeInTheDocument();
    });

    it('should submit form with valid data', async () => {
      fireEvent.change(screen.getByLabelText('Task Name'), { 
        target: { value: 'Test Task' } 
      });
      fireEvent.change(screen.getByLabelText('Description'), { 
        target: { value: 'Test description' } 
      });
      fireEvent.change(screen.getByLabelText('Target'), { 
        target: { value: 'http://example.com/api' } 
      });
      
      const submitButton = screen.getByText('Create Task');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockTaskSchedulingService.createTask).toHaveBeenCalledWith(
          expect.objectContaining({
            name: 'Test Task',
            description: 'Test description'
          })
        );
      });
    });

    it('should cancel form and return to tasks tab', () => {
      const cancelButton = screen.getByText('Cancel');
      fireEvent.click(cancelButton);
      
      // Should return to tasks tab
      expect(screen.getByText('Daily Backup')).toBeInTheDocument();
    });

    it('should validate required fields', () => {
      const submitButtons = screen.getAllByText('Create Task');
      const submitButton = submitButtons[submitButtons.length - 1]; // Get the form submit button
      fireEvent.click(submitButton);

      // Form should not submit without required fields
      expect(mockTaskSchedulingService.createTask).not.toHaveBeenCalled();
    });
  });

  describe('Executions Display', () => {
    beforeEach(async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      const executionsButtons = screen.getAllByText('Executions');
      fireEvent.click(executionsButtons[0]); // Click the first Executions button

      await waitFor(() => {
        expect(mockTaskSchedulingService.getTaskExecutions).toHaveBeenCalled();
      });
    });

    it('should display execution history for tasks', () => {
      expect(screen.getByText('Daily Backup')).toBeInTheDocument();
      expect(screen.getByText('Agent Health Check')).toBeInTheDocument();
    });

    it('should show execution status and duration', () => {
      expect(screen.getByText('300000ms')).toBeInTheDocument(); // Duration for task-1
      expect(screen.getByText('30000ms')).toBeInTheDocument(); // Duration for task-2
    });

    it('should limit executions display to 5 per task', async () => {
      // Mock more executions
      const manyExecutions = Array.from({ length: 10 }, (_, i) => ({
        id: `exec-${i}`,
        taskId: 'task-1',
        status: 'completed' as const,
        startTime: new Date(),
        endTime: new Date(),
        duration: 1000,
        result: 'Success',
        logs: []
      }));

      // Update the mock to return many executions
      mockTaskSchedulingService.getTaskExecutions.mockReturnValue(manyExecutions);

      // Re-render the component to get the updated executions
      // Use a fresh render instead of cleanup/render to avoid timer conflicts
      const { rerender } = render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      rerender(<TaskScheduler isOpen={true} onClose={jest.fn()} />);

      const executionsButtons = screen.getAllByText('Executions');
      fireEvent.click(executionsButtons[0]);

      await waitFor(() => {
        // Should only show 5 executions
        const executionItems = screen.getAllByText(/Success/);
        expect(executionItems.length).toBeLessThanOrEqual(5);
      }, { timeout: 3000 });
    });
  });

  describe('Auto-refresh', () => {
    it('should auto-refresh every 30 seconds', async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);

      // Initial load
      await waitFor(() => {
        expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(1);
      }, { timeout: 2000 });

      // Fast-forward 30 seconds
      jest.advanceTimersByTime(30000);

      await waitFor(() => {
        expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(2);
      }, { timeout: 2000 });
    });

    it('should stop auto-refresh when component is closed', async () => {
      const { rerender } = render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(1);
      }, { timeout: 2000 });

      // Close component
      rerender(<TaskScheduler isOpen={false} onClose={jest.fn()} />);

      // Fast-forward time
      jest.advanceTimersByTime(30000);

      // Should not call again
      expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(1);
    });
  });

  describe('Refresh Button', () => {
    it('should refresh data when refresh button is clicked', async () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(1);
      });
      
      const refreshButton = screen.getByRole('button', { name: /refresh/i });
      fireEvent.click(refreshButton);
      
      await waitFor(() => {
        expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(2);
      });
    });
  });

  describe('Close Functionality', () => {
    it('should call onClose when close button is clicked', () => {
      const onCloseMock = jest.fn();
      render(<TaskScheduler isOpen={true} onClose={onCloseMock} />);
      
      const closeButton = screen.getByText('×');
      fireEvent.click(closeButton);
      
      expect(onCloseMock).toHaveBeenCalled();
    });
  });

  describe('Error Handling', () => {
    it('should handle task loading errors gracefully', async () => {
      mockTaskSchedulingService.getAllTasks.mockImplementation(() => {
        throw new Error('Loading failed');
      });
      
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      // Should not crash
      expect(screen.getByText('Task Scheduler')).toBeInTheDocument();
    });

    it('should handle task execution errors gracefully', async () => {
      mockTaskSchedulingService.executeTask.mockRejectedValue(new Error('Execution failed'));
      
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const playButtons = screen.getAllByTestId('play-icon');
        fireEvent.click(playButtons[0]);
      });
      
      // Should not crash
      expect(screen.getByText('Task Scheduler')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels', () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
    });

    it('should support keyboard navigation', () => {
      render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      
      const tasksTab = screen.getByText('Tasks').closest('button');
      const createTab = screen.getByText('Create Task').closest('button');
      
      expect(tasksTab).toBeInTheDocument();
      expect(createTab).toBeInTheDocument();
      
      // Tabs should be focusable
      tasksTab?.focus();
      expect(document.activeElement).toBe(tasksTab);
    });
  });
});
