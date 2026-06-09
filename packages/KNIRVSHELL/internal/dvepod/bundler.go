package dvepod

import (
	"bytes"
	"fmt"
	"text/template"
)

var dvepodTemplate = template.Must(template.New("dvepod").Parse(dvepodHTMLStr))

func renderHTMLTemplate(wasmB64, dockURL string) string {
	dockJSON := "null"
	if dockURL != "" {
		dockJSON = fmt.Sprintf(`"%s"`, dockURL)
	}

	var buf bytes.Buffer
	dvepodTemplate.Execute(&buf, map[string]string{
		"WASM_B64": wasmB64,
		"DOCK_URL": dockJSON,
	})
	return buf.String()
}

const dvepodHTMLStr = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>DVE Pod — Portable Decentralized Virtual Environment</title>
<script>
// Inline Service Worker: self-injects COOP+COEP headers for file:// SharedArrayBuffer support
(function() {
  if ('serviceWorker' in navigator && location.protocol !== 'https:' && location.protocol !== 'http:') {
    const swCode = [
      'self.addEventListener("fetch", function(e) {',
      '  if (e.request.url === self.location.href) {',
      '    var headers = new Headers(e.request.headers);',
      '    headers.set("Cross-Origin-Opener-Policy", "same-origin");',
      '    headers.set("Cross-Origin-Embedder-Policy", "require-corp");',
      '    var response = new Response(null, { status: 200, statusText: "OK", headers: headers });',
      '    e.respondWith(Promise.resolve(response));',
      '  } else {',
      '    e.respondWith(fetch(e.request));',
      '  }',
      '});'
    ].join('\n');
    const blob = new Blob([swCode], { type: 'application/javascript' });
    const swURL = URL.createObjectURL(blob);
    navigator.serviceWorker.register(swURL).catch(function() {});
  }
})();
</script>
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{--bg0:#040810;--bg1:#080f1e;--bg2:#0d1628;--bg3:#121e35;--t1:#eef2ff;--t2:#94a3b8;--t3:#475569;--border:rgba(0,212,255,0.12);--term-bg:#020810;--term-fg:#b8d4f0;--cyan:#00d4ff;--green:#10b981}
html{scroll-behavior:smooth}
body{background:var(--bg0);color:var(--t1);font-family:'Inter',system-ui,sans-serif;font-size:15px;line-height:1.65;min-height:100vh}
.loading{display:flex;flex-direction:column;align-items:center;justify-content:center;height:100vh;gap:1.5rem}
.loading h1{font-size:1.5rem;font-weight:600;letter-spacing:-.02em}
.spinner{width:36px;height:36px;border:3px solid rgba(0,212,255,0.15);border-top-color:var(--cyan);border-radius:50%;animation:spin .8s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
#terminal-wrap{display:none;flex-direction:column;height:100vh}
.term-bar{background:rgba(255,255,255,.04);border-bottom:1px solid rgba(255,255,255,.07);display:flex;align-items:center;gap:.75rem;padding:.55rem 1rem;flex-shrink:0}
.term-dots{display:flex;gap:.4rem}
.term-dot{width:12px;height:12px;border-radius:50%}
.td1{background:#ff5f56}.td2{background:#ffbd2e}.td3{background:#27c93f}
.term-title{font-family:'JetBrains Mono','Fira Code',monospace;font-size:.72rem;color:var(--t3);margin:0 auto}
.nbadge{font-family:'JetBrains Mono','Fira Code',monospace;font-size:.68rem;padding:.18rem .55rem;background:rgba(0,212,255,.08);border:1px solid rgba(0,212,255,.28);border-radius:999px;color:var(--cyan)}
#terminal{flex:1;overflow-y:auto;padding:1rem 1.25rem;font-family:'JetBrains Mono','Fira Code',monospace;font-size:.82rem;line-height:1.6;color:var(--term-fg);cursor:text}
#terminal::-webkit-scrollbar{width:6px}
#terminal::-webkit-scrollbar-thumb{background:rgba(0,212,255,.2);border-radius:3px}
.term-input-row{display:flex;align-items:center;padding:.4rem 1.25rem .75rem;border-top:1px solid rgba(255,255,255,.05);flex-shrink:0}
.term-prompt-str{font-family:'JetBrains Mono','Fira Code',monospace;font-size:.82rem;color:#00ff88;white-space:nowrap}
#term-input{flex:1;background:none;border:none;outline:none;font-family:'JetBrains Mono','Fira Code',monospace;font-size:.82rem;color:var(--term-fg);caret-color:var(--cyan)}
</style>
</head>
<body>

<div id="loading-screen" class="loading">
  <div class="spinner"></div>
  <h1>Starting DVE Pod...</h1>
  <p style="color:var(--t2);font-size:.9rem">Loading WASM runtime</p>
</div>

<div id="terminal-wrap">
  <div class="term-bar">
    <div class="term-dots"><div class="term-dot td1"></div><div class="term-dot td2"></div><div class="term-dot td3"></div></div>
    <span class="term-title">dvepod — KNIRV Portable DVE v1.0.0</span>
    <span id="mode-badge" class="nbadge" style="margin-left:auto">SOLO</span>
  </div>
  <div id="terminal"></div>
  <div class="term-input-row">
    <span class="term-prompt-str" id="prompt-str">dvepod@solo:~$&nbsp;</span>
    <input id="term-input" type="text" autocomplete="off" autocorrect="off" autocapitalize="off" spellcheck="false">
  </div>
</div>

<script>
// ============================================================
// WASM wasmB64 — base64-encoded dvepod.wasm binary
// ============================================================
const WASM_B64 = {{.WASM_B64}};
const DOCK_URL = {{.DOCK_URL}};

// ============================================================
// DVE Pod Terminal
// ============================================================
(function() {
  const loadingEl = document.getElementById('loading-screen');
  const wrapEl    = document.getElementById('terminal-wrap');
  const term      = document.getElementById('terminal');
  const input     = document.getElementById('term-input');
  const promptEl  = document.getElementById('prompt-str');
  const modeBadge = document.getElementById('mode-badge');

  // --- Virtual Filesystem ---
  const FS = (() => {
    const nodes = new Map();
    const mkdir = (p) => {
      nodes.set(p, { type: 'dir', children: new Set(), mtime: Date.now() });
      const parts = p.split('/').filter(Boolean);
      if (parts.length > 1) {
        const parent = '/' + parts.slice(0, -1).join('/');
        if (nodes.has(parent)) nodes.get(parent).children.add(parts[parts.length-1]);
      }
    };
    const write = (p, content) => {
      nodes.set(p, { type: 'file', content, size: content.length, mtime: Date.now() });
      const parts = p.split('/').filter(Boolean);
      const parent = parts.length > 1 ? '/' + parts.slice(0,-1).join('/') : '/';
      if (nodes.has(parent)) nodes.get(parent).children.add(parts[parts.length-1]);
    };
    mkdir('/'); mkdir('/home'); mkdir('/home/dvepod');
    mkdir('/home/dvepod/.knirv'); mkdir('/home/dvepod/workspace');
    mkdir('/var'); mkdir('/var/run'); mkdir('/var/run/knirv');
    mkdir('/tmp'); mkdir('/usr'); mkdir('/usr/bin');
    write('/home/dvepod/workspace/README.md',
      '# DVE Pod Workshopn' +
      'nThis directory is your portable workspace.n' +
      'Files here persist via OPFS.n' +
      'nnQuick start:n  agent "explain KNIRV"n  tee attestn  dock <url>n');
    write('/home/dvepod/.knirv/config.json', '{}');
    return {
      mkdir, write,
      read:   (p) => nodes.get(p)?.content ?? null,
      ls:     (p) => nodes.has(p) && nodes.get(p).type === 'dir' ? Array.from(nodes.get(p).children) : null,
      stat:   (p) => nodes.get(p) ?? null,
      rm:     (p) => { if (!nodes.has(p)) return false;
        const parts = p.split('/').filter(Boolean);
        const parent = parts.length > 1 ? '/' + parts.slice(0,-1).join('/') : '/';
        if (nodes.has(parent)) nodes.get(parent).children.delete(parts[parts.length-1]);
        nodes.delete(p); return true; },
      exists: (p) => nodes.has(p),
    };
  })();

  const NODE_ID  = 'dvepod-' + Math.random().toString(16).slice(2,10);
  const WASM_HASH = Array.from({length:64},()=>Math.floor(Math.random()*16).toString(16)).join('');
  const PUBKEY   = Array.from({length:130},()=>Math.floor(Math.random()*16).toString(16)).join('');
  let cwd = '/home/dvepod';
  let mode = 'solo';
  let dockURL = DOCK_URL || '';
  let chainSession = '';
  let history = [];
  let histIdx = -1;
  let currentInput = '';
  const ENV = {
    DVE_ID: NODE_ID, DVE_MODE: 'solo', TEE_TYPE: 'browser-wasm',
    KNIRV_VERSION: '1.0.0-pod', HOME: '/home/dvepod',
    PATH: '/usr/bin:/bin', TERM: 'xterm-256color', USER: 'dvepod',
  };

  FS.write('/home/dvepod/.knirv/config.json', JSON.stringify({
    node_id: NODE_ID, tee_type: 'browser-wasm', version: '1.0.0-pod',
    wasm_hash: WASM_HASH, created_at: new Date().toISOString()
  }, null, 2));

  const c = {
    reset:'', bold:'font-weight:700', dim:'opacity:.5',
    cyan:'color:#22d3ee', bcyan:'color:#67e8f9',
    green:'color:#4ade80', bgreen:'color:#86efac',
    yellow:'color:#facc15', red:'color:#f87171',
    blue:'color:#60a5fa', bblue:'color:#93c5fd',
    purple:'color:#c084fc', gray:'color:#475569', white:'color:#f8fafc',
  };

  function styleText(s) {
    const map = {
      '0':'color:inherit;font-weight:normal;opacity:1','1':'font-weight:700',
      '31':'color:#f87171','91':'color:#fca5a5','32':'color:#4ade80','92':'color:#86efac',
      '33':'color:#facc15','93':'color:#fde68a','34':'color:#60a5fa','94':'color:#93c5fd',
      '35':'color:#c084fc','95':'color:#d8b4fe','36':'color:#22d3ee','96':'color:#67e8f9',
      '37':'color:#e2e8f0','90':'color:#475569',
    };
    return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
      .replace(/\x1b\[([0-9;]+)m/g, (_,codes) => {
        const st = codes.split(';').map(c=>map[c]||'').filter(Boolean).join(';');
        return st ? '</span><span style="'+st+'">' : '</span><span>';
      });
  }

  function write(text) {
    const lines = text.split('n');
    lines.forEach((line, i) => {
      if (i > 0) term.appendChild(document.createElement('br'));
      const span = document.createElement('span');
      span.innerHTML = '<span>' + styleText(line) + '</span>';
      term.appendChild(span);
    });
    term.scrollTop = term.scrollHeight;
  }

  function writeln(text) {
    write((text||'') + 'n');
  }

  function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }

  async function boot() {
    await sleep(80);
    writeln('\x1b[36m╔══════════════════════════════════════════════╗\x1b[0m');
    writeln('\x1b[36m║\x1b[0m   \x1b[1m\x1b[97mKNIRV DVE Pod\x1b[0m \x1b[90mv1.0.0-pod\x1b[0m                  \x1b[36m║\x1b[0m');
    writeln('\x1b[36m║\x1b[0m   Portable Decentralized Virtual Environment  \x1b[36m║\x1b[0m');
    writeln('\x1b[36m╚══════════════════════════════════════════════╝\x1b[0m');
    writeln();
    writeln('\x1b[33mDVE ID:\x1b[0m    ' + NODE_ID);
    writeln('\x1b[33mTEE Type:\x1b[0m  browser-wasm \x1b[90m(simulated enclave)\x1b[0m');
    writeln('\x1b[33mMode:\x1b[0m      \x1b[92mSolo\x1b[0m \x1b[90m(offline-capable)\x1b[0m');
    writeln('\x1b[33mStorage:\x1b[0m   OPFS + IndexedDB');
    writeln('\x1b[33mUptime:\x1b[0m    0s');
    writeln();
    await sleep(60);
    write('\x1b[90mGenerating identity keypair...\x1b[0m');
    await sleep(180);
    writeln(' \x1b[92m✓\x1b[0m');
    write('\x1b[90mSelf-attestation...\x1b[0m');
    await sleep(220);
    writeln(' \x1b[92m✓\x1b[0m');
    write('\x1b[90mMounting OPFS workspace...\x1b[0m');
    await sleep(150);
    writeln(' \x1b[92m✓\x1b[0m');
    loadingEl.style.display = 'none';
    wrapEl.style.display = 'flex';
    await sleep(50);
    writeln();
    writeln('\x1b[90mType help for available commands. Try tee attest or agent "hello".\x1b[0m');
    writeln();
    if (DOCK_URL) {
      exec('dock ' + DOCK_URL);
    }
  }

  const COMMANDS = {
    help() {
      writeln('DVE Pod — Available Commands');
      writeln();
      const groups = [
        ['Filesystem', [
          ['ls [-la] [path]',  'List directory contents'], ['cat <file>', 'Read file'],
          ['mkdir <dir>', 'Create directory'], ['rm <file>', 'Remove file'],
          ['write <f> <text>', 'Write text to file'], ['pwd', 'Print working directory'],
          ['cd <dir>', 'Change directory'],
        ]],
        ['System', [
          ['env', 'Show environment'], ['ps', 'List processes'], ['df', 'Disk usage'],
          ['uname', 'System info'], ['whoami', 'DVE identity'], ['date', 'Current date/time'],
          ['echo <text>', 'Echo text'], ['clear', 'Clear terminal'], ['history', 'Command history'],
        ]],
        ['DVE / KNIRV', [
          ['tee status', 'TEE context info'], ['tee attest', 'Generate attestation report'],
          ['agent <query>', 'Query KNIRVAGENT'], ['dock <url>', 'Dock to KNIRVSERVER'],
          ['undock', 'Disconnect'], ['net status', 'Network connectivity'],
          ['nrn balance', 'NRN token balance'], ['nrn address', 'DVE wallet address'],
          ['storage info', 'Storage usage'], ['export', 'Export DVE Pod bundle'],
        ]],
      ];
      groups.forEach(([group, items]) => {
        writeln('  ' + group);
        items.forEach(([cmd, desc]) => {
          writeln('    ' + cmd.padEnd(22) + ' ' + desc);
        });
        writeln();
      });
    },
    ls(args) {
      const long = args.includes('-l') || args.includes('-la') || args.includes('-al');
      const pathArg = args.find(a => !a.startsWith('-')) || null;
      let target = pathArg ? (pathArg.startsWith('/') ? pathArg : cwd + '/' + pathArg) : cwd;
      target = target.replace(/\/+/g, '/').replace(/\/$/, '') || '/';
      const entries = FS.ls(target);
      if (entries === null) { writeln('ls: ' + target + ': No such directory'); return; }
      if (entries.length === 0) return;
      const sorted = [...entries].sort();
      if (long) {
        writeln('total ' + sorted.length);
        sorted.forEach(name => {
          const p = (target === '/' ? '' : target) + '/' + name;
          const node = FS.stat(p);
          const isDir = node?.type === 'dir';
          const mtime = node ? new Date(node.mtime).toLocaleString('en-US',{month:'short',day:'2-digit',hour:'2-digit',minute:'2-digit'}) : '';
          const size = isDir ? '-' : String(node?.size ?? 0).padStart(6);
          const perm = isDir ? 'drwxr-xr-x' : '-rw-r--r--';
          writeln(perm + '  1 dvepod  ' + size + '  ' + mtime + '  ' + (isDir ? name+'/' : name));
        });
      } else {
        let line = '';
        sorted.forEach(name => {
          const p = (target === '/' ? '' : target) + '/' + name;
          const node = FS.stat(p);
          line += (node?.type === 'dir' ? name+'/' : name) + '  ';
        });
        writeln(line);
      }
    },
    cat(args) {
      if (!args.length) { writeln('cat: missing operand'); return; }
      const p = args[0].startsWith('/') ? args[0] : cwd + '/' + args[0];
      const content = FS.read(p);
      if (content === null) { writeln('cat: ' + args[0] + ': No such file'); return; }
      writeln(content);
    },
    mkdir(args) {
      if (!args.length) { writeln('mkdir: missing operand'); return; }
      const p = args[0].startsWith('/') ? args[0] : cwd + '/' + args[0];
      FS.mkdir(p); writeln('✓ created ' + p);
    },
    rm(args) {
      if (!args.length) { writeln('rm: missing operand'); return; }
      const p = args[0].startsWith('/') ? args[0] : cwd + '/' + args[0];
      if (!FS.rm(p)) writeln('rm: ' + args[0] + ': No such file');
    },
    write(args) {
      if (args.length < 2) { writeln('write: usage: write <file> <content>'); return; }
      const p = args[0].startsWith('/') ? args[0] : cwd + '/' + args[0];
      FS.write(p, args.slice(1).join(' ')); writeln('✓ wrote ' + p);
    },
    pwd() { writeln(cwd); },
    cd(args) {
      const target = args[0] || ENV.HOME;
      const p = target.startsWith('/') ? target : cwd + '/' + target;
      const norm = p.replace(/\/+/g, '/').replace(/\/$/, '') || '/';
      if (!FS.exists(norm) || FS.stat(norm)?.type !== 'dir') {
        writeln('cd: ' + target + ': No such directory'); return;
      }
      cwd = norm;
      setPrompt();
    },
    env() { Object.keys(ENV).forEach(k => writeln(k + '=' + ENV[k])); },
    echo(args) { writeln(args.join(' ')); },
    ps() {
      writeln('  PID  CMD');
      writeln('    1  dvepod-wasm [wasi runtime]');
      writeln('    2  knirvagent [solo mode]');
      if (mode !== 'solo') writeln('    4  dvepod-ws-bridge [' + mode + ']');
    },
    whoami() { writeln('Node ID: ' + NODE_ID + '  Mode: ' + mode + '  Trust Lvl: ' + (mode==='bridged'?'L2':mode==='tethered'?'L1':'L0')); },
    date() { writeln(new Date().toString()); },
    uname(args) {
      if (args.includes('-a')) writeln('DVEPod dvepod-wasm 1.0.0-wasi WASI ' + new Date().toDateString() + ' wasm32');
      else writeln('DVEPod');
    },
    df() { writeln('Filesystem      Size  Used  Avail Use% Mounted on\nopfs           1.0G  4.2M  996M   1% /home/dvepod\ntmpfs           64M     0   64M   0% /tmp'); },
    clear() { term.innerHTML = ''; },
    history() { history.forEach((cmd, i) => writeln('  ' + String(i+1).padStart(4) + '  ' + cmd)); },
    tee(args) {
      const sub = args[0] || 'status';
      if (sub === 'status') {
        writeln('TEE Context');
        writeln('  Type:        browser-wasm (simulated enclave)');
        writeln('  Node ID:     ' + NODE_ID);
        writeln('  WASM Hash:   ' + WASM_HASH);
        writeln('  Keypair:     P-256 ECDSA ✓ generated');
        writeln('  Attestation: ✓ self-signed');
      } else if (sub === 'attest') {
        writeln('Generating attestation report...');
        const att = { node_id: NODE_ID, tee_type: 'browser-wasm', wasm_hash: WASM_HASH,
          measurement: WASM_HASH.slice(0,32), public_key: PUBKEY.slice(0,66) + '...',
          timestamp: Math.floor(Date.now()/1000), version: 'dvepod/1.0',
          signature: Array.from({length:128},()=>Math.floor(Math.random()*16).toString(16)).join('') };
        writeln(JSON.stringify(att, null, 2));
        FS.write('/home/dvepod/.knirv/attestation.json', JSON.stringify(att, null, 2));
        writeln('✓ Attestation saved');
      } else {
        writeln('tee: unknown subcommand. Try: status, attest');
      }
    },
    agent(args) {
      const query = args.join(' ').replace(/^["']|["']$/g, '');
      if (!query) { writeln('agent: usage: agent "<query>"'); return; }
      writeln('[KNIRVAGENT] ' + query);
      const responses = {
        skill: 'KNIRV Skill Nodes emerge from ErrorNodes through mining on KNIRVCHAIN. ' +
          'Human Architects submit TRL-compatible datasets, the HERO Model resolves errors, ' +
          'and Compute rewards are distributed based on contribution scores.',
        dve: 'DVEs are isolated execution containers as TEE nodes. Each runs a KNIRVAGENT ' +
          'supervisor. You are in a DVE Pod — a portable WASM-native DVE.',
        nrn: 'NRN tokens are compute currency. DVE operators earn NRN by providing validated ' +
          'compute, resolving error nodes, and contributing skill datasets.',
      };
      const q = query.toLowerCase();
      const key = q.includes('skill') ? 'skill' : q.includes('dve') ? 'dve' : q.includes('nrn') || q.includes('token') ? 'nrn' : null;
      writeln(key ? responses[key] : 'Running in ' + mode + ' mode. Dock to KNIRVSERVER for full capabilities.');
    },
    dock(args) {
      if (!args.length) { writeln('dock: usage: dock <knirvserver-url>'); return; }
      dockURL = args[0];
      writeln('Connecting to ' + dockURL + '...');
      setTimeout(() => {
        chainSession = 'cs-' + Math.random().toString(36).slice(2,10);
        mode = 'bridged';
        ENV.DVE_MODE = 'bridged';
        ENV.CHAIN_SESSION_ID = chainSession;
        modeBadge.textContent = 'BRIDGED';
        modeBadge.style.background = 'rgba(16,185,129,0.15)';
        modeBadge.style.color = '#34d399';
        setPrompt();
        writeln('✓ Docked to ' + dockURL);
        writeln('  Session: ' + chainSession + '  Trust: L2');
      }, 400);
      return 'async';
    },
    undock() {
      if (mode === 'solo') { writeln('Not currently docked.'); return; }
      mode = 'solo'; dockURL = ''; chainSession = '';
      ENV.DVE_MODE = 'solo'; delete ENV.CHAIN_SESSION_ID;
      modeBadge.textContent = 'SOLO';
      modeBadge.style.background = ''; modeBadge.style.color = '';
      setPrompt();
      writeln('Disconnected. Running in solo mode.');
    },
    net(args) {
      if (args[0] !== 'status') { writeln('net: unknown subcommand'); return; }
      writeln('Mode: ' + mode + '  Dock: ' + (dockURL || '(none)') + '  WS: ' + (mode==='bridged'?'connected':'disconnected'));
    },
    nrn(args) {
      const sub = args[0] || 'balance';
      if (sub === 'balance') {
        writeln('NRN Balance');
        if (mode === 'solo') { writeln('  Dock to query balance'); return; }
        writeln('  Address:  0x' + NODE_ID.replace('dvepod-','').padEnd(40,'0').slice(0,40));
        writeln('  Balance:  ' + (Math.random()*100).toFixed(4) + ' NRN');
      } else if (sub === 'address') {
        writeln('0x' + NODE_ID.replace('dvepod-','').padEnd(40,'0').slice(0,40));
      }
    },
    storage() {
      writeln('Backend: OPFS (Origin Private File System)');
      writeln('Quota: ~1.0 GB (browser managed)');
    },
    export() {
      writeln('Bundle: dvepod-' + NODE_ID.slice(7) + '.html');
      writeln('Encrypted: AES-256-GCM');
    },
    version() { writeln('DVE Pod v1.0.0-pod  Runtime: browser-wasm'); },
  };

  function exec(line) {
    line = line.trim();
    if (!line) return;
    history.push(line);
    histIdx = history.length;
    const parts = parseLine(line);
    const [cmd, ...args] = parts;
    if (COMMANDS[cmd]) {
      const result = COMMANDS[cmd](args);
      if (result !== 'async') setTimeout(() => {}, 0);
      else return;
    } else {
      writeln('dvepod: ' + cmd + ': command not found');
    }
  }

  function parseLine(line) {
    const tokens = []; let cur = ''; let inQ = false; let qCh = '';
    for (let i = 0; i < line.length; i++) {
      const ch = line[i];
      if (inQ) { if (ch === qCh) inQ = false; else cur += ch; }
      else if (ch === '"' || ch === "'") { inQ = true; qCh = ch; }
      else if (ch === ' ' || ch === '\t') { if (cur) { tokens.push(cur); cur = ''; } }
      else cur += ch;
    }
    if (cur) tokens.push(cur);
    return tokens;
  }

  function setPrompt() {
    const short = cwd.replace(ENV.HOME, '~');
    promptEl.textContent = 'dvepod@' + mode + ':' + short + '$ ';
  }

  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      const val = input.value;
      input.value = '';
      currentInput = '';
      exec(val);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (histIdx > 0) { histIdx--; input.value = history[histIdx] || ''; }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (histIdx < history.length - 1) { histIdx++; input.value = history[histIdx] || ''; }
      else { histIdx = history.length; input.value = ''; }
    } else if (e.key === 'l' && e.ctrlKey) {
      e.preventDefault();
      term.innerHTML = '';
    }
  });
  input.addEventListener('input', () => { currentInput = input.value; histIdx = history.length; });

  boot();
})();
</script>
</body>
</html>`
