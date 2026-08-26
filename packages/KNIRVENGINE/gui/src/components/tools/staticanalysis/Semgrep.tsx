import React, { useState } from 'react';
import { ScanSearch, Play } from 'lucide-react';
import { useToolScan } from '../../hooks/useToolScan';
import { useSandboxSession } from '../../hooks/useSandboxSession';

type Severity = 'ERROR' | 'WARNING' | 'INFO';

interface Finding {
  id: string;
  ruleId: string;
  severity: Severity;
  file: string;
  line: number;
  message: string;
  hasFix: boolean;
}

const sevColor: Record<Severity, string> = {
  ERROR: 'bg-red-500/20 text-red-300 border-red-500/30',
  WARNING: 'bg-yellow-500/20 text-yellow-300 border-yellow-500/30',
  INFO: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
};

const Semgrep: React.FC = () => {
  const { session } = useSandboxSession();
  const { structured, running, error, run } = useToolScan({
    sessionID: session?.id ?? '',
    tool: 'semgrep',
  });

  const [ruleset, setRuleset] = useState('p/owasp-top-ten p/secrets p/golang');
  const [selected, setSelected] = useState<Finding | null>(null);
  const [severityFilter, setSeverityFilter] = useState<Severity | 'ALL'>('ALL');

  const findings: Finding[] = (structured as Finding[]) ?? [];

  const runScan = () => {
    run({ ruleset });
  };

  const visible = findings.filter(f => severityFilter === 'ALL' || f.severity === severityFilter);
  const counts = { ERROR: 0, WARNING: 0, INFO: 0 } as Record<Severity, number>;
  findings.forEach(f => counts[f.severity]++);

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-emerald-500/20 rounded-lg">
            <ScanSearch className="w-6 h-6 text-emerald-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">Semgrep</h1>
            <p className="text-slate-400 text-sm font-mono">semgrep --config {ruleset} --json .</p>
          </div>
        </div>
        <button
          onClick={runScan}
          disabled={running || !session}
          className="flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 hover:bg-emerald-500/30 disabled:opacity-50"
        >
          <Play className="w-4 h-4" />
          <span>{running ? 'Scanning…' : 'Run scan'}</span>
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-red-300 text-sm">
          {error}
        </div>
      )}

      <div className="flex items-center space-x-2 mb-4">
        <span className="text-slate-500 font-mono text-sm">--config</span>
        <input
          value={ruleset}
          onChange={e => setRuleset(e.target.value)}
          className="flex-1 bg-slate-800/50 border border-slate-700/50 rounded-lg px-3 py-2 text-sm font-mono text-slate-200"
        />
      </div>

      <div className="flex items-center space-x-2 mb-4">
        {(['ALL', 'ERROR', 'WARNING', 'INFO'] as const).map(s => (
          <button
            key={s}
            onClick={() => setSeverityFilter(s)}
            className={`px-3 py-1.5 rounded-lg text-xs font-mono border ${
              severityFilter === s ? 'bg-slate-700/50 border-slate-600 text-white' : 'border-slate-700/50 text-slate-500'
            }`}
          >
            {s === 'ALL' ? `all (${findings.length})` : `${s.toLowerCase()} (${counts[s]})`}
          </button>
        ))}
      </div>

      {!session && (
        <div className="text-center py-8 text-slate-500">
          Start a sandbox session to run Semgrep against a mounted target.
        </div>
      )}

      {session && findings.length === 0 && !running && (
        <div className="text-center py-8 text-slate-500">
          No findings yet. Click "Run scan" to analyze the mounted target.
        </div>
      )}

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="xl:col-span-2 bg-slate-800/50 border border-slate-700/50 rounded-lg divide-y divide-slate-700/50">
          {visible.map(f => (
            <button
              key={f.id}
              onClick={() => setSelected(f)}
              className={`w-full text-left p-3 hover:bg-slate-700/30 transition-colors ${selected?.id === f.id ? 'bg-emerald-500/5' : ''}`}
            >
              <div className="flex items-center space-x-2 mb-1">
                <span className={`text-[10px] px-1.5 py-0.5 rounded border font-mono ${sevColor[f.severity]}`}>{f.severity}</span>
                <span className="text-xs font-mono text-slate-500 truncate">{f.ruleId}</span>
              </div>
              <div className="text-sm text-slate-200">{f.message}</div>
              <div className="text-xs font-mono text-slate-600 mt-1">{f.file}:{f.line}</div>
            </button>
          ))}
        </div>

        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          {selected ? (
            <div className="space-y-3">
              <div>
                <span className={`text-[10px] px-1.5 py-0.5 rounded border font-mono ${sevColor[selected.severity]}`}>{selected.severity}</span>
                <div className="text-xs font-mono text-slate-500 mt-2">{selected.ruleId}</div>
              </div>
              <div className="text-sm text-slate-300">{selected.message}</div>
              <div>
                <div className="text-xs text-slate-600 uppercase mb-1">{selected.file}:{selected.line}</div>
                <pre className="text-xs font-mono bg-slate-900/60 rounded p-2 text-slate-400 overflow-x-auto">
{`${selected.line - 1} │ ...
${selected.line}   │ // flagged expression
${selected.line + 1} │ ...`}
                </pre>
              </div>
              {selected.hasFix ? (
                <button className="w-full text-xs px-2 py-1.5 rounded bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30">
                  Apply autofix
                </button>
              ) : (
                <div className="text-xs text-slate-600">no autofix registered for this rule</div>
              )}
            </div>
          ) : (
            <div className="text-slate-600 text-sm text-center py-12">select a finding</div>
          )}
        </div>
      </div>
    </div>
  );
};

export default Semgrep;
