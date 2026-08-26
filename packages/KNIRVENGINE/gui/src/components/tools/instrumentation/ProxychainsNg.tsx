import React, { useState } from 'react';
import { Waypoints, Play, Plus, Trash2 } from 'lucide-react';
import { useSandboxSession } from '../../../hooks/useSandboxSession';
import { configureProxychains } from '../../../services/sandboxToolService';

type ChainType = 'dynamic_chain' | 'strict_chain' | 'random_chain';

interface ProxyEntry {
  id: number;
  type: 'socks5' | 'socks4' | 'http';
  host: string;
  port: number;
}

const ProxychainsNg: React.FC = () => {
	const { session, isReady } = useSandboxSession();
  const [chainType, setChainType] = useState<ChainType>('dynamic_chain');
  const [quietMode, setQuietMode] = useState(true);
  const [proxyDns, setProxyDns] = useState(true);
  const [proxies, setProxies] = useState<ProxyEntry[]>([]);
  const [command, setCommand] = useState('');
  const [output, setOutput] = useState<string[]>([]);
  const [running, setRunning] = useState(false);

  const addProxy = () => {
    setProxies(prev => [...prev, { id: Date.now(), type: 'socks5', host: '', port: 1080 }]);
  };

  const removeProxy = (id: number) => setProxies(prev => prev.filter(p => p.id !== id));

	const runCommand = async () => {
		if (!session) return;
		setRunning(true);
		try {
			const result = await configureProxychains(session.id, {
				chainType: chainType.replace('_chain', ''),
				quietMode,
				dnsServers: proxyDns ? ['8.8.8.8'] : [],
				proxyList: proxies.filter(proxy => proxy.host && proxy.port > 0),
			});
			setOutput([
				`[proxychains] configuration written: ${result.configPath}`,
				`[proxychains] ${proxies.length} proxy endpoint(s) configured`,
				'[proxychains] restart Bubble Wrap to launch the target through this chain.',
			]);
		} catch (error) {
			setOutput([`[proxychains] ${error instanceof Error ? error.message : String(error)}`]);
		} finally {
			setRunning(false);
		}
	};

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-blue-500/20 rounded-lg">
          <Waypoints className="w-6 h-6 text-blue-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">proxychains-ng</h1>
          <p className="text-slate-400 text-sm font-mono">LD_PRELOAD=libproxychains4.so · config: proxychains.conf</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* config */}
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 space-y-4">
          <div>
            <div className="text-xs uppercase text-slate-500 mb-2">Chain type</div>
            <div className="flex flex-wrap gap-2">
              {(['dynamic_chain', 'strict_chain', 'random_chain'] as ChainType[]).map(t => (
                <button
                  key={t}
                  onClick={() => setChainType(t)}
                  className={`px-3 py-1.5 rounded text-xs font-mono border transition-colors ${
                    chainType === t
                      ? 'bg-blue-500/20 border-blue-500/40 text-blue-300'
                      : 'bg-slate-900/40 border-slate-700/50 text-slate-400 hover:text-white'
                  }`}
                >
                  {t}
                </button>
              ))}
            </div>
          </div>

          <div className="flex items-center space-x-6 text-sm">
            <label className="flex items-center space-x-2 text-slate-300">
              <input type="checkbox" checked={quietMode} onChange={e => setQuietMode(e.target.checked)} className="accent-blue-500" />
              <span className="font-mono text-xs">quiet_mode</span>
            </label>
            <label className="flex items-center space-x-2 text-slate-300">
              <input type="checkbox" checked={proxyDns} onChange={e => setProxyDns(e.target.checked)} className="accent-blue-500" />
              <span className="font-mono text-xs">proxy_dns</span>
            </label>
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="text-xs uppercase text-slate-500">[ProxyList]</div>
              <button onClick={addProxy} className="flex items-center space-x-1 text-xs text-blue-400 hover:text-blue-300">
                <Plus className="w-3.5 h-3.5" />
                <span>add</span>
              </button>
            </div>
            <div className="space-y-1.5">
              {proxies.map(p => (
                <div key={p.id} className="flex items-center space-x-2 font-mono text-xs">
                  <select
                    value={p.type}
                    onChange={e => setProxies(prev => prev.map(pr => pr.id === p.id ? { ...pr, type: e.target.value as ProxyEntry['type'] } : pr))}
                    className="bg-slate-900/60 border border-slate-700/50 rounded px-1.5 py-1 text-slate-300"
                  >
                    <option value="socks5">socks5</option>
                    <option value="socks4">socks4</option>
                    <option value="http">http</option>
                  </select>
                  <input
                    value={p.host}
                    onChange={e => setProxies(prev => prev.map(pr => pr.id === p.id ? { ...pr, host: e.target.value } : pr))}
                    placeholder="host"
                    className="flex-1 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-slate-300"
                  />
                  <input
                    value={p.port}
                    onChange={e => setProxies(prev => prev.map(pr => pr.id === p.id ? { ...pr, port: Number(e.target.value) || 0 } : pr))}
                    placeholder="port"
                    className="w-16 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-slate-300"
                  />
                  <button onClick={() => removeProxy(p.id)} className="text-slate-600 hover:text-red-400">
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* launcher */}
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 flex flex-col">
		  <div className="text-xs uppercase text-slate-500 mb-2">Configure next launch</div>
          <div className="flex items-center space-x-2 mb-3">
            <span className="text-slate-500 font-mono text-xs">proxychains4 -q</span>
            <input
              value={command}
              onChange={e => setCommand(e.target.value)}
              className="flex-1 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1.5 text-xs font-mono text-slate-200"
            />
            <button
              onClick={runCommand}
			  disabled={running || !isReady}
              className="flex items-center space-x-1 px-2.5 py-1.5 rounded bg-blue-500/20 text-blue-300 hover:bg-blue-500/30 text-xs disabled:opacity-50"
            >
              <Play className="w-3.5 h-3.5" />
			  <span>apply</span>
            </button>
          </div>
          <div className="flex-1 bg-slate-900/60 rounded p-2 font-mono text-xs text-slate-400 overflow-y-auto min-h-[220px]">
            {output.length === 0 ? (
              <span className="text-slate-700">output will appear here</span>
            ) : (
              output.map((line, i) => (
                <div key={i} className={line.includes('OK') ? 'text-green-400' : 'text-slate-400'}>{line}</div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default ProxychainsNg;
