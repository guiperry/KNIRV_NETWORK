import React, { useEffect, useState, useRef, useCallback } from 'react';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import { useNavigation } from '../hooks/useNavigation';

export default function ValidationExplorer() {
  const { activePage } = useNavigation('validation-explorer');
  const [rules, setRules] = useState([]);
  const [resolutions, setResolutions] = useState([]);
  const [validations, setValidations] = useState([]);
  const [policies, setPolicies] = useState([]);
  const [consoleLog, setConsoleLog] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('validations');
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
      logToConsole('Connecting to Validation Chain...');
      try {
        const [skillsResp, errorsResp] = await Promise.all([
          fetch('/api/chain/nrv/skills'),
          fetch('/api/chain/nrv/errors'),
        ]);
        if (skillsResp.ok) {
          const skills = await skillsResp.json();
          setValidations(skills.filter(s => s.validation?.is_validated));
          setPolicies(skills.filter(s => s.validation));
        }
        if (errorsResp.ok) {
          const errors = await errorsResp.json();
          setRules(errors.filter(e => e.error_type?.toLowerCase().includes('rule') || e.severity >= 3));
          setResolutions(errors.filter(e => e.resolution_status === 'resolved' || e.resolved_by?.length > 0));
        }
        logToConsole('Validation Chain data loaded');
      } catch (e) {
        logToConsole(`Error: ${e.message}`);
      } finally {
        setIsLoading(false);
      }
    }
    fetchData();
    const interval = setInterval(fetchData, 15000);
    return () => clearInterval(interval);
  }, [logToConsole]);

  const handleSearch = () => {};

  const tabs = ['validations', 'policies', 'rules', 'resolutions', 'console'];

  return (
    <PageLayout activePage={activePage} pageTitle="Validation Explorer" onSearch={handleSearch}>
      <PageHeader title="Validation Chain Explorer" subtitle="Immutable validation signoff, policy commits, rules, resolutions, evidence anchoring" titleColor="#28a745" />

      <div style={{ display: 'flex', gap: '10px', marginBottom: '16px', flexWrap: 'wrap' }}>
        {tabs.map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab)}
            style={{
              padding: '8px 16px', borderRadius: '6px', border: '1px solid rgba(100,200,130,0.3)',
              background: activeTab === tab ? 'rgba(40,120,70,0.7)' : 'rgba(30,40,70,0.5)',
              color: activeTab === tab ? '#fff' : '#b0ffb0', cursor: 'pointer', fontSize: '14px'
            }}>
            {tab.charAt(0).toUpperCase() + tab.slice(1)}
          </button>
        ))}
      </div>

      <div style={{ display: 'flex', gap: '16px', flex: 1 }}>
        <div style={{ flex: 1, maxWidth: '400px', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          <GlassyCard style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
            <div style={{ padding: '12px', borderBottom: '1px solid rgba(100,200,130,0.2)', display: 'flex', justifyContent: 'space-between' }}>
              <h4 style={{ margin: 0 }}>
                {activeTab === 'validations' && `Validated (${validations.length})`}
                {activeTab === 'policies' && `Policies (${policies.length})`}
                {activeTab === 'rules' && `Rules (${rules.length})`}
                {activeTab === 'resolutions' && `Resolutions (${resolutions.length})`}
                {activeTab === 'console' && 'Audit Log'}
              </h4>
              {activeTab === 'console' && (
                <button onClick={() => setConsoleLog([])} style={{ background: 'rgba(40,120,70,0.3)', border: '1px solid rgba(100,200,130,0.3)', color: '#b0ffb0', borderRadius: '4px', padding: '2px 8px', cursor: 'pointer', fontSize: '12px' }}>Clear</button>
              )}
            </div>
            <div style={{ flex: 1, overflowY: 'auto', padding: '8px' }}>
              {activeTab === 'validations' && (validations.length > 0 ? validations.map(v => renderCard(v, 'skill_type', 'validation', '#81c784')) : emptyState('No validated records'))}
              {activeTab === 'policies' && (policies.length > 0 ? policies.map(p => renderPolicyCard(p)) : emptyState('No policy commits'))}
              {activeTab === 'rules' && (rules.length > 0 ? rules.map(r => renderRuleCard(r)) : emptyState('No rule violations'))}
              {activeTab === 'resolutions' && (resolutions.length > 0 ? resolutions.map(r => renderResolutionCard(r)) : emptyState('No resolutions'))}
              {activeTab === 'console' && (
                <div style={{ fontFamily: 'monospace', fontSize: '13px' }}>
                  {consoleLog.map((log, i) => <div key={i} style={{ padding: '3px 0', borderBottom: '1px solid rgba(100,200,130,0.1)', color: '#a0d0a0' }}>{log}</div>)}
                  <div ref={consoleEndRef} />
                </div>
              )}
            </div>
          </GlassyCard>
        </div>
        <div style={{ flex: 2 }}>
          <GlassyCard style={{ padding: '20px', minHeight: '300px' }}>
            <h3 style={{ margin: '0 0 16px 0' }}>
              {activeTab === 'validations' ? 'Validation Signoff' :
               activeTab === 'policies' ? 'Policy Commit Records' :
               activeTab === 'rules' ? 'Rule Enforcement' :
               activeTab === 'resolutions' ? 'Resolution Tracking' : 'Audit Trail'}
            </h3>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
              <GlassyCard style={{ padding: '16px', textAlign: 'center' }}>
                <div style={{ fontSize: '28px', fontWeight: 'bold', color: '#81c784' }}>{validations.length}</div>
                <div style={{ fontSize: '12px', color: '#80c080', marginTop: '4px' }}>Immutable Signoffs</div>
              </GlassyCard>
              <GlassyCard style={{ padding: '16px', textAlign: 'center' }}>
                <div style={{ fontSize: '28px', fontWeight: 'bold', color: '#4caf50' }}>{policies.length}</div>
                <div style={{ fontSize: '12px', color: '#80c080', marginTop: '4px' }}>Policy Commits</div>
              </GlassyCard>
              <GlassyCard style={{ padding: '16px', textAlign: 'center' }}>
                <div style={{ fontSize: '28px', fontWeight: 'bold', color: '#ffd54f' }}>{rules.length}</div>
                <div style={{ fontSize: '12px', color: '#80c080', marginTop: '4px' }}>Rules Enforced</div>
              </GlassyCard>
              <GlassyCard style={{ padding: '16px', textAlign: 'center' }}>
                <div style={{ fontSize: '28px', fontWeight: 'bold', color: '#ce93d8' }}>{resolutions.length}</div>
                <div style={{ fontSize: '12px', color: '#80c080', marginTop: '4px' }}>Resolutions</div>
              </GlassyCard>
            </div>
          </GlassyCard>
        </div>
      </div>
    </PageLayout>
  );
}

function emptyState(msg) {
  return <div style={{ color: '#406040', textAlign: 'center', padding: '40px' }}>{msg}</div>;
}

function renderCard(item, nameKey, detailKey, color) {
  return (
    <div key={item.id} style={{ background: 'rgba(30,60,70,0.5)', border: '1px solid rgba(100,200,130,0.2)', borderRadius: '8px', padding: '10px', marginBottom: '8px' }}>
      <div style={{ fontWeight: 500, color }}>{item[nameKey]}</div>
      <div style={{ fontFamily: 'monospace', fontSize: '12px', color: '#a0d0a0' }}>{item.id?.substring(0, 12)}...</div>
      {item[detailKey] && <div style={{ fontSize: '12px', color: '#80c080' }}>Score: {item[detailKey]?.validation_score || 'N/A'}</div>}
      <div style={{ fontSize: '11px', color: '#609060' }}>{new Date(item.timestamp).toLocaleString()}</div>
    </div>
  );
}

function renderPolicyCard(p) {
  return (
    <div key={p.id} style={{ background: 'rgba(30,60,70,0.5)', border: '1px solid rgba(100,200,130,0.2)', borderRadius: '8px', padding: '10px', marginBottom: '8px' }}>
      <div style={{ fontWeight: 500, color: '#81c784' }}>{p.skill_type}</div>
      <div style={{ fontSize: '12px', color: '#80c080' }}>Validated by: {p.validation?.validated_by?.length || 0} peers</div>
      <div style={{ fontSize: '11px', color: '#609060' }}>{new Date(p.validation?.last_validated).toLocaleString()}</div>
    </div>
  );
}

function renderRuleCard(r) {
  return (
    <div key={r.id} style={{ background: 'rgba(70,40,40,0.5)', border: '1px solid rgba(200,130,100,0.2)', borderRadius: '8px', padding: '10px', marginBottom: '8px' }}>
      <div style={{ fontWeight: 500, color: '#ffb74d' }}>{r.error_type}</div>
      <div style={{ fontSize: '12px', color: '#c0a080' }}>Severity: {r.severity}</div>
      <div style={{ fontSize: '12px', color: '#c0a080' }}>{r.description?.substring(0, 60)}</div>
      <div style={{ fontSize: '11px', color: '#906040' }}>{new Date(r.timestamp).toLocaleString()}</div>
    </div>
  );
}

function renderResolutionCard(r) {
  return (
    <div key={r.id} style={{ background: 'rgba(50,60,50,0.5)', border: '1px solid rgba(130,200,130,0.2)', borderRadius: '8px', padding: '10px', marginBottom: '8px' }}>
      <div style={{ fontWeight: 500, color: '#ce93d8' }}>{r.error_type}</div>
      <div style={{ fontSize: '12px', color: '#b0c0b0' }}>Status: {r.resolution_status || 'resolved'}</div>
      {r.resolved_by && <div style={{ fontSize: '12px', color: '#b0c0b0' }}>By: {r.resolved_by.join(', ')}</div>}
      <div style={{ fontSize: '11px', color: '#609060' }}>{new Date(r.timestamp).toLocaleString()}</div>
    </div>
  );
}
