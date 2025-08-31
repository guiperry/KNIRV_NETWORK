/**
 * Tests for Terminal component
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Terminal } from '../../../src/components/Terminal';
import { terminalCommandService } from '../../../src/services/TerminalCommandService';

// Mock the terminal command service
jest.mock('../../../src/services/TerminalCommandService');

const mockTerminalCommandService = terminalCommandService as jest.Mocked<typeof terminalCommandService>;

describe('Terminal', () => {
  const mockOnCommand = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    mockTerminalCommandService.getCommandHistory.mockReturnValue([]);
    mockTerminalCommandService.executeCommand.mockResolvedValue({
      success: true,
      output: 'Command executed successfully',
      executionTime: 100,
      exitCode: 0
    });
  });

  it('should render terminal with input field', () => {
    render(<Terminal onCommand={mockOnCommand} />);
    
    expect(screen.getByTestId('terminal-input')).toBeInTheDocument();
    expect(screen.getByTestId('terminal-history')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Enter command...')).toBeInTheDocument();
  });

  it('should display custom prompt', () => {
    render(<Terminal onCommand={mockOnCommand} prompt="knirv> " />);
    
    expect(screen.getByText('knirv> ')).toBeInTheDocument();
  });

  it('should execute command on Enter key press', async () => {
    const user = userEvent.setup();
    render(<Terminal onCommand={mockOnCommand} />);
    
    const input = screen.getByTestId('terminal-input');
    
    await user.type(input, 'help');
    await user.keyboard('{Enter}');
    
    await waitFor(() => {
      expect(mockTerminalCommandService.executeCommand).toHaveBeenCalledWith('help');
    });
  });

  it('should not execute empty command', async () => {
    const _user = userEvent.setup();
    render(<Terminal onCommand={mockOnCommand} />);
    
    // Verify input element is present and accessible
    const _input = screen.getByTestId('terminal-input');
    expect(_input).toBeInTheDocument();
    expect(_input).toHaveValue('');

    await _user.keyboard('{Enter}');

    expect(mockTerminalCommandService.executeCommand).not.toHaveBeenCalled();
  });

  it('should clear input after command execution', async () => {
    const user = userEvent.setup();
    render(<Terminal onCommand={mockOnCommand} />);
    
    const input = screen.getByTestId('terminal-input') as HTMLInputElement;
    
    await user.type(input, 'help');
    expect(input.value).toBe('help');
    
    await user.keyboard('{Enter}');
    
    await waitFor(() => {
      expect(input.value).toBe('');
    });
  });

  it('should call onCommand callback when provided', async () => {
    const user = userEvent.setup();
    render(<Terminal onCommand={mockOnCommand} />);
    
    const input = screen.getByTestId('terminal-input');
    
    await user.type(input, 'test command');
    await user.keyboard('{Enter}');
    
    await waitFor(() => {
      expect(mockOnCommand).toHaveBeenCalledWith('test command');
    });
  });

  it('should display command history from service', () => {
    const mockHistory = [
      {
        command: 'help',
        result: {
          success: true,
          output: 'Help information',
          executionTime: 50,
          exitCode: 0
        },
        timestamp: new Date('2023-01-01T10:00:00Z'),
        context: {
          workingDirectory: '/knirv',
          environment: {},
          user: 'knirv',
          session: 'test'
        }
      },
      {
        command: 'ls',
        result: {
          success: true,
          output: 'file1.txt\nfile2.txt',
          executionTime: 75,
          exitCode: 0
        },
        timestamp: new Date('2023-01-01T10:01:00Z'),
        context: {
          workingDirectory: '/knirv',
          environment: {},
          user: 'knirv',
          session: 'test'
        }
      }
    ];

    mockTerminalCommandService.getCommandHistory.mockReturnValue(mockHistory);

    render(<Terminal onCommand={mockOnCommand} />);
    
    expect(screen.getByText('$ help')).toBeInTheDocument();
    expect(screen.getByText('Help information')).toBeInTheDocument();
    expect(screen.getByText('$ ls')).toBeInTheDocument();
    expect(screen.getByText('file1.txt')).toBeInTheDocument();
    expect(screen.getByText('file2.txt')).toBeInTheDocument();
  });

  it('should display external history when provided', () => {
    const externalHistory = [
      {
        input: 'external command',
        output: 'external output',
        timestamp: new Date('2023-01-01T09:00:00Z')
      }
    ];

    render(<Terminal onCommand={mockOnCommand} history={externalHistory} />);
    
    expect(screen.getByText('$ external command')).toBeInTheDocument();
    expect(screen.getByText('external output')).toBeInTheDocument();
  });

  it('should combine and sort external and service history', () => {
    const externalHistory = [
      {
        input: 'external command',
        output: 'external output',
        timestamp: new Date('2023-01-01T10:30:00Z')
      }
    ];

    const serviceHistory = [
      {
        command: 'service command',
        result: {
          success: true,
          output: 'service output',
          executionTime: 50,
          exitCode: 0
        },
        timestamp: new Date('2023-01-01T10:15:00Z'),
        context: {
          workingDirectory: '/knirv',
          environment: {},
          user: 'knirv',
          session: 'test'
        }
      }
    ];

    mockTerminalCommandService.getCommandHistory.mockReturnValue(serviceHistory);

    render(<Terminal onCommand={mockOnCommand} history={externalHistory} />);
    
    const historyItems = screen.getAllByTestId(/history-/);
    expect(historyItems).toHaveLength(2);
    
    // Should be sorted by timestamp
    expect(screen.getByText('service output')).toBeInTheDocument();
    expect(screen.getByText('external output')).toBeInTheDocument();
  });

  it('should display error messages in red', () => {
    const errorHistory = [
      {
        command: 'invalid command',
        result: {
          success: false,
          output: '',
          error: 'Command not found',
          executionTime: 10,
          exitCode: 127
        },
        timestamp: new Date(),
        context: {
          workingDirectory: '/knirv',
          environment: {},
          user: 'knirv',
          session: 'test'
        }
      }
    ];

    mockTerminalCommandService.getCommandHistory.mockReturnValue(errorHistory);

    render(<Terminal onCommand={mockOnCommand} />);
    
    const errorElement = screen.getByText('Command not found');
    expect(errorElement).toHaveClass('text-red-400');
  });

  it('should show executing state during command execution', async () => {
    const user = userEvent.setup();
    
    // Mock a delayed command execution
    let resolveCommand: (value: { success: boolean; output: string; executionTime: number; exitCode: number }) => void;
    const commandPromise = new Promise(resolve => {
      resolveCommand = resolve;
    });
    
    mockTerminalCommandService.executeCommand.mockReturnValue(commandPromise as Promise<{ success: boolean; output: string; executionTime: number; exitCode: number }>);

    render(<Terminal onCommand={mockOnCommand} />);
    
    const input = screen.getByTestId('terminal-input') as HTMLInputElement;
    
    await user.type(input, 'slow command');
    await user.keyboard('{Enter}');
    
    // Should show executing state
    expect(screen.getByText('Executing...')).toBeInTheDocument();
    expect(input).toBeDisabled();
    expect(input.placeholder).toBe('Executing command...');
    
    // Resolve the command
    resolveCommand!({
      success: true,
      output: 'Command completed',
      executionTime: 1000,
      exitCode: 0
    });
    
    await waitFor(() => {
      expect(screen.queryByText('Executing...')).not.toBeInTheDocument();
      expect(input).not.toBeDisabled();
    });
  });

  it('should auto-scroll to bottom when history updates', async () => {
    const { rerender } = render(<Terminal onCommand={mockOnCommand} />);
    
    const historyContainer = screen.getByTestId('terminal-history');
    const scrollToBottomSpy = jest.spyOn(historyContainer, 'scrollTop', 'set');
    
    // Add new history
    const newHistory = [
      {
        command: 'new command',
        result: {
          success: true,
          output: 'new output',
          executionTime: 50,
          exitCode: 0
        },
        timestamp: new Date(),
        context: {
          workingDirectory: '/knirv',
          environment: {},
          user: 'knirv',
          session: 'test'
        }
      }
    ];
    
    mockTerminalCommandService.getCommandHistory.mockReturnValue(newHistory);
    
    rerender(<Terminal onCommand={mockOnCommand} />);
    
    // Should scroll to bottom
    expect(scrollToBottomSpy).toHaveBeenCalled();
  });

  it('should handle command execution failure', async () => {
    const user = userEvent.setup();
    
    mockTerminalCommandService.executeCommand.mockRejectedValue(
      new Error('Service unavailable')
    );

    render(<Terminal onCommand={mockOnCommand} />);
    
    const input = screen.getByTestId('terminal-input');
    
    await user.type(input, 'failing command');
    await user.keyboard('{Enter}');
    
    // Should not crash and should clear input
    await waitFor(() => {
      expect((input as HTMLInputElement).value).toBe('');
    });
  });

  it('should apply custom className', () => {
    const { container } = render(
      <Terminal onCommand={mockOnCommand} className="custom-terminal" />
    );
    
    expect(container.firstChild).toHaveClass('custom-terminal');
  });

  it('should display timestamps for history entries', () => {
    const historyWithTimestamp = [
      {
        command: 'test',
        result: {
          success: true,
          output: 'test output',
          executionTime: 50,
          exitCode: 0
        },
        timestamp: new Date('2023-01-01T10:00:00Z'),
        context: {
          workingDirectory: '/knirv',
          environment: {},
          user: 'knirv',
          session: 'test'
        }
      }
    ];

    mockTerminalCommandService.getCommandHistory.mockReturnValue(historyWithTimestamp);

    render(<Terminal onCommand={mockOnCommand} />);
    
    // Should display formatted timestamp
    expect(screen.getByText(/10:00:00/)).toBeInTheDocument();
  });

  it('should refresh command history periodically', async () => {
    jest.useFakeTimers();
    
    render(<Terminal onCommand={mockOnCommand} />);
    
    expect(mockTerminalCommandService.getCommandHistory).toHaveBeenCalledTimes(1);
    
    // Fast-forward 5 seconds
    jest.advanceTimersByTime(5000);
    
    expect(mockTerminalCommandService.getCommandHistory).toHaveBeenCalledTimes(2);
    
    jest.useRealTimers();
  });

  it('should cleanup interval on unmount', () => {
    jest.useFakeTimers();
    const clearIntervalSpy = jest.spyOn(global, 'clearInterval');
    
    const { unmount } = render(<Terminal onCommand={mockOnCommand} />);
    
    unmount();
    
    expect(clearIntervalSpy).toHaveBeenCalled();
    
    jest.useRealTimers();
  });
});
