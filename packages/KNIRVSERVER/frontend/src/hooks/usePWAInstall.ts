import { useEffect, useState, useCallback } from 'react';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
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
    const handleBeforeInstallPrompt = (e: Event): void => {
      e.preventDefault();
      if (!isDismissed) {
        setInstallPrompt(e as BeforeInstallPromptEvent);
      }
    };

    const handleAppInstalled = (): void => {
      setIsInstalled(true);
      setInstallPrompt(null);
    };

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
    window.addEventListener('appinstalled', handleAppInstalled);

    if (window.matchMedia('(display-mode: standalone)').matches) {
      setIsInstalled(true);
    }

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
      window.removeEventListener('appinstalled', handleAppInstalled);
    };
  }, [isDismissed]);

  const install = useCallback(async (): Promise<void> => {
    if (!installPrompt) {
      return;
    }

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