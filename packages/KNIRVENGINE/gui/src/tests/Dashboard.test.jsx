import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { Dashboard } from '../components/Dashboard';

describe('Dashboard', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'showDirectoryPicker', { configurable: true, value: undefined });
  });

  test('starts with an add target action', () => {
    render(<Dashboard />);

    expect(screen.getByRole('heading', { name: /open a target project/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /add target/i })).toBeInTheDocument();
  });

  test('shows a file explorer and page explorer after choosing a folder', () => {
    const { container } = render(<Dashboard />);
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
    const { container } = render(<Dashboard />);
    const file = new File(['const title = "Home";'], 'page.tsx', { type: 'text/plain' });
    Object.defineProperty(file, 'webkitRelativePath', { value: 'sample-project/page.tsx' });

    fireEvent.change(container.querySelector('input[type="file"]'), { target: { files: [file] } });
    fireEvent.click(screen.getByRole('button', { name: 'page.tsx' }));

    expect(await screen.findByDisplayValue('const title = "Home";')).toBeInTheDocument();
  });

  test('allows a selected page to be edited', async () => {
    const { container } = render(<Dashboard />);
    const file = new File(['const title = "Home";'], 'page.tsx', { type: 'text/plain' });
    Object.defineProperty(file, 'webkitRelativePath', { value: 'sample-project/page.tsx' });

    fireEvent.change(container.querySelector('input[type="file"]'), { target: { files: [file] } });
    fireEvent.click(screen.getByRole('button', { name: 'page.tsx' }));
    const editor = await screen.findByLabelText(/page editor/i);
    fireEvent.change(editor, { target: { value: 'const title = "Updated";' } });

    expect(editor).toHaveValue('const title = "Updated";');
    expect(screen.getByRole('button', { name: /download/i })).toBeEnabled();
  });
});
