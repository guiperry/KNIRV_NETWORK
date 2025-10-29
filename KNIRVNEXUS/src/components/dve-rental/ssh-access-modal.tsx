"use client";

import React, { useState, useEffect } from 'react';
import { X, Terminal, Download, Copy, CheckCircle, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { useToast } from '@/hooks/use-toast';
import { useSSHSession } from '@/hooks/use-ssh-session';
import type { SSHSession } from '@/types/api';

interface SSHAccessModalProps {
  isOpen: boolean;
  onClose: () => void;
  rentalId: string;
}

export const SSHAccessModal: React.FC<SSHAccessModalProps> = ({
  isOpen,
  onClose,
  rentalId,
}) => {
  const { toast } = useToast();
  const sshSession = useSSHSession();
  const [session, setSession] = useState<SSHSession | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [copiedCommand, setCopiedCommand] = useState(false);

  useEffect(() => {
    if (isOpen && rentalId) {
      loadOrCreateSession();
    }
  }, [isOpen, rentalId]);

  const loadOrCreateSession = async () => {
    setIsLoading(true);
    try {
      // First try to get existing session
      let existingSession = await sshSession.getSession(rentalId);

      // If no existing session, create a new one
      if (!existingSession) {
        existingSession = await sshSession.createSession(rentalId);
      }

      setSession(existingSession);
    } catch (error) {
      console.error('Failed to load/create SSH session:', error);
      toast({
        title: "SSH Session Error",
        description: "Failed to create SSH access session. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleDownloadKey = async () => {
    if (!session) return;

    try {
      const success = await sshSession.downloadPrivateKey(session.id);
      if (success) {
        toast({
          title: "SSH Key Downloaded",
          description: "Private key has been downloaded to your device.",
        });
      } else {
        toast({
          title: "Download Failed",
          description: "Failed to download SSH private key.",
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Download Error",
        description: "An error occurred while downloading the SSH key.",
        variant: "destructive",
      });
    }
  };

  const handleCopyCommand = async () => {
    if (!session) return;

    const command = `ssh -i dve-ssh-key.pem ${session.username}@${session.endpoint} -p ${session.port}`;
    try {
      await navigator.clipboard.writeText(command);
      setCopiedCommand(true);
      toast({
        title: "Command Copied",
        description: "SSH command copied to clipboard.",
      });
      setTimeout(() => setCopiedCommand(false), 2000);
    } catch (error) {
      toast({
        title: "Copy Failed",
        description: "Failed to copy command to clipboard.",
        variant: "destructive",
      });
    }
  };

  const handleTerminateSession = async () => {
    if (!session) return;

    try {
      const success = await sshSession.terminateSession(rentalId);
      if (success) {
        toast({
          title: "Session Terminated",
          description: "SSH session has been terminated.",
        });
        setSession(null);
        onClose();
      } else {
        toast({
          title: "Termination Failed",
          description: "Failed to terminate SSH session.",
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
            <div className="w-10 h-10 bg-green-600 rounded-lg flex items-center justify-center">
              <Terminal className="w-5 h-5 text-white" />
            </div>
            <div>
              <h2 className="text-2xl font-bold text-blue-300">SSH Access</h2>
              <p className="text-sm text-slate-400">Secure Shell access to your DVE container</p>
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
              <span className="ml-2 text-slate-300">Setting up SSH session...</span>
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

              {/* Connection Details */}
              <Card className="bg-slate-800/50 border border-blue-600/30">
                <CardHeader>
                  <CardTitle className="text-lg text-blue-300">Connection Details</CardTitle>
                  <CardDescription>
                    Use these credentials to connect to your DVE container via SSH
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <label className="text-slate-400 block mb-1">Host</label>
                      <code className="bg-slate-700 px-2 py-1 rounded text-slate-200 font-mono">
                        {session.endpoint}
                      </code>
                    </div>
                    <div>
                      <label className="text-slate-400 block mb-1">Port</label>
                      <code className="bg-slate-700 px-2 py-1 rounded text-slate-200 font-mono">
                        {session.port}
                      </code>
                    </div>
                    <div>
                      <label className="text-slate-400 block mb-1">Username</label>
                      <code className="bg-slate-700 px-2 py-1 rounded text-slate-200 font-mono">
                        {session.username}
                      </code>
                    </div>
                    <div>
                      <label className="text-slate-400 block mb-1">Key File</label>
                      <code className="bg-slate-700 px-2 py-1 rounded text-slate-200 font-mono">
                        dve-ssh-key.pem
                      </code>
                    </div>
                  </div>

                  {/* SSH Command */}
                  <div>
                    <label className="text-slate-400 block mb-2">SSH Command</label>
                    <div className="bg-slate-700 p-3 rounded-lg font-mono text-sm text-slate-200 break-all">
                      ssh -i dve-ssh-key.pem {session.username}@{session.endpoint} -p {session.port}
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      className="mt-2"
                      onClick={handleCopyCommand}
                    >
                      {copiedCommand ? (
                        <>
                          <CheckCircle className="w-3 h-3 mr-1" />
                          Copied!
                        </>
                      ) : (
                        <>
                          <Copy className="w-3 h-3 mr-1" />
                          Copy Command
                        </>
                      )}
                    </Button>
                  </div>
                </CardContent>
              </Card>

              {/* Actions */}
              <div className="flex space-x-3">
                <Button
                  onClick={handleDownloadKey}
                  className="flex-1 bg-blue-600 hover:bg-blue-700"
                >
                  <Download className="w-4 h-4 mr-2" />
                  Download SSH Key
                </Button>
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
                      <li>• Keep your SSH private key secure and never share it</li>
                      <li>• The private key can only be downloaded once</li>
                      <li>• This session will automatically expire</li>
                      <li>• Monitor your SSH access logs for security</li>
                    </ul>
                  </div>
                </div>
              </div>
            </>
          ) : (
            <div className="text-center py-8">
              <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
              <p className="text-slate-300 mb-2">Failed to create SSH session</p>
              <p className="text-slate-500 text-sm">Please try again or contact support if the issue persists.</p>
              <Button
                variant="outline"
                className="mt-4"
                onClick={loadOrCreateSession}
              >
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

export default SSHAccessModal;
