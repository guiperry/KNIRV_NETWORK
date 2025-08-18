import type { Metadata } from 'next';
import { Geist, Geist_Mono } from 'next/font/google';
import './globals.css';
import MainLayout from '@/components/layout/MainLayout';

const geistSans = Geist({
  variable: '--font-geist-sans',
  subsets: ['latin'],
});

const geistMono = Geist_Mono({
  variable: '--font-geist-mono',
  subsets: ['latin'],
});

export const metadata: Metadata = {
  title: 'AgentVerse Registry - NANDA+ANS',
  description: 'A federated registry for secure, capability-aware AI agent discovery, based on the NANDA+ANS Security Blueprint. Features a dual-trust foundation, decoupled lookup model, cryptographic assurance for agent capabilities, and privacy-preserving discovery for a resilient Internet of AI Agents.',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${geistSans.variable} ${geistMono.variable} antialiased`}>
        <MainLayout>{children}</MainLayout>
      </body>
    </html>
  );
}
