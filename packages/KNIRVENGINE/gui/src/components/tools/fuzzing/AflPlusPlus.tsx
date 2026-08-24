import React, { useState } from 'react';
import { Bug, Play, Pause } from 'lucide-react';

const AflPlusPlus: React.FC = () => {
  const [running, setRunning] = useState(true);
  const [mode, setMode] = useState<'genetic' | 'qemu' | 'frida'>('genetic');
  const [cmplog, setCmplog] = useState(true);

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-orange-500/20 rounded-lg">
            <Bug className="w-6 h-6 text-orange-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">AFL++</h1>
            <p className="text-slate-400 text-sm font-mono">
              afl-fuzz -i corpus -o findings -x dict/manifest.dict {mode === 'qemu' ? '-Q ' : mode === 'frida' ? '-O ' : ''}-- ./target_binary @@
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <div className="flex gap-1">
            {(['genetic', 'qemu', 'frida'] as const).map(m => (
              <button
                key={m}
                onClick={() => setMode(m)}
                className={`px-2.5 py-1.5 rounded text-xs font-mono border ${
                  mode === m ? 'bg-orange-500/20 border-orange-500/40 text-orange-300' : 'border-slate-700/50 text-slate-400'
                }`}
              >
                {m}
              </button>
            ))}
          </div>
          <button
            onClick={() => setRunning(v => !v)}
            className={`flex items-center space-x-2 px-3 py-1.5 rounded-lg text-sm font-medium border ${
              running ? 'bg-orange-500/20 border-orange-500/40 text-orange-300' : 'bg-slate-800/50 border-slate-700/50 text-slate-300'
            }`}
          >
            {running ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
          </button>
        </div>
      </div>

      {/* classic afl-fuzz status screen, reskinned */}
      <div className="bg-black/60 border border-orange-500/30 rounded-lg p-4 font-mono text-xs text-orange-300 mb-4 overflow-x-auto">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-x-8 gap-y-1">
          <div>
            <div className="text-slate-500">process timing</div>
            <div>run time : 3 hrs, 14 min</div>
            <div>last new find : 0 days, 0 hrs, 6 min</div>
            <div>last saved crash : 0 days, 1 hrs, 40 min</div>
          </div>
          <div>
            <div className="text-slate-500">overall results</div>
            <div>cycles done : 14</div>
            <div>corpus count : 3,842</div>
            <div>saved crashes : <span className="text-red-400">5</span></div>
            <div>saved hangs : 1</div>
          </div>
          <div>
            <div className="text-slate-500">cycle progress</div>
            <div>now processing : 2841 (73.9%)</div>
            <div>runs timed out : 0 (0.00%)</div>
          </div>
          <div>
            <div className="text-slate-500">map coverage</div>
            <div>map density : 4.12%</div>
            <div>count coverage : 2.87 bits/tuple</div>
          </div>
          <div>
            <div className="text-slate-500">stage progress</div>
            <div>now trying : {cmplog ? 'cmplog-colorization' : 'havoc'}</div>
            <div>stage execs : 812k/1.2M (67.7%)</div>
            <div>total execs : 214M</div>
          </div>
          <div>
            <div className="text-slate-500">findings in depth</div>
            <div>favored items : 412 (10.7%)</div>
            <div>new edges on : 3,201 (83.3%)</div>
            <div>total crashes : 9 (5 unique)</div>
          </div>
        </div>
        <div className="mt-3 pt-3 border-t border-orange-500/20 flex items-center justify-between">
          <span>exec speed : <span className="text-white">21.8k/sec</span></span>
          <span className={`flex items-center space-x-1 ${running ? 'text-green-400' : 'text-slate-600'}`}>
            <span className="w-2 h-2 rounded-full bg-current"></span>
            <span>{running ? 'fuzzing' : 'paused'}</span>
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="text-xs uppercase text-slate-500 mb-3">Instrumentation</div>
          <div className="space-y-2 text-sm">
            <div className="flex items-center justify-between font-mono text-xs">
              <span className="text-slate-400">AFL_LLVM_INSTRUMENT</span>
              <span className="text-orange-300">CFG</span>
            </div>
            <div className="flex items-center justify-between font-mono text-xs">
              <span className="text-slate-400">persistent mode</span>
              <span className="text-green-400">enabled (LLVMFuzzerTestOneInput)</span>
            </div>
            <label className="flex items-center space-x-2 font-mono text-xs text-slate-300">
              <input type="checkbox" checked={cmplog} onChange={e => setCmplog(e.target.checked)} className="accent-orange-500" />
              <span>CmpLog binary (-c) — input-to-state solving on comparisons</span>
            </label>
          </div>
        </div>

        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="text-xs uppercase text-slate-500 mb-3">Saved crashes</div>
          <table className="w-full text-xs font-mono">
            <thead>
              <tr className="text-slate-600 border-b border-slate-700/50">
                <th className="text-left py-1.5">file</th>
                <th className="text-left py-1.5">sig</th>
                <th className="text-right py-1.5">size</th>
              </tr>
            </thead>
            <tbody>
              {[
                { file: 'id:000000,sig:11,src:000032', sig: 'SIGSEGV', size: '212b' },
                { file: 'id:000001,sig:06,src:000104', sig: 'SIGABRT', size: '64b' },
                { file: 'id:000002,sig:11,src:000201+000032', sig: 'SIGSEGV', size: '4.1kb' },
              ].map(c => (
                <tr key={c.file} className="border-b border-slate-800">
                  <td className="py-1.5 text-slate-300 truncate max-w-[220px]">{c.file}</td>
                  <td className="py-1.5 text-red-400">{c.sig}</td>
                  <td className="py-1.5 text-right text-slate-500">{c.size}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default AflPlusPlus;
