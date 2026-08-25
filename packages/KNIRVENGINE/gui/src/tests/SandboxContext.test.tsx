import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { SandboxProvider, useSandbox } from '../components/SandboxContext';

const mockCreate = jest.fn();
const mockList = jest.fn();
const mockClose = jest.fn();
const mockGetDeps = jest.fn();
const mockInstallDeps = jest.fn();
const mockWs = { close: jest.fn() } as unknown as WebSocket;
let statusHandlers: { onMessage?: (event: MessageEvent) => void } = {};

jest.mock('../services/sandboxService', () => ({
  createSandboxSession: (...args: unknown[]) => mockCreate(...args),
  listSandboxSessions: () => mockList(),
  closeSandboxSession: (...args: unknown[]) => mockClose(...args),
  createSandboxStatusWebSocket: (_id: string, handlers: typeof statusHandlers) => {
    statusHandlers = handlers;
    return mockWs;
  },
  getSandboxDependencies: () => mockGetDeps(),
  installSandboxDependencies: () => mockInstallDeps(),
}));

const Probe = () => {
  const { session, isReady, targetLabel, status, log } = useSandbox();
  return <div>
    <span data-testid="ready">{String(isReady)}</span>
    <span data-testid="status">{status ?? 'none'}</span>
    <span data-testid="label">{targetLabel}</span>
    <span data-testid="session">{session?.id ?? 'null'}</span>
    <span data-testid="log">{log.join('|')}</span>
  </div>;
};

describe('SandboxContext', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    statusHandlers = {};
    mockList.mockResolvedValue([]);
    mockGetDeps.mockResolvedValue({ dependencies: [] });
    mockInstallDeps.mockResolvedValue({ dependencies: [] });
    mockCreate.mockResolvedValue({
      id: 'sess-1', status: 'provisioning', targetLabel: 'my-app',
      targetCommand: '/bin/true', display: ':99', netnsId: 'sess-1',
      vncPort: 5999, vncWsPath: '/api/v1/sandboxes/sess-1/vnc', statusWsPath: '/api/v1/sandboxes/sess-1/ws',
    });
  });

  it('launches, receives status/log frames, and stops the active session', async () => {
    let sandbox: ReturnType<typeof useSandbox> | null = null;
    const Capture = () => {
      sandbox = useSandbox();
      return <Probe />;
    };
    render(<SandboxProvider><Capture /></SandboxProvider>);
    await waitFor(() => expect(mockList).toHaveBeenCalled());

    act(() => sandbox!.setTargetLabel('my-app'));
    expect(screen.getByTestId('label')).toHaveTextContent('my-app');
    await act(async () => { await sandbox!.launch({ targetLabel: 'my-app', targetCommand: '/bin/true' }); });
    expect(mockCreate).toHaveBeenCalledWith(expect.objectContaining({ targetLabel: 'my-app' }));
    expect(screen.getByTestId('status')).toHaveTextContent('provisioning');

    act(() => statusHandlers.onMessage?.({ data: JSON.stringify({ type: 'status', status: 'running' }) } as MessageEvent));
    act(() => statusHandlers.onMessage?.({ data: JSON.stringify({ type: 'log', data: 'sandbox ready' }) } as MessageEvent));
    expect(screen.getByTestId('ready')).toHaveTextContent('true');
    expect(screen.getByTestId('log')).toHaveTextContent('sandbox ready');

    await act(async () => { await sandbox!.stop(); });
    expect(mockClose).toHaveBeenCalledWith('sess-1');
    expect(screen.getByTestId('session')).toHaveTextContent('null');
  });
});
