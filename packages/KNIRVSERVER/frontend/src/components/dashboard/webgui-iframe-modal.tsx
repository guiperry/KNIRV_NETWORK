'use client';

import React, { useState, useEffect, useCallback, useRef } from 'react';
import { X, ArrowLeft, ExternalLink, Loader2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useAuth } from '@/lib/auth-context';

// Port scanning — same logic as p2p-transport-access-modal
let cachedGatewayPort: string | null = null;

type GatewayStatus = 'healthy' | 'found_but_down' | 'not_gateway';

// Check whether a port hosts the KNIRVGATEWAY.
//
// 'healthy'        — gateway is up, /health returns 200 with a nodeRole field.
// 'found_but_down' — gateway process is present but returning 503 (starting
//                    up or temporarily overloaded). We still record this port
//                    and show the error UI rather than landing on an unrelated
//                    service that happens to have a /health endpoint (e.g. KNIRVCHAIN).
// 'not_gateway'    — wrong service, connection refused, or timeout.
//
// GET (not HEAD) so we can read the JSON body and confirm it's the KNIRVGATEWAY
// via the `nodeRole` field that only its health handler emits (see
// packages/KNIRVGATEWAY/internal/server/server.go handleHealth) — KNIRVCHAIN's
// own /health returns only {status, timestamp, uptime}, with no nodeRole.
async function checkGatewayPort(port: string): Promise<GatewayStatus> {
  try {
    const resp = await fetch(`http://localhost:${port}/health`, {
      method: 'GET',
      signal: AbortSignal.timeout(2000),
    });
    if (resp.type === 'opaque') return 'healthy'; // no-cors opaque — assume ok
    if (resp.status === 503) return 'found_but_down';
    if (!resp.ok) return 'not_gateway';
    const data = await resp.json();
    if (typeof data.nodeRole === 'string' && data.status === 'healthy') return 'healthy';
    return 'not_gateway';
  } catch {
    return 'not_gateway'; // connection refused, timeout, or JSON parse error
  }
}

async function scanForGateway(): Promise<{ port: string; healthy: boolean }> {
  if (cachedGatewayPort) return { port: cachedGatewayPort, healthy: true };

  // Priority order: env override, then KNIRVGATEWAY's actual default (:8080,
  // see packages/KNIRVGATEWAY/internal/config/config.go), then dev override
  // (:8081), then the testnet-only override (:8888), then KNIRVCHAIN (:8090)
  // as last resort.
  const candidates = [
    process.env.NEXT_PUBLIC_GATEWAY_PORT,
    '8080', '8081', '8888', '8090',
  ].filter(Boolean) as string[];

  let foundButDown: string | null = null;

  for (const port of candidates) {
    const status = await checkGatewayPort(port);
    if (status === 'healthy') {
      cachedGatewayPort = port;
      return { port, healthy: true };
    }
    if (status === 'found_but_down' && !foundButDown) {
      foundButDown = port; // keep scanning in case a later port is healthy
    }
  }

  // Gateway present but not ready — use the identified port, report unhealthy.
  if (foundButDown) return { port: foundButDown, healthy: false };

  // Nothing found — fall back to the actual default and report unhealthy.
  return { port: '8080', healthy: false };
}

function buildWebGuiUrl(port: string, page?: string): string {
  // If the discovered gateway port matches the KNIRVSERVER's own port, traffic
  // is routed through the server's /gateway proxy — use that path prefix.
  // Any other port means the gateway is running standalone; serve from root.
  const serverPort = String(window.location.port || '80');
  if (port === serverPort) {
    return page
      ? `http://localhost:${port}/gateway/${encodeURIComponent(page)}`
      : `http://localhost:${port}/gateway`;
  }
  return page
    ? `http://localhost:${port}/${encodeURIComponent(page)}`
    : `http://localhost:${port}/dashboard`;
}

interface WebguiIframeModalProps {
  isOpen: boolean;
  onClose: () => void;
  page?: string; // WebGUI page id (e.g. 'chain-explorer', 'network-inference-dao')
}

export function WebguiIframeModal({ isOpen, onClose, page }: WebguiIframeModalProps) {
  const { user } = useAuth();
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const [webGuiUrl, setWebGuiUrl] = useState<string>('');
  const [webGuiLoading, setWebGuiLoading] = useState(false);
  const [webGuiError, setWebGuiError] = useState(false);

  useEffect(() => {
    if (isOpen && !webGuiUrl) {
      // Reset cached port so each open re-probes rather than reusing a stale
      // result from a previous scan (e.g. gateway restarted on a different port).
      cachedGatewayPort = null;
      setWebGuiError(false);
      setWebGuiLoading(true);
      scanForGateway().then(({ port, healthy }) => {
        setWebGuiUrl(buildWebGuiUrl(port, page));
        setWebGuiLoading(false);
        if (!healthy) setWebGuiError(true);
      });
    }
  }, [isOpen, page]);

  // Respond to KNIRV_AUTH_REQUEST from the WebGUI iframe.
  // The iframe sends this when it has no local auth and detects it is embedded.
  // We reply with the KNIRVSERVER's current user so the WebGUI can skip the
  // KNIRV.NETWORK redirect and authenticate directly.
  useEffect(() => {
    if (!isOpen) return;

    const handleAuthRequest = (event: MessageEvent) => {
      if (!event.data || event.data.type !== 'KNIRV_AUTH_REQUEST') return;
      if (!iframeRef.current) return;

      const token = localStorage.getItem('knirv_nexus_token') ?? '';
      const role  = user?.role ?? localStorage.getItem('knirv_nexus_role') ?? 'admin';

      iframeRef.current.contentWindow?.postMessage(
        { type: 'KNIRV_AUTH_RESPONSE', role, network: 'local', token },
        '*'
      );
    };

    window.addEventListener('message', handleAuthRequest);
    return () => window.removeEventListener('message', handleAuthRequest);
  }, [isOpen, user]);

  const openWebGui = useCallback(async () => {
    setWebGuiError(false);
    setWebGuiLoading(true);
    setWebGuiUrl(''); // force re-probe on retry
    cachedGatewayPort = null;
    const { port, healthy } = await scanForGateway();
    setWebGuiUrl(buildWebGuiUrl(port, page));
    setWebGuiLoading(false);
    if (!healthy) setWebGuiError(true);
  }, [page]);

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
            ref={iframeRef}
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
