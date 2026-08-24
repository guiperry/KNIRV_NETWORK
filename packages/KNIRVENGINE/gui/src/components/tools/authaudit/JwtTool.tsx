import React, { useState } from 'react';
import { KeyRound, Play } from 'lucide-react';

const sampleToken =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c3JfODhmMiIsInJvbGUiOiJ1c2VyIiwiaWF0IjoxNzI0NDU3NjAwLCJleHAiOjE3MjQ0NjEyMDB9.4x2h1z9kLmQwv8t3rB2nS6cQpJd1YV5aZ0N7oT4uKfE';

const header = { alg: 'HS256', typ: 'JWT' };
const payload = { sub: 'usr_88f2', role: 'user', iat: 1724457600, exp: 1724461200 };

const attackModes = [
  { id: 'none', label: 'alg:none', desc: 'Strip signature, set alg to "none" — tests for missing signature verification.' },
  { id: 'confusion', label: 'RS256 → HS256 confusion', desc: 'Re-sign with HS256 using the RS256 public key as the HMAC secret.' },
  { id: 'crack', label: 'Known-key / dictionary crack', desc: 'Brute-force the HMAC secret against a wordlist (jwt_tool -C -d wordlist.txt).' },
  { id: 'kid', label: 'kid header injection', desc: 'Inject a path or SQLi payload into the "kid" header to redirect key lookup.' },
];

const JwtTool: React.FC = () => {
  const [token, setToken] = useState(sampleToken);
  const [role, setRole] = useState(payload.role);
  const [mode, setMode] = useState<string | null>(null);
  const [output, setOutput] = useState<string[]>([]);

  const runAttack = (id: string) => {
    setMode(id);
    const lines: Record<string, string[]> = {
      none: [
        '[+] Original token has algorithm HS256',
        '[+] Tampered token 1 (alg none): eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ1c3JfODhmMiIsInJvbGUiOiJhZG1pbiJ9.',
        '[+] Sending to target... 200 OK — server accepted unsigned token',
      ],
      confusion: [
        '[+] Fetched RSA public key from /.well-known/jwks.json',
        '[+] Re-signed payload using HS256 with the PEM public key as secret',
        '[+] Forged token: eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c3JfODhmMiIsInJvbGUiOiJhZG1pbiJ9.k3f...',
        '[+] Sending to target... 401 Unauthorized — server validates key type, attack blocked',
      ],
      crack: [
        '[+] Loaded wordlist.txt (14,344,392 candidates)',
        '[*] Trying: password, 123456, secret, changeme...',
        '[-] No match after 14,344,392 attempts — secret is not in the wordlist',
      ],
      kid: [
        `[+] Injected header: {"alg":"HS256","kid":"../../../../dev/null"}`,
        '[+] Server resolved kid to /dev/null (empty key) — signature check passed with empty secret',
        '[!] Forged admin token accepted — kid path traversal confirmed',
      ],
    };
    setOutput(lines[id] ?? []);
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-indigo-500/20 rounded-lg">
          <KeyRound className="w-6 h-6 text-indigo-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">jwt_tool</h1>
          <p className="text-slate-400 text-sm font-mono">python3 jwt_tool.py -t $TOKEN -M pb</p>
        </div>
      </div>

      <div className="mb-4">
        <textarea
          value={token}
          onChange={e => setToken(e.target.value)}
          spellCheck={false}
          className="w-full h-16 bg-slate-800/50 border border-slate-700/50 rounded-lg px-3 py-2 text-xs font-mono text-slate-200 resize-none focus:outline-none focus:border-indigo-500/50"
        />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="text-xs text-red-400 uppercase mb-1">Header</div>
          <pre className="text-xs font-mono text-red-300 bg-slate-900/60 rounded p-2 mb-4">{JSON.stringify(header, null, 2)}</pre>

          <div className="text-xs text-purple-400 uppercase mb-1">Payload</div>
          <pre className="text-xs font-mono text-purple-300 bg-slate-900/60 rounded p-2">
{JSON.stringify({ ...payload, role }, null, 2)}
          </pre>
          <div className="mt-2 flex items-center space-x-2">
            <span className="text-xs text-slate-500 font-mono">role:</span>
            <select
              value={role}
              onChange={e => setRole(e.target.value)}
              className="bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-xs font-mono text-slate-200"
            >
              <option value="user">user</option>
              <option value="admin">admin</option>
              <option value="root">root</option>
            </select>
          </div>

          <div className="text-xs text-blue-400 uppercase mb-1 mt-4">Signature</div>
          <div className="text-xs font-mono text-blue-300 bg-slate-900/60 rounded p-2 truncate">
            HMACSHA256(base64(header)+"."+base64(payload), secret)
          </div>
        </div>

        <div className="xl:col-span-2 bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="text-xs text-slate-500 uppercase mb-3">Attack playbook</div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 mb-4">
            {attackModes.map(a => (
              <button
                key={a.id}
                onClick={() => runAttack(a.id)}
                className={`text-left p-3 rounded-lg border transition-colors ${
                  mode === a.id ? 'bg-indigo-500/15 border-indigo-500/40' : 'bg-slate-900/40 border-slate-700/50 hover:border-slate-600'
                }`}
              >
                <div className="flex items-center space-x-2 mb-1">
                  <Play className="w-3.5 h-3.5 text-indigo-400" />
                  <span className="text-sm font-mono text-indigo-300">{a.label}</span>
                </div>
                <div className="text-xs text-slate-500">{a.desc}</div>
              </button>
            ))}
          </div>

          <div className="text-xs text-slate-500 uppercase mb-1">Output</div>
          <div className="bg-slate-900/60 rounded p-2 font-mono text-xs min-h-[110px] space-y-1">
            {output.length === 0 ? (
              <span className="text-slate-700">select an attack mode</span>
            ) : (
              output.map((l, i) => (
                <div key={i} className={l.includes('!') ? 'text-red-400' : l.includes('Sending') || l.includes('Loaded') ? 'text-slate-400' : 'text-green-400'}>
                  {l}
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default JwtTool;
