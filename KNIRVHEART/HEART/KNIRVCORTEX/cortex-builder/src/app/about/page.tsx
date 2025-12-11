'use client';

import React from 'react';
import { Network, Brain, Zap, Shield, Globe, Users, Target, Lightbulb } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import Link from 'next/link';
import KnirvNetworkLogo from '../../components/KnirvNetworkLogo.jsx';
import KnirvLogo from '@/components/KnirvLogo';

export default function About() {
  const features = [
    {
      icon: Brain,
      title: 'Neural Intelligence',
      description: 'Advanced AI models optimized for decentralized execution and edge deployment.'
    },
    {
      icon: Network,
      title: 'Decentralized Network',
      description: 'Distributed architecture ensuring resilience, scalability, and global accessibility.'
    },
    {
      icon: Shield,
      title: 'Trusted Execution',
      description: 'Secure environments with cryptographic verification and privacy protection.'
    },
    {
      icon: Zap,
      title: 'High Performance',
      description: 'Optimized for speed and efficiency across mobile, desktop, and cloud platforms.'
    },
    {
      icon: Globe,
      title: 'Cross-Platform',
      description: 'Deploy anywhere - from mobile apps to desktop software to cloud infrastructure.'
    },
    {
      icon: Users,
      title: 'Developer Friendly',
      description: 'Intuitive tools and comprehensive APIs for seamless integration and development.'
    }
  ];

  const stats = [
    { label: 'Active Nodes', value: '10,000+' },
    { label: 'Models Deployed', value: '50,000+' },
    { label: 'Developers', value: '5,000+' },
    { label: 'Countries', value: '120+' }
  ];



  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-black">
      {/* Navigation */}
      <nav className="border-b border-white/10 bg-slate-900/50 backdrop-blur-lg">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <KnirvLogo />
            <Link href="/" className="text-white/70 hover:text-white transition-colors">
              ← Back to Home
            </Link>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="py-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <div className="flex justify-center mb-8">
            <div className="w-[400px] h-[200px] bg-gradient-to-br from-slate-900 via-slate-800 to-black rounded-2xl p-4 shadow-2xl knirv-glow">
              <KnirvNetworkLogo />
            </div>
          </div>

          <h1 className="text-5xl md:text-6xl font-bold text-white mb-6">
            About <span className="knirv-gradient-text">KNIRV</span>
          </h1>
          <p className="text-xl text-white/70 mb-12 max-w-3xl mx-auto leading-relaxed">
            KNIRV (Knowledge Network for Intelligent Reasoning and Verification) is a revolutionary
            decentralized platform for neural intelligence deployment. We're building the future of
            AI where models run everywhere, securely and efficiently.
          </p>

          {/* Stats */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mb-16">
            {stats.map((stat, index) => (
              <div key={index} className="text-center">
                <div className="text-3xl font-bold knirv-gradient-text mb-2">{stat.value}</div>
                <div className="text-white/70">{stat.label}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Mission Section */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto">
          <Card className="knirv-card-gradient backdrop-blur-lg">
            <CardHeader className="text-center">
              <CardTitle className="text-3xl text-white mb-4">Our Mission</CardTitle>
            </CardHeader>
            <CardContent className="text-center">
              <p className="text-lg text-white/80 max-w-4xl mx-auto leading-relaxed">
                To democratize artificial intelligence by creating a decentralized network where
                neural models can be deployed, executed, and verified across any platform. We believe
                AI should be accessible, secure, and efficient for everyone - from individual developers
                to enterprise organizations.
              </p>
            </CardContent>
          </Card>
        </div>
      </section>

      {/* Values Section */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-6xl mx-auto">
          <h2 className="text-3xl font-bold text-white text-center mb-12">
            Why Choose <span className="knirv-gradient-text">KNIRV</span>?
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            {features.map((feature, index) => {
              const IconComponent = feature.icon;
              return (
                <Card key={index} className="knirv-card-gradient backdrop-blur-lg">
                  <CardHeader>
                    <IconComponent className="h-12 w-12 knirv-text-primary mb-4" />
                    <CardTitle className="text-white">{feature.title}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-white/70">{feature.description}</p>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </div>
      </section>

      {/* Technology Stack */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-6xl mx-auto">
          <Card className="knirv-card-gradient backdrop-blur-lg">
            <CardHeader className="text-center">
              <CardTitle className="text-3xl text-white mb-4">Technology Stack</CardTitle>
              <CardDescription className="text-white/70 text-lg">
                Built on cutting-edge technologies for maximum performance and reliability
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {[
                  'WebAssembly', 'Rust', 'TypeScript', 'Go',
                  'PyTorch', 'TensorFlow', 'ONNX', 'Blockchain',
                  'Docker', 'Kubernetes', 'AWS', 'CloudFlare'
                ].map((tech, index) => (
                  <Badge key={index} variant="outline" className="knirv-border-primary knirv-text-primary text-center py-2">
                    {tech}
                  </Badge>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <Card className="knirv-card-gradient backdrop-blur-lg">
            <CardHeader>
              <CardTitle className="text-3xl text-white mb-4">The Future is Decentralized</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-lg text-white/80 leading-relaxed mb-6">
                We envision a world where AI models run seamlessly across billions of devices,
                from smartphones to IoT sensors, all connected through the KNIRV network.
                A future where intelligence is distributed, privacy is preserved, and innovation
                knows no boundaries.
              </p>
              <Link href="/" className="inline-block">
                <button className="knirv-gradient hover:opacity-90 px-8 py-3 rounded-lg text-white font-semibold transition-opacity">
                  Start Building with KNIRV
                </button>
              </Link>
            </CardContent>
          </Card>
        </div>
      </section>
    </div>
  );
}
