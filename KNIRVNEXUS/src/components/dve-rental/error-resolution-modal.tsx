"use client";

import React, { useState, useEffect } from 'react';
import { X, Zap, ExternalLink, AlertCircle, RefreshCw, AlertTriangle, Bug, Code, Database } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { useToast } from '@/hooks/use-toast';
import { useErrorResolutionSession } from '@/hooks/use-error-resolution-session';
import type { ErrorResolutionSession } from '@/types/api';

interface ErrorResolutionModalProps {
  isOpen: boolean;
  onClose: () => void;
  rentalId: string;
}

export const ErrorResolutionModal: React.FC<ErrorResolutionModalProps> = ({
  isOpen,
  onClose,
  rentalId,
}) => {
  const { toast } = useToast();
  const errorResolutionSession = useErrorResolutionSession();
  const [session, setSession] = useState<ErrorResolutionSession | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (isOpen && rentalId) {
      loadOrCreateSession();
    }
  }, [isOpen, rentalId]);

  const loadOrCreateSession = async () => {
    setIsLoading(true);
    try {
      // First try to get existing session
      let existingSession = await errorResolutionSession.getSession(rentalId);

      // If no existing session, create a new one
      if (!existingSession) {
        existingSession = await errorResolutionSession.createSession(rentalId);
      }

      setSession(existingSession);
    } catch (error) {
      console.error('Failed to load/create error resolution session:', error);
      toast({
        title: "Error Resolution Session Error",
        description: "Failed to create error resolution access session. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleOpenErrorResolutionInterface = () => {
    if (!session) return;

    const success = errorResolutionSession.openErrorResolutionInterface(session);
    if (success) {
      toast({
        title: "Error Resolution Interface Opened",
        description: "Error resolution interface opened in a new tab.",
      });
    } else {
      toast({
        title: "Failed to Open Interface",
        description: "Could not open error resolution interface. Please try again.",
        variant: "destructive",
      });
    }
  };

  const handleTerminateSession = async () => {
    if (!session) return;

    try {
      const success = await errorResolutionSession.terminateSession(rentalId);
      if (success) {
        toast({
          title: "Session Terminated",
          description: "Error resolution session has been terminated.",
        });
        setSession(null);
        onClose();
      } else {
        toast({
          title: "Termination Failed",
          description: "Failed to terminate error resolution session.",
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Termination Error",
        description: "An error occurred while terminating the session.",
        variant: "destructive",
      });
    }
  };

  const getErrorTypeIcon = (errorType: string) => {
    switch (errorType.toLowerCase()) {
      case 'connection_timeout':
        return <AlertTriangle className="w-4 h-4 text-yellow-500" />;
      case 'validation_failed':
        return <Bug className="w-4 h-4 text-red-500" />;
      case 'resource_exhausted':
        return <Database className="w-4 h-4 text-orange-500" />;
      case 'custom_error':
        return <Code className="w-4 h-4 text-blue-500" />;
      default:
        return <AlertCircle className="w-4 h-4 text-gray-500" />;
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-gradient-to-br from-slate-900 via-blue-950 to-slate-950 rounded-lg border-2 border-blue-600/50 shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-gradient-to-r from-slate-900 to-blue-950 border-b border-blue-600/30 p-6 flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="w-10 h-10 bg-orange-600 rounded-lg flex items-center justify-center">
              <Zap className="w-5 h-5 text-white" />
            </div>
            <div>
              <h2 className="text-2xl font-bold text-blue-300">Error Resolution</h2>
              <p className="text-sm text-slate-400">Access error resolution and debugging tools</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="bg-slate-700 hover:bg-slate-600 p-2 rounded-lg transition-colors"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
              <span className="ml-2 text-slate-300">Setting up error resolution session...</span>
            </div>
          ) : session ? (
            <>
              {/* Session Status */}
              <div className="flex items-center space-x-3">
                <Badge className="bg-green-500 text-white">
                  <Zap className="w-3 h-3 mr-1" />
                  Session Active
                </Badge>
                <span className="text-sm text-slate-400">
                  Expires: {new Date(session.expires_at).toLocaleString()}
                </span>
              </div>

              {/* Error Resolution Interface */}
              <Card className="bg-slate-800/50 border border-blue-600/30">
                <CardHeader>
                  <CardTitle className="text-lg text-blue-300">Error Resolution Interface</CardTitle>
                  <CardDescription>
                    Access the error resolution environment to debug issues, analyze failures, and resolve computational errors.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-1 gap-4 text-sm">
                    <div>
                      <label className="text-slate-400 block mb-1">Error Resolution Endpoint</label>
                      <code className="bg-slate-700 px-3 py-2 rounded text-slate-200 font-mono text-xs break-all block">
                        {session.endpoint_url}
                      </code>
                    </div>
                    <div>
                      <label className="text-slate-400 block mb-1">Session Token</label>
                      <code className="bg-slate-700 px-3 py-2 rounded text-slate-200 font-mono text-xs break-all block">
                        {session.session_token}
                      </code>
                    </div>
                  </div>

                  {/* Supported Error Types */}
                  <div className="mt-4">
                    <h4 className="text-sm font-medium text-slate-300 mb-3">Supported Error Types:</h4>
                    <div className="grid grid-cols-1 gap-2">
                      {session.supported_error_types.map((errorType, index) => (
                        <div key={index} className="flex items-center space-x-3 p-2 bg-slate-700/30 rounded-lg">
                          {getErrorTypeIcon(errorType)}
                          <span className="text-sm text-slate-300 capitalize">
                            {errorType.replace('_', ' ')}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Features */}
                  <div className="mt-4">
                    <h4 className="text-sm font-medium text-slate-300 mb-2">Available Resolution Features:</h4>
                    <div className="grid grid-cols-2 gap-2 text-xs">
                      <div className="flex items-center space-x-2">
                        <AlertTriangle className="w-3 h-3 text-yellow-500" />
                        <span className="text-slate-400">Error Analysis</span>
                      </div>
                      <div className="flex items-center space-x-2">
                        <Bug className="w-3 h-3 text-red-500" />
                        <span className="text-slate-400">Debug Tools</span>
                      </div>
                      <div className="flex items-center space-x-2">
                        <Code className="w-3 h-3 text-blue-500" />
                        <span className="text-slate-400">Code Inspection</span>
                      </div>
                      <div className="flex items-center space-x-2">
                        <Database className="w-3 h-3 text-orange-500" />
                        <span className="text-slate-400">Log Analysis</span>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Actions */}
              <div className="flex space-x-3">
                <Button
                  onClick={handleOpenErrorResolutionInterface}
                  className="flex-1 bg-orange-600 hover:bg-orange-700"
                >
                  <ExternalLink className="w-4 h-4 mr-2" />
                  Open Error Resolution Interface
                </Button>
                <Button
                  variant="outline"
                  onClick={handleTerminateSession}
                  className="border-red-500 text-red-400 hover:bg-red-500/10"
                >
                  Terminate Session
                </Button>
              </div>

              {/* Usage Instructions */}
              <div className="bg-orange-500/10 border border-orange-500/30 rounded-lg p-4">
                <div className="flex items-start space-x-3">
                  <Zap className="w-5 h-5 text-orange-500 mt-0.5" />
                  <div className="text-sm">
                    <p className="font-medium text-orange-400 mb-1">How to Use Error Resolution</p>
                    <ul className="text-slate-300 space-y-1 text-xs">
                      <li>• Click "Open Error Resolution Interface" to launch the debugging environment</li>
                      <li>• Upload error logs or describe the issue</li>
                      <li>• Use automated analysis tools to identify root causes</li>
                      <li>• Access debugging utilities and performance monitors</li>
                      <li>• Generate detailed error reports and resolution steps</li>
                    </ul>
                  </div>
                </div>
              </div>

              {/* Security Notice */}
              <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4">
                <div className="flex items-start space-x-3">
                  <AlertCircle className="w-5 h-5 text-yellow-500 mt-0.5" />
                  <div className="text-sm">
                    <p className="font-medium text-yellow-400 mb-1">Security Notice</p>
                    <ul className="text-slate-300 space-y-1 text-xs">
                      <li>• Error resolution sessions are isolated and secure</li>
                      <li>• Session tokens expire automatically</li>
                      <li>• All debugging activities are logged for audit</li>
                      <li>• Sensitive error data is encrypted in transit and at rest</li>
                    </ul>
                  </div>
                </div>
              </div>
            </>
          ) : (
            <div className="text-center py-8">
              <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
              <p className="text-slate-300 mb-2">Failed to create error resolution session</p>
              <p className="text-slate-500 text-sm">Please try again or contact support if the issue persists.</p>
              <Button
                variant="outline"
                className="mt-4"
                onClick={loadOrCreateSession}
              >
                <RefreshCw className="w-4 h-4 mr-2" />
                Retry
              </Button>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="sticky bottom-0 bg-gradient-to-r from-slate-900 to-blue-950 border-t border-blue-600/30 p-4 flex justify-end space-x-2">
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </div>
  );
};

export default ErrorResolutionModal;
