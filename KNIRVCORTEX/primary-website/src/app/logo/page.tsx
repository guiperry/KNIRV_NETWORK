'use client';

import React, { useState } from 'react';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Download, Copy, Check, Palette, Monitor, Smartphone, Printer } from 'lucide-react';
import KnirvNetworkLogo from '../../components/KnirvNetworkLogo.jsx';
import KnirvLogo from '@/components/KnirvLogo'
import Link from 'next/link';

const LogoPage = () => {
  const [copiedText, setCopiedText] = useState('');

  const copyToClipboard = (text: string, type: string) => {
    navigator.clipboard.writeText(text);
    setCopiedText(type);
    setTimeout(() => setCopiedText(''), 2000);
  };

  const colorPalette = [
    { name: 'Primary Blue', hex: '#00c0fa', rgb: 'rgb(0, 192, 250)', usage: 'Primary brand color, main UI elements' },
    { name: 'Secondary Blue', hex: '#2b56f5', rgb: 'rgb(43, 86, 245)', usage: 'Secondary accents, gradients' },
    { name: 'Neural Purple', hex: '#8b5cf6', rgb: 'rgb(139, 92, 246)', usage: 'Neural network elements, highlights' },
    { name: 'Cyber Cyan', hex: '#00f5ff', rgb: 'rgb(0, 245, 255)', usage: 'Glow effects, electrical activity' },
    { name: 'Deep Space', hex: '#1a1a2e', rgb: 'rgb(26, 26, 46)', usage: 'Dark backgrounds, depth' },
  ];

  const usageGuidelines = [
    {
      icon: Monitor,
      title: 'Digital Usage',
      description: 'Use the animated logo for web interfaces and digital applications. Maintain minimum size of 120px width.',
      formats: ['SVG (Recommended)', 'PNG (High-res)', 'WebP']
    },
    {
      icon: Smartphone,
      title: 'Mobile Applications',
      description: 'For mobile apps, use simplified version without animation. Ensure readability at small sizes.',
      formats: ['PNG', 'SVG', 'ICO (for favicons)']
    },
    {
      icon: Printer,
      title: 'Print Materials',
      description: 'Use high-resolution static version for print. Ensure sufficient contrast on light backgrounds.',
      formats: ['PDF (Vector)', 'PNG (300 DPI)', 'EPS']
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

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        {/* Hero Section with Large Logo */}
        <div className="text-center mb-16">
          <h1 className="text-5xl font-bold text-white mb-6">
            <span className="knirv-gradient-text">KNIRV</span> Brand Assets
          </h1>
          <p className="text-xl text-white/70 mb-12 max-w-3xl mx-auto">
            Official brand assets, logos, and guidelines for the KNIRV Neural Intelligence Network. 
            Use these resources to maintain consistent branding across all KNIRV applications.
          </p>

          {/* Large Animated Logo Display */}
          <div className="flex justify-center mb-12">
            <div className="relative">
              <div className="w-[600px] h-[300px] bg-gradient-to-br from-slate-900 via-slate-800 to-black rounded-3xl p-8 shadow-2xl knirv-glow border border-white/10">
                <KnirvNetworkLogo />
              </div>
              <div className="absolute -top-4 -right-4">
                <Badge className="knirv-gradient text-white font-semibold px-4 py-2">
                  Official Logo
                </Badge>
              </div>
            </div>
          </div>
        </div>

        {/* Color Palette Section */}
        <div className="mb-16">
          <Card className="knirv-card-gradient backdrop-blur-lg">
            <CardHeader>
              <CardTitle className="text-white flex items-center">
                <Palette className="h-6 w-6 mr-2 knirv-text-primary" />
                Brand Color Palette
              </CardTitle>
              <CardDescription className="text-white/70">
                Official KNIRV colors with hex codes and usage guidelines
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {colorPalette.map((color, index) => (
                  <div key={index} className="space-y-3">
                    <div 
                      className="w-full h-24 rounded-lg border border-white/20 shadow-lg"
                      style={{ backgroundColor: color.hex }}
                    ></div>
                    <div>
                      <h3 className="text-white font-semibold">{color.name}</h3>
                      <div className="flex items-center space-x-2 mt-2">
                        <code className="text-sm bg-slate-800 text-white px-2 py-1 rounded">
                          {color.hex}
                        </code>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => copyToClipboard(color.hex, `hex-${index}`)}
                          className="h-6 w-6 p-0 text-white/60 hover:text-white"
                        >
                          {copiedText === `hex-${index}` ? (
                            <Check className="h-3 w-3" />
                          ) : (
                            <Copy className="h-3 w-3" />
                          )}
                        </Button>
                      </div>
                      <div className="flex items-center space-x-2 mt-1">
                        <code className="text-xs bg-slate-800 text-white/80 px-2 py-1 rounded">
                          {color.rgb}
                        </code>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => copyToClipboard(color.rgb, `rgb-${index}`)}
                          className="h-6 w-6 p-0 text-white/60 hover:text-white"
                        >
                          {copiedText === `rgb-${index}` ? (
                            <Check className="h-3 w-3" />
                          ) : (
                            <Copy className="h-3 w-3" />
                          )}
                        </Button>
                      </div>
                      <p className="text-white/60 text-xs mt-2">{color.usage}</p>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Usage Guidelines */}
        <div className="mb-16">
          <h2 className="text-3xl font-bold text-white mb-8 text-center">Usage Guidelines</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {usageGuidelines.map((guideline, index) => {
              const IconComponent = guideline.icon;
              return (
                <Card key={index} className="knirv-card-gradient backdrop-blur-lg">
                  <CardHeader>
                    <CardTitle className="text-white flex items-center">
                      <IconComponent className="h-6 w-6 mr-2 knirv-text-primary" />
                      {guideline.title}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-white/70 mb-4">{guideline.description}</p>
                    <div className="space-y-2">
                      <h4 className="text-white font-medium text-sm">Recommended Formats:</h4>
                      <div className="flex flex-wrap gap-2">
                        {guideline.formats.map((format, formatIndex) => (
                          <Badge key={formatIndex} variant="outline" className="knirv-border-primary knirv-text-primary">
                            {format}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </div>

        {/* Download Section */}
        <div className="text-center">
          <Card className="knirv-card-gradient backdrop-blur-lg max-w-2xl mx-auto">
            <CardHeader>
              <CardTitle className="text-white flex items-center justify-center">
                <Download className="h-6 w-6 mr-2 knirv-text-primary" />
                Download Assets
              </CardTitle>
              <CardDescription className="text-white/70">
                Get the complete KNIRV brand asset package
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Button className="knirv-gradient hover:opacity-90">
                  <Download className="h-4 w-4 mr-2" />
                  Logo Package (ZIP)
                </Button>
                <Button variant="outline" className="knirv-border-primary knirv-text-primary hover:bg-white/10">
                  <Download className="h-4 w-4 mr-2" />
                  Brand Guidelines (PDF)
                </Button>
              </div>
              <p className="text-white/60 text-sm">
                By downloading these assets, you agree to use them in accordance with KNIRV brand guidelines.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default LogoPage;
