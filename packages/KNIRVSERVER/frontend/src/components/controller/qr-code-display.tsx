"use client";

import React, { useState, useEffect, useRef } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useToast } from '@/hooks/use-toast';
import { 
  QrCode, 
  Smartphone, 
  RefreshCw, 
  Copy, 
  CheckCircle, 
  Clock, 
  X,
  Wifi,
  Shield
} from 'lucide-react';
import { useControllerIntegration, QRCode, QRCodeRequest } from '@/hooks/use-controller-integration';

interface QRCodeDisplayProps {
  isOpen: boolean;
  onClose: () => void;
  userId: string;
  deviceType?: string;
  capabilities?: string[];
}

export default function QRCodeDisplay({
  isOpen,
  onClose,
  userId,
  deviceType = 'desktop',
  capabilities = ['remote_control', 'file_transfer', 'screen_share']
}: QRCodeDisplayProps) {
  const { toast } = useToast();
  const [timeLeft, setTimeLeft] = useState<number>(0);
  const [copied, setCopied] = useState(false);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  const {
    qrCode,
    pairingRequest,
    activeSessions,
    isLoading,
    error,
    isConnected,
    generateQRCode,
    confirmPairing,
    clearQRCode,
    connectWebSocket
  } = useControllerIntegration();

  // Generate QR code when component opens
  useEffect(() => {
    if (isOpen && (!qrCode || qrCode.status !== 'active') && !isLoading) {
      handleGenerateQRCode();
    }
  }, [isOpen, qrCode, isLoading]);

  // Connect WebSocket for real-time updates
  useEffect(() => {
    if (isOpen) {
      connectWebSocket();
    }
  }, [isOpen, connectWebSocket]);

  // Update countdown timer
  useEffect(() => {
    if (qrCode && qrCode.status === 'active') {
      const updateTimer = () => {
        const expiresAt = new Date(qrCode.expires_at).getTime();
        const now = Date.now();
        const remaining = Math.max(0, Math.floor((expiresAt - now) / 1000));
        setTimeLeft(remaining);

        if (remaining === 0) {
          toast({
            title: "QR Code Expired",
            description: "Please generate a new QR code to continue.",
            variant: "destructive",
          });
        }
      };

      updateTimer();
      intervalRef.current = setInterval(updateTimer, 1000);

      return () => {
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
        }
      };
    }
  }, [qrCode, toast]);

  // Handle pairing request
  useEffect(() => {
    if (pairingRequest && pairingRequest.status === 'pending') {
      toast({
        title: "Pairing Request Received",
        description: `Device ${pairingRequest.mobile_device_id} wants to connect.`,
        action: (
          <div className="flex gap-2">
            <Button size="sm" onClick={() => handleConfirmPairing(true)}>
              Accept
            </Button>
            <Button size="sm" variant="outline" onClick={() => handleConfirmPairing(false)}>
              Deny
            </Button>
          </div>
        ),
      });
    }
  }, [pairingRequest]);

  const handleGenerateQRCode = async () => {
    const request: QRCodeRequest = {
      user_id: userId,
      device_type: deviceType,
      capabilities: capabilities,
    };

    const result = await generateQRCode(request);
    if (result) {
      toast({
        title: "QR Code Generated",
        description: "Scan with your mobile device to connect.",
      });
    }
  };

  const handleConfirmPairing = async (approved: boolean) => {
    if (!pairingRequest) return;

    const result = await confirmPairing(pairingRequest.id, approved);
    if (result && approved) {
      toast({
        title: "Device Connected",
        description: "Your mobile device is now connected.",
        variant: "default",
      });
      onClose();
    } else if (result && !approved) {
      toast({
        title: "Pairing Denied",
        description: "The pairing request was denied.",
        variant: "destructive",
      });
    }
  };

  const handleCopyQRData = async () => {
    if (!qrCode?.data) return;

    try {
      const qrDataString = JSON.stringify(qrCode.data);
      await navigator.clipboard.writeText(qrDataString);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
      
      toast({
        title: "QR Data Copied",
        description: "QR code data copied to clipboard.",
      });
    } catch (err) {
      toast({
        title: "Copy Failed",
        description: "Failed to copy QR code data.",
        variant: "destructive",
      });
    }
  };

  const handleRefresh = () => {
    clearQRCode();
    handleGenerateQRCode();
  };

  const formatTime = (seconds: number): string => {
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-8 pt-16 pb-16">
      <Card className="w-full max-w-md bg-background border shadow-2xl">
        <CardHeader className="text-center">
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <QrCode className="h-5 w-5" />
              Controller Pairing
            </CardTitle>
            <Button variant="ghost" size="sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
          <p className="text-sm text-muted-foreground">
            Scan this QR code with your mobile device to establish a secure connection
          </p>
        </CardHeader>

        <CardContent className="space-y-4">
          {error && (
            <div className="p-3 bg-destructive/10 border border-destructive/20 rounded-lg">
              <p className="text-sm text-destructive">{error}</p>
            </div>
          )}

          {isLoading && (
            <div className="flex items-center justify-center p-8">
              <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">Generating QR code...</span>
            </div>
          )}

          {qrCode && qrCode.status === 'active' && (
            <>
              {/* QR Code Display */}
              <div className="flex justify-center p-4 bg-white rounded-lg">
                <QRCodeSVG
                  value={JSON.stringify(qrCode.data)}
                  size={200}
                  level="M"
                  includeMargin={true}
                  fgColor="#000000"
                  bgColor="#ffffff"
                />
              </div>

              {/* Status Information */}
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary" className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {formatTime(timeLeft)}
                    </Badge>
                    <Badge variant="outline" className="flex items-center gap-1">
                      <Shield className="h-3 w-3" />
                      Secure
                    </Badge>
                  </div>
                  <div className="flex items-center gap-1">
                    <Wifi className={`h-4 w-4 ${isConnected ? 'text-green-500' : 'text-gray-400'}`} />
                    <span className="text-xs text-muted-foreground">
                      {isConnected ? 'Connected' : 'Connecting...'}
                    </span>
                  </div>
                </div>

                <div className="text-xs text-muted-foreground space-y-1">
                  <p><strong>Session ID:</strong> {qrCode.session_id.slice(0, 8)}...</p>
                  <p><strong>Device Type:</strong> {qrCode.device_type}</p>
                  <p><strong>Capabilities:</strong> {qrCode.capabilities.join(', ')}</p>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCopyQRData}
                  className="flex-1"
                >
                  {copied ? (
                    <>
                      <CheckCircle className="h-4 w-4 mr-2" />
                      Copied
                    </>
                  ) : (
                    <>
                      <Copy className="h-4 w-4 mr-2" />
                      Copy Data
                    </>
                  )}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleRefresh}
                  disabled={isLoading}
                >
                  <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                  Refresh
                </Button>
              </div>

              {/* Instructions */}
              <div className="p-3 bg-muted/50 rounded-lg">
                <div className="flex items-start gap-2">
                  <Smartphone className="h-4 w-4 mt-0.5 text-muted-foreground" />
                  <div className="text-xs text-muted-foreground">
                    <p className="font-medium mb-1">Instructions:</p>
                    <ol className="list-decimal list-inside space-y-1">
                      <li>Open KNIRVCONTROLLER app on your mobile device</li>
                      <li>Tap the QR scanner button</li>
                      <li>Point your camera at this QR code</li>
                      <li>Confirm the pairing request when prompted</li>
                    </ol>
                  </div>
                </div>
              </div>
            </>
          )}

          {qrCode && qrCode.status === 'expired' && (
            <div className="text-center p-8">
              <Clock className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <p className="text-muted-foreground mb-4">QR code has expired</p>
              <Button onClick={handleRefresh} disabled={isLoading}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Generate New Code
              </Button>
            </div>
          )}

          {qrCode && qrCode.status === 'used' && (
            <div className="text-center p-8">
              <CheckCircle className="h-12 w-12 text-green-500 mx-auto mb-4" />
              <p className="text-muted-foreground mb-4">QR code has been used successfully</p>
              <p className="text-sm text-muted-foreground">
                Active sessions: {activeSessions.length}
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
