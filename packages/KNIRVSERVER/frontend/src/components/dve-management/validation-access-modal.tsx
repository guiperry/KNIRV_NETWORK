"use client";

import React, { useState, useEffect } from 'react';
import { X, CheckCircle, AlertCircle, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useToast } from '@/hooks/use-toast';
import { useValidationSession } from '@/hooks/use-validation-session';
import ValidationInterface from './validation-interface';
import type { ValidationSession } from '@/types/api';

interface ValidationAccessModalProps {
  isOpen: boolean;
  onClose: () => void;
  creationId: string;
}

export const ValidationAccessModal: React.FC<ValidationAccessModalProps> = ({
  isOpen,
  onClose,
  creationId,
}) => {
  const { toast } = useToast();
  const validationSession = useValidationSession();
  const [session, setSession] = useState<ValidationSession | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (isOpen && creationId) {
      loadOrCreateSession();
    }
  }, [isOpen, creationId]);

  const loadOrCreateSession = async () => {
    setIsLoading(true);
    try {
      // First try to get existing session
      let existingSession = await validationSession.getSession(creationId);

      // If no existing session, create a new one
      if (!existingSession) {
        existingSession = await validationSession.createSession(creationId);
      }

      setSession(existingSession);
    } catch (error) {
      console.error('Failed to load/create validation session:', error);
      toast({
        title: "Validation Session Error",
        description: "Failed to create validation access session. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleOpenValidationInterface = () => {
    if (!session) return;

    const success = validationSession.openValidationInterface(session);
    if (success) {
      toast({
        title: "Validation Interface Opened",
        description: "Validation interface opened in a new tab.",
      });
    } else {
      toast({
        title: "Failed to Open Interface",
        description: "Could not open validation interface. Please try again.",
        variant: "destructive",
      });
    }
  };

  const handleTerminateSession = async () => {
    if (!session) return;

    try {
      const success = await validationSession.terminateSession(creationId);
      if (success) {
        toast({
          title: "Session Terminated",
          description: "Validation session has been terminated.",
        });
        setSession(null);
        onClose();
      } else {
        toast({
          title: "Termination Failed",
          description: "Failed to terminate validation session.",
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

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-gradient-to-br from-slate-900 via-blue-950 to-slate-950 rounded-lg border-2 border-blue-600/50 shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-gradient-to-r from-slate-900 to-blue-950 border-b border-blue-600/30 p-6 flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="w-10 h-10 bg-blue-600 rounded-lg flex items-center justify-center">
              <CheckCircle className="w-5 h-5 text-white" />
            </div>
            <div>
              <h2 className="text-2xl font-bold text-blue-300">Reasoning Validation</h2>
              <p className="text-sm text-slate-400">Access reasoning validation and fact-checking tools</p>
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
              <span className="ml-2 text-slate-300">Setting up validation session...</span>
            </div>
          ) : session ? (
            <>
              {/* Session Status */}
              <div className="flex items-center space-x-3">
                <Badge className="bg-green-500 text-white">
                  <CheckCircle className="w-3 h-3 mr-1" />
                  Session Active
                </Badge>
                <span className="text-sm text-slate-400">
                  Expires: {new Date(session.expires_at).toLocaleString()}
                </span>
              </div>

              {/* Tabs for Interface and Details */}
              <Tabs defaultValue="interface" className="space-y-4">
                <TabsList className="grid w-full grid-cols-2">
                  <TabsTrigger value="interface" className="flex items-center space-x-2">
                    <CheckCircle className="w-4 h-4" />
                    <span>Validation Interface</span>
                  </TabsTrigger>
                  <TabsTrigger value="details" className="flex items-center space-x-2">
                    <AlertCircle className="w-4 h-4" />
                    <span>Session Details</span>
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="interface" className="space-y-4">
                  <ValidationInterface
                    sessionId={session.id}
                    validationType={session.validation_type}
                  />
                </TabsContent>

                <TabsContent value="details" className="space-y-4">
                  {/* Session Details */}
                  <Card className="bg-slate-800/50 border border-blue-600/30">
                    <CardHeader>
                      <CardTitle className="text-lg text-blue-300">Session Information</CardTitle>
                      <CardDescription>
                        Technical details about your validation session
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="grid grid-cols-1 gap-4 text-sm">
                        <div>
                          <label className="text-slate-400 block mb-1">Session ID</label>
                          <code className="bg-slate-700 px-3 py-2 rounded text-slate-200 font-mono text-xs break-all block">
                            {session.id}
                          </code>
                        </div>
                        <div>
                          <label className="text-slate-400 block mb-1">Validation Endpoint</label>
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
                        <div>
                          <label className="text-slate-400 block mb-1">Validation Type</label>
                          <Badge variant="outline" className="text-xs">
                            {session.validation_type}
                          </Badge>
                        </div>
                      </div>

                      {/* Features */}
                      <div className="mt-4">
                        <h4 className="text-sm font-medium text-slate-300 mb-2">Available Validation Features:</h4>
                        <div className="grid grid-cols-2 gap-2 text-xs">
                          <div className="flex items-center space-x-2">
                            <CheckCircle className="w-3 h-3 text-green-500" />
                            <span className="text-slate-400">Reasoning Analysis</span>
                          </div>
                          <div className="flex items-center space-x-2">
                            <CheckCircle className="w-3 h-3 text-green-500" />
                            <span className="text-slate-400">Fact Checking</span>
                          </div>
                          <div className="flex items-center space-x-2">
                            <CheckCircle className="w-3 h-3 text-green-500" />
                            <span className="text-slate-400">Mathematical Verification</span>
                          </div>
                          <div className="flex items-center space-x-2">
                            <CheckCircle className="w-3 h-3 text-green-500" />
                            <span className="text-slate-400">Logic Validation</span>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </TabsContent>
              </Tabs>

              {/* Actions */}
              <div className="flex space-x-3">
                <Button
                  variant="outline"
                  onClick={handleTerminateSession}
                  className="border-red-500 text-red-400 hover:bg-red-500/10"
                >
                  Terminate Session
                </Button>
              </div>

              {/* Security Notice */}
              <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4">
                <div className="flex items-start space-x-3">
                  <AlertCircle className="w-5 h-5 text-yellow-500 mt-0.5" />
                  <div className="text-sm">
                    <p className="font-medium text-yellow-400 mb-1">Security Notice</p>
                    <ul className="text-slate-300 space-y-1 text-xs">
                      <li>• Validation sessions are isolated and secure</li>
                      <li>• Session tokens expire automatically</li>
                      <li>• All validation activities are logged for audit</li>
                      <li>• Sensitive data is encrypted in transit and at rest</li>
                    </ul>
                  </div>
                </div>
              </div>
            </>
          ) : (
            <div className="text-center py-8">
              <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
              <p className="text-slate-300 mb-2">Failed to create validation session</p>
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

export default ValidationAccessModal;
