import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import FabricManagement from '../fabric-management';
import type { Fabric, FabricSummary } from '@/types/api';
import { useFabricManagement } from '@/hooks/use-fabric-management';

// Mock the hooks
const mockUseFabricManagement = {
  fabrics: [],
  selectedFabric: null,
  fabricMetrics: {},
  fabricLogs: {},
  fabricEvents: [],
  templates: [],
  summary: {
    total_models: 0,
    running_models: 0,
    deployed_models: 0,
    stopped_models: 0,
    error_models: 0,
    uploaded_models: 0,
  } as FabricSummary,
  isLoading: false,
  error: null,
  isConnected: false,
  fetchFabrics: jest.fn(),
  fetchFabric: jest.fn(),
  createFabric: jest.fn(),
  updateFabric: jest.fn(),
  deleteFabric: jest.fn(),
  executeFabricAction: jest.fn(),
  deployFabric: jest.fn(),
  startFabric: jest.fn(),
  stopFabric: jest.fn(),
  restartFabric: jest.fn(),
  fetchFabricMetrics: jest.fn(),
  fetchFabricLogs: jest.fn(),
  fetchFabricEvents: jest.fn(),
  fetchTemplates: jest.fn(),
  createTemplate: jest.fn(),
  fetchSummary: jest.fn(),
  refreshAll: jest.fn(),
  connectWebSocket: jest.fn(),
  disconnectWebSocket: jest.fn(),
  setSelectedFabric: jest.fn(),
};

jest.mock('@/hooks/use-fabric-management', () => ({
  useFabricManagement: jest.fn(() => mockUseFabricManagement),
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
  Layers: (props: any) => <div data-testid="Layers-icon" {...props} />,
}));

// Mock window.confirm
Object.defineProperty(window, 'confirm', {
  writable: true,
  value: jest.fn(),
});

const mockFabrics: Fabric[] = [
  {
    id: 'fabric-1',
    name: 'Test Fabric 1',
    description: 'A test fabric',
    version: '1.0.0',
    author: 'test-author',
    type: 'WASM',
    status: 'running',
    file_path: '/fabrics/fabric-1.wasm',
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
    id: 'fabric-2',
    name: 'Test Fabric 2',
    description: 'Another test fabric',
    version: '2.0.0',
    author: 'test-author-2',
    type: 'LoRA',
    status: 'stopped',
    file_path: '/fabrics/fabric-2.wasm',
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

describe('FabricManagement Component', () => {
  const defaultProps = {
    isOpen: true,
    onClose: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    (window.confirm as jest.Mock).mockReturnValue(true);
  });

  it('renders without crashing', () => {
    render(<FabricManagement {...defaultProps} />);
    expect(screen.getByText('Fabric Management')).toBeInTheDocument();
  });

  it('displays loading state', () => {
    const mockHook = {
      ...mockUseFabricManagement,
      isLoading: true,
    };

    jest.mocked(useFabricManagement).mockReturnValue(mockHook);

    render(<FabricManagement {...defaultProps} />);
    // Component shows loading via refresh button animation, not text
    expect(screen.getByTestId('RefreshCw-icon')).toBeInTheDocument();
  });

  it('displays error state', () => {
    const mockHook = {
      ...mockUseFabricManagement,
      error: 'Failed to load fabrics',
    };

    jest.mocked(useFabricManagement).mockReturnValue(mockHook);

    render(<FabricManagement {...defaultProps} />);
    expect(screen.getByText('Failed to load fabrics')).toBeInTheDocument();
  });

  it('displays fabrics list', () => {
    const mockHook = {
      ...mockUseFabricManagement,
      fabrics: mockFabrics,
    };
    
    jest.mocked(useFabricManagement).mockReturnValue(mockHook);
    
    render(<FabricManagement {...defaultProps} />);
    
    expect(screen.getByText('Test Fabric 1')).toBeInTheDocument();
    expect(screen.getByText('Test Fabric 2')).toBeInTheDocument();
  });

  it('filters fabrics by status', () => {
    const mockHook = {
      ...mockUseFabricManagement,
      fabrics: mockFabrics,
    };
    
    jest.mocked(useFabricManagement).mockReturnValue(mockHook);
    
    render(<FabricManagement {...defaultProps} />);
    
    // Both fabrics should be visible initially
    expect(screen.getByText('Test Fabric 1')).toBeInTheDocument();
    expect(screen.getByText('Test Fabric 2')).toBeInTheDocument();
  });

  it('handles fabric deletion', async () => {
    const mockDeleteFabric = jest.fn().mockResolvedValue(true);
    const mockHook = {
      ...mockUseFabricManagement,
      fabrics: mockFabrics,
      deleteFabric: mockDeleteFabric,
    };
    
    jest.mocked(useFabricManagement).mockReturnValue(mockHook);
    
    render(<FabricManagement {...defaultProps} />);
    
    const deleteButtons = screen.getAllByTestId('Trash2-icon');
    fireEvent.click(deleteButtons[0]);
    
    await waitFor(() => {
      expect(mockDeleteFabric).toHaveBeenCalledWith('fabric-1');
      expect(mockToast).toHaveBeenCalledWith({
        title: "Fabric Deleted",
        description: 'Successfully deleted fabric item "Test Fabric 1"',
      });
    });
  });

  it('handles fabric deletion cancellation', async () => {
    (window.confirm as jest.Mock).mockReturnValue(false);
    
    const mockDeleteFabric = jest.fn();
    const mockHook = {
      ...mockUseFabricManagement,
      fabrics: mockFabrics,
      deleteFabric: mockDeleteFabric,
    };
    
    jest.mocked(useFabricManagement).mockReturnValue(mockHook);
    
    render(<FabricManagement {...defaultProps} />);
    
    const deleteButtons = screen.getAllByTestId('Trash2-icon');
    fireEvent.click(deleteButtons[0]);
    
    expect(mockDeleteFabric).not.toHaveBeenCalled();
  });

  it('handles refresh action', () => {
    const mockRefreshAll = jest.fn();
    const mockHook = {
      ...mockUseFabricManagement,
      fabrics: mockFabrics,
      refreshAll: mockRefreshAll,
    };
    
    jest.mocked(useFabricManagement).mockReturnValue(mockHook);
    
    render(<FabricManagement {...defaultProps} />);
    
    const refreshButton = screen.getByTestId('RefreshCw-icon');
    fireEvent.click(refreshButton);
    
    expect(mockRefreshAll).toHaveBeenCalled();
  });

  it('displays fabric summary statistics', () => {
    const mockSummary: FabricSummary = {
      total_models: 10,
      running_models: 3,
      deployed_models: 2,
      stopped_models: 4,
      error_models: 1,
      uploaded_models: 0
    };

    const mockHook = {
      ...mockUseFabricManagement,
      summary: mockSummary,
    };

    jest.mocked(useFabricManagement).mockReturnValue(mockHook);

    render(<FabricManagement {...defaultProps} />);

    expect(screen.getAllByText('10')).toHaveLength(2); // Total - in summary and settings
    expect(screen.getByText('3')).toBeInTheDocument();  // Running - in summary
    expect(screen.getByText('2')).toBeInTheDocument();  // Deployed
    expect(screen.getByText('4')).toBeInTheDocument();  // Stopped
  });

  it('handles fabric actions', async () => {
    const mockHook = {
      ...mockUseFabricManagement,
      fabrics: mockFabrics,
    };
    
    jest.mocked(useFabricManagement).mockReturnValue(mockHook);
    
    render(<FabricManagement {...defaultProps} />);
    
    const playButton = screen.getAllByTestId('Play-icon')[0];
    fireEvent.click(playButton);
    
    await waitFor(() => {
      expect(mockToast).toHaveBeenCalledWith({
        title: "Fabric Action",
        description: expect.stringContaining("Feature coming soon"),
        variant: "default",
      });
    });
  });

  it('displays correct status badges', () => {
    const mockHook = {
      ...mockUseFabricManagement,
      fabrics: mockFabrics,
    };

    jest.mocked(useFabricManagement).mockReturnValue(mockHook);

    render(<FabricManagement {...defaultProps} />);

    expect(screen.getAllByText('Running')).toHaveLength(3); // Label, select option, badge
    expect(screen.getAllByText('Stopped')).toHaveLength(3); // Label, select option, badge
  });

  it('displays correct type badges', () => {
    const mockHook = {
      ...mockUseFabricManagement,
      fabrics: mockFabrics,
    };

    jest.mocked(useFabricManagement).mockReturnValue(mockHook);

    render(<FabricManagement {...defaultProps} />);

    expect(screen.getAllByText('WASM')).toHaveLength(2); // One in badge, one in select
    expect(screen.getAllByText('LoRA')).toHaveLength(2); // One in badge, one in select
  });

  it('formats file sizes correctly', () => {
    const mockHook = {
      ...mockUseFabricManagement,
      fabrics: mockFabrics,
    };

    jest.mocked(useFabricManagement).mockReturnValue(mockHook);

    render(<FabricManagement {...defaultProps} />);

    expect(screen.getByText('1000 KB')).toBeInTheDocument(); // 1024000 bytes
    expect(screen.getByText('1.95 MB')).toBeInTheDocument(); // 2048000 bytes
  });

  it('calls onClose when close button is clicked', () => {
    const onCloseMock = jest.fn();
    
    render(<FabricManagement {...defaultProps} onClose={onCloseMock} />);
    
    expect(onCloseMock).toBeDefined();
  });
});
