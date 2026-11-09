/**
 * Which KNIRVSERVER (and its gateway; they share one origin) the arena talks
 * to. KNIRVSERVER serves the embedded arena bundle and publishes its
 * deployment class at /arena/runtime-config.json: testnet by default, mainnet
 * when the server runs with -prod. VITE_KNIRVSERVER_URL / VITE_KNIRV_GATEWAY_URL
 * only apply when no server config is available (standalone `vite dev`).
 */

export const TESTNET_SERVER_URL = 'https://testnet-gateway.knirv.com';
export const MAINNET_SERVER_URL = 'https://gateway.knirv.com';

export interface ArenaRuntimeConfig {
  network: 'testnet' | 'mainnet';
  serverUrl: string;
  /**
   * KNIRVGATEWAY running on this computer, supplied by the desktop Client only.
   * Never a testnet or mainnet endpoint; absent in every server-published build.
   */
  localGatewayUrl?: string;
}

let runtimeConfig: ArenaRuntimeConfig | null = null;

const trim = (url: string) => url.replace(/\/$/, '');

const isRuntimeConfig = (value: unknown): value is ArenaRuntimeConfig => {
  if (!value || typeof value !== 'object') return false;
  const v = value as Record<string, unknown>;
  if (v.localGatewayUrl !== undefined && (typeof v.localGatewayUrl !== 'string' || !/^https?:\/\//.test(v.localGatewayUrl))) return false;
  return (v.network === 'testnet' || v.network === 'mainnet') && typeof v.serverUrl === 'string' && v.serverUrl !== '';
};

/** Fetches the server-published config once; resolves null when unavailable. */
export const loadRuntimeConfig = async (timeoutMs = 3000): Promise<ArenaRuntimeConfig | null> => {
  if (runtimeConfig) return runtimeConfig;
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(`${import.meta.env.BASE_URL || '/'}runtime-config.json`, {
      cache: 'no-store',
      signal: controller.signal,
    });
    if (!response.ok) return null;
    const body: unknown = await response.json();
    if (isRuntimeConfig(body)) {
      runtimeConfig = {
        network: body.network,
        serverUrl: trim(body.serverUrl),
        ...(body.localGatewayUrl ? { localGatewayUrl: trim(body.localGatewayUrl) } : {}),
      };
    }
  } catch {
    /* standalone dev server or offline: fall back to env/testnet */
  } finally {
    clearTimeout(timer);
  }
  return runtimeConfig;
};

/** Test hook: set or clear the loaded config. */
export const setRuntimeConfig = (config: ArenaRuntimeConfig | null): void => {
  runtimeConfig = config
    ? { ...config, serverUrl: trim(config.serverUrl), ...(config.localGatewayUrl ? { localGatewayUrl: trim(config.localGatewayUrl) } : {}) }
    : null;
};

/** KNIRVSERVER origin: server config → env override → testnet. */
export const getKnirvServerUrl = (): string => {
  if (runtimeConfig) return runtimeConfig.serverUrl;
  // Read each variable by name: `import.meta.env` used as a value makes Vite
  // inline every VITE_* variable into the bundle.
  const override = import.meta.env.VITE_KNIRVSERVER_URL || import.meta.env.VITE_KNIRV_GATEWAY_URL;
  return trim(override || TESTNET_SERVER_URL);
};

/** Gateway origin. Same origin as KNIRVSERVER in every deployment. */
export const getKnirvGatewayUrl = (): string => getKnirvServerUrl();

export const getKnirvNetwork = (): 'testnet' | 'mainnet' => {
  if (runtimeConfig) return runtimeConfig.network;
  return getKnirvServerUrl() === MAINNET_SERVER_URL ? 'mainnet' : 'testnet';
};

/**
 * The KNIRVGATEWAY on this computer, when the desktop Client supplies one.
 * Returns null otherwise, so no request is ever made to a guessed local port.
 */
export const getLocalGatewayUrl = (): string | null => {
  const url = runtimeConfig?.localGatewayUrl;
  return url ? trim(url) : null;
};
