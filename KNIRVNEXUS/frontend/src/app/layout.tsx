import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Toaster } from "@/components/ui/toaster";
import { AuthProvider } from "@/lib/auth-context";
import { DemoModeProvider } from "@/contexts/demo-mode-context";
import { DHTProvider } from "@/contexts/dht-context";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "KNIRV-NEXUS DVE - Deterministic Validation Environment",
  description: "The Crucible of Verifiable AI Intelligence - Powering Trustless Validation, Secure Execution, and Collective Learning in the KNIRV D-TEN",
  keywords: ["KNIRV-NEXUS", "DVE", "Decentralized Validation", "AI Intelligence", "Trusted Execution", "CLEAN", "KNIRV D-TEN", "Blockchain", "TEE"],
  authors: [{ name: "KNIRV Network Team" }],
  openGraph: {
    title: "KNIRV-NEXUS DVE",
    description: "The Crucible of Verifiable AI Intelligence - Powering Trustless Validation and Secure Execution",
    url: "https://knirv-nexus.network",
    siteName: "KNIRV-NEXUS",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "KNIRV-NEXUS DVE",
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
        <AuthProvider>
          <DemoModeProvider>
            <DHTProvider>
              {children}
              <Toaster />
            </DHTProvider>
          </DemoModeProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
