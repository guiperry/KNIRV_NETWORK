export interface XionConfig {
  chainId: string;
  rpcEndpoint: string;
  restEndpoint?: string;
  gasPrice: string;
  gasAdjustment: number;
  prefix: string;
  coinType: number;
  hdPath: string;
  faucetUrl?: string;
  explorerUrl?: string;
}

const gatewayOrigin = ((import.meta as ImportMeta & { env?: Record<string, string> }).env?.VITE_KNIRV_GATEWAY_URL || 'https://gateway.knirv.network').replace(/\/$/, '');

export const XION_TESTNET_CONFIG: XionConfig = {
  chainId: 'xion-testnet-2',
  rpcEndpoint: `${gatewayOrigin}/xion/rpc`,
  restEndpoint: `${gatewayOrigin}/xion/rest`,
  gasPrice: '0.025uxion',
  gasAdjustment: 1.5,
  prefix: 'xion',
  coinType: 118,
  hdPath: "m/44'/118'/0'/0/0",
  explorerUrl: `${gatewayOrigin}/xion/rpc`
};

export const XION_MAINNET_CONFIG: XionConfig = {
  chainId: 'xion-mainnet-1',
  rpcEndpoint: `${gatewayOrigin}/xion/rpc`,
  restEndpoint: `${gatewayOrigin}/xion/rest`,
  gasPrice: '0.025uxion',
  gasAdjustment: 1.5,
  prefix: 'xion',
  coinType: 118,
  hdPath: "m/44'/118'/0'/0/0",
  explorerUrl: `${gatewayOrigin}/xion/rpc`
};

export function getXionConfig(network: 'testnet' | 'mainnet' = 'testnet'): XionConfig {
  switch (network) {
    case 'mainnet':
      return XION_MAINNET_CONFIG;
    case 'testnet':
    default:
      return XION_TESTNET_CONFIG;
  }
}

export function validateXionConfig(config: XionConfig): boolean {
  return !!(
    config.chainId &&
    config.rpcEndpoint &&
    config.gasPrice &&
    config.gasAdjustment > 0 &&
    config.prefix &&
    config.coinType >= 0 &&
    config.hdPath
  );
}

export default {
  getXionConfig,
  validateXionConfig,
  XION_TESTNET_CONFIG,
  XION_MAINNET_CONFIG
};
