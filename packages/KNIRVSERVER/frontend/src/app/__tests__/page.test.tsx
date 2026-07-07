import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import Page from '../page';

// Mock the auth context to avoid complex setup
jest.mock('../../lib/auth-context', () => ({
  useAuth: () => ({
    user: { id: '1', email: 'test@example.com', role: 'admin' },
    isAuthenticated: true,
    login: jest.fn(),
    logout: jest.fn(),
    loading: false,
  }),
}));

// Mock the dashboard wrapper to focus on page structure
jest.mock('../../components/dashboard/dashboard-wrapper', () => ({
  DashboardWrapper: function MockDashboardWrapper() {
    return <div data-testid="dashboard-wrapper">Dashboard Content</div>;
  },
}));

describe('Page Component', () => {
  it('renders without crashing', () => {
    render(<Page />);
    expect(screen.getByTestId('dashboard-wrapper')).toBeInTheDocument();
  });

  it('displays dashboard content when authenticated', () => {
    render(<Page />);
    expect(screen.getByText('Dashboard Content')).toBeInTheDocument();
  });
});
