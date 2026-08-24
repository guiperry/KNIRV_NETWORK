import React, { useState } from 'react';
import { Compass, Terminal } from 'lucide-react';

const graphBlocks = [
  { id: 'b0', addr: '0x1a20', label: 'entry\ntest edi, edi\nje 0x1a4c', x: 0 },
  { id: 'b1', addr: '0x1a2c', label: 'cmp eax, 0x20\njl 0x1a9a', x: 1 },
  { id: 'b2', addr: '0x1a9a', label: 'xor eax, eax\nret', x: 2 },
  { id: 'b3', addr: '0x1a4c', label: 'call sym.hmac_sha256\ncmp eax, 1\nje 0x1a2c', x: 1 },
];

const consoleHistory = [
  { cmd: 'aaa', out: '[x] Analyze all flags starting with sym. and entry0\n[x] Analyze function calls\n[x] 214 functions analyzed' },
  { cmd: 'afl~verify', out: '0x00101f10  388  44  sym.verify_license_token' },
  { cmd: 's sym.verify_license_token', out: '' },
  { cmd: 'pdf', out: '            ; CALL XREFS from main @ 0x101a5c\n┌ 388: sym.verify_license_token ();\n│           0x00101f10      55             push rbp\n│           0x00101f11      4889e5         mov rbp, rsp\n│           0x00101f14      4883ec30       sub rsp, 0x30\n│           0x00101f18      48897dd8       mov qword [rbp-0x28], rdi\n│           0x00101f1c      488b45d8       mov rax, qword [rbp-0x28]\n│           0x00101f20      e8abfeffff     call sym.imp.strlen\n└           0x00101f25      83f820         cmp eax, 0x20' },
];

const Cutter: React.FC = () => {
  const [address, setAddress] = useState('sym.verify_license_token');
  const [history, setHistory] = useState(consoleHistory);
  const [cmd, setCmd] = useState('');

  const runCmd = () => {
    if (!cmd.trim()) return;
    setHistory(prev => [...prev, { cmd, out: `[Cutter] no r2 session attached — connect a target to execute '${cmd}'` }]);
    setCmd('');
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-red-500/20 rounded-lg">
          <Compass className="w-6 h-6 text-red-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Cutter</h1>
          <p className="text-slate-400 text-sm font-mono">radare2 GUI · r2 -A target_binary</p>
        </div>
      </div>

      <div className="flex items-center space-x-2 mb-4">
        <span className="text-slate-500 font-mono text-sm">seek:</span>
        <input
          value={address}
          onChange={e => setAddress(e.target.value)}
          className="flex-1 bg-slate-800/50 border border-slate-700/50 rounded-lg px-3 py-2 text-sm font-mono text-slate-200 focus:outline-none focus:border-red-500/50"
        />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-5 gap-4">
        {/* graph view */}
        <div className="xl:col-span-3 bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 overflow-x-auto">
          <div className="text-xs text-slate-600 uppercase mb-3">Graph — {address}</div>
          <div className="flex space-x-8">
            {[0, 1, 2].map(col => (
              <div key={col} className="flex flex-col space-y-6">
                {graphBlocks.filter(b => b.x === col).map(b => (
                  <div key={b.id} className="bg-slate-900/70 border border-red-500/30 rounded px-3 py-2 min-w-[160px]">
                    <div className="text-[10px] text-red-400 font-mono mb-1">{b.addr}</div>
                    <pre className="text-[10px] font-mono text-slate-300 whitespace-pre-wrap">{b.label}</pre>
                  </div>
                ))}
              </div>
            ))}
          </div>
        </div>

        {/* console */}
        <div className="xl:col-span-2 bg-slate-800/50 border border-slate-700/50 rounded-lg flex flex-col">
          <div className="flex items-center space-x-2 px-3 py-2 border-b border-slate-700/50 text-xs text-slate-500">
            <Terminal className="w-3.5 h-3.5" />
            <span>r2 console</span>
          </div>
          <div className="flex-1 overflow-y-auto p-3 space-y-2 font-mono text-[11px]">
            {history.map((h, i) => (
              <div key={i}>
                <div className="text-red-400">[0x00101f10]&gt; {h.cmd}</div>
                <pre className="text-slate-400 whitespace-pre-wrap">{h.out}</pre>
              </div>
            ))}
          </div>
          <div className="flex items-center space-x-2 p-2 border-t border-slate-700/50">
            <span className="text-red-400 font-mono text-xs">&gt;</span>
            <input
              value={cmd}
              onChange={e => setCmd(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && runCmd()}
              placeholder="afl, pdf, iz, aaa..."
              className="flex-1 bg-transparent text-slate-200 font-mono text-xs focus:outline-none"
            />
          </div>
        </div>
      </div>
    </div>
  );
};

export default Cutter;
