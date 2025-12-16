// src/components/layout/Footer.tsx
import Link from 'next/link';

const Footer = () => {
  const currentYear = new Date().getFullYear();
  return (
    <footer className="border-t border-border/40 bg-background/95">
      <div className="container mx-auto py-6 px-4 md:px-6">
        <div className="flex flex-col md:flex-row items-center justify-between">
          <p className="text-sm text-muted-foreground">
            &copy; {currentYear} Operator Registry (AgentVerse). All rights reserved.
          </p>
          <nav className="flex gap-4 sm:gap-6 mt-4 md:mt-0">
            <Link
              href="/terms"
              className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              prefetch={false}
            >
              Terms of Service
            </Link>
            <Link
              href="/privacy"
              className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              prefetch={false}
            >
              Privacy Policy
            </Link>
          </nav>
        </div>
        <p className="mt-4 text-xs text-center text-muted-foreground/70">
          NANDA+ANS Security Blueprint Implementation
        </p>
      </div>
    </footer>
  );
};

export default Footer;
