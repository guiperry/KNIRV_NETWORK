"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.makeAccount = void 0;
var airgap_account_1 = require("./airgap-account");
var ledger_account_1 = require("./ledger-account");
var seed_account_1 = require("./seed-account");
var single_account_1 = require("./single-account");
function makeAccount(accountData) {
    switch (accountData.type) {
        case 'HD':
            return new seed_account_1.SeedAccount(accountData);
        case 'LEDGER':
            return new ledger_account_1.LedgerAccount(accountData);
        case 'PRIVATE_KEY':
        case 'WEB3_AUTH':
            return new single_account_1.SingleAccount(accountData);
        case 'ADDRESS':
            return new airgap_account_1.AirgapAccount(accountData);
        default:
            throw new Error('Invalid account type');
    }
}
exports.makeAccount = makeAccount;
//# sourceMappingURL=account.js.map