/**
 * Links an error node into KNIRVENGINE for analysis.
 *
 * The desktop engine registers the knirvengine:// protocol and opens the
 * node in its error-node analysis view. When no desktop engine handles the
 * link, the same node opens in a browser-launched engine (`--browser`) via
 * its `?errorNode=` dashboard URL.
 */

// 8380: KNIRVENGINE's GUI port (8080 is KNIRVGATEWAY's default).
const DEFAULT_ENGINE_URL = 'http://localhost:8380';

// Read import.meta.env directly so Vite substitutes it at build time.
export const getKnirvEngineUrl = (): string =>
  (import.meta.env.VITE_KNIRVENGINE_URL || DEFAULT_ENGINE_URL).replace(/\/$/, '');

export const knirvEngineDeepLink = (errorNodeId: string): string =>
  `knirvengine://error-node/${encodeURIComponent(errorNodeId)}`;

export const knirvEngineBrowserLink = (errorNodeId: string, engineUrl = getKnirvEngineUrl()): string =>
  `${engineUrl}/dashboard?errorNode=${encodeURIComponent(errorNodeId)}`;

/**
 * Hands the node to the desktop engine. Resolves true when the OS took the
 * link (the page lost focus to the engine) and false when nothing handled it
 * within `waitMs`, so the caller can offer the browser engine instead.
 */
export const openInKnirvEngine = (
  errorNodeId: string,
  {
    waitMs = 1500,
    win = window,
    doc = document,
  }: { waitMs?: number; win?: Pick<Window, 'addEventListener' | 'removeEventListener'>; doc?: Document } = {}
): Promise<boolean> =>
  new Promise((resolve) => {
    let settled = false;
    const finish = (handled: boolean) => {
      if (settled) return;
      settled = true;
      win.removeEventListener('blur', onBlur);
      doc.removeEventListener('visibilitychange', onHidden);
      clearTimeout(timer);
      resolve(handled);
    };
    const onBlur = () => finish(true);
    const onHidden = () => {
      if (doc.visibilityState === 'hidden') finish(true);
    };
    win.addEventListener('blur', onBlur);
    doc.addEventListener('visibilitychange', onHidden);
    const timer = setTimeout(() => finish(false), waitMs);
    try {
      // An unhandled custom scheme is a silent no-op in browsers, so the
      // link is launched without navigating this page away.
      const anchor = doc.createElement('a');
      anchor.href = knirvEngineDeepLink(errorNodeId);
      anchor.rel = 'noopener';
      anchor.style.display = 'none';
      doc.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
    } catch {
      finish(false);
    }
  });
