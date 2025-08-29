import React, { useState, useEffect } from 'react';
import { 
  User, 
  Mail, 
  Key, 
  Save, 
  AlertTriangle, 
  CheckCircle, 
  RefreshCw,
  Calendar,
  Shield
} from 'lucide-react';
import { useAuth } from './AuthContext';

const UserProfile = () => {
  const { user, updateUser } = useAuth();
  const [profileData, setProfileData] = useState({
    username: '',
    email: '',
  });
  const [passwordData, setPasswordData] = useState({
    current_password: '',
    new_password: '',
    confirm_password: '',
  });
  const [profileErrors, setProfileErrors] = useState({});
  const [passwordErrors, setPasswordErrors] = useState({});
  const [isSubmittingProfile, setIsSubmittingProfile] = useState(false);
  const [isSubmittingPassword, setIsSubmittingPassword] = useState(false);
  const [profileStatus, setProfileStatus] = useState(null); // 'success', 'error', or null
  const [passwordStatus, setPasswordStatus] = useState(null); // 'success', 'error', or null
  const [profileError, setProfileError] = useState('');
  const [passwordError, setPasswordError] = useState('');

  // Initialize form data with user data
  useEffect(() => {
    if (user) {
      setProfileData({
        username: user.username || '',
        email: user.email || '',
      });
    }
  }, [user]);

  const handleProfileChange = (e) => {
    const { name, value } = e.target;
    setProfileData(prev => ({ ...prev, [name]: value }));
    
    // Clear error for this field
    if (profileErrors[name]) {
      setProfileErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
    
    // Clear status and error message
    if (profileStatus) {
      setProfileStatus(null);
      setProfileError('');
    }
  };

  const handlePasswordChange = (e) => {
    const { name, value } = e.target;
    setPasswordData(prev => ({ ...prev, [name]: value }));
    
    // Clear error for this field
    if (passwordErrors[name]) {
      setPasswordErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
    
    // Clear status and error message
    if (passwordStatus) {
      setPasswordStatus(null);
      setPasswordError('');
    }
  };

  const validateProfileForm = () => {
    const newErrors = {};
    
    if (!profileData.username.trim()) {
      newErrors.username = 'Username is required';
    }
    
    if (!profileData.email.trim()) {
      newErrors.email = 'Email is required';
    } else if (!/\S+@\S+\.\S+/.test(profileData.email)) {
      newErrors.email = 'Email is invalid';
    }
    
    setProfileErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const validatePasswordForm = () => {
    const newErrors = {};
    
    if (!passwordData.current_password.trim()) {
      newErrors.current_password = 'Current password is required';
    }
    
    if (!passwordData.new_password.trim()) {
      newErrors.new_password = 'New password is required';
    } else if (passwordData.new_password.length < 8) {
      newErrors.new_password = 'Password must be at least 8 characters';
    }
    
    if (!passwordData.confirm_password.trim()) {
      newErrors.confirm_password = 'Please confirm your new password';
    } else if (passwordData.new_password !== passwordData.confirm_password) {
      newErrors.confirm_password = 'Passwords do not match';
    }
    
    setPasswordErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleProfileSubmit = async (e) => {
    e.preventDefault();
    
    if (!validateProfileForm()) {
      return;
    }
    
    setIsSubmittingProfile(true);
    setProfileStatus(null);
    setProfileError('');
    
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/users/profile', {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(profileData),
      });
      
      const data = await response.json();
      
      if (!response.ok) {
        throw new Error(data.error || 'Failed to update profile');
      }
      
      // Update user in context
      updateUser(data.profile);
      
      setProfileStatus('success');
    } catch (error) {
      console.error('Error updating profile:', error);
      setProfileStatus('error');
      setProfileError(error.message);
    } finally {
      setIsSubmittingProfile(false);
    }
  };

  const handlePasswordSubmit = async (e) => {
    e.preventDefault();
    
    if (!validatePasswordForm()) {
      return;
    }
    
    setIsSubmittingPassword(true);
    setPasswordStatus(null);
    setPasswordError('');
    
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/users/profile/password', {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          current_password: passwordData.current_password,
          new_password: passwordData.new_password,
        }),
      });
      
      const data = await response.json();
      
      if (!response.ok) {
        throw new Error(data.error || 'Failed to update password');
      }
      
      setPasswordStatus('success');
      
      // Reset password form
      setPasswordData({
        current_password: '',
        new_password: '',
        confirm_password: '',
      });
    } catch (error) {
      console.error('Error updating password:', error);
      setPasswordStatus('error');
      setPasswordError(error.message);
    } finally {
      setIsSubmittingPassword(false);
    }
  };

  if (!user) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="w-8 h-8 border-4 border-purple-500/30 border-t-purple-500 rounded-full animate-spin"></div>
        <span className="ml-3 text-slate-400">Loading profile...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h3 className="text-lg font-semibold text-white">User Profile</h3>
      
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Profile Information */}
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <h4 className="text-white font-medium mb-4">Profile Information</h4>
          
          {profileStatus === 'success' && (
            <div className="mb-4 p-3 bg-green-500/20 border border-green-500/30 rounded-lg">
              <div className="flex items-center space-x-2">
                <CheckCircle className="w-5 h-5 text-green-400" />
                <span className="text-green-400 font-medium">Profile updated successfully</span>
              </div>
            </div>
          )}
          
          {profileStatus === 'error' && (
            <div className="mb-4 p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
              <div className="flex items-center space-x-2">
                <AlertTriangle className="w-5 h-5 text-red-400" />
                <span className="text-red-400 font-medium">{profileError || 'Failed to update profile'}</span>
              </div>
            </div>
          )}
          
          <form onSubmit={handleProfileSubmit} className="space-y-4">
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
                  value={profileData.username}
                  onChange={handleProfileChange}
                  className={`w-full bg-slate-700/50 border ${profileErrors.username ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                />
              </div>
              {profileErrors.username && (
                <p className="mt-1 text-sm text-red-400">{profileErrors.username}</p>
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
                  value={profileData.email}
                  onChange={handleProfileChange}
                  className={`w-full bg-slate-700/50 border ${profileErrors.email ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                />
              </div>
              {profileErrors.email && (
                <p className="mt-1 text-sm text-red-400">{profileErrors.email}</p>
              )}
            </div>
            
            <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
              <div className="flex items-center space-x-3 mb-2">
                <Shield className="w-4 h-4 text-slate-400" />
                <span className="text-white font-medium">Account Information</span>
              </div>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div>
                  <span className="text-slate-400">User ID:</span>
                  <p className="text-white">{user.id}</p>
                </div>
                <div>
                  <span className="text-slate-400">Role:</span>
                  <p className="text-white capitalize">{user.role}</p>
                </div>
                <div>
                  <span className="text-slate-400">Created:</span>
                  <p className="text-white">{new Date(user.created_at).toLocaleDateString()}</p>
                </div>
              </div>
            </div>
            
            <div className="flex justify-end">
              <button
                type="submit"
                disabled={isSubmittingProfile}
                className="px-6 py-2 bg-gradient-to-r from-purple-500 to-blue-500 text-white rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
              >
                {isSubmittingProfile ? (
                  <>
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    <span>Saving...</span>
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4" />
                    <span>Save Profile</span>
                  </>
                )}
              </button>
            </div>
          </form>
        </div>
        
        {/* Change Password */}
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <h4 className="text-white font-medium mb-4">Change Password</h4>
          
          {passwordStatus === 'success' && (
            <div className="mb-4 p-3 bg-green-500/20 border border-green-500/30 rounded-lg">
              <div className="flex items-center space-x-2">
                <CheckCircle className="w-5 h-5 text-green-400" />
                <span className="text-green-400 font-medium">Password updated successfully</span>
              </div>
            </div>
          )}
          
          {passwordStatus === 'error' && (
            <div className="mb-4 p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
              <div className="flex items-center space-x-2">
                <AlertTriangle className="w-5 h-5 text-red-400" />
                <span className="text-red-400 font-medium">{passwordError || 'Failed to update password'}</span>
              </div>
            </div>
          )}
          
          <form onSubmit={handlePasswordSubmit} className="space-y-4">
            <div>
              <label htmlFor="current_password" className="block text-sm font-medium text-slate-300 mb-2">
                Current Password
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <Key className="h-5 w-5 text-slate-400" />
                </div>
                <input
                  id="current_password"
                  name="current_password"
                  type="password"
                  value={passwordData.current_password}
                  onChange={handlePasswordChange}
                  className={`w-full bg-slate-700/50 border ${passwordErrors.current_password ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                  placeholder="Enter current password"
                />
              </div>
              {passwordErrors.current_password && (
                <p className="mt-1 text-sm text-red-400">{passwordErrors.current_password}</p>
              )}
            </div>
            
            <div>
              <label htmlFor="new_password" className="block text-sm font-medium text-slate-300 mb-2">
                New Password
              </label>
              <input
                id="new_password"
                name="new_password"
                type="password"
                value={passwordData.new_password}
                onChange={handlePasswordChange}
                className={`w-full bg-slate-700/50 border ${passwordErrors.new_password ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                placeholder="Enter new password"
              />
              {passwordErrors.new_password && (
                <p className="mt-1 text-sm text-red-400">{passwordErrors.new_password}</p>
              )}
            </div>
            
            <div>
              <label htmlFor="confirm_password" className="block text-sm font-medium text-slate-300 mb-2">
                Confirm New Password
              </label>
              <input
                id="confirm_password"
                name="confirm_password"
                type="password"
                value={passwordData.confirm_password}
                onChange={handlePasswordChange}
                className={`w-full bg-slate-700/50 border ${passwordErrors.confirm_password ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                placeholder="Confirm new password"
              />
              {passwordErrors.confirm_password && (
                <p className="mt-1 text-sm text-red-400">{passwordErrors.confirm_password}</p>
              )}
            </div>
            
            <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
              <div className="flex items-start space-x-3">
                <Shield className="w-5 h-5 text-slate-400 mt-0.5" />
                <div>
                  <p className="text-white font-medium">Password Requirements</p>
                  <ul className="mt-2 space-y-1 text-sm text-slate-400">
                    <li>• At least 8 characters long</li>
                    <li>• Include a mix of letters, numbers, and symbols</li>
                    <li>• Avoid using easily guessable information</li>
                  </ul>
                </div>
              </div>
            </div>
            
            <div className="flex justify-end">
              <button
                type="submit"
                disabled={isSubmittingPassword}
                className="px-6 py-2 bg-gradient-to-r from-purple-500 to-blue-500 text-white rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
              >
                {isSubmittingPassword ? (
                  <>
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    <span>Updating...</span>
                  </>
                ) : (
                  <>
                    <Key className="w-4 h-4" />
                    <span>Change Password</span>
                  </>
                )}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default UserProfile;