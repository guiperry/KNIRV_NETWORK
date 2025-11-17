"use client";

import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Terminal, CheckCircle, Zap, ExternalLink, AlertCircle, Play } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { useSSHSession } from '@/hooks/use-ssh-session';
import { useValidationSession } from '@/hooks/use-validation-session';
import { useErrorResolutionSession } from '@/hooks/use-error-resolution-session';
import SSHAccessModal from './ssh-access-modal';
import ValidationAccessModal from './validation-access-modal';
import ErrorResolutionModal from './error-resolution-modal';
import type { DVEAccessInfo } from '@/types/api';

interface DVEAccessFlowProps {
  rentalId: string;
  accessInfo?: DVEAccessInfo;
  className?: string;
}

export const DVEAccessFlow: React.FC<DVEAccessFlowProps> = ({
  rentalId,
  accessInfo,
  className,
}) => {
  const { toast } = useToast();
  const [activeTab, setActiveTab] = useState<string>('overview');
  const [showSSHModal, setShowSSHModal] = useState(false);
  const [showValidationModal, setShowValidationModal] = useState(false);
  const [showErrorResolutionModal, setShowErrorResolutionModal] = useState(false);

  const sshSession = useSSHSession();
  const validationSession = useValidationSession();
  const errorResolutionSession = useErrorResolutionSession();

  const handleQuickSSHAccess = async () => {
    try {
      const session = await sshSession.createSession(rentalId);
      if (session) {
        // Download key and show command
        await sshSession.downloadPrivateKey(session.id);
        const command = `ssh -i dve-ssh-key.pem ${session.username}@${session.endpoint} -p ${session.port}`;
        navigator.clipboard.writeText(command);

        toast({
          title: "SSH Access Ready",
          description: "SSH key downloaded and command copied to clipboard.",
        });
      }
    } catch (error) {
      toast({
        title: "SSH Access Failed",
        description: "Failed to set up SSH access. Please try again.",
        variant: "destructive",
      });
    }
  };

  const handleQuickValidationAccess = async () => {
    try {
      const session = await validationSession.createSession(rentalId);
      if (session) {
        validationSession.openValidationInterface(session);
      }
    } catch (error) {
      toast({
        title: "Validation Access Failed",
        description: "Failed to open validation interface. Please try again.",
        variant: "destructive",
      });
    }
  };

  const handleQuickErrorResolutionAccess = async () => {
    try {
      const session = await errorResolutionSession.createSession(rentalId);
      if (session) {
        errorResolutionSession.openErrorResolutionInterface(session);
      }
    } catch (error) {
      toast({
        title: "Error Resolution Access Failed",
        description: "Failed to open error resolution interface. Please try again.",
        variant: "destructive",
      });
    }
  };

  const getServiceStatus = (serviceType: string) => {
    if (!accessInfo) return 'unavailable';

    switch (serviceType) {
      case 'ssh':
        return accessInfo.ssh ? 'available' : 'unavailable';
      case 'validation':
        return accessInfo.reasoning_validation ? 'available' : 'unavailable';
      case 'error-resolution':
        return accessInfo.error_resolution ? 'available' : 'unavailable';
      default:
        return 'unavailable';
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'available':
        return <Badge className="bg-green-500 text-white text-xs">Available</Badge>;
      case 'unavailable':
        return <Badge className="bg-red-500 text-white text-xs">Unavailable</Badge>;
      default:
        return <Badge className="bg-yellow-500 text-white text-xs">Loading</Badge>;
    }
  };

  return (
    <div className={`space-y-6 ${className}`}>
      <Card className="knirv-card-gradient">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ExternalLink className="w-5 h-5" />
            DVE Access Portal
          </CardTitle>
          <CardDescription>
            Access your rented DVE container through SSH, validation tools, and error resolution services
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
            <TabsList className="grid w-full grid-cols-4">
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="ssh">SSH Terminal</TabsTrigger>
              <TabsTrigger value="validation">Validation</TabsTrigger>
              <TabsTrigger value="error-resolution">Error Resolution</TabsTrigger>
            </TabsList>

            <TabsContent value="overview" className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {/* SSH Access Card */}
                <Card className="bg-slate-800/50 border border-green-500/30">
                  <CardContent className="p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center space-x-3">
                        <Terminal className="w-6 h-6 text-green-500" />
                        <div>
                          <h3 className="font-medium text-slate-200">SSH Terminal</h3>
                          <p className="text-sm text-slate-400">Secure shell access</p>
                        </div>
                      </div>
                      {getStatusBadge(getServiceStatus('ssh'))}
                    </div>
                    <p className="text-xs text-slate-400 mb-3">
                      Direct terminal access to your DVE container with full command-line capabilities.
                    </p>
                    <div className="flex space-x-2">
                      <Button
                        size="sm"
                        className="flex-1 bg-green-600 hover:bg-green-700"
                        onClick={handleQuickSSHAccess}
                        disabled={getServiceStatus('ssh') !== 'available'}
                      >
                        <Play className="w-3 h-3 mr-1" />
                        Quick Access
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setShowSSHModal(true)}
                      >
                        <ExternalLink className="w-3 h-3" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>

                {/* Validation Access Card */}
                <Card className="bg-slate-800/50 border border-blue-500/30">
                  <CardContent className="p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center space-x-3">
                        <CheckCircle className="w-6 h-6 text-blue-500" />
                        <div>
                          <h3 className="font-medium text-slate-200">Reasoning Validation</h3>
                          <p className="text-sm text-slate-400">AI output validation</p>
                        </div>
                      </div>
                      {getStatusBadge(getServiceStatus('validation'))}
                    </div>
                    <p className="text-xs text-slate-400 mb-3">
                      Validate AI reasoning, check factual accuracy, and verify computational results.
                    </p>
                    <div className="flex space-x-2">
                      <Button
                        size="sm"
                        className="flex-1 bg-blue-600 hover:bg-blue-700"
                        onClick={handleQuickValidationAccess}
                        disabled={getServiceStatus('validation') !== 'available'}
                      >
                        <Play className="w-3 h-3 mr-1" />
                        Quick Access
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setShowValidationModal(true)}
                      >
                        <ExternalLink className="w-3 h-3" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>

                {/* Error Resolution Access Card */}
                <Card className="bg-slate-800/50 border border-orange-500/30">
                  <CardContent className="p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center space-x-3">
                        <Zap className="w-6 h-6 text-orange-500" />
                        <div>
                          <h3 className="font-medium text-slate-200">Error Resolution</h3>
                          <p className="text-sm text-slate-400">Debugging tools</p>
                        </div>
                      </div>
                      {getStatusBadge(getServiceStatus('error-resolution'))}
                    </div>
                    <p className="text-xs text-slate-400 mb-3">
                      Debug issues, analyze failures, and resolve computational errors.
                    </p>
                    <div className="flex space-x-2">
                      <Button
                        size="sm"
                        className="flex-1 bg-orange-600 hover:bg-orange-700"
                        onClick={handleQuickErrorResolutionAccess}
                        disabled={getServiceStatus('error-resolution') !== 'available'}
                      >
                        <Play className="w-3 h-3 mr-1" />
                        Quick Access
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setShowErrorResolutionModal(true)}
                      >
                        <ExternalLink className="w-3 h-3" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </div>

              {/* Container Info */}
              {accessInfo?.container_info && (
                <Card className="bg-slate-700/30 border border-slate-600/50">
                  <CardContent className="p-4">
                    <h4 className="font-medium text-slate-200 mb-2">Container Information</h4>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                      <div>
                        <span className="text-slate-400">Container ID:</span>
                        <p className="font-mono text-slate-200">{accessInfo.container_info.container_id.slice(-12)}</p>
                      </div>
                      <div>
                        <span className="text-slate-400">Status:</span>
                        <p className="text-slate-200 capitalize">{accessInfo.container_info.status}</p>
                      </div>
                      <div>
                        <span className="text-slate-400">CPU Cores:</span>
                        <p className="text-slate-200">{accessInfo.container_info.allocated_resources.cpu_cores}</p>
                      </div>
                      <div>
                        <span className="text-slate-400">Memory:</span>
                        <p className="text-slate-200">{accessInfo.container_info.allocated_resources.memory_gb}GB</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* Access Instructions */}
              <div className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-4">
                <div className="flex items-start space-x-3">
                  <AlertCircle className="w-5 h-5 text-blue-500 mt-0.5" />
                  <div className="text-sm">
                    <p className="font-medium text-blue-400 mb-1">Access Instructions</p>
                    <ul className="text-slate-300 space-y-1 text-xs">
                      <li>• Use "Quick Access" buttons for immediate access to services</li>
                      <li>• Click the external link icon for detailed access modals</li>
                      <li>• SSH provides terminal access, Validation offers reasoning tools</li>
                      <li>• Error Resolution provides debugging and analysis utilities</li>
                      <li>• All sessions are secured and automatically expire</li>
                    </ul>
                  </div>
                </div>
              </div>
            </TabsContent>

            <TabsContent value="ssh" className="space-y-4">
              <Card className="bg-slate-800/50 border border-green-500/30">
                <CardContent className="p-6 text-center">
                  <Terminal className="w-16 h-16 text-green-500 mx-auto mb-4" />
                  <h3 className="text-lg font-medium text-slate-200 mb-2">SSH Terminal Access</h3>
                  <p className="text-slate-400 mb-4">
                    Secure shell access to your DVE container with full command-line capabilities.
                  </p>
                  <Button
                    onClick={() => setShowSSHModal(true)}
                    className="bg-green-600 hover:bg-green-700"
                  >
                    <Terminal className="w-4 h-4 mr-2" />
                    Open SSH Access Modal
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="validation" className="space-y-4">
              <Card className="bg-slate-800/50 border border-blue-500/30">
                <CardContent className="p-6 text-center">
                  <CheckCircle className="w-16 h-16 text-blue-500 mx-auto mb-4" />
                  <h3 className="text-lg font-medium text-slate-200 mb-2">Reasoning Validation</h3>
                  <p className="text-slate-400 mb-4">
                    Validate AI outputs, check factual accuracy, and verify computational results.
                  </p>
                  <Button
                    onClick={() => setShowValidationModal(true)}
                    className="bg-blue-600 hover:bg-blue-700"
                  >
                    <CheckCircle className="w-4 h-4 mr-2" />
                    Open Validation Interface
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="error-resolution" className="space-y-4">
              <Card className="bg-slate-800/50 border border-orange-500/30">
                <CardContent className="p-6 text-center">
                  <Zap className="w-16 h-16 text-orange-500 mx-auto mb-4" />
                  <h3 className="text-lg font-medium text-slate-200 mb-2">Error Resolution</h3>
                  <p className="text-slate-400 mb-4">
                    Debug issues, analyze failures, and resolve computational errors.
                  </p>
                  <Button
                    onClick={() => setShowErrorResolutionModal(true)}
                    className="bg-orange-600 hover:bg-orange-700"
                  >
                    <Zap className="w-4 h-4 mr-2" />
                    Open Error Resolution Interface
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      {/* Access Modals */}
      <SSHAccessModal
        isOpen={showSSHModal}
        onClose={() => setShowSSHModal(false)}
        rentalId={rentalId}
      />

      <ValidationAccessModal
        isOpen={showValidationModal}
        onClose={() => setShowValidationModal(false)}
        rentalId={rentalId}
      />

      <ErrorResolutionModal
        isOpen={showErrorResolutionModal}
        onClose={() => setShowErrorResolutionModal(false)}
        rentalId={rentalId}
      />
    </div>
  );
};

export default DVEAccessFlow;
