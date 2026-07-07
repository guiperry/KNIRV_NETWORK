import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { HasherTrainingControls } from '../cognitive-engine-panel';

jest.mock('@/lib/api', () => ({
  API_BASE_URL: '',
  apiRequest: jest.fn(),
}));

jest.mock('@/hooks/use-cognitive-engine', () => ({
  useCognitiveEngine: jest.fn(() => ({
    cognitiveEngine: null,
    isLoading: false,
    error: null,
  })),
  useQualityMode: jest.fn(() => ({
    qualityMode: 'standard',
    setQualityMode: jest.fn(),
  })),
}));

jest.mock('../neural-desktop-panel', () => ({
  __esModule: true,
  default: () => <div data-testid="neural-desktop-panel" />,
}));

jest.mock('lucide-react', () => {
  const React = require('react');
  const icon = (name: string) => (props: Record<string, unknown>) =>
    React.createElement('span', { 'data-testid': `${name}-icon`, ...props });

  return {
    __esModule: true,
    Activity: icon('Activity'),
    AlertCircle: icon('AlertCircle'),
    BarChart3: icon('BarChart3'),
    BookOpen: icon('BookOpen'),
    Bot: icon('Bot'),
    Brain: icon('Brain'),
    Bug: icon('Bug'),
    CheckCircle: icon('CheckCircle'),
    Clock: icon('Clock'),
    Cpu: icon('Cpu'),
    Database: icon('Database'),
    Eye: icon('Eye'),
    EyeOff: icon('EyeOff'),
    GitBranch: icon('GitBranch'),
    Heart: icon('Heart'),
    Loader2: icon('Loader2'),
    Play: icon('Play'),
    RefreshCw: icon('RefreshCw'),
    Server: icon('Server'),
    Shield: icon('Shield'),
    Square: icon('Square'),
    Terminal: icon('Terminal'),
    TrendingUp: icon('TrendingUp'),
    Zap: icon('Zap'),
  };
});

class MockEventSource {
  static instances: MockEventSource[] = [];
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  close = jest.fn();

  constructor(public url: string) {
    MockEventSource.instances.push(this);
  }
}

describe('HasherTrainingControls', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    MockEventSource.instances = [];
    global.EventSource = MockEventSource as unknown as typeof EventSource;
    (global.fetch as jest.Mock).mockImplementation((url: string) => {
      if (url.includes('/api/v1/hasher/status')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            available: true,
            running: true,
            training: {
              status: 'running',
              message: 'completed batch 2/8',
              current_batch: 3,
              batches_done: 2,
              total_batches: 8,
              records_loaded: 128,
              errors: ['Seed materialization incomplete for token 42'],
            },
          }),
        });
      }

      if (url.includes('/api/logs/history')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            logs: [
              {
                timestamp: '2026-07-05T00:00:00Z',
                level: 'info',
                message: 'pipeline still running',
              },
            ],
          }),
        });
      }

      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
    });
  });

  it('rehydrates active training state and reconnects logs after remount', async () => {
    const { unmount } = render(<HasherTrainingControls />);

    await waitFor(() => {
      expect(screen.getByText('Training pipeline active')).toBeInTheDocument();
      expect(MockEventSource.instances).toHaveLength(1);
    });

    unmount();
    expect(MockEventSource.instances[0].close).toHaveBeenCalled();

    render(<HasherTrainingControls />);

    await waitFor(() => {
      expect(screen.getByText('Training pipeline active')).toBeInTheDocument();
      expect(screen.getByText('pipeline still running')).toBeInTheDocument();
      expect(MockEventSource.instances).toHaveLength(2);
    });
  });

  it('renders training batch metadata and pipeline errors', async () => {
    render(<HasherTrainingControls />);

    await waitFor(() => {
      expect(screen.getByText('Current Batch')).toBeInTheDocument();
      expect(screen.getByText('3/8')).toBeInTheDocument();
      expect(screen.getByText('Batches Done')).toBeInTheDocument();
      expect(screen.getByText('2/8')).toBeInTheDocument();
      expect(screen.getByText('Seed materialization incomplete for token 42')).toBeInTheDocument();
    });
  });

  it('lets the training log height be resized in page', async () => {
    render(<HasherTrainingControls />);

    const resizeHandle = await screen.findByLabelText('Resize training log');
    const logWindow = resizeHandle.previousElementSibling as HTMLElement;

    expect(logWindow).toHaveStyle({ height: '144px' });

    fireEvent.mouseDown(resizeHandle, { clientY: 100 });
    fireEvent.mouseMove(document, { clientY: 220 });
    fireEvent.mouseUp(document);

    expect(logWindow).toHaveStyle({ height: '264px' });
  });
});
