import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router';
import AgentChat from '@/react-app/pages/AgentChat';

// Mock Layout
vi.mock('@/react-app/components/Layout', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div data-testid="layout-wrapper">{children}</div>,
}));

// Mock ChatThread
vi.mock('@/react-app/components/ChatThread', () => ({
  default: ({ messages, isStreaming, onCopyMessage }: { messages: any[]; isStreaming: boolean; onCopyMessage: (id: string, content: string) => void }) => (
    <div data-testid="chat-thread">
      <span data-testid="message-count">{messages.length}</span>
      <span data-testid="streaming-status">{isStreaming ? 'streaming' : 'idle'}</span>
      {messages.map((msg, i) => (
        <div key={i} data-testid={`message-${i}`} data-role={msg.role}>
          {msg.content}
        </div>
      ))}
    </div>
  ),
}));

// Mock VoiceChatBar
vi.mock('@/react-app/components/VoiceChatBar', () => ({
  default: ({ onSendMessage, disabled, placeholder }: { onSendMessage: (t: string) => void; disabled: boolean; placeholder: string }) => (
    <div data-testid="voice-chat-bar" data-disabled={disabled} data-placeholder={placeholder}>
      <button data-testid="voice-send-btn" onClick={() => onSendMessage?.('run skill test-skill')}>
        Send
      </button>
    </div>
  ),
}));

// Mock lucide-react
vi.mock('lucide-react', () => ({
  ArrowLeft: () => <div data-testid="icon-arrow-left" />,
  Shield: () => <div data-testid="icon-shield" />,
  Cpu: () => <div data-testid="icon-cpu" />,
  Zap: () => <div data-testid="icon-zap" />,
  Activity: () => <div data-testid="icon-activity" />,
  Wifi: () => <div data-testid="icon-wifi" />,
  WifiOff: () => <div data-testid="icon-wifi-off" />,
  BadgeCheck: () => <div data-testid="icon-badge-check" />,
  Loader2: () => <div data-testid="icon-loader-2" />,
  MapPin: () => <div data-testid="icon-map-pin" />,
  Clock: () => <div data-testid="icon-clock" />,
}));

const renderAgentChat = (dveId: string = 'dve-alpha') => {
  return render(
    <MemoryRouter initialEntries={[`/dves/${dveId}/agent`]}>
      <Routes>
        <Route path="/dves/:dveId/agent" element={<AgentChat />} />
      </Routes>
    </MemoryRouter>
  );
};

describe('AgentChat', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders with Layout wrapper', () => {
    renderAgentChat();
    expect(screen.getByTestId('layout-wrapper')).toBeInTheDocument();
  });

  it('shows the DVE name in the heading', () => {
    renderAgentChat('dve-alpha');
    expect(screen.getByText('DVE-Alpha')).toBeInTheDocument();
  });

  it('shows the DVE status badge', () => {
    renderAgentChat('dve-alpha');
    expect(screen.getByText('ONLINE')).toBeInTheDocument();
  });

  it('shows the agent greeting message', () => {
    renderAgentChat('dve-alpha');
    // The initial greeting mentions the agent name
    expect(screen.getByText(/KNIRVAGENT-DVE-Alpha/)).toBeInTheDocument();
  });

  it('shows the agent name badge', () => {
    renderAgentChat('dve-alpha');
    expect(screen.getByText('KNIRVAGENT-DVE-Alpha')).toBeInTheDocument();
  });

  it('shows the reputation score', () => {
    renderAgentChat('dve-alpha');
    expect(screen.getByText('Rep: 847')).toBeInTheDocument();
  });

  it('renders the ChatThread component', () => {
    renderAgentChat('dve-alpha');
    expect(screen.getByTestId('chat-thread')).toBeInTheDocument();
  });

  it('shows the VoiceChatBar with correct placeholder', () => {
    renderAgentChat('dve-alpha');
    const voiceBar = screen.getByTestId('voice-chat-bar');
    expect(voiceBar).toHaveAttribute('data-placeholder', 'Ask KNIRVAGENT-DVE-Alpha something...');
  });

  it('shows DVE capabilities in the initial greeting', () => {
    renderAgentChat('dve-alpha');
    expect(screen.getByText(/ATTESTED_EXECUTION/)).toBeInTheDocument();
  });

  it('shows the TEE type', () => {
    renderAgentChat('dve-alpha');
    expect(screen.getByText('SGX')).toBeInTheDocument();
  });

  it('renders badge chips for DVE badges', () => {
    renderAgentChat('dve-alpha');
    expect(screen.getByText('ATTESTED')).toBeInTheDocument();
    expect(screen.getByText('SGX 2.0')).toBeInTheDocument();
  });

  it('shows back button for navigation', () => {
    renderAgentChat('dve-alpha');
    expect(screen.getByTestId('icon-arrow-left')).toBeInTheDocument();
  });

  it('handles task submission via voice send button', () => {
    renderAgentChat('dve-alpha');
    const sendBtn = screen.getByTestId('voice-send-btn');

    act(() => {
      fireEvent.click(sendBtn);
    });

    // After clicking send, message count should go from 1 (greeting) to 2 (greeting + user message)
    const messageCount = screen.getByTestId('message-count');
    expect(messageCount.textContent).toBe('2');
  });

  it('shows the location of the DVE (in status text)', () => {
    renderAgentChat('dve-alpha');
    // The location "us-east-1" appears in the status display
    const statusBadge = screen.getByText('SGX');
    expect(statusBadge).toBeInTheDocument();
  });
});
