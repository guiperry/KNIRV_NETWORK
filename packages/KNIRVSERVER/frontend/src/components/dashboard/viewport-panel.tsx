'use client';

import React, { useState, useRef, useCallback } from 'react';
import { X, RefreshCw, Globe, AlertTriangle, ExternalLink, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

interface ViewportPanelProps {
  isOpen: boolean;
  onClose: () => void;
  nodeIp?: string;
  nodeName?: string;
  isSidebarOpen?: boolean;
  isMonitorOpen?: boolean;
}

const DEFAULT_PORT = 8080;

const ViewportPanel: React.FC<ViewportPanelProps> = ({
  isOpen,
  onClose,
  nodeIp,
  nodeName,
  isSidebarOpen,
  isMonitorOpen,
}) => {
  const defaultUrl = nodeIp ? `http://${nodeIp}:${DEFAULT_PORT}` : '';
  const [urlInput, setUrlInput] = useState(defaultUrl);
  const [loadedUrl, setLoadedUrl] = useState(defaultUrl);
  const [loadError, setLoadError] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [key, setKey] = useState(0); // used to force iframe reload
  const iframeRef = useRef<HTMLIFrameElement>(null);

  const navigate = useCallback((url: string) => {
    let target = url.trim();
    if (!target) return;
    if (!/^https?:\/\//i.test(target)) target = `http://${target}`;
    setLoadedUrl(target);
    setUrlInput(target);
    setLoadError(false);
    setIsLoading(true);
    setKey(k => k + 1);
  }, []);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') navigate(urlInput);
  };

  if (!isOpen) return null;

  const leftOffset = isSidebarOpen ? 300 : 0;
  const bottomOffset = isMonitorOpen ? 280 : 0;

  return (
    <div
      className="absolute top-0 right-0 bottom-0 z-40 flex flex-col bg-[#080c14] border-l border-blue-600/30 transition-interactive duration-200"
      style={{
        left: leftOffset,
        bottom: bottomOffset,
        width: `calc(100% - ${leftOffset}px)`,
      }}
    >
      {/* Toolbar */}
      <div className="flex items-center gap-2 h-10 px-3 border-b border-slate-800 bg-slate-900/80 shrink-0">
        <Globe className="w-4 h-4 text-blue-400 shrink-0" />
        <div className="flex-1 flex items-center gap-1">
          <Input
            value={urlInput}
            onChange={e => setUrlInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={`http://${nodeIp || '<node-ip>'}:${DEFAULT_PORT}`}
            className="h-7 text-xs bg-slate-950 border-slate-700 font-mono text-slate-200 px-2"
          />
          <Button
            size="sm"
            variant="ghost"
            className="h-7 w-7 p-0 text-slate-400 hover:text-blue-300"
            onClick={() => navigate(urlInput)}
            title="Go"
          >
            <ChevronRight className="w-4 h-4" />
          </Button>
        </div>
        <Button
          size="sm"
          variant="ghost"
          className="h-7 w-7 p-0 text-slate-400 hover:text-blue-300"
          onClick={() => { setLoadError(false); setIsLoading(true); setKey(k => k + 1); }}
          title="Reload"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
        </Button>
        <a
          href={loadedUrl || undefined}
          target="_blank"
          rel="noreferrer"
          className="text-slate-400 hover:text-blue-300 p-1"
          title="Open in new tab"
        >
          <ExternalLink className="w-3.5 h-3.5" />
        </a>
        <button
          onClick={onClose}
          className="ml-1 text-slate-500 hover:text-red-400 p-1 rounded transition-colors"
          title="Close Viewport"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* Viewport area */}
      <div className="flex-1 relative overflow-hidden bg-white">
        {loadError ? (
          <div className="absolute inset-0 flex flex-col items-center justify-center bg-[#080c14] gap-4 text-center p-8">
            <AlertTriangle className="w-12 h-12 text-yellow-500" />
            <div>
              <p className="text-slate-200 font-semibold mb-1">Cannot display in viewport</p>
              <p className="text-slate-400 text-sm">
                The container's interface at <span className="font-mono text-blue-400">{loadedUrl}</span> refused
                to be embedded (X-Frame-Options or CSP). You can still open it directly:
              </p>
            </div>
            <a
              href={loadedUrl}
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
            >
              <ExternalLink className="w-4 h-4" />
              Open in Browser
            </a>
          </div>
        ) : !loadedUrl ? (
          <div className="absolute inset-0 flex flex-col items-center justify-center bg-[#080c14] gap-3 text-slate-400">
            <Globe className="w-12 h-12 text-slate-700" />
            <p className="text-sm">Enter the container URL above to load the interface.</p>
            {nodeIp && (
              <button
                onClick={() => navigate(defaultUrl)}
                className="text-blue-400 hover:underline text-xs font-mono"
              >
                {defaultUrl}
              </button>
            )}
          </div>
        ) : (
          <iframe
            key={key}
            ref={iframeRef}
            src={loadedUrl}
            className="w-full h-full border-0"
            title={`DVE Viewport — ${nodeName || 'Container UI'}`}
            onLoad={() => setIsLoading(false)}
            onError={() => { setIsLoading(false); setLoadError(true); }}
          />
        )}
      </div>

      {/* Status bar */}
      <div className="h-6 px-3 flex items-center gap-2 bg-slate-950/80 border-t border-slate-800 shrink-0">
        {isLoading && !loadError && (
          <span className="text-[10px] text-blue-400 animate-pulse">Loading...</span>
        )}
        {!isLoading && !loadError && loadedUrl && (
          <span className="text-[10px] text-green-500">Connected to {nodeName || 'container'}</span>
        )}
        <span className="ml-auto text-[10px] text-slate-600 font-mono">{loadedUrl}</span>
      </div>
    </div>
  );
};

export default ViewportPanel;
