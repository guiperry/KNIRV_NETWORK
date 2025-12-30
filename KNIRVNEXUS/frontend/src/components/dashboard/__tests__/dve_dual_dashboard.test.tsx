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

test('renders DVE dual dashboard', () => {
  render(<DVEDualDashboard />)
  // Add more assertions as needed
})