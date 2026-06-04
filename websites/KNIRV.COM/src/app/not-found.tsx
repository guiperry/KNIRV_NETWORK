'use client';

import React from 'react';
import { Home, ArrowLeft } from 'lucide-react';
import { Button } from "@/components/ui/button";
import Link from 'next/link';
import KnirvLogo from '@/components/KnirvLogo'

export default function NotFound() {
  return (
  <div className="dve-page flex items-center justify-center">
      <div className="dve-card max-w-md mx-auto text-center px-6 py-8 sm:px-8">
        {/* Logo */}
        <div className="flex justify-center mb-8">
          <KnirvLogo />
        </div>
        
        {/* 404 Text */}
  <h1 className="mb-4 text-8xl font-bold knirv-gradient-text">
          404
        </h1>
        
        {/* Error Message */}
        <h2 className="mb-4 text-2xl font-bold text-white">
          Page Not Found
        </h2>
        
        <p className="text-white/70 mb-8 leading-relaxed">
          Oops! The page you're looking for doesn't exist. It might have been moved, deleted, or you entered the wrong URL.
        </p>
        
        {/* Action Buttons */}
        <div className="space-y-4">
          <Link href="/">
            <Button className="w-full bg-gradient-to-r from-knirv-primary to-knirv-secondary hover:from-knirv-secondary hover:to-knirv-primary text-white">
              <Home className="w-4 h-4 mr-2" />
              Go Home
            </Button>
          </Link>
          
          <Button 
            variant="outline" 
            className="w-full border-white/20 text-white/70 hover:bg-white/10"
            onClick={() => window.history.back()}
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Go Back
          </Button>
        </div>
        
        {/* Additional Help */}
        <div className="mt-12 border-t border-white/10 pt-8">
          <p className="mb-4 text-sm text-white/50">
            Need help? Try these popular pages:
          </p>
          <div className="space-y-2">
            <Link href="/features" data-config-nav="features" className="block knirv-text-primary hover:knirv-text-primary/80 transition-colors text-sm">
              Features
            </Link>
            <Link href="/pricing" data-config-nav="pricing" className="block knirv-text-primary hover:knirv-text-primary/80 transition-colors text-sm">
              Pricing
            </Link>
            <Link href="/documentation" data-config-nav="documentation" className="block knirv-text-primary hover:knirv-text-primary/80 transition-colors text-sm">
              Documentation
            </Link>
            <Link href="/about" data-config-nav="about" className="block knirv-text-primary hover:knirv-text-primary/80 transition-colors text-sm">
              About
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
