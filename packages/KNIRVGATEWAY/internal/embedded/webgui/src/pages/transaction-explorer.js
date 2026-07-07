import React, { useEffect, useState, useRef, useCallback } from 'react';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import { useNavigation } from '../hooks/useNavigation';

export default function TransactionExplorer() {
  const { activePage } = useNavigation('transaction-explorer');
  const [transactions, setTransactions] = useState([]);
  const [blocks, setBlocks] = useState([]);
  const [consoleLog, setConsoleLog] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('transactions');
  const consoleEndRef = useRef(null);

  const logToConsole = useCallback((message) => {
    const timestamp = new Date().toLocaleTimeString();
    setConsoleLog(prev => [...prev, `[${timestamp}] ${message}`]);
  }, []);

  useEffect(() => {
    if (consoleEndRef.current) {
      consoleEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [consoleLog]);

  useEffect(() => {
    async function fetchData() {
      setIsLoading(true);
      logToConsole('Connecting to Transaction Chain...');
      try {
        const txResp = await fetch('/api/transactions');
        if (txResp.ok) setTransactions(await txResp.json());
        const blockResp = await fetch('/api/blocks');
        if (blockResp.ok) setBlocks(await blockResp.json());
        logToConsole('Transaction Chain data loaded');
      } catch (e) {
        logToConsole(`Error: ${e.message}`);
      } finally {
        setIsLoading(false);
      }
    }
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, [logToConsole]);

  const handleSearch = () => {};

  return (
    <PageLayout activePage={activePage} pageTitle="Transaction Explorer" onSearch={handleSearch}>
      <PageHeader title="Transaction Chain Explorer" subtitle="Fast transaction execution, payment verification, DVE registration" titleColor="#007bff" />

      <div style={{ display: 'flex', gap: '10px', marginBottom: '16px' }}>
        {['transactions', 'blocks', 'console'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab)}
            style={{
              padding: '8px 16px', borderRadius: '6px', border: '1px solid rgba(100,130,255,0.3)',
              background: activeTab === tab ? 'rgba(60,80,170,0.7)' : 'rgba(30,40,70,0.5)',
              color: activeTab === tab ? '#fff' : '#b0b0ff', cursor: 'pointer'
            }}>
            {tab.charAt(0).toUpperCase() + tab.slice(1)}
          </button>
        ))}
      </div>

      <div style={{ display: 'flex', gap: '16px', flex: 1 }}>
        <div style={{ flex: 1, maxWidth: '400px', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          <GlassyCard style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
            <div style={{ padding: '12px', borderBottom: '1px solid rgba(100,130,255,0.2)', display: 'flex', justifyContent: 'space-between' }}>
              <h4 style={{ margin: 0 }}>
                {activeTab === 'transactions' ? `Transactions (${transactions.length})` :
                 activeTab === 'blocks' ? `Blocks (${blocks.length})` : 'Chain Log'}
              </h4>
              {activeTab === 'console' && (
                <button onClick={() => setConsoleLog([])} style={{ background: 'rgba(60,80,170,0.3)', border: '1px solid rgba(100,130,255,0.3)', color: '#b0b0ff', borderRadius: '4px', padding: '2px 8px', cursor: 'pointer', fontSize: '12px' }}>Clear</button>
              )}
            </div>
            <div style={{ flex: 1, overflowY: 'auto', padding: '8px' }}>
              {activeTab === 'transactions' && (transactions.length > 0 ? transactions.map(tx => (
                <div key={tx.id} style={{ background: 'rgba(30,40,80,0.5)', border: '1px solid rgba(100,130,255,0.2)', borderRadius: '8px', padding: '10px', marginBottom: '8px' }}>
                  <div style={{ fontFamily: 'monospace', fontSize: '13px', color: '#a0a0ff' }}>{tx.id?.substring(0, 12)}...</div>
                  <div style={{ fontWeight: 500 }}>{tx.operation || 'Transfer'}</div>
                  <div style={{ fontSize: '12px', color: '#8080c0' }}>{new Date(tx.timestamp).toLocaleString()}</div>
                </div>
              )) : <div style={{ color: '#6060a0', textAlign: 'center', padding: '40px' }}>No transactions</div>)}
              {activeTab === 'blocks' && (blocks.length > 0 ? blocks.map(block => (
                <div key={block.id} style={{ background: 'rgba(30,40,80,0.5)', border: '1px solid rgba(100,130,255,0.2)', borderRadius: '8px', padding: '10px', marginBottom: '8px' }}>
                  <div style={{ fontWeight: 500 }}>Block #{block.index || block.height}</div>
                  <div style={{ fontFamily: 'monospace', fontSize: '12px', color: '#8080c0' }}>Hash: {block.hash?.substring(0, 10)}...</div>
                  <div style={{ fontSize: '11px', color: '#7070a0' }}>{new Date(block.timestamp * 1000).toLocaleString()}</div>
                </div>
              )) : <div style={{ color: '#6060a0', textAlign: 'center', padding: '40px' }}>No blocks</div>)}
              {activeTab === 'console' && (
                <div style={{ fontFamily: 'monospace', fontSize: '13px' }}>
                  {consoleLog.map((log, i) => <div key={i} style={{ padding: '3px 0', borderBottom: '1px solid rgba(100,130,255,0.1)', color: '#a0a0d0' }}>{log}</div>)}
                  <div ref={consoleEndRef} />
                </div>
              )}
            </div>
          </GlassyCard>
        </div>
        <div style={{ flex: 2 }}>
          <GlassyCard style={{ padding: '20px', minHeight: '300px' }}>
            <h3 style={{ margin: '0 0 16px 0' }}>
              {activeTab === 'transactions' ? 'Transaction Details' :
               activeTab === 'blocks' ? 'Block Details' : 'Chain Console'}
            </h3>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
              <GlassyCard style={{ padding: '16px', textAlign: 'center' }}>
                <div style={{ fontSize: '28px', fontWeight: 'bold', color: '#64b5f6' }}>{transactions.length}</div>
                <div style={{ fontSize: '12px', color: '#8080c0', marginTop: '4px' }}>Total Transactions</div>
              </GlassyCard>
              <GlassyCard style={{ padding: '16px', textAlign: 'center' }}>
                <div style={{ fontSize: '28px', fontWeight: 'bold', color: '#81c784' }}>{blocks.length}</div>
                <div style={{ fontSize: '12px', color: '#8080c0', marginTop: '4px' }}>Blocks Mined</div>
              </GlassyCard>
              <GlassyCard style={{ padding: '16px', textAlign: 'center' }}>
                <div style={{ fontSize: '28px', fontWeight: 'bold', color: '#ffb74d' }}>⚡</div>
                <div style={{ fontSize: '12px', color: '#8080c0', marginTop: '4px' }}>Fast Execution</div>
              </GlassyCard>
              <GlassyCard style={{ padding: '16px', textAlign: 'center' }}>
                <div style={{ fontSize: '28px', fontWeight: 'bold', color: '#ba68c8' }}>💳</div>
                <div style={{ fontSize: '12px', color: '#8080c0', marginTop: '4px' }}>Payment Verification</div>
              </GlassyCard>
            </div>
          </GlassyCard>
        </div>
      </div>
    </PageLayout>
  );
}
