import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import App from '@/react-app/App';

// Mock all page components
vi.mock('@/react-app/pages/DVEList', () => ({
  default: () => <div data-testid="page-dve-list">DVEList Page</div>,
}));

vi.mock('@/react-app/pages/DVECreate', () => ({
  default: () => <div data-testid="page-dve-create">DVECreate Page</div>,
}));

vi.mock('@/react-app/pages/Home', () => ({
  default: () => <div data-testid="page-home">Home Page</div>,
}));

vi.mock('@/react-app/pages/Scanner', () => ({
  default: () => <div data-testid="page-scanner">Scanner Page</div>,
}));

vi.mock('@/react-app/pages/Badges', () => ({
  default: () => <div data-testid="page-badges">Badges Page</div>,
}));

vi.mock('@/react-app/pages/Settings', () => ({
  default: () => <div data-testid="page-settings">Settings Page</div>,
}));

vi.mock('@/react-app/pages/UDC', () => ({
  default: () => <div data-testid="page-udc">UDC Page</div>,
}));

vi.mock('@/react-app/pages/VaultPage', () => ({
  default: () => <div data-testid="page-vault">VaultPage</div>,
}));

vi.mock('@/react-app/pages/CognitiveEngineChat', () => ({
  default: () => <div data-testid="page-cognitive">CognitiveEngineChat Page</div>,
}));

vi.mock('@/react-app/pages/AgentChat', () => ({
  default: () => <div data-testid="page-agent-chat">AgentChat Page</div>,
}));

vi.mock('@/react-app/pages/Onboarding', () => ({
  default: () => <div data-testid="page-onboarding">Onboarding Page</div>,
}));

describe('App', () => {
  it('renders Onboarding at root path "/"', () => {
    window.history.pushState({}, '', '/');
    render(<App />);
    expect(screen.getByTestId('page-onboarding')).toBeInTheDocument();
  });

  it('renders DVEList at "/dves"', () => {
    window.history.pushState({}, '', '/dves');
    render(<App />);
    expect(screen.getByTestId('page-dve-list')).toBeInTheDocument();
  });

  it('renders DVECreate at "/dves/new"', () => {
    window.history.pushState({}, '', '/dves/new');
    render(<App />);
    expect(screen.getByTestId('page-dve-create')).toBeInTheDocument();
  });

  it('renders AgentChat at "/dves/:dveId/agent"', () => {
    window.history.pushState({}, '', '/dves/dve-alpha/agent');
    render(<App />);
    expect(screen.getByTestId('page-agent-chat')).toBeInTheDocument();
  });

  it('renders VaultPage at "/vault"', () => {
    window.history.pushState({}, '', '/vault');
    render(<App />);
    expect(screen.getByTestId('page-vault')).toBeInTheDocument();
  });

  it('renders CognitiveEngineChat at "/cognitive"', () => {
    window.history.pushState({}, '', '/cognitive');
    render(<App />);
    expect(screen.getByTestId('page-cognitive')).toBeInTheDocument();
  });

  it('renders HomePage at "/workflows"', () => {
    window.history.pushState({}, '', '/workflows');
    render(<App />);
    expect(screen.getByTestId('page-home')).toBeInTheDocument();
  });

  it('renders Scanner at "/scanner"', () => {
    window.history.pushState({}, '', '/scanner');
    render(<App />);
    expect(screen.getByTestId('page-scanner')).toBeInTheDocument();
  });

  it('renders Badges at "/badges"', () => {
    window.history.pushState({}, '', '/badges');
    render(<App />);
    expect(screen.getByTestId('page-badges')).toBeInTheDocument();
  });

  it('renders UDC at "/udc"', () => {
    window.history.pushState({}, '', '/udc');
    render(<App />);
    expect(screen.getByTestId('page-udc')).toBeInTheDocument();
  });

  it('renders Onboarding at "/onboarding"', () => {
    window.history.pushState({}, '', '/onboarding');
    render(<App />);
    expect(screen.getByTestId('page-onboarding')).toBeInTheDocument();
  });

  it('renders Settings at "/settings"', () => {
    window.history.pushState({}, '', '/settings');
    render(<App />);
    expect(screen.getByTestId('page-settings')).toBeInTheDocument();
  });

  it('navigates correctly between routes', () => {
    window.history.pushState({}, '', '/vault');
    const { unmount } = render(<App />);
    expect(screen.getByTestId('page-vault')).toBeInTheDocument();
    unmount();

    window.history.pushState({}, '', '/cognitive');
    render(<App />);
    expect(screen.getByTestId('page-cognitive')).toBeInTheDocument();
  });
});
