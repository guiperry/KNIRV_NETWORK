"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.makeSignedTx = exports.useTm2Wallet = exports.hasPrivateKey = exports.isAddressKeyring = exports.isWeb3AuthKeyring = exports.isPrivateKeyKeyring = exports.isLedgerKeyring = exports.isHDWalletKeyring = exports.SimpleKNIRVWallet = void 0;
var messages_1 = require("../../utils/messages");
// Simple KNIRV wallet implementation
var SimpleKNIRVWallet = /** @class */ (function () {
    function SimpleKNIRVWallet(privateKey) {
        this.privateKey = privateKey;
    }
    SimpleKNIRVWallet.prototype.connect = function (provider) {
        this.provider = provider;
    };
    SimpleKNIRVWallet.prototype.signTransaction = function (tx, decodeFn) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                // For now, return the transaction as-is
                // In a real implementation, this would sign the transaction with the private key
                return [2 /*return*/, tx];
            });
        });
    };
    SimpleKNIRVWallet.prototype.getPublicKey = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                // This would derive the public key from the private key
                // For now, return a placeholder
                return [2 /*return*/, new Uint8Array(32)];
            });
        });
    };
    SimpleKNIRVWallet.prototype.getAddress = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                // This would derive the address from the public key
                // For now, return a placeholder
                return [2 /*return*/, 'knirv1placeholder'];
            });
        });
    };
    SimpleKNIRVWallet.fromPrivateKey = function (privateKey) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                return [2 /*return*/, new SimpleKNIRVWallet(privateKey)];
            });
        });
    };
    SimpleKNIRVWallet.fromMnemonic = function (mnemonic, options) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                // This would derive the private key from the mnemonic
                // For now, return a placeholder wallet
                return [2 /*return*/, new SimpleKNIRVWallet(new Uint8Array(32))];
            });
        });
    };
    SimpleKNIRVWallet.fromLedger = function (connector, options) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                // This would connect to a Ledger device
                // For now, return a placeholder wallet
                return [2 /*return*/, new SimpleKNIRVWallet(new Uint8Array(32))];
            });
        });
    };
    return SimpleKNIRVWallet;
}());
exports.SimpleKNIRVWallet = SimpleKNIRVWallet;
function isHDWalletKeyring(keyring) {
    return keyring.type === 'HD';
}
exports.isHDWalletKeyring = isHDWalletKeyring;
function isLedgerKeyring(keyring) {
    return keyring.type === 'LEDGER';
}
exports.isLedgerKeyring = isLedgerKeyring;
function isPrivateKeyKeyring(keyring) {
    return keyring.type === 'PRIVATE_KEY';
}
exports.isPrivateKeyKeyring = isPrivateKeyKeyring;
function isWeb3AuthKeyring(keyring) {
    return keyring.type === 'WEB3_AUTH';
}
exports.isWeb3AuthKeyring = isWeb3AuthKeyring;
function isAddressKeyring(keyring) {
    return keyring.type === 'ADDRESS';
}
exports.isAddressKeyring = isAddressKeyring;
function hasPrivateKey(keyring) {
    if (isHDWalletKeyring(keyring)) {
        return true;
    }
    if (isPrivateKeyKeyring(keyring)) {
        return true;
    }
    if (isWeb3AuthKeyring(keyring)) {
        return true;
    }
    return false;
}
exports.hasPrivateKey = hasPrivateKey;
function useTm2Wallet(document) {
    return SimpleKNIRVWallet;
}
exports.useTm2Wallet = useTm2Wallet;
function makeSignedTx(wallet, document) {
    var tx = (0, messages_1.documentToTx)(document);
    var decodeTxMessageFunction = messages_1.decodeTxMessages;
    return wallet.signTransaction(tx, decodeTxMessageFunction);
}
exports.makeSignedTx = makeSignedTx;
//# sourceMappingURL=keyring-util.js.map