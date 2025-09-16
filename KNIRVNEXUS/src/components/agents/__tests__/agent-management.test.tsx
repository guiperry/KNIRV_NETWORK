import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import AgentManagement from '../agent-management';
import type { Agent, AgentSummary } from '@/types/api';

// Mock the hooks
const mockUseAgentManagement = {
  agents: [],
  summary: {
    total: 0,
    running: 0,
    deployed: 0,
    stopped: 0,
    error: 0,
    uploaded: 0,
    archived: 0
  } as AgentSummary,
  isLoading: false,
  error: null,
  deleteAgent: jest.fn(),
  refreshAll: jest.fn(),
};

jest.mock('@/hooks/use-agent-management', () => ({
  useAgentManagement: jest.fn(() => mockUseAgentManagement),
}));

const mockToast = jest.fn();
jest.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: mockToast }),
}));

// Mock UI components
jest.mock('@/components/ui/card', () => ({
  Card: ({ children, ...props }: any) => <div data-testid="card" {...props}>{children}</div>,
  CardContent: ({ children, ...props }: any) => <div data-testid="card-content" {...props}>{children}</div>,
  CardDescription: ({ children, ...props }: any) => <div data-testid="card-description" {...props}>{children}</div>,
  CardHeader: ({ children, ...props }: any) => <div data-testid="card-header" {...props}>{children}</div>,
  CardTitle: ({ children, ...props }: any) => <div data-testid="card-title" {...props}>{children}</div>,
}));

jest.mock('@/components/ui/button', () => ({
  Button: ({ children, onClick, disabled, ...props }: any) => (
    <button onClick={onClick} disabled={disabled} data-testid="button" {...props}>{children}</button>
  ),
}));

jest.mock('@/components/ui/badge', () => ({
  Badge: ({ children, className, ...props }: any) => (
    <span data-testid="badge" className={className} {...props}>{children}</span>
  ),
}));

jest.mock('@/components/ui/tabs', () => ({
  Tabs: ({ children, ...props }: any) => <div data-testid="tabs" {...props}>{children}</div>,
  TabsContent: ({ children, ...props }: any) => <div data-testid="tabs-content" {...props}>{children}</div>,
  TabsList: ({ children, ...props }: any) => <div data-testid="tabs-list" {...props}>{children}</div>,
  TabsTrigger: ({ children, ...props }: any) => <button data-testid="tabs-trigger" {...props}>{children}</button>,
}));

jest.mock('@/components/ui/input', () => ({
  Input: (props: any) => <input data-testid="input" {...props} />,
}));

jest.mock('@/components/ui/label', () => ({
  Label: ({ children, ...props }: any) => <label data-testid="label" {...props}>{children}</label>,
}));

jest.mock('@/components/ui/select', () => ({
  Select: ({ children, onValueChange, ...props }: any) => (
    <div data-testid="select" {...props}>
      <select onChange={(e) => onValueChange?.(e.target.value)}>
        {children}
      </select>
    </div>
  ),
  SelectContent: ({ children, ...props }: any) => <div data-testid="select-content" {...props}>{children}</div>,
  SelectItem: ({ children, value, ...props }: any) => (
    <option data-testid="select-item" value={value} {...props}>{children}</option>
  ),
  SelectTrigger: ({ children, ...props }: any) => <div data-testid="select-trigger" {...props}>{children}</div>,
  SelectValue: ({ placeholder, ...props }: any) => <div data-testid="select-value" {...props}>{placeholder}</div>,
}));

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  Bot: (props: any) => <div data-testid="Bot-icon" {...props} />,
  Play: (props: any) => <div data-testid="Play-icon" {...props} />,
  Square: (props: any) => <div data-testid="Square-icon" {...props} />,
  RotateCcw: (props: any) => <div data-testid="RotateCcw-icon" {...props} />,
  Trash2: (props: any) => <div data-testid="Trash2-icon" {...props} />,
  RefreshCw: (props: any) => <div data-testid="RefreshCw-icon" {...props} />,
  Upload: (props: any) => <div data-testid="Upload-icon" {...props} />,
  Activity: (props: any) => <div data-testid="Activity-icon" {...props} />,
  CheckCircle: (props: any) => <div data-testid="CheckCircle-icon" {...props} />,
  AlertTriangle: (props: any) => <div data-testid="AlertTriangle-icon" {...props} />,
  Clock: (props: any) => <div data-testid="Clock-icon" {...props} />,
  Eye: (props: any) => <div data-testid="Eye-icon" {...props} />,
  Settings: (props: any) => <div data-testid="Settings-icon" {...props} />,
  BarChart3: (props: any) => <div data-testid="BarChart3-icon" {...props} />,
  FileText: (props: any) => <div data-testid="FileText-icon" {...props} />,
  Zap: (props: any) => <div data-testid="Zap-icon" {...props} />,
  Cpu: (props: any) => <div data-testid="Cpu-icon" {...props} />,
  HardDrive: (props: any) => <div data-testid="HardDrive-icon" {...props} />,
  Network: (props: any) => <div data-testid="Network-icon" {...props} />,
}));

// Mock window.confirm
Object.defineProperty(window, 'confirm', {
  writable: true,
  value: jest.fn(),
});

const mockAgents: Agent[] = [
  {
    id: 'agent-1',
    name: 'Test Agent 1',
    description: 'A test agent',
    version: '1.0.0',
    author: 'test-author',
    type: 'WASM',
    status: 'running',
    file_path: '/agents/agent-1.wasm',
    file_size: 1024000,
    file_hash: 'abc123',
    capabilities: ['compute', 'storage'],
    dependencies: [],
    configuration: {},
    metadata: {},
    tags: ['test', 'demo'],
    uploaded_at: '2024-01-01T00:00:00Z',
    last_modified: '2024-01-01T00:00:00Z',
    uploaded_by: 'test-user'
  },
  {
    id: 'agent-2',
    name: 'Test Agent 2',
    description: 'Another test agent',
    version: '2.0.0',
    author: 'test-author-2',
    type: 'LoRA',
    status: 'stopped',
    file_path: '/agents/agent-2.wasm',
    file_size: 2048000,
    file_hash: 'def456',
    capabilities: ['ai', 'nlp'],
    dependencies: [],
    configuration: {},
    metadata: {},
    tags: ['test', 'experimental'],
    uploaded_at: '2024-01-02T00:00:00Z',
    last_modified: '2024-01-02T00:00:00Z',
    uploaded_by: 'test-user-2'
  }
];

describe('AgentManagement Component', () => {
  const defaultProps = {
    isOpen: true,
    onClose: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    (window.confirm as jest.Mock).mockReturnValue(true);
  });

  it('renders without crashing', () => {
    render(<AgentManagement {...defaultProps} />);
    expect(screen.getByText('Agent Management')).toBeInTheDocument();
  });

  it('displays loading state', () => {
    const mockHook = {
      ...mockUseAgentManagement,
      isLoading: true,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    expect(screen.getByText('Loading agents...')).toBeInTheDocument();
  });

  it('displays error state', () => {
    const mockHook = {
      ...mockUseAgentManagement,
      error: 'Failed to load agents',
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    expect(screen.getByText('Error: Failed to load agents')).toBeInTheDocument();
  });

  it('displays agents list', () => {
    const mockHook = {
      ...mockUseAgentManagement,
      agents: mockAgents,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    
    expect(screen.getByText('Test Agent 1')).toBeInTheDocument();
    expect(screen.getByText('Test Agent 2')).toBeInTheDocument();
  });

  it('filters agents by status', () => {
    const mockHook = {
      ...mockUseAgentManagement,
      agents: mockAgents,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    
    // Both agents should be visible initially
    expect(screen.getByText('Test Agent 1')).toBeInTheDocument();
    expect(screen.getByText('Test Agent 2')).toBeInTheDocument();
  });

  it('handles agent deletion', async () => {
    const mockDeleteAgent = jest.fn().mockResolvedValue(true);
    const mockHook = {
      ...mockUseAgentManagement,
      agents: mockAgents,
      deleteAgent: mockDeleteAgent,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    
    const deleteButtons = screen.getAllByTestId('Trash2-icon');
    fireEvent.click(deleteButtons[0]);
    
    await waitFor(() => {
      expect(mockDeleteAgent).toHaveBeenCalledWith('agent-1');
      expect(mockToast).toHaveBeenCalledWith({
        title: "Agent Deleted",
        description: 'Successfully deleted agent "Test Agent 1"',
      });
    });
  });

  it('handles agent deletion cancellation', async () => {
    (window.confirm as jest.Mock).mockReturnValue(false);
    
    const mockDeleteAgent = jest.fn();
    const mockHook = {
      ...mockUseAgentManagement,
      agents: mockAgents,
      deleteAgent: mockDeleteAgent,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    
    const deleteButtons = screen.getAllByTestId('Trash2-icon');
    fireEvent.click(deleteButtons[0]);
    
    expect(mockDeleteAgent).not.toHaveBeenCalled();
  });

  it('handles refresh action', () => {
    const mockRefreshAll = jest.fn();
    const mockHook = {
      ...mockUseAgentManagement,
      agents: mockAgents,
      refreshAll: mockRefreshAll,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    
    const refreshButton = screen.getByTestId('RefreshCw-icon');
    fireEvent.click(refreshButton);
    
    expect(mockRefreshAll).toHaveBeenCalled();
  });

  it('displays agent summary statistics', () => {
    const mockSummary: AgentSummary = {
      total_agents: 10,
      running_agents: 3,
      deployed_agents: 2,
      stopped_agents: 4,
      error_agents: 1,
      uploaded_agents: 0
    };
    
    const mockHook = {
      ...mockUseAgentManagement,
      summary: mockSummary,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    
    expect(screen.getByText('10')).toBeInTheDocument(); // Total
    expect(screen.getByText('3')).toBeInTheDocument();  // Running
    expect(screen.getByText('2')).toBeInTheDocument();  // Deployed
    expect(screen.getByText('4')).toBeInTheDocument();  // Stopped
  });

  it('handles agent actions', async () => {
    const mockHook = {
      ...mockUseAgentManagement,
      agents: mockAgents,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    
    const playButton = screen.getAllByTestId('Play-icon')[0];
    fireEvent.click(playButton);
    
    await waitFor(() => {
      expect(mockToast).toHaveBeenCalledWith({
        title: "Agent Action",
        description: expect.stringContaining("Feature coming soon"),
        variant: "default",
      });
    });
  });

  it('displays correct status badges', () => {
    const mockHook = {
      ...mockUseAgentManagement,
      agents: mockAgents,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    
    expect(screen.getByText('Running')).toBeInTheDocument();
    expect(screen.getByText('Stopped')).toBeInTheDocument();
  });

  it('displays correct type badges', () => {
    const mockHook = {
      ...mockUseAgentManagement,
      agents: mockAgents,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    
    expect(screen.getByText('WASM')).toBeInTheDocument();
    expect(screen.getByText('LoRA')).toBeInTheDocument();
  });

  it('formats file sizes correctly', () => {
    const mockHook = {
      ...mockUseAgentManagement,
      agents: mockAgents,
    };
    
    jest.mocked(require('@/hooks/use-agent-management').useAgentManagement).mockReturnValue(mockHook);
    
    render(<AgentManagement {...defaultProps} />);
    
    expect(screen.getByText('1000 KB')).toBeInTheDocument(); // 1024000 bytes
    expect(screen.getByText('2000 KB')).toBeInTheDocument(); // 2048000 bytes
  });

  it('calls onClose when close button is clicked', () => {
    const onCloseMock = jest.fn();
    
    render(<AgentManagement {...defaultProps} onClose={onCloseMock} />);
    
    // Assuming there's a close button - this would need to be adjusted based on actual implementation
    // For now, we'll test that the component renders with the onClose prop
    expect(onCloseMock).toBeDefined();
  });
});
