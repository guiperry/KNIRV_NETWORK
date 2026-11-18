import React from 'react';
import { render, screen, within } from '@testing-library/react';
import { DVEMembershipView } from '../../../src/components/DVEMembershipView';
import type { DveStagedRecord } from '../../../src/storage/dveStaging';
import type { DveMembershipEntry } from '../../../src/services/dveMembershipClient';

const staged: DveStagedRecord[] = [
  {
    dveId: 'dve-a',
    errorId: 'local-err-1',
    state: 'staged',
    payload: {},
    createdAt: '2026-11-16T00:00:00Z',
    updatedAt: '2026-11-16T00:00:00Z'
  },
  {
    dveId: 'dve-a',
    errorId: 'local-err-2',
    state: 'promoted',
    payload: {},
    networkErrorNodeId: 'node-p',
    createdAt: '2026-11-16T00:00:00Z',
    updatedAt: '2026-11-16T00:00:00Z'
  }
];

const members: DveMembershipEntry[] = [
  {
    error_node_id: 'node-a',
    role: 'originator',
    seen_at: '2026-11-16T00:00:00Z',
    found: true,
    status: 'open',
    suite: { status: 'collecting', count: 5, required: 8, progress: '5/8' }
  },
  {
    error_node_id: 'node-b',
    role: 'witness',
    seen_at: '2026-11-16T00:00:00Z',
    found: true,
    status: 'open',
    suite: { status: 'sealed', count: 8, required: 8, progress: '8/8', suite_hash: 'h' }
  },
  {
    error_node_id: 'node-gone',
    role: 'witness',
    seen_at: '2026-11-16T00:00:00Z',
    found: false,
    suite: { status: 'collecting', count: 0, required: 8, progress: '0/8' }
  }
];

describe('DVEMembershipView', () => {
  it('lists staged items and the membership with n/8 for each node', () => {
    render(<DVEMembershipView dveId="dve-a" staged={staged} members={members} />);

    const stagedList = screen.getByText('Staged in this session').parentElement as HTMLElement;
    expect(within(stagedList).getByText('local-err-1')).toBeInTheDocument();
    expect(within(stagedList).getByText('promoted')).toBeInTheDocument();
    expect(within(stagedList).getByText(/node-p/)).toBeInTheDocument();

    const nodeRow = (id: string) => screen.getByText(id).closest('li') as HTMLElement;
    expect(within(nodeRow('node-a')).getByText('originator')).toBeInTheDocument();
    expect(within(nodeRow('node-a')).getByText('5/8 collecting')).toBeInTheDocument();
    expect(within(nodeRow('node-b')).getByText('witness')).toBeInTheDocument();
    expect(within(nodeRow('node-b')).getByText('8/8 sealed')).toBeInTheDocument();
    expect(within(nodeRow('node-gone')).getByText('not listed by KNIRVGRAPH')).toBeInTheDocument();
  });

  it('shows the empty states when nothing is staged or joined', () => {
    render(<DVEMembershipView dveId="dve-a" staged={[]} members={[]} />);
    expect(screen.getByText(/Nothing staged/)).toBeInTheDocument();
    expect(screen.getByText('No error nodes yet.')).toBeInTheDocument();
  });

  it('shows the error instead of a list when the membership failed to load', () => {
    render(<DVEMembershipView dveId="dve-a" staged={staged} members={null} error="error nodes unavailable (503)" />);
    expect(screen.getByRole('alert')).toHaveTextContent('error nodes unavailable (503)');
    expect(screen.queryByText('node-a')).not.toBeInTheDocument();
  });
});
