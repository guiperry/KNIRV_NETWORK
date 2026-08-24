import React from 'react';
import { KeyRound, FileSignature } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../AuthContext';
import JwtTool from './authaudit/JwtTool';
import SamlRaider from './authaudit/SamlRaider';

export const AuthAudit: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  const isSubRoute = location.pathname !== '/auth-audit';

  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/jwt-tool" element={<JwtTool />} />
        <Route path="/saml-raider" element={<SamlRaider />} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-indigo-500/20 rounded-lg">
          <KeyRound className="w-6 h-6 text-indigo-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Auth Audit</h1>
          <p className="text-slate-400">Token and SSO assertion tampering for auth flow testing</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {canAccessSubPage('auth-audit', 'jwt-tool') && (
          <button
            onClick={() => navigate('/auth-audit/jwt-tool')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-indigo-500/20 rounded-lg group-hover:bg-indigo-500/30 transition-colors">
                <KeyRound className="w-6 h-6 text-indigo-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">jwt_tool</h3>
            </div>
            <p className="text-slate-400 mb-4 font-mono text-sm">
              Decode, tamper, and re-sign JWTs — alg confusion, none-alg, and known-key attacks.
            </p>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-indigo-500 rounded-full"></div>
              <span className="text-slate-300">token loaded</span>
            </div>
          </button>
        )}

        {canAccessSubPage('auth-audit', 'saml-raider') && (
          <button
            onClick={() => navigate('/auth-audit/saml-raider')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-violet-500/20 rounded-lg group-hover:bg-violet-500/30 transition-colors">
                <FileSignature className="w-6 h-6 text-violet-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">SAML Raider</h3>
            </div>
            <p className="text-slate-400 mb-4 font-mono text-sm">
              XML signature wrapping, certificate swap, and assertion editing for SAML 2.0 flows.
            </p>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-violet-500 rounded-full"></div>
              <span className="text-slate-300">assertion loaded</span>
            </div>
          </button>
        )}
      </div>
    </div>
  );
};
