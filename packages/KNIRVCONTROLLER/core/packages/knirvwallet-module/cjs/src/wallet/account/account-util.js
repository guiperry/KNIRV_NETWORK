"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deserializeAccount = exports.serializeAccount = exports.hasHDPath = exports.hasPrivateKeyAccount = exports.isAirgapAccount = exports.isSingleAccount = exports.isLedgerAccount = exports.isSeedAccount = void 0;
var ledger_account_1 = require("./ledger-account");
var seed_account_1 = require("./seed-account");
var single_account_1 = require("./single-account");
function isSeedAccount(account) {
    return account.type === 'HD_WALLET';
}
exports.isSeedAccount = isSeedAccount;
function isLedgerAccount(account) {
    return account.type === 'LEDGER';
}
exports.isLedgerAccount = isLedgerAccount;
function isSingleAccount(account) {
    return account.type === 'WEB3_AUTH' || account.type === 'PRIVATE_KEY';
}
exports.isSingleAccount = isSingleAccount;
function isAirgapAccount(account) {
    return account.type === 'AIRGAP';
}
exports.isAirgapAccount = isAirgapAccount;
function hasPrivateKeyAccount(account) {
    return isSeedAccount(account) || isSingleAccount(account);
}
exports.hasPrivateKeyAccount = hasPrivateKeyAccount;
function hasHDPath(account) {
    return isSeedAccount(account) || isLedgerAccount(account);
}
exports.hasHDPath = hasHDPath;
function serializeAccount(account) {
    return JSON.stringify(account.toData());
}
exports.serializeAccount = serializeAccount;
function deserializeAccount(plain) {
    var accountInfo = JSON.parse(plain);
    if (accountInfo.type === 'HD_WALLET') {
        return seed_account_1.SeedAccount.fromData(accountInfo);
    }
    if (accountInfo.type === 'LEDGER') {
        return ledger_account_1.LedgerAccount.fromData(accountInfo);
    }
    if (accountInfo.type === 'PRIVATE_KEY' || accountInfo.type === 'WEB3_AUTH') {
        return single_account_1.SingleAccount.fromData(accountInfo);
    }
    throw new Error('Invalid account type');
}
exports.deserializeAccount = deserializeAccount;
//# sourceMappingURL=account-util.js.map