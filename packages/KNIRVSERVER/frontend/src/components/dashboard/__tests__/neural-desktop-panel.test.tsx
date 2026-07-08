import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import NeuralDesktopPanel from '../neural-desktop-panel';

jest.mock('lucide-react', () => {
  const React = require('react');
  const icon = (name: string) => (props: Record<string, unknown>) =>
    React.createElement('span', { 'data-testid': `${name}-icon`, ...props });

  return {
    __esModule: true,
    Activity: icon('Activity'),
    AlertCircle: icon('AlertCircle'),
    Badge: icon('Badge'),
    CheckCircle: icon('CheckCircle'),
    ChevronRight: icon('ChevronRight'),
    Compass: icon('Compass'),
    Database: icon('Database'),
    GitBranch: icon('GitBranch'),
    Loader2: icon('Loader2'),
    Mic: icon('Mic'),
    MicOff: icon('MicOff'),
    Network: icon('Network'),
    RefreshCw: icon('RefreshCw'),
    Search: icon('Search'),
    Server: icon('Server'),
    Shield: icon('Shield'),
    ShieldAlert: icon('ShieldAlert'),
    Send: icon('Send'),
    Terminal: icon('Terminal'),
    Users: icon('Users'),
  };
});

jest.mock('@/lib/websocket-service', () => ({
  webSocketService: {
    on: jest.fn(),
    off: jest.fn(),
    subscribe: jest.fn(),
    getConnectionStatus: jest.fn(() => false),
  },
}));

describe('NeuralDesktopPanel', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    window.localStorage.clear();
  });

  it('renders routed cognitive engine activity in Current Processing', async () => {
    render(
      <NeuralDesktopPanel
        processingActivities={[
          {
            id: 'processing-1',
            timestamp: '2026-07-05T00:00:01Z',
            type: 'cognitive_engine_log',
            title: 'Cognitive Engine Activity',
            description: 'Contacting KNIRV Cognitive Engine Chat API...',
            status: 'active',
          },
        ]}
      />
    );

    expect(await screen.findByText('Current Processing')).toBeInTheDocument();
    expect(screen.getByText('Cognitive Engine Activity')).toBeInTheDocument();
    expect(screen.getByText('Contacting KNIRV Cognitive Engine Chat API...')).toBeInTheDocument();
  });
});
