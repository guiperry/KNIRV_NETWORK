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
exports.PrivateKeyKeyring = void 0;
var uuid_1 = require("uuid");
var __1 = require("../..");
var PrivateKeyKeyring = /** @class */ (function () {
    function PrivateKeyKeyring(_a) {
        var id = _a.id, publicKey = _a.publicKey, privateKey = _a.privateKey;
        this.type = 'PRIVATE_KEY';
        if (!publicKey || !privateKey) {
            throw new Error('Invalid parameter values');
        }
        this.id = id || (0, uuid_1.v4)();
        this.publicKey = Uint8Array.from(publicKey);
        this.privateKey = Uint8Array.from(privateKey);
    }
    PrivateKeyKeyring.prototype.toData = function () {
        return {
            id: this.id,
            type: this.type,
            publicKey: Array.from(this.publicKey),
            privateKey: Array.from(this.privateKey),
        };
    };
    PrivateKeyKeyring.prototype.sign = function (provider, document) {
        return __awaiter(this, void 0, void 0, function () {
            var wallet;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, (0, __1.useTm2Wallet)(document).fromPrivateKey(this.privateKey)];
                    case 1:
                        wallet = _a.sent();
                        wallet.connect(provider);
                        return [2 /*return*/, this.signByWallet(wallet, document)];
                }
            });
        });
    };
    PrivateKeyKeyring.prototype.signByWallet = function (wallet, document) {
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
    PrivateKeyKeyring.prototype.broadcastTxSync = function (provider, signedTx) {
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
    PrivateKeyKeyring.prototype.broadcastTxCommit = function (provider, signedTx) {
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
    PrivateKeyKeyring.fromPrivateKeyStr = function (privateKeyStr) {
        return __awaiter(this, void 0, void 0, function () {
            var adjustPrivateKeyStr, privateKey, wallet, publicKey;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        adjustPrivateKeyStr = privateKeyStr.replace('0x', '');
                        privateKey = Uint8Array.from(Buffer.from(adjustPrivateKeyStr, 'hex'));
                        return [4 /*yield*/, (0, __1.useTm2Wallet)({}).fromPrivateKey(privateKey)];
                    case 1:
                        wallet = _a.sent();
                        return [4 /*yield*/, wallet.getPublicKey()];
                    case 2:
                        publicKey = _a.sent();
                        return [2 /*return*/, new PrivateKeyKeyring({
                                publicKey: Array.from(publicKey),
                                privateKey: Array.from(privateKey),
                            })];
                }
            });
        });
    };
    return PrivateKeyKeyring;
}());
exports.PrivateKeyKeyring = PrivateKeyKeyring;
//# sourceMappingURL=private-key-keyring.js.map