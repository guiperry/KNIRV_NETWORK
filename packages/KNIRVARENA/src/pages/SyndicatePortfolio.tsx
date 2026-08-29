import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  actuarialSyndicateService,
  type ExposureSummary,
  type PoolReport,
  type SyndicatePool,
} from '../services/ActuarialSyndicateService';
import { walletIntegrationService } from '../services/WalletIntegrationService';
import { ResolverClaims } from '../components/syndicate/ResolverClaims';
import type { ResolverClaim } from '../services/ActuarialSyndicateService';

const amount = (value: number) => new Intl.NumberFormat('en-US').format(value ?? 0);

export default function SyndicatePortfolio() {
  const [wallet, setWallet] = useState(
    () => walletIntegrationService.getCurrentAccount()?.address ?? ''
  );
  const [pools, setPools] = useState<SyndicatePool[]>([]);
  const [exposure, setExposure] = useState<ExposureSummary | null>(null);
  const [reports, setReports] = useState<Record<string, PoolReport>>({});
  const [claims, setClaims] = useState<ResolverClaim[]>([]);
  const [claimsLoading, setClaimsLoading] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    actuarialSyndicateService
      .listPools()
      .then(value => {
        if (active) setPools(value);
      })
      .catch(() => {
        if (active) setError('Pool capacity is unavailable on the selected network.');
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  const loadExposure = async () => {
    if (!wallet.trim()) {
      setError('Connect a wallet or enter its address to view its testnet exposure.');
      return;
    }
    setError('');
    try {
      setExposure(await actuarialSyndicateService.getExposure(wallet.trim()));
    } catch (cause) {
      setExposure(null);
      setError(cause instanceof Error ? cause.message : 'Exposure could not be loaded.');
    }
  };

  const loadClaims = async () => {
    if (!wallet.trim()) {
      setError('Connect a wallet or enter its address to view assigned work.');
      return;
    }
    setClaimsLoading(true);
    try {
      setClaims(await actuarialSyndicateService.listResolverClaims(wallet.trim()));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Assigned work could not be loaded.');
    } finally {
      setClaimsLoading(false);
    }
  };

  const loadReport = async (poolID: string) => {
    try {
      const report = await actuarialSyndicateService.getPoolReport(poolID);
      setReports(current => ({
        ...current,
        [poolID]: report,
      }));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Pool report could not be loaded.');
    }
  };

  return (
    <main className="min-h-screen bg-gray-900 p-6 text-white">
      <div className="mx-auto max-w-5xl">
        <Link to="/manager/bounties" className="text-sm text-blue-300 hover:text-blue-200">
          ← Bounty Board
        </Link>
        <h1 className="mt-3 text-3xl font-bold">Syndicate Portfolio</h1>
        <p className="mt-2 text-gray-300">
          Live testnet pool capacity, reserve commitments, and wallet exposure. Payout entries are
          drawn from the reserve ledger.
        </p>

        <section className="mt-6 rounded-lg border border-gray-700 bg-gray-800 p-5">
          <h2 className="text-lg font-semibold">Your capital exposure</h2>
          <div className="mt-3 flex flex-col gap-2 sm:flex-row">
            <input
              aria-label="Wallet address"
              value={wallet}
              onChange={event => setWallet(event.target.value)}
              placeholder="KNIRV wallet address"
              className="flex-1 rounded bg-gray-950 px-3 py-2 text-sm"
            />
            <button onClick={loadExposure} className="rounded bg-blue-600 px-4 py-2 font-medium hover:bg-blue-500">Load exposure</button>
            <button onClick={loadClaims} className="rounded bg-cyan-700 px-4 py-2 font-medium hover:bg-cyan-600">Load work</button>
          </div>
          {exposure && (
            <div className="mt-4 grid gap-3 sm:grid-cols-3 text-sm">
              <Metric label="Gross exposure" value={amount(exposure.gross_exposure)} />
              <Metric label="Realized loss" value={amount(exposure.realized_loss)} />
              <Metric label="Stake positions" value={String(exposure.positions.length)} />
              {exposure.positions.map(position => (
                <div
                  key={position.stake.id}
                  className="sm:col-span-3 rounded bg-gray-900 p-3 text-gray-300"
                >
                  <span className="font-mono text-xs">{position.stake.id}</span> ·{' '}
                  {position.stake.status} · locked {amount(position.stake.locked_amount)} · loss{' '}
                  {amount(position.realized_loss)}
                </div>
              ))}
            </div>
          )}
        </section>
        <ResolverClaims claims={claims} loading={claimsLoading} />

        {error && (
          <p role="alert" className="mt-4 text-red-300">
            {error}
          </p>
        )}
        <section className="mt-6 grid gap-4 md:grid-cols-2">
          {loading && <p className="text-gray-300">Loading live pools…</p>}
          {pools.map(pool => (
            <article key={pool.id} className="rounded-lg border border-gray-700 bg-gray-800 p-5">
              <h2 className="font-semibold">
                Pool <span className="font-mono text-xs text-gray-400">{pool.id}</span>
              </h2>
              <p className="mt-2 text-sm text-gray-300">
                {pool.currency} via {pool.rail} · {pool.status}
              </p>
              <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                <Metric label="Total stake" value={amount(pool.total_stake)} />
                <Metric
                  label="Available liquidity"
                  value={amount(pool.liquid_balance - pool.reserved_balance)}
                />
              </div>
              <button
                onClick={() => loadReport(pool.id)}
                className="mt-4 text-sm text-blue-300 hover:text-blue-200"
              >
                View reserve and payout entries
              </button>
              {reports[pool.id] && (
                <p className="mt-3 text-sm text-gray-300">
                  {reports[pool.id].active_stake_count} active stakers ·{' '}
                  {reports[pool.id].reserve_entries.length} ledger entries · available{' '}
                  {amount(reports[pool.id].available_liquidity)}
                </p>
              )}
            </article>
          ))}
        </section>
        {!loading && pools.length === 0 && (
          <p className="mt-6 text-gray-400">No pools are available on this network.</p>
        )}
      </div>
    </main>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded bg-gray-900 p-3">
      <p className="text-xs uppercase tracking-wide text-gray-400">{label}</p>
      <p className="mt-1 font-medium">{value}</p>
    </div>
  );
}
