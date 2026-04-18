'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function LoginPage() {
  const router = useRouter();
  const [loginMode, setLoginMode] = useState<'credentials' | 'token'>('credentials');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [showRegister, setShowRegister] = useState(false);

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [token, setToken] = useState('');

  const [regUsername, setRegUsername] = useState('');
  const [regEmail, setRegEmail] = useState('');
  const [regRole, setRegRole] = useState('observer');
  const [regPassword, setRegPassword] = useState('');
  const [regConfirm, setRegConfirm] = useState('');
  const [registerSuccess, setRegisterSuccess] = useState('');

  const testnetTokens = {
    'testnet-admin-123': { role: 'admin', user: 'testnet-admin' },
    'testnet-validator-456': { role: 'validator', user: 'testnet-validator' },
    'testnet-observer-789': { role: 'observer', user: 'testnet-observer' },
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      let authToken: string | null = null;
      let role = '';
      const serverUrl = window.location.origin;

      if (loginMode === 'credentials') {
        if (!username || !password) {
          setError('Username and password are required.');
          setLoading(false);
          return;
        }

        const res = await fetch(`${serverUrl}/api/auth/login`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password }),
        });

        const body = await res.json().catch(() => ({}));
        if (!res.ok) {
          setError(body.error || body.message || `Authentication failed (${res.status})`);
          setLoading(false);
          return;
        }

        authToken = body.token;
        role = body.role || '';
      } else {
        const rawToken = token.trim();
        if (!rawToken) {
          setError('Please enter an access token.');
          setLoading(false);
          return;
        }

        if (testnetTokens[rawToken as keyof typeof testnetTokens]) {
          authToken = rawToken;
          role = testnetTokens[rawToken as keyof typeof testnetTokens].role;
        } else {
          if (rawToken.startsWith('ey')) {
            try {
              const payload = JSON.parse(atob(rawToken.split('.')[1]));
              if (payload.exp && payload.exp * 1000 < Date.now()) {
                setError('Token has expired.');
                setLoading(false);
                return;
              }
            } catch {}
          }

          const res = await fetch(`${serverUrl}/api/auth/me`, {
            headers: { 'Authorization': `Bearer ${rawToken}` },
          });

          const body = await res.json().catch(() => ({}));
          if (!res.ok) {
            setError(body.error || body.message || 'Invalid or expired token.');
            setLoading(false);
            return;
          }

          authToken = rawToken;
          role = body.role || '';
        }
      }

      if (authToken) {
        localStorage.setItem('knirv_nexus_token', authToken);
        localStorage.setItem('knirv_nexus_role', role);
        localStorage.setItem('knirv_nexus_user', username || 'user');
        localStorage.setItem('knirv_auth_token', authToken);
        localStorage.setItem('knirv_auth_role', role);

        router.push('/menu');
      }
    } catch {
      setError('Cannot reach the KNIRV server. Ensure the server is running.');
      setLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    setRegisterSuccess('');

    if (!regUsername || !regEmail || !regPassword) {
      setError('Please fill in all required fields.');
      setLoading(false);
      return;
    }

    if (regPassword !== regConfirm) {
      setError('Passwords do not match.');
      setLoading(false);
      return;
    }

    if (regPassword.length < 8) {
      setError('Password must be at least 8 characters.');
      setLoading(false);
      return;
    }

    try {
      const serverUrl = window.location.origin;
      const res = await fetch(`${serverUrl}/api/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username: regUsername,
          email: regEmail,
          password: regPassword,
          first_name: regUsername,
          last_name: regUsername,
          role: regRole,
        }),
      });

      const body = await res.json().catch(() => ({}));
      if (res.ok) {
        setRegisterSuccess('Registration successful! You can now log in.');
        setTimeout(() => {
          setShowRegister(false);
          setRegUsername('');
          setRegEmail('');
          setRegPassword('');
          setRegConfirm('');
        }, 2500);
      } else {
        setError(body.error || body.message || 'Registration failed. Please try again.');
      }
    } catch {
      setError('Cannot reach the KNIRV server.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const existingToken = localStorage.getItem('knirv_nexus_token') || localStorage.getItem('knirv_auth_token');
    if (existingToken) {
      router.push('/menu');
    }
  }, [router]);

  return (
    <div className="min-h-screen bg-black text-white relative overflow-hidden">
      <div className="login-scanline absolute inset-0 pointer-events-none opacity-10" />
      
      <div className="phase-overlay flex items-center justify-center min-h-screen">
        {!showRegister ? (
          <div id="login-panel" className="login-panel">
            <div className="text-center mb-8">
              <div className="login-logo mx-auto mb-4">
                <div className="login-ring lr1" />
                <div className="login-ring lr2" />
                <div className="login-ring lr3" />
                <div className="login-core">KNIRV</div>
              </div>
              <div className="login-brand-title text-xl font-bold tracking-wider">KNIRVSERVER</div>
              <div className="login-brand-sub text-xs text-gray-400 tracking-widest mb-6">DISTRIBUTED VALIDATION ENVIRONMENT</div>
            </div>

            <div className="login-tabs flex mb-4 border-b border-gray-800">
              <button
                className={`login-tab flex-1 py-2 ${loginMode === 'credentials' ? 'active border-b border-blue-500 text-blue-400' : 'text-gray-500'}`}
                onClick={() => setLoginMode('credentials')}
              >
                CREDENTIALS
              </button>
              <button
                className={`login-tab flex-1 py-2 ${loginMode === 'token' ? 'active border-b border-blue-500 text-blue-400' : 'text-gray-500'}`}
                onClick={() => setLoginMode('token')}
              >
                ACCESS TOKEN
              </button>
            </div>

            <form onSubmit={handleLogin} className="login-form space-y-4">
              {loginMode === 'credentials' ? (
                <div id="creds-fields" className="space-y-4">
                  <div className="login-field">
                    <label className="login-label text-xs text-gray-500 tracking-wider block mb-1">USERNAME</label>
                    <input
                      className="login-input w-full bg-gray-900 border border-gray-700 px-3 py-2 rounded text-sm"
                      type="text"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      placeholder="Enter username"
                      spellCheck={false}
                      autoComplete="off"
                    />
                  </div>
                  <div className="login-field">
                    <label className="login-label text-xs text-gray-500 tracking-wider block mb-1">PASSWORD</label>
                    <input
                      className="login-input w-full bg-gray-900 border border-gray-700 px-3 py-2 rounded text-sm"
                      type="password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      placeholder="Enter password"
                      autoComplete="current-password"
                    />
                  </div>
                </div>
              ) : (
                <div id="token-fields" className="space-y-4">
                  <div className="login-field">
                    <label className="login-label text-xs text-gray-500 tracking-wider block mb-1">ACCESS TOKEN</label>
                    <input
                      className="login-input w-full bg-gray-900 border border-gray-700 px-3 py-2 rounded text-sm"
                      type="password"
                      value={token}
                      onChange={(e) => setToken(e.target.value)}
                      placeholder="Enter access token"
                    />
                  </div>

                  <div className="testnet-section bg-gray-900/50 rounded p-3 mt-4">
                    <div className="testnet-title text-xs text-gray-400 tracking-wider mb-3">TESTNET TOKENS <span className="testnet-badge text-blue-400">DEVELOPMENT</span></div>
                    
                    {Object.entries(testnetTokens).map(([tokenValue, info]) => (
                      <div key={tokenValue} className="testnet-row flex justify-between items-center py-2 border-b border-gray-800 last:border-0">
                        <div className="testnet-info">
                          <span className="testnet-role text-sm">{info.role.charAt(0).toUpperCase() + info.role.slice(1)}</span>
                          <span className={`testnet-access ml-2 text-xs ${info.role === 'admin' ? 'access-full text-green-400' : info.role === 'validator' ? 'access-scoped text-yellow-400' : 'access-read text-blue-400'}`}>
                            {info.role === 'admin' ? 'Full Access' : info.role === 'validator' ? 'Scoped' : 'Read Only'}
                          </span>
                        </div>
                        <button
                          type="button"
                          className="testnet-btn text-xs px-3 py-1 border border-gray-600 rounded hover:border-blue-500 hover:text-blue-400 transition-colors"
                          onClick={() => setToken(tokenValue)}
                        >
                          USE
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {error && <div id="login-error" className="login-error text-red-400 text-sm py-2">{error}</div>}
              
              <button type="submit" className="login-btn w-full bg-blue-600 hover:bg-blue-700 py-2 rounded text-sm font-medium transition-colors disabled:opacity-50" disabled={loading}>
                <span id="login-btn-text">{loading ? 'AUTHENTICATING...' : 'AUTHENTICATE'}</span>
              </button>
            </form>

            <div className="login-register-link text-center mt-4 text-xs text-gray-500">
              <span>No account?</span>
              <button type="button" onClick={() => setShowRegister(true)} className="link-btn ml-1 text-blue-400 hover:underline">REGISTER</button>
            </div>
          </div>
        ) : (
          <div id="register-panel" className="login-panel">
            <div className="register-header text-center text-lg font-bold mb-6">CREATE ACCOUNT</div>

            <form onSubmit={handleRegister} className="login-form space-y-4">
              <div className="login-field">
                <label className="login-label text-xs text-gray-500 tracking-wider block mb-1">USERNAME *</label>
                <input
                  className="login-input w-full bg-gray-900 border border-gray-700 px-3 py-2 rounded text-sm"
                  type="text"
                  value={regUsername}
                  onChange={(e) => setRegUsername(e.target.value)}
                  placeholder="Choose a username"
                  spellCheck={false}
                />
              </div>
              <div className="login-field">
                <label className="login-label text-xs text-gray-500 tracking-wider block mb-1">EMAIL *</label>
                <input
                  className="login-input w-full bg-gray-900 border border-gray-700 px-3 py-2 rounded text-sm"
                  type="email"
                  value={regEmail}
                  onChange={(e) => setRegEmail(e.target.value)}
                  placeholder="your@email.com"
                />
              </div>
              <div className="login-field">
                <label className="login-label text-xs text-gray-500 tracking-wider block mb-1">ROLE *</label>
                <select
                  className="login-input login-select w-full bg-gray-900 border border-gray-700 px-3 py-2 rounded text-sm"
                  value={regRole}
                  onChange={(e) => setRegRole(e.target.value)}
                >
                  <option value="observer">Developer (Observer) — Read-only</option>
                  <option value="validator">Operator (Validator) — Scoped</option>
                  <option value="admin">Root (Admin) — Full access</option>
                </select>
              </div>
              <div className="login-field">
                <label className="login-label text-xs text-gray-500 tracking-wider block mb-1">PASSWORD * <span className="hint text-gray-600">(min 8 chars)</span></label>
                <input
                  className="login-input w-full bg-gray-900 border border-gray-700 px-3 py-2 rounded text-sm"
                  type="password"
                  value={regPassword}
                  onChange={(e) => setRegPassword(e.target.value)}
                  placeholder="Min 8 characters"
                />
              </div>
              <div className="login-field">
                <label className="login-label text-xs text-gray-500 tracking-wider block mb-1">CONFIRM PASSWORD *</label>
                <input
                  className="login-input w-full bg-gray-900 border border-gray-700 px-3 py-2 rounded text-sm"
                  type="password"
                  value={regConfirm}
                  onChange={(e) => setRegConfirm(e.target.value)}
                  placeholder="Confirm your password"
                />
              </div>

              {error && <div className="login-error text-red-400 text-sm py-2">{error}</div>}
              {registerSuccess && <div className="register-success text-green-400 text-sm py-2">{registerSuccess}</div>}

              <div className="login-btn-row flex gap-3 mt-4">
                <button type="button" onClick={() => setShowRegister(false)} className="login-btn login-btn-secondary flex-1 bg-gray-800 hover:bg-gray-700 py-2 rounded text-sm font-medium transition-colors">CANCEL</button>
                <button type="submit" className="login-btn login-btn-primary flex-1 bg-blue-600 hover:bg-blue-700 py-2 rounded text-sm font-medium transition-colors disabled:opacity-50" disabled={loading}>
                  <span id="register-btn-text">{loading ? 'REGISTERING...' : 'REGISTER'}</span>
                </button>
              </div>
            </form>
          </div>
        )}
      </div>

      <style jsx>{`
        .login-ring {
          position: absolute;
          border: 2px solid rgba(72, 136, 255, 0.3);
          border-radius: 50%;
          animation: ring-rotate 20s linear infinite;
        }
        .lr1 { width: 80px; height: 80px; top: 50%; left: 50%; transform: translate(-50%, -50%); animation-duration: 15s; }
        .lr2 { width: 100px; height: 100px; top: 50%; left: 50%; transform: translate(-50%, -50%); animation-duration: 20s; animation-direction: reverse; }
        .lr3 { width: 120px; height: 120px; top: 50%; left: 50%; transform: translate(-50%, -50%); animation-duration: 25s; }
        @keyframes ring-rotate { from { transform: translate(-50%, -50%) rotate(0deg); } to { transform: translate(-50%, -50%) rotate(360deg); } }
        .login-scanline { background: linear-gradient(to bottom, transparent 50%, rgba(0, 0, 0, 0.1) 50%); background-size: 100% 4px; }
      `}</style>
    </div>
  );
}