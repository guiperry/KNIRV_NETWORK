import type { ResolverClaim } from '../../services/ActuarialSyndicateService';

type Props = { claims: ResolverClaim[]; loading: boolean };

/** Resolver labour is displayed independently of pool capital positions. */
export function ResolverClaims({ claims, loading }: Props) {
  return (
    <section className="mt-6 rounded-lg border border-cyan-800 bg-gray-800 p-5">
      <h2 className="text-lg font-semibold">What I’m working</h2>
      <p className="mt-1 text-sm text-gray-300">
        Claimed resolver assignments earn the labour payout; capital positions remain listed above.
      </p>
      {loading && <p className="mt-3 text-sm text-gray-300">Loading claimed work…</p>}
      {!loading && claims.length === 0 && (
        <p className="mt-3 text-sm text-gray-400">No bounty submissions are currently assigned to this wallet.</p>
      )}
      <div className="mt-3 space-y-2">
        {claims.map(claim => (
          <article key={claim.id} className="rounded bg-gray-900 p-3 text-sm">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-mono text-xs text-cyan-300">{claim.id}</span>
              <span className="rounded bg-cyan-950 px-2 py-1 text-xs text-cyan-200">{claim.status}</span>
            </div>
            <p className="mt-2 text-gray-300">
              {claim.domain.replace('_', ' ')} · risk class {claim.claimed_risk_class} · validation {claim.validation_state}
            </p>
          </article>
        ))}
      </div>
    </section>
  );
}
