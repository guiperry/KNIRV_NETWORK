export var WalletResponseStatus;
(function (WalletResponseStatus) {
    WalletResponseStatus["SUCCESS"] = "success";
    WalletResponseStatus["FAILURE"] = "failure";
    WalletResponseStatus["REJECT"] = "reject";
})(WalletResponseStatus || (WalletResponseStatus = {}));
export var WalletResponseType;
(function (WalletResponseType) {
    WalletResponseType["ESTABLISH"] = "establish";
    WalletResponseType["ACCOUNT"] = "account";
    WalletResponseType["NETWORK"] = "network";
    WalletResponseType["SIGN"] = "sign";
    WalletResponseType["TRANSACTION"] = "transaction";
})(WalletResponseType || (WalletResponseType = {}));
export var WalletResponseExecuteType;
(function (WalletResponseExecuteType) {
    WalletResponseExecuteType["ADD_ESTABLISH"] = "ADD_ESTABLISH";
    WalletResponseExecuteType["GET_ACCOUNT"] = "GET_ACCOUNT";
    WalletResponseExecuteType["ADD_NETWORK"] = "ADD_NETWORK";
    WalletResponseExecuteType["SWITCH_NETWORK"] = "SWITCH_NETWORK";
    WalletResponseExecuteType["DO_CONTRACT"] = "DO_CONTRACT";
    WalletResponseExecuteType["SIGN_TX"] = "SIGN_TX";
})(WalletResponseExecuteType || (WalletResponseExecuteType = {}));
export var WalletResponseFailureType;
(function (WalletResponseFailureType) {
    WalletResponseFailureType["NETWORK_TIMEOUT"] = "NETWORK_TIMEOUT";
    WalletResponseFailureType["UNAPPROVED_CHAIN"] = "UNAPPROVED_CHAIN";
    WalletResponseFailureType["UNAPPROVED_HOST"] = "UNAPPROVED_HOST";
    WalletResponseFailureType["LOCKED_ACCOUNT"] = "LOCKED_ACCOUNT";
    WalletResponseFailureType["INVALID_FORMAT"] = "INVALID_FORMAT";
    WalletResponseFailureType["INVALID_TRANSACTION"] = "INVALID_TRANSACTION";
    WalletResponseFailureType["UNEXPECTED_ERROR"] = "UNEXPECTED_ERROR";
})(WalletResponseFailureType || (WalletResponseFailureType = {}));
export var WalletResponseRejectType;
(function (WalletResponseRejectType) {
    WalletResponseRejectType["ESTABLISH_REJECTED"] = "ESTABLISH_REJECTED";
    WalletResponseRejectType["SIGN_REJECTED"] = "SIGN_REJECTED";
    WalletResponseRejectType["TRANSACTION_REJECTED"] = "TRANSACTION_REJECTED";
})(WalletResponseRejectType || (WalletResponseRejectType = {}));
export var WalletResponseSuccessType;
(function (WalletResponseSuccessType) {
    WalletResponseSuccessType["ESTABLISH_SUCCESS"] = "ESTABLISH_SUCCESS";
    WalletResponseSuccessType["SIGN_SUCCESS"] = "SIGN_SUCCESS";
    WalletResponseSuccessType["TRANSACTION_SUCCESS"] = "TRANSACTION_SUCCESS";
})(WalletResponseSuccessType || (WalletResponseSuccessType = {}));
const DEFAULT_GATEWAYS = [
    'https://gateway.knirv.network',
    'https://testnet-gateway.knirv.network',
    'http://localhost:8080',
];
function approvalJSON(value) {
    return JSON.stringify(value, (_key, item) => {
        if (typeof item === 'bigint')
            return item.toString();
        if (!(item instanceof Uint8Array))
            return item;
        let binary = '';
        for (let start = 0; start < item.length; start += 0x8000) {
            binary += String.fromCharCode(...item.subarray(start, start + 0x8000));
        }
        return btoa(binary);
    });
}
export class KNIRVWallet {
    constructor(config = {}) {
        this.gateways = config.provider
            ? [config.provider.replace(/\/$/, ''), ...DEFAULT_GATEWAYS.filter((url) => url !== config.provider)]
            : DEFAULT_GATEWAYS;
        this.chainId = config.chainId || 'knirv-1';
        this.apiKey = config.apiKey;
        this.approvalTimeoutMs = config.approvalTimeoutMs || 5 * 60000;
    }
    async request(path, init = {}) {
        let lastError;
        for (const gateway of this.gateways) {
            try {
                const response = await fetch(`${gateway}${path}`, {
                    ...init,
                    headers: {
                        'Content-Type': 'application/json',
                        ...(this.apiKey ? { Authorization: `Bearer ${this.apiKey}` } : {}),
                        ...init.headers,
                    },
                });
                if (response.status >= 500)
                    throw new Error(`${gateway} returned HTTP ${response.status}`);
                if (!response.ok)
                    throw new Error(`HTTP ${response.status}: ${await response.text()}`);
                return await response.json();
            }
            catch (error) {
                lastError = error;
            }
        }
        throw lastError instanceof Error ? lastError : new Error('No KNIRV gateway is available');
    }
    async approvedOperation(path, payload, type) {
        const approval = await this.request(path, { method: 'POST', body: approvalJSON(payload) });
        const deadline = Date.now() + this.approvalTimeoutMs;
        while (Date.now() < deadline) {
            const state = await this.request(`/api/controller/signing/requests/${encodeURIComponent(approval.request_id)}`);
            if (state.status === 'approved')
                return { code: 0, status: WalletResponseStatus.SUCCESS, type, data: state.result };
            if (state.status === 'rejected')
                return { code: 1, status: WalletResponseStatus.REJECT, type, message: state.reason || 'Rejected in KNIRVCONTROLLER' };
            if (state.status === 'expired')
                return { code: 1, status: WalletResponseStatus.FAILURE, type, message: 'KNIRVCONTROLLER approval expired' };
            await new Promise((resolve) => setTimeout(resolve, 1500));
        }
        return { code: 1, status: WalletResponseStatus.FAILURE, type, message: 'KNIRVCONTROLLER approval timed out' };
    }
    async getAccount() {
        const data = await this.request('/api/controller/account');
        return { code: 0, status: WalletResponseStatus.SUCCESS, type: WalletResponseType.ACCOUNT, data };
    }
    async addNetwork(params) {
        return this.approvedOperation('/api/controller/networks', params, WalletResponseType.NETWORK);
    }
    async switchNetwork(chainId) {
        return this.approvedOperation('/api/controller/networks/switch', { chain_id: chainId }, WalletResponseType.NETWORK);
    }
    async signTransaction(transaction) {
        return this.approvedOperation('/api/controller/signing/requests', { kind: 'transaction', chain_id: this.chainId, transaction }, WalletResponseType.SIGN);
    }
    async signMessage(envelope) {
        return this.approvedOperation('/api/controller/signing/requests', { kind: 'message', chain_id: this.chainId, envelope }, WalletResponseType.SIGN);
    }
    async doContract(transaction) {
        return this.approvedOperation('/api/controller/signing/requests', { kind: 'transaction', chain_id: transaction.chainId, transaction }, WalletResponseType.TRANSACTION);
    }
    async invokeSkill(params) {
        const transaction = {
            action: {
                action: 'knirv.skill.invoke',
                sender: params.sender,
                recipient: params.skill_id,
                amount: params.amount,
                payload: new TextEncoder().encode(JSON.stringify({ user_id: params.user_id, metadata: params.metadata ?? {} })),
                timestampUnix: Math.floor(Date.now() / 1000),
            },
            chainId: this.chainId,
            accountNumber: params.account_number,
            sequence: params.sequence,
            fee: params.fee,
        };
        return this.approvedOperation('/api/controller/signing/requests', { kind: 'transaction', chain_id: this.chainId, transaction }, WalletResponseType.TRANSACTION);
    }
    async resolveKnirvURI(uri) {
        const data = await this.request(`/api/transmission/resolve?uri=${encodeURIComponent(uri)}`);
        return { code: 0, status: WalletResponseStatus.SUCCESS, type: WalletResponseType.TRANSACTION, data };
    }
    async broadcastToNetwork(data) {
        const result = await this.request('/api/transmission/broadcast', { method: 'POST', body: JSON.stringify(data) });
        return { code: 0, status: WalletResponseStatus.SUCCESS, type: WalletResponseType.TRANSACTION, data: result };
    }
}
export { KNIRVWallet as AdenaWallet };
export { generateHDPath, TransactionEndpoint, Tx, Wallet as Tm2Wallet, TxSignature, } from '@gnolang/tm2-js-client';
export { LedgerConnector } from '@cosmjs/ledger-amino';
