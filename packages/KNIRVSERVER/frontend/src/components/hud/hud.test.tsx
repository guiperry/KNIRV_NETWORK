import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { HudOverlay } from './HudOverlay';

// --- module mocks -------------------------------------------------------

const mockRouterPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockRouterPush }),
}));

jest.mock('@/hooks/use-telemetry', () => ({
  useTelemetry: () => ({ telemetry: { data: null } }),
}));

jest.mock('@/hooks/use-ontology', () => ({
  useOntology: () => ({ stats: { data: null } }),
}));

const mockApplyUpdate = jest.fn();
let mockUpdateStatus: { available: boolean; latest_tag: string; current_version: string } | null = null;
let mockUpdateApplying = false;

jest.mock('@/hooks/use-update-check', () => ({
  useUpdateCheck: () => ({
    status: mockUpdateStatus,
    applying: mockUpdateApplying,
    applyUpdate: mockApplyUpdate,
  }),
}));

// --- fetch mock ---------------------------------------------------------

global.fetch = jest.fn();
const mockFetch = fetch as jest.MockedFunction<typeof fetch>;

const metricsResponse = () =>
  Promise.resolve({
    ok: true,
    json: () =>
      Promise.resolve({
        cpu: 45.5,
        memory: { total_mb: 16384, used_mb: 8192, available_mb: 8192, percentage: 50 },
        uptime_seconds: 3600,
        os: 'linux',
        arch: 'amd64',
        hostname: 'test-host',
      }),
  } as unknown as Response);

// --- helpers ------------------------------------------------------------

function setUpdateStatus(available: boolean, latestTag = 'v2.0.0') {
  mockUpdateStatus = { available, latest_tag: latestTag, current_version: 'v1.0.0' };
}

// ========================================================================

describe('HudOverlay', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.clearAllMocks();
    mockUpdateStatus = null;
    mockUpdateApplying = false;
    mockFetch.mockImplementation(metricsResponse);
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('renders without crashing and shows connection status', () => {
    render(<HudOverlay />);
    expect(screen.getAllByText('CONNECTING').length).toBeGreaterThan(0);
  });

  it('renders ONLINE once metrics are fetched', async () => {
    render(<HudOverlay />);
    await waitFor(() => {
      expect(screen.getAllByText('ONLINE').length).toBeGreaterThan(0);
    });
    expect(screen.getAllByText('45.5%').length).toBeGreaterThan(0);
  });

  it('shows ERROR connection status on fetch failure', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));
    render(<HudOverlay />);
    await waitFor(() => {
      expect(screen.getAllByText('ERROR').length).toBeGreaterThan(0);
    });
  });

  it('minimizes and restores the HUD', async () => {
    render(<HudOverlay />);
    await waitFor(() => expect(screen.getAllByText('ONLINE').length).toBeGreaterThan(0));

    fireEvent.click(screen.getByTitle('Minimize'));
    expect(screen.getByTitle('Restore HUD')).toBeInTheDocument();

    fireEvent.click(screen.getByTitle('Restore HUD'));
    expect(screen.getByTitle('Minimize')).toBeInTheDocument();
  });

  it('MENU button navigates to /menu', async () => {
    render(<HudOverlay />);
    await waitFor(() => expect(screen.getAllByText('ONLINE').length).toBeGreaterThan(0));

    fireEvent.click(screen.getByText(/MENU/));
    expect(mockRouterPush).toHaveBeenCalledWith('/menu');
  });

  it('formats uptime correctly', async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve({
        ok: true,
        json: () =>
          Promise.resolve({
            cpu: 50,
            memory: { total_mb: 8192, used_mb: 4096, available_mb: 4096, percentage: 50 },
            uptime_seconds: 3661,
            os: 'linux',
            arch: 'amd64',
            hostname: 'host',
          }),
      } as unknown as Response),
    );

    render(<HudOverlay />);
    await waitFor(() => {
      expect(screen.getAllByText('1h 1m').length).toBeGreaterThan(0);
    });
  });

  // ── UPDATE button ─────────────────────────────────────────────────────

  describe('UPDATE button', () => {
    it('shows UPDATE button when an update is available', async () => {
      setUpdateStatus(true, 'v2.0.0');
      render(<HudOverlay />);

      expect(screen.getByRole('button', { name: 'Update available' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Update available' })).toHaveTextContent('UPDATE');
    });

    it('does not show UPDATE button when up to date', () => {
      setUpdateStatus(false, 'v1.0.0');
      render(<HudOverlay />);
      expect(screen.queryByRole('button', { name: 'Update available' })).not.toBeInTheDocument();
    });

    it('does not show UPDATE button when update status is unknown', () => {
      mockUpdateStatus = null;
      render(<HudOverlay />);
      expect(screen.queryByRole('button', { name: 'Update available' })).not.toBeInTheDocument();
    });

    it('calls applyUpdate when UPDATE is clicked', () => {
      setUpdateStatus(true, 'v2.0.0');
      render(<HudOverlay />);

      fireEvent.click(screen.getByRole('button', { name: 'Update available' }));
      expect(mockApplyUpdate).toHaveBeenCalledTimes(1);
    });

    it('shows RESTARTING text and is disabled while applying', () => {
      setUpdateStatus(true, 'v2.0.0');
      mockUpdateApplying = true;
      render(<HudOverlay />);

      const btn = screen.getByRole('button', { name: 'Update available' });
      expect(btn).toHaveTextContent('RESTARTING');
      expect(btn).toBeDisabled();
    });

    it('title contains the latest version tag', () => {
      setUpdateStatus(true, 'v3.1.4');
      render(<HudOverlay />);

      const btn = screen.getByRole('button', { name: 'Update available' });
      expect(btn.getAttribute('title')).toContain('v3.1.4');
    });

    it('has aria-label "Update available"', () => {
      setUpdateStatus(true);
      render(<HudOverlay />);

      expect(
        screen.getByRole('button', { name: 'Update available' }),
      ).toHaveAttribute('aria-label', 'Update available');
    });
  });
});
