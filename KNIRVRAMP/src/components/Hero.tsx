'use client';

import Link from 'next/link';
import { Button } from '@/components/ui/button';

interface HeroProps {
  onGetStarted?: () => void;
}

const Hero = ({ onGetStarted }: HeroProps) => {
  return (
    <div className="relative overflow-hidden bg-slate-900">
      <div className="absolute inset-0 knirv-gradient opacity-20"></div>
      <div className="absolute inset-0 bg-grid-white/[0.02] bg-[length:20px_20px]"></div>
      <div className="relative max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-24 md:py-32 lg:py-40">
        <div className="text-center">
          <h1 className="text-4xl md:text-5xl lg:text-6xl font-bold text-white mb-6 tracking-tight">
            Build and Deploy <span className="knirv-gradient-text">Neural Intelligence Models</span> with KNIRV
          </h1>
          <p className="text-xl md:text-2xl text-slate-300 max-w-3xl mx-auto mb-10">
            Create, train, and deploy advanced neural intelligence models using the KNIRV NIM Builder. Build custom AI language models for your specific needs.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            {onGetStarted ? (
              <Button
                onClick={onGetStarted}
                size="lg"
                className="knirv-gradient hover:opacity-90 text-white font-semibold"
              >
                Build Your AI Model
              </Button>
            ) : (
              <Button asChild size="lg" className="knirv-gradient hover:opacity-90 text-white font-semibold">
                <Link href="/dashboard" data-config-nav="dashboard">Build Your AI Model</Link>
              </Button>
            )}
            <Button asChild size="lg" variant="outline" className="knirv-border-primary knirv-text-primary hover:bg-slate-800">
              <Link href="/documentation" data-config-nav="documentation">View Documentation</Link>
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Hero;