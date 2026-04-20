'use client';

import React from 'react';
import { Bot, CheckCircle, Zap, Crown } from 'lucide-react';
import KnirvLogo from '@/components/KnirvLogo'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import Link from 'next/link';

export default function Pricing() {
  const plans = [
    {
      name: "Starter",
      price: "Free",
      description: "Perfect for trying out KNIRV",
      features: [
        "1 Neural Intellegence Model",
        "1,000 messages/month",
        "Basic integrations",
        "Community support",
        "Standard response time"
      ],
      buttonText: "Get Started",
      popular: false,
  icon: <Zap className="h-6 w-6 knirv-text-primary" />
    },
    {
      name: "Professional",
      price: "$49",
      period: "/month",
      description: "Ideal for growing businesses",
      features: [
        "5 Neural Intellegence Models",
        "50,000 messages/month",
        "Advanced integrations",
        "Priority support",
        "Custom branding",
        "Analytics dashboard",
        "API access"
      ],
      buttonText: "Start Free Trial",
      popular: true,
  icon: <Bot className="h-6 w-6 knirv-text-primary" />
    },
    {
      name: "Enterprise",
      price: "Custom",
      description: "For large-scale deployments",
      features: [
        "Unlimited Neural Intellegence Models",
        "Unlimited messages",
        "Custom integrations",
        "24/7 dedicated support",
        "White-label solution",
        "Advanced analytics",
        "SLA guarantee",
        "On-premise deployment"
      ],
      buttonText: "Contact Sales",
      popular: false,
  icon: <Crown className="h-6 w-6 text-yellow-400" />
    }
  ];

  return (
  <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-black">
      {/* Navigation */}
  <nav className="border-b border-white/10 bg-slate-900/50 backdrop-blur-lg">
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
            Simple, Transparent Pricing
          </h1>
          <p className="text-xl text-white/70 mb-12 max-w-3xl mx-auto leading-relaxed">
            Choose the perfect plan for your needs. Start free and scale as you grow.
          </p>
        </div>
      </section>

      {/* Pricing Cards */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {plans.map((plan, index) => (
              <Card 
                key={index} 
                className={`relative bg-white/5 border-white/10 backdrop-blur-sm hover:bg-white/10 transition-all duration-300 ${
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
                <CardContent className="space-y-6">
                  <ul className="space-y-3">
                    {plan.features.map((feature, idx) => (
                      <li key={idx} className="flex items-center space-x-3">
                        <CheckCircle className="h-5 w-5 text-green-400 flex-shrink-0" />
                        <span className="text-white/80">{feature}</span>
                      </li>
                    ))}
                  </ul>
                  <Button 
                    className={`w-full ${
                      plan.popular 
                        ? 'bg-gradient-to-r from-knirv-primary to-knirv-secondary hover:from-knirv-secondary hover:to-knirv-primary text-white' 
                        : 'bg-white/10 hover:bg-white/20 text-white border border-white/20'
                    }`}
                  >
                    {plan.buttonText}
                  </Button>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* FAQ Section */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl font-bold text-white text-center mb-12">Frequently Asked Questions</h2>
          <div className="space-y-6">
            <Card className="bg-white/5 border-white/10 backdrop-blur-sm">
              <CardContent className="p-6">
                <h3 className="text-white font-semibold text-lg mb-2">Can I change plans anytime?</h3>
                <p className="text-white/70">Yes, you can upgrade or downgrade your plan at any time. Changes take effect immediately.</p>
              </CardContent>
            </Card>
            <Card className="bg-white/5 border-white/10 backdrop-blur-sm">
              <CardContent className="p-6">
                <h3 className="text-white font-semibold text-lg mb-2">What happens if I exceed my message limit?</h3>
                <p className="text-white/70">We'll notify you when you're approaching your limit. You can upgrade your plan or purchase additional messages.</p>
              </CardContent>
            </Card>
            <Card className="bg-white/5 border-white/10 backdrop-blur-sm">
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
