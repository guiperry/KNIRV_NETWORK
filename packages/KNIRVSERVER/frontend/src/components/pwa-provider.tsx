"use client";

import { useEffect, useState } from "react";

export function PWAProvider({ children }: { children: React.ReactNode }) {
  const [swReady, setSwReady] = useState(false);

  useEffect(() => {
    if ("serviceWorker" in navigator) {
      const reloadForUpdatedWorker = () => window.location.reload();
      navigator.serviceWorker.addEventListener('controllerchange', reloadForUpdatedWorker);

      navigator.serviceWorker
        .register("/service-worker.js")
        .then((registration) => {
          console.log("Service Worker registered:", registration.scope);
          setSwReady(true);
        })
        .catch((error) => {
          console.error("Service Worker registration failed:", error);
        });

      return () => {
        navigator.serviceWorker.removeEventListener('controllerchange', reloadForUpdatedWorker);
      };
    }
  }, []);

  return <>{children}</>;
}
