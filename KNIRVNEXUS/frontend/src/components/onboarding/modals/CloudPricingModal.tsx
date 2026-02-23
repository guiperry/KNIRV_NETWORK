'use client';

import React from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { 
  Cloud,
  Check,
  Zap,
  Server,
  ChevronRight,
  X
} from "lucide-react";

interface CloudPricingModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectPlan: (planName: string) => void;
}

const cloudTiers = [
  {
    name: 'Starter',
    price: '$9',
    period: '/month',
    description: 'Perfect for individual developers',
    specs: { cpu: '2 vCPU', memory: '4 GB RAM', storage: '50 GB SSD' },
    features: ['1 Fabric Instance', '10,000 API calls/month', 'Basic analytics', 'Email support']
  },
  {
    name: 'Professional',
    price: '$49',
    period: '/month',
    description: 'For growing teams and startups',
    specs: { cpu: '4 vCPU', memory: '16 GB RAM', storage: '200 GB SSD' },
    features: ['5 Fabric Instances', '100,000 API calls/month', 'Advanced analytics', 'Priority support', 'Custom domains'],
    popular: true
  },
  {
    name: 'Enterprise',
    price: '$199',
    period: '/month',
    description: 'For organizations at scale',
    specs: { cpu: '16 vCPU', memory: '64 GB RAM', storage: '1 TB SSD' },
    features: ['Unlimited Instances', 'Unlimited API calls', 'Custom integrations', 'Dedicated support', 'SLA guarantee', 'Audit logs']
  }
];

export function CloudPricingModal({ isOpen, onClose, onSelectPlan }: CloudPricingModalProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-6xl bg-[#0a0a0c] border-white/10 text-slate-200 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-blue-600/20 rounded-lg">
                <Cloud className="text-blue-500" size={24} />
              </div>
              <div>
                <DialogTitle className="text-2xl font-bold text-white">Cloud Hosting Plans</DialogTitle>
                <DialogDescription className="text-slate-400">
                  Choose the perfect plan for your Data Fabric deployment
                </DialogDescription>
              </div>
            </div>
            <button
              onClick={onClose}
              className="text-slate-500 hover:text-white transition-colors p-2"
            >
              <X size={20} />
            </button>
          </div>
        </DialogHeader>

        <div className="py-6">
          {/* Pricing Cards */}
          <div className="grid md:grid-cols-3 gap-6">
            {cloudTiers.map((tier) => (
              <Card 
                key={tier.name} 
                className={`bg-white/5 border-white/10 hover:border-white/20 transition-all relative ${
                  tier.popular ? 'border-blue-500/30 shadow-[0_0_30px_rgba(59,130,246,0.1)]' : ''
                }`}
              >
                {tier.popular && (
                  <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                    <Badge className="bg-blue-600 text-white border-0">
                      <Zap size={12} className="mr-1" />
                      Most Popular
                    </Badge>
                  </div>
                )}
                <CardHeader className="pb-4">
                  <CardTitle className="text-xl text-white">{tier.name}</CardTitle>
                  <div className="mt-2">
                    <span className="text-4xl font-bold text-white">{tier.price}</span>
                    <span className="text-slate-400">{tier.period}</span>
                  </div>
                  <p className="text-slate-400 text-sm mt-2">{tier.description}</p>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="p-3 bg-slate-800/50 rounded-lg">
                    <div className="flex items-center space-x-2 text-xs text-slate-400 mb-2">
                      <Server size={14} />
                      <span>Resources</span>
                    </div>
                    <div className="text-sm text-slate-300 space-y-1">
                      <div>{tier.specs.cpu}</div>
                      <div>{tier.specs.memory}</div>
                      <div>{tier.specs.storage}</div>
                    </div>
                  </div>

                  <div className="space-y-2">
                    {tier.features.map((feature, idx) => (
                      <div key={idx} className="flex items-center space-x-2 text-sm">
                        <Check size={14} className="text-blue-500 shrink-0" />
                        <span className="text-slate-300">{feature}</span>
                      </div>
                    ))}
                  </div>

                  <Button
                    variant={tier.popular ? 'default' : 'outline'}
                    className={`w-full ${
                      tier.popular 
                        ? 'bg-blue-600 hover:bg-blue-500 text-white' 
                        : 'border-white/20 text-slate-300 hover:bg-white/5'
                    }`}
                    onClick={() => onSelectPlan(tier.name)}
                  >
                    Select {tier.name}
                    <ChevronRight size={16} className="ml-1" />
                  </Button>
                </CardContent>
              </Card>
            ))}
          </div>

          {/* Bottom Info */}
          <div className="mt-8 p-4 bg-blue-500/5 border border-blue-500/20 rounded-xl">
            <div className="flex items-start space-x-3">
              <div className="p-2 bg-blue-600/20 rounded-lg shrink-0">
                <Cloud className="text-blue-500" size={18} />
              </div>
              <div>
                <h4 className="font-bold text-white mb-1">All Cloud Plans Include</h4>
                <div className="grid md:grid-cols-2 gap-2 text-sm text-slate-400">
                  <div className="flex items-center space-x-2">
                    <Check size={14} className="text-green-500" />
                    <span>99.9% uptime guarantee</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Check size={14} className="text-green-500" />
                    <span>Automatic backups</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Check size={14} className="text-green-500" />
                    <span>DDoS protection</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Check size={14} className="text-green-500" />
                    <span>SSL certificates included</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Enterprise CTA */}
          <div className="mt-6 text-center">
            <p className="text-slate-400 text-sm">
              Need a custom solution?{' '}
              <button 
                onClick={() => onSelectPlan('Enterprise')}
                className="text-blue-400 hover:text-blue-300 underline"
              >
                Contact our sales team
              </button>
            </p>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
