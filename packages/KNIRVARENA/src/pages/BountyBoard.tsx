import React, { useEffect, useMemo, useState } from 'react';
import {
  actuarialSyndicateService,
  type BountyDomain,
  type BountyPosting,
} from '../services/ActuarialSyndicateService';

export default function BountyBoard() {
  const [postings, setPostings] = useState<BountyPosting[]>([]);
  const [domain, setDomain] = useState<BountyDomain | 'all'>('all');
  const [state, setState] = useState<'loading' | 'ready' | 'error'>('loading');
  const [selectedCodeError, setSelectedCodeError] = useState<BountyPosting | null>(null);
  const [wallet, setWallet] = useState('');
  const [credentialID, setCredentialID] = useState('');
  const [pocReference, setPocReference] = useState('');
  const [reportReference, setReportReference] = useState('');
  const [submissionState, setSubmissionState] = useState<'idle' | 'submitting' | 'done' | 'error'>('idle');
  const [submissionMessage, setSubmissionMessage] = useState('');

  useEffect(() => {
    let current = true;
    actuarialSyndicateService
      .listPostings()
      .then(items => {
        if (current) {
          setPostings(items);
          setState('ready');
        }
      })
      .catch(() => {
        if (current) setState('error');
      });
    return () => {
      current = false;
    };
  }, []);

  const filtered = useMemo(
    () =>
      postings.filter(
        posting => posting.status === 'active' && (domain === 'all' || posting.domain === domain)
      ),
    [domain, postings]
  );

  const sha256 = async (value: string) => {
    const bytes = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(value));
    return Array.from(new Uint8Array(bytes), byte => byte.toString(16).padStart(2, '0')).join('');
  };

  const submitCodeError = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!selectedCodeError) return;
    setSubmissionState('submitting');
    setSubmissionMessage('');
    try {
      const submission = await actuarialSyndicateService.createResearcherSubmission({
        researcher_credential_id: credentialID.trim(),
        researcher_wallet: wallet.trim(),
        claimed_risk_class: selectedCodeError.id,
        idempotency_key: crypto.randomUUID(),
      }) as { id: string };
      await actuarialSyndicateService.createArtifacts(submission.id, {
        researcher_wallet: wallet.trim(),
        poc_reference: pocReference.trim(),
        report_reference: reportReference.trim(),
        poc_hash: await sha256(pocReference.trim()),
        report_hash: await sha256(reportReference.trim()),
        scope_hash: await sha256(selectedCodeError.id),
        dve_id: 'knirvana-cde',
        dve_session_id: crypto.randomUUID(),
      });
      setSubmissionState('done');
      setSubmissionMessage('Solution submitted to the shared validation pipeline.');
    } catch (cause) {
      setSubmissionState('error');
      setSubmissionMessage(cause instanceof Error ? cause.message : 'Solution submission failed.');
    }
  };

  return (
    <main className="min-h-screen bg-gray-900 text-white p-6">
      <div className="max-w-5xl mx-auto">
        <h1 className="text-3xl font-bold">Bounty Board</h1>
        <p className="text-gray-300 mt-2">
          One settlement pipeline for curated code errors and security exploits.
        </p>
        <div className="flex gap-2 mt-6" role="group" aria-label="Filter postings by domain">
          {(['all', 'code_error', 'security_exploit'] as const).map(value => (
            <button
              key={value}
              onClick={() => setDomain(value)}
              className={`px-3 py-2 rounded ${domain === value ? 'bg-blue-600' : 'bg-gray-700'}`}
            >
              {value === 'all' ? 'All' : value.replace('_', ' ')}
            </button>
          ))}
        </div>
        {state === 'loading' && (
          <p className="mt-8 text-gray-300">Loading backend-owned postings…</p>
        )}
        {state === 'error' && (
          <p className="mt-8 text-red-300">
            The Bounty Board could not reach KNIRVSERVER. Check the selected network.
          </p>
        )}
        {state === 'ready' && (
          <section className="grid md:grid-cols-2 gap-4 mt-6">
            {filtered.map(posting => (
              <article
                key={posting.id}
                className="rounded-lg border border-gray-700 bg-gray-800 p-5"
              >
                <span className="text-xs uppercase tracking-wide text-blue-300">
                  {posting.domain.replace('_', ' ')}
                </span>
                <h2 className="text-xl font-semibold mt-2">{posting.display_name}</h2>
                <p className="mt-2 text-gray-300">{posting.description}</p>
                {posting.curated_challenge && (
                  <details className="mt-4 text-sm text-gray-300">
                    <summary className="cursor-pointer text-blue-300">
                      Open curated challenge
                    </summary>
                    <p className="mt-2">{posting.curated_challenge.context}</p>
                    <pre className="mt-2 overflow-auto rounded bg-gray-950 p-3 text-xs">
                      {posting.curated_challenge.buggy_code}
                    </pre>
                  </details>
                )}
                {posting.eligibility_policy?.difficulty_tier && (
                  <p className="mt-4 text-sm text-gray-400">
                    Curated difficulty tier: {posting.eligibility_policy.difficulty_tier}
                  </p>
                )}
                {posting.domain === 'code_error' && (
                  <button
                    onClick={() => { setSelectedCodeError(posting); setSubmissionState('idle'); setSubmissionMessage(''); }}
                    className="mt-4 rounded bg-cyan-700 px-3 py-2 text-sm font-medium hover:bg-cyan-600"
                  >
                    Submit solution
                  </button>
                )}
              </article>
            ))}
            {filtered.length === 0 && (
              <p className="text-gray-400">No active postings match this filter.</p>
            )}
          </section>
        )}
        {selectedCodeError && (
          <section className="mt-6 rounded-lg border border-cyan-800 bg-gray-800 p-5">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h2 className="text-lg font-semibold">Submit code-error solution</h2>
                <p className="text-sm text-gray-300">{selectedCodeError.display_name}</p>
              </div>
              <button onClick={() => setSelectedCodeError(null)} className="text-sm text-gray-300 hover:text-white">Close</button>
            </div>
            <form onSubmit={submitCodeError} className="mt-4 grid gap-3">
              <input required value={wallet} onChange={event => setWallet(event.target.value)} placeholder="Resolver wallet" className="rounded bg-gray-950 px-3 py-2 text-sm" />
              <input required value={credentialID} onChange={event => setCredentialID(event.target.value)} placeholder="Researcher credential ID" className="rounded bg-gray-950 px-3 py-2 text-sm" />
              <input required type="url" value={pocReference} onChange={event => setPocReference(event.target.value)} placeholder="HTTPS proof-of-concept reference" className="rounded bg-gray-950 px-3 py-2 text-sm" />
              <input required type="url" value={reportReference} onChange={event => setReportReference(event.target.value)} placeholder="HTTPS resolution report reference" className="rounded bg-gray-950 px-3 py-2 text-sm" />
              <p className="text-xs text-gray-400">References are hashed locally; the shared CDE/DVE validator receives commitments, not solution text.</p>
              <button disabled={submissionState === 'submitting'} className="w-fit rounded bg-cyan-700 px-4 py-2 text-sm font-medium disabled:opacity-60">
                {submissionState === 'submitting' ? 'Submitting…' : 'Submit for validation'}
              </button>
              {submissionMessage && <p role="status" className={submissionState === 'error' ? 'text-sm text-red-300' : 'text-sm text-cyan-200'}>{submissionMessage}</p>}
            </form>
          </section>
        )}
      </div>
    </main>
  );
}
