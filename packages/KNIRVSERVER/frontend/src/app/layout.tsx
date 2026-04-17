import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "@/lib/auth-context";
import { DemoModeProvider } from "@/contexts/demo-mode-context";
import { DHTProvider } from "@/contexts/dht-context";
import { OnboardingProvider } from "@/contexts/onboarding-context";
import QueryProvider from "@/components/providers/query-provider";
import { PWAProvider } from "@/components/pwa-provider";
import { ConditionalHudLayout } from "@/components/conditional-hud-layout";

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
      {/* Capture beforeinstallprompt before React hydrates — the event fires
          very early and would otherwise be missed by the useEffect listener */}
      <head>
        <script dangerouslySetInnerHTML={{ __html: `
          window.__pwaPrompt = null;
          window.addEventListener('beforeinstallprompt', function(e) {
            e.preventDefault();
            window.__pwaPrompt = e;
            document.dispatchEvent(new Event('pwa-installable'));
          });
        `}} />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
        style={{ overflow: 'hidden', background: '#000818', height: '100vh' }}
      >
        <QueryProvider>
          <PWAProvider>
            <AuthProvider>
              <DemoModeProvider>
                <DHTProvider>
                  <OnboardingProvider>
                    <ConditionalHudLayout>
                      {children}
                    </ConditionalHudLayout>
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
