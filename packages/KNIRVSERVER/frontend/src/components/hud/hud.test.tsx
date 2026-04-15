import { describe, it, expect, vi, beforeEach, afterEach } from '@jest/globals';
import { render, screen, waitFor } from '@testing-library/react';
import { HudOverlay } from './HudOverlay';

global.fetch = vi.fn();

describe('HudOverlay', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders loading state initially', async () => {
    vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));

    render(<HudOverlay />);

    expect(screen.getByText('Loading metrics...')).toBeInTheDocument();
  });

  it('renders metrics when fetched successfully', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        cpu: 45.5,
        memory: {
          total_mb: 16384,
          used_mb: 8192,
          available_mb: 8192,
          percentage: 50,
        },
        uptime_seconds: 3600,
        os: 'linux',
        arch: 'amd64',
        hostname: 'test-host',
      }),
    } as unknown as Response);

    render(<HudOverlay />);

    await waitFor(() => {
      expect(screen.queryByText('Loading metrics...')).not.toBeInTheDocument();
    });

    expect(screen.getByText('45.5%')).toBeInTheDocument();
    expect(screen.getByText(/8GB/)).toBeInTheDocument();
    expect(screen.getByText('linux amd64')).toBeInTheDocument();
    expect(screen.getByText('test-host')).toBeInTheDocument();
  });

  it('renders error state on fetch failure', async () => {
    vi.mocked(fetch).mockRejectedValueOnce(new Error('Network error'));

    render(<HudOverlay />);

    await waitFor(() => {
      expect(screen.queryByText('Loading metrics...')).not.toBeInTheDocument();
    });

    expect(screen.getByText('●')).toBeInTheDocument();
  });

  it('handles minimize toggle', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        cpu: 50,
        memory: { total_mb: 16384, used_mb: 8192, available_mb: 8192, percentage: 50 },
        uptime_seconds: 3600,
        os: 'linux',
        arch: 'amd64',
        hostname: 'test',
      }),
    } as unknown as Response);

    render(<HudOverlay />);

    await waitFor(() => {
      expect(screen.queryByText('Loading metrics...')).not.toBeInTheDocument();
    });

    const minimizeButton = screen.getByRole('button', { name: /minimize/i });
    minimizeButton.click();

    expect(screen.getByRole('button', { name: /restore/i })).toBeInTheDocument();
  });

  it('formats uptime correctly', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        cpu: 50,
        memory: { total_mb: 16384, used_mb: 8192, available_mb: 8192, percentage: 50 },
        uptime_seconds: 90061,
        os: 'linux',
        arch: 'amd64',
        hostname: 'test',
      }),
    } as unknown as Response);

    render(<HudOverlay />);

    await waitFor(() => {
      expect(screen.queryByText('Loading metrics...')).not.toBeInTheDocument();
    });

    expect(screen.getByText('1d 1h 1m')).toBeInTheDocument();
  });
});