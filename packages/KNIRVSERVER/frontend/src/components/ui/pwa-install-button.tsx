"use client";

import { Button } from "@/components/ui/button";
import { Download, X } from "lucide-react";
import { usePWAInstall } from "@/hooks/usePWAInstall";

export function PWAInstallButton() {
  const { installPrompt, isInstalled, install, dismiss } = usePWAInstall();

  if (isInstalled) {
    return null;
  }

  if (!installPrompt) {
    return null;
  }

  return (
    <div className="flex items-center gap-2 pwa-install-prompt">
      <Button
        onClick={install}
        size="sm"
        className="bg-primary/90 hover:bg-primary"
      >
        <Download className="w-4 h-4 mr-2" />
        Install App
      </Button>
      <Button
        onClick={dismiss}
        size="sm"
        variant="ghost"
        className="text-muted-foreground hover:text-foreground"
      >
        <X className="w-4 h-4" />
      </Button>
    </div>
  );
}