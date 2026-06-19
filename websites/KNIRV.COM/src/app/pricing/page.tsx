'use client';

import React from 'react';
import { Server, Zap, Shield, Crown } from 'lucide-react';
import KnirvLogo from '@/components/KnirvLogo'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import Link from 'next/link';

export default function Pricing() {
  const plans = [
    {
      name: "Free",
      price: "Free",
      description: "Sovereign DVE management with blockchain-native policy controls",
      features: [
        "1 free DVE (Distributed Validation Environment)",
        "DVE Management dashboard",
        "Policy Editor — define governance rules",
        "Blockchain Commit — immutable audit trail",
        "KNIRVCONTROLLER NRN Wallet",
        "Community support",
      ],
      buttonText: "Get Started",
      popular: false,
      icon: <Server className="h-6 w-6 knirv-text-primary" />,
      href: "/auth",
    },
    {
      name: "Professional",
      price: "$199",
      period: "/month",
      description: "Add adaptive AI governance with Cognitive Engine policies",
      features: [
        "Everything in Free",
        "Unlimited DVEs",
        "Cognitive Engine adaptive policies",
        "Automated policy tuning & anomaly detection",
        "KNIRVARENA tournament access",
        "Priority support",
        "Advanced analytics dashboard",
        "API access",
      ],
      buttonText: "Start Free Trial",
      popular: true,
      icon: <Zap className="h-6 w-6 knirv-text-primary" />,
      href: "/register/key",
    },
    {
      name: "Enterprise",
      price: "Custom",
      description: "Full enforcement stack for compliance-critical deployments",
      features: [
        "Everything in Professional",
        "eBPF kernel bridge for hardware-level enforcement",
        "Badge Lab — attestation & credential issuance",
        "Guardrails — runtime behavioral boundaries",
        "KNIRVCHAIN dedicated subnet",
        "24/7 dedicated support",
        "SLA guarantee",
        "On-premise or private cloud deployment",
      ],
      buttonText: "Contact Sales",
      popular: false,
      icon: <Shield className="h-6 w-6 knirv-text-primary" />,
      href: "/contact",
    },
    {
      name: "Custom",
      price: "Custom",
      description: "Full KNIRVSERVER platform with ICME persistent memory",
      features: [
        "Everything in Enterprise",
        "Full self-hosted KNIRVSERVER deployment",
        "ICME (In-Context Memory Engine) integration",
        "Root node oracle access",
        "KNIRVARENA private tournaments",
        "Source code license",
        "Dedicated engineering support",
        "Custom SLA & compliance contracts",
      ],
      buttonText: "Talk to Engineering",
      popular: false,
      icon: <Crown className="h-6 w-6 text-yellow-400" />,
      href: "/contact",
    },
  ];

  return (
    <div className="dve-page">
      {/* Navigation */}
      <nav className="dve-nav">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <KnirvLogo />
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="py-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <h1 className="text-5xl md:text-6xl font-bold knirv-gradient-text mb-6">
            DVE-Powered Pricing
          </h1>
          <p className="text-xl text-white/70 mb-12 max-w-3xl mx-auto leading-relaxed">
            Start free with your first sovereign DVE. Scale your Distributed Validation
            Environments as your deployment grows.
          </p>
        </div>
      </section>

      {/* Pricing Cards */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            {plans.map((plan, index) => (
              <Card
                key={index}
                className={`relative bg-white/5 border-white/10 backdrop-blur-sm hover:bg-white/10 transition-all duration-300 flex flex-col ${
                  plan.popular ? 'ring-2 ring-knirv-border-primary/50 scale-105' : ''
                }`}
              >
                {plan.popular && (
                  <Badge className="absolute -top-3 left-1/2 transform -translate-x-1/2 bg-knirv-secondary text-white">
                    Most Popular
                  </Badge>
                )}
                <CardHeader className="text-center pb-8">
                  <div className="flex justify-center mb-4">
                    {plan.icon}
                  </div>
                  <CardTitle className="text-white text-2xl mb-2">{plan.name}</CardTitle>
                  <div className="mb-4">
                    <span className="text-4xl font-bold text-white">{plan.price}</span>
                    {plan.period && <span className="text-white/60">{plan.period}</span>}
                  </div>
                  <CardDescription className="text-white/70">
                    {plan.description}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6 flex flex-col flex-1">
                  <ul className="space-y-3 flex-1">
                    {plan.features.map((feature, idx) => (
                      <li key={idx} className="flex items-start space-x-3">
                        <span className="text-green-400 mt-0.5 flex-shrink-0">✓</span>
                        <span className="text-white/80 text-sm">{feature}</span>
                      </li>
                    ))}
                  </ul>
                  <Link href={plan.href}>
                    <Button
                      className={`w-full ${
                        plan.popular
                          ? 'bg-gradient-to-r from-knirv-primary to-knirv-secondary hover:from-knirv-secondary hover:to-knirv-primary text-white'
                          : 'bg-white/10 hover:bg-white/20 text-white border border-white/20'
                      }`}
                    >
                      {plan.buttonText}
                    </Button>
                  </Link>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* Expansion Path */}
      <section className="py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-2xl font-bold text-white mb-4">Phased Rollout Path</h2>
          <p className="text-white/60 mb-8">Services are activated in phases as the network matures.</p>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div className="bg-white/5 rounded-lg p-4 border border-white/10">
              <div className="text-green-400 font-bold mb-1">Phase 1 — Now</div>
              <div className="text-white/70">DVE Management · Policy Editor · Blockchain Commit</div>
            </div>
            <div className="bg-white/5 rounded-lg p-4 border border-white/10">
              <div className="text-blue-400 font-bold mb-1">Phase 2</div>
              <div className="text-white/70">Cognitive Engine · KNIRVARENA · Adaptive Policies</div>
            </div>
            <div className="bg-white/5 rounded-lg p-4 border border-white/10">
              <div className="text-purple-400 font-bold mb-1">Phase 3</div>
              <div className="text-white/70">eBPF Bridge · Badge Lab · Guardrails</div>
            </div>
            <div className="bg-white/5 rounded-lg p-4 border border-white/10">
              <div className="text-yellow-400 font-bold mb-1">Phase 4</div>
              <div className="text-white/70">Full KNIRVSERVER · ICME Memory · Root Oracle</div>
            </div>
          </div>
        </div>
      </section>

      {/* FAQ Section */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl font-bold text-white text-center mb-12">Frequently Asked Questions</h2>
          <div className="space-y-6">
            <Card className="dve-card">
              <CardContent className="p-6">
                <h3 className="text-white font-semibold text-lg mb-2">What is a DVE?</h3>
                <p className="text-white/70">A Distributed Validation Environment (DVE) is a sovereign compute container on the KNIRV network with cryptographic attestation, policy governance, and blockchain-committed audit trails.</p>
              </CardContent>
            </Card>
            <Card className="dve-card">
              <CardContent className="p-6">
                <h3 className="text-white font-semibold text-lg mb-2">How do I pay for additional DVEs?</h3>
                <p className="text-white/70">Additional DVEs are rented using your KNIRVCONTROLLER NRN Wallet balance. You can fund your wallet via the Payment Gateway in the KNIRVGATEWAY dashboard.</p>
              </CardContent>
            </Card>
            <Card className="dve-card">
              <CardContent className="p-6">
                <h3 className="text-white font-semibold text-lg mb-2">Can I upgrade or downgrade plans?</h3>
                <p className="text-white/70">Yes, you can change plans at any time. Upgrades take effect immediately; downgrades apply at the end of your billing cycle.</p>
              </CardContent>
            </Card>
            <Card className="dve-card">
              <CardContent className="p-6">
                <h3 className="text-white font-semibold text-lg mb-2">Do you offer refunds?</h3>
                <p className="text-white/70">Yes, we offer a 30-day money-back guarantee for all paid plans.</p>
              </CardContent>
            </Card>
          </div>
        </div>
      </section>
    </div>
  );
}
