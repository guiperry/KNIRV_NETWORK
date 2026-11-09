import {
  getKnirvGatewayUrl,
  getLocalGatewayUrl,
  setRuntimeConfig,
  TESTNET_SERVER_URL,
  MAINNET_SERVER_URL,
} from '../../../src/config/runtimeConfig';

describe('local gateway (supplied by the desktop Client only)', () => {
  afterEach(() => setRuntimeConfig(null));

  it('is absent unless the Client supplies one', () => {
    setRuntimeConfig({ network: 'testnet', serverUrl: TESTNET_SERVER_URL });
    expect(getLocalGatewayUrl()).toBeNull();
  });

  it('is returned from the Client-supplied config, without a trailing slash', () => {
    setRuntimeConfig({ network: 'testnet', serverUrl: TESTNET_SERVER_URL, localGatewayUrl: 'http://127.0.0.1:8080/' });
    expect(getLocalGatewayUrl()).toBe('http://127.0.0.1:8080');
  });

  it('never replaces the network gateway', () => {
    setRuntimeConfig({ network: 'mainnet', serverUrl: MAINNET_SERVER_URL, localGatewayUrl: 'http://127.0.0.1:8080' });
    expect(getKnirvGatewayUrl()).toBe(MAINNET_SERVER_URL);
  });
});
