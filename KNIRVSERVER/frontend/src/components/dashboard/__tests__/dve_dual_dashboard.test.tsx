import { render } from '@testing-library/react'
import DVEDualDashboard from '../dve_dual_dashboard'

jest.mock('next/navigation', () => ({
  useRouter() {
    return {
      push: jest.fn(),
      replace: jest.fn(),
      prefetch: jest.fn(),
    }
  },
  useSearchParams() {
    return {
      get: jest.fn(),
    }
  },
}))

jest.mock('@/contexts/demo-mode-context', () => ({
  useDemoMode: () => ({ isDemoMode: false }),
}))

jest.mock('@/hooks/use-dve-nodes', () => ({
  useDVENodes: () => ({
    nodes: [],
    loading: false,
    error: null,
  }),
}))

jest.mock('lucide-react', () => ({
  Play: () => <span data-testid="PlayIcon" />,
  Upload: () => <span data-testid="UploadIcon" />,
  Shield: () => <span data-testid="ShieldIcon" />,
  Activity: () => <span data-testid="ActivityIcon" />,
  FileCode: () => <span data-testid="FileCodeIcon" />,
  Network: () => <span data-testid="NetworkIcon" />,
  AlertTriangle: () => <span data-testid="AlertTriangleIcon" />,
  CheckCircle: () => <span data-testid="CheckCircleIcon" />,
  XCircle: () => <span data-testid="XCircleIcon" />,
  Search: () => <span data-testid="SearchIcon" />,
  Filter: () => <span data-testid="FilterIcon" />,
  Terminal: () => <span data-testid="TerminalIcon" />,
  Eye: () => <span data-testid="EyeIcon" />,
  Lock: () => <span data-testid="LockIcon" />,
  Settings: () => <span data-testid="SettingsIcon" />,
  Brain: () => <span data-testid="BrainIcon" />,
}));

test('renders DVE dual dashboard', () => {
  render(<DVEDualDashboard />)
  // Add more assertions as needed
})