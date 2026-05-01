import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router';
import DVECreate from '@/react-app/pages/DVECreate';

// Mock the Layout component
vi.mock('@/react-app/components/Layout', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div data-testid="layout-wrapper">{children}</div>,
}));

// Mock lucide-react icons used in DVECreate
vi.mock('lucide-react', () => ({
  Cpu: () => <div data-testid="icon-cpu" />,
  Shield: () => <div data-testid="icon-shield" />,
  Check: () => <div data-testid="icon-check" />,
  ArrowLeft: () => <div data-testid="icon-arrow-left" />,
  ArrowRight: () => <div data-testid="icon-arrow-right" />,
  BadgeCheck: () => <div data-testid="icon-badge-check" />,
  Zap: () => <div data-testid="icon-zap" />,
  Loader: () => <div data-testid="icon-loader" />,
}));

const renderWithRouter = (ui: React.ReactElement) => {
  return render(<BrowserRouter>{ui}</BrowserRouter>);
};

describe('DVECreate', () => {
  it('renders without crashing', () => {
    const { container } = renderWithRouter(<DVECreate />);
    expect(container).toBeTruthy();
  });

  it('shows the Create DVE header', () => {
    renderWithRouter(<DVECreate />);
    expect(screen.getByText('Create')).toBeInTheDocument();
    expect(screen.getByText('DVE')).toBeInTheDocument();
  });

  it('shows the "Name & Type" step label', () => {
    renderWithRouter(<DVECreate />);
    expect(screen.getByText('Name & Type')).toBeInTheDocument();
  });

  it('renders the TEE type selection cards', () => {
    renderWithRouter(<DVECreate />);
    expect(screen.getByText('Hardware TEE')).toBeInTheDocument();
    expect(screen.getByText('Browser Extension')).toBeInTheDocument();
  });

  it('shows the DVE Name input field', () => {
    renderWithRouter(<DVECreate />);
    expect(screen.getByText('DVE Name')).toBeInTheDocument();
  });

  it('shows step counter starting at Step 1', () => {
    renderWithRouter(<DVECreate />);
    expect(screen.getByText('Step 1 of 5')).toBeInTheDocument();
  });

  it('navigates to step 2 (Stake) via "Next" button', () => {
    renderWithRouter(<DVECreate />);

    // Fill in name and select TEE type to enable Next button
    const nameInput = screen.getByPlaceholderText('e.g. DVE-Alpha');
    fireEvent.change(nameInput, { target: { value: 'My Test DVE' } });

    // Click Hardware TEE to select it
    fireEvent.click(screen.getByText('Hardware TEE'));

    // Now click Next
    const nextButton = screen.getByText('Next');
    fireEvent.click(nextButton);

    // Should now see Stake NRN heading
    expect(screen.getByText('Stake NRN')).toBeInTheDocument();
    expect(screen.getByText('Step 2 of 5')).toBeInTheDocument();
  });

  it('shows the step indicator with "Stake" label', () => {
    renderWithRouter(<DVECreate />);
    expect(screen.getByText('Stake')).toBeInTheDocument();
  });

  it('shows "Step 1 of 5" initially', () => {
    renderWithRouter(<DVECreate />);
    expect(screen.getByText('Step 1 of 5')).toBeInTheDocument();
  });

  it('renders page description text', () => {
    renderWithRouter(<DVECreate />);
    expect(screen.getByText('New Distributed Verification Environment')).toBeInTheDocument();
  });
});
