// src/components/layout/Header.tsx
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet';
import { Menu, BotMessageSquare } from 'lucide-react';

const Header = () => {
  const navItems = [
    { href: '/', label: 'Discover Operators' },
    { href: '/register', label: 'Register Operator' },
    // Add more navigation items as needed
  ];

  return (
    <header className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-16 max-w-screen-2xl items-center justify-between">
          <Link href="/" className="flex items-center gap-2" prefetch={false}>
          <BotMessageSquare className="h-7 w-7 text-primary" />
          <span className="text-xl font-bold tracking-tight text-foreground">
            Operator Registry <span className="text-primary">(AgentVerse)</span> - NANDA+ANS
          </span>
        </Link>

        <nav className="hidden md:flex items-center gap-6 text-sm font-medium">
          {navItems.map((item) => (
            <Link
              key={item.label}
              href={item.href}
              className="text-foreground/80 transition-colors hover:text-foreground"
              prefetch={false}
            >
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="md:hidden">
          <Sheet>
            <SheetTrigger asChild>
              <Button variant="outline" size="icon">
                <Menu className="h-5 w-5" />
                <span className="sr-only">Toggle Menu</span>
              </Button>
            </SheetTrigger>
            <SheetContent side="right">
              <div className="grid gap-4 py-6">
                <Link href="/" className="flex items-center gap-2 mb-4 px-2" prefetch={false}>
                  <BotMessageSquare className="h-6 w-6 text-primary" />
                  <span className="text-lg font-semibold text-foreground">Operator Registry - NANDA+ANS</span>
                </Link>
                {navItems.map((item) => (
                  <Link
                    key={item.label}
                    href={item.href}
                    className="block px-2 py-2 text-base font-medium text-foreground/80 hover:bg-accent hover:text-accent-foreground rounded-md"
                    prefetch={false}
                  >
                    {item.label}
                  </Link>
                ))}
              </div>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  );
};

export default Header;
