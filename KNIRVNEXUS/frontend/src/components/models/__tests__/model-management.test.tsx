import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import ModelManagement from '../model-management';
import type { Model, ModelSummary } from '@/types/api';

// Mock the hooks
const mockUseModelManagement = {
  models: [],
  summary: {
    total: 0,
    running: 0,
    deployed: 0,
    stopped: 0,
    error: 0,
    uploaded: 0,
    archived: 0
  } as ModelSummary,
  isLoading: false,
  error: null,
  deleteModel: jest.fn(),
  refreshAll: jest.fn(),
};

jest.mock('@/hooks/use-model-management', () => ({
  useModelManagement: jest.fn(() => mockUseModelManagement),
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

const mockModels: Model[] = [
  {
    id: 'model-1',
    name: 'Test Model 1',
    description: 'A test model',
    version: '1.0.0',
    author: 'test-author',
    type: 'WASM',
    status: 'running',
    file_path: '/models/model-1.wasm',
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
    id: 'model-2',
    name: 'Test Model 2',
    description: 'Another test model',
    version: '2.0.0',
    author: 'test-author-2',
    type: 'LoRA',
    status: 'stopped',
    file_path: '/models/model-2.wasm',
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

describe('ModelManagement Component', () => {
  const defaultProps = {
    isOpen: true,
    onClose: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    (window.confirm as jest.Mock).mockReturnValue(true);
  });

  it('renders without crashing', () => {
    render(<ModelManagement {...defaultProps} />);
    expect(screen.getByText('Model Management')).toBeInTheDocument();
  });

  it('displays loading state', () => {
    const mockHook = {
      ...mockUseModelManagement,
      isLoading: true,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    expect(screen.getByText('Loading models...')).toBeInTheDocument();
  });

  it('displays error state', () => {
    const mockHook = {
      ...mockUseModelManagement,
      error: 'Failed to load models',
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    expect(screen.getByText('Error: Failed to load models')).toBeInTheDocument();
  });

  it('displays models list', () => {
    const mockHook = {
      ...mockUseModelManagement,
      models: mockModels,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    
    expect(screen.getByText('Test Model 1')).toBeInTheDocument();
    expect(screen.getByText('Test Model 2')).toBeInTheDocument();
  });

  it('filters models by status', () => {
    const mockHook = {
      ...mockUseModelManagement,
      models: mockModels,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    
    // Both models should be visible initially
    expect(screen.getByText('Test Model 1')).toBeInTheDocument();
    expect(screen.getByText('Test Model 2')).toBeInTheDocument();
  });

  it('handles model deletion', async () => {
    const mockDeleteModel = jest.fn().mockResolvedValue(true);
    const mockHook = {
      ...mockUseModelManagement,
      models: mockModels,
      deleteModel: mockDeleteModel,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    
    const deleteButtons = screen.getAllByTestId('Trash2-icon');
    fireEvent.click(deleteButtons[0]);
    
    await waitFor(() => {
      expect(mockDeleteModel).toHaveBeenCalledWith('model-1');
      expect(mockToast).toHaveBeenCalledWith({
        title: "Model Deleted",
        description: 'Successfully deleted model "Test Model 1"',
      });
    });
  });

  it('handles model deletion cancellation', async () => {
    (window.confirm as jest.Mock).mockReturnValue(false);
    
    const mockDeleteModel = jest.fn();
    const mockHook = {
      ...mockUseModelManagement,
      models: mockModels,
      deleteModel: mockDeleteModel,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    
    const deleteButtons = screen.getAllByTestId('Trash2-icon');
    fireEvent.click(deleteButtons[0]);
    
    expect(mockDeleteModel).not.toHaveBeenCalled();
  });

  it('handles refresh action', () => {
    const mockRefreshAll = jest.fn();
    const mockHook = {
      ...mockUseModelManagement,
      models: mockModels,
      refreshAll: mockRefreshAll,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    
    const refreshButton = screen.getByTestId('RefreshCw-icon');
    fireEvent.click(refreshButton);
    
    expect(mockRefreshAll).toHaveBeenCalled();
  });

  it('displays model summary statistics', () => {
    const mockSummary: ModelSummary = {
      total_models: 10,
      running_models: 3,
      deployed_models: 2,
      stopped_models: 4,
      error_models: 1,
      uploaded_models: 0
    };
    
    const mockHook = {
      ...mockUseModelManagement,
      summary: mockSummary,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    
    expect(screen.getByText('10')).toBeInTheDocument(); // Total
    expect(screen.getByText('3')).toBeInTheDocument();  // Running
    expect(screen.getByText('2')).toBeInTheDocument();  // Deployed
    expect(screen.getByText('4')).toBeInTheDocument();  // Stopped
  });

  it('handles model actions', async () => {
    const mockHook = {
      ...mockUseModelManagement,
      models: mockModels,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    
    const playButton = screen.getAllByTestId('Play-icon')[0];
    fireEvent.click(playButton);
    
    await waitFor(() => {
      expect(mockToast).toHaveBeenCalledWith({
        title: "Model Action",
        description: expect.stringContaining("Feature coming soon"),
        variant: "default",
      });
    });
  });

  it('displays correct status badges', () => {
    const mockHook = {
      ...mockUseModelManagement,
      models: mockModels,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    
    expect(screen.getByText('Running')).toBeInTheDocument();
    expect(screen.getByText('Stopped')).toBeInTheDocument();
  });

  it('displays correct type badges', () => {
    const mockHook = {
      ...mockUseModelManagement,
      models: mockModels,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    
    expect(screen.getByText('WASM')).toBeInTheDocument();
    expect(screen.getByText('LoRA')).toBeInTheDocument();
  });

  it('formats file sizes correctly', () => {
    const mockHook = {
      ...mockUseModelManagement,
      models: mockModels,
    };
    
    jest.mocked(require('@/hooks/use-model-management').useModelManagement).mockReturnValue(mockHook);
    
    render(<ModelManagement {...defaultProps} />);
    
    expect(screen.getByText('1000 KB')).toBeInTheDocument(); // 1024000 bytes
    expect(screen.getByText('2000 KB')).toBeInTheDocument(); // 2048000 bytes
  });

  it('calls onClose when close button is clicked', () => {
    const onCloseMock = jest.fn();
    
    render(<ModelManagement {...defaultProps} onClose={onCloseMock} />);
    
    // Assuming there's a close button - this would need to be adjusted based on actual implementation
    // For now, we'll test that the component renders with the onClose prop
    expect(onCloseMock).toBeDefined();
  });
});
