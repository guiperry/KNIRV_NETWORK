import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import SandboxDock from '../components/layout/SandboxDock';

const mockUseSandbox = jest.fn();
const mockCanvas = jest.fn();

jest.mock('../components/SandboxContext', () => ({ useSandbox: () => mockUseSandbox() }));
jest.mock('../components/tools/sandbox/SandboxVncCanvas', () => ({
  __esModule: true,
  default: (props: unknown) => {
    mockCanvas(props);
    return <div data-testid="sandbox-vnc-canvas" className="min-h-0 min-w-0 flex-1" />;
  },
}));

describe('SandboxDock', () => {
  beforeEach(() => {
    mockCanvas.mockClear();
    mockUseSandbox.mockReturnValue({
      session: { id: 'sess-1', targetLabel: 'target', vncWsPath: '/api/v1/sandboxes/sess-1/vnc' },
      status: 'running',
      isReady: true,
      stop: jest.fn(),
      log: [],
      error: null,
      clearLog: jest.fn(),
    });
  });

  it('gives the docked target view a bounded flex viewport for the live canvas', async () => {
    render(<SandboxDock />);

    expect(await screen.findByTestId('sandbox-vnc-canvas')).toBeInTheDocument();
    expect(screen.getByLabelText('Target application view')).toHaveClass('flex', 'min-h-0', 'min-w-0', 'overflow-hidden');
    expect(screen.getByTestId('sandbox-vnc-canvas')).toHaveClass('flex-1', 'min-h-0', 'min-w-0');
    expect(mockCanvas).toHaveBeenCalledWith(expect.objectContaining({
      wsUrl: 'ws://localhost/api/v1/sandboxes/sess-1/vnc',
    }));
  });
});
