import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Toaster } from "@/components/ui/toaster";
import { AuthProvider } from "@/lib/auth-context";
import { DemoModeProvider } from "@/contexts/demo-mode-context";
import { DHTProvider } from "@/contexts/dht-context";
import { OnboardingProvider } from "@/contexts/onboarding-context";
import QueryProvider from "@/components/providers/query-provider";
import { PWAProvider } from "@/components/pwa-provider";
import { HudOverlay } from "@/components/hud";
import { PWAInstallButton } from "@/components/ui/pwa-install-button";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "KNIRV-SERVER DVE - Deterministic Validation Environment",
  description: "The Crucible of Verifiable AI Intelligence - Powering Trustless Validation, Secure Execution, and Collective Learning in the KNIRV D-TEN",
  keywords: ["KNIRV-SERVER", "DVE", "Decentralized Validation", "AI Intelligence", "Trusted Execution", "CLEAN", "KNIRV D-TEN", "Blockchain", "TEE"],
  authors: [{ name: "KNIRV Network Team" }],
  manifest: "/manifest.json",
  openGraph: {
    title: "KNIRV-SERVER DVE",
    description: "The Crucible of Verifiable AI Intelligence - Powering Trustless Validation and Secure Execution",
    url: "https://knirv-server.network",
    siteName: "KNIRV-SERVER",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "KNIRV-SERVER DVE",
    description: "The Crucible of Verifiable AI Intelligence",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased bg-background text-foreground`}
      >
        <QueryProvider>
          <PWAProvider>
            <AuthProvider>
              <DemoModeProvider>
                <DHTProvider>
                  <OnboardingProvider>
                    <PWAInstallButton />
                    <HudOverlay />
                    {children}
                    <Toaster />
                  </OnboardingProvider>
                </DHTProvider>
              </DemoModeProvider>
            </AuthProvider>
          </PWAProvider>
        </QueryProvider>
      </body>
    </html>
  );
}
