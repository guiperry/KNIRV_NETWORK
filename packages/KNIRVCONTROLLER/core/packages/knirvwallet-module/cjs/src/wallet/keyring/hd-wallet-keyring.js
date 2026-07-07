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
exports.HDWalletKeyring = void 0;
// Utility function to replace generateKeyPair
function generateKeyPair(mnemonic, hdPath) {
    return __awaiter(this, void 0, void 0, function () {
        return __generator(this, function (_a) {
            // This is a placeholder implementation
            // In a real implementation, this would derive keys from the mnemonic using BIP44
            return [2 /*return*/, {
                    privateKey: new Uint8Array(32),
                    publicKey: new Uint8Array(33),
                }];
        });
    });
}
var uuid_1 = require("uuid");
var crypto_1 = require("../../crypto");
var __1 = require("../..");
var HDWalletKeyring = /** @class */ (function () {
    function HDWalletKeyring(_a) {
        var id = _a.id, mnemonicEntropy = _a.mnemonicEntropy, seed = _a.seed;
        this.type = 'HD';
        if (!mnemonicEntropy || !seed) {
            throw new Error('Invalid parameter values');
        }
        this.id = id || (0, uuid_1.v4)();
        this.mnemonicEntropy = Uint8Array.from(mnemonicEntropy);
        this.seed = Uint8Array.from(seed);
    }
    HDWalletKeyring.prototype.getMnemonic = function () {
        return (0, crypto_1.entropyToMnemonic)(this.mnemonicEntropy);
    };
    HDWalletKeyring.prototype.getKeypair = function (hdPath) {
        return __awaiter(this, void 0, void 0, function () {
            var _a, privateKey, publicKey;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0: return [4 /*yield*/, generateKeyPair(this.getMnemonic(), hdPath)];
                    case 1:
                        _a = _b.sent(), privateKey = _a.privateKey, publicKey = _a.publicKey;
                        return [2 /*return*/, { privateKey: privateKey, publicKey: publicKey }];
                }
            });
        });
    };
    HDWalletKeyring.prototype.getPrivateKey = function (hdPath) {
        return __awaiter(this, void 0, void 0, function () {
            var privateKey;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getKeypair(hdPath)];
                    case 1:
                        privateKey = (_a.sent()).privateKey;
                        return [2 /*return*/, privateKey];
                }
            });
        });
    };
    HDWalletKeyring.prototype.getPublicKey = function (hdPath) {
        return __awaiter(this, void 0, void 0, function () {
            var publicKey;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getKeypair(hdPath)];
                    case 1:
                        publicKey = (_a.sent()).publicKey;
                        return [2 /*return*/, publicKey];
                }
            });
        });
    };
    HDWalletKeyring.prototype.toData = function () {
        return {
            id: this.id,
            type: this.type,
            seed: Array.from(this.seed),
            mnemonicEntropy: Array.from(this.mnemonicEntropy),
        };
    };
    HDWalletKeyring.prototype.sign = function (provider, document, hdPath) {
        if (hdPath === void 0) { hdPath = 0; }
        return __awaiter(this, void 0, void 0, function () {
            var wallet;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, (0, __1.useTm2Wallet)(document).fromMnemonic(this.getMnemonic(), {
                            accountIndex: hdPath,
                        })];
                    case 1:
                        wallet = _a.sent();
                        wallet.connect(provider);
                        return [2 /*return*/, this.signByWallet(wallet, document)];
                }
            });
        });
    };
    HDWalletKeyring.prototype.signByWallet = function (wallet, document) {
        return __awaiter(this, void 0, void 0, function () {
            var signedTx, signatures;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, (0, __1.makeSignedTx)(wallet, document)];
                    case 1:
                        signedTx = _a.sent();
                        signatures = (signedTx.signatures || []).length > 0
                            ? signedTx.signatures.map(function (sig) { return ({
                                pub_key: {
                                    key: '',
                                },
                                signature: sig,
                            }); })
                            : [
                                {
                                    pub_key: {
                                        key: '',
                                    },
                                    signature: '',
                                },
                            ];
                        return [2 /*return*/, {
                                signed: signedTx,
                                signature: signatures,
                            }];
                }
            });
        });
    };
    HDWalletKeyring.prototype.broadcastTxSync = function (provider, signedTx, hdPath) {
        if (hdPath === void 0) { hdPath = 0; }
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                // For KNIRV, we'll use the transaction SDK to submit transactions
                // This is a placeholder implementation
                return [2 /*return*/, {
                        hash: 'placeholder-hash',
                        code: 0,
                        log: 'Transaction broadcasting not implemented yet - use KNIRV transaction SDK',
                    }];
            });
        });
    };
    HDWalletKeyring.prototype.broadcastTxCommit = function (provider, signedTx, hdPath) {
        if (hdPath === void 0) { hdPath = 0; }
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                // For KNIRV, we'll use the transaction SDK to submit transactions
                // This is a placeholder implementation
                return [2 /*return*/, {
                        hash: 'placeholder-hash',
                        height: 0,
                        code: 0,
                        log: 'Transaction broadcasting not implemented yet - use KNIRV transaction SDK',
                        gasUsed: 0,
                        gasWanted: 0,
                    }];
            });
        });
    };
    HDWalletKeyring.fromMnemonic = function (mnemonic) {
        return __awaiter(this, void 0, void 0, function () {
            var englishMnemonic, seed, mnemonicEntropy;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        englishMnemonic = new crypto_1.EnglishMnemonic(mnemonic);
                        return [4 /*yield*/, crypto_1.Bip39.mnemonicToSeed(englishMnemonic)];
                    case 1:
                        seed = _a.sent();
                        return [4 /*yield*/, (0, crypto_1.mnemonicToEntropy)(englishMnemonic.toString())];
                    case 2:
                        mnemonicEntropy = _a.sent();
                        return [2 /*return*/, new HDWalletKeyring({
                                mnemonicEntropy: Array.from(mnemonicEntropy),
                                seed: Array.from(seed),
                            })];
                }
            });
        });
    };
    return HDWalletKeyring;
}());
exports.HDWalletKeyring = HDWalletKeyring;
//# sourceMappingURL=hd-wallet-keyring.js.map