import React, { useState } from 'react';
import { Cpu, Play, Square, Smartphone, Terminal } from 'lucide-react';

interface ConsoleLine {
  id: number;
  kind: 'info' | 'hook' | 'error' | 'send';
  text: string;
}

const processes = [
  { pid: 4821, name: 'com.targetapp.android', device: 'Pixel 7 (USB)' },
  { pid: 1187, name: 'target_binary', device: 'local' },
  { pid: 9032, name: 'com.targetapp.ios', device: 'iPhone 13 (usbmux)' },
];

const defaultScript = `Java.perform(function () {
  const TrustManagerImpl = Java.use('com.android.org.conscrypt.TrustManagerImpl');
  TrustManagerImpl.verifyChain.implementation = function (untrustedChain, trustAnchorChain,
      host, clientAuth, ocspData, tlsSctData) {
    send('[TLS-UNPIN] verifyChain called, host=' + host);
    return untrustedChain;
  };

  const SSLContext = Java.use('javax.net.ssl.SSLContext');
  send('[+] SSLContext hooks installed');
});`;

const seedLog: ConsoleLine[] = [
  { id: 1, kind: 'info', text: 'Attached to com.targetapp.android (pid 4821)' },
  { id: 2, kind: 'info', text: 'Spawned agent, script.load()' },
  { id: 3, kind: 'send', text: '[+] SSLContext hooks installed' },
  { id: 4, kind: 'hook', text: '[TLS-UNPIN] verifyChain called, host=api.targetapp.local' },
  { id: 5, kind: 'hook', text: '[TLS-UNPIN] verifyChain called, host=telemetry.targetapp.local' },
];

const lineColor: Record<ConsoleLine['kind'], string> = {
  info: 'text-slate-500',
  hook: 'text-cyan-300',
  error: 'text-red-400',
  send: 'text-green-400',
};

const Frida: React.FC = () => {
  const [attachedPid, setAttachedPid] = useState<number | null>(4821);
  const [script, setScript] = useState(defaultScript);
  const [log, setLog] = useState(seedLog);
  const [running, setRunning] = useState(true);

  const run = () => {
    setRunning(true);
    setLog(prev => [
      ...prev,
      { id: prev.length + 1, kind: 'info', text: `script.load() — ${script.split('\n').length} lines compiled` },
      { id: prev.length + 2, kind: 'send', text: '[+] SSLContext hooks installed' },
    ]);
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-cyan-500/20 rounded-lg">
          <Cpu className="w-6 h-6 text-cyan-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Frida</h1>
          <p className="text-slate-400 text-sm font-mono">frida-core 16.x · gadget: injected · runtime: QJS</p>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-4 gap-4">
        {/* process picker */}
        <div className="xl:col-span-1 bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <h3 className="text-xs uppercase text-slate-500 mb-3 flex items-center space-x-1">
            <Smartphone className="w-3.5 h-3.5" />
            <span>Devices &amp; Processes</span>
          </h3>
          <div className="space-y-1">
            {processes.map(p => (
              <button
                key={p.pid}
                onClick={() => setAttachedPid(p.pid)}
                className={`w-full text-left px-2 py-2 rounded text-sm font-mono transition-colors ${
                  attachedPid === p.pid ? 'bg-cyan-500/15 text-cyan-300 border border-cyan-500/30' : 'text-slate-400 hover:bg-slate-700/30'
                }`}
              >
                <div className="truncate">{p.name}</div>
                <div className="text-xs text-slate-600">pid {p.pid} · {p.device}</div>
              </button>
            ))}
          </div>
        </div>

        {/* script editor */}
        <div className="xl:col-span-3 space-y-4">
          <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg overflow-hidden">
            <div className="flex items-center justify-between px-3 py-2 border-b border-slate-700/50">
              <span className="text-xs text-slate-500 font-mono">agent.js</span>
              <div className="flex items-center space-x-2">
                <button
                  onClick={run}
                  className="flex items-center space-x-1 text-xs px-2 py-1 rounded bg-green-500/20 text-green-300 hover:bg-green-500/30"
                >
                  <Play className="w-3.5 h-3.5" />
                  <span>Load script</span>
                </button>
                <button
                  onClick={() => setRunning(false)}
                  className="flex items-center space-x-1 text-xs px-2 py-1 rounded bg-slate-700/50 text-slate-300 hover:bg-slate-700"
                >
                  <Square className="w-3.5 h-3.5" />
                  <span>Detach</span>
                </button>
              </div>
            </div>
            <textarea
              value={script}
              onChange={e => setScript(e.target.value)}
              spellCheck={false}
              className="w-full h-56 bg-slate-900/60 text-slate-200 font-mono text-xs p-3 focus:outline-none resize-none"
            />
          </div>

          {/* console */}
          <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg">
            <div className="flex items-center space-x-2 px-3 py-2 border-b border-slate-700/50 text-xs text-slate-500">
              <Terminal className="w-3.5 h-3.5" />
              <span>console</span>
              <span className={`ml-auto flex items-center space-x-1 ${running ? 'text-green-400' : 'text-slate-600'}`}>
                <span className="w-1.5 h-1.5 rounded-full bg-current"></span>
                <span>{running ? `attached · pid ${attachedPid}` : 'detached'}</span>
              </span>
            </div>
            <div className="p-3 h-40 overflow-y-auto font-mono text-xs space-y-1">
              {log.map(line => (
                <div key={line.id} className={lineColor[line.kind]}>
                  <span className="text-slate-700 mr-2">[{String(line.id).padStart(3, '0')}]</span>
                  {line.text}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Frida;
