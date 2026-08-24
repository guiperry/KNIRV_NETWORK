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
              </article>
            ))}
            {filtered.length === 0 && (
              <p className="text-gray-400">No active postings match this filter.</p>
            )}
          </section>
        )}
      </div>
    </main>
  );
}
