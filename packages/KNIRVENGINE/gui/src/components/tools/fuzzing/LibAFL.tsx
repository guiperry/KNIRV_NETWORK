import React, { useState } from 'react';
import { Boxes, Play, Pause } from 'lucide-react';

const crashes = [
  { id: 'id:000012', signal: 'SIGSEGV', input: '0x00000041 x 4096', stage: 'havoc', time: '00:04:12' },
  { id: 'id:000031', signal: 'SIGABRT', input: 'malformed_cbor_header', stage: 'token_mutator', time: '00:11:47' },
];

const stats = {
  corpus: 1842,
  crashes: 2,
  execPerSec: '18.4k',
  coverage: '61.2%',
  cycles: 7,
  uptime: '00:22:16',
};

const LibAFL: React.FC = () => {
  const [running, setRunning] = useState(true);
  const [executor, setExecutor] = useState<'in-process' | 'forkserver' | 'qemu'>('in-process');
  const [stages, setStages] = useState({
    havoc: true,
    tokenMutator: true,
    i2s: true,
    grimoire: false,
  });

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-yellow-500/20 rounded-lg">
            <Boxes className="w-6 h-6 text-yellow-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">LibAFL</h1>
            <p className="text-slate-400 text-sm font-mono">fuzzer_main · target: parse_agent_manifest() · corpus: ./corpus</p>
          </div>
        </div>
        <button
          onClick={() => setRunning(v => !v)}
          className={`flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium border ${
            running ? 'bg-yellow-500/20 border-yellow-500/40 text-yellow-300' : 'bg-slate-800/50 border-slate-700/50 text-slate-300'
          }`}
        >
          {running ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
          <span>{running ? 'Running' : 'Paused'}</span>
        </button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-6 gap-4 mb-6">
        {Object.entries({
          'corpus': stats.corpus,
          'crashes': stats.crashes,
          'exec/s': stats.execPerSec,
          'coverage': stats.coverage,
          'cycles': stats.cycles,
          'uptime': stats.uptime,
        }).map(([label, value]) => (
          <div key={label} className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 text-center">
            <div className={`text-xl font-bold ${label === 'crashes' ? 'text-red-400' : 'text-yellow-400'}`}>{value}</div>
            <div className="text-xs text-slate-500 mt-1 font-mono">{label}</div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="text-xs uppercase text-slate-500 mb-3">Harness</div>
          <div className="space-y-3">
            <div>
              <div className="text-xs text-slate-500 mb-1">Executor</div>
              <div className="flex gap-2">
                {(['in-process', 'forkserver', 'qemu'] as const).map(e => (
                  <button
                    key={e}
                    onClick={() => setExecutor(e)}
                    className={`px-2.5 py-1 rounded text-xs font-mono border ${
                      executor === e ? 'bg-yellow-500/20 border-yellow-500/40 text-yellow-300' : 'border-slate-700/50 text-slate-400'
                    }`}
                  >
                    {e}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <div className="text-xs text-slate-500 mb-1">Mutation stages</div>
              <div className="space-y-1">
                {Object.entries(stages).map(([k, v]) => (
                  <label key={k} className="flex items-center space-x-2 text-xs font-mono text-slate-300">
                    <input
                      type="checkbox"
                      checked={v}
                      onChange={e => setStages(prev => ({ ...prev, [k]: e.target.checked }))}
                      className="accent-yellow-500"
                    />
                    <span>{k}</span>
                  </label>
                ))}
              </div>
            </div>
            <pre className="text-[10px] font-mono text-slate-500 bg-slate-900/60 rounded p-2 mt-2 overflow-x-auto">
{`let mut mgr = SimpleEventManager::new(monitor);
let mutator = StdScheduledMutator::new(havoc_mutations());
let mut fuzzer = StdFuzzer::new(scheduler, feedback, objective);
fuzzer.fuzz_loop(&mut stages, &mut executor, &mut state, &mut mgr)?;`}
            </pre>
          </div>
        </div>

        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="text-xs uppercase text-slate-500 mb-3">Crashing inputs</div>
          <table className="w-full text-xs font-mono">
            <thead>
              <tr className="text-slate-600 border-b border-slate-700/50">
                <th className="text-left py-1.5">id</th>
                <th className="text-left py-1.5">signal</th>
                <th className="text-left py-1.5">stage</th>
                <th className="text-right py-1.5">found</th>
              </tr>
            </thead>
            <tbody>
              {crashes.map(c => (
                <tr key={c.id} className="border-b border-slate-800">
                  <td className="py-1.5 text-slate-300">{c.id}</td>
                  <td className="py-1.5 text-red-400">{c.signal}</td>
                  <td className="py-1.5 text-slate-500">{c.stage}</td>
                  <td className="py-1.5 text-right text-slate-500">{c.time}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default LibAFL;
