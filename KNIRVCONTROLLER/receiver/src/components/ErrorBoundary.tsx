import React, { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
  errorInfo?: ErrorInfo;
}

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    this.setState({ error, errorInfo });
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div style={{
          padding: '20px',
          backgroundColor: '#1a1a1a',
          color: '#white',
          minHeight: '100vh',
          fontFamily: 'Arial, sans-serif'
        }}>
          <h1 style={{ color: '#ff4444' }}>🚨 KNIRV-CONTROLLER Error</h1>
          
          <div style={{
            backgroundColor: '#2a2a2a',
            padding: '15px',
            borderRadius: '8px',
            margin: '20px 0'
          }}>
            <h2>Something went wrong:</h2>
            <pre style={{
              backgroundColor: '#1a1a1a',
              padding: '10px',
              borderRadius: '4px',
              overflow: 'auto',
              fontSize: '14px',
              color: '#ff6666'
            }}>
              {this.state.error?.toString()}
            </pre>
          </div>

          {this.state.errorInfo && (
            <div style={{
              backgroundColor: '#2a2a2a',
              padding: '15px',
              borderRadius: '8px',
              margin: '20px 0'
            }}>
              <h3>Component Stack:</h3>
              <pre style={{
                backgroundColor: '#1a1a1a',
                padding: '10px',
                borderRadius: '4px',
                overflow: 'auto',
                fontSize: '12px',
                color: '#cccccc'
              }}>
                {this.state.errorInfo.componentStack}
              </pre>
            </div>
          )}

          <div style={{
            backgroundColor: '#2a2a2a',
            padding: '15px',
            borderRadius: '8px',
            margin: '20px 0'
          }}>
            <h3>Troubleshooting:</h3>
            <ul>
              <li>Check browser console for additional errors</li>
              <li>Verify WASM modules are loading correctly</li>
              <li>Check if TensorFlow.js is compatible with your browser</li>
              <li>Try refreshing the page</li>
            </ul>
          </div>

          <button
            onClick={() => window.location.reload()}
            style={{
              backgroundColor: '#00ff88',
              color: '#1a1a1a',
              border: 'none',
              padding: '10px 20px',
              borderRadius: '5px',
              cursor: 'pointer',
              fontSize: '16px'
            }}
          >
            Reload Page
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
