'use client';

import Link from 'next/link';
import { Network, Heart, Github, Twitter, Linkedin } from 'lucide-react';

const Footer = () => {
  return (
    <footer className="bg-slate-900/95 border-t border-white/10 backdrop-blur-lg mt-auto">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
          {/* Brand Section */}
          <div className="space-y-4">
            <Link href="/" data-config-nav="home" className="flex items-center space-x-2" aria-label="Go to homepage">
              <Network className="h-8 w-8 knirv-text-primary" />
              <span className="text-2xl font-bold knirv-gradient-text">
                KNIRV
              </span>
            </Link>
            <p className="text-white/70 text-sm leading-relaxed">
              Build and deploy neural intelligence models with the KNIRV platform. Create, configure, and manage AI models for the decentralized future.
            </p>
            <div className="flex space-x-4">
              <a href="#" className="text-white/60 hover:knirv-text-primary transition-colors">
                <Twitter className="h-5 w-5" />
              </a>
              <a href="#" className="text-white/60 hover:knirv-text-primary transition-colors">
                <Github className="h-5 w-5" />
              </a>
              <a href="#" className="text-white/60 hover:knirv-text-primary transition-colors">
                <Linkedin className="h-5 w-5" />
              </a>
            </div>
          </div>
          {/* Resources */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-white">Resources</h3>
            <ul className="space-y-3">
              <li>
                <Link href="/documentation" data-config-nav="documentation" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Documentation
                </Link>
              </li>
              <li>
                <Link href="/tutorials" data-config-nav="tutorials" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Tutorials
                </Link>
              </li>
              <li>
                <Link href="/blog" data-config-nav="blog" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Blog
                </Link>
              </li>
              <li>
                <Link href="/api-docs" data-config-nav="api_docs" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  API Reference
                </Link>
              </li>
              <li>
                <Link href="/logo" data-config-nav="logo" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Brand Assets
                </Link>
              </li>
              <li>
                <Link href="/community" data-config-nav="community" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Community
                </Link>
              </li>
            </ul>
          </div>
          {/* Company */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-white">Company</h3>
            <ul className="space-y-3">
              <li>
                <Link href="/about" data-config-nav="about" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  About KNIRV
                </Link>
              </li>
              <li>
                <Link href="/pricing" data-config-nav="pricing" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Pricing
                </Link>
              </li>
              <li>
                <Link href="/contact" data-config-nav="contact" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Contact
                </Link>
              </li>
              <li>
                <Link href="/support" data-config-nav="support" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Support
                </Link>
              </li>
            </ul>
          </div>

          {/* Legal */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-white">Legal</h3>
            <ul className="space-y-3">
              <li>
                <Link href="/privacy-policy" data-config-footer="legal.privacy" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Privacy Policy
                </Link>
              </li>
              <li>
                <Link href="/terms-of-service" data-config-footer="legal.terms" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Terms of Service
                </Link>
              </li>
              <li>
                <Link href="/cookie-policy" data-config-footer="legal.cookie" className="text-white/70 hover:knirv-text-primary transition-colors text-sm">
                  Cookie Policy
                </Link>
              </li>
            </ul>
          </div>
        </div>

        {/* Bottom Section */}
        <div className="mt-12 pt-8 border-t border-white/10">
          <div className="flex flex-col md:flex-row justify-between items-center space-y-4 md:space-y-0">
            <div className="flex flex-col md:flex-row items-center space-y-2 md:space-y-0 md:space-x-6 text-sm text-white/60">
              <p>&copy; 2024 <Link href="knirv.network" className="hover:text-white transition-colors">KNIRV NETWORK</Link>. All rights reserved.</p>
              <div className="flex space-x-6">
                <Link href="/privacy-policy" data-config-footer="legal.privacy" className="hover:text-white transition-colors">Privacy Policy</Link>
                <Link href="/terms-of-service" data-config-footer="legal.terms" className="hover:text-white transition-colors">Terms of Service</Link>
                <Link href="/cookie-policy" data-config-footer="legal.cookie" className="hover:text-white transition-colors">Cookie Policy</Link>
              </div>
            </div>
            <div className="flex items-center space-x-1 text-sm text-white/60">
                <span>Built with</span>
                <Heart className="h-4 w-4 text-red-400" />
                <span>for the decentralized future</span>
              </div>
          </div>
        </div>
      </div>
    </footer>
  );
};

export default Footer;