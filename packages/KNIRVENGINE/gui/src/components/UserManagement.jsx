import React, { useState, useEffect } from 'react';
import { 
  User, 
  UserPlus, 
  Edit, 
  Trash2, 
  Search, 
  Shield, 
  Mail, 
  Calendar, 
  CheckCircle, 
  XCircle,
  AlertTriangle,
  RefreshCw
} from 'lucide-react';
import { useAuth } from './AuthContext';

// KNIRVENGINE API configuration. Keep engine traffic local even when the UI
// was opened from a file URL or another KNIRV site.
const getApiBaseUrl = () => {
  return (import.meta.env.VITE_API_BASE_URL || (window.location.protocol === 'file:' ? 'http://localhost:8081' : '')).replace(/\/$/, '');
};

const UserManagement = () => {
  const [users, setUsers] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isAddUserModalOpen, setIsAddUserModalOpen] = useState(false);
  const [isEditUserModalOpen, setIsEditUserModalOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState(null);
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    role: 'user'
  });
  const [formErrors, setFormErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null
  const [showLoginPrompt, setShowLoginPrompt] = useState(false);
  const { user: currentUser, isAuthenticated } = useAuth();

  // Fetch users when authenticated
  useEffect(() => {
    if (isAuthenticated) {
      fetchUsers();
    }
  }, [isAuthenticated]);

  const fetchUsers = async () => {
    if (!isAuthenticated) {
      setError('Authentication required to access user management');
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const token = localStorage.getItem('token');
      console.log('Fetching users with token:', token ? 'present' : 'missing');

      if (!token) {
        throw new Error('No authentication token found');
      }

      const apiBaseUrl = getApiBaseUrl();
      const response = await fetch(`${apiBaseUrl}/api/v1/users`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

      console.log('Response status:', response.status);

      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('Authentication failed. Please log in again.');
        }
        const errorText = await response.text();
        console.error('Response error text:', errorText);
        throw new Error(`Failed to fetch users: ${response.status} ${response.statusText}`);
      }

      const data = await response.json();
      console.log('Users data:', data);
      setUsers(data.users || []);
    } catch (error) {
      console.error('Error fetching users:', error);
      setError(error.message);

      // If authentication failed, show login prompt
      if (error.message.includes('Authentication failed') || error.message.includes('No authentication token')) {
        setShowLoginPrompt(true);
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleAddUser = () => {
    setFormData({
      username: '',
      email: '',
      password: '',
      role: 'user'
    });
    setFormErrors({});
    setSubmitStatus(null);
    setIsAddUserModalOpen(true);
  };

  const handleEditUser = (user) => {
    setSelectedUser(user);
    setFormData({
      username: user.username,
      email: user.email,
      password: '', // Don't include password for edit
      role: user.role
    });
    setFormErrors({});
    setSubmitStatus(null);
    setIsEditUserModalOpen(true);
  };

  const handleDeleteUser = async (userId) => {
    if (!confirm('Are you sure you want to delete this user?')) {
      return;
    }
    
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`/api/v1/users/${userId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to delete user');
      }
      
      // Remove user from state
      setUsers(users.filter(user => user.id !== userId));
    } catch (error) {
      console.error('Error deleting user:', error);
      alert('Failed to delete user: ' + error.message);
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
    
    if (!formData.username.trim()) {
      newErrors.username = 'Username is required';
    }
    
    if (!formData.email.trim()) {
      newErrors.email = 'Email is required';
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      newErrors.email = 'Email is invalid';
    }
    
    if (!isEditUserModalOpen && !formData.password.trim()) {
      newErrors.password = 'Password is required';
    } else if (!isEditUserModalOpen && formData.password.length < 8) {
      newErrors.password = 'Password must be at least 8 characters';
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
      const method = isEditUserModalOpen ? 'PUT' : 'POST';
      const apiBaseUrl = getApiBaseUrl();
      const url = isEditUserModalOpen ? `${apiBaseUrl}/api/v1/users/${selectedUser.id}` : `${apiBaseUrl}/api/v1/users`;
      
      // For edit, don't include password if it's empty
      const requestData = { ...formData };
      if (isEditUserModalOpen && !requestData.password) {
        delete requestData.password;
      }
      
      const response = await fetch(url, {
        method,
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestData),
      });
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Operation failed');
      }
      
      const data = await response.json();
      
      // Update users list
      if (isEditUserModalOpen) {
        setUsers(users.map(user => user.id === selectedUser.id ? data.user : user));
      } else {
        setUsers([...users, data.user]);
      }
      
      setSubmitStatus('success');
      
      // Close modal after success
      setTimeout(() => {
        if (isEditUserModalOpen) {
          setIsEditUserModalOpen(false);
        } else {
          setIsAddUserModalOpen(false);
        }
        setSubmitStatus(null);
      }, 1500);
      
    } catch (error) {
      console.error('Error saving user:', error);
      setSubmitStatus('error');
    } finally {
      setIsSubmitting(false);
    }
  };

  const filteredUsers = users.filter(user => 
    user.username.toLowerCase().includes(searchTerm.toLowerCase()) ||
    user.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
    user.role.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <h3 className="text-lg font-semibold text-white">User Management</h3>
        <div className="mt-4 lg:mt-0 flex items-center space-x-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-4 h-4" />
            <input
              type="text"
              placeholder="Search users..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="bg-slate-700/50 border border-slate-600/50 rounded-lg pl-9 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 w-full lg:w-64"
            />
          </div>
          <button
            onClick={handleAddUser}
            className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-4 py-2 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2"
          >
            <UserPlus className="w-4 h-4" />
            <span>Add User</span>
          </button>
        </div>
      </div>

      {/* Authentication prompt */}
      {showLoginPrompt && !isAuthenticated ? (
        <div className="max-w-md mx-auto bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
          <div className="text-center mb-6">
            <AlertCircle className="w-12 h-12 text-yellow-500 mx-auto mb-4" />
            <h3 className="text-xl font-semibold text-white mb-2">Authentication Required</h3>
            <p className="text-slate-400">
              User management requires authentication. Please log in with admin credentials.
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
        </div>
      ) : isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="w-8 h-8 border-4 border-purple-500/30 border-t-purple-500 rounded-full animate-spin"></div>
          <span className="ml-3 text-slate-400">Loading users...</span>
        </div>
      ) : error ? (
        <div className="p-4 bg-red-500/20 border border-red-500/30 rounded-lg">
          <div className="flex items-start space-x-3">
            <AlertTriangle className="w-5 h-5 text-red-400 mt-0.5" />
            <div>
              <p className="text-red-400 font-medium">Error loading users</p>
              <p className="text-red-300/70 text-sm">{error}</p>
              <button
                onClick={fetchUsers}
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
                  <th className="text-left py-3 px-6 text-slate-300 font-medium">User</th>
                  <th className="text-left py-3 px-6 text-slate-300 font-medium">Email</th>
                  <th className="text-left py-3 px-6 text-slate-300 font-medium">Role</th>
                  <th className="text-left py-3 px-6 text-slate-300 font-medium">Created</th>
                  <th className="text-left py-3 px-6 text-slate-300 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700/50">
                {filteredUsers.length > 0 ? (
                  filteredUsers.map((user) => (
                    <tr key={user.id} className="hover:bg-slate-700/20 transition-colors duration-200">
                      <td className="py-4 px-6">
                        <div className="flex items-center space-x-3">
                          <div className="p-2 bg-slate-700/50 rounded-full">
                            <User className="w-5 h-5 text-slate-300" />
                          </div>
                          <span className="text-white font-medium">{user.username}</span>
                        </div>
                      </td>
                      <td className="py-4 px-6 text-slate-300">{user.email}</td>
                      <td className="py-4 px-6">
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                          user.role === 'admin' 
                            ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30' 
                            : 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                        }`}>
                          {user.role}
                        </span>
                      </td>
                      <td className="py-4 px-6 text-slate-300">
                        {new Date(user.created_at).toLocaleDateString()}
                      </td>
                      <td className="py-4 px-6">
                        <div className="flex items-center space-x-2">
                          <button
                            onClick={() => handleEditUser(user)}
                            className="p-1 text-slate-400 hover:text-white transition-colors duration-200"
                            title="Edit user"
                          >
                            <Edit className="w-4 h-4" />
                          </button>
                          {user.id !== currentUser?.id && (
                            <button
                              onClick={() => handleDeleteUser(user.id)}
                              className="p-1 text-slate-400 hover:text-red-400 transition-colors duration-200"
                              title="Delete user"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={5} className="py-8 text-center text-slate-400">
                      {searchTerm ? 'No users match your search criteria' : 'No users found'}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
      
      {/* Add User Modal */}
      {isAddUserModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div 
            className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
            onClick={() => !isSubmitting && setIsAddUserModalOpen(false)}
          ></div>
          
          <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-md max-h-[90vh] overflow-hidden">
            <div className="flex items-center justify-between p-6 border-b border-slate-700">
              <div className="flex items-center space-x-3">
                <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
                  <UserPlus className="w-5 h-5 text-purple-400" />
                </div>
                <h2 className="text-xl font-bold text-white">Add User</h2>
              </div>
              <button
                onClick={() => !isSubmitting && setIsAddUserModalOpen(false)}
                aria-label="Close"
                className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
                disabled={isSubmitting}
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            
            <div className="p-6 overflow-y-auto max-h-[calc(90vh-140px)]">
              <form onSubmit={handleSubmit} className="space-y-4">
                {submitStatus === 'error' && (
                  <div className="p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
                    <div className="flex items-center space-x-2">
                      <AlertTriangle className="w-5 h-5 text-red-400" />
                      <span className="text-red-400 font-medium">Failed to create user</span>
                    </div>
                  </div>
                )}
                
                {submitStatus === 'success' && (
                  <div className="p-3 bg-green-500/20 border border-green-500/30 rounded-lg">
                    <div className="flex items-center space-x-2">
                      <CheckCircle className="w-5 h-5 text-green-400" />
                      <span className="text-green-400 font-medium">User created successfully</span>
                    </div>
                  </div>
                )}
                
                <div>
                  <label htmlFor="username" className="block text-sm font-medium text-slate-300 mb-2">
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
                      className={`w-full bg-slate-700/50 border ${formErrors.username ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                      placeholder="Enter username"
                    />
                  </div>
                  {formErrors.username && (
                    <p className="mt-1 text-sm text-red-400">{formErrors.username}</p>
                  )}
                </div>
                
                <div>
                  <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-2">
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
                      className={`w-full bg-slate-700/50 border ${formErrors.email ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                      placeholder="Enter email address"
                    />
                  </div>
                  {formErrors.email && (
                    <p className="mt-1 text-sm text-red-400">{formErrors.email}</p>
                  )}
                </div>
                
                <div>
                  <label htmlFor="password" className="block text-sm font-medium text-slate-300 mb-2">
                    Password
                  </label>
                  <input
                    id="password"
                    name="password"
                    type="password"
                    value={formData.password}
                    onChange={handleChange}
                    className={`w-full bg-slate-700/50 border ${formErrors.password ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                    placeholder="Enter password"
                  />
                  {formErrors.password && (
                    <p className="mt-1 text-sm text-red-400">{formErrors.password}</p>
                  )}
                </div>
                
                <div>
                  <label htmlFor="role" className="block text-sm font-medium text-slate-300 mb-2">
                    Role
                  </label>
                  <div className="relative">
                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                      <Shield className="h-5 w-5 text-slate-400" />
                    </div>
                    <select
                      id="role"
                      name="role"
                      value={formData.role}
                      onChange={handleChange}
                      className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg pl-10 pr-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                    >
                      <option value="user">User</option>
                      <option value="admin">Admin</option>
                    </select>
                  </div>
                </div>
              </form>
            </div>
            
            <div className="p-6 border-t border-slate-700 bg-slate-800/80">
              <div className="flex items-center justify-between">
                <button
                  onClick={() => !isSubmitting && setIsAddUserModalOpen(false)}
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
                      <UserPlus className="w-4 h-4" />
                      <span>Create User</span>
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
      
      {/* Edit User Modal */}
      {isEditUserModalOpen && selectedUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div 
            className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
            onClick={() => !isSubmitting && setIsEditUserModalOpen(false)}
          ></div>
          
          <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-md max-h-[90vh] overflow-hidden">
            <div className="flex items-center justify-between p-6 border-b border-slate-700">
              <div className="flex items-center space-x-3">
                <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
                  <Edit className="w-5 h-5 text-purple-400" />
                </div>
                <h2 className="text-xl font-bold text-white">Edit User</h2>
              </div>
              <button
                onClick={() => !isSubmitting && setIsEditUserModalOpen(false)}
                aria-label="Close"
                className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
                disabled={isSubmitting}
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            
            <div className="p-6 overflow-y-auto max-h-[calc(90vh-140px)]">
              <form onSubmit={handleSubmit} className="space-y-4">
                {submitStatus === 'error' && (
                  <div className="p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
                    <div className="flex items-center space-x-2">
                      <AlertTriangle className="w-5 h-5 text-red-400" />
                      <span className="text-red-400 font-medium">Failed to update user</span>
                    </div>
                  </div>
                )}
                
                {submitStatus === 'success' && (
                  <div className="p-3 bg-green-500/20 border border-green-500/30 rounded-lg">
                    <div className="flex items-center space-x-2">
                      <CheckCircle className="w-5 h-5 text-green-400" />
                      <span className="text-green-400 font-medium">User updated successfully</span>
                    </div>
                  </div>
                )}
                
                <div>
                  <label htmlFor="username" className="block text-sm font-medium text-slate-300 mb-2">
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
                      className={`w-full bg-slate-700/50 border ${formErrors.username ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                      placeholder="Enter username"
                    />
                  </div>
                  {formErrors.username && (
                    <p className="mt-1 text-sm text-red-400">{formErrors.username}</p>
                  )}
                </div>
                
                <div>
                  <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-2">
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
                      className={`w-full bg-slate-700/50 border ${formErrors.email ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                      placeholder="Enter email address"
                    />
                  </div>
                  {formErrors.email && (
                    <p className="mt-1 text-sm text-red-400">{formErrors.email}</p>
                  )}
                </div>
                
                <div>
                  <label htmlFor="password" className="block text-sm font-medium text-slate-300 mb-2">
                    Password (leave blank to keep current)
                  </label>
                  <input
                    id="password"
                    name="password"
                    type="password"
                    value={formData.password}
                    onChange={handleChange}
                    className={`w-full bg-slate-700/50 border ${formErrors.password ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                    placeholder="Enter new password (optional)"
                  />
                  {formErrors.password && (
                    <p className="mt-1 text-sm text-red-400">{formErrors.password}</p>
                  )}
                </div>
                
                <div>
                  <label htmlFor="role" className="block text-sm font-medium text-slate-300 mb-2">
                    Role
                  </label>
                  <div className="relative">
                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                      <Shield className="h-5 w-5 text-slate-400" />
                    </div>
                    <select
                      id="role"
                      name="role"
                      value={formData.role}
                      onChange={handleChange}
                      className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg pl-10 pr-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                      disabled={selectedUser?.id === currentUser?.id} // Prevent changing own role
                    >
                      <option value="user">User</option>
                      <option value="admin">Admin</option>
                    </select>
                  </div>
                  {selectedUser?.id === currentUser?.id && (
                    <p className="mt-1 text-xs text-slate-400">You cannot change your own role</p>
                  )}
                </div>
                
                <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                  <div className="flex items-center space-x-3 mb-2">
                    <Calendar className="w-4 h-4 text-slate-400" />
                    <span className="text-white font-medium">User Information</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div>
                      <span className="text-slate-400">User ID:</span>
                      <p className="text-white">{selectedUser?.id}</p>
                    </div>
                    <div>
                      <span className="text-slate-400">Created:</span>
                      <p className="text-white">{new Date(selectedUser?.created_at).toLocaleDateString()}</p>
                    </div>
                  </div>
                </div>
              </form>
            </div>
            
            <div className="p-6 border-t border-slate-700 bg-slate-800/80">
              <div className="flex items-center justify-between">
                <button
                  onClick={() => !isSubmitting && setIsEditUserModalOpen(false)}
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
                      <span>Updating...</span>
                    </>
                  ) : (
                    <>
                      <CheckCircle className="w-4 h-4" />
                      <span>Save Changes</span>
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default UserManagement;
