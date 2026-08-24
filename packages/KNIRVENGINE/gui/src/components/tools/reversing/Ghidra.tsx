import React, { useState } from 'react';
import { Binary, Search } from 'lucide-react';

interface Fn {
  name: string;
  addr: string;
  size: number;
  xrefs: number;
}

const functions: Fn[] = [
  { name: 'main', addr: '0x00101a20', size: 214, xrefs: 1 },
  { name: 'verify_license_token', addr: '0x00101f10', size: 388, xrefs: 4 },
  { name: 'decrypt_config_blob', addr: '0x001020c4', size: 156, xrefs: 7 },
  { name: 'setup_tls_context', addr: '0x00102300', size: 92, xrefs: 3 },
  { name: 'FUN_00102ab8', addr: '0x00102ab8', size: 48, xrefs: 12 },
  { name: 'check_debugger_present', addr: '0x00102c40', size: 64, xrefs: 2 },
];

const decompiled: Record<string, string> = {
  verify_license_token: `undefined8 verify_license_token(char *param_1)

{
  int iVar1;
  undefined8 local_res[4];

  iVar1 = strlen(param_1);
  if (iVar1 < 0x20) {
    return 0;
  }
  hmac_sha256(g_license_secret,0x20,param_1,iVar1,local_res);
  iVar1 = memcmp(local_res,param_1 + iVar1 - 0x20,0x20);
  return CONCAT71((int7)((ulong)local_res >> 8), iVar1 == 0);
}`,
  decrypt_config_blob: `void decrypt_config_blob(undefined1 *param_1,int param_2,undefined1 *param_3)

{
  AES_KEY local_1a8;

  AES_set_decrypt_key(g_config_key,0x100,&local_1a8);
  AES_cbc_encrypt(param_1,param_3,param_2,&local_1a8,g_config_iv,0);
  return;
}`,
};

const listing: Record<string, string[]> = {
  verify_license_token: [
    '00101f10  push  rbp',
    '00101f11  mov   rbp, rsp',
    '00101f14  sub   rsp, 0x30',
    '00101f18  mov   [rbp-0x28], rdi',
    '00101f1c  mov   rax, [rbp-0x28]',
    '00101f20  call  strlen',
    '00101f25  cmp   eax, 0x20',
    '00101f28  jl    0x00101f9a',
    '00101f2e  lea   rdx, [g_license_secret]',
  ],
};

const Ghidra: React.FC = () => {
  const [selected, setSelected] = useState('verify_license_token');
  const [query, setQuery] = useState('');

  const filtered = functions.filter(f => f.name.toLowerCase().includes(query.toLowerCase()));

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-green-500/20 rounded-lg">
          <Binary className="w-6 h-6 text-green-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Ghidra</h1>
          <p className="text-slate-400 text-sm font-mono">analyzeHeadless target.gpr TargetProject -import target_binary -analyze</p>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-4 gap-4" style={{ minHeight: 480 }}>
        {/* symbol tree */}
        <div className="xl:col-span-1 bg-slate-800/50 border border-slate-700/50 rounded-lg p-3 flex flex-col">
          <div className="flex items-center space-x-2 mb-2">
            <Search className="w-3.5 h-3.5 text-slate-500" />
            <input
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder="filter symbols..."
              className="flex-1 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-xs font-mono text-slate-200"
            />
          </div>
          <div className="text-xs text-slate-600 uppercase mb-1 px-1">Functions ({filtered.length})</div>
          <div className="flex-1 overflow-y-auto space-y-0.5">
            {filtered.map(f => (
              <button
                key={f.addr}
                onClick={() => setSelected(f.name)}
                className={`w-full text-left px-2 py-1.5 rounded font-mono text-xs transition-colors ${
                  selected === f.name ? 'bg-green-500/15 text-green-300' : 'text-slate-400 hover:bg-slate-700/30'
                }`}
              >
                <div className="truncate">{f.name}</div>
                <div className="text-slate-600 text-[10px]">{f.addr} · {f.size}b · {f.xrefs} xref{f.xrefs !== 1 ? 's' : ''}</div>
              </button>
            ))}
          </div>
        </div>

        {/* listing */}
        <div className="xl:col-span-1 bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
          <div className="text-xs text-slate-600 uppercase mb-2">Listing</div>
          <div className="font-mono text-[11px] text-slate-400 space-y-0.5">
            {(listing[selected] ?? ['-- no disassembly cached for this symbol --']).map((l, i) => (
              <div key={i} className="hover:bg-slate-700/20 px-1 rounded">
                <span className="text-slate-700">{l.slice(0, 8)}</span>
                <span className="text-slate-300">{l.slice(8)}</span>
              </div>
            ))}
          </div>
        </div>

        {/* decompile */}
        <div className="xl:col-span-2 bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
          <div className="text-xs text-slate-600 uppercase mb-2">Decompile — {selected}</div>
          <pre className="font-mono text-xs text-slate-200 whitespace-pre-wrap leading-relaxed">
            {decompiled[selected] ?? '// select a function with cached decompilation\n// (headless re-analysis required for others)'}
          </pre>
        </div>
      </div>
    </div>
  );
};

export default Ghidra;
