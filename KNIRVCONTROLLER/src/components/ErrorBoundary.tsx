import React, { Component, ErrorInfo, ReactNode } from 'react';
import {
  AlertTriangle,
  RefreshCw,
  Bug,
  ChevronDown,
  ChevronUp,
  Copy,
  Download
} from 'lucide-react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  enableRetry?: boolean;
}

interface State {
  hasError: boolean;
  error?: Error;
  errorInfo?: ErrorInfo;
  retryCount: number;
  showDetails: boolean;
  errorId: string;
  copySuccess: boolean;
}

class ErrorBoundary extends Component<Props, State> {
  private retryTimeout: number | null = null;

  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      retryCount: 0,
      showDetails: false,
      errorId: '',
      copySuccess: false
    };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    const errorId = `err_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    return {
      hasError: true,
      error,
      errorId
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    const { onError } = this.props;

    // Log error for debugging
    console.error('ErrorBoundary caught an error:', error, errorInfo);

    // Call optional error handler
    if (onError) {
      onError(error, errorInfo);
    }

    this.setState({
      error,
      errorInfo
    });

    // Report error to analytics if available
    this.reportError(error, errorInfo);
  }

  componentWillUnmount() {
    if (this.retryTimeout) {
      clearTimeout(this.retryTimeout);
    }
  }

  private reportError = (error: Error, errorInfo: ErrorInfo) => {
    // In a real implementation, this would send error reports to your analytics service
    console.log('Error reported:', {
      errorId: this.state.errorId,
      message: error.message,
      stack: error.stack,
      componentStack: errorInfo.componentStack,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent,
      url: window.location.href
    });
  };

  private handleRetry = () => {
    const { enableRetry = true } = this.props;
    const { retryCount } = this.state;

    if (!enableRetry) return;

    this.setState({ retryCount: retryCount + 1 });

    // Reset error state and try to recover
    this.setState({
      hasError: false,
      error: undefined,
      errorInfo: undefined
    });

    // Optionally implement exponential backoff for retries
    if (retryCount > 2) {
      this.retryTimeout = setTimeout(() => {
        console.log('Retrying after error recovery...');
      }, 1000 * Math.pow(2, retryCount));
    }
  };

  private toggleDetails = () => {
    this.setState(prevState => ({
      showDetails: !prevState.showDetails
    }));
  };

  private copyErrorDetails = async () => {
    const { error, errorInfo, errorId } = this.state;

    const errorDetails = {
      errorId,
      message: error?.message,
      stack: error?.stack,
      componentStack: errorInfo?.componentStack,
      timestamp: new Date().toISOString(),
      url: window.location.href
    };

    try {
      await navigator.clipboard.writeText(JSON.stringify(errorDetails, null, 2));
      this.setState({ copySuccess: true });
      setTimeout(() => this.setState({ copySuccess: false }), 2000);
    } catch (err) {
      console.error('Failed to copy error details:', err);
    }
  };

  private downloadErrorReport = () => {
    const { error, errorInfo, errorId } = this.state;

    const errorReport = {
      errorId,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent,
      url: window.location.href,
      error: {
        name: error?.name,
        message: error?.message,
        stack: error?.stack
      },
      componentStack: errorInfo?.componentStack
    };

    const blob = new Blob([JSON.stringify(errorReport, null, 2)], {
      type: 'application/json'
    });

    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `knirv-error-${errorId}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  render() {
    const { hasError, error, errorInfo, errorId, retryCount, showDetails, copySuccess } = this.state;
    const { fallback, enableRetry = true } = this.props;

    if (!hasError) {
      return this.props.children;
    }

    if (fallback) {
      return fallback;
    }

    return (
      <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
        <div className="bg-gray-900 border border-gray-700 rounded-xl w-full max-w-4xl h-[90vh] overflow-hidden">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b border-gray-700">
            <div className="flex items-center space-x-3">
              <div className="w-10 h-10 bg-red-500/20 rounded-lg flex items-center justify-center border border-red-500/20">
                <AlertTriangle className="w-5 h-5 text-red-400" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-white">Application Error</h2>
                <p className="text-sm text-gray-400">An unexpected error occurred</p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-xs text-gray-500 bg-gray-800 px-2 py-1 rounded">
                ID: {errorId}
              </span>
              <button
                onClick={() => window.location.reload()}
                className="p-2 hover:bg-gray-700 rounded-lg text-gray-400 hover:text-white transition-all"
                title="Reload application"
              >
                <RefreshCw className="w-4 h-4" />
              </button>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto p-6">
            {/* Error Summary */}
            <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-4 mb-6">
              <div className="flex items-center space-x-3 mb-3">
                <Bug className="w-5 h-5 text-red-400" />
                <h3 className="text-lg font-semibold text-white">Error Summary</h3>
              </div>
              <p className="text-gray-300 mb-2">
                {error?.message || 'An unexpected error occurred in the application.'}
              </p>
              <div className="flex items-center space-x-4 text-sm text-gray-400">
                <span>Retry attempts: {retryCount}</span>
                <span>Time: {new Date().toLocaleTimeString()}</span>
              </div>
            </div>

            {/* Quick Actions */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
              <button
                onClick={this.handleRetry}
                disabled={!enableRetry}
                className="flex items-center justify-center space-x-2 py-3 bg-blue-500/20 hover:bg-blue-500/30 border border-blue-500/30 text-blue-400 hover:text-blue-300 rounded-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <RefreshCw className="w-4 h-4" />
                <span className="text-sm">Retry</span>
              </button>

              <button
                onClick={this.toggleDetails}
                className="flex items-center justify-center space-x-2 py-3 bg-gray-500/20 hover:bg-gray-500/30 border border-gray-500/30 text-gray-400 hover:text-white rounded-lg transition-all"
              >
                {showDetails ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                <span className="text-sm">Details</span>
              </button>

              <button
                onClick={this.copyErrorDetails}
                className="flex items-center justify-center space-x-2 py-3 bg-green-500/20 hover:bg-green-500/30 border border-green-500/30 text-green-400 hover:text-green-300 rounded-lg transition-all"
              >
                <Copy className="w-4 h-4" />
                <span className="text-sm">{copySuccess ? 'Copied!' : 'Copy'}</span>
              </button>

              <button
                onClick={this.downloadErrorReport}
                className="flex items-center justify-center space-x-2 py-3 bg-purple-500/20 hover:bg-purple-500/30 border border-purple-500/30 text-purple-400 hover:text-purple-300 rounded-lg transition-all"
              >
                <Download className="w-4 h-4" />
                <span className="text-sm">Download</span>
              </button>
            </div>

            {/* Error Details (Collapsible) */}
            {showDetails && (
              <div className="space-y-4">
                <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-4">
                  <h4 className="text-md font-semibold text-white mb-3">Error Message</h4>
                  <pre className="text-sm text-red-400 bg-gray-900/50 p-3 rounded overflow-x-auto">
                    {error?.toString() || 'No error message available'}
                  </pre>
                </div>

                {error?.stack && (
                  <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-4">
                    <h4 className="text-md font-semibold text-white mb-3">Stack Trace</h4>
                    <pre className="text-sm text-gray-300 bg-gray-900/50 p-3 rounded overflow-x-auto whitespace-pre-wrap">
                      {error.stack}
                    </pre>
                  </div>
                )}

                {errorInfo?.componentStack && (
                  <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-4">
                    <h4 className="text-md font-semibold text-white mb-3">Component Stack</h4>
                    <pre className="text-sm text-gray-400 bg-gray-900/50 p-3 rounded overflow-x-auto">
                      {errorInfo.componentStack}
                    </pre>
                  </div>
                )}
              </div>
            )}

            {/* Troubleshooting Info */}
            <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 mt-6">
              <h4 className="text-md font-semibold text-white mb-3">Troubleshooting Steps</h4>
              <div className="space-y-2 text-sm text-gray-300">
                <p>• Check your internet connection and try refreshing the page</p>
                <p>• Clear your browser cache and cookies</p>
                <p>• Ensure JavaScript is enabled in your browser</p>
                <p>• Try using a different browser or incognito mode</p>
                <p>• If the problem persists, contact support with the error ID</p>
              </div>
            </div>

            {/* Footer Actions */}
            <div className="flex items-center justify-center space-x-4 mt-6 pt-6 border-t border-gray-700">
              <button
                onClick={() => window.location.reload()}
                className="px-6 py-3 bg-gradient-to-r from-red-600 to-orange-600 hover:from-red-700 hover:to-orange-700 text-white rounded-lg font-medium transition-all"
              >
                Reload Application
              </button>
              <button
                onClick={() => this.setState({ hasError: false, error: undefined, errorInfo: undefined })}
                className="px-6 py-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg font-medium transition-all"
              >
                Dismiss
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default ErrorBoundary;
