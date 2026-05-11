'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { X, ArrowLeft, ExternalLink, Loader2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

// Port scanning — same logic as p2p-transport-access-modal
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

function buildWebGuiUrl(port: string, page?: string): string {
  // Match the P2PTransportAccessModal pattern: use /gateway proxy when on wrapper port
  if (port === String(window.location.port || '80') || port === '8090') {
    if (page) return `http://localhost:${port}/gateway/${encodeURIComponent(page)}`;
    return `http://localhost:${port}/gateway`;
  }
  if (page) return `http://localhost:${port}/${encodeURIComponent(page)}`;
  return `http://localhost:${port}/dashboard`;
}

interface WebguiIframeModalProps {
  isOpen: boolean;
  onClose: () => void;
  page?: string; // WebGUI page id (e.g. 'chain-explorer', 'network-inference-dao')
}

export function WebguiIframeModal({ isOpen, onClose, page }: WebguiIframeModalProps) {
  const [webGuiUrl, setWebGuiUrl] = useState<string>('');
  const [webGuiLoading, setWebGuiLoading] = useState(false);
  const [webGuiError, setWebGuiError] = useState(false);

  useEffect(() => {
    if (isOpen && !webGuiUrl) {
      scanForGateway().then(port => setWebGuiUrl(buildWebGuiUrl(port, page)));
    }
  }, [isOpen, page]);

  const openWebGui = useCallback(async () => {
    setWebGuiError(false);
    setWebGuiLoading(true);
    try {
      const port = await scanForGateway();
      const baseUrl = `http://localhost:${port}`;
      await fetch(`${baseUrl}/health`, { method: 'HEAD', mode: 'no-cors', signal: AbortSignal.timeout(4000) });
      setWebGuiLoading(false);
    } catch {
      setWebGuiLoading(false);
      setWebGuiError(true);
    }
  }, []);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-[#0a0f1e] overflow-hidden">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-indigo-900/40 bg-[#0a0f1e]/95 backdrop-blur-sm shrink-0">
        <div className="flex items-center space-x-3">
          <Button variant="ghost" size="sm" onClick={onClose}>
            <ArrowLeft className="w-4 h-4 mr-1" />
            Back
          </Button>
          <span className="text-sm font-medium text-indigo-300">KNIRV Network WebGUI</span>
          {page && <Badge variant="outline" className="text-xs text-indigo-400 border-indigo-700">{page}</Badge>}
        </div>
        <div className="flex items-center space-x-1">
          <a href={webGuiUrl} target="_blank" rel="noopener noreferrer">
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
        {webGuiLoading && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-[#0a0f1e] z-10">
            <Loader2 className="w-8 h-8 animate-spin text-indigo-400" />
            <span className="text-sm text-indigo-300">Connecting to gateway...</span>
          </div>
        )}

        {!webGuiLoading && webGuiError && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-4 bg-[#0a0f1e] z-10">
            <AlertCircle className="w-10 h-10 text-amber-400" />
            <div className="text-center space-y-1">
              <p className="text-sm font-medium text-amber-300">Gateway not reachable</p>
              <p className="text-xs text-muted-foreground">
                Ensure the KNIRVGATEWAY service is running.
              </p>
            </div>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" onClick={openWebGui}>Retry</Button>
              <a href={webGuiUrl} target="_blank" rel="noopener noreferrer">
                <Button size="sm" variant="ghost">
                  <ExternalLink className="w-3 h-3 mr-1" />
                  Open in browser
                </Button>
              </a>
            </div>
          </div>
        )}

        {!webGuiError && (
          <iframe
            src={webGuiLoading ? undefined : webGuiUrl}
            className="w-full h-full border-0"
            title="KNIRV Network WebGUI"
            onLoad={() => setWebGuiLoading(false)}
            onError={() => { setWebGuiLoading(false); setWebGuiError(true); }}
          />
        )}
      </div>
    </div>
  );
}
