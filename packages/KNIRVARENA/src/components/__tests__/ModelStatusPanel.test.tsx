import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { readFileSync } from 'fs';
import { join } from 'path';
import { ModelStatusPanel } from '../ModelStatusPanel';
import * as modelStatus from '../../services/modelStatus';
import * as serverLLM from '../../services/serverLLM';
import type { PlatformStatus } from '../../services/modelStatus';

const ready = (JSON.parse(
  readFileSync(join(__dirname, '../../../../KNIRVSDK/testvectors/models/platform_status.json'), 'utf8'),
) as Record<string, { status: PlatformStatus }>).ready_all.status;

describe('ModelStatusPanel', () => {
  beforeEach(() => { jest.spyOn(serverLLM, 'authToken').mockReturnValue('tok'); });
  afterEach(() => jest.restoreAllMocks());

  it('shows every role with its model, version and state, and the routing history', async () => {
    jest.spyOn(modelStatus, 'fetchModelStatus').mockResolvedValue({
      ...ready,
      routing: [
        { id: 'a', time: '2026-10-13T12:00:00Z', kind: 'execute_skill', chosen: 'llama', degraded: true, fallbacks: [], latency_ms: 5 },
        { id: 'b', time: '2026-10-13T11:59:00Z', kind: 'generate', chosen: 'external', fallbacks: ['heart'], latency_ms: 800 },
      ],
    });
    render(<ModelStatusPanel />);
    await waitFor(() => expect(screen.getByTestId('model-status')).toBeInTheDocument());
    for (const role of ['decide', 'generate', 'validate', 'server_tasks', 'external']) {
      expect(screen.getByTestId(`role-${role}`)).toBeInTheDocument();
    }
    expect(screen.getByTestId('role-decide')).toHaveTextContent('v3-1a2b3c4d');
    expect(screen.getByTestId('role-server_tasks')).toHaveTextContent('adapters: skill-aaaa1111');
    const events = screen.getAllByTestId('routing-event');
    expect(events[0]).toHaveTextContent('DEGRADED: skill did not run');
    expect(events[1]).toHaveTextContent('after failing: heart');
  });

  it('says why when the server cannot be read', async () => {
    jest.spyOn(modelStatus, 'fetchModelStatus').mockRejectedValue(new Error('This KNIRV server does not report model status yet'));
    render(<ModelStatusPanel />);
    await waitFor(() => expect(screen.getByTestId('model-status-error')).toHaveTextContent('does not report model status'));
  });
});
