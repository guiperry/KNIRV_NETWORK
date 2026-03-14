'use client';

import React, { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { QrCode, Copy, CheckCircle, RefreshCw, Smartphone, Download } from "lucide-react";
import { QRCodeSVG } from 'qrcode.react';
import { useToast } from "@/hooks/use-toast";

// Dynamic linking system for releases
const RELEASE_LINKS = {
  pwa_url: "https://beta-controller.knirv.com/",
  android: "https://releases.knirv.network/knirvcontroller-android-pwa.zip",
  ios: "https://releases.knirv.network/knirvcontroller-ios-pwa.zip"
};

interface QRConnectionModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConnected: (appData: { url: string; name: string; type: string }) => void;
}

interface ConnectionData {
  connectionId: string;
  sessionToken: string;
  gatewayUrl: string;
  controllerUrl: string;
  expiresAt: number;
  modelConfig?: any;
}

const QRConnectionModal = ({ isOpen, onClose, onConnected }: QRConnectionModalProps) => {
  const { toast } = useToast();
  const [connectionData, setConnectionData] = useState<ConnectionData | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [isCopied, setIsCopied] = useState(false);
  const [timeRemaining, setTimeRemaining] = useState(900); // 15 minutes

  // Generate connection data
  const generateConnectionData = async () => {
    setIsGenerating(true);
    try {
      // Generate unique connection ID and session token
      const connectionId = `conn_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const sessionToken = `sess_${Math.random().toString(36).substr(2, 16)}`;
      
      const data: ConnectionData = {
        connectionId,
        sessionToken,
        gatewayUrl: window.location.origin,
        controllerUrl: 'knirvcontroller://connect', // Custom protocol for KNIRVCONTROLLER
        expiresAt: Date.now() + (15 * 60 * 1000), // 15 minutes from now
      };

      setConnectionData(data);
      setTimeRemaining(300);
    } catch (error) {
      toast({
        title: "Connection Error",
        description: "Failed to generate connection data",
        variant: "destructive",
      });
    } finally {
      setIsGenerating(false);
    }
  };

  // Generate QR code data string
  const getQRCodeData = () => {
    if (!connectionData) return '';

    // Create a mobile-friendly URL that can be opened directly on phones
    const mobileUrl = `${connectionData.gatewayUrl}/connect/${connectionData.connectionId}?token=${connectionData.sessionToken}`;

    return mobileUrl;
  };

  // Copy connection URL to clipboard
  const copyConnectionUrl = async () => {
    if (!connectionData) return;
    
    const url = `${connectionData.gatewayUrl}/connect/${connectionData.connectionId}`;
    try {
      await navigator.clipboard.writeText(url);
      setIsCopied(true);
      toast({
        title: "Copied!",
        description: "Connection URL copied to clipboard",
      });
      setTimeout(() => setIsCopied(false), 2000);
    } catch (error) {
      toast({
        title: "Copy Failed",
        description: "Failed to copy connection URL",
        variant: "destructive",
      });
    }
  };

  // Simulate connection polling (in real implementation, this would use WebSockets)
  useEffect(() => {
    if (!isOpen || !connectionData) return;

    const interval = setInterval(() => {
      setTimeRemaining(prev => {
        if (prev <= 1) {
          clearInterval(interval);
          return 0;
        }
        return prev - 1;
      });

      // Simulate connection detection (every 10 seconds)
      // Only auto-connect if the modal has been open for more than 30 seconds
      // This gives users time to scan the QR code manually
      if (timeRemaining < 270 && Math.random() < 0.1) { // 10% chance per check after 30 seconds
        onConnected({
          url: connectionData.gatewayUrl,
          name: 'KNIRV Controller',
          type: 'knirv-controller'
        });
        onClose();
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [isOpen, connectionData, onConnected, onClose]);

  // Generate new connection when modal opens
  useEffect(() => {
    if (isOpen) {
      generateConnectionData();
    }
  }, [isOpen]);

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-md bg-slate-800 border-slate-700">
        <DialogHeader>
          <DialogTitle className="text-white flex items-center gap-2">
            <Smartphone className="w-5 h-5" />
            Connect KNIRV Controller
          </DialogTitle>
          <DialogDescription className="text-slate-300">
            Scan this QR code with your KNIRV Controller app to establish a secure connection
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Connection Status */}
          <div className="flex items-center justify-between">
            <Badge variant={timeRemaining > 60 ? "default" : "destructive"} className="bg-blue-500/20 text-blue-400">
              {timeRemaining > 0 ? `Expires in ${formatTime(timeRemaining)}` : 'Expired'}
            </Badge>
            <Button
              variant="outline"
              size="sm"
              onClick={generateConnectionData}
              disabled={isGenerating}
              className="border-slate-600 text-slate-300"
            >
              <RefreshCw className={`w-4 h-4 mr-2 ${isGenerating ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
          </div>

          {/* QR Code */}
          <div className="flex flex-col items-center space-y-4">
            {connectionData ? (
              <div className="p-4 bg-white rounded-lg">
                <QRCodeSVG
                  value={getQRCodeData()}
                  size={200}
                  level="M"
                  includeMargin
                />
              </div>
            ) : (
              <div className="w-52 h-52 bg-slate-700 rounded-lg flex items-center justify-center">
                <RefreshCw className="w-8 h-8 text-slate-400 animate-spin" />
              </div>
            )}

            <div className="text-center">
              <p className="text-sm text-slate-400 mb-2">
                Scan with any QR code scanner or tap to open on mobile
              </p>
              <Button
                variant="outline"
                size="sm"
                onClick={copyConnectionUrl}
                disabled={!connectionData || isCopied}
                className="border-slate-600 text-slate-300"
              >
                {isCopied ? (
                  <CheckCircle className="w-4 h-4 mr-2 text-green-400" />
                ) : (
                  <Copy className="w-4 h-4 mr-2" />
                )}
                {isCopied ? 'Copied!' : 'Copy Connection URL'}
              </Button>
            </div>
          </div>

          {/* PWA Install */}
          <div className="space-y-3">
            <h4 className="text-white font-medium">Install KNIRV Controller</h4>
            <p className="text-sm text-slate-300">
              Open the live PWA to install directly on your device
            </p>
            <Button
              variant="outline"
              onClick={() => window.open(RELEASE_LINKS.pwa_url, '_blank')}
              className="border-slate-600 text-slate-300 hover:bg-slate-700 w-full"
            >
              <Smartphone className="w-4 h-4 mr-2" />
              Open Live PWA - Install with One Click
            </Button>
            <div className="text-xs text-slate-400 text-center">
              Alternative downloads (if PWA install doesn't work):
            </div>
            <div className="grid grid-cols-2 gap-3">
              <Button
                variant="outline"
                size="sm"
                onClick={() => window.open(RELEASE_LINKS.ios, '_blank')}
                className="border-slate-600 text-slate-300 hover:bg-slate-700"
              >
                <Download className="w-4 h-4 mr-2" />
                iOS ZIP
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => window.open(RELEASE_LINKS.android, '_blank')}
                className="border-slate-600 text-slate-300 hover:bg-slate-700"
              >
                <Download className="w-4 h-4 mr-2" />
                Android ZIP
              </Button>
            </div>
          </div>

          {/* Instructions */}
          <div className="space-y-3">
            <h4 className="text-white font-medium">How to connect:</h4>
            <ol className="text-sm text-slate-300 space-y-2 list-decimal list-inside">
              <li>Download and install KNIRV Controller app for your device</li>
              <li>Open KNIRV Controller app on your device</li>
              <li>Tap the QR scanner button</li>
              <li>Point your camera at this QR code</li>
              <li>Confirm the connection when prompted</li>
            </ol>
          </div>

          {/* Connection Info */}
          {connectionData && (
            <div className="p-3 bg-slate-700/50 rounded-lg">
              <div className="text-xs text-slate-400 space-y-1">
                <div>Connection ID: <span className="text-slate-300 font-mono">{connectionData.connectionId}</span></div>
                <div>Gateway: <span className="text-slate-300">{connectionData.gatewayUrl}</span></div>
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default QRConnectionModal;