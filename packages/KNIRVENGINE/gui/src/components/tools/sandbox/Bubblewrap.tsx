import React, { useState } from 'react';
import { Box, Play, Square } from 'lucide-react';

interface Bind {
  id: number;
  mode: 'ro-bind' | 'bind' | 'tmpfs';
  src: string;
  dst: string;
}

const Bubblewrap: React.FC = () => {
  const [binds, setBinds] = useState<Bind[]>([
    { id: 1, mode: 'ro-bind', src: '/usr', dst: '/usr' },
    { id: 2, mode: 'ro-bind', src: '/lib', dst: '/lib' },
    { id: 3, mode: 'ro-bind', src: '/lib64', dst: '/lib64' },
    { id: 4, mode: 'tmpfs', src: '-', dst: '/tmp' },
  ]);
  const [unshareAll, setUnshareAll] = useState(true);
  const [shareNet, setShareNet] = useState(true);
  const [dieWithParent, setDieWithParent] = useState(true);
  const [display, setDisplay] = useState(':99');
  const [target, setTarget] = useState('/path/to/target_binary');
  const [running, setRunning] = useState(false);
  const [log, setLog] = useState<string[]>([]);

  const command = [
    'bwrap',
    ...binds.flatMap(b => b.mode === 'tmpfs' ? [`--tmpfs ${b.dst}`] : [`--${b.mode} ${b.src} ${b.dst}`]),
    '--proc /proc',
    '--dev /dev',
    unshareAll ? '--unshare-all' : '',
    shareNet ? '--share-net' : '',
    dieWithParent ? '--die-with-parent' : '',
    `--setenv DISPLAY ${display}`,
    '--',
    target,
  ].filter(Boolean).join(' \\\n  ');

  const launch = () => {
    setRunning(true);
    setLog([
      `+ ${command.split('\\\n').join(' ')}`,
      `[bwrap] new user namespace created (uid 0 -> ${1000})`,
      `[bwrap] mount namespace isolated, ${binds.length} binds applied`,
      shareNet ? '[bwrap] network namespace shared with host' : '[bwrap] network namespace isolated (no egress)',
      `[bwrap] child pid 84213 started in namespace, DISPLAY=${display}`,
    ]);
  };

  const addBind = () => setBinds(prev => [...prev, { id: Date.now(), mode: 'ro-bind', src: '', dst: '' }]);
  const removeBind = (id: number) => setBinds(prev => prev.filter(b => b.id !== id));

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-fuchsia-500/20 rounded-lg">
            <Box className="w-6 h-6 text-fuchsia-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">Bubblewrap</h1>
            <p className="text-slate-400 text-sm font-mono">unprivileged namespace sandbox</p>
          </div>
        </div>
        <button
          onClick={launch}
          className="flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium bg-fuchsia-500/20 border border-fuchsia-500/40 text-fuchsia-300 hover:bg-fuchsia-500/30"
        >
          {running ? <Square className="w-4 h-4" /> : <Play className="w-4 h-4" />}
          <span>{running ? 'Relaunch' : 'Launch'}</span>
        </button>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 space-y-4">
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-slate-500 uppercase">Bind mounts</span>
              <button onClick={addBind} className="text-xs text-fuchsia-400 hover:text-fuchsia-300">+ add</button>
            </div>
            <div className="space-y-1.5">
              {binds.map(b => (
                <div key={b.id} className="flex items-center space-x-2 font-mono text-xs">
                  <select
                    value={b.mode}
                    onChange={e => setBinds(prev => prev.map(bb => bb.id === b.id ? { ...bb, mode: e.target.value as Bind['mode'] } : bb))}
                    className="bg-slate-900/60 border border-slate-700/50 rounded px-1.5 py-1 text-slate-300"
                  >
                    <option value="ro-bind">ro-bind</option>
                    <option value="bind">bind</option>
                    <option value="tmpfs">tmpfs</option>
                  </select>
                  {b.mode !== 'tmpfs' && (
                    <input
                      value={b.src}
                      onChange={e => setBinds(prev => prev.map(bb => bb.id === b.id ? { ...bb, src: e.target.value } : bb))}
                      placeholder="host path"
                      className="flex-1 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-slate-300"
                    />
                  )}
                  <input
                    value={b.dst}
                    onChange={e => setBinds(prev => prev.map(bb => bb.id === b.id ? { ...bb, dst: e.target.value } : bb))}
                    placeholder="sandbox path"
                    className="flex-1 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-slate-300"
                  />
                  <button onClick={() => removeBind(b.id)} className="text-slate-600 hover:text-red-400">×</button>
                </div>
              ))}
            </div>
          </div>

          <div className="flex flex-wrap gap-4 text-xs font-mono">
            <label className="flex items-center space-x-2 text-slate-300">
              <input type="checkbox" checked={unshareAll} onChange={e => setUnshareAll(e.target.checked)} className="accent-fuchsia-500" />
              <span>--unshare-all</span>
            </label>
            <label className="flex items-center space-x-2 text-slate-300">
              <input type="checkbox" checked={shareNet} onChange={e => setShareNet(e.target.checked)} className="accent-fuchsia-500" />
              <span>--share-net</span>
            </label>
            <label className="flex items-center space-x-2 text-slate-300">
              <input type="checkbox" checked={dieWithParent} onChange={e => setDieWithParent(e.target.checked)} className="accent-fuchsia-500" />
              <span>--die-with-parent</span>
            </label>
          </div>

          <div className="flex items-center space-x-2 text-xs font-mono">
            <span className="text-slate-500">DISPLAY</span>
            <input value={display} onChange={e => setDisplay(e.target.value)} className="w-16 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-slate-300" />
            <span className="text-slate-500">target</span>
            <input value={target} onChange={e => setTarget(e.target.value)} className="flex-1 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-slate-300" />
          </div>
        </div>

        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 flex flex-col">
          <div className="text-xs text-slate-500 uppercase mb-2">Generated command</div>
          <pre className="text-xs font-mono text-fuchsia-300 bg-slate-900/60 rounded p-3 overflow-x-auto mb-4">{command}</pre>

          <div className="text-xs text-slate-500 uppercase mb-2">Namespace log</div>
          <div className="flex-1 bg-slate-900/60 rounded p-2 font-mono text-xs text-slate-400 overflow-y-auto min-h-[140px]">
            {log.length === 0 ? (
              <span className="text-slate-700">launch to provision the namespace</span>
            ) : (
              log.map((l, i) => <div key={i} className={l.startsWith('+') ? 'text-slate-500' : 'text-green-400'}>{l}</div>)
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Bubblewrap;
