import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Capacitor } from "@capacitor/core";
import { Keyboard, KeyboardResize } from "@capacitor/keyboard";
import "@/react-app/index.css";
import App from "@/react-app/App.tsx";

if (Capacitor.isNativePlatform()) {
  Keyboard.setResizeMode({ mode: KeyboardResize.Body });
}

const SW_CLEANUP_FLAG = 'knirv-sw-cleaned-dev';

async function cleanupDevelopmentServiceWorkers() {
  if (!('serviceWorker' in navigator) || !window.isSecureContext || import.meta.env.PROD) {
    return false;
  }
  if (sessionStorage.getItem(SW_CLEANUP_FLAG) === '1') { return false; }

  const registrations = await navigator.serviceWorker.getRegistrations();
  const hadController = Boolean(navigator.serviceWorker.controller);
  if (!registrations.length && !hadController) { return false; }

  await Promise.all(registrations.map((registration) => registration.unregister()));
  if ('caches' in window) {
    const cacheKeys = await caches.keys();
    await Promise.all(cacheKeys.map((key) => caches.delete(key)));
  }
  sessionStorage.setItem(SW_CLEANUP_FLAG, '1');
  window.location.reload();
  return true;
}

void cleanupDevelopmentServiceWorkers();

if ('serviceWorker' in navigator && window.isSecureContext && import.meta.env.PROD) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch((error) => {
      console.warn('PWA service worker registration failed:', error);
    });
  });
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
