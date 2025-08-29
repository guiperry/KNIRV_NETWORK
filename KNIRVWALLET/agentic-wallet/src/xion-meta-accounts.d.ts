export interface MetaAccountConfig {
    chainId: string;
    rpcEndpoint: string;
    gasPrice: string;
    nrnTokenAddress: string;
    faucetAddress: string;
}
export declare class XionMetaAccount {
    private signer;
    private client;
    private config;
    private address;
    constructor(config: MetaAccountConfig);
    initialize(mnemonic?: string): Promise<void>;
    getAddress(): Promise<string>;
    getMnemonic(): Promise<string>;
    getNRNBalance(): Promise<string>;
    transferNRN(recipient: string, amount: string): Promise<string>;
    requestNRNFromFaucet(usdcAmount: string): Promise<string>;
    burnNRNForSkill(skillId: string, amount: string): Promise<string>;
    enableGaslessTransactions(): Promise<void>;
}
export declare class WalletManager {
    private metaAccounts;
    private config;
    constructor(config: MetaAccountConfig);
    createWallet(name: string): Promise<XionMetaAccount>;
    importWallet(name: string, mnemonic: string): Promise<XionMetaAccount>;
    getWallet(name: string): Promise<XionMetaAccount | undefined>;
    listWallets(): Promise<string[]>;
    private saveWallet;
    private loadWallet;
    private encrypt;
    private decrypt;
}
//# sourceMappingURL=xion-meta-accounts.d.ts.map