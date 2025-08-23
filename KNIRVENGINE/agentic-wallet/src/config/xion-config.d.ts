import { MetaAccountConfig } from '../xion-meta-accounts';
export declare const XION_TESTNET_CONFIG: MetaAccountConfig;
export declare const XION_MAINNET_CONFIG: MetaAccountConfig;
export declare const getXionConfig: (network?: "testnet" | "mainnet") => MetaAccountConfig;
export declare const NETWORK_ENDPOINTS: {
    testnet: {
        rpc: string;
        rest: string;
        explorer: string;
    };
    mainnet: {
        rpc: string;
        rest: string;
        explorer: string;
    };
};
//# sourceMappingURL=xion-config.d.ts.map