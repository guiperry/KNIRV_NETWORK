import React from 'react';
import { render, screen } from '@testing-library/react';
import { readFileSync } from 'fs';
import { join } from 'path';
import { ModelsStatusCard } from '../models-status-card';
import { useModelsStatus } from '@/hooks/use-models-status';
import type { PlatformStatus } from '@/lib/models-status';

jest.mock('@/hooks/use-models-status');
const mockHook = useModelsStatus as jest.MockedFunction<typeof useModelsStatus>;

const cases = JSON.parse(
  readFileSync(join(__dirname, '../../../../../../KNIRVSDK/testvectors/models/platform_status.json'), 'utf8'),
) as Record<string, { status: PlatformStatus }>;

const result = (over: Record<string, unknown>) => ({ data: undefined, error: null, isLoading: false, ...over }) as unknown as ReturnType<typeof useModelsStatus>;

describe('ModelsStatusCard', () => {
  it('shows each role with model, version and state', () => {
    mockHook.mockReturnValue(result({ data: { ...cases.ready_all.status, routing: [] } }));
    render(<ModelsStatusCard />);
    expect(screen.getByTestId('models-role-decide')).toHaveTextContent('v3-1a2b3c4d');
    expect(screen.getByTestId('models-role-decide')).toHaveTextContent('ready');
    expect(screen.getByTestId('models-role-server_tasks')).toHaveTextContent('adapters: skill-aaaa1111');
    expect(screen.getByText('None since the server started')).toBeInTheDocument();
  });

  it('shows outages and their reason instead of hiding them', () => {
    mockHook.mockReturnValue(result({ data: cases.daemon_down.status }));
    render(<ModelsStatusCard />);
    expect(screen.getByTestId('models-role-decide')).toHaveTextContent('unavailable');
    expect(screen.getByTestId('models-role-decide')).toHaveTextContent('connection refused');
  });

  it('lists recent routing including fallbacks and a skill that did not run', () => {
    mockHook.mockReturnValue(result({ data: { ...cases.ready_all.status, routing: [
      { id: '1', time: 't', kind: 'execute_skill', chosen: 'llama', degraded: true, fallbacks: [], latency_ms: 3 },
      { id: '2', time: 't', kind: 'generate', chosen: 'external', fallbacks: ['heart'], latency_ms: 3 },
    ] } }));
    render(<ModelsStatusCard />);
    const ev = screen.getAllByTestId('models-routing-event');
    expect(ev[0]).toHaveTextContent('DEGRADED: skill did not run');
    expect(ev[1]).toHaveTextContent('after failing: heart');
  });

  it('reports a failed read when there is no earlier data', () => {
    mockHook.mockReturnValue(result({ error: new Error('This server does not report model status yet') }));
    render(<ModelsStatusCard />);
    expect(screen.getByRole('alert')).toHaveTextContent('does not report model status');
  });
});
