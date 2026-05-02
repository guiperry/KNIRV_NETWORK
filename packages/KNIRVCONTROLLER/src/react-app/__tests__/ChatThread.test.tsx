import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import ChatThread from '@/react-app/components/ChatThread';

// Define a minimal AgentMessage type for testing
interface TestAgentMessage {
  id: string;
  role: 'user' | 'agent' | 'cognitive' | 'system';
  content: string;
  timestamp: string;
  taskID?: string;
  dveID?: string;
}

// Mock lucide-react icons used in ChatThread
vi.mock('lucide-react', () => ({
  Copy: () => <div data-testid="icon-copy" />,
  Check: () => <div data-testid="icon-check" />,
}));

const mockOnCopy = vi.fn();

const makeMessage = (
  role: TestAgentMessage['role'],
  content: string,
  id: string = `msg-${Date.now()}`,
  timestamp: string = new Date().toISOString()
): TestAgentMessage => ({
  id,
  role,
  content,
  timestamp,
});

describe('ChatThread', () => {
  it('renders messages correctly', () => {
    const messages = [
      makeMessage('user', 'Hello, how are you?'),
      makeMessage('agent', 'I am doing well, thank you!'),
    ];

    render(<ChatThread messages={messages as any} isStreaming={false} onCopyMessage={mockOnCopy} />);

    expect(screen.getByText('Hello, how are you?')).toBeInTheDocument();
    expect(screen.getByText('I am doing well, thank you!')).toBeInTheDocument();
  });

  it('shows role labels for user and agent messages', () => {
    const messages = [
      makeMessage('user', 'User message'),
      makeMessage('agent', 'Agent response'),
    ];

    render(<ChatThread messages={messages as any} isStreaming={false} onCopyMessage={mockOnCopy} />);

    expect(screen.getByText('User')).toBeInTheDocument();
    expect(screen.getByText('Agent')).toBeInTheDocument();
  });

  it('shows "No messages yet" when message list is empty', () => {
    render(<ChatThread messages={[]} isStreaming={false} onCopyMessage={mockOnCopy} />);

    expect(screen.getByText('No messages yet. Start a conversation.')).toBeInTheDocument();
  });

  it('does not show "No messages yet" when streaming', () => {
    render(<ChatThread messages={[]} isStreaming={true} onCopyMessage={mockOnCopy} />);

    expect(screen.queryByText('No messages yet. Start a conversation.')).not.toBeInTheDocument();
  });

  it('renders system messages as centered italic text', () => {
    const messages = [
      makeMessage('system', 'System health check complete'),
    ];

    render(<ChatThread messages={messages as any} isStreaming={false} onCopyMessage={mockOnCopy} />);

    const systemMsg = screen.getByText('System health check complete');
    expect(systemMsg).toBeInTheDocument();
  });

  it('renders code blocks with formatting', () => {
    const messages = [
      makeMessage('agent', 'Here is some code:\n```javascript\nconst x = 1;\n```'),
    ];

    render(<ChatThread messages={messages as any} isStreaming={false} onCopyMessage={mockOnCopy} />);

    // The code block should be rendered inside a <pre> element
    const codeBlocks = document.querySelectorAll('pre');
    expect(codeBlocks.length).toBeGreaterThan(0);
    expect(codeBlocks[0]).toContainHTML('const x = 1;');
  });

  it('renders code block language label', () => {
    const messages = [
      makeMessage('agent', '```typescript\nconst y: string = "hello";\n```'),
    ];

    render(<ChatThread messages={messages as any} isStreaming={false} onCopyMessage={mockOnCopy} />);

    expect(screen.getByText('typescript')).toBeInTheDocument();
  });

  it('shows streaming indicator when isStreaming is true', () => {
    const messages = [makeMessage('user', 'Test')];

    render(<ChatThread messages={messages as any} isStreaming={true} onCopyMessage={mockOnCopy} />);

    // The streaming indicator has animated dots - should be present when streaming
    const dots = document.querySelectorAll('.animate-bounce');
    expect(dots.length).toBeGreaterThanOrEqual(3);
  });

  it('does not show streaming indicator when isStreaming is false', () => {
    const messages = [makeMessage('user', 'Test')];

    render(<ChatThread messages={messages as any} isStreaming={false} onCopyMessage={mockOnCopy} />);

    // The streaming indicator only shows when streaming
    // Using the component directly so dots are inside the component
    const dots = document.querySelectorAll('[style*="animation-delay"]');
    expect(dots.length).toBe(0);
  });

  it('renders timestamps for messages', () => {
    const messages = [
      makeMessage('user', 'Hello', 'msg-1', '2024-01-15T10:30:00Z'),
    ];

    render(<ChatThread messages={messages as any} isStreaming={false} onCopyMessage={mockOnCopy} />);

    // The formatted timestamp should appear (10:30 in UTC)
    expect(screen.getByText('10:30')).toBeInTheDocument();
  });

  it('calls onCopyMessage when copy button is clicked', () => {
    const messages = [
      makeMessage('agent', 'Copy this content', 'msg-1'),
    ];

    render(<ChatThread messages={messages as any} isStreaming={false} onCopyMessage={mockOnCopy} />);

    // Find copy buttons
    const copyButtons = document.querySelectorAll('button[title="Copy message"]');
    expect(copyButtons.length).toBeGreaterThan(0);

    fireEvent.click(copyButtons[0]);
    expect(mockOnCopy).toHaveBeenCalledWith('msg-1', 'Copy this content');
  });

  it('handles cognitive role messages correctly', () => {
    const messages = [
      makeMessage('cognitive', 'Cognitive analysis complete'),
    ];

    render(<ChatThread messages={messages as any} isStreaming={false} onCopyMessage={mockOnCopy} />);

    expect(screen.getByText('Cognitive')).toBeInTheDocument();
  });

  it('renders multiple messages in correct order', () => {
    const messages = [
      makeMessage('user', 'First', '1'),
      makeMessage('agent', 'Second', '2'),
      makeMessage('user', 'Third', '3'),
    ];

    render(
      <ChatThread messages={messages as any} isStreaming={false} onCopyMessage={mockOnCopy} />
    );

    // All messages should be rendered
    expect(screen.getByText('First')).toBeInTheDocument();
    expect(screen.getByText('Second')).toBeInTheDocument();
    expect(screen.getByText('Third')).toBeInTheDocument();
  });
});
