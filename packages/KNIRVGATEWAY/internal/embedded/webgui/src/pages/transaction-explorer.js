import React, { useCallback, useEffect, useState } from 'react';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import { useNavigation } from '../hooks/useNavigation';
import { transactionChainApi } from '../services/api-client';

function timestamp(value) {
  if (!value) return 'N/A';
  const date = typeof value === 'number' ? new Date(value < 1e12 ? value * 1000 : value) : new Date(value);
  return Number.isNaN(date.getTime()) ? 'N/A' : date.toLocaleString();
}

function transactionsFromBlocks(blocks) {
  return blocks.flatMap((block) => block.transactions || block.data || block.txs || []);
}

export default function TransactionExplorer() {
  const { activePage } = useNavigation('transaction-explorer');
  const [transactions, setTransactions] = useState([]);
  const [blocks, setBlocks] = useState([]);
  const [pendingTransactions, setPendingTransactions] = useState([]);
  const [health, setHealth] = useState(null);
  const [error, setError] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('transactions');

  const refresh = useCallback(async () => {
    setIsLoading(true);
    const [healthResult, chainResult, blocksResult, pendingResult] = await Promise.allSettled([
      transactionChainApi.getHealth(),
      transactionChainApi.getChain(),
      transactionChainApi.getBlocks(),
      transactionChainApi.getPendingTransactions(),
    ]);

    const failures = [healthResult, chainResult, blocksResult, pendingResult]
      .filter((result) => result.status === 'rejected')
      .map((result) => result.reason.message);

    if (healthResult.status === 'fulfilled') setHealth(healthResult.value);
    if (blocksResult.status === 'fulfilled') {
      setBlocks(blocksResult.value);
      setTransactions(transactionsFromBlocks(blocksResult.value));
    } else if (chainResult.status === 'fulfilled') {
      const chainBlocks = chainResult.value.blocks || [];
      setBlocks(chainBlocks);
      setTransactions(transactionsFromBlocks(chainBlocks));
    }
    if (pendingResult.status === 'fulfilled') setPendingTransactions(pendingResult.value);
    setError(failures.length ? failures.join(' • ') : null);
    setIsLoading(false);
  }, []);

  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, 10_000);
    return () => clearInterval(interval);
  }, [refresh]);

  const items = activeTab === 'blocks' ? blocks : activeTab === 'pending' ? pendingTransactions : transactions;

  return (
    <PageLayout activePage={activePage} pageTitle="Transaction Explorer">
      <PageHeader title="Transaction Chain Explorer" subtitle="Live transaction-chain blocks and transaction pool" titleColor="#007bff" />
      {error && <GlassyCard darker style={{ padding: '12px', marginBottom: '16px', color: '#ffb4b4' }}>Unable to load some transaction-chain data: {error}</GlassyCard>}
      <div style={{ display: 'flex', gap: '10px', marginBottom: '16px', flexWrap: 'wrap' }}>
        {['transactions', 'blocks', 'pending'].map((tab) => <button key={tab} onClick={() => setActiveTab(tab)} style={tabStyle(activeTab === tab)}>{tab[0].toUpperCase() + tab.slice(1)}</button>)}
        <button onClick={refresh} disabled={isLoading} style={tabStyle(false)}>{isLoading ? 'Refreshing…' : 'Refresh'}</button>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', gap: '16px', marginBottom: '16px' }}>
        <Metric label="Status" value={health?.healthy === false ? 'Unhealthy' : health?.status || 'Unknown'} />
        <Metric label="Height" value={health?.height ?? blocks.length} />
        <Metric label="Confirmed transactions" value={transactions.length} />
        <Metric label="Pending transactions" value={pendingTransactions.length} />
      </div>
      <GlassyCard darker style={{ padding: '16px' }}>
        <h3 style={{ marginTop: 0 }}>{activeTab[0].toUpperCase() + activeTab.slice(1)} ({items.length})</h3>
        {items.length === 0 ? <div>No {activeTab} available.</div> : items.map((item, index) => activeTab === 'blocks' ? <Block key={item.hash || item.id || index} block={item} /> : <Transaction key={item.hash || item.id || index} transaction={item} />)}
      </GlassyCard>
    </PageLayout>
  );
}

function tabStyle(active) { return { padding: '8px 16px', borderRadius: '6px', border: '1px solid rgba(100,130,255,.3)', background: active ? 'rgba(60,80,170,.7)' : 'rgba(30,40,70,.5)', color: active ? '#fff' : '#b0b0ff', cursor: 'pointer' }; }
function Metric({ label, value }) { return <GlassyCard darker style={{ padding: '16px', textAlign: 'center' }}><div style={{ fontSize: '24px', fontWeight: 'bold' }}>{value}</div><div style={{ fontSize: '12px', marginTop: '4px' }}>{label}</div></GlassyCard>; }
function Block({ block }) { return <div style={cardStyle}><strong>Block #{block.height ?? block.index ?? 'N/A'}</strong><div>Hash: {(block.hash || block.id || 'N/A').slice(0, 18)}</div><div>Transactions: {(block.transactions || block.data || block.txs || []).length}</div><small>{timestamp(block.timestamp)}</small></div>; }
function Transaction({ transaction }) { return <div style={cardStyle}><strong>{transaction.type || transaction.operation || 'Transaction'}</strong><div>Hash: {(transaction.hash || transaction.id || 'N/A').slice(0, 18)}</div><small>{timestamp(transaction.timestamp || transaction.created_at)}</small></div>; }
const cardStyle = { background: 'rgba(30,40,80,.5)', border: '1px solid rgba(100,130,255,.2)', borderRadius: '8px', padding: '10px', marginBottom: '8px' };
