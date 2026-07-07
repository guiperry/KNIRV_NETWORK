import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import CognitiveEngineChat from '@/react-app/pages/CognitiveEngineChat';

// Mock Layout
vi.mock('@/react-app/components/Layout', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div data-testid="layout-wrapper">{children}</div>,
}));

// Mock ChatThread
vi.mock('@/react-app/components/ChatThread', () => ({
  default: ({ messages, isStreaming }: { messages: any[]; isStreaming: boolean }) => (
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
  default: ({ onSendMessage, disabled }: { onSendMessage: (t: string) => void; disabled: boolean }) => (
    <div data-testid="voice-chat-bar" data-disabled={disabled}>
      <button data-testid="voice-send-btn" onClick={() => onSendMessage?.('test voice command')}>
        Send
      </button>
    </div>
  ),
}));

// Mock lucide-react
vi.mock('lucide-react', () => ({
  Brain: () => <div data-testid="icon-brain" />,
  Activity: () => <div data-testid="icon-activity" />,
  AlertTriangle: () => <div data-testid="icon-alert-triangle" />,
  Wifi: () => <div data-testid="icon-wifi" />,
  WifiOff: () => <div data-testid="icon-wifi-off" />,
  Server: () => <div data-testid="icon-server" />,
}));

describe('CognitiveEngineChat', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows loading state initially', () => {
    render(<CognitiveEngineChat />);
    expect(screen.getByText('Connecting to Cognitive Engine...')).toBeInTheDocument();
  });

  it('renders the Brain icon during loading', () => {
    render(<CognitiveEngineChat />);
    expect(screen.getByTestId('icon-brain')).toBeInTheDocument();
  });

  it('shows health check system message after loading completes', () => {
    render(<CognitiveEngineChat />);

    // Advance past the 800ms health check timeout
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByText('Cognitive Engine active. Oracle loaded. DVE fleet: 12 nodes, 8 online.')).toBeInTheDocument();
  });

  it('renders header with COGNITIVE ENGINE text', () => {
    render(<CognitiveEngineChat />);

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByText('◈ COGNITIVE ENGINE')).toBeInTheDocument();
  });

  it('shows root node indicator after loading', () => {
    render(<CognitiveEngineChat />);

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByText('Root: knirv-1')).toBeInTheDocument();
  });

  it('displays CPU health status', () => {
    render(<CognitiveEngineChat />);

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByText('CPU 23%')).toBeInTheDocument();
  });

  it('displays online nodes count', () => {
    render(<CognitiveEngineChat />);

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByText('8/12 online')).toBeInTheDocument();
  });

  it('shows Oracle synced indicator', () => {
    render(<CognitiveEngineChat />);

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByText('Oracle synced')).toBeInTheDocument();
  });

  it('renders the ChatThread component after loading', () => {
    render(<CognitiveEngineChat />);

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByTestId('chat-thread')).toBeInTheDocument();
  });

  it('renders the VoiceChatBar component after loading', () => {
    render(<CognitiveEngineChat />);

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByTestId('voice-chat-bar')).toBeInTheDocument();
  });

  it('shows the correct placeholder text in loading state', () => {
    render(<CognitiveEngineChat />);
    expect(screen.getByText('Connecting to Cognitive Engine...')).toBeInTheDocument();
  });
});
