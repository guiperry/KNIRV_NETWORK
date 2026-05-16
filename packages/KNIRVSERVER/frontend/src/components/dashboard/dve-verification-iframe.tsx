'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { X, ExternalLink, Loader2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

// Port scanning — same logic as webgui-iframe-modal
let cachedGatewayPort: string | null = null;

async function checkHealth(port: string): Promise<boolean> {
  try {
    const resp = await fetch(`http://localhost:${port}/health`, {
      method: 'HEAD',
      signal: AbortSignal.timeout(2000),
    });
    return resp.ok || resp.type === 'opaque';
  } catch {
    return false;
  }
}

async function scanForGateway(): Promise<string> {
  if (cachedGatewayPort) return cachedGatewayPort;
  const envPort = process.env.NEXT_PUBLIC_GATEWAY_PORT;
  if (envPort) {
    if (await checkHealth(envPort)) { cachedGatewayPort = envPort; return cachedGatewayPort; }
  }
  const currentPort = String(window.location.port || '80');
  if (await checkHealth(currentPort)) { cachedGatewayPort = currentPort; return cachedGatewayPort; }
  for (const port of ['8081', '8080', '8090']) {
    if (await checkHealth(port)) { cachedGatewayPort = port; return cachedGatewayPort; }
  }
  return '8090';
}

function buildDvePageUrl(port: string, dveId: string): string {
  // When frontend dev server is on a different port, go through the gateway
  // which proxies /dve/{dveId}* to the backend Unix socket.
  const currentPort = String(window.location.port || '80');
  if (port === currentPort || port === '8090') {
    return `http://localhost:${port}/dve/${encodeURIComponent(dveId)}`;
  }
  return `http://localhost:${port}/dve/${encodeURIComponent(dveId)}`;
}

interface DVEVerificationIframeProps {
  isOpen: boolean;
  onClose: () => void;
  dveId: string;
  dveName?: string;
}

export function DVEVerificationIframe({ isOpen, onClose, dveId, dveName }: DVEVerificationIframeProps) {
  const [dvePageUrl, setDvePageUrl] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (isOpen && !dvePageUrl) {
      scanForGateway().then(port => {
        setDvePageUrl(buildDvePageUrl(port, dveId));
        setLoading(false);
      });
    }
  }, [isOpen, dveId]);

  const retryConnection = useCallback(async () => {
    setError(false);
    setLoading(true);
    try {
      const port = await scanForGateway();
      setDvePageUrl(buildDvePageUrl(port, dveId));
      setLoading(false);
    } catch {
      setLoading(false);
      setError(true);
    }
  }, [dveId]);

  const handleIframeMessage = useCallback((event: MessageEvent) => {
    // Listen for validation result messages from the DVE verification page
    if (event.data?.type === 'dve-validation-result') {
      // Validation complete — could show a toast or panel update
      console.log('[DVE Verification] Validation result:', event.data);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      window.addEventListener('message', handleIframeMessage);
      return () => window.removeEventListener('message', handleIframeMessage);
    }
  }, [isOpen, handleIframeMessage]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-[#0a0f1e] overflow-hidden">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-purple-900/40 bg-[#0a0f1e]/95 backdrop-blur-sm shrink-0">
        <div className="flex items-center space-x-3">
          <Button variant="ghost" size="sm" onClick={onClose}>
            <svg className="w-4 h-4 mr-1" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M19 12H5m7-7-7 7 7 7" />
            </svg>
            Back
          </Button>
          <span className="text-sm font-medium text-purple-300">DVE Public Verification</span>
          {dveName && <Badge variant="outline" className="text-xs text-purple-400 border-purple-700">{dveName}</Badge>}
        </div>
        <div className="flex items-center space-x-1">
          <a href={dvePageUrl} target="_blank" rel="noopener noreferrer">
            <Button variant="ghost" size="sm" title="Open in browser">
              <ExternalLink className="w-4 h-4" />
            </Button>
          </a>
          <Button variant="ghost" size="sm" onClick={onClose} title="Close">
            <X className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* Content area */}
      <div className="flex-1 relative overflow-hidden">
        {loading && !dvePageUrl && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-[#0a0f1e] z-10">
            <Loader2 className="w-8 h-8 animate-spin text-purple-400" />
            <span className="text-sm text-purple-300">Connecting to gateway...</span>
          </div>
        )}

        {error && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-4 bg-[#0a0f1e] z-10">
            <AlertCircle className="w-10 h-10 text-amber-400" />
            <div className="text-center space-y-1">
              <p className="text-sm font-medium text-amber-300">DVE page not available</p>
              <p className="text-xs text-muted-foreground">
                The KNIRVGATEWAY or KNIRVSERVER may not be running.
              </p>
            </div>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" onClick={retryConnection}>Retry</Button>
              <a href={dvePageUrl} target="_blank" rel="noopener noreferrer">
                <Button size="sm" variant="ghost">
                  <ExternalLink className="w-3 h-3 mr-1" />
                  Open in browser
                </Button>
              </a>
            </div>
          </div>
        )}

        {dvePageUrl && !error && (
          <iframe
            src={dvePageUrl}
            className="w-full h-full border-0"
            title={`DVE Public Verification — ${dveName || dveId}`}
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
            onLoad={() => setLoading(false)}
            onError={() => { setLoading(false); setError(true); }}
          />
        )}

        {loading && dvePageUrl && (
          <div className="absolute top-0 left-0 right-0 flex items-center justify-center h-10 bg-[#0a0f1e]/80 z-10">
            <Loader2 className="w-4 h-4 animate-spin text-purple-400 mr-2" />
            <span className="text-xs text-purple-300">Loading DVE page...</span>
          </div>
        )}
      </div>
    </div>
  );
}
