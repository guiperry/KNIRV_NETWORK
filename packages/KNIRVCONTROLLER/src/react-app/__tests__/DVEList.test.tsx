import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import DVEList from '@/react-app/pages/DVEList';

// Mock the Layout component
vi.mock('@/react-app/components/Layout', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div data-testid="layout-wrapper">{children}</div>,
}));

// Mock lucide-react icons
vi.mock('lucide-react', () => ({
  Cpu: () => <div data-testid="icon-cpu" />,
  Zap: () => <div data-testid="icon-zap" />,
  Shield: () => <div data-testid="icon-shield" />,
  MapPin: () => <div data-testid="icon-map-pin" />,
  Activity: () => <div data-testid="icon-activity" />,
  ChevronRight: () => <div data-testid="icon-chevron-right" />,
  Plus: () => <div data-testid="icon-plus" />,
  MessageSquare: () => <div data-testid="icon-message-square" />,
  BadgeCheck: () => <div data-testid="icon-badge-check" />,
  Wifi: () => <div data-testid="icon-wifi" />,
  WifiOff: () => <div data-testid="icon-wifi-off" />,
}));

describe('DVEList', () => {
  it('renders with Layout wrapper', () => {
    render(<DVEList />);
    expect(screen.getByTestId('layout-wrapper')).toBeInTheDocument();
  });

  it('renders "MY DVEs" section header', () => {
    render(<DVEList />);
    expect(screen.getByText('My DVEs')).toBeInTheDocument();
  });

  it('renders "NETWORK DVEs" section header', () => {
    render(<DVEList />);
    expect(screen.getByText('NETWORK DVEs')).toBeInTheDocument();
  });

  it('renders DVE card names (DVE-Alpha, DVE-Beta, DVE-Gamma)', () => {
    render(<DVEList />);
    expect(screen.getByText('DVE-Alpha')).toBeInTheDocument();
    expect(screen.getByText('DVE-Beta')).toBeInTheDocument();
    expect(screen.getByText('DVE-Gamma')).toBeInTheDocument();
  });

  it('renders the [+ New DVE] button or create section', () => {
    render(<DVEList />);
    expect(screen.getByText('Add New DVE')).toBeInTheDocument();
  });

  it('renders the KNIRVCONTROLLER header', () => {
    render(<DVEList />);
    expect(screen.getByText('KNIRV')).toBeInTheDocument();
    expect(screen.getByText('CONTROLLER')).toBeInTheDocument();
  });

  it('renders the Vault Balance section', () => {
    render(<DVEList />);
    expect(screen.getByText('Vault Balance')).toBeInTheDocument();
  });

  it('renders the node count indicator', () => {
    render(<DVEList />);
    expect(screen.getByText('3 nodes')).toBeInTheDocument();
  });

  it('renders network DVE nodes list', () => {
    render(<DVEList />);
    expect(screen.getByText('DVE-Delta')).toBeInTheDocument();
    expect(screen.getByText('DVE-Epsilon')).toBeInTheDocument();
  });
});
