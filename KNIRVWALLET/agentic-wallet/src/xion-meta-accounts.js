import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { SigningCosmWasmClient } from '@cosmjs/cosmwasm-stargate';
import { GasPrice } from '@cosmjs/stargate';
export class XionMetaAccount {
    signer = null;
    client = null;
    config;
    address = '';
    constructor(config) {
        this.config = config;
    }
    async initialize(mnemonic) {
        // Create or restore wallet
        if (mnemonic) {
            this.signer = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
                prefix: 'xion',
            });
        }
        else {
            this.signer = await DirectSecp256k1HdWallet.generate(24, {
                prefix: 'xion',
            });
        }
        // Get address
        const accounts = await this.signer.getAccounts();
        this.address = accounts[0].address;
        // Initialize signing client
        this.client = await SigningCosmWasmClient.connectWithSigner(this.config.rpcEndpoint, this.signer, {
            gasPrice: GasPrice.fromString(this.config.gasPrice),
        });
    }
    async getAddress() {
        return this.address;
    }
    async getMnemonic() {
        if (!this.signer) {
            throw new Error('Wallet not initialized');
        }
        return this.signer.mnemonic;
    }
    async getNRNBalance() {
        if (!this.client) {
            throw new Error('Client not initialized');
        }
        const queryMsg = { balance: { address: this.address } };
        try {
            const result = await this.client.queryContractSmart(this.config.nrnTokenAddress, queryMsg);
            return result.balance;
        }
        catch (error) {
            console.error('Error querying NRN balance:', error);
            return '0';
        }
    }
    async transferNRN(recipient, amount) {
        if (!this.client) {
            throw new Error('Client not initialized');
        }
        const executeMsg = {
            transfer: {
                recipient,
                amount,
            },
        };
        const fee = {
            amount: [{ denom: 'uxion', amount: '5000' }],
            gas: '200000',
        };
        try {
            const result = await this.client.execute(this.address, this.config.nrnTokenAddress, executeMsg, fee);
            return result.transactionHash;
        }
        catch (error) {
            throw new Error(`Transfer failed: ${error.message}`);
        }
    }
    async requestNRNFromFaucet(usdcAmount) {
        if (!this.client) {
            throw new Error('Client not initialized');
        }
        const executeMsg = {
            exchange_usdc_for_nrn: {
                usdc_amount: usdcAmount,
                recipient: this.address,
            },
        };
        const fee = {
            amount: [{ denom: 'uxion', amount: '10000' }],
            gas: '300000',
        };
        try {
            const result = await this.client.execute(this.address, this.config.faucetAddress, executeMsg, fee);
            return result.transactionHash;
        }
        catch (error) {
            throw new Error(`Faucet request failed: ${error.message}`);
        }
    }
    async burnNRNForSkill(skillId, amount) {
        if (!this.client) {
            throw new Error('Client not initialized');
        }
        const executeMsg = {
            burn_for_skill: {
                skill_id: skillId,
                amount,
            },
        };
        const fee = {
            amount: [{ denom: 'uxion', amount: '8000' }],
            gas: '250000',
        };
        try {
            const result = await this.client.execute(this.address, this.config.nrnTokenAddress, executeMsg, fee);
            return result.transactionHash;
        }
        catch (error) {
            throw new Error(`Skill invocation failed: ${error.message}`);
        }
    }
    async enableGaslessTransactions() {
        // Implementation for XION's account abstraction
        // This would involve setting up meta account permissions
        const setupMsg = {
            setup_meta_account: {
                owner: this.address,
                permissions: {
                    allow_gasless: true,
                    allowed_contracts: [
                        this.config.nrnTokenAddress,
                        this.config.faucetAddress,
                    ],
                },
            },
        };
        // Execute setup transaction
        // This is a placeholder - actual implementation depends on XION's AA system
        console.log('Setting up gasless transactions...', setupMsg);
    }
}
export class WalletManager {
    metaAccounts = new Map();
    config;
    constructor(config) {
        this.config = config;
    }
    async createWallet(name) {
        const metaAccount = new XionMetaAccount(this.config);
        await metaAccount.initialize();
        this.metaAccounts.set(name, metaAccount);
        // Save wallet to secure storage
        await this.saveWallet(name, await metaAccount.getMnemonic());
        return metaAccount;
    }
    async importWallet(name, mnemonic) {
        const metaAccount = new XionMetaAccount(this.config);
        await metaAccount.initialize(mnemonic);
        this.metaAccounts.set(name, metaAccount);
        // Save wallet to secure storage
        await this.saveWallet(name, mnemonic);
        return metaAccount;
    }
    async getWallet(name) {
        if (this.metaAccounts.has(name)) {
            return this.metaAccounts.get(name);
        }
        // Try to load from storage
        const mnemonic = await this.loadWallet(name);
        if (mnemonic) {
            return await this.importWallet(name, mnemonic);
        }
        return undefined;
    }
    async listWallets() {
        // Return list of saved wallet names
        return Array.from(this.metaAccounts.keys());
    }
    async saveWallet(name, mnemonic) {
        // Implement secure storage (encrypted)
        // This could use browser's IndexedDB, secure enclave, etc.
        const encrypted = await this.encrypt(mnemonic);
        if (typeof localStorage !== 'undefined') {
            localStorage.setItem(`wallet_${name}`, encrypted);
        }
    }
    async loadWallet(name) {
        if (typeof localStorage === 'undefined')
            return null;
        const encrypted = localStorage.getItem(`wallet_${name}`);
        if (!encrypted)
            return null;
        return await this.decrypt(encrypted);
    }
    async encrypt(data) {
        // Implement proper encryption
        // This is a placeholder - use proper crypto library
        return btoa(data);
    }
    async decrypt(data) {
        // Implement proper decryption
        // This is a placeholder - use proper crypto library
        return atob(data);
    }
}
//# sourceMappingURL=xion-meta-accounts.js.map