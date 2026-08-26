import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import Bubblewrap from '../components/tools/sandbox/Bubblewrap';

const mockUseSandbox = jest.fn();
jest.mock('../components/SandboxContext', () => ({ useSandbox: () => mockUseSandbox() }));

describe('Bubblewrap', () => {
  beforeEach(() => {
    mockUseSandbox.mockReturnValue({
      launch: jest.fn(), stop: jest.fn(), status: 'running',
      log: ['[sandbox] started', 'target output'], targetLabel: 'demo',
      error: null, deps: [], depsInstalling: false, installDeps: jest.fn(),
    });
  });

  it('uses Electron native file selection for the target binary', async () => {
    const selectSandboxBinary = jest.fn().mockResolvedValue('/opt/demo/target');
    const launch = jest.fn().mockResolvedValue(undefined);
    Object.assign(window, { electronAPI: { selectSandboxBinary } });
    mockUseSandbox.mockReturnValue({
      launch, stop: jest.fn(), status: 'idle', log: [], targetLabel: 'demo', projectPath: '/opt/demo',
      error: null, deps: [], depsInstalling: false, installDeps: jest.fn(),
    });
    render(<Bubblewrap />);

    fireEvent.click(screen.getByRole('button', { name: /select target file/i }));

    expect(selectSandboxBinary).toHaveBeenCalledWith('/opt/demo');
    expect(await screen.findByDisplayValue('/opt/demo/target')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Launch' }));
    expect(launch).toHaveBeenCalledWith(expect.objectContaining({
      binds: expect.arrayContaining([expect.objectContaining({ mode: 'ro-bind', src: '/opt/demo', dst: '/opt/demo' })]),
    }));
  });

  it('launches Node scripts with explicit arguments', async () => {
    const launch = jest.fn().mockResolvedValue(undefined);
    mockUseSandbox.mockReturnValue({
      launch, stop: jest.fn(), status: 'idle', log: [], targetLabel: '', projectPath: '/project with spaces',
      error: null, deps: [], depsInstalling: false, installDeps: jest.fn(),
    });
    render(<Bubblewrap />);

    fireEvent.change(screen.getByLabelText('Execution mode'), { target: { value: 'command' } });
    fireEvent.change(screen.getByLabelText('Script path'), { target: { value: '/project with spaces/app.js' } });
    fireEvent.change(screen.getByLabelText('Command arguments'), { target: { value: '--port\n3000' } });
    fireEvent.click(screen.getByRole('button', { name: 'Launch' }));

    expect(launch).toHaveBeenCalledWith(expect.objectContaining({
      targetCommand: 'node', targetArgs: ['/project with spaces/app.js', '--port', '3000'],
      binds: expect.arrayContaining([expect.objectContaining({ mode: 'ro-bind', src: '/project with spaces', dst: '/project with spaces' })]),
    }));
  });

  it('automatically mounts the dashboard project and uses its selected file', async () => {
    const launch = jest.fn().mockResolvedValue(undefined);
    mockUseSandbox.mockReturnValue({
      launch, stop: jest.fn(), status: 'idle', log: [], targetLabel: 'demo', projectPath: '/opt/demo', projectTargetPath: '/opt/demo/bin/target',
      error: null, deps: [], depsInstalling: false, installDeps: jest.fn(),
    });
    render(<Bubblewrap />);

    expect(await screen.findByDisplayValue('/opt/demo/bin/target')).toBeInTheDocument();
    expect(screen.getByText('/opt/demo → /opt/demo')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Launch' }));

    expect(launch).toHaveBeenCalledWith(expect.objectContaining({
      targetCommand: '/opt/demo/bin/target',
      binds: expect.arrayContaining([expect.objectContaining({ mode: 'ro-bind', src: '/opt/demo', dst: '/opt/demo' })]),
    }));
  });
});
