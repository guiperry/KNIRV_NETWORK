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

export const XION_TESTNET_CONFIG: XionConfig = {
  chainId: 'xion-testnet-1',
  rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443',
  restEndpoint: 'https://api.xion-testnet-1.burnt.com:443',
  gasPrice: '0.025unrn',
  gasAdjustment: 1.5,
  prefix: 'xion',
  coinType: 118,
  hdPath: "m/44'/118'/0'/0/0",
  faucetUrl: 'https://faucet.xion-testnet-1.burnt.com',
  explorerUrl: 'https://explorer.xion-testnet-1.burnt.com'
};

export const XION_MAINNET_CONFIG: XionConfig = {
  chainId: 'xion-mainnet-1',
  rpcEndpoint: 'https://rpc.xion-mainnet-1.burnt.com:443',
  restEndpoint: 'https://api.xion-mainnet-1.burnt.com:443',
  gasPrice: '0.025unrn',
  gasAdjustment: 1.5,
  prefix: 'xion',
  coinType: 118,
  hdPath: "m/44'/118'/0'/0/0",
  explorerUrl: 'https://explorer.xion-mainnet-1.burnt.com'
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
