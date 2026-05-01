import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
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

vi.mock('@/react-app/pages/Skills', () => ({
  default: () => <div data-testid="page-skills">Skills Page</div>,
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
  it('renders DVEList at root path "/"', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-dve-list')).toBeInTheDocument();
  });

  it('renders DVEList at "/dves"', () => {
    render(
      <MemoryRouter initialEntries={['/dves']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-dve-list')).toBeInTheDocument();
  });

  it('renders DVECreate at "/dves/new"', () => {
    render(
      <MemoryRouter initialEntries={['/dves/new']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-dve-create')).toBeInTheDocument();
  });

  it('renders AgentChat at "/dves/:dveId/agent"', () => {
    render(
      <MemoryRouter initialEntries={['/dves/dve-alpha/agent']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-agent-chat')).toBeInTheDocument();
  });

  it('renders VaultPage at "/vault"', () => {
    render(
      <MemoryRouter initialEntries={['/vault']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-vault')).toBeInTheDocument();
  });

  it('renders CognitiveEngineChat at "/cognitive"', () => {
    render(
      <MemoryRouter initialEntries={['/cognitive']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-cognitive')).toBeInTheDocument();
  });

  it('renders HomePage at "/workflows"', () => {
    render(
      <MemoryRouter initialEntries={['/workflows']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-home')).toBeInTheDocument();
  });

  it('renders Scanner at "/scanner"', () => {
    render(
      <MemoryRouter initialEntries={['/scanner']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-scanner')).toBeInTheDocument();
  });

  it('renders Skills at "/skills"', () => {
    render(
      <MemoryRouter initialEntries={['/skills']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-skills')).toBeInTheDocument();
  });

  it('renders UDC at "/udc"', () => {
    render(
      <MemoryRouter initialEntries={['/udc']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-udc')).toBeInTheDocument();
  });

  it('renders Onboarding at "/onboarding"', () => {
    render(
      <MemoryRouter initialEntries={['/onboarding']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-onboarding')).toBeInTheDocument();
  });

  it('navigates correctly between routes', () => {
    // Test "/vault" route without affecting other tests
    const { unmount } = render(
      <MemoryRouter initialEntries={['/vault']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-vault')).toBeInTheDocument();
    unmount();

    // Test "/cognitive" route
    render(
      <MemoryRouter initialEntries={['/cognitive']}>
        <App />
      </MemoryRouter>
    );
    expect(screen.getByTestId('page-cognitive')).toBeInTheDocument();
  });
});
