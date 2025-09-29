export const XION_TESTNET_CONFIG = {
    chainId: 'xion-testnet-1',
    rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443',
    gasPrice: '0.025uxion',
    nrnTokenAddress: process.env.EXPO_PUBLIC_NRN_CONTRACT || 'xion1...',
    faucetAddress: process.env.EXPO_PUBLIC_FAUCET_CONTRACT || 'xion1...',
};
export const XION_MAINNET_CONFIG = {
    chainId: 'xion-mainnet-1',
    rpcEndpoint: 'https://rpc.xion-mainnet-1.burnt.com:443',
    gasPrice: '0.025uxion',
    nrnTokenAddress: process.env.EXPO_PUBLIC_NRN_CONTRACT_MAINNET || 'xion1...',
    faucetAddress: process.env.EXPO_PUBLIC_FAUCET_CONTRACT_MAINNET || 'xion1...',
};
export const getXionConfig = (network = 'testnet') => {
    return network === 'mainnet' ? XION_MAINNET_CONFIG : XION_TESTNET_CONFIG;
};
export const NETWORK_ENDPOINTS = {
    testnet: {
        rpc: 'https://rpc.xion-testnet-1.burnt.com:443',
        rest: 'https://api.xion-testnet-1.burnt.com',
        explorer: 'https://explorer.xion-testnet-1.burnt.com',
    },
    mainnet: {
        rpc: 'https://rpc.xion-mainnet-1.burnt.com:443',
        rest: 'https://api.xion-mainnet-1.burnt.com',
        explorer: 'https://explorer.xion-mainnet-1.burnt.com',
    },
};
//# sourceMappingURL=xion-config.js.map