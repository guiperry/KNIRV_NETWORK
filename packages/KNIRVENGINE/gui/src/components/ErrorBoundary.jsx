import React from 'react';
import { handleApiError } from '../utils/errorHandler';
import { ReloadLoadingScreen } from './MiniLoadingScreen';

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      isReloading: false,
      showNotification: false,
      errorId: null
    };
    
    this.reloadTimeout = null;
    this.notificationTimeout = null;
  }

  static getDerivedStateFromError(error) {
    // Update state so the next render will show the fallback UI
    return { hasError: true };
  }

  componentDidCatch(error, errorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    
    // Handle the error with the existing error handler system
    const errorDetails = handleApiError(error, {
      operation: 'react_render_error',
      component: this.props.componentName || 'Unknown',
      timestamp: new Date().toISOString(),
      context: 'React component rendering error',
      errorInfo: errorInfo,
      componentStack: errorInfo.componentStack,
      errorBoundary: true,
      autoReload: true
    });

    this.setState({
      error,
      errorInfo,
      errorId: errorDetails.id
    });

    // Automatically reload the view after a short delay
    this.initiateAutoReload();
  }

  initiateAutoReload = () => {
    this.setState({ isReloading: true });
    
    // Clear any existing timeout
    if (this.reloadTimeout) {
      clearTimeout(this.reloadTimeout);
    }

    // Reload after 2 seconds to give user time to see the error
    this.reloadTimeout = setTimeout(() => {
      this.reloadView();
    }, 2000);
  };

  reloadView = () => {
    // Reset the error boundary state
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
      isReloading: false,
      showNotification: true
    });

    // Show notification about the error and recovery
    this.showRecoveryNotification();
  };

  showRecoveryNotification = () => {
    // Clear any existing notification timeout
    if (this.notificationTimeout) {
      clearTimeout(this.notificationTimeout);
    }

    // Hide notification after 5 seconds
    this.notificationTimeout = setTimeout(() => {
      this.setState({ showNotification: false });
    }, 5000);
  };

  handleManualReload = () => {
    if (this.reloadTimeout) {
      clearTimeout(this.reloadTimeout);
    }
    this.reloadView();
  };

  handleDismissNotification = () => {
    this.setState({ showNotification: false });
    if (this.notificationTimeout) {
      clearTimeout(this.notificationTimeout);
    }
  };

  componentWillUnmount() {
    // Clean up timeouts
    if (this.reloadTimeout) {
      clearTimeout(this.reloadTimeout);
    }
    if (this.notificationTimeout) {
      clearTimeout(this.notificationTimeout);
    }
  }

  render() {
    // Show loading screen when reloading
    if (this.state.isReloading) {
      return <ReloadLoadingScreen message="Recovering from error..." />;
    }

    if (this.state.hasError) {
      // Error fallback UI with auto-reload functionality
      return (
        <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900 flex items-center justify-center p-6">
          <div className="bg-slate-800/90 backdrop-blur-sm rounded-xl border border-slate-700 p-8 max-w-md w-full text-center">
            <div className="mb-6">
              <div className="w-16 h-16 bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
                </svg>
              </div>
              <h2 className="text-xl font-bold text-white mb-2">Rendering Error Detected</h2>
              <p className="text-slate-400 text-sm">
                A rendering error occurred in the {this.props.componentName || 'application'}. 
                The view is being automatically reloaded.
              </p>
            </div>

            <button
              onClick={this.handleManualReload}
              className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors duration-200"
            >
              Reload View
            </button>

            {import.meta.env.VITE_NODE_ENV === 'development' && this.state.error && (
              <details className="mt-4 text-left">
                <summary className="text-slate-400 text-xs cursor-pointer hover:text-white">
                  Error Details (Development)
                </summary>
                <pre className="mt-2 text-xs text-red-400 bg-slate-900/50 p-2 rounded overflow-auto max-h-32">
                  {this.state.error.toString()}
                  {this.state.errorInfo?.componentStack}
                </pre>
              </details>
            )}
          </div>
        </div>
      );
    }

    // Render recovery notification if needed
    const children = (
      <>
        {this.props.children}
        {this.state.showNotification && (
          <RecoveryNotification
            errorId={this.state.errorId}
            componentName={this.props.componentName}
            onDismiss={this.handleDismissNotification}
          />
        )}
      </>
    );

    return children;
  }
}

// Recovery notification component
const RecoveryNotification = ({ errorId, componentName, onDismiss }) => {
  return (
    <div className="fixed top-4 right-4 z-50 max-w-sm">
      <div className="bg-slate-800 border border-slate-700 rounded-lg shadow-lg p-4">
        <div className="flex items-start space-x-3">
          <div className="flex-shrink-0">
            <div className="w-8 h-8 bg-yellow-500/20 rounded-full flex items-center justify-center">
              <svg className="w-4 h-4 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
              </svg>
            </div>
          </div>
          <div className="flex-1 min-w-0">
            <h4 className="text-sm font-medium text-white">View Recovered</h4>
            <p className="text-xs text-slate-400 mt-1">
              A rendering error was detected and automatically fixed. Check the error bell 🔔 for analysis.
            </p>
            {componentName && (
              <p className="text-xs text-slate-500 mt-1">
                Component: {componentName}
              </p>
            )}
          </div>
          <button
            onClick={onDismiss}
            className="flex-shrink-0 text-slate-400 hover:text-white transition-colors duration-200"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
};

export default ErrorBoundary;
