import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import NoVnc from '../components/tools/sandbox/NoVnc';

const mockUseSandbox = jest.fn();
const mockCanvas = jest.fn();

jest.mock('../components/SandboxContext', () => ({ useSandbox: () => mockUseSandbox() }));
jest.mock('../components/tools/sandbox/SandboxVncCanvas', () => ({
  __esModule: true,
  default: (props: unknown) => {
    mockCanvas(props);
    return <div data-testid="sandbox-vnc-canvas" />;
  },
}));

describe('NoVnc', () => {
  it('uses the active sandbox VNC WebSocket path for the live canvas', () => {
    mockUseSandbox.mockReturnValue({
      isReady: true,
      session: { id: 'sess-1', vncWsPath: '/api/v1/sandboxes/sess-1/vnc' },
    });
    render(<NoVnc />);
    expect(screen.getByTestId('sandbox-vnc-canvas')).toBeInTheDocument();
    expect(mockCanvas).toHaveBeenCalledWith(expect.objectContaining({
      wsUrl: 'ws://localhost/api/v1/sandboxes/sess-1/vnc', quality: 6, viewOnly: false,
    }));
  });

  it('explains that a sandbox must be launched when no session exists', () => {
    mockUseSandbox.mockReturnValue({ isReady: false, session: null });
    render(<NoVnc />);
    expect(screen.getByText(/launch a sandbox to open its framebuffer/i)).toBeInTheDocument();
  });
});
