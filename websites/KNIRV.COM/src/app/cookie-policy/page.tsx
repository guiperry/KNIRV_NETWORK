'use client';

import React from "react";
import { Bot } from "lucide-react";
import Link from "next/link";
import KnirvLogo from '@/components/KnirvLogo'

export default function CookiePolicyPage() {
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

      {/* Content */}
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <h1 className="text-4xl font-bold text-white mb-8">Cookie Policy</h1>
        
        <div className="dve-prose max-w-none">
          <div className="text-white/70 space-y-6">
            <p className="text-lg">
              Last updated: December 2024
            </p>
            
            <section>
              <h2 className="text-2xl font-semibold text-white mb-4">What Are Cookies</h2>
              <p>
                Cookies are small text files that are placed on your computer by websites that you visit. 
                They are widely used to make websites work more efficiently.
              </p>
            </section>

            <section>
              <h2 className="text-2xl font-semibold text-white mb-4">How We Use Cookies</h2>
              <p>
                We use cookies to enhance your experience, analyze site usage, and assist in our 
                marketing efforts.
              </p>
            </section>

            <section>
              <h2 className="text-2xl font-semibold text-white mb-4">Types of Cookies We Use</h2>
              <ul className="list-disc list-inside space-y-2">
                <li>Essential cookies: Required for the website to function properly</li>
                <li>Analytics cookies: Help us understand how visitors interact with our website</li>
                <li>Functional cookies: Enable enhanced functionality and personalization</li>
              </ul>
            </section>

            <section>
              <h2 className="text-2xl font-semibold text-white mb-4">Managing Cookies</h2>
              <p>
                You can control and/or delete cookies as you wish. You can delete all cookies 
                that are already on your computer and set most browsers to prevent them from being placed.
              </p>
            </section>

            <section>
              <h2 className="text-2xl font-semibold text-white mb-4">Contact Us</h2>
              <p>
                If you have any questions about our use of cookies, please contact us at{' '}
                <Link href="/contact" className="text-knirv-text-primary hover:text-knirv-text-primary/80">
                  our contact page
                </Link>.
              </p>
            </section>
          </div>
        </div>
      </div>
    </div>
  );
}
