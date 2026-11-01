import React from 'react';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import ErrorBoundary from './components/ErrorBoundary';
import { initLocal } from '@knirv/sdk/signing';
import { loadRuntimeConfig } from './config/runtimeConfig';
import './index.css';

// Several modules resolve the KNIRVSERVER origin when they are first
// imported, so the server-published network (testnet, or mainnet under -prod)
// is loaded before the app module graph.
// The SDK's Rust core (signing, relay envelopes, model status) is a WebAssembly
// module that browsers will not compile synchronously on the main thread.
const start = async () => {
  await Promise.all([loadRuntimeConfig(), initLocal()]);
  const { default: App } = await import('./App');
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <ErrorBoundary>
        <App />
      </ErrorBoundary>
    </StrictMode>
  );
};

void start();
