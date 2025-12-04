import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { DashboardWrapper } from '../dashboard-wrapper';

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

const mockUseAuth = require('@/lib/auth-context').useAuth as jest.MockedFunction<typeof import('@/lib/auth-context').useAuth>;
const mockUseDemoMode = require('@/contexts/demo-mode-context').useDemoMode as jest.MockedFunction<typeof import('@/contexts/demo-mode-context').useDemoMode>;
const mockUseSystemHealth = require('@/hooks/use-system-health').useSystemHealth as jest.MockedFunction<typeof import('@/hooks/use-system-health').useSystemHealth>;
const mockUseDVENodes = require('@/hooks/use-dve-nodes').useDVENodes as jest.MockedFunction<typeof import('@/hooks/use-dve-nodes').useDVENodes>;
const mockUseValidationTasks = require('@/hooks/use-validation-tasks').useValidationTasks as jest.MockedFunction<typeof import('@/hooks/use-validation-tasks').useValidationTasks>;
const mockUseCognitiveEngine = require('@/hooks/use-cognitive-engine').useCognitiveEngine as jest.MockedFunction<typeof import('@/hooks/use-cognitive-engine').useCognitiveEngine>;

describe('DashboardWrapper', () => {
  const mockUser = {
    id: '1',
    email: 'test@example.com',
    role: 'admin',
    name: 'Test User',
    user: 'Test User',
    nexus_access: ['dve:read', 'validation:write', 'system:admin'],
  };

  const mockSystemHealth = {
    metrics: {
      cpu: 75,
      memory: 60,
      disk: 45,
      network: 90,
    },
    status: 'healthy' as const,
    alerts: [],
    isLoading: false,
    error: null,
    refreshMetrics: jest.fn(),
  };

  const mockDVENodes = {
    nodes: [
      {
        id: 'node-1',
        name: 'Test Node',
        status: 'active' as const,
        capabilities: ['compute'],
        stake: 1000,
        location: 'US-East',
        lastSeen: '2023-12-01T00:00:00Z',
        performance: { cpu: 80, memory: 60, storage: 40, network: 90 },
        earnings: { total: 500, thisMonth: 50, lastMonth: 45 },
      },
    ],
    isLoading: false,
    error: null,
    fetchNodes: jest.fn(),
    refreshNodes: jest.fn(),
  };

  const mockValidationTasks = {
    tasks: [
      {
        id: 'task-1',
        type: 'validation' as const,
        status: 'pending' as const,
        priority: 5,
        createdAt: '2023-12-01T00:00:00Z',
        nodeId: 'node-1',
      },
    ],
    isLoading: false,
    error: null,
    fetchTasks: jest.fn(),
    refreshTasks: jest.fn(),
  };

  const mockCognitiveEngine = {
    models: [],
    activeModel: null,
    isLoading: false,
    error: null,
    loadModel: jest.fn(),
    unloadModel: jest.fn(),
    processPrompt: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    
    mockUseAuth.mockReturnValue({
      user: mockUser,
      login: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      error: null,
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
      logout: jest.fn(),
      isLoading: false,
      error: null,
      hasPermission: jest.fn(() => true),
      hasNodeAccess: jest.fn(() => true),
    });

    render(<DashboardWrapper><div>Test Content</div></DashboardWrapper>);

    expect(screen.getByText('KNIRV NEXUS')).toBeInTheDocument();
    expect(screen.getByText('Decentralized Validation Environment')).toBeInTheDocument();
    expect(screen.getByText('Test Content')).toBeInTheDocument();
  });

  it('should show loading state', () => {
    mockUseAuth.mockReturnValue({
      user: null,
      login: jest.fn(),
      logout: jest.fn(),
      isLoading: true,
      error: null,
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
      logout: jest.fn(),
      isLoading: false,
      error: null,
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
      logout: jest.fn(),
      isLoading: false,
      error: null,
      hasPermission: jest.fn(() => true),
      hasNodeAccess: jest.fn(() => true),
    });

    render(<DashboardWrapper><div>Test Content</div></DashboardWrapper>);

    expect(screen.getByText('KNIRV NEXUS')).toBeInTheDocument();
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument(); // Login form is shown
  });
});
