import React from 'react';
import { render, screen, fireEvent, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import { DemoModeProvider, useDemoMode } from '../demo-mode-context';

// Test component to use the context
const TestComponent: React.FC = () => {
  const { isDemoMode, toggleDemoMode, demoData } = useDemoMode();

  return (
    <div>
      <div data-testid="demo-mode-status">
        {isDemoMode ? 'Demo Mode On' : 'Demo Mode Off'}
      </div>
      <button onClick={toggleDemoMode} data-testid="toggle-button">
        Toggle Demo Mode
      </button>
      <div data-testid="demo-data">
        {JSON.stringify(demoData)}
      </div>
    </div>
  );
};

// Component to test context outside provider
// This component *will* throw if used outside provider
const TestComponentWithoutProvider: React.FC = () => {
  // Call the hook directly. It will throw if no provider is found.
  const { isDemoMode } = useDemoMode();
  // If the hook did not throw, render something. This path should not be hit in the test.
  return <div data-testid="context-value">{isDemoMode.toString()}</div>;
};


describe('DemoModeContext', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear();
  });

  it('should provide default demo mode state', () => {
    render(
      <DemoModeProvider>
        <TestComponent />
      </DemoModeProvider>
    );

    expect(screen.getByTestId('demo-mode-status')).toHaveTextContent('Demo Mode On');
  });

  it('should toggle demo mode state', () => {
    render(
      <DemoModeProvider>
        <TestComponent />
      </DemoModeProvider>
    );

    const toggleButton = screen.getByTestId('toggle-button');
    const statusElement = screen.getByTestId('demo-mode-status');

    expect(statusElement).toHaveTextContent('Demo Mode On');

    act(() => {
      fireEvent.click(toggleButton);
    });

    expect(statusElement).toHaveTextContent('Demo Mode Off');

    act(() => {
      fireEvent.click(toggleButton);
    });

    expect(statusElement).toHaveTextContent('Demo Mode On');
  });

  it('should persist demo mode state in localStorage', () => {
    render(
      <DemoModeProvider>
        <TestComponent />
      </DemoModeProvider>
    );

    const toggleButton = screen.getByTestId('toggle-button');

    act(() => {
      fireEvent.click(toggleButton);
    });

    expect(localStorage.getItem('knirv-demo-mode')).toBe('false');

    act(() => {
      fireEvent.click(toggleButton);
    });

    expect(localStorage.getItem('knirv-demo-mode')).toBe('true');
  });

  it('should load demo mode state from localStorage', () => {
    // Set initial state in localStorage
    localStorage.setItem('knirv-demo-mode', 'false');

    render(
      <DemoModeProvider>
        <TestComponent />
      </DemoModeProvider>
    );

    expect(screen.getByTestId('demo-mode-status')).toHaveTextContent('Demo Mode Off');
  });

  it('should provide demo data when in demo mode', () => {
    render(
      <DemoModeProvider>
        <TestComponent />
      </DemoModeProvider>
    );

    const toggleButton = screen.getByTestId('toggle-button');
    const demoDataElement = screen.getByTestId('demo-data');

    // Initially should have demo data (since default is true)
    const initialDemoDataText = demoDataElement.textContent;
    expect(initialDemoDataText).not.toBe('{}');

    // Parse and verify initial demo data structure
    const initialDemoData = JSON.parse(initialDemoDataText || '{}');
    expect(initialDemoData).toHaveProperty('nodes');
    expect(initialDemoData).toHaveProperty('tasks');
    expect(initialDemoData).toHaveProperty('metrics');

    act(() => {
      fireEvent.click(toggleButton);
    });

    // After disabling demo mode, should have no demo data
    expect(demoDataElement).toHaveTextContent('{}');
  });

  it('should throw error when used outside provider', () => {
    // Suppress console.error for this test, as the hook is expected to throw an error
    // which React or the hook might log.
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    // We expect calling render with TestComponentWithoutProvider (which calls useDemoMode directly)
    // to throw an error because it's not wrapped in DemoModeProvider.
    expect(() => {
      render(<TestComponentWithoutProvider />);
    }).toThrow(); // Assert that the render call throws an error.

    // Check if an error was logged to the console. This confirms the hook threw.
    expect(consoleSpy).toHaveBeenCalled();

    // Restore console.error
    consoleSpy.mockRestore();
  });
  it('should handle invalid localStorage data gracefully', () => {
    // Set invalid data in localStorage
    localStorage.setItem('knirv-demo-mode', 'invalid-json');

    render(
      <DemoModeProvider>
        <TestComponent />
      </DemoModeProvider>
    );

    // Should treat invalid data as false (since 'invalid-json' !== 'true')
    expect(screen.getByTestId('demo-mode-status')).toHaveTextContent('Demo Mode Off');
  });
});
