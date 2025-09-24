'use client';

import React from 'react';
import { Network, Book, Code, Zap, Settings, Database, Target, ExternalLink, FileText, Play, Download } from 'lucide-react';
import KnirvLogo from '@/components/KnirvLogo'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import Link from 'next/link';

export default function DocumentationPage() {
  const sections = [
    {
      icon: Play,
      title: 'Getting Started',
      description: 'Quick start guide to building your first KNIRV neural model',
      items: [
        'Installation & Setup',
        'Your First Model',
        'Deployment Options',
        'Basic Configuration'
      ]
    },
    {
      icon: Code,
      title: 'API Reference',
      description: 'Complete API documentation for KNIRV Cortex Builder',
      items: [
        'Model Configuration API',
        'Deployment API',
        'Training API',
        'Monitoring API'
      ]
    },
    {
      icon: Settings,
      title: 'Configuration',
      description: 'Advanced configuration options and customization',
      items: [
        'Model Templates',
        'Training Parameters',
        'Deployment Settings',
        'Environment Variables'
      ]
    },
    {
      icon: Database,
      title: 'Model Management',
      description: 'Managing and versioning your neural models',
      items: [
        'Model Versioning',
        'Import/Export',
        'Model Registry',
        'Performance Monitoring'
      ]
    },
    {
      icon: Target,
      title: 'Deployment',
      description: 'Deploy models across the KNIRV ecosystem',
      items: [
        'Mobile (KNIRVCONTROLLER)',
        'Desktop (KNIRVENGINE)',
        'Game (KNIRVANA)',
        'Cloud (KNIRVNEXUS)'
      ]
    },
    {
      icon: Zap,
      title: 'Advanced Topics',
      description: 'Advanced features and optimization techniques',
      items: [
        'Custom Architectures',
        'Performance Optimization',
        'Security Best Practices',
        'Troubleshooting'
      ]
    }
  ];

  const quickLinks = [
    { title: 'Installation Guide', href: 'https://knirv.network/agent-developer-portal/', icon: Download },
    { title: 'API Reference', href: 'https://knirv.network/agent-developer-portal/', icon: Code },
    { title: 'Examples', href: 'https://knirv.network/agent-developer-portal/', icon: FileText },
    { title: 'Tutorials', href: '/tutorials', icon: Play }
  ];

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-black">
      <nav className="border-b border-white/10 bg-slate-900/50 backdrop-blur-lg">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <KnirvLogo />
            <Link href="/" className="text-white/70 hover:text-white transition-colors">
               Back to Home
            </Link>
          </div>
        </div>
      </nav>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="text-center mb-16">
          <Book className="h-16 w-16 knirv-text-primary mx-auto mb-6" />
          <h1 className="text-5xl font-bold text-white mb-6">
            <span className="knirv-gradient-text">KNIRV</span> Documentation
          </h1>
          <p className="text-xl text-white/70 mb-8 max-w-3xl mx-auto">
            Complete documentation for building, training, and deploying neural models 
            across the KNIRV decentralized network.
          </p>
          
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 max-w-4xl mx-auto">
            {quickLinks.map((link, index) => {
              const IconComponent = link.icon;
              return (
                <Link key={index} href={link.href}>
                  <Card className="knirv-card-gradient backdrop-blur-lg hover:scale-105 transition-transform cursor-pointer">
                    <CardContent className="p-4 text-center">
                      <IconComponent className="h-8 w-8 knirv-text-primary mx-auto mb-2" />
                      <p className="text-white font-medium">{link.title}</p>
                    </CardContent>
                  </Card>
                </Link>
              );
            })}
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 mb-16">
          {sections.map((section, index) => {
            const IconComponent = section.icon;
            return (
              <Card key={index} className="knirv-card-gradient backdrop-blur-lg">
                <CardHeader>
                  <IconComponent className="h-12 w-12 knirv-text-primary mb-4" />
                  <CardTitle className="text-white">{section.title}</CardTitle>
                  <CardDescription className="text-white/70">
                    {section.description}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-2">
                    {section.items.map((item, itemIndex) => (
                      <li key={itemIndex} className="flex items-center text-white/80">
                        <div className="w-2 h-2 knirv-bg-primary rounded-full mr-3"></div>
                        {item}
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
            );
          })}
        </div>

        <div className="mb-16">
          <h2 className="text-3xl font-bold text-white text-center mb-12">
            Featured <span className="knirv-gradient-text">Guides</span>
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
            <Card className="knirv-card-gradient backdrop-blur-lg">
              <CardHeader>
                <CardTitle className="text-white flex items-center">
                  <Play className="h-6 w-6 knirv-text-primary mr-2" />
                  Quick Start Guide
                </CardTitle>
                <CardDescription className="text-white/70">
                  Get up and running with KNIRV Cortex Builder in under 10 minutes
                </CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-white/80 mb-4">
                  Learn how to create, configure, and deploy your first neural model using 
                  our intuitive interface and pre-built templates.
                </p>
                <div className="flex items-center justify-between">
                  <Badge variant="outline" className="knirv-border-primary knirv-text-primary">
                    Beginner
                  </Badge>
                  <Link href="https://knirv.network/agent-developer-portal/" data-config-nav="documentation" className="text-blue-400 hover:text-blue-300 flex items-center">
                    Read Guide <ExternalLink className="h-4 w-4 ml-1" />
                  </Link>
                </div>
              </CardContent>
            </Card>

            <Card className="knirv-card-gradient backdrop-blur-lg">
              <CardHeader>
                <CardTitle className="text-white flex items-center">
                  <Target className="h-6 w-6 knirv-text-primary mr-2" />
                  Deployment Strategies
                </CardTitle>
                <CardDescription className="text-white/70">
                  Advanced deployment patterns across the KNIRV ecosystem
                </CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-white/80 mb-4">
                  Explore different deployment strategies for mobile, desktop, gaming, 
                  and cloud environments with performance optimization tips.
                </p>
                <div className="flex items-center justify-between">
                  <Badge variant="outline" className="knirv-border-primary knirv-text-primary">
                    Advanced
                  </Badge>
                  <Link href="#deployment" data-config-nav="documentation" className="text-blue-400 hover:text-blue-300 flex items-center">
                    Read Guide <ExternalLink className="h-4 w-4 ml-1" />
                  </Link>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>

        <div className="text-center">
          <Card className="knirv-card-gradient backdrop-blur-lg max-w-4xl mx-auto">
            <CardHeader>
              <CardTitle className="text-3xl text-white mb-4">Need Help?</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-lg text-white/80 leading-relaxed mb-6">
                Can't find what you're looking for? Our community and support team 
                are here to help you succeed with KNIRV.
              </p>
              <div className="flex flex-col sm:flex-row gap-4 justify-center">
                <Link href="/community">
                  <button className="knirv-gradient hover:opacity-90 px-6 py-3 rounded-lg text-white font-semibold transition-opacity">
                    Join Community
                  </button>
                </Link>
                <Link href="/support">
                  <button className="border border-blue-500 text-blue-400 hover:bg-blue-500/10 px-6 py-3 rounded-lg font-semibold transition-colors">
                    Contact Support
                  </button>
                </Link>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
