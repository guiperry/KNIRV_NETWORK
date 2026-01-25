import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { DashboardWrapper } from '../dashboard-wrapper';
import * as authContext from '@/lib/auth-context';
import * as demoModeContext from '@/contexts/demo-mode-context';
import * as systemHealthHook from '@/hooks/use-system-health';
import * as dveNodesHook from '@/hooks/use-dve-nodes';
import * as validationTasksHook from '@/hooks/use-validation-tasks';
import * as cognitiveEngineHook from '@/hooks/use-cognitive-engine';

// Mock the auth context
jest.mock('@/lib/auth-context', () => ({
  useAuth: jest.fn(),
  ROLES: {
    admin: {
      permissions: ['*:*'],
      nexus_access: ['dve:*', 'validation:*', 'system:*'],
      description: 'Full administrative access',
      displayName: 'Root'
    },
    validator: {
      permissions: ['nexus:read', 'nexus:validate', 'nexus:update_assigned'],
      nexus_access: ['dve:read', 'validation:read', 'validation:execute', 'system:read'],
      description: 'Validator node operator with scoped access',
      displayName: 'Operator'
    },
    observer: {
      permissions: ['*:read'],
      nexus_access: ['dve:read', 'validation:read', 'system:read'],
      description: 'Read-only access to all services',
      displayName: 'Developer'
    }
  }
}));

// Mock the UserProfile component
jest.mock('@/components/auth/user-profile', () => ({
  UserProfile: () => <div data-testid="user-profile">User Profile</div>,
}));

// Mock the demo mode context
jest.mock('@/contexts/demo-mode-context', () => ({
  useDemoMode: jest.fn(),
}));

// Mock all the hooks
jest.mock('@/hooks/use-system-health', () => ({
  useSystemHealth: jest.fn(),
}));

jest.mock('@/hooks/use-dve-nodes', () => ({
  useDVENodes: jest.fn(),
}));

jest.mock('@/hooks/use-validation-tasks', () => ({
  useValidationTasks: jest.fn(),
}));

jest.mock('@/hooks/use-cognitive-engine', () => ({
  useCognitiveEngine: jest.fn(),
}));

// Mock Next.js router
jest.mock('next/navigation', () => ({
  useRouter: jest.fn(() => ({
    push: jest.fn(),
    replace: jest.fn(),
  })),
}));

const mockUseAuth = authContext.useAuth as jest.MockedFunction<typeof authContext.useAuth>;
const mockUseDemoMode = demoModeContext.useDemoMode as jest.MockedFunction<typeof demoModeContext.useDemoMode>;
const mockUseSystemHealth = systemHealthHook.useSystemHealth as jest.MockedFunction<typeof systemHealthHook.useSystemHealth>;
const mockUseDVENodes = dveNodesHook.useDVENodes as jest.MockedFunction<typeof dveNodesHook.useDVENodes>;
const mockUseValidationTasks = validationTasksHook.useValidationTasks as jest.MockedFunction<typeof validationTasksHook.useValidationTasks>;
const mockUseCognitiveEngine = cognitiveEngineHook.useCognitiveEngine as jest.MockedFunction<typeof cognitiveEngineHook.useCognitiveEngine>;

describe('DashboardWrapper', () => {
  const mockUser = {
    id: '1',
    email: 'test@example.com',
    role: 'admin' as const,
    name: 'Test User',
    user: 'Test User',
    nexus_access: ['dve:read', 'validation:write', 'system:admin'],
    permissions: ['*:*'],
    authenticated: true,
  };

  const mockSystemHealth = {
    systemHealth: null,
    alerts: [],
    metrics: {
      system_load: 0.5,
      memory_usage: 60,
      disk_usage: 45,
      network_throughput: 90,
      active_connections: 10,
      goroutine_count: 100,
      cpu_usage: 75,
    },
    components: {},
    isLoading: false,
    error: null as string | null,
    isConnected: false,
    fetchSystemHealth: jest.fn(),
    fetchAlerts: jest.fn(),
    fetchMetrics: jest.fn(),
    fetchComponents: jest.fn(),
    executeAction: jest.fn(),
    resolveAlert: jest.fn(),
    runDiagnostics: jest.fn(),
    addAlert: jest.fn(),
    refreshAll: jest.fn(),
    connectWebSocket: jest.fn(),
    disconnectWebSocket: jest.fn(),
  };

  const mockDVENodes = {
    nodes: [
      {
        id: 'node-1',
        name: 'Test Node',
        status: 'online' as const,
        tee_type: 'sgx' as const,
        stake_amount: 1000,
        reputation_score: 5,
        location: 'US-East',
        ip_address: '192.168.1.1',
        public_key: 'abc',
        capabilities: ['compute'],
        last_heartbeat: '2023-12-01T00:00:00Z',
        created_at: '2023-12-01T00:00:00Z',
        updated_at: '2023-12-01T00:00:00Z',
        cpu_usage: 80,
        memory_usage: 60,
        network_latency: 10,
      },
    ],
    isLoading: false,
    error: null as string | null,
    isConnected: false,
    fetchNodes: jest.fn(),
    getNode: jest.fn(),
    registerNode: jest.fn(),
    updateNode: jest.fn(),
    updateNodeStatus: jest.fn(),
    deleteNode: jest.fn(),
    getOnlineNodes: jest.fn(),
    getNodesByTEE: jest.fn(),
    refreshNodes: jest.fn(),
    connectWebSocket: jest.fn(),
    disconnectWebSocket: jest.fn(),
    getNodeEndpoints: jest.fn(),
    getNodeSSHEndpoint: jest.fn(),
    getNodeValidationEndpoint: jest.fn(),
    getNodeErrorResolutionEndpoint: jest.fn(),
  };

  const mockValidationTasks = {
    tasks: [
      {
        id: 'task-1',
        type: 'skillnode' as const,
        status: 'pending' as const,
        priority: 5,
        skill_code: undefined,
        failure_context: undefined,
        test_cases: [],
        required_tee_type: 'sgx',
        assigned_node_id: undefined,
        requested_by: 'test-user',
        parameters: {},
        completion_percentage: undefined,
        estimated_completion_time: undefined,
        created_at: '2023-12-01T00:00:00Z',
        updated_at: '2023-12-01T00:00:00Z',
        started_at: undefined,
        completed_at: undefined,
        timeout_at: '2023-12-02T00:00:00Z',
      },
    ],
    isLoading: false,
    error: null as string | null,
    isConnected: false,
    fetchTasks: jest.fn(),
    getTask: jest.fn(),
    createTask: jest.fn(),
    executeTask: jest.fn(),
    getPendingTasks: jest.fn(),
    getRunningTasks: jest.fn(),
    getTasksByType: jest.fn(),
    refreshTasks: jest.fn(),
    connectWebSocket: jest.fn(),
    disconnectWebSocket: jest.fn(),
  };

  const mockCognitiveEngine = {
    cognitiveEngine: null,
    isLoading: false,
    error: null,
    isPolling: false,
    isConnected: false,
    fetchCognitiveEngine: jest.fn(),
    performAction: jest.fn(),
    startTraining: jest.fn(),
    stopTraining: jest.fn(),
    resetMetrics: jest.fn(),
    clearConversationHistory: jest.fn(),
    updateModel: jest.fn(),
    startPolling: jest.fn(),
    stopPolling: jest.fn(),
    connectWebSocket: jest.fn(),
    disconnectWebSocket: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    
    mockUseAuth.mockReturnValue({
      user: mockUser,
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      hasPermission: jest.fn(() => true),
      hasNodeAccess: jest.fn(() => true),
      isLoading: false,
    });

    mockUseDemoMode.mockReturnValue({
      isDemoMode: false,
      toggleDemoMode: jest.fn(),
      setDemoMode: jest.fn(),
      demoData: {},
    });

    mockUseSystemHealth.mockReturnValue(mockSystemHealth);
    mockUseDVENodes.mockReturnValue(mockDVENodes);
    mockUseValidationTasks.mockReturnValue(mockValidationTasks);
    mockUseCognitiveEngine.mockReturnValue(mockCognitiveEngine);
  });

  it('should render dashboard with all panels', () => {
    mockUseAuth.mockReturnValue({
      user: { ...mockUser, authenticated: true },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      hasPermission: jest.fn(() => true),
      hasNodeAccess: jest.fn(() => true),
    });

    render(<DashboardWrapper><div>Test Content</div></DashboardWrapper>);

    expect(screen.getByText('KNIRV NEXUS')).toBeInTheDocument();
    expect(screen.getByText('Deterministic Validation Environment')).toBeInTheDocument();
    expect(screen.getByText('Test Content')).toBeInTheDocument();
  });

  it('should show loading state', () => {
    mockUseAuth.mockReturnValue({
      user: null,
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: true,
      hasPermission: jest.fn(() => true),
      hasNodeAccess: jest.fn(() => true),
    });

    render(<DashboardWrapper><div>Test Content</div></DashboardWrapper>);

    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument(); // Loading spinner
  });

  it('should show user greeting', () => {
    mockUseAuth.mockReturnValue({
      user: { ...mockUser, authenticated: true, user: 'Test User' },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      hasPermission: jest.fn(() => true),
      hasNodeAccess: jest.fn(() => true),
    });

    render(<DashboardWrapper><div>Test Content</div></DashboardWrapper>);

    expect(screen.getByText(/welcome/i)).toBeInTheDocument();
    expect(screen.getByText('Test User')).toBeInTheDocument();
  });

  it('should handle unauthorized access', () => {
    mockUseAuth.mockReturnValue({
      user: null,
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      hasPermission: jest.fn(() => true),
      hasNodeAccess: jest.fn(() => true),
    });

    render(<DashboardWrapper><div>Test Content</div></DashboardWrapper>);

    expect(screen.getByText('KNIRV NEXUS')).toBeInTheDocument();
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument(); // Login form is shown
  });
});
