import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { Dashboard } from '../components/Dashboard';
import { SandboxProvider, useSandbox } from '../components/SandboxContext';

const renderDashboard = () =>
  render(
    <SandboxProvider>
      <Dashboard />
    </SandboxProvider>
  );

describe('Dashboard', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'showDirectoryPicker', { configurable: true, value: undefined });
    Object.defineProperty(window, 'electronAPI', { configurable: true, value: undefined, writable: true });
  });

  test('starts with an add target action', () => {
    renderDashboard();

    expect(screen.getByRole('heading', { name: /open a target project/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /add target/i })).toBeInTheDocument();
  });

  test('shows a file explorer and page explorer after choosing a folder', () => {
    const { container } = renderDashboard();
    const file = new File(['export default function Home() {}'], 'page.tsx', { type: 'text/plain' });
    Object.defineProperty(file, 'webkitRelativePath', { value: 'sample-project/src/page.tsx' });
    const input = container.querySelector('input[type="file"]');

    fireEvent.change(input, { target: { files: [file] } });

    expect(screen.getByRole('heading', { name: 'sample-project' })).toBeInTheDocument();
    expect(screen.getByLabelText(/project files/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/page explorer/i)).toBeInTheDocument();
    expect(screen.getByText('page.tsx')).toBeInTheDocument();
  });

  test('opens a selected file in the page explorer', async () => {
    const { container } = renderDashboard();
    const file = new File(['const title = "Home";'], 'page.tsx', { type: 'text/plain' });
    Object.defineProperty(file, 'webkitRelativePath', { value: 'sample-project/page.tsx' });

    fireEvent.change(container.querySelector('input[type="file"]'), { target: { files: [file] } });
    fireEvent.click(screen.getByRole('button', { name: 'page.tsx' }));

    expect(await screen.findByDisplayValue('const title = "Home";')).toBeInTheDocument();
  });

  test('allows a selected page to be edited', async () => {
    const { container } = renderDashboard();
    const file = new File(['const title = "Home";'], 'page.tsx', { type: 'text/plain' });
    Object.defineProperty(file, 'webkitRelativePath', { value: 'sample-project/page.tsx' });

    fireEvent.change(container.querySelector('input[type="file"]'), { target: { files: [file] } });
    fireEvent.click(screen.getByRole('button', { name: 'page.tsx' }));
    const editor = await screen.findByLabelText(/page editor/i);
    fireEvent.change(editor, { target: { value: 'const title = "Updated";' } });

    expect(editor).toHaveValue('const title = "Updated";');
    expect(screen.getByRole('button', { name: /download/i })).toBeEnabled();
  });

  test('loads native Electron project files into the explorer', async () => {
    window.electronAPI = {
      selectSandboxProject: jest.fn().mockResolvedValue('/workspace/demo-project'),
      listSandboxProjectFiles: jest.fn().mockResolvedValue([
        { name: 'main.py', path: 'src/main.py', size: 15 },
      ]),
      readSandboxProjectFile: jest.fn().mockResolvedValue('print("hello")'),
    };

    renderDashboard();
    fireEvent.click(screen.getByRole('button', { name: /add target/i }));

    expect(await screen.findByRole('heading', { name: 'demo-project' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'main.py' }));
    expect(await screen.findByDisplayValue('print("hello")')).toBeInTheDocument();
    expect(window.electronAPI.readSandboxProjectFile).toHaveBeenCalledWith('src/main.py');
  });

  test('keeps a loaded project when the dashboard route remounts', async () => {
    window.electronAPI = {
      selectSandboxProject: jest.fn().mockResolvedValue('/workspace/persistent-project'),
      listSandboxProjectFiles: jest.fn().mockResolvedValue([
        { name: 'index.js', path: 'index.js', size: 20 },
      ]),
    };
    const RoutedDashboard = () => {
      const [visible, setVisible] = React.useState(true);
      return <SandboxProvider><button type="button" onClick={() => setVisible((current) => !current)}>navigate</button>{visible ? <Dashboard /> : null}</SandboxProvider>;
    };

    render(<RoutedDashboard />);
    fireEvent.click(screen.getByRole('button', { name: /add target/i }));
    expect(await screen.findByRole('heading', { name: 'persistent-project' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'navigate' }));
    fireEvent.click(screen.getByRole('button', { name: 'navigate' }));

    expect(screen.getByRole('heading', { name: 'persistent-project' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'index.js' })).toBeInTheDocument();
  });

  test('uses a selected native project file as the sandbox target', async () => {
    window.electronAPI = {
      selectSandboxProject: jest.fn().mockResolvedValue('/workspace/demo-project'),
      listSandboxProjectFiles: jest.fn().mockResolvedValue([
        { name: 'target', path: 'bin/target', size: 128 },
      ]),
      readSandboxProjectFile: jest.fn().mockResolvedValue('binary data'),
    };
    const TargetPath = () => <span data-testid="sandbox-target-path">{useSandbox().projectTargetPath}</span>;

    render(<SandboxProvider><Dashboard /><TargetPath /></SandboxProvider>);
    fireEvent.click(screen.getByRole('button', { name: /add target/i }));
    expect(await screen.findByRole('button', { name: 'target' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'target' }));

    expect(await screen.findByDisplayValue('binary data')).toBeInTheDocument();
    expect(screen.getByTestId('sandbox-target-path')).toHaveTextContent('/workspace/demo-project/bin/target');
  });
});
