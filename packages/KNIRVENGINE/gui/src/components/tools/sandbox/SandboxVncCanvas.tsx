import React, { useEffect, useRef } from 'react';
import RFB from '@novnc/novnc';

interface SandboxVncCanvasProps {
  wsUrl: string | undefined;
  viewOnly?: boolean;
  quality?: number;
  onStatus?: (connected: boolean) => void;
}

/**
 * Renders a live noVNC RFB stream for the sandbox's framebuffer. Shared by the
 * persistent dock and the standalone noVNC tool page so both use the exact same
 * connection lifecycle. The dock and the page can be mounted simultaneously
 * because `SandboxManager` launches x11vnc with `-shared`.
 */
export const SandboxVncCanvas: React.FC<SandboxVncCanvasProps> = ({
  wsUrl,
  viewOnly = false,
  quality = 6,
  onStatus,
}) => {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const rfbRef = useRef<RFB | null>(null);

  useEffect(() => {
    if (!wsUrl || !containerRef.current) {
      onStatus?.(false);
      return;
    }

    let rfb: RFB;
    try {
      rfb = new RFB(containerRef.current, wsUrl, {
        shared: true,
        viewOnly,
        qualityLevel: quality,
      });
      rfb.background = 'rgb(0,0,0)';
      rfb.scaleViewport = true;
      // Xvfb has a fixed server-side geometry. Requesting an RFB resize makes
      // x11vnc emit an administrative-prohibition warning without changing it.
      rfb.resizeSession = false;
      rfbRef.current = rfb;

      const handleConnect = () => onStatus?.(true);
      const handleDisconnect = () => onStatus?.(false);
      rfb.addEventListener('connect', handleConnect);
      rfb.addEventListener('disconnect', handleDisconnect);

      return () => {
        rfb.removeEventListener('connect', handleConnect);
        rfb.removeEventListener('disconnect', handleDisconnect);
        rfb.disconnect();
        rfbRef.current = null;
      };
    } catch {
      onStatus?.(false);
      return;
    }
  }, [wsUrl, viewOnly, quality, onStatus]);

  // The RFB client creates a 100%-sized child. In the dock this needs to be a
  // bounded flex item (rather than a percentage height in an auto-sized grid
  // cell), otherwise the client can connect successfully with a 0px viewport.
  return <div ref={containerRef} className="min-h-0 min-w-0 flex-1 overflow-hidden bg-black" />;
};

export default SandboxVncCanvas;
