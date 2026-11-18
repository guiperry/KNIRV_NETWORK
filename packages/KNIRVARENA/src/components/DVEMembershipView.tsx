import React from 'react';
import type { DveStagedRecord } from '../storage/dveStaging';
import type { DveMembershipEntry } from '../services/dveMembershipClient';

/**
 * The DVE session's view of its own errors (DVE-U26, plan D7.6): staged items
 * held locally in this session, then the error nodes this DVE is a member of,
 * each with its NRV suite progress (n/8). The main Arena does not use this view.
 */
export interface DVEMembershipViewProps {
  dveId: string;
  staged: DveStagedRecord[];
  /** null while loading or when the list could not be loaded. */
  members: DveMembershipEntry[] | null;
  loading?: boolean;
  error?: string | null;
}

const suiteLabel = (m: DveMembershipEntry): string => {
  const progress = `${m.suite.count}/${m.suite.required}`;
  return m.suite.status === 'sealed' ? `${progress} sealed` : `${progress} collecting`;
};

export const DVEMembershipView: React.FC<DVEMembershipViewProps> = ({
  dveId,
  staged,
  members,
  loading = false,
  error = null
}) => (
  <section className="dve-membership-view" aria-label={`DVE ${dveId} errors`} data-dve-id={dveId}>
    <h2>Staged in this session</h2>
    {staged.length === 0 ? (
      <p className="dve-membership-empty">Nothing staged. Staged errors stay here until promoted.</p>
    ) : (
      <ul className="dve-membership-staged">
        {staged.map((s) => (
          <li key={`${s.dveId}:${s.errorId}`} data-state={s.state}>
            <span className="dve-membership-error-id">{s.errorId}</span>{' '}
            <span className="dve-membership-state">{s.state}</span>
            {s.networkErrorNodeId ? (
              <span className="dve-membership-node-link"> → {s.networkErrorNodeId}</span>
            ) : null}
          </li>
        ))}
      </ul>
    )}

    <h2>Error nodes this DVE is a member of</h2>
    {loading ? (
      <p className="dve-membership-loading">Loading…</p>
    ) : error ? (
      <p className="dve-membership-error" role="alert">{error}</p>
    ) : !members || members.length === 0 ? (
      <p className="dve-membership-empty">No error nodes yet.</p>
    ) : (
      <ul className="dve-membership-nodes">
        {members.map((m) => (
          <li key={m.error_node_id} data-role={m.role} data-found={m.found ? 'true' : 'false'}>
            <span className="dve-membership-error-id">{m.error_node_id}</span>{' '}
            <span className="dve-membership-role">{m.role}</span>{' '}
            {m.found ? (
              <span className="dve-membership-status">{m.status ?? 'unknown'}</span>
            ) : (
              <span className="dve-membership-status">not listed by KNIRVGRAPH</span>
            )}{' '}
            <span className="dve-membership-suite" data-status={m.suite.status}>{suiteLabel(m)}</span>
          </li>
        ))}
      </ul>
    )}
  </section>
);

export default DVEMembershipView;
