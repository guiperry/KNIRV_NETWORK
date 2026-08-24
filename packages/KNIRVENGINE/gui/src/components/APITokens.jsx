import React, { useState, useEffect } from 'react';
import {
  Key,
  Plus,
  Trash2,
  Copy,
  Eye,
  EyeOff,
  Calendar,
  Clock,
  AlertTriangle,
  CheckCircle,
  RefreshCw
} from 'lucide-react';
import { useAuth } from './AuthContext';

// KNIRVENGINE API configuration. Keep engine traffic local even when the UI
// was opened from a file URL or another KNIRV site.
const getApiBaseUrl = () => {
  return (import.meta.env.VITE_API_BASE_URL || (window.location.protocol === 'file:' ? 'http://localhost:8081' : '')).replace(/\/$/, '');
};

const APITokens = () => {
  const [tokens, setTokens] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isAddTokenModalOpen, setIsAddTokenModalOpen] = useState(false);
  const [formData, setFormData] = useState({
    description: '',
    expires_at: ''
  });
  const [formErrors, setFormErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null
  const [newToken, setNewToken] = useState(null);
  const [showTokens, setShowTokens] = useState({});
  const [showLoginPrompt, setShowLoginPrompt] = useState(false);
  const { isAuthenticated } = useAuth();

  // Fetch tokens when authenticated
  useEffect(() => {
    if (isAuthenticated) {
      fetchTokens();
    }
  }, [isAuthenticated]);

  const fetchTokens = async () => {
    if (!isAuthenticated) {
      setError('Authentication required to access API tokens');
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const token = localStorage.getItem('token');
      console.log('Fetching API tokens with token:', token ? 'present' : 'missing');

      if (!token) {
        throw new Error('No authentication token found');
      }

      const apiBaseUrl = getApiBaseUrl();
      const response = await fetch(`${apiBaseUrl}/api/v1/auth/tokens`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

      console.log('API tokens response status:', response.status);

      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('Authentication failed. Please log in again.');
        }
        const errorText = await response.text();
        console.error('API tokens response error:', errorText);
        throw new Error(`Failed to fetch API tokens: ${response.status} ${response.statusText}`);
      }

      const data = await response.json();
      console.log('API tokens data:', data);
      setTokens(data.tokens || []);
    } catch (error) {
      console.error('Error fetching API tokens:', error);
      setError(error.message);

      // If authentication failed, show login prompt
      if (error.message.includes('Authentication failed') || error.message.includes('No authentication token')) {
        setShowLoginPrompt(true);
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleAddToken = () => {
    setFormData({
      description: '',
      expires_at: ''
    });
    setFormErrors({});
    setSubmitStatus(null);
    setNewToken(null);
    setIsAddTokenModalOpen(true);
  };

  const handleDeleteToken = async (tokenId) => {
    if (!confirm('Are you sure you want to delete this API token?')) {
      return;
    }
    
    try {
      const token = localStorage.getItem('token');
      const apiBaseUrl = getApiBaseUrl();
      const response = await fetch(`${apiBaseUrl}/api/v1/auth/tokens/${tokenId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to delete API token');
      }
      
      // Remove token from state
      setTokens(tokens.filter(t => t.id !== tokenId));
    } catch (error) {
      console.error('Error deleting API token:', error);
      alert('Failed to delete API token: ' + error.message);
    }
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    
    // Clear error for this field
    if (formErrors[name]) {
      setFormErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
  };

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.description.trim()) {
      newErrors.description = 'Description is required';
    }
    
    setFormErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }
    
    setIsSubmitting(true);
    setSubmitStatus(null);
    
    try {
      const token = localStorage.getItem('token');
      
      // Prepare request data
      const requestData = {
        description: formData.description,
      };
      
      // Add expiration date if provided
      if (formData.expires_at) {
        requestData.expires_at = new Date(formData.expires_at).toISOString();
      }
      
      const apiBaseUrl = getApiBaseUrl();
      const response = await fetch(`${apiBaseUrl}/api/v1/auth/tokens`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestData),
      });
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to create API token');
      }
      
      const data = await response.json();
      
      // Store the new token to display to the user
      setNewToken(data.token);
      
      // Add token to the list
      setTokens([...tokens, data.token]);
      
      setSubmitStatus('success');
    } catch (error) {
      console.error('Error creating API token:', error);
      setSubmitStatus('error');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCopyToken = (tokenValue) => {
    navigator.clipboard.writeText(tokenValue)
      .then(() => {
        alert('Token copied to clipboard');
      })
      .catch(err => {
        console.error('Failed to copy token:', err);
        alert('Failed to copy token to clipboard');
      });
  };

  const toggleShowToken = (tokenId) => {
    setShowTokens(prev => ({
      ...prev,
      [tokenId]: !prev[tokenId]
    }));
  };

  const formatDate = (dateString) => {
    if (!dateString) return 'Never';
    return new Date(dateString).toLocaleString();
  };

  const isExpired = (expiresAt) => {
    if (!expiresAt) return false;
    return new Date(expiresAt) < new Date();
  };

  // Authentication prompt
  if (showLoginPrompt && !isAuthenticated) {
    return (
      <div className="space-y-6">
        <h3 className="text-lg font-semibold text-white">API Tokens</h3>
        <div className="max-w-md mx-auto bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
          <div className="text-center mb-6">
            <AlertTriangle className="w-12 h-12 text-yellow-500 mx-auto mb-4" />
            <h3 className="text-xl font-semibold text-white mb-2">Authentication Required</h3>
            <p className="text-slate-400">
              API token management requires authentication. Please log in with admin credentials.
            </p>
          </div>

          <div className="space-y-4">
            <button
              onClick={() => { window.location.href = '/login'; }}
              className="w-full bg-purple-600 hover:bg-purple-700 text-white px-4 py-2 rounded-lg transition-colors"
            >
              Sign in
            </button>

            <button
              onClick={() => setShowLoginPrompt(false)}
              className="w-full bg-slate-700 hover:bg-slate-600 text-white px-4 py-2 rounded-lg transition-colors"
            >
              Cancel
            </button>
          </div>

          {error && (
            <div className="mt-4 p-3 bg-red-900/30 border border-red-600/30 rounded-lg">
              <p className="text-red-400 text-sm">{error}</p>
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <h3 className="text-lg font-semibold text-white">API Tokens</h3>
        <div className="mt-4 lg:mt-0">
          <button
            onClick={handleAddToken}
            className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-4 py-2 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2"
          >
            <Plus className="w-4 h-4" />
            <span>Create Token</span>
          </button>
        </div>
      </div>
      
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="w-8 h-8 border-4 border-purple-500/30 border-t-purple-500 rounded-full animate-spin"></div>
          <span className="ml-3 text-slate-400">Loading API tokens...</span>
        </div>
      ) : error ? (
        <div className="p-4 bg-red-500/20 border border-red-500/30 rounded-lg">
          <div className="flex items-start space-x-3">
            <AlertTriangle className="w-5 h-5 text-red-400 mt-0.5" />
            <div>
              <p className="text-red-400 font-medium">Error loading API tokens</p>
              <p className="text-red-300/70 text-sm">{error}</p>
              <button
                onClick={fetchTokens}
                className="mt-2 px-3 py-1 bg-red-500/20 text-red-400 rounded-lg hover:bg-red-500/30 transition-colors duration-200 flex items-center space-x-1 text-sm"
              >
                <RefreshCw className="w-3 h-3" />
                <span>Retry</span>
              </button>
            </div>
          </div>
        </div>
      ) : (
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700/50 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-slate-700/30 border-b border-slate-600/50">
                <tr>
                  <th className="text-left py-3 px-6 text-slate-300 font-medium">Description</th>
                  <th className="text-left py-3 px-6 text-slate-300 font-medium">Token</th>
                  <th className="text-left py-3 px-6 text-slate-300 font-medium">Created</th>
                  <th className="text-left py-3 px-6 text-slate-300 font-medium">Expires</th>
                  <th className="text-left py-3 px-6 text-slate-300 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700/50">
                {tokens.length > 0 ? (
                  tokens.map((token) => (
                    <tr key={token.id} className="hover:bg-slate-700/20 transition-colors duration-200">
                      <td className="py-4 px-6 text-white font-medium">{token.description}</td>
                      <td className="py-4 px-6">
                        <div className="flex items-center space-x-2">
                          <span className="text-slate-300 font-mono text-sm">
                            {showTokens[token.id] ? token.token : token.token.substring(0, 8) + '...'}
                          </span>
                          <button
                            onClick={() => toggleShowToken(token.id)}
                            className="p-1 text-slate-400 hover:text-white transition-colors duration-200"
                            title={showTokens[token.id] ? 'Hide token' : 'Show token'}
                          >
                            {showTokens[token.id] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                          </button>
                          <button
                            onClick={() => handleCopyToken(token.token)}
                            className="p-1 text-slate-400 hover:text-white transition-colors duration-200"
                            title="Copy token"
                          >
                            <Copy className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                      <td className="py-4 px-6 text-slate-300">{formatDate(token.created_at)}</td>
                      <td className="py-4 px-6">
                        {token.expires_at ? (
                          <span className={isExpired(token.expires_at) ? 'text-red-400' : 'text-slate-300'}>
                            {formatDate(token.expires_at)}
                            {isExpired(token.expires_at) && ' (Expired)'}
                          </span>
                        ) : (
                          <span className="text-green-400">Never</span>
                        )}
                      </td>
                      <td className="py-4 px-6">
                        <button
                          onClick={() => handleDeleteToken(token.id)}
                          className="p-1 text-slate-400 hover:text-red-400 transition-colors duration-200"
                          title="Delete token"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={5} className="py-8 text-center text-slate-400">
                      No API tokens found. Create a token to get started.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
      
      {/* Add Token Modal */}
      {isAddTokenModalOpen && (
        <div className="fixed inset-0 z-50 flex items-start justify-center pt-20 pb-8 px-4">
          <div
            className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
            onClick={() => !isSubmitting && !newToken && setIsAddTokenModalOpen(false)}
          ></div>

          <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-md max-h-[calc(100vh-160px)] overflow-hidden">
            <div className="flex items-center justify-between p-6 border-b border-slate-700">
              <div className="flex items-center space-x-3">
                <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
                  <Key className="w-5 h-5 text-purple-400" />
                </div>
                <h2 className="text-xl font-bold text-white">Create API Token</h2>
              </div>
              <button
                onClick={() => !isSubmitting && !newToken && setIsAddTokenModalOpen(false)}
                aria-label="Close"
                className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
                disabled={isSubmitting || newToken}
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            
            <div className="p-6 overflow-y-auto max-h-[calc(90vh-140px)]">
              {newToken ? (
                <div className="space-y-4">
                  <div className="p-4 bg-green-500/20 border border-green-500/30 rounded-lg">
                    <div className="flex items-start space-x-3">
                      <CheckCircle className="w-5 h-5 text-green-400 mt-0.5" />
                      <div>
                        <p className="text-green-400 font-medium">API Token Created Successfully</p>
                        <p className="text-green-300/70 text-sm mt-1">
                          Make sure to copy your token now. You won't be able to see it again!
                        </p>
                      </div>
                    </div>
                  </div>
                  
                  <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                    <label className="block text-sm font-medium text-slate-300 mb-2">
                      Your New API Token
                    </label>
                    <div className="flex items-center space-x-2">
                      <input
                        type="text"
                        value={newToken.token}
                        readOnly
                        className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white font-mono text-sm focus:outline-none focus:ring-2 focus:ring-purple-500"
                      />
                      <button
                        onClick={() => handleCopyToken(newToken.token)}
                        className="p-2 bg-slate-700/50 border border-slate-600/50 rounded-lg text-slate-300 hover:text-white transition-colors duration-200"
                        title="Copy token"
                      >
                        <Copy className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                  
                  <div className="p-4 bg-yellow-500/10 border border-yellow-500/20 rounded-lg">
                    <div className="flex items-start space-x-3">
                      <AlertTriangle className="w-5 h-5 text-yellow-400 mt-0.5" />
                      <div>
                        <p className="text-yellow-400 font-medium">Security Warning</p>
                        <p className="text-yellow-300/70 text-sm">
                          This token provides access to your account. Do not share it with anyone or expose it in client-side code.
                        </p>
                      </div>
                    </div>
                  </div>
                  
                  <div className="flex justify-end">
                    <button
                      onClick={() => {
                        setIsAddTokenModalOpen(false);
                        setNewToken(null);
                      }}
                      className="px-6 py-2 bg-gradient-to-r from-purple-500 to-blue-500 text-white rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200"
                    >
                      Done
                    </button>
                  </div>
                </div>
              ) : (
                <form onSubmit={handleSubmit} className="space-y-4">
                  {submitStatus === 'error' && (
                    <div className="p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
                      <div className="flex items-center space-x-2">
                        <AlertTriangle className="w-5 h-5 text-red-400" />
                        <span className="text-red-400 font-medium">Failed to create API token</span>
                      </div>
                    </div>
                  )}
                  
                  <div>
                    <label htmlFor="description" className="block text-sm font-medium text-slate-300 mb-2">
                      Token Description
                    </label>
                    <input
                      id="description"
                      name="description"
                      type="text"
                      value={formData.description}
                      onChange={handleChange}
                      className={`w-full bg-slate-700/50 border ${formErrors.description ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                      placeholder="e.g., Development API, Production Server, etc."
                    />
                    {formErrors.description && (
                      <p className="mt-1 text-sm text-red-400">{formErrors.description}</p>
                    )}
                  </div>
                  
                  <div>
                    <label htmlFor="expires_at" className="block text-sm font-medium text-slate-300 mb-2">
                      Expiration Date (Optional)
                    </label>
                    <div className="relative">
                      <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                        <Calendar className="h-5 w-5 text-slate-400" />
                      </div>
                      <input
                        id="expires_at"
                        name="expires_at"
                        type="datetime-local"
                        value={formData.expires_at}
                        onChange={handleChange}
                        className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg pl-10 pr-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                      />
                    </div>
                    <p className="mt-1 text-xs text-slate-400">
                      Leave blank for a token that never expires
                    </p>
                  </div>
                  
                  <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                    <div className="flex items-start space-x-3">
                      <AlertTriangle className="w-5 h-5 text-yellow-400 mt-0.5" />
                      <div>
                        <p className="text-white font-medium">Token Security</p>
                        <ul className="mt-2 space-y-1 text-sm text-slate-400">
                          <li>• Tokens provide full access to your account via the API</li>
                          <li>• Store tokens securely and never expose them in client-side code</li>
                          <li>• Use expiration dates for temporary access</li>
                          <li>• Delete tokens when they are no longer needed</li>
                        </ul>
                      </div>
                    </div>
                  </div>
                </form>
              )}
            </div>
            
            {!newToken && (
              <div className="p-6 border-t border-slate-700 bg-slate-800/80">
                <div className="flex items-center justify-between">
                  <button
                    onClick={() => !isSubmitting && setIsAddTokenModalOpen(false)}
                    className="px-4 py-2 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200"
                    disabled={isSubmitting}
                  >
                    Cancel
                  </button>
                  
                  <button
                    onClick={handleSubmit}
                    disabled={isSubmitting}
                    className="px-6 py-2 bg-gradient-to-r from-purple-500 to-blue-500 text-white rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
                  >
                    {isSubmitting ? (
                      <>
                        <RefreshCw className="w-4 h-4 animate-spin" />
                        <span>Creating...</span>
                      </>
                    ) : (
                      <>
                        <Key className="w-4 h-4" />
                        <span>Create Token</span>
                      </>
                    )}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default APITokens;
