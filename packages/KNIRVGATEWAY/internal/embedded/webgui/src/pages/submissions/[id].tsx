import { useRouter } from 'next/router';
import { useEffect, useState } from 'react';
import { useRole } from '../../contexts/RoleContext';

type Status = { id: string; domain: string; status: string; validation_state: string; validation_result_hash?: string; decision_id?: string; updated_at: string; quote?: { amount: number; reserve_amount: number; expires_at: string }; settlement?: { id: string; status: string; receipt_hash?: string; chain_tx_hash?: string } };

export default function EnterpriseSubmissionTracking() {
  const router = useRouter(); const { isAuthenticated, network } = useRole(); const id = typeof router.query.id === 'string' ? router.query.id : '';
  const [status, setStatus] = useState<Status | null>(null); const [error, setError] = useState('');
  useEffect(() => {
    if (!isAuthenticated || !id) return;
    let active = true;
    const load = () => fetch(`/api/actuarial/submissions/${encodeURIComponent(id)}/status`, { headers: { authorization: `Bearer ${localStorage.getItem('knirv_auth_token') || ''}`, 'x-knirv-network-id': network || 'local' } }).then(async response => { const data = await response.json(); if (!response.ok) throw new Error(data.error || 'Unable to load submission'); if (active) setStatus(data); }).catch(cause => active && setError(cause instanceof Error ? cause.message : 'Unable to load submission'));
    void load(); const timer = window.setInterval(() => void load(), 10000); return () => { active = false; window.clearInterval(timer); };
  }, [id, isAuthenticated, network]);
  if (!isAuthenticated) return <main className="p-8"><p>Open this page from an authenticated KNIRVSERVER dashboard session.</p></main>;
  return <main className="mx-auto max-w-2xl p-8"><h1 className="text-3xl font-bold">Submission tracking</h1>{error && <p className="mt-4 text-red-700">{error}</p>}{!status && !error && <p className="mt-4">Loading…</p>}{status && <section className="mt-6 space-y-2 rounded border p-4"><p><strong>ID:</strong> {status.id}</p><p><strong>Domain:</strong> {status.domain}</p><p><strong>State:</strong> {status.status}</p><p><strong>Validation:</strong> {status.validation_state}</p><p><strong>Quote decision:</strong> {status.decision_id || 'pending validation and pricing'}</p>{status.quote && <div className="mt-4 border-t pt-3"><strong>Quote</strong><p>Amount: {status.quote.amount}</p><p>Reserved: {status.quote.reserve_amount}</p><p>Expires: {status.quote.expires_at}</p></div>}{status.settlement && <div className="mt-4 border-t pt-3"><strong>Settlement</strong><p>Status: {status.settlement.status}</p><p>Receipt: {status.settlement.receipt_hash || 'pending'}</p></div>}<p className="text-sm text-gray-500">Updates automatically every 10 seconds.</p></section>}</main>;
}
