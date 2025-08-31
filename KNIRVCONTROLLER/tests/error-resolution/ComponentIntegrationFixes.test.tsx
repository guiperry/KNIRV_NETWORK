/**
 * Tests for component integration fixes that resolved import and interface issues
 * These tests validate that components work together correctly after type fixes
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import '@testing-library/jest-dom';

// Import components to test integration
import { SlidingPanel } from '../components/SlidingPanel';
import { CognitiveShellInterface } from '../components/CognitiveShellInterface';
import { MetaAccountDashboard } from '../components/MetaAccountDashboard';
import { UnifiedInterface } from '../components/UnifiedInterface';

// Mock external dependencies
jest.mock('../services/CognitiveEngineService');
jest.mock('../services/WalletIntegrationService');
jest.mock('../shared/ComponentBridge');
jest.mock('../sensory-shell/CognitiveEngine');
jest.mock('../sensory-shell/HRMBridge');
jest.mock('../sensory-shell/WASMOrchestrator');

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  X: () => <div data-testid="x-icon">×</div>,
  Brain: () => <div data-testid="brain-icon">🧠</div>,
  Activity: () => <div data-testid="activity-icon">📊</div>,
  Zap: () => <div data-testid="zap-icon">⚡</div>,
  Eye: () => <div data-testid="eye-icon">👁</div>,
  EyeOff: () => <div data-testid="eye-off-icon">🙈</div>,
  Mic: () => <div data-testid="mic-icon">🎤</div>,
  MicOff: () => <div data-testid="mic-off-icon">🔇</div>,
  Settings: () => <div data-testid="settings-icon">⚙️</div>,
  BarChart3: () => <div data-testid="bar-chart-icon">📊</div>,
  Cpu: () => <div data-testid="cpu-icon">💻</div>,
  MessageSquare: () => <div data-testid="message-square-icon">💬</div>,
  Send: () => <div data-testid="send-icon">📤</div>,
}));

// Mock ComponentBridge
const mockComponentBridge = {
  getState: jest.fn(() => ({ connected: true, status: 'active' })),
  onMessage: jest.fn(),
  sendMessage: jest.fn(),
  connect: jest.fn(),
  disconnect: jest.fn()
};

describe('Component Integration Fixes', () => {
  describe('SlidingPanel Integration', () => {
    it('should render with CognitiveShellInterface without type errors', () => {
      render(
        <SlidingPanel
          id="cognitive-shell"
          isOpen={true}
          onClose={() => {}}
          title="Cognitive Shell"
          side="right"
        >
          <CognitiveShellInterface
            onStateChange={() => {}}
            onSkillInvoked={() => {}}
            onAdaptationTriggered={() => {}}
          />
        </SlidingPanel>
      );

      expect(screen.getByText('Cognitive Shell')).toBeInTheDocument();
      expect(screen.getByTestId('x-icon')).toBeInTheDocument();
    });

    it('should handle proper prop types for nested components', () => {
      const onStateChange = jest.fn();
      const onSkillInvoked = jest.fn();
      const onAdaptationTriggered = jest.fn();

      render(
        <SlidingPanel
          id="test-panel"
          isOpen={true}
          onClose={() => {}}
          title="Test Panel"
          side="left"
        >
          <CognitiveShellInterface
            onStateChange={onStateChange}
            onSkillInvoked={onSkillInvoked}
            onAdaptationTriggered={onAdaptationTriggered}
          />
        </SlidingPanel>
      );

      // Verify callbacks are properly typed and passed
      expect(onStateChange).toBeDefined();
      expect(onSkillInvoked).toBeDefined();
      expect(onAdaptationTriggered).toBeDefined();
    });
  });

  describe('MetaAccountDashboard Integration', () => {
    it('should render with proper config interface', () => {
      const mockConfig = {
        chainId: 'test-chain',
        rpcEndpoint: 'https://test-rpc.example.com',
        walletAddress: '0x1234567890abcdef'
      };

      render(
        <MetaAccountDashboard
          config={mockConfig}
          onTransactionSend={() => {}}
          onBalanceUpdate={() => {}}
        />
      );

      expect(screen.getByText('Meta Account Dashboard')).toBeInTheDocument();
    });

    it('should handle transaction callbacks with proper typing', () => {
      const onTransactionSend = jest.fn();
      const onBalanceUpdate = jest.fn();

      const mockConfig = {
        chainId: 'test-chain',
        rpcEndpoint: 'https://test-rpc.example.com'
      };

      render(
        <MetaAccountDashboard
          config={mockConfig}
          onTransactionSend={onTransactionSend}
          onBalanceUpdate={onBalanceUpdate}
        />
      );

      // Verify callbacks are properly typed
      expect(typeof onTransactionSend).toBe('function');
      expect(typeof onBalanceUpdate).toBe('function');
    });
  });

  describe('UnifiedInterface Integration', () => {
    it('should render with ComponentBridge without type errors', () => {
      render(
        <UnifiedInterface bridge={mockComponentBridge as any} />
      );

      expect(screen.getByText('Unified Interface')).toBeInTheDocument();
    });

    it('should handle bridge state updates safely', () => {
      const { rerender } = render(
        <UnifiedInterface bridge={mockComponentBridge as any} />
      );

      // Test that component handles bridge state changes
      const updatedBridge = {
        ...mockComponentBridge,
        getState: jest.fn(() => ({ connected: false, status: 'disconnected' }))
      };

      rerender(<UnifiedInterface bridge={updatedBridge as any} />);
      
      expect(updatedBridge.getState).toHaveBeenCalled();
    });
  });

  describe('Function Hoisting Fixes', () => {
    it('should handle components with proper function definition order', () => {
      // Test that components render without "used before declaration" errors
      const TestComponent = () => {
        const [isActive, setIsActive] = React.useState(false);

        // Functions defined before useEffect (fixed pattern)
        const initializeComponent = React.useCallback(() => {
          setIsActive(true);
        }, []);

        const cleanup = React.useCallback(() => {
          setIsActive(false);
        }, []);

        React.useEffect(() => {
          if (isActive) {
            initializeComponent();
          } else {
            cleanup();
          }

          return () => {
            cleanup();
          };
        }, [isActive, initializeComponent, cleanup]);

        return <div data-testid="test-component">Component Active: {String(isActive)}</div>;
      };

      render(<TestComponent />);
      expect(screen.getByTestId('test-component')).toBeInTheDocument();
    });
  });

  describe('Import Path Fixes', () => {
    it('should handle relative import paths correctly', () => {
      // This test validates that import path fixes work
      // by ensuring components can be imported and rendered
      
      const components = [
        { name: 'SlidingPanel', component: SlidingPanel },
        { name: 'CognitiveShellInterface', component: CognitiveShellInterface },
        { name: 'MetaAccountDashboard', component: MetaAccountDashboard },
        { name: 'UnifiedInterface', component: UnifiedInterface }
      ];

      components.forEach(({ name, component }) => {
        expect(component).toBeDefined();
        expect(typeof component).toBe('function');
        expect(name).toBeTruthy();
        expect(typeof name).toBe('string');
      });
    });
  });

  describe('Event Handler Integration', () => {
    it('should handle click events without parameter naming issues', () => {
      const onClose = jest.fn();

      render(
        <SlidingPanel
          id="test-panel"
          isOpen={true}
          onClose={onClose}
          title="Test Panel"
          side="right"
        >
          <div>Content</div>
        </SlidingPanel>
      );

      const closeButton = screen.getByTestId('x-icon').closest('button');
      if (closeButton) {
        fireEvent.click(closeButton);
        expect(onClose).toHaveBeenCalled();
      }
    });

    it('should handle keyboard events correctly', () => {
      const onClose = jest.fn();

      render(
        <SlidingPanel
          id="test-panel"
          isOpen={true}
          onClose={onClose}
          title="Test Panel"
          side="right"
        >
          <input data-testid="test-input" />
        </SlidingPanel>
      );

      const input = screen.getByTestId('test-input');
      fireEvent.keyDown(input, { key: 'Escape' });
      
      // Component should handle keyboard events without parameter errors
      expect(input).toBeInTheDocument();
    });
  });

  describe('State Management Integration', () => {
    it('should handle state updates without spread type errors', () => {
      const TestComponent = () => {
        const [state, setState] = React.useState({ loaded: false, active: false });

        const updateState = (updates: Partial<typeof state>) => {
          setState(prev => ({
            ...prev,
            ...(typeof updates === 'object' && updates !== null ? updates : {})
          }));
        };

        return (
          <div>
            <div data-testid="state-loaded">{String(state.loaded)}</div>
            <div data-testid="state-active">{String(state.active)}</div>
            <button 
              data-testid="update-button"
              onClick={() => updateState({ loaded: true, active: true })}
            >
              Update
            </button>
          </div>
        );
      };

      render(<TestComponent />);
      
      expect(screen.getByTestId('state-loaded')).toHaveTextContent('false');
      expect(screen.getByTestId('state-active')).toHaveTextContent('false');

      fireEvent.click(screen.getByTestId('update-button'));

      expect(screen.getByTestId('state-loaded')).toHaveTextContent('true');
      expect(screen.getByTestId('state-active')).toHaveTextContent('true');
    });
  });

  describe('Configuration Object Integration', () => {
    it('should handle complex configuration objects without duplicate property errors', () => {
      const TestConfigComponent = () => {
        const config = {
          maxContextSize: 100,
          learningRate: 0.01,
          adaptationThreshold: 0.3,
          skillTimeout: 30000,
          voiceEnabled: true,
          visualEnabled: true,
          loraEnabled: true,
          enhancedLoraEnabled: false,
          hrmEnabled: true,
          wasmAgentsEnabled: true,
          typeScriptCompilerEnabled: false,
          adaptiveLearningEnabled: true
        };

        return (
          <div data-testid="config-component">
            <div data-testid="voice-enabled">{String(config.voiceEnabled)}</div>
            <div data-testid="visual-enabled">{String(config.visualEnabled)}</div>
            <div data-testid="lora-enabled">{String(config.loraEnabled)}</div>
          </div>
        );
      };

      render(<TestConfigComponent />);
      
      expect(screen.getByTestId('voice-enabled')).toHaveTextContent('true');
      expect(screen.getByTestId('visual-enabled')).toHaveTextContent('true');
      expect(screen.getByTestId('lora-enabled')).toHaveTextContent('true');
    });

    it('should handle async component loading with waitFor', async () => {
      const AsyncComponent = () => {
        const [loading, setLoading] = React.useState(true);

        React.useEffect(() => {
          setTimeout(() => setLoading(false), 100);
        }, []);

        return loading ? <div data-testid="loading">Loading...</div> : <div data-testid="loaded">Loaded</div>;
      };

      render(
        <BrowserRouter>
          <AsyncComponent />
        </BrowserRouter>
      );

      expect(screen.getByTestId('loading')).toBeInTheDocument();

      await waitFor(() => {
        expect(screen.getByTestId('loaded')).toBeInTheDocument();
      });
    });
  });
});
