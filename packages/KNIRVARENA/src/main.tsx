import React from 'react';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import ErrorBoundary from './components/ErrorBoundary';
import { loadRuntimeConfig } from './config/runtimeConfig';
import './index.css';

// Several modules resolve the KNIRVSERVER origin when they are first
// imported, so the server-published network (testnet, or mainnet under -prod)
// is loaded before the app module graph.
const start = async () => {
  await loadRuntimeConfig();
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
