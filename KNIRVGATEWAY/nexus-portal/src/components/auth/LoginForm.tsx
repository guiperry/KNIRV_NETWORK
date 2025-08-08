import React, { useState } from 'react';
import { Shield, User, Lock, Eye, EyeOff, Wifi } from 'lucide-react';

interface LoginFormProps {
  onLogin: (credentials: { username: string; password: string; role: string }) => void;
}

export function LoginForm({ onLogin }: LoginFormProps) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [selectedRole, setSelectedRole] = useState('validator');
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    
    // Simulate authentication delay
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    onLogin({
      username,
      password,
      role: selectedRole
    });
    
    setIsLoading(false);
  };

  const testnetCredentials = [
    { role: 'admin', username: 'admin', password: 'admin123', description: 'Full system access' },
    { role: 'validator', username: 'validator', password: 'val123', description: 'Node validation access' },
    { role: 'observer', username: 'observer', password: 'obs123', description: 'Read-only monitoring' }
  ];

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center mb-4">
            <Shield className="w-12 h-12 text-blue-400" />
          </div>
          <h1 className="text-3xl font-bold text-white mb-2">KNIRV NEXUS</h1>
          <p className="text-gray-300">Decentralized Validation Environment</p>
          <div className="flex items-center justify-center mt-2 text-green-400">
            <Wifi className="w-4 h-4 mr-2" />
            <span className="text-sm">Testnet Environment</span>
          </div>
        </div>

        {/* Login Form */}
        <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Username */}
            <div>
              <label className="block text-sm font-medium text-gray-200 mb-2">
                Username
              </label>
              <div className="relative">
                <User className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
                <input
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full pl-10 pr-4 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Enter username"
                  required
                />
              </div>
            </div>

            {/* Password */}
            <div>
              <label className="block text-sm font-medium text-gray-200 mb-2">
                Password
              </label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full pl-10 pr-12 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Enter password"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-white"
                >
                  {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                </button>
              </div>
            </div>

            {/* Role Selection */}
            <div>
              <label className="block text-sm font-medium text-gray-200 mb-2">
                Role
              </label>
              <select
                value={selectedRole}
                onChange={(e) => setSelectedRole(e.target.value)}
                className="w-full px-4 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="admin" className="bg-gray-800">Administrator</option>
                <option value="validator" className="bg-gray-800">Validator</option>
                <option value="observer" className="bg-gray-800">Observer</option>
              </select>
            </div>

            {/* Submit Button */}
            <button
              type="submit"
              disabled={isLoading}
              className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 text-white font-medium py-2 px-4 rounded-lg transition-colors flex items-center justify-center"
            >
              {isLoading ? (
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white"></div>
              ) : (
                'Access NEXUS'
              )}
            </button>
          </form>
        </div>

        {/* Testnet Credentials */}
        <div className="mt-6 bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4">
          <h3 className="text-yellow-400 font-medium mb-3">Testnet Credentials</h3>
          <div className="space-y-2">
            {testnetCredentials.map((cred) => (
              <div key={cred.role} className="text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-gray-300">
                    <span className="font-medium text-white">{cred.role}</span>: {cred.username} / {cred.password}
                  </span>
                  <button
                    type="button"
                    onClick={() => {
                      setUsername(cred.username);
                      setPassword(cred.password);
                      setSelectedRole(cred.role);
                    }}
                    className="text-blue-400 hover:text-blue-300 text-xs"
                  >
                    Use
                  </button>
                </div>
                <p className="text-gray-400 text-xs">{cred.description}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Footer */}
        <div className="text-center mt-6 text-gray-400 text-sm">
          <p>KNIRV D-TEN Testnet Environment</p>
          <p>For development and testing purposes only</p>
        </div>
      </div>
    </div>
  );
}
