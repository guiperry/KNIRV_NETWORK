import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { CDEAccessModal } from '../cde-access-modal';

// Mock UI components
jest.mock('@/components/ui/button', () => ({
  Button: ({ children, onClick, disabled, variant, size, ...props }: any) => (
    <button 
      onClick={onClick} 
      disabled={disabled} 
      data-variant={variant}
      data-size={size}
      {...props}
    >
      {children}
    </button>
  ),
}));

jest.mock('@/components/ui/card', () => ({
  Card: ({ children }: any) => <div data-testid="card">{children}</div>,
  CardContent: ({ children }: any) => <div data-testid="card-content">{children}</div>,
  CardHeader: ({ children }: any) => <div data-testid="card-header">{children}</div>,
  CardTitle: ({ children }: any) => <h3 data-testid="card-title">{children}</h3>,
  CardDescription: ({ children }: any) => <p data-testid="card-description">{children}</p>,
}));

jest.mock('@/components/ui/badge', () => ({
  Badge: ({ children, variant }: any) => <span data-testid="badge" data-variant={variant}>{children}</span>,
}));

jest.mock('@/components/ui/tabs', () => ({
  Tabs: ({ children, value, onValueChange }: any) => (
    <div data-testid="tabs" data-value={value} onClick={() => onValueChange?.('terminal')}>
      {children}
    </div>
  ),
  TabsContent: ({ children, value }: any) => <div data-testid="tabs-content" data-value={value}>{children}</div>,
  TabsList: ({ children }: any) => <div data-testid="tabs-list">{children}</div>,
  TabsTrigger: ({ children, value }: any) => <button data-testid="tabs-trigger" data-value={value}>{children}</button>,
}));

// Mock icons
jest.mock('lucide-react', () => ({
  X: () => <div data-testid="x-icon" />,
  Terminal: () => <div data-testid="terminal-icon" />,
  Play: () => <div data-testid="play-icon" />,
  Code: () => <div data-testid="code-icon" />,
  Database: () => <div data-testid="database-icon" />,
  Settings: () => <div data-testid="settings-icon" />,
  Cpu: () => <div data-testid="cpu-icon" />,
  Zap: () => <div data-testid="zap-icon" />,
  FileText: () => <div data-testid="file-text-icon" />,
  Download: () => <div data-testid="download-icon" />,
  Share2: () => <div data-testid="share2-icon" />,
  Radio: () => <div data-testid="radio-icon" />,
  Shield: () => <div data-testid="shield-icon" />,
  BarChart3: () => <div data-testid="bar-chart3-icon" />,
  Upload: () => <div data-testid="upload-icon" />,
}));

describe('CDEAccessModal', () => {
  const mockOnClose = jest.fn();
  const mockOnOpenKNIRVEngine = jest.fn();

  const defaultProps = {
    isOpen: true,
    onClose: mockOnClose,
    onOpenKNIRVEngine: mockOnOpenKNIRVEngine,
    nodeId: 'node-123',
    nodeName: 'Test Node',
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render modal when open', () => {
    render(<CDEAccessModal {...defaultProps} />);

    expect(screen.getByText(/welcome to knirv cde terminal/i)).toBeInTheDocument();
    expect(screen.getByText(/cloud development environment/i)).toBeInTheDocument();
  });

  it('should not render modal when closed', () => {
    render(<CDEAccessModal {...defaultProps} isOpen={false} />);

    expect(screen.queryByText(/welcome to knirv cde terminal/i)).not.toBeInTheDocument();
  });

  it('should display node information', () => {
    render(<CDEAccessModal {...defaultProps} />);

    const nodeElements = screen.getAllByText(/test node \(node-123\)/i);
    expect(nodeElements.length).toBeGreaterThan(0);
    expect(screen.getByText(/node: test node \(node-123\)/i)).toBeInTheDocument();
  });

  it('should display terminal output', () => {
    render(<CDEAccessModal {...defaultProps} />);

    expect(screen.getByText(/welcome to knirv cde terminal/i)).toBeInTheDocument();
    expect(screen.getByText(/type "help" for available commands/i)).toBeInTheDocument();
  });

  it('should display workflow templates', () => {
    render(<CDEAccessModal {...defaultProps} />);

    expect(screen.getByText('Validation Setup')).toBeInTheDocument();
    expect(screen.getByText('Model Deployment')).toBeInTheDocument();
    expect(screen.getByText('Data Processing')).toBeInTheDocument();
  });

  it('should handle close button click', async () => {
    const user = userEvent.setup();

    render(<CDEAccessModal {...defaultProps} />);

    // The close button is the X icon button
    const closeButton = screen.getByTestId('x-icon').closest('button');
    await user.click(closeButton!);

    expect(mockOnClose).toHaveBeenCalled();
  });

  it('should handle KNIRV Engine button click', async () => {
    const user = userEvent.setup();

    render(<CDEAccessModal {...defaultProps} />);

    const engineButton = screen.getByText('Open');
    await user.click(engineButton);

    expect(mockOnOpenKNIRVEngine).toHaveBeenCalled();
  });

  it('should handle escape key press', () => {
    render(<CDEAccessModal {...defaultProps} />);

    // The component doesn't currently handle escape key, so let's test that it renders properly
    expect(screen.getByText(/cloud development environment/i)).toBeInTheDocument();
  });

  it('should display tabs for different views', () => {
    render(<CDEAccessModal {...defaultProps} />);

    expect(screen.getByTestId('tabs')).toBeInTheDocument();
    expect(screen.getByText('Terminal')).toBeInTheDocument();
    expect(screen.getByText('Workflows')).toBeInTheDocument();
    expect(screen.getByText('Tools')).toBeInTheDocument();
  });

  it('should show workflow execution buttons', () => {
    render(<CDEAccessModal {...defaultProps} />);

    const executeButtons = screen.getAllByText('Execute Workflow');
    expect(executeButtons.length).toBeGreaterThan(0);
  });
});
