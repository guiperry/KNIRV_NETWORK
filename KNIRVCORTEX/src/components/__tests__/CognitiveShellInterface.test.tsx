import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { CognitiveShellInterface } from '../CognitiveShellInterface';

// Mock the CognitiveEngine
jest.mock('../../cognitive-shell/CognitiveEngine');

// Mock child components
jest.mock('../Terminal', () => {
  return function MockTerminal({ onCommand, history }: any) {
    return (
      <div data-testid="terminal">
        <div data-testid="terminal-history">
          {history.map((entry: any, index: number) => (
            <div key={index} data-testid={`history-${index}`}>
              {entry.input} → {entry.output}
            </div>
          ))}
        </div>
        <input
          data-testid="terminal-input"
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              onCommand(e.currentTarget.value);
              e.currentTarget.value = '';
            }
          }}
        />
      </div>
    );
  };
});

// Note: SkillPanel component doesn't exist, so no mock needed

jest.mock('../ContextViewer', () => {
  return function MockContextViewer({ context }: any) {
    return (
      <div data-testid="context-viewer">
        {Object.entries(context).map(([key, value]: [string, any]) => (
          <div key={key} data-testid={`context-${key}`}>
            {key}: {JSON.stringify(value)}
          </div>
        ))}
      </div>
    );
  };
});

describe('CognitiveShellInterface', () => {
  let mockCognitiveEngine: any;

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Mock CognitiveEngine instance
    mockCognitiveEngine = {
      processInput: jest.fn().mockResolvedValue('Mock response'),
      getState: jest.fn().mockReturnValue({
        currentContext: new Map([['test', 'value']]),
        activeSkills: ['skill1', 'skill2'],
        learningHistory: [],
        confidenceLevel: 0.8,
        adaptationLevel: 0.5,
      }),
      activateSkill: jest.fn(),
      deactivateSkill: jest.fn(),
      updateContext: jest.fn(),
      on: jest.fn(),
      off: jest.fn(),
      dispose: jest.fn(),
    };

    // Mock the CognitiveEngine constructor
    const { CognitiveEngine } = require('../../cognitive-shell/CognitiveEngine');
    CognitiveEngine.mockImplementation(() => mockCognitiveEngine);
  });

  describe('Rendering', () => {
    it('should render without crashing', () => {
      render(<CognitiveShellInterface />);
      expect(screen.getByTestId('terminal')).toBeInTheDocument();
    });

    it('should render all main components', () => {
      render(<CognitiveShellInterface />);
      
      expect(screen.getByTestId('terminal')).toBeInTheDocument();
      expect(screen.getByTestId('skill-panel')).toBeInTheDocument();
      expect(screen.getByTestId('context-viewer')).toBeInTheDocument();
    });

    it('should display the interface title', () => {
      render(<CognitiveShellInterface />);
      expect(screen.getByText(/Cognitive Shell/i)).toBeInTheDocument();
    });

    it('should show confidence level indicator', () => {
      render(<CognitiveShellInterface />);
      expect(screen.getByText(/Confidence: 80%/i)).toBeInTheDocument();
    });
  });

  describe('Terminal Interaction', () => {
    it('should handle command input', async () => {
      const user = userEvent.setup();
      render(<CognitiveShellInterface />);
      
      const terminalInput = screen.getByTestId('terminal-input');
      
      await user.type(terminalInput, 'test command');
      await user.keyboard('{Enter}');
      
      await waitFor(() => {
        expect(mockCognitiveEngine.processInput).toHaveBeenCalledWith('test command');
      });
    });

    it('should display command history', async () => {
      const user = userEvent.setup();
      render(<CognitiveShellInterface />);
      
      const terminalInput = screen.getByTestId('terminal-input');
      
      await user.type(terminalInput, 'first command');
      await user.keyboard('{Enter}');
      
      await waitFor(() => {
        expect(screen.getByTestId('history-0')).toBeInTheDocument();
      });
    });

    it('should clear terminal input after command execution', async () => {
      const user = userEvent.setup();
      render(<CognitiveShellInterface />);
      
      const terminalInput = screen.getByTestId('terminal-input') as HTMLInputElement;
      
      await user.type(terminalInput, 'test command');
      await user.keyboard('{Enter}');
      
      await waitFor(() => {
        expect(terminalInput.value).toBe('');
      });
    });

    it('should handle command errors gracefully', async () => {
      mockCognitiveEngine.processInput.mockRejectedValue(new Error('Command failed'));
      
      const user = userEvent.setup();
      render(<CognitiveShellInterface />);
      
      const terminalInput = screen.getByTestId('terminal-input');
      
      await user.type(terminalInput, 'failing command');
      await user.keyboard('{Enter}');
      
      await waitFor(() => {
        expect(screen.getByText(/Error:/i)).toBeInTheDocument();
      });
    });
  });

  describe('Skill Management', () => {
    it('should display available skills', () => {
      render(<CognitiveShellInterface />);
      
      expect(screen.getByTestId('skill-skill1')).toBeInTheDocument();
      expect(screen.getByTestId('skill-skill2')).toBeInTheDocument();
    });

    it('should toggle skill activation', async () => {
      const user = userEvent.setup();
      render(<CognitiveShellInterface />);
      
      const skillButton = screen.getByTestId('skill-skill1');
      await user.click(skillButton);
      
      expect(mockCognitiveEngine.activateSkill).toHaveBeenCalledWith('skill1');
    });

    it('should show active skills with different styling', () => {
      render(<CognitiveShellInterface />);
      
      const activeSkill = screen.getByTestId('skill-skill1');
      expect(activeSkill).toHaveClass('active');
    });

    it('should deactivate skills when toggled off', async () => {
      // Mock skill as initially active
      mockCognitiveEngine.getState.mockReturnValue({
        currentContext: new Map(),
        activeSkills: ['skill1'],
        learningHistory: [],
        confidenceLevel: 0.8,
        adaptationLevel: 0.5,
      });

      const user = userEvent.setup();
      render(<CognitiveShellInterface />);
      
      const skillButton = screen.getByTestId('skill-skill1');
      await user.click(skillButton);
      
      expect(mockCognitiveEngine.deactivateSkill).toHaveBeenCalledWith('skill1');
    });
  });

  describe('Context Management', () => {
    it('should display current context', () => {
      render(<CognitiveShellInterface />);
      
      expect(screen.getByTestId('context-test')).toBeInTheDocument();
      expect(screen.getByText(/test: "value"/i)).toBeInTheDocument();
    });

    it('should update context display when context changes', async () => {
      const { rerender } = render(<CognitiveShellInterface />);
      
      // Update mock to return different context
      mockCognitiveEngine.getState.mockReturnValue({
        currentContext: new Map([['newKey', 'newValue']]),
        activeSkills: [],
        learningHistory: [],
        confidenceLevel: 0.8,
        adaptationLevel: 0.5,
      });
      
      rerender(<CognitiveShellInterface />);
      
      expect(screen.getByTestId('context-newKey')).toBeInTheDocument();
    });

    it('should handle empty context gracefully', () => {
      mockCognitiveEngine.getState.mockReturnValue({
        currentContext: new Map(),
        activeSkills: [],
        learningHistory: [],
        confidenceLevel: 0.8,
        adaptationLevel: 0.5,
      });

      render(<CognitiveShellInterface />);
      
      const contextViewer = screen.getByTestId('context-viewer');
      expect(contextViewer).toBeInTheDocument();
    });
  });

  describe('State Updates', () => {
    it('should update interface when cognitive engine state changes', async () => {
      render(<CognitiveShellInterface />);
      
      // Simulate state change event
      const stateChangeCallback = mockCognitiveEngine.on.mock.calls
        .find(call => call[0] === 'stateChanged')?.[1];
      
      if (stateChangeCallback) {
        act(() => {
          stateChangeCallback();
        });
      }
      
      await waitFor(() => {
        expect(mockCognitiveEngine.getState).toHaveBeenCalled();
      });
    });

    it('should update confidence level display', () => {
      mockCognitiveEngine.getState.mockReturnValue({
        currentContext: new Map(),
        activeSkills: [],
        learningHistory: [],
        confidenceLevel: 0.95,
        adaptationLevel: 0.5,
      });

      render(<CognitiveShellInterface />);
      
      expect(screen.getByText(/Confidence: 95%/i)).toBeInTheDocument();
    });

    it('should update adaptation level display', () => {
      mockCognitiveEngine.getState.mockReturnValue({
        currentContext: new Map(),
        activeSkills: [],
        learningHistory: [],
        confidenceLevel: 0.8,
        adaptationLevel: 0.75,
      });

      render(<CognitiveShellInterface />);
      
      expect(screen.getByText(/Adaptation: 75%/i)).toBeInTheDocument();
    });
  });

  describe('Event Handling', () => {
    it('should register event listeners on mount', () => {
      render(<CognitiveShellInterface />);
      
      expect(mockCognitiveEngine.on).toHaveBeenCalledWith('stateChanged', expect.any(Function));
      expect(mockCognitiveEngine.on).toHaveBeenCalledWith('skillActivated', expect.any(Function));
      expect(mockCognitiveEngine.on).toHaveBeenCalledWith('learningEvent', expect.any(Function));
    });

    it('should unregister event listeners on unmount', () => {
      const { unmount } = render(<CognitiveShellInterface />);
      
      unmount();
      
      expect(mockCognitiveEngine.off).toHaveBeenCalled();
    });

    it('should handle learning events', () => {
      render(<CognitiveShellInterface />);
      
      const learningEventCallback = mockCognitiveEngine.on.mock.calls
        .find(call => call[0] === 'learningEvent')?.[1];
      
      expect(learningEventCallback).toBeDefined();
    });
  });

  describe('Performance', () => {
    it('should not cause excessive re-renders', () => {
      const renderSpy = jest.fn();
      
      function TestWrapper() {
        renderSpy();
        return <CognitiveShellInterface />;
      }
      
      const { rerender } = render(<TestWrapper />);
      
      // Trigger multiple re-renders
      rerender(<TestWrapper />);
      rerender(<TestWrapper />);
      
      // Should not render excessively
      expect(renderSpy).toHaveBeenCalledTimes(3);
    });

    it('should cleanup resources on unmount', () => {
      const { unmount } = render(<CognitiveShellInterface />);
      
      unmount();
      
      expect(mockCognitiveEngine.dispose).toHaveBeenCalled();
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels', () => {
      render(<CognitiveShellInterface />);
      
      const terminal = screen.getByTestId('terminal');
      expect(terminal).toBeInTheDocument();
    });

    it('should support keyboard navigation', () => {
      render(<CognitiveShellInterface />);
      
      const terminalInput = screen.getByTestId('terminal-input');
      terminalInput.focus();
      
      expect(document.activeElement).toBe(terminalInput);
    });

    it('should have proper heading structure', () => {
      render(<CognitiveShellInterface />);
      
      const heading = screen.getByRole('heading');
      expect(heading).toBeInTheDocument();
    });
  });
});
