import React, { useState } from 'react';
import { KeySquare, Play, CheckCircle2, HelpCircle } from 'lucide-react';

interface Secret {
  id: string;
  detector: string;
  verified: boolean;
  file: string;
  line: number;
  commit: string;
  redacted: string;
}

const secrets: Secret[] = [
  { id: 's1', detector: 'AWS', verified: true, file: 'scripts/deploy.sh', line: 7, commit: 'a13f9c2', redacted: 'AKIA****************WXYZ' },
  { id: 's2', detector: 'GitHub', verified: true, file: '.github/workflows/release.yml', line: 22, commit: '9e480917f', redacted: 'ghp_****************************abcd' },
  { id: 's3', detector: 'Slack', verified: true, file: 'docs/notes/incident-report.md', line: 4, commit: 'df39cf6', redacted: 'xoxb-****-****-****abcd' },
  { id: 's4', detector: 'Generic High Entropy', verified: false, file: 'packages/KNIRVGATEWAY/.env.testnet', line: 11, commit: 'b0261714e', redacted: 'kQ2f9zR8...aH3v (base64, 44 chars)' },
  { id: 's5', detector: 'PrivateKey', verified: false, file: 'packages/KNIRVSERVER/bin/root.key', line: 1, commit: '2fe865c4c', redacted: '-----BEGIN ENCRYPTED PRIVATE KEY-----' },
];

const TruffleHog: React.FC = () => {
  const [target, setTarget] = useState<'filesystem' | 'git' | 'github'> ('git');
  const [scanning, setScanning] = useState(false);
  const [onlyVerified, setOnlyVerified] = useState(false);
  const [selected, setSelected] = useState<Secret | null>(secrets[0]);

  const runScan = () => {
    setScanning(true);
    setTimeout(() => setScanning(false), 900);
  };

  const targetArg: Record<typeof target, string> = {
    filesystem: 'filesystem . --no-update',
    git: 'git file://. --since-commit df39cf6',
    github: 'github --org=knirvcorp --token=$GITHUB_TOKEN',
  };

  const visible = secrets.filter(s => !onlyVerified || s.verified);

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-amber-500/20 rounded-lg">
            <KeySquare className="w-6 h-6 text-amber-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">TruffleHog</h1>
            <p className="text-slate-400 text-sm font-mono">trufflehog {targetArg[target]} --json</p>
          </div>
        </div>
        <button
          onClick={runScan}
          disabled={scanning}
          className="flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium bg-amber-500/20 border border-amber-500/40 text-amber-300 hover:bg-amber-500/30 disabled:opacity-50"
        >
          <Play className="w-4 h-4" />
          <span>{scanning ? 'Scanning…' : 'Run scan'}</span>
        </button>
      </div>

      <div className="flex items-center justify-between mb-4">
        <div className="flex gap-2">
          {(['filesystem', 'git', 'github'] as const).map(t => (
            <button
              key={t}
              onClick={() => setTarget(t)}
              className={`px-3 py-1.5 rounded-lg text-xs font-mono border ${
                target === t ? 'bg-slate-700/50 border-slate-600 text-white' : 'border-slate-700/50 text-slate-500'
              }`}
            >
              {t}
            </button>
          ))}
        </div>
        <label className="flex items-center space-x-2 text-xs text-slate-300 font-mono">
          <input type="checkbox" checked={onlyVerified} onChange={e => setOnlyVerified(e.target.checked)} className="accent-amber-500" />
          <span>--only-verified</span>
        </label>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="xl:col-span-2 bg-slate-800/50 border border-slate-700/50 rounded-lg divide-y divide-slate-700/50">
          {visible.map(s => (
            <button
              key={s.id}
              onClick={() => setSelected(s)}
              className={`w-full text-left p-3 hover:bg-slate-700/30 transition-colors flex items-center justify-between ${selected?.id === s.id ? 'bg-amber-500/5' : ''}`}
            >
              <div>
                <div className="flex items-center space-x-2 mb-1">
                  {s.verified ? (
                    <span className="flex items-center space-x-1 text-[10px] px-1.5 py-0.5 rounded bg-red-500/20 text-red-300 border border-red-500/30">
                      <CheckCircle2 className="w-3 h-3" />
                      <span>VERIFIED</span>
                    </span>
                  ) : (
                    <span className="flex items-center space-x-1 text-[10px] px-1.5 py-0.5 rounded bg-slate-700/50 text-slate-400 border border-slate-600">
                      <HelpCircle className="w-3 h-3" />
                      <span>unverified</span>
                    </span>
                  )}
                  <span className="text-xs font-mono text-slate-500">{s.detector}</span>
                </div>
                <div className="text-sm font-mono text-slate-200">{s.redacted}</div>
                <div className="text-xs font-mono text-slate-600 mt-1">{s.file}:{s.line} @ {s.commit}</div>
              </div>
            </button>
          ))}
        </div>

        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          {selected ? (
            <div className="space-y-3">
              <div className="flex items-center space-x-2">
                {selected.verified ? (
                  <span className="text-xs px-2 py-1 rounded bg-red-500/20 text-red-300 border border-red-500/30">verified live credential</span>
                ) : (
                  <span className="text-xs px-2 py-1 rounded bg-slate-700/50 text-slate-400 border border-slate-600">could not verify</span>
                )}
              </div>
              <div>
                <div className="text-xs text-slate-600 uppercase mb-1">Detector</div>
                <div className="text-sm text-slate-200">{selected.detector}</div>
              </div>
              <div>
                <div className="text-xs text-slate-600 uppercase mb-1">Location</div>
                <div className="text-xs font-mono text-slate-400">{selected.file}:{selected.line}</div>
                <div className="text-xs font-mono text-slate-600">commit {selected.commit}</div>
              </div>
              <div>
                <div className="text-xs text-slate-600 uppercase mb-1">Redacted secret</div>
                <pre className="text-xs font-mono bg-slate-900/60 rounded p-2 text-amber-300">{selected.redacted}</pre>
              </div>
              {selected.verified && (
                <div className="text-xs text-red-300 bg-red-500/10 border border-red-500/20 rounded px-2 py-1.5">
                  Rotate this credential immediately — it responded successfully to a live API check.
                </div>
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

export default TruffleHog;
