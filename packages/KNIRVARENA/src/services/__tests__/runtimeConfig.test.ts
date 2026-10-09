import {
  MAINNET_SERVER_URL, TESTNET_SERVER_URL, getKnirvNetwork, getKnirvServerUrl, loadRuntimeConfig, setRuntimeConfig,
} from '../../config/runtimeConfig';

describe('arena runtime config', () => {
  const env = process.env;
  beforeEach(() => {
    process.env = { ...env };
    delete process.env.VITE_KNIRVSERVER_URL;
    delete process.env.VITE_KNIRV_GATEWAY_URL;
    setRuntimeConfig(null);
  });
  afterAll(() => { process.env = env; });

  it('defaults to the testnet gateway', () => {
    expect(getKnirvServerUrl()).toBe(TESTNET_SERVER_URL);
    expect(getKnirvNetwork()).toBe('testnet');
  });

  it('uses the env override only when KNIRVSERVER published no config', () => {
    process.env.VITE_KNIRVSERVER_URL = 'http://localhost:8082/';
    expect(getKnirvServerUrl()).toBe('http://localhost:8082');
    setRuntimeConfig({ network: 'mainnet', serverUrl: MAINNET_SERVER_URL });
    expect(getKnirvServerUrl()).toBe(MAINNET_SERVER_URL);
    expect(getKnirvNetwork()).toBe('mainnet');
  });

  it('loads the network KNIRVSERVER publishes (mainnet under -prod)', async () => {
    const fetchMock = jest.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ network: 'mainnet', serverUrl: 'https://gateway.knirv.com/' }) });
    (global as unknown as { fetch: unknown }).fetch = fetchMock;
    await expect(loadRuntimeConfig()).resolves.toEqual({ network: 'mainnet', serverUrl: MAINNET_SERVER_URL });
    expect(String(fetchMock.mock.calls[0][0])).toMatch(/runtime-config\.json$/);
    expect(getKnirvServerUrl()).toBe(MAINNET_SERVER_URL);
  });

  it('falls back to testnet when the config is missing or malformed', async () => {
    (global as unknown as { fetch: unknown }).fetch = jest.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ network: 'devnet' }) });
    await expect(loadRuntimeConfig()).resolves.toBeNull();
    expect(getKnirvServerUrl()).toBe(TESTNET_SERVER_URL);
  });
});
