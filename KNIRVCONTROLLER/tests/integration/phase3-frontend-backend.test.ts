/**
 * Phase 3: Frontend-Backend Integration Tests
 * Tests for AgentManager, CognitiveShellInterface, Skills page, and QRScanner integration
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import React from 'react';

// Mock external dependencies
global.fetch = jest.fn();

// Mock QR Scanner
jest.mock('qr-scanner', () => {
  return jest.fn().mockImplementation(() => ({
    start: jest.fn().mockResolvedValue(undefined),
    stop: jest.fn().mockResolvedValue(undefined),
    destroy: jest.fn().mockResolvedValue(undefined)
  }));
});

describe('Phase 3.1: AgentManager Backend Integration', () => {
  beforeEach(() => {
    // Mock successful API responses
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        agents: [
          {
            id: 'agent-1',
            name: 'Real Agent 1',
            type: 'KNIRV-CORTEX',
            status: 'active',
            performance: 94,
            tasks: 12,
            specialization: ['error-detection', 'system-analysis'],
            nrnCost: 50,
            lastActive: '2 min ago',
            wasmBytes: new Uint8Array([1, 2, 3, 4]),
            compilationMetrics: {
              compilationTime: 1500,
              wasmSize: 2048,
              optimizationLevel: 'O2'
            }
          }
        ]
      })
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should fetch real agent data from backend API', async () => {
    const { AgentManager } = await import('../../src/components/AgentManager');
    
    render(<AgentManager />);

    // Should call the API
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/agents', expect.any(Object));
    });

    // Should display real agent data (not mock data)
    await waitFor(() => {
      expect(screen.getByText('Real Agent 1')).toBeInTheDocument();
      expect(screen.queryByText('CyberPunk Agent #7804')).not.toBeInTheDocument(); // Mock data should not appear
    });
  });

  it('should compile agents using real AgentCoreCompiler', async () => {
    const { AgentManager } = await import('../../src/components/AgentManager');
    
    render(<AgentManager />);

    // Find and click compile button
    await waitFor(() => {
      const compileButton = screen.getByText('Compile Agent');
      fireEvent.click(compileButton);
    });

    // Should call compilation API
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/agents/compile', expect.objectContaining({
        method: 'POST'
      }));
    });
  });

  it('should assign agents with backend persistence', async () => {
    const { AgentManager } = await import('../../src/components/AgentManager');
    
    render(<AgentManager />);

    // Wait for agents to load
    await waitFor(() => {
      expect(screen.getByText('Real Agent 1')).toBeInTheDocument();
    });

    // Find and click assign button
    const assignButton = screen.getByText('Assign Task');
    fireEvent.click(assignButton);

    // Should call assignment API
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/agents/assign', expect.objectContaining({
        method: 'POST'
      }));
    });
  });

  it('should display real-time agent status updates', async () => {
    const { AgentManager } = await import('../../src/components/AgentManager');
    
    // Mock WebSocket for real-time updates
    const mockWebSocket = {
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
      send: jest.fn(),
      close: jest.fn()
    };
    
    global.WebSocket = jest.fn(() => mockWebSocket) as any;

    render(<AgentManager />);

    // Simulate WebSocket message
    const statusUpdate = {
      type: 'agent-status-update',
      agentId: 'agent-1',
      status: 'busy',
      currentTask: 'Processing data'
    };

    const messageHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'message'
    )?.[1];

    if (messageHandler) {
      messageHandler({ data: JSON.stringify(statusUpdate) });
    }

    // Should update agent status in real-time
    await waitFor(() => {
      expect(screen.getByText('busy')).toBeInTheDocument();
    });
  });
});

describe('Phase 3.2: CognitiveShellInterface Integration', () => {
  beforeEach(() => {
    // Mock WASM orchestrator responses
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        success: true,
        result: 'Real cognitive processing result',
        processingTime: 150,
        wasmModulesLoaded: ['agent-core', 'hrm-model'],
        cognitiveState: {
          activeSkills: ['error-handling'],
          memoryUsage: 45,
          processingLoad: 23
        }
      })
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should connect to real WASM orchestrator', async () => {
    const { CognitiveShellInterface } = await import('../../src/components/CognitiveShellInterface');
    
    render(<CognitiveShellInterface />);

    // Should initialize with real WASM orchestrator
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/cognitive/initialize', expect.any(Object));
    });

    // Should display real cognitive state
    await waitFor(() => {
      expect(screen.getByText(/Memory Usage: 45%/)).toBeInTheDocument();
      expect(screen.getByText(/Processing Load: 23%/)).toBeInTheDocument();
    });
  });

  it('should generate ErrorContext on cognitive failures', async () => {
    const { CognitiveShellInterface } = await import('../../src/components/CognitiveShellInterface');
    
    // Mock cognitive processing failure
    (global.fetch as jest.Mock).mockRejectedValueOnce(new Error('Cognitive processing failed'));

    render(<CognitiveShellInterface />);

    // Trigger cognitive processing
    const inputField = screen.getByPlaceholderText('Enter cognitive input...');
    fireEvent.change(inputField, { target: { value: 'test cognitive input' } });
    fireEvent.click(screen.getByText('Process'));

    // Should generate ErrorContext and query KNIRVGRAPH
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/knirvgraph/query-similar-errors', expect.objectContaining({
        method: 'POST'
      }));
    });
  });

  it('should replace mock responses with KNIRVGRAPH-based skill resolution', async () => {
    const { CognitiveShellInterface } = await import('../../src/components/CognitiveShellInterface');
    
    render(<CognitiveShellInterface />);

    // Process input
    const inputField = screen.getByPlaceholderText('Enter cognitive input...');
    fireEvent.change(inputField, { target: { value: 'complex cognitive task' } });
    fireEvent.click(screen.getByText('Process'));

    // Should not show mock responses
    await waitFor(() => {
      expect(screen.queryByText(/Mock skill execution result/)).not.toBeInTheDocument();
      expect(screen.getByText('Real cognitive processing result')).toBeInTheDocument();
    });
  });
});

describe('Phase 3.3: Skills Page LoRA Integration', () => {
  beforeEach(() => {
    // Mock KNIRVROUTER skill responses
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        skills: [
          {
            id: 'skill-1',
            name: 'Real Error Handling Skill',
            description: 'Advanced error handling and recovery',
            category: 'error-handling',
            complexity: 8,
            nrnCost: 25,
            isActive: true,
            skillNodeUri: 'knirv://skills/error-handling/v1.0.0',
            loraAdapter: {
              rank: 16,
              alpha: 32.0,
              weightsA: new Float32Array(128),
              weightsB: new Float32Array(128)
            }
          }
        ]
      })
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should connect to KNIRVROUTER for skill retrieval', async () => {
    const { Skills } = await import('../../src/pages/Skills');
    
    render(<Skills />);

    // Should fetch skills from KNIRVROUTER
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/knirvrouter/skills', expect.any(Object));
    });

    // Should display real skills (not mock data)
    await waitFor(() => {
      expect(screen.getByText('Real Error Handling Skill')).toBeInTheDocument();
      expect(screen.queryByText('Code Analysis')).not.toBeInTheDocument(); // Mock skill should not appear
    });
  });

  it('should implement SkillNode URI to LoRA adapter resolution', async () => {
    const { Skills } = await import('../../src/pages/Skills');
    
    render(<Skills />);

    // Wait for skills to load
    await waitFor(() => {
      expect(screen.getByText('Real Error Handling Skill')).toBeInTheDocument();
    });

    // Click on skill to view details
    fireEvent.click(screen.getByText('Real Error Handling Skill'));

    // Should resolve SkillNode URI to LoRA adapter
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/knirvrouter/resolve-skill', expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('knirv://skills/error-handling/v1.0.0')
      }));
    });
  });

  it('should integrate NRN token payments for skill usage', async () => {
    const { Skills } = await import('../../src/pages/Skills');
    
    render(<Skills />);

    // Wait for skills to load
    await waitFor(() => {
      expect(screen.getByText('Real Error Handling Skill')).toBeInTheDocument();
    });

    // Click activate skill button
    fireEvent.click(screen.getByText('Activate Skill'));

    // Should process NRN payment
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/wallet/pay-nrn', expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"amount":"25"')
      }));
    });
  });
});

describe('Phase 3.4: QR Scanner Wallet Integration', () => {
  beforeEach(() => {
    // Mock wallet API responses
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        success: true,
        sessionId: 'session-123',
        walletAddress: 'knirv1wallet123',
        nrnBalance: 1000,
        connectionStatus: 'connected'
      })
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should complete QR to wallet connection flow', async () => {
    const { QRScanner } = await import('../../src/components/QRScanner');
    
    const mockOnScan = jest.fn();
    const mockOnClose = jest.fn();

    render(<QRScanner onScan={mockOnScan} onClose={mockOnClose} isOpen={true} />);

    // Simulate QR code scan
    const qrData = JSON.stringify({
      version: '1.0',
      type: 'wallet-connection',
      session_id: 'session-123',
      desktop_id: 'desktop-456',
      endpoint: 'ws://localhost:8080',
      public_key: 'pubkey123',
      signature: 'signature123'
    });

    // Trigger scan result
    mockOnScan(qrData);

    // Should establish wallet connection
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/wallet/connect', expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('session-123')
      }));
    });
  });

  it('should implement real wallet operations via backend API', async () => {
    const { QRScanner } = await import('../../src/components/QRScanner');
    
    const mockOnScan = jest.fn();
    const mockOnClose = jest.fn();

    render(<QRScanner onScan={mockOnScan} onClose={mockOnClose} isOpen={true} />);

    // Simulate successful wallet connection
    const qrData = JSON.stringify({
      type: 'wallet-connection',
      session_id: 'session-123'
    });

    mockOnScan(qrData);

    // Should fetch wallet balance
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/wallet/balance', expect.any(Object));
    });

    // Should display real wallet information
    await waitFor(() => {
      expect(screen.getByText(/Balance: 1000 NRN/)).toBeInTheDocument();
    });
  });

  it('should manage NRN tokens for skill payments', async () => {
    const { QRScanner } = await import('../../src/components/QRScanner');
    
    const mockOnScan = jest.fn();
    const mockOnClose = jest.fn();

    render(<QRScanner onScan={mockOnScan} onClose={mockOnClose} isOpen={true} />);

    // Simulate payment request
    const paymentData = JSON.stringify({
      type: 'nrn-payment',
      amount: '25',
      recipient: 'knirv1skill123',
      skillId: 'error-handling-skill'
    });

    mockOnScan(paymentData);

    // Should process NRN payment
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/wallet/transfer-nrn', expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"amount":"25"')
      }));
    });
  });

  it('should implement secure session management', async () => {
    const { QRScanner } = await import('../../src/components/QRScanner');
    
    const mockOnScan = jest.fn();
    const mockOnClose = jest.fn();

    render(<QRScanner onScan={mockOnScan} onClose={mockOnClose} isOpen={true} />);

    // Simulate session establishment
    const sessionData = JSON.stringify({
      type: 'secure-session',
      session_id: 'session-123',
      encrypted_payload: 'encrypted-data',
      signature: 'signature123'
    });

    mockOnScan(sessionData);

    // Should validate session security
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/session/validate', expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('signature123')
      }));
    });
  });
});

describe('Phase 3: Frontend-Backend Integration Verification', () => {
  beforeEach(() => {
    // Mock all API endpoints for integration test
    (global.fetch as jest.Mock)
      .mockResolvedValueOnce({ // Agents API
        ok: true,
        json: () => Promise.resolve({ agents: [] })
      })
      .mockResolvedValueOnce({ // Cognitive API
        ok: true,
        json: () => Promise.resolve({ success: true })
      })
      .mockResolvedValueOnce({ // Skills API
        ok: true,
        json: () => Promise.resolve({ skills: [] })
      })
      .mockResolvedValueOnce({ // Wallet API
        ok: true,
        json: () => Promise.resolve({ success: true })
      });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should have all frontend components connected to real backend APIs', async () => {
    // This test verifies that no component uses mock data
    const { AgentManager } = await import('../../src/components/AgentManager');
    const { CognitiveShellInterface } = await import('../../src/components/CognitiveShellInterface');
    const { Skills } = await import('../../src/pages/Skills');

    // Render all components
    render(
      <div>
        <AgentManager />
        <CognitiveShellInterface />
        <Skills />
      </div>
    );

    // All components should make real API calls
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/agents', expect.any(Object));
      expect(global.fetch).toHaveBeenCalledWith('/api/cognitive/initialize', expect.any(Object));
      expect(global.fetch).toHaveBeenCalledWith('/api/knirvrouter/skills', expect.any(Object));
    });

    // No mock data should be displayed
    expect(screen.queryByText('CyberPunk Agent #7804')).not.toBeInTheDocument();
    expect(screen.queryByText('Mock skill execution result')).not.toBeInTheDocument();
    expect(screen.queryByText('Code Analysis')).not.toBeInTheDocument(); // Mock skill
  });
});
