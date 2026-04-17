import { useEffect, useState, useCallback } from 'react';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

declare global {
  interface Window {
    __pwaPrompt: BeforeInstallPromptEvent | null;
  }
}

interface UsePWAInstallReturn {
  installPrompt: BeforeInstallPromptEvent | null;
  isInstalled: boolean;
  isInstallable: boolean;
  install: () => Promise<void>;
  dismiss: () => void;
}

export function usePWAInstall(): UsePWAInstallReturn {
  const [installPrompt, setInstallPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const [isInstalled, setIsInstalled] = useState(false);
  const [isDismissed, setIsDismissed] = useState(false);

  useEffect(() => {
    // Pick up the event captured in the inline <head> script before React mounted
    if (window.__pwaPrompt && !isDismissed) {
      setInstallPrompt(window.__pwaPrompt);
    }

    const handlePrompt = (e: Event): void => {
      e.preventDefault();
      window.__pwaPrompt = e as BeforeInstallPromptEvent;
      if (!isDismissed) setInstallPrompt(e as BeforeInstallPromptEvent);
    };

    // Also fired by the early-capture script via custom event
    const handleReady = (): void => {
      if (window.__pwaPrompt && !isDismissed) setInstallPrompt(window.__pwaPrompt);
    };

    const handleInstalled = (): void => {
      setIsInstalled(true);
      setInstallPrompt(null);
    };

    window.addEventListener('beforeinstallprompt', handlePrompt);
    document.addEventListener('pwa-installable', handleReady);
    window.addEventListener('appinstalled', handleInstalled);

    if (window.matchMedia('(display-mode: standalone)').matches) {
      setIsInstalled(true);
    }

    return () => {
      window.removeEventListener('beforeinstallprompt', handlePrompt);
      document.removeEventListener('pwa-installable', handleReady);
      window.removeEventListener('appinstalled', handleInstalled);
    };
  }, [isDismissed]);

  const install = useCallback(async (): Promise<void> => {
    if (!installPrompt) return;
    installPrompt.prompt();
    const choice = await installPrompt.userChoice;
    if (choice.outcome === 'accepted') {
      console.log('PWA installation accepted');
    }
    setInstallPrompt(null);
  }, [installPrompt]);

  const dismiss = useCallback((): void => {
    setIsDismissed(true);
    setInstallPrompt(null);
  }, []);

  return {
    installPrompt,
    isInstalled,
    isInstallable: !isInstalled && !!installPrompt && !isDismissed,
    install,
    dismiss,
  };
}
