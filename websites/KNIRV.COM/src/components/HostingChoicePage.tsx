'use client';

import React, { useState } from 'react';
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CloudPricingModal } from "./modals/CloudPricingModal";
import { 
  Cloud,
  Download,
  Check,
  Shield,
  Globe,
  HardDrive,
  Cpu,
  ChevronRight,
  Home
} from "lucide-react";

interface HostingChoicePageProps {
  onSelectLocal?: () => void;
  onSelectCloud?: () => void;
  onReset?: () => void;
}

const localFeatures = [
  'Complete data sovereignty',
  'Local-only processing',
  'No subscription fees',
  'Self-hosted infrastructure',
  'Offline capability',
  'Full source code access',
  'Community support',
  'Manual updates required'
];

const cloudFeatures = [
  'Managed infrastructure',
  '99.9% uptime guarantee',
  'Automatic updates',
  '24/7 technical support',
  'Global CDN access',
  '99.99% data durability',
  'DDoS protection included',
  'Priority feature access'
];

export default function HostingChoicePage({ onSelectLocal, onSelectCloud, onReset }: HostingChoicePageProps) {
  const [isPricingModalOpen, setIsPricingModalOpen] = useState(false);

  const handleSelectPlan = (planName: string) => {
    setIsPricingModalOpen(false);
    if (onSelectCloud) {
      onSelectCloud();
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
          <span className="text-xl font-extrabold tracking-tighter uppercase">KNIRV <span className="text-blue-500 font-light italic">DEPLOYMENT</span></span>
        </div>
        {onReset && (
          <button
            onClick={onReset}
            className="flex items-center space-x-1 text-xs text-slate-500 hover:text-red-400 transition-colors"
            title="Return to home"
          >
            <Home size={14} />
            <span>Exit</span>
          </button>
        )}
      </nav>

      <main className="relative z-10 max-w-7xl mx-auto px-6 py-12 md:py-16">
        {/* Header Section */}
        <div className="text-center mb-12">
          <h1 className="text-4xl md:text-5xl font-extrabold tracking-tight mb-4">
            Choose Your <span className="text-blue-500">Deployment</span>
          </h1>
          <p className="text-slate-400 max-w-2xl mx-auto text-lg">
            Select how you want to deploy your Data Fabric. Both options give you complete control over your data.
          </p>
        </div>

        {/* Main Choice Cards */}
        <div className="grid md:grid-cols-2 gap-8 mb-16">
          {/* Local Sovereign Edition */}
          <Card className="bg-white/5 border-white/10 hover:border-green-500/30 transition-all group">
            <CardHeader className="pb-4">
              <div className="flex items-center justify-between mb-2">
                <div className="p-3 bg-green-600/10 rounded-lg">
                  <Download className="text-green-500" size={28} />
                </div>
                <Badge variant="outline" className="border-green-500/30 text-green-400 bg-green-500/10">
                  Free
                </Badge>
              </div>
              <CardTitle className="text-2xl text-white">Local Sovereign Edition</CardTitle>
              <p className="text-slate-400 text-sm mt-2">
                Download and run KNIRV NEXUS on your own infrastructure. Full control, zero recurring costs.
              </p>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-3">
                {localFeatures.map((feature, idx) => (
                  <div key={idx} className="flex items-center space-x-3">
                    <div className="w-5 h-5 rounded-full bg-green-500/10 flex items-center justify-center">
                      <Check size={12} className="text-green-500" />
                    </div>
                    <span className="text-slate-300 text-sm">{feature}</span>
                  </div>
                ))}
              </div>

              <div className="pt-4 border-t border-white/10">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div className="flex items-center space-x-2 text-slate-400">
                    <Cpu size={16} className="text-slate-500" />
                    <span>Self-hosted</span>
                  </div>
                  <div className="flex items-center space-x-2 text-slate-400">
                    <HardDrive size={16} className="text-slate-500" />
                    <span>Local storage</span>
                  </div>
                </div>
              </div>

              <Button
                onClick={onSelectLocal}
                className="w-full bg-green-600 hover:bg-green-500 text-white py-6 text-lg font-bold"
              >
                <Download size={20} className="mr-2" />
                Download NEXUS
              </Button>
            </CardContent>
          </Card>

          {/* Cloud Hosted */}
          <Card className="bg-white/5 border-white/10 hover:border-blue-500/30 transition-all group">
            <CardHeader className="pb-4">
              <div className="flex items-center justify-between mb-2">
                <div className="p-3 bg-blue-600/10 rounded-lg">
                  <Cloud className="text-blue-500" size={28} />
                </div>
                <Badge variant="outline" className="border-blue-500/30 text-blue-400 bg-blue-500/10">
                  Starting at $9/mo
                </Badge>
              </div>
              <CardTitle className="text-2xl text-white">Cloud Hosting</CardTitle>
              <p className="text-slate-400 text-sm mt-2">
                Let us manage the infrastructure. Focus on your data while we handle uptime and scaling.
              </p>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-3">
                {cloudFeatures.map((feature, idx) => (
                  <div key={idx} className="flex items-center space-x-3">
                    <div className="w-5 h-5 rounded-full bg-blue-500/10 flex items-center justify-center">
                      <Check size={12} className="text-blue-500" />
                    </div>
                    <span className="text-slate-300 text-sm">{feature}</span>
                  </div>
                ))}
              </div>

              <div className="pt-4 border-t border-white/10">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div className="flex items-center space-x-2 text-slate-400">
                    <Globe size={16} className="text-slate-500" />
                    <span>Global CDN</span>
                  </div>
                  <div className="flex items-center space-x-2 text-slate-400">
                    <Shield size={16} className="text-slate-500" />
                    <span>Managed security</span>
                  </div>
                </div>
              </div>

              <Button
                onClick={() => setIsPricingModalOpen(true)}
                className="w-full bg-blue-600 hover:bg-blue-500 text-white py-6 text-lg font-bold"
              >
                <Cloud size={20} className="mr-2" />
                View Cloud Plans
              </Button>
            </CardContent>
          </Card>
        </div>

        {/* Bottom CTA */}
        <div className="text-center pt-8 border-t border-white/10">
          <p className="text-slate-400 text-sm mb-4">
            Not sure which option is right for you? Start with the Local Sovereign Edition and upgrade to Cloud anytime.
          </p>
          <div className="flex items-center justify-center space-x-2 text-xs text-slate-500">
            <Shield size={14} />
            <span>All deployments include end-to-end encryption and data sovereignty guarantees</span>
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

      {/* Cloud Pricing Modal */}
      <CloudPricingModal
        isOpen={isPricingModalOpen}
        onClose={() => setIsPricingModalOpen(false)}
        onSelectPlan={handleSelectPlan}
      />
    </div>
  );
}
