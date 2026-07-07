"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.makeKeyring = void 0;
var address_keyring_1 = require("./address-keyring");
var hd_wallet_keyring_1 = require("./hd-wallet-keyring");
var ledger_keyring_1 = require("./ledger-keyring");
var private_key_keyring_1 = require("./private-key-keyring");
var web3_auth_keyring_1 = require("./web3-auth-keyring");
function makeKeyring(keyringData) {
    switch (keyringData.type) {
        case 'HD':
            return new hd_wallet_keyring_1.HDWalletKeyring(keyringData);
        case 'LEDGER':
            return new ledger_keyring_1.LedgerKeyring(keyringData);
        case 'PRIVATE_KEY':
            return new private_key_keyring_1.PrivateKeyKeyring(keyringData);
        case 'WEB3_AUTH':
            return new web3_auth_keyring_1.Web3AuthKeyring(keyringData);
        case 'ADDRESS':
            return new address_keyring_1.AddressKeyring(keyringData);
        default:
            throw new Error('Invalid Account type');
    }
}
exports.makeKeyring = makeKeyring;
//# sourceMappingURL=keyring.js.map