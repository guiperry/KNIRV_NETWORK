'use client';

import React, { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { QRCodeSVG } from 'qrcode.react';
import { 
  Smartphone,
  Download,
  Mail,
  CheckCircle2,
  Shield,
  ChevronRight,
  ChevronLeft,
  Copy,
  RefreshCw,
  Clock,
  Home
} from "lucide-react";
import { useToast } from "@/hooks/use-toast";

interface WelcomePageProps {
  onContinue: () => void;
  onBack?: () => void;
  onReset?: () => void;
}

interface ConnectionData {
  connectionId: string;
  sessionToken: string;
  downloadUrl: string;
  expiresAt: number;
}

const RELEASE_LINKS = {
  pwa_url: "https://beta-controller.knirv.com/",
  ios: "https://releases.knirv.network/knirvcontroller-ios-pwa.zip",
  android: "https://releases.knirv.network/knirvcontroller-android-pwa.zip"
};

const WelcomePage = ({ onContinue, onBack, onReset }: WelcomePageProps) => {
  const { toast } = useToast();
  const [connectionData, setConnectionData] = useState<ConnectionData | null>(null);
  const [timeRemaining, setTimeRemaining] = useState(900); // 15 minutes
  const [isCopied, setIsCopied] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);
  const [emailVerified, setEmailVerified] = useState(false);

  // Generate connection data
  const generateConnectionData = async () => {
    setIsGenerating(true);
    try {
      const connectionId = `conn_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const sessionToken = `sess_${Math.random().toString(36).substr(2, 16)}`;
      
      const data: ConnectionData = {
        connectionId,
        sessionToken,
        downloadUrl: `${window.location.origin}/download/${connectionId}`,
        expiresAt: Date.now() + (15 * 60 * 1000),
      };

      setConnectionData(data);
      setTimeRemaining(900);
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to generate download link",
        variant: "destructive",
      });
    } finally {
      setIsGenerating(false);
    }
  };

  // Countdown timer
  useEffect(() => {
    if (!connectionData) return;

    const interval = setInterval(() => {
      setTimeRemaining(prev => {
        if (prev <= 1) {
          clearInterval(interval);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [connectionData]);

  // Generate on mount
  useEffect(() => {
    generateConnectionData();
    
    // Simulate email verification after 3 seconds
    const timer = setTimeout(() => {
      setEmailVerified(true);
    }, 3000);

    return () => clearTimeout(timer);
  }, []);

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const getQRCodeData = () => {
    if (!connectionData) return '';
    return `${connectionData.downloadUrl}?token=${connectionData.sessionToken}`;
  };

  const copyDownloadUrl = async () => {
    if (!connectionData) return;
    
    try {
      await navigator.clipboard.writeText(connectionData.downloadUrl);
      setIsCopied(true);
      toast({
        title: "Copied!",
        description: "Download URL copied to clipboard",
      });
      setTimeout(() => setIsCopied(false), 2000);
    } catch (error) {
      toast({
        title: "Copy Failed",
        description: "Failed to copy URL",
        variant: "destructive",
      });
    }
  };

  return (
    <div className="min-h-screen bg-[#0a0a0c] text-slate-200 font-sans selection:bg-blue-500/30">
      {/* Background Effects */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none opacity-20">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/20 blur-[120px] rounded-full" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-indigo-600/10 blur-[120px] rounded-full" />
      </div>

      {/* Header */}
      <nav className="relative z-10 p-6 flex justify-between items-center border-b border-white/5 bg-black/40 backdrop-blur-md">
        <div className="flex items-center space-x-2">
          <div className="w-6 h-6 bg-blue-600 rounded-sm transform rotate-45" />
          <span className="text-xl font-extrabold tracking-tighter uppercase">KNIRV <span className="text-blue-500 font-light italic">WELCOME</span></span>
        </div>
        <div className="flex items-center space-x-4">
          <Badge variant="outline" className="border-green-500/30 text-green-400 bg-green-500/10">
            <span className="flex h-2 w-2 rounded-full bg-green-500 mr-2" />
            Email Verified
          </Badge>
          {onReset && (
            <button
              onClick={onReset}
              className="flex items-center space-x-1 text-xs text-slate-500 hover:text-red-400 transition-colors ml-4"
              title="Exit onboarding and return to home"
            >
              <Home size={14} />
              <span className="hidden sm:inline">Exit</span>
            </button>
          )}
        </div>
      </nav>

      <main className="relative z-10 max-w-5xl mx-auto px-6 py-12 md:py-16">
        {/* Success Message */}
        <div className="text-center mb-12">
          <div className="inline-flex items-center justify-center p-4 bg-green-500/10 rounded-full mb-6">
            <CheckCircle2 className="text-green-500" size={48} />
          </div>
          <h1 className="text-4xl md:text-5xl font-extrabold tracking-tight mb-4">
            Welcome to Your <span className="text-blue-500">Data Fabric.</span>
          </h1>
          <p className="text-slate-400 max-w-2xl mx-auto text-lg">
            Your private cloud cortex is ready. Download the mobile app to complete your setup and secure your vault.
          </p>
        </div>

        <div className="grid md:grid-cols-2 gap-12 items-start">
          {/* Left Column - Instructions */}
          <div className="space-y-6">
            {/* Email Confirmation Notice */}
            <div className="p-6 bg-green-500/5 border border-green-500/20 rounded-2xl">
              <div className="flex items-start space-x-4">
                <Mail className="text-green-500 shrink-0 mt-1" size={24} />
                <div>
                  <h3 className="font-bold text-lg mb-2 text-green-400">Email Confirmed</h3>
                  <p className="text-sm text-slate-400">
                    Your email has been verified successfully. You can now proceed with downloading 
                    the Data Wallet mobile application to complete your setup.
                  </p>
                </div>
              </div>
            </div>

            {/* Download Options */}
            <div className="p-6 bg-white/5 border border-white/10 rounded-2xl">
              <h3 className="font-bold text-lg mb-4 flex items-center">
                <Download className="mr-2 text-blue-500" size={20} />
                Download Options
              </h3>
              
              <div className="space-y-3">
                <Button
                  variant="outline"
                  onClick={() => window.open(RELEASE_LINKS.pwa_url, '_blank')}
                  className="w-full justify-start border-white/10 hover:border-blue-500/50 hover:bg-white/5 h-auto py-4"
                >
                  <Smartphone className="mr-3 text-blue-500" size={20} />
                  <div className="text-left">
                    <div className="font-bold">Open Live PWA</div>
                    <div className="text-xs text-slate-500">Install directly on your device</div>
                  </div>
                </Button>

                <div className="grid grid-cols-2 gap-3">
                  <Button
                    variant="outline"
                    onClick={() => window.open(RELEASE_LINKS.ios, '_blank')}
                    className="border-white/10 hover:border-blue-500/50 hover:bg-white/5 h-auto py-3"
                  >
                    <Download size={16} className="mr-2" />
                    iOS ZIP
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => window.open(RELEASE_LINKS.android, '_blank')}
                    className="border-white/10 hover:border-blue-500/50 hover:bg-white/5 h-auto py-3"
                  >
                    <Download size={16} className="mr-2" />
                    Android
                  </Button>
                </div>
              </div>
            </div>

            {/* Setup Steps */}
            <div className="p-6 bg-white/5 border border-white/10 rounded-2xl">
              <h3 className="font-bold text-lg mb-4">Next Steps</h3>
              <div className="space-y-4">
                {[
                  { step: 1, text: 'Download the KNIRV Mobile Wallet app' },
                  { step: 2, text: 'Install and open the application' },
                  { step: 3, text: 'Scan the QR code to pair your device' },
                  { step: 4, text: 'Complete biometric authorization' }
                ].map((item) => (
                  <div key={item.step} className="flex items-center space-x-3">
                    <div className="w-6 h-6 rounded-full bg-blue-600 flex items-center justify-center text-xs font-bold">
                      {item.step}
                    </div>
                    <span className="text-sm text-slate-300">{item.text}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Right Column - QR Code */}
          <div className="flex flex-col items-center">
            <div className="bg-white/5 border border-white/10 p-8 rounded-2xl w-full">
              <div className="flex items-center justify-between mb-6">
                <h3 className="font-bold flex items-center">
                  <Shield className="mr-2 text-blue-500" size={20} />
                  Secure Download
                </h3>
                <Badge 
                  variant={timeRemaining > 60 ? "default" : "destructive"}
                  className="bg-blue-500/20 text-blue-400 border-blue-500/30"
                >
                  <Clock size={12} className="mr-1" />
                  {timeRemaining > 0 ? formatTime(timeRemaining) : 'Expired'}
                </Badge>
              </div>

              {/* QR Code */}
              <div className="flex flex-col items-center space-y-4 mb-6">
                {connectionData ? (
                  <div className="p-4 bg-white rounded-xl shadow-[0_0_50px_rgba(59,130,246,0.2)]">
                    <QRCodeSVG
                      value={getQRCodeData()}
                      size={200}
                      level="M"
                      includeMargin
                    />
                  </div>
                ) : (
                  <div className="w-[232px] h-[232px] bg-slate-800 rounded-xl flex items-center justify-center">
                    <RefreshCw className="w-8 h-8 text-slate-400 animate-spin" />
                  </div>
                )}

                <p className="text-sm text-slate-400 text-center">
                  Scan with your mobile device to download
                </p>

                {/* Copy URL Button */}
                <Button
                  variant="outline"
                  size="sm"
                  onClick={copyDownloadUrl}
                  disabled={!connectionData || isCopied}
                  className="border-white/10 hover:border-blue-500/50"
                >
                  {isCopied ? (
                    <>
                      <CheckCircle2 size={14} className="mr-2 text-green-400" />
                      Copied!
                    </>
                  ) : (
                    <>
                      <Copy size={14} className="mr-2" />
                      Copy Download URL
                    </>
                  )}
                </Button>
              </div>

              {/* Refresh Button */}
              <Button
                variant="outline"
                onClick={generateConnectionData}
                disabled={isGenerating}
                className="w-full border-white/10 hover:border-blue-500/50"
              >
                <RefreshCw className={`w-4 h-4 mr-2 ${isGenerating ? 'animate-spin' : ''}`} />
                Generate New QR Code
              </Button>

              {/* Connection Info */}
              {connectionData && (
                <div className="mt-6 pt-6 border-t border-white/10">
                  <div className="text-xs text-slate-500 space-y-1 font-mono">
                    <div>Session ID: <span className="text-slate-400">{connectionData.connectionId}</span></div>
                    <div>Gateway: <span className="text-slate-400">{window.location.origin}</span></div>
                  </div>
                </div>
              )}
            </div>

            {/* Navigation Buttons */}
            <div className="mt-8 flex flex-col sm:flex-row gap-4 w-full">
              {onBack && (
                <Button
                  variant="ghost"
                  onClick={onBack}
                  className="text-slate-500 hover:text-white transition-colors flex items-center justify-center space-x-2 text-sm font-bold uppercase tracking-widest"
                >
                  <ChevronLeft size={16} />
                  <span>Back</span>
                </Button>
              )}
              <Button
                onClick={onContinue}
                className="flex-1 group bg-blue-600 hover:bg-blue-500 text-white px-8 py-4 rounded-xl font-bold transition-all transform active:scale-95 flex items-center justify-center space-x-3 h-auto"
              >
                <span>I&apos;ve Downloaded the App</span>
                <ChevronRight size={18} className="group-hover:translate-x-1 transition-transform" />
              </Button>
            </div>
          </div>
        </div>
      </main>

      {/* Footer Meta */}
      <footer className="max-w-7xl mx-auto p-12 text-center">
        <div className="inline-flex items-center space-x-4 px-6 py-2 rounded-full border border-white/5 bg-white/5 text-[10px] mono text-slate-500 font-bold uppercase tracking-[0.2em]">
          <span className="flex h-2 w-2 rounded-full bg-green-500 animate-pulse" />
          <span>Nexus Network Secure</span>
          <span className="h-3 w-[1px] bg-white/10" />
          <span>Sovereign Encryption Active</span>
        </div>
      </footer>
    </div>
  );
};

export default WelcomePage;
