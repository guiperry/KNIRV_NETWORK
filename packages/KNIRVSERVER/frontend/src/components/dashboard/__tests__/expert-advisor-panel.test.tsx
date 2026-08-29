import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { ExpertAdvisorPanel } from '../expert-advisor-panel';

const updatePolicy = jest.fn();
const toast = jest.fn();
const mockUseDVEManagement = jest.fn();

jest.mock('@/hooks/use-dve-management', () => ({ useDVEManagement: () => mockUseDVEManagement() }));
jest.mock('@/hooks/use-dve-sessions', () => ({ useDVESessions: () => ({ sessions: [], loading: false, error: null }) }));
jest.mock('@/hooks/use-toast', () => ({ useToast: () => ({ toast }) }));
jest.mock('@/components/ui/card', () => ({
  Card: ({ children }: any) => <div>{children}</div>, CardContent: ({ children }: any) => <div>{children}</div>,
  CardDescription: ({ children }: any) => <div>{children}</div>, CardHeader: ({ children }: any) => <div>{children}</div>,
  CardTitle: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/components/ui/button', () => ({ Button: ({ children, ...props }: any) => <button {...props}>{children}</button> }));
jest.mock('@/components/ui/badge', () => ({ Badge: ({ children }: any) => <span>{children}</span> }));
jest.mock('@/components/ui/alert', () => ({ Alert: ({ children }: any) => <div role="alert">{children}</div>, AlertDescription: ({ children }: any) => <div>{children}</div> }));
jest.mock('@/components/ui/input', () => ({ Input: (props: any) => <input {...props} /> }));
jest.mock('@/components/ui/label', () => ({ Label: ({ children }: any) => <label>{children}</label> }));
jest.mock('@/components/ui/select', () => ({ Select: ({ children }: any) => <div>{children}</div>, SelectContent: ({ children }: any) => <div>{children}</div>, SelectItem: ({ children }: any) => <div>{children}</div>, SelectTrigger: ({ children }: any) => <div>{children}</div>, SelectValue: () => null }));
jest.mock('@/components/ui/switch', () => ({ Switch: () => <button /> }));
jest.mock('lucide-react', () => new Proxy({}, { get: () => () => <span /> }));

const creation: any = {
  id: 'creation-1', name: 'Advisor One', owner_id: 'owner-1', dve_node_id: 'node-1', status: 'active', tee_type: 'sgx',
  stake_amount: 100, persistent: true,
  policy: { mode: 'enforce', allowed_providers: ['openai'], allowed_models: [], denied_models: [], max_requests_per_hour: 10, max_token_budget_daily: 1000, fail_open: false, updated_at: '2026-01-01T00:00:00Z' },
};

describe('ExpertAdvisorPanel', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseDVEManagement.mockReturnValue({ creations: [creation], isLoading: false, error: null, updatePolicy });
  });

  it('renders advisor policy, preserves it for editing, and saves updates', () => {
    render(<ExpertAdvisorPanel />);
    fireEvent.click(screen.getByRole('button', { name: 'Expand Advisor One' }));

    const providers = screen.getByPlaceholderText('e.g. openai, anthropic');
    expect(providers).toHaveValue('openai');
    fireEvent.change(providers, { target: { value: 'openai, anthropic' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save Policy' }));

    expect(updatePolicy).toHaveBeenCalledWith('creation-1', expect.objectContaining({
      allowed_providers: ['openai', 'anthropic'],
      mode: 'enforce',
    }));
  });

  it('shows the management error', () => {
    mockUseDVEManagement.mockReturnValue({ creations: [], isLoading: false, error: 'request failed', updatePolicy });
    render(<ExpertAdvisorPanel />);
    expect(screen.getByRole('alert')).toHaveTextContent('request failed');
  });
});
