/**
 * TaskScheduler Component Fixes Tests
 * Tests to verify the architectural fixes for useEffect dependencies and timer management
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

describe('TaskScheduler Architectural Fixes', () => {
  const mockTaskSchedulingService = taskSchedulingService as jest.Mocked<typeof taskSchedulingService>;

  beforeEach(() => {
    // Setup DOM container
    const container = document.createElement('div');
    container.setAttribute('id', 'root');
    document.body.appendChild(container);

    // Reset all mocks before each test
    jest.clearAllMocks();

    // Setup default mock implementations
    mockTaskSchedulingService.getAllTasks.mockReturnValue([]);
    mockTaskSchedulingService.getTaskExecutions.mockReturnValue([]);
  });

  afterEach(() => {
    cleanup();
    jest.clearAllTimers();

    // Clean up DOM container
    const container = document.getElementById('root');
    if (container) {
      document.body.removeChild(container);
    }
  });

  it('should properly manage timer lifecycle when modal opens and closes', async () => {
    jest.useFakeTimers();
    
    const { rerender } = render(
      <TaskScheduler isOpen={false} onClose={jest.fn()} />
    );

    // Verify no timers are set when modal is closed
    expect(jest.getTimerCount()).toBe(0);

    // Open the modal
    rerender(<TaskScheduler isOpen={true} onClose={jest.fn()} />);

    // Wait for useEffect to run
    await waitFor(() => {
      expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalled();
    });

    // Verify timer is set when modal is open
    expect(jest.getTimerCount()).toBe(1);

    // Close the modal
    rerender(<TaskScheduler isOpen={false} onClose={jest.fn()} />);

    // Verify timer is cleared when modal is closed
    expect(jest.getTimerCount()).toBe(0);

    jest.useRealTimers();
  });

  it('should call loadTasks on mount when modal is open', async () => {
    const container = document.getElementById('root');
    render(<TaskScheduler isOpen={true} onClose={jest.fn()} />, container ? { container } : {});

    await waitFor(() => {
      expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(1);
    }, container ? { container } : {});
  });

  it('should not call loadTasks when modal is closed', () => {
    render(<TaskScheduler isOpen={false} onClose={jest.fn()} />);

    expect(mockTaskSchedulingService.getAllTasks).not.toHaveBeenCalled();
  });

  it('should properly cleanup timers on component unmount', async () => {
    jest.useFakeTimers();
    
    const { unmount } = render(
      <TaskScheduler isOpen={true} onClose={jest.fn()} />
    );

    await waitFor(() => {
      expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalled();
    });

    // Verify timer is set
    expect(jest.getTimerCount()).toBe(1);

    // Unmount component
    unmount();

    // Verify timer is cleaned up
    expect(jest.getTimerCount()).toBe(0);

    jest.useRealTimers();
  });

  it('should handle rapid open/close cycles without timer leaks', async () => {
    jest.useFakeTimers();
    
    const { rerender } = render(
      <TaskScheduler isOpen={false} onClose={jest.fn()} />
    );

    // Rapidly open and close the modal multiple times
    for (let i = 0; i < 5; i++) {
      rerender(<TaskScheduler isOpen={true} onClose={jest.fn()} />);
      rerender(<TaskScheduler isOpen={false} onClose={jest.fn()} />);
    }

    // Verify no timer leaks
    expect(jest.getTimerCount()).toBe(0);

    jest.useRealTimers();
  });

  it('should refresh tasks every 30 seconds when modal is open', async () => {
    jest.useFakeTimers();
    
    render(<TaskScheduler isOpen={true} onClose={jest.fn()} />);

    // Wait for initial load
    await waitFor(() => {
      expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(1);
    });

    // Fast-forward 30 seconds
    jest.advanceTimersByTime(30000);

    // Verify loadTasks was called again
    await waitFor(() => {
      expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(2);
    });

    // Fast-forward another 30 seconds
    jest.advanceTimersByTime(30000);

    // Verify loadTasks was called a third time
    await waitFor(() => {
      expect(mockTaskSchedulingService.getAllTasks).toHaveBeenCalledTimes(3);
    });

    jest.useRealTimers();
  });
});
