"use client";

import { useEffect, useState } from "react";

/**
 * Unregisters any service worker registered at the origin root scope ("/").
 *
 * The app's worker used to register at "/" and so controlled the whole origin,
 * including the gateway WebGUI. It now registers under /app/ only. Browsers keep
 * an existing root registration until it is removed, so this retires it. It is
 * a no-op once no root-scope registration remains. Returns the number removed.
 */
export async function retireRootServiceWorkers(
  container: Pick<ServiceWorkerContainer, "getRegistrations">,
): Promise<number> {
  const registrations = await container.getRegistrations();
  let removed = 0;
  for (const registration of registrations) {
    let scopePath: string;
    try {
      scopePath = new URL(registration.scope).pathname;
    } catch {
      continue;
    }
    if (scopePath === "/") {
      if (await registration.unregister()) {
        removed += 1;
      }
    }
  }
  return removed;
}

export function PWAProvider({ children }: { children: React.ReactNode }) {
  const [swReady, setSwReady] = useState(false);

  useEffect(() => {
    if ("serviceWorker" in navigator) {
      const reloadForUpdatedWorker = () => window.location.reload();
      navigator.serviceWorker.addEventListener('controllerchange', reloadForUpdatedWorker);

      // Retire a stale root-scope worker first, then register the /app/ worker.
      // A cleanup failure must not block registration.
      retireRootServiceWorkers(navigator.serviceWorker)
        .catch((error) => {
          console.warn("Retiring root-scope service worker failed:", error);
        })
        .then(() =>
          navigator.serviceWorker.register("/app/service-worker.js", { scope: "/app/" }),
        )
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
