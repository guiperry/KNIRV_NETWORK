import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Shield, Home, ArrowLeft } from 'lucide-react';

const UnauthorizedPage = () => {
  const navigate = useNavigate();

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900 flex items-center justify-center p-4">
      <div className="bg-slate-800/90 backdrop-blur-sm rounded-xl border border-slate-700/50 shadow-xl p-8 max-w-md w-full">
        <div className="flex flex-col items-center text-center">
          <div className="p-4 bg-red-500/20 rounded-full border border-red-500/30 mb-6">
            <Shield className="w-12 h-12 text-red-400" />
          </div>
          
          <h1 className="text-3xl font-bold text-white mb-2">Access Denied</h1>
          <p className="text-slate-400 mb-6">
            You don't have permission to access this page. Please contact your administrator if you believe this is an error.
          </p>
          
          <div className="flex flex-col sm:flex-row space-y-3 sm:space-y-0 sm:space-x-3 w-full">
            <button
              onClick={() => navigate(-1)}
              className="flex-1 bg-slate-700/50 border border-slate-600/50 text-white py-2 px-4 rounded-lg hover:bg-slate-700 transition-all duration-200 flex items-center justify-center space-x-2"
            >
              <ArrowLeft className="w-5 h-5" />
              <span>Go Back</span>
            </button>
            
            <button
              onClick={() => navigate('/dashboard')}
              className="flex-1 bg-gradient-to-r from-purple-500 to-blue-500 text-white py-2 px-4 rounded-lg hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center justify-center space-x-2"
            >
              <Home className="w-5 h-5" />
              <span>Dashboard</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default UnauthorizedPage;