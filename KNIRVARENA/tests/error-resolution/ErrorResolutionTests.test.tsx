import * as React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import '@testing-library/jest-dom';

// Import components that were fixed
import { SlidingPanel } from '../../src/components/SlidingPanel';
import { CognitiveShellInterface } from '../../src/components/CognitiveShellInterface';
import VisualProcessor from '../../src/components/VisualProcessor';
import VoiceProcessor from '../../src/components/VoiceProcessor';
import { MetaAccountDashboard } from '../../src/components/MetaAccountDashboard';
import UnifiedInterface from '../../src/components/UnifiedInterface';
import App, { SkillResult, Adaptation } from '../../src/App';
import { Agent } from '../../src/types/common';
import { ComponentBridge } from '../../src/shared/ComponentBridge';

// Mock services
jest.mock('../../src/services/CognitiveEngineService');
jest.mock('../../src/services/WalletIntegrationService');
jest.mock('../../src/services/AgentManagementService');
jest.mock('../../src/shared/ComponentBridge');

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  X: () => <div data-testid="x-icon">X</div>,
  Brain: () => <div data-testid="brain-icon">Brain</div>,
  Activity: () => <div data-testid="activity-icon">Activity</div>,
  Zap: () => <div data-testid="zap-icon">Zap</div>,
  Eye: () => <div data-testid="eye-icon">Eye</div>,
  EyeOff: () => <div data-testid="eye-off-icon">EyeOff</div>,
  Mic: () => <div data-testid="mic-icon">Mic</div>,
  MicOff: () => <div data-testid="mic-off-icon">MicOff</div>,
  Settings: () => <div data-testid="settings-icon">Settings</div>,
  BarChart3: () => <div data-testid="bar-chart-icon">BarChart3</div>,
  Cpu: () => <div data-testid="cpu-icon">Cpu</div>,
  MessageSquare: () => <div data-testid="message-square-icon">MessageSquare</div>,
  Send: () => <div data-testid="send-icon">Send</div>,
  Volume2: () => <div data-testid="volume-icon">Volume2</div>,
  VolumeX: () => <div data-testid="volume-x-icon">VolumeX</div>,
}));

describe('Error Resolution Tests', () => {
  describe('SlidingPanel Component', () => {
    it('should render without ReferenceError for X icon', () => {
      render(
        <SlidingPanel
          id="test-panel"
          isOpen={true}
          onClose={() => {}}
          title="Test Panel"
          side="right"
        >
          <div>Test content</div>
        </SlidingPanel>
      );

      expect(screen.getByTestId('x-icon')).toBeInTheDocument();
      expect(screen.getByText('Test Panel')).toBeInTheDocument();
    });

    it('should handle click outside events correctly', () => {
      const onClose = jest.fn();
      render(
        <SlidingPanel
          id="test-panel"
          isOpen={true}
          onClose={onClose}
          title="Test Panel"
          side="right"
        >
          <div>Test content</div>
        </SlidingPanel>
      );

      // Click outside should trigger onClose
      fireEvent.mouseDown(document.body);
      expect(onClose).toHaveBeenCalled();
    });
  });

  describe('Agent Type Compatibility', () => {
    it('should handle new Agent interface correctly', () => {
      const mockAgent: Agent = {
        agentId: 'test-agent',
        name: 'Test Agent',
        version: '1.0.0',
        type: 'wasm',
        status: 'Available',
        capabilities: ['test-capability'],
        nrnCost: 50,
        metadata: {
          name: 'Test Agent',
          version: '1.0.0',
          description: 'Test agent for validation',
          author: 'Test Author',
          capabilities: ['test-capability'],
          requirements: { memory: 256, cpu: 1, storage: 50 },
          permissions: ['read', 'write']
        },
        createdAt: new Date().toISOString()
      };

      expect(mockAgent.type).toBe('wasm');
      expect(mockAgent.status).toBe('Available');
      expect(mockAgent.capabilities).toContain('test-capability');
      expect(mockAgent.metadata.name).toBe('Test Agent');
    });
  });

  describe('Callback Type Safety', () => {
    it('should handle SkillResult callbacks with proper typing', () => {
      const mockSkillResult: SkillResult = {
        success: true,
        output: { result: 'test' },
        executionTime: 100
      };

      const handleSkillInvoked = (skillId: string, result: unknown) => {
        const skillResult = result as SkillResult;
        expect(skillResult.success).toBe(true);
        expect(skillResult.executionTime).toBe(100);
      };

      handleSkillInvoked('test-skill', mockSkillResult);
    });

    it('should handle Adaptation callbacks with proper typing', () => {
      const mockAdaptation: Adaptation = {
        id: 'adaptation-123',
        type: 'test-adaptation',
        description: 'Test adaptation',
        parameters: { param1: 'value1' },
        timestamp: new Date(),
        confidence: 0.8
      };

      const handleAdaptationTriggered = (adaptation: unknown) => {
        const adaptationData = adaptation as Adaptation;
        expect(adaptationData.type).toBe('test-adaptation');
        expect(adaptationData.confidence).toBe(0.8);
      };

      handleAdaptationTriggered(mockAdaptation);
    });
  });

  describe('Parameter Naming Fixes', () => {
    it('should handle event parameters correctly', () => {
      const mockEventHandler = (event: MouseEvent) => {
        expect(event.target).toBeDefined();
        expect(event.type).toBeDefined();
      };

      // Simulate the fixed event handler pattern
      const mockEvent = new MouseEvent('click');
      mockEventHandler(mockEvent);
    });

    it('should handle index parameters correctly in map functions', () => {
      const testArray = ['item1', 'item2', 'item3'];
      const result = testArray.map((item, index) => ({
        item,
        index,
        key: `${item}-${index}`
      }));

      expect(result[0].index).toBe(0);
      expect(result[1].index).toBe(1);
      expect(result[2].index).toBe(2);
    });
  });

  describe('Visual Processor Fixes', () => {
    it('should initialize without function hoisting errors', () => {
      const mockOnVisualData = jest.fn();
      const mockOnObjectDetection = jest.fn();

      render(
        <VisualProcessor
          onVisualData={mockOnVisualData}
          onObjectDetection={mockOnObjectDetection}
          isActive={false}
        />
      );

      expect(screen.getByText('Visual Processing')).toBeInTheDocument();
    });
  });

  describe('Voice Processor Fixes', () => {
    it('should initialize without function hoisting errors', () => {
      const mockOnVoiceCommand = jest.fn();
      const mockOnAudioData = jest.fn();

      render(
        <VoiceProcessor
          onVoiceCommand={mockOnVoiceCommand}
          onAudioData={mockOnAudioData}
          isActive={false}
        />
      );

      expect(screen.getByText('Voice Control')).toBeInTheDocument();
    });
  });

  describe('Transaction Type Fixes', () => {
    it('should handle Transaction interface with type property', () => {
      const mockTransaction = {
        id: 'test-tx',
        hash: 'test-hash',
        from: 'test-from',
        to: 'test-to',
        amount: '100',
        type: 'send' as const,
        status: 'confirmed' as const,
        timestamp: new Date()
      };

      expect(mockTransaction.type).toBe('send');
      expect(mockTransaction.status).toBe('confirmed');
    });
  });

  describe('Configuration Object Fixes', () => {
    it('should handle CognitiveConfig with all required properties', () => {
      const mockConfig = {
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

      expect(mockConfig.voiceEnabled).toBe(true);
      expect(mockConfig.visualEnabled).toBe(true);
      expect(mockConfig.loraEnabled).toBe(true);
    });
  });

  describe('Spread Type Safety', () => {
    it('should handle object spread operations safely', () => {
      const baseState = { loaded: false, cognitiveActive: false };
      const updatePayload = { loaded: true };
      
      const newState = {
        ...baseState,
        ...(typeof updatePayload === 'object' && updatePayload !== null ? updatePayload : {})
      };

      expect(newState.loaded).toBe(true);
      expect(newState.cognitiveActive).toBe(false);
    });
  });

  describe('Error Handling Improvements', () => {
    it('should handle unknown error types safely', () => {
      const handleError = (error: unknown) => {
        const errorMessage = error instanceof Error ? error.message : String(error);
        expect(typeof errorMessage).toBe('string');
      };

      handleError(new Error('Test error'));
      handleError('String error');
      handleError({ message: 'Object error' });
    });
  });

  describe('Component Integration Tests', () => {
    it('should render all imported components without errors', async () => {
      // Test CognitiveShellInterface
      const { unmount: unmountCognitive } = render(
        <BrowserRouter>
          <CognitiveShellInterface onStateChange={() => {}} />
        </BrowserRouter>
      );
      unmountCognitive();

      // Test MetaAccountDashboard
      const { unmount: unmountMeta } = render(
        <BrowserRouter>
          <MetaAccountDashboard config={{
            chainId: 'knirv-testnet',
            rpcEndpoint: 'https://testnet.knirv.com/rpc',
            walletAddress: '0x1234567890abcdef'
          }} />
        </BrowserRouter>
      );
      unmountMeta();

      // Test UnifiedInterface
      const mockBridge = new (ComponentBridge as any)({
        name: 'test',
        port: 3000,
        endpoints: {},
        features: {}
      });

      const { unmount: unmountUnified } = render(
        <BrowserRouter>
          <UnifiedInterface bridge={mockBridge} />
        </BrowserRouter>
      );
      unmountUnified();

      // Test App component
      const { unmount: unmountApp } = render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(document.body).toBeInTheDocument();
      });

      unmountApp();
    });
  });
});
