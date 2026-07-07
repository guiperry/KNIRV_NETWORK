'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { ExternalLink, Loader2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';

// Build the DVE page URL using the current origin so the request routes
// through KNIRVSERVER's /dve/* proxy → gateway → backend socket.
function buildDvePageUrl(dveId: string): string {
  return `/dve/${encodeURIComponent(dveId)}`;
}

interface DVEVerificationIframeProps {
  isOpen: boolean;
  onClose: () => void;
  dveId?: string;
  dveName?: string;
}

export function DVEVerificationIframe({ isOpen, onClose, dveId, dveName }: DVEVerificationIframeProps) {
  const dvePageUrl = dveId ? buildDvePageUrl(dveId) : '';
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setLoading(true);
      setError(false);
    }
  }, [isOpen, dveId]);

  const retryConnection = useCallback(() => {
    setError(false);
    setLoading(true);
  }, []);

  const handleIframeMessage = useCallback((event: MessageEvent) => {
    if (event.data?.type === 'dve:validation:complete') {
      console.log('[DVE Verification] Validation complete:', event.data);
    }
    if (event.data?.type === 'dve:navigate:back' || event.data?.type === 'dve:navigate:close') {
      onClose();
    }
  }, [onClose]);

  useEffect(() => {
    if (isOpen) {
      window.addEventListener('message', handleIframeMessage);
      return () => window.removeEventListener('message', handleIframeMessage);
    }
  }, [isOpen, handleIframeMessage]);

  if (!isOpen) return null;

  return (
    <div className="absolute inset-0 z-50 flex flex-col bg-[#0a0f1e] overflow-hidden">
      {/* Loading overlay */}
      {loading && (
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-[#0a0f1e] z-10">
          <Loader2 className="w-8 h-8 animate-spin text-purple-400" />
          <span className="text-sm text-purple-300">Loading DVE File Manager...</span>
        </div>
      )}

      {/* Error state */}
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

      {/* Full-bleed iframe — the DVE File Manager page handles its own Back/Close */}
      {dvePageUrl && !error && (
        <iframe
          src={dvePageUrl}
          className="w-full h-full border-0"
          title={`DVE File Manager — ${dveName || dveId}`}
          onLoad={() => setLoading(false)}
          onError={() => { setLoading(false); setError(true); }}
        />
      )}
    </div>
  );
}
