'use client';

import Link from 'next/link';
import { Network, Heart, Github, Twitter, Linkedin } from 'lucide-react';

const Footer = () => {
  return (
    <footer className="mt-auto border-t border-white/10 bg-[#040810]/95 backdrop-blur-xl">
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 gap-8 md:grid-cols-2 lg:grid-cols-4">
          {/* Brand Section */}
          <div className="space-y-4">
            <Link href="/" data-config-nav="home" className="flex items-center space-x-3" aria-label="Go to homepage">
              <Network className="h-8 w-8 knirv-text-primary" />
              <span className="text-2xl font-extrabold tracking-tight knirv-gradient-text">
                KNIRV
              </span>
            </Link>
            <p className="text-sm leading-relaxed text-white/70">
              Build and deploy neural intelligence models with the KNIRV platform. Create, configure, and manage AI models for the decentralized future.
            </p>
            <div className="flex space-x-4">
              <a href="#" className="text-white/60 transition-colors hover:knirv-text-primary">
                <Twitter className="h-5 w-5" />
              </a>
              <a href="#" className="text-white/60 transition-colors hover:knirv-text-primary">
                <Github className="h-5 w-5" />
              </a>
              <a href="#" className="text-white/60 transition-colors hover:knirv-text-primary">
                <Linkedin className="h-5 w-5" />
              </a>
            </div>
          </div>
          {/* Resources */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-white">Resources</h3>
            <ul className="space-y-3">
              <li>
                <Link href="/documentation" data-config-nav="documentation" className="text-sm text-white/70 transition-colors hover:knirv-text-primary">
                  Documentation
                </Link>
              </li>
              <li>
                <Link href="/blog" data-config-nav="blog" className="text-sm text-white/70 transition-colors hover:knirv-text-primary">
                  Blog
                </Link>
              </li>
              <li>
                <Link href="/community" data-config-nav="community" className="text-sm text-white/70 transition-colors hover:knirv-text-primary">
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
                <Link href="/about" data-config-nav="about" className="text-sm text-white/70 transition-colors hover:knirv-text-primary">
                  About KNIRV
                </Link>
              </li>
              <li>
                <Link href="/pricing" data-config-nav="pricing" className="text-sm text-white/70 transition-colors hover:knirv-text-primary">
                  Pricing
                </Link>
              </li>
              <li>
                <Link href="/contact" data-config-nav="contact" className="text-sm text-white/70 transition-colors hover:knirv-text-primary">
                  Contact
                </Link>
              </li>
              <li>
                <Link href="/support" data-config-nav="support" className="text-sm text-white/70 transition-colors hover:knirv-text-primary">
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
                <Link href="/privacy-policy" data-config-footer="legal.privacy" className="text-sm text-white/70 transition-colors hover:knirv-text-primary">
                  Privacy Policy
                </Link>
              </li>
              <li>
                <Link href="/terms-of-service" data-config-footer="legal.terms" className="text-sm text-white/70 transition-colors hover:knirv-text-primary">
                  Terms of Service
                </Link>
              </li>
              <li>
                <Link href="/cookie-policy" data-config-footer="legal.cookie" className="text-sm text-white/70 transition-colors hover:knirv-text-primary">
                  Cookie Policy
                </Link>
              </li>
            </ul>
          </div>
        </div>

        {/* Bottom Section */}
        <div className="mt-12 border-t border-white/10 pt-8">
          <div className="flex flex-col items-center justify-between space-y-4 md:flex-row md:space-y-0">
            <div className="flex flex-col items-center space-y-2 text-sm text-white/60 md:flex-row md:space-y-0 md:space-x-6">
              <p>&copy; 2024 <Link href="knirv.network" className="transition-colors hover:text-white">KNIRV NETWORK</Link>. All rights reserved.</p>
              <div className="flex space-x-6">
                <Link href="/privacy-policy" data-config-footer="legal.privacy" className="transition-colors hover:text-white">Privacy Policy</Link>
                <Link href="/terms-of-service" data-config-footer="legal.terms" className="transition-colors hover:text-white">Terms of Service</Link>
                <Link href="/cookie-policy" data-config-footer="legal.cookie" className="transition-colors hover:text-white">Cookie Policy</Link>
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
