'use client';

import React, { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { QRCodeSVG } from 'qrcode.react';
import { 
  Smartphone,
  Scan,
  CheckCircle2,
  Clock,
  RefreshCw,
  ChevronRight,
  AlertCircle,
  X
} from "lucide-react";
import { useToast } from "@/hooks/use-toast";

interface VerificationQRModalProps {
  isOpen: boolean;
  onClose: () => void;
  onVerified: () => void;
}

interface VerificationData {
  verificationId: string;
  sessionToken: string;
  expiresAt: number;
}

const VerificationQRModal = ({ isOpen, onClose, onVerified }: VerificationQRModalProps) => {
  const { toast } = useToast();
  const [verificationData, setVerificationData] = useState<VerificationData | null>(null);
  const [timeRemaining, setTimeRemaining] = useState(300); // 5 minutes
  const [isGenerating, setIsGenerating] = useState(false);
  const [isVerifying, setIsVerifying] = useState(false);
  const [showInstructions, setShowInstructions] = useState(true);

  // Generate verification data
  const generateVerificationData = async () => {
    setIsGenerating(true);
    try {
      const verificationId = `vfy_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const sessionToken = `sess_${Math.random().toString(36).substr(2, 16)}`;
      
      const data: VerificationData = {
        verificationId,
        sessionToken,
        expiresAt: Date.now() + (5 * 60 * 1000), // 5 minutes
      };

      setVerificationData(data);
      setTimeRemaining(300);
      setShowInstructions(true);
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to generate verification code",
        variant: "destructive",
      });
    } finally {
      setIsGenerating(false);
    }
  };

  // Countdown timer
  useEffect(() => {
    if (!isOpen || !verificationData) return;

    const interval = setInterval(() => {
      setTimeRemaining(prev => {
        if (prev <= 1) {
          clearInterval(interval);
          return 0;
        }
        return prev - 1;
      });

      // Simulate verification after 30 seconds (for demo)
      if (timeRemaining === 270) {
        simulateVerification();
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [isOpen, verificationData, timeRemaining]);

  // Generate on open
  useEffect(() => {
    if (isOpen) {
      generateVerificationData();
    }
  }, [isOpen]);

  const simulateVerification = () => {
    setIsVerifying(true);
    setTimeout(() => {
      setIsVerifying(false);
      onVerified();
    }, 2000);
  };

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const getQRCodeData = () => {
    if (!verificationData) return '';
    return JSON.stringify({
      type: 'knirv-verification',
      id: verificationData.verificationId,
      token: verificationData.sessionToken,
      action: 'verify-device'
    });
  };

  const handleDone = () => {
    setShowInstructions(false);
  };

  const handleManualVerify = () => {
    simulateVerification();
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-md bg-[#0a0a0c] border-white/10 text-slate-200">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <DialogTitle className="text-white flex items-center gap-2 text-xl">
              <Scan className="w-5 h-5 text-blue-500" />
              Verify Your Device
            </DialogTitle>
            <Badge 
              variant={timeRemaining > 60 ? "default" : "destructive"}
              className="bg-blue-500/20 text-blue-400 border-blue-500/30"
            >
              <Clock size={12} className="mr-1" />
              {timeRemaining > 0 ? formatTime(timeRemaining) : 'Expired'}
            </Badge>
          </div>
          <DialogDescription className="text-slate-400">
            Scan this QR code with your KNIRV Mobile Wallet to verify your device
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {showInstructions ? (
            // Instructions View
            <div className="space-y-4">
              <div className="p-4 bg-blue-500/10 border border-blue-500/30 rounded-xl">
                <h4 className="font-bold text-blue-400 mb-3 flex items-center">
                  <Smartphone className="w-4 h-4 mr-2" />
                  How to Verify
                </h4>
                <ol className="text-sm text-slate-300 space-y-2 list-decimal list-inside">
                  <li>Open your KNIRV Mobile Wallet app</li>
                  <li>Tap the &quot;Scan QR&quot; button</li>
                  <li>Point your camera at the blue QR code</li>
                  <li>Confirm the verification on your device</li>
                </ol>
              </div>

              <Button
                onClick={handleDone}
                className="w-full bg-blue-600 hover:bg-blue-500 text-white h-auto py-3"
              >
                Show QR Code
                <ChevronRight size={16} className="ml-2" />
              </Button>
            </div>
          ) : (
            // QR Code View
            <div className="space-y-4">
              {/* Blue QR Code */}
              <div className="flex flex-col items-center space-y-4">
                {verificationData ? (
                  <div className="relative">
                    {/* Blue glow effect */}
                    <div className="absolute inset-0 bg-blue-500 blur-xl opacity-30 rounded-xl" />
                    <div className="relative p-4 bg-blue-600 rounded-xl shadow-[0_0_30px_rgba(59,130,246,0.5)]">
                      <QRCodeSVG
                        value={getQRCodeData()}
                        size={180}
                        level="M"
                        includeMargin
                        bgColor="#2563eb"
                        fgColor="#ffffff"
                      />
                    </div>
                  </div>
                ) : (
                  <div className="w-[212px] h-[212px] bg-slate-800 rounded-xl flex items-center justify-center">
                    <RefreshCw className="w-8 h-8 text-slate-400 animate-spin" />
                  </div>
                )}

                <div className="text-center space-y-2">
                  <p className="text-sm text-blue-400 font-bold">
                    Scan with KNIRV Mobile Wallet
                  </p>
                  <p className="text-xs text-slate-500">
                    This QR code will expire in {formatTime(timeRemaining)}
                  </p>
                </div>

                {/* Verification Status */}
                {isVerifying ? (
                  <div className="flex items-center space-x-2 text-amber-400">
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    <span className="text-sm font-medium">Verifying device...</span>
                  </div>
                ) : (
                  <div className="flex items-center space-x-2 text-slate-500">
                    <AlertCircle className="w-4 h-4" />
                    <span className="text-sm">Waiting for scan...</span>
                  </div>
                )}
              </div>

              {/* Refresh Button */}
              <Button
                variant="outline"
                onClick={generateVerificationData}
                disabled={isGenerating}
                className="w-full border-white/10 hover:border-blue-500/50 text-slate-300"
              >
                <RefreshCw className={`w-4 h-4 mr-2 ${isGenerating ? 'animate-spin' : ''}`} />
                Generate New Code
              </Button>

              {/* Manual Verify (for testing) */}
              <Button
                variant="ghost"
                onClick={handleManualVerify}
                className="w-full text-slate-500 hover:text-slate-300 text-xs"
              >
                Simulate Verification (Testing)
              </Button>

              {/* Verification Info */}
              {verificationData && (
                <div className="p-3 bg-slate-800/50 rounded-lg border border-white/5">
                  <div className="text-[10px] text-slate-500 space-y-1 font-mono">
                    <div>Verification ID: <span className="text-slate-400">{verificationData.verificationId}</span></div>
                    <div>Status: <span className="text-amber-400">PENDING</span></div>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default VerificationQRModal;
