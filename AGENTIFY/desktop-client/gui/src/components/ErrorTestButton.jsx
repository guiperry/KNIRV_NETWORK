import React from 'react';
import { handleApiError } from '../utils/errorHandler';

export const ErrorTestButton = () => {
  const triggerTestError = () => {
    console.log('Triggering test error...');
    
    // Create a test error
    const testError = new Error('Test error for AI Error Engine');
    
    // Handle it with the error handler
    const errorInfo = handleApiError(testError, {
      operation: 'test_error',
      component: 'ErrorTestButton',
      timestamp: new Date().toISOString(),
      context: 'Manual test error triggered by user'
    });
    
    console.log('Error handled with ID:', errorInfo.id);
    console.log('Error info:', errorInfo);
  };

  const triggerNetworkError = () => {
    console.log('Triggering network error...');
    
    // Simulate a network error
    const networkError = new TypeError('Failed to fetch');
    networkError.code = 'NETWORK_ERROR';
    
    const errorInfo = handleApiError(networkError, {
      operation: 'connect_service',
      component: 'WebConnections',
      service: 'AuthKit',
      timestamp: new Date().toISOString(),
      context: 'Simulated network error for testing'
    });
    
    console.log('Network error handled with ID:', errorInfo.id);
  };

  return (
    <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/50">
      <h3 className="text-white font-medium mb-3">🧪 Error Engine Test</h3>
      <div className="space-y-2">
        <button
          onClick={triggerTestError}
          className="w-full px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
        >
          Trigger Test Error
        </button>
        <button
          onClick={triggerNetworkError}
          className="w-full px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
        >
          Trigger Network Error
        </button>
      </div>
      <p className="text-slate-400 text-sm mt-2">
        Click these buttons to test if errors appear in the notification bell.
      </p>
    </div>
  );
};
