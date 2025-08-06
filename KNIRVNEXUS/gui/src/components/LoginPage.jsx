import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { User, Lock, Mail, AlertCircle, Loader } from 'lucide-react';
import { useAuth } from './AuthContext';
import { useAppLogo } from '../hooks/useAssetPath';

const LoginPage = () => {
  const logoPath = useAppLogo();
  const [isLogin, setIsLogin] = useState(true);
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: ''
  });
  const [errors, setErrors] = useState({});
  const [isLoading, setIsLoading] = useState(false);
  const [apiError, setApiError] = useState('');
  const navigate = useNavigate();
  const { login, register } = useAuth();

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    
    // Clear error for this field
    if (errors[name]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
    
    // Clear API error when user starts typing
    if (apiError) {
      setApiError('');
    }
  };

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.username.trim()) {
      newErrors.username = 'Username is required';
    }
    
    if (!isLogin && !formData.email.trim()) {
      newErrors.email = 'Email is required';
    } else if (!isLogin && !/\S+@\S+\.\S+/.test(formData.email)) {
      newErrors.email = 'Email is invalid';
    }
    
    if (!formData.password.trim()) {
      newErrors.password = 'Password is required';
    } else if (!isLogin && formData.password.length < 8) {
      newErrors.password = 'Password must be at least 8 characters';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsLoading(true);
    setApiError('');

    try {
      const credentials = {
        username: formData.username,
        password: formData.password,
      };

      if (!isLogin) {
        credentials.email = formData.email;
      }

      const result = isLogin
        ? await login(credentials)
        : await register(credentials);

      if (result.success) {
        // Redirect to dashboard
        navigate('/dashboard');
      } else {
        setApiError(result.error || 'Authentication failed');
      }

    } catch (error) {
      console.error('Authentication error:', error);
      setApiError(error.message || 'An unexpected error occurred');
    } finally {
      setIsLoading(false);
    }
  };

  const toggleAuthMode = () => {
    setIsLogin(!isLogin);
    setErrors({});
    setApiError('');
  };

  return (
    <div className="min-h-screen bg-knirv-gradient flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="bg-slate-800/90 backdrop-blur-sm rounded-xl border border-slate-700/50 shadow-xl overflow-hidden">
          {/* Header */}
          <div className="p-6 border-b border-slate-700/50 flex items-center justify-center">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-white rounded-lg">
                <img
                  src="/favicon.png"
                  alt="KNIRV-NEXUS Logo"
                  className="w-6 h-6 object-contain"
                />
              </div>
              <div>
                <h1 className="text-xl font-bold text-white">KNIRV-NEXUS</h1>
                <p className="text-sm text-slate-400">TEE-Enabled Agent Platform</p>
              </div>
            </div>
          </div>
          
          {/* Form */}
          <div className="p-6">
            <h2 className="text-2xl font-bold text-white mb-6 text-center">
              {isLogin ? 'Sign In' : 'Create Account'}
            </h2>
            
            {apiError && (
              <div className="mb-4 p-3 bg-red-500/20 border border-red-500/30 rounded-lg flex items-start space-x-2">
                <AlertCircle className="w-5 h-5 text-red-400 mt-0.5" />
                <p className="text-red-400 text-sm">{apiError}</p>
              </div>
            )}
            
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label htmlFor="username" className="block text-sm font-medium text-slate-300 mb-1">
                  Username
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <User className="h-5 w-5 text-slate-400" />
                  </div>
                  <input
                    id="username"
                    name="username"
                    type="text"
                    value={formData.username}
                    onChange={handleChange}
                    className={`w-full bg-slate-700/50 border ${errors.username ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-knirv-primary`}
                    placeholder="Enter your username"
                  />
                </div>
                {errors.username && (
                  <p className="mt-1 text-sm text-red-400">{errors.username}</p>
                )}
              </div>
              
              {!isLogin && (
                <div>
                  <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-1">
                    Email
                  </label>
                  <div className="relative">
                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                      <Mail className="h-5 w-5 text-slate-400" />
                    </div>
                    <input
                      id="email"
                      name="email"
                      type="email"
                      value={formData.email}
                      onChange={handleChange}
                      className={`w-full bg-slate-700/50 border ${errors.email ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-knirv-primary`}
                      placeholder="Enter your email"
                    />
                  </div>
                  {errors.email && (
                    <p className="mt-1 text-sm text-red-400">{errors.email}</p>
                  )}
                </div>
              )}
              
              <div>
                <label htmlFor="password" className="block text-sm font-medium text-slate-300 mb-1">
                  Password
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <Lock className="h-5 w-5 text-slate-400" />
                  </div>
                  <input
                    id="password"
                    name="password"
                    type="password"
                    value={formData.password}
                    onChange={handleChange}
                    className={`w-full bg-slate-700/50 border ${errors.password ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-knirv-primary`}
                    placeholder={isLogin ? "Enter your password" : "Create a password"}
                  />
                </div>
                {errors.password && (
                  <p className="mt-1 text-sm text-red-400">{errors.password}</p>
                )}
              </div>
              
              {isLogin && (
                <div className="flex items-center justify-end">
                  <button type="button" className="text-sm text-knirv-primary hover:text-knirv-secondary">
                    Forgot password?
                  </button>
                </div>
              )}
              
              <button
                type="submit"
                disabled={isLoading}
                className="w-full bg-knirv-primary-gradient text-white py-2 px-4 rounded-lg font-medium hover:opacity-90 transition-all duration-200 flex items-center justify-center space-x-2 disabled:opacity-70"
              >
                {isLoading ? (
                  <>
                    <Loader className="w-5 h-5 animate-spin" />
                    <span>{isLogin ? 'Signing in...' : 'Creating account...'}</span>
                  </>
                ) : (
                  <span>{isLogin ? 'Sign In' : 'Create Account'}</span>
                )}
              </button>
            </form>
          </div>
          
          {/* Footer */}
          <div className="p-6 bg-slate-800/50 border-t border-slate-700/50 text-center">
            <p className="text-slate-400 text-sm">
              {isLogin ? "Don't have an account?" : "Already have an account?"}
              <button
                onClick={toggleAuthMode}
                className="ml-2 text-knirv-primary hover:text-knirv-secondary font-medium"
              >
                {isLogin ? 'Sign Up' : 'Sign In'}
              </button>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default LoginPage;