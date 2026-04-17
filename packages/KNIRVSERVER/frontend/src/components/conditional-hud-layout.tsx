'use client';

import { usePathname } from 'next/navigation';
import { HudOverlay } from '@/components/hud';
import { PWAInstallButton } from '@/components/ui/pwa-install-button';
import { Toaster } from '@/components/ui/toaster';

const NO_HUD_ROUTES = ['/login', '/menu'];

export function ConditionalHudLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isNoHudRoute = NO_HUD_ROUTES.some(
    (route) => pathname === route || pathname?.startsWith(route + '/')
  );

  if (isNoHudRoute) {
    return (
      <div style={{ position: 'fixed', inset: 0, overflow: 'auto', background: '#000818' }}>
        {children}
        <Toaster />
      </div>
    );
  }

  return (
    <>
      <HudOverlay />
      <div
        id="hud-content-area"
        style={{
          position: 'fixed',
          top: '60px',
          left: '250px',
          right: '250px',
          bottom: '60px',
          overflow: 'auto',
          background: '#0a0a14',
          zIndex: 1,
        }}
      >
        {children}
      </div>
      <PWAInstallButton />
      <Toaster />
    </>
  );
}
