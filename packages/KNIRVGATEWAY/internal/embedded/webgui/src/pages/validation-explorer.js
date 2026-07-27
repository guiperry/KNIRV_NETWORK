import React, { useCallback, useEffect, useState } from 'react';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import { useNavigation } from '../hooks/useNavigation';
import { validationChainApi } from '../services/api-client';

function formatTime(value) {
  if (!value) return 'N/A';
  const date = typeof value === 'number' ? new Date(value < 1e12 ? value * 1000 : value) : new Date(value);
  return Number.isNaN(date.getTime()) ? 'N/A' : date.toLocaleString();
}

export default function ValidationExplorer() {
  const { activePage } = useNavigation('validation-explorer');
  const [health, setHealth] = useState(null);
  const [height, setHeight] = useState(null);
  const [blocks, setBlocks] = useState([]);
  const [error, setError] = useState(null);
  const [isLoading, setIsLoading] = useState(true);

  const refresh = useCallback(async () => {
    setIsLoading(true);
    const [healthResult, heightResult, blocksResult] = await Promise.allSettled([
      validationChainApi.getHealth(),
      validationChainApi.getHeight(),
      validationChainApi.getBlocks(),
    ]);
    const failures = [healthResult, heightResult, blocksResult]
      .filter((result) => result.status === 'rejected')
      .map((result) => result.reason.message);
    if (healthResult.status === 'fulfilled') setHealth(healthResult.value);
    if (heightResult.status === 'fulfilled') setHeight(heightResult.value.height ?? heightResult.value);
    if (blocksResult.status === 'fulfilled') setBlocks(blocksResult.value);
    setError(failures.length ? failures.join(' • ') : null);
    setIsLoading(false);
  }, []);

  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, 15_000);
    return () => clearInterval(interval);
  }, [refresh]);

  return (
    <PageLayout activePage={activePage} pageTitle="Validation Explorer">
      <PageHeader title="Validation Chain Explorer" subtitle="Live checkpoint-merkle source status and blocks" titleColor="#28a745" />
      {error && <GlassyCard darker style={{ padding: '12px', marginBottom: '16px', color: '#ffb4b4' }}>Unable to load validation-chain data: {error}</GlassyCard>}
      <div style={{ display: 'flex', gap: '10px', marginBottom: '16px' }}><button onClick={refresh} disabled={isLoading} style={buttonStyle}>{isLoading ? 'Refreshing…' : 'Refresh'}</button></div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px', marginBottom: '16px' }}>
        <Metric label="Status" value={health?.healthy === false ? 'Unhealthy' : health?.status || 'Unknown'} />
        <Metric label="Chain ID" value={health?.chain_id || 'validation-chain'} />
        <Metric label="Height" value={height ?? health?.height ?? 'N/A'} />
        <Metric label="Blocks available" value={blocks.length} />
      </div>
      <GlassyCard darker style={{ padding: '16px' }}>
        <h3 style={{ marginTop: 0 }}>Recent validation blocks</h3>
        {blocks.length === 0 ? <div>No validation blocks available.</div> : blocks.map((block, index) => <div key={block.hash || block.id || index} style={blockStyle}><strong>Block #{block.index ?? block.height ?? index}</strong><div>Hash: {(block.hash || block.id || 'N/A').slice(0, 24)}</div><small>{formatTime(block.timestamp || block.time)}</small></div>)}
      </GlassyCard>
    </PageLayout>
  );
}

function Metric({ label, value }) { return <GlassyCard darker style={{ padding: '16px', textAlign: 'center' }}><div style={{ fontSize: '24px', fontWeight: 'bold' }}>{value}</div><div style={{ fontSize: '12px', marginTop: '4px' }}>{label}</div></GlassyCard>; }
const buttonStyle = { padding: '8px 16px', borderRadius: '6px', border: '1px solid rgba(100,200,130,.3)', background: 'rgba(30,40,70,.5)', color: '#b0ffb0', cursor: 'pointer' };
const blockStyle = { background: 'rgba(30,60,70,.5)', border: '1px solid rgba(100,200,130,.2)', borderRadius: '8px', padding: '10px', marginBottom: '8px' };
