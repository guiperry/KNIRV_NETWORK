"use strict";
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
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
var __read = (this && this.__read) || function (o, n) {
    var m = typeof Symbol === "function" && o[Symbol.iterator];
    if (!m) return o;
    var i = m.call(o), r, ar = [], e;
    try {
        while ((n === void 0 || n-- > 0) && !(r = i.next()).done) ar.push(r.value);
    }
    catch (error) { e = { error: error }; }
    finally {
        try {
            if (r && !r.done && (m = i["return"])) m.call(i);
        }
        finally { if (e) throw e.error; }
    }
    return ar;
};
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3AuthKeyring = void 0;
var uuid_1 = require("uuid");
var amino_1 = require("@cosmjs/amino");
var encoding_1 = require("@cosmjs/encoding");
var data_1 = require("../../utils/data");
function convertToStdSignDoc(doc) {
    return {
        chain_id: doc.chain_id,
        account_number: doc.account_number.toString(),
        sequence: doc.sequence.toString(),
        fee: {
            amount: __spreadArray([], __read(doc.fee.amount.map(function (coin) { return ({
                denom: coin.denom,
                amount: coin.amount.toString()
            }); })), false),
            gas: doc.fee.gas.toString()
        },
        msgs: __spreadArray([], __read(doc.msgs), false),
        memo: doc.memo || ''
    };
}
function convertToTx(privateKey, signed, signature) {
    return __awaiter(this, void 0, void 0, function () {
        var wallet, accounts, signer_info, messages;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0: return [4 /*yield*/, amino_1.Secp256k1Wallet.fromKey(privateKey)];
                case 1:
                    wallet = _a.sent();
                    return [4 /*yield*/, wallet.getAccounts()];
                case 2:
                    accounts = _a.sent();
                    signer_info = {
                        public_key: {
                            key: (0, data_1.arrayToHex)(accounts[0].pubkey)
                        },
                        mode_info: { single: { mode: 1 } },
                        sequence: signed.sequence // Already a string from StdSignDoc
                    };
                    messages = signed.msgs.map(function (msg) { return ({
                        type_url: msg.type,
                        value: msg.value
                    }); });
                    return [2 /*return*/, {
                            body: {
                                messages: messages,
                                memo: signed.memo,
                                timeout_height: '0',
                                extension_options: [],
                                non_critical_extension_options: []
                            },
                            auth_info: {
                                signer_infos: [signer_info],
                                fee: {
                                    amount: __spreadArray([], __read(signed.fee.amount), false),
                                    gas: signed.fee.gas,
                                    granter: '',
                                    payer: ''
                                }
                            },
                            signatures: [(0, data_1.arrayToHex)(signature)]
                        }];
            }
        });
    });
}
var Web3AuthKeyring = /** @class */ (function () {
    function Web3AuthKeyring(_a) {
        var id = _a.id, publicKey = _a.publicKey, privateKey = _a.privateKey;
        this.type = 'WEB3_AUTH';
        if (!publicKey || !privateKey) {
            throw new Error('Invalid parameter values');
        }
        this.id = id || (0, uuid_1.v4)();
        this.publicKey = Uint8Array.from(publicKey);
        this.privateKey = Uint8Array.from(privateKey);
    }
    Web3AuthKeyring.prototype.toData = function () {
        return {
            id: this.id,
            type: this.type,
            publicKey: Array.from(this.publicKey),
            privateKey: Array.from(this.privateKey),
        };
    };
    Web3AuthKeyring.prototype.sign = function (provider, document, hdPath) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                throw new Error('This stale CJS Web3Auth build cannot sign Amino; use the SIGN_MODE_DIRECT package build');
            });
        });
    };
    Web3AuthKeyring.prototype.broadcastTxSync = function (provider, signedTx) {
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
    Web3AuthKeyring.prototype.broadcastTxCommit = function (provider, signedTx) {
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
    Web3AuthKeyring.fromPrivateKey = function (privateKey) {
        return __awaiter(this, void 0, void 0, function () {
            var wallet, accounts, publicKey;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, amino_1.Secp256k1Wallet.fromKey(privateKey)];
                    case 1:
                        wallet = _a.sent();
                        return [4 /*yield*/, wallet.getAccounts()];
                    case 2:
                        accounts = _a.sent();
                        publicKey = accounts[0].pubkey;
                        return [2 /*return*/, new Web3AuthKeyring({
                                publicKey: Array.from(publicKey),
                                privateKey: Array.from(privateKey)
                            })];
                }
            });
        });
    };
    Web3AuthKeyring.fromPrivateKeyStr = function (privateKeyStr) {
        return __awaiter(this, void 0, void 0, function () {
            var privateKey, wallet, accounts, publicKey;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        privateKey = (0, data_1.hexToArray)(privateKeyStr);
                        return [4 /*yield*/, amino_1.Secp256k1Wallet.fromKey(privateKey)];
                    case 1:
                        wallet = _a.sent();
                        return [4 /*yield*/, wallet.getAccounts()];
                    case 2:
                        accounts = _a.sent();
                        publicKey = accounts[0].pubkey;
                        return [2 /*return*/, new Web3AuthKeyring({
                                publicKey: Array.from(publicKey),
                                privateKey: Array.from(privateKey)
                            })];
                }
            });
        });
    };
    return Web3AuthKeyring;
}());
exports.Web3AuthKeyring = Web3AuthKeyring;
//# sourceMappingURL=web3-auth-keyring.js.map
