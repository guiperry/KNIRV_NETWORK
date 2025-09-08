// Example integration of Error Inference System
// Shows how to integrate the error notification bell into your application

import React, { useEffect } from 'react';
import { ErrorNotificationBell } from './ErrorInferenceNotification';
import { useErrorInference, useErrorStats } from '../hooks/useErrorInference';
import { handleApiError, enableIntelligentErrorAnalysis } from '../utils/errorHandler';

// Example header component with error notification
export const AppHeaderWithErrorNotification = () => {
  const { criticalErrors, highSeverityErrors } = useErrorInference();
  const stats = useErrorStats();

  useEffect(() => {
    // Enable intelligent error analysis on app startup
    enableIntelligentErrorAnalysis(true);
  }, []);

  return (
    <header className="bg-white shadow-sm border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-16">
          {/* Logo and navigation */}
          <div className="flex items-center">
            <h1 className="text-xl font-semibold text-gray-900">
              KNIRVENGINE
            </h1>
          </div>

          {/* Right side with error notification */}
          <div className="flex items-center space-x-4">
            {/* Error statistics display */}
            {stats.total > 0 && (
              <div className="text-sm text-gray-600">
                {stats.critical > 0 && (
                  <span className="text-red-600 font-medium">
                    {stats.critical} critical
                  </span>
                )}
                {stats.high > 0 && stats.critical > 0 && <span className="mx-1">•</span>}
                {stats.high > 0 && (
                  <span className="text-yellow-600">
                    {stats.high} high
                  </span>
                )}
              </div>
            )}

            {/* Error notification bell */}
            <ErrorNotificationBell className="relative" />

            {/* User menu or other controls */}
            <div className="text-sm text-gray-500">
              User Menu
            </div>
          </div>
        </div>
      </div>
    </header>
  );
};

// Example error trigger component for testing
export const ErrorTestingComponent = () => {
  const triggerNetworkError = () => {
    const error = new Error('Failed to fetch data from server');
    error.status = 500;
    handleApiError(error, {
      operation: 'fetch_user_data',
      url: '/api/users',
      method: 'GET',
    });
  };

  const triggerValidationError = () => {
    const error = new Error('Validation failed');
    error.name = 'ValidationError';
    handleApiError(error, {
      operation: 'create_agent',
      fields: ['name', 'description'],
      values: { name: '', description: 'too long...' },
    });
  };

  const triggerAuthError = () => {
    const error = new Error('Unauthorized');
    error.status = 401;
    handleApiError(error, {
      operation: 'access_protected_resource',
      resource: '/api/agents',
    });
  };

  const triggerCriticalError = () => {
    const error = new Error('System failure: Database connection lost');
    error.status = 500;
    handleApiError(error, {
      operation: 'database_connection',
      severity: 'critical',
      component: 'database_service',
    });
  };

  return (
    <div className="p-6 bg-gray-50 rounded-lg">
      <h3 className="text-lg font-medium text-gray-900 mb-4">
        Error Testing (Development Only)
      </h3>
      <div className="grid grid-cols-2 gap-4">
        <button
          onClick={triggerNetworkError}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          Trigger Network Error
        </button>
        <button
          onClick={triggerValidationError}
          className="px-4 py-2 bg-yellow-600 text-white rounded-lg hover:bg-yellow-700"
        >
          Trigger Validation Error
        </button>
        <button
          onClick={triggerAuthError}
          className="px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700"
        >
          Trigger Auth Error
        </button>
        <button
          onClick={triggerCriticalError}
          className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
        >
          Trigger Critical Error
        </button>
      </div>
    </div>
  );
};

// Example of how to handle errors in API calls
export const ExampleAPIComponent = () => {
  const [data, setData] = React.useState(null);
  const [loading, setLoading] = React.useState(false);

  const fetchData = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/example-endpoint');
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      
      const result = await response.json();
      setData(result);
    } catch (error) {
      // Handle error with intelligent analysis
      const errorInfo = handleApiError(error, {
        operation: 'fetch_example_data',
        component: 'ExampleAPIComponent',
        endpoint: '/api/example-endpoint',
        timestamp: new Date().toISOString(),
      });
      
      console.log('Error handled with ID:', errorInfo.id);
      // The error will automatically trigger LLM analysis if it's high severity
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-4 border rounded-lg">
      <h4 className="font-medium mb-2">Example API Component</h4>
      <button
        onClick={fetchData}
        disabled={loading}
        className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
      >
        {loading ? 'Loading...' : 'Fetch Data'}
      </button>
      {data && (
        <div className="mt-2 text-sm text-gray-600">
          Data loaded successfully
        </div>
      )}
    </div>
  );
};

// Complete example layout
export const ErrorInferenceExampleLayout = () => {
  return (
    <div className="min-h-screen bg-gray-100">
      <AppHeaderWithErrorNotification />
      
      <main className="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
        <div className="space-y-6">
          <div className="bg-white p-6 rounded-lg shadow">
            <h2 className="text-xl font-semibold mb-4">
              Intelligent Error Analysis System
            </h2>
            <p className="text-gray-600 mb-4">
              This system automatically detects and analyzes errors using AI, providing
              intelligent suggestions for resolution. Click the bell icon in the header
              to see error analysis and chat with the AI assistant.
            </p>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <ErrorTestingComponent />
              <ExampleAPIComponent />
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow">
            <h3 className="text-lg font-medium mb-4">Features</h3>
            <ul className="space-y-2 text-sm text-gray-600">
              <li>• Automatic error categorization and severity assessment</li>
              <li>• AI-powered error analysis with suggested fixes</li>
              <li>• Real-time error notifications with bell indicator</li>
              <li>• Interactive chat interface for follow-up questions</li>
              <li>• Automatic retry and recovery strategies</li>
              <li>• System context collection for better analysis</li>
              <li>• Confidence scoring for suggested solutions</li>
              <li>• Integration with existing error handling workflows</li>
            </ul>
          </div>
        </div>
      </main>
    </div>
  );
};
