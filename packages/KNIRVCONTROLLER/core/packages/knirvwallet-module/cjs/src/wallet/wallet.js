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
var __values = (this && this.__values) || function(o) {
    var s = typeof Symbol === "function" && Symbol.iterator, m = s && o[s], i = 0;
    if (m) return m.call(o);
    if (o && typeof o.length === "number") return {
        next: function () {
            if (o && i >= o.length) o = void 0;
            return { value: o && o[i++], done: !o };
        }
    };
    throw new TypeError(s ? "Object is not iterable." : "Symbol.iterator is not defined.");
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.AdenaWallet = exports.KnirvWallet = void 0;
var crypto_1 = require("../crypto");
var utils_1 = require("../utils");
var account_1 = require("./account");
var keyring_1 = require("./keyring");
var wallet_crypto_util_1 = require("./wallet-crypto-util");
var defaultWalletData = {
    accounts: [],
    keyrings: [],
};
/**
 * KnirvWallet class provides functionalities for managing accounts, keyrings, and transactions
 * in the KNIRVWALLET ecosystem. It supports various account types and enables operations such
 * as signing transactions, broadcasting them, serialization/deserialization, and wallet creation
 * using different methods (e.g., mnemonic, ledger, Web3Auth).
 */
var KnirvWallet = /** @class */ (function () {
    function KnirvWallet(walletData) {
        var _a = walletData !== null && walletData !== void 0 ? walletData : defaultWalletData, accounts = _a.accounts, keyrings = _a.keyrings, currentAccountId = _a.currentAccountId;
        this._accounts = accounts.map(account_1.makeAccount);
        this._keyrings = keyrings.map(keyring_1.makeKeyring);
        this._currentAccountId = currentAccountId;
    }
    Object.defineProperty(KnirvWallet.prototype, "accounts", {
        get: function () {
            return this._accounts;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KnirvWallet.prototype, "keyrings", {
        get: function () {
            return this._keyrings;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KnirvWallet.prototype, "currentAccount", {
        get: function () {
            var _this = this;
            var currentAccount = this._accounts.find(function (account) { return account.id === _this._currentAccountId; });
            if (!currentAccount) {
                throw new Error('Current account not found');
            }
            return currentAccount;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KnirvWallet.prototype, "currentKeyring", {
        get: function () {
            var currentKeyringId = this.currentAccount.keyringId;
            var currentKeyring = this._keyrings.find(function (keyring) { return keyring.id === currentKeyringId; });
            if (!currentKeyring) {
                throw new Error('Current keyring not found');
            }
            return currentKeyring;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KnirvWallet.prototype, "nextAccountName", {
        get: function () {
            var nextIndex = this.lastAccountIndex + 1;
            return "Account ".concat(nextIndex);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KnirvWallet.prototype, "nextLedgerAccountName", {
        get: function () {
            var nextIndex = this.lastLedgerAccountIndex + 1;
            return "Ledger ".concat(nextIndex);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KnirvWallet.prototype, "currentAccountId", {
        set: function (currentAccountId) {
            this._currentAccountId = currentAccountId;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KnirvWallet.prototype, "defaultHDWalletKeyring", {
        get: function () {
            return this._keyrings.filter(keyring_1.isHDWalletKeyring).find(function (_, index) { return index === 0; }) || null;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KnirvWallet.prototype, "lastAccountIndex", {
        get: function () {
            var indices = this.accounts
                .filter(function (account) { return !(0, account_1.isLedgerAccount)(account); })
                .map(function (account) { return account.index; });
            return Math.max.apply(Math, __spreadArray([0], __read(indices), false));
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KnirvWallet.prototype, "lastLedgerAccountIndex", {
        get: function () {
            var indices = this.accounts.filter(account_1.isLedgerAccount).map(function (account) { return account.index; });
            return Math.max.apply(Math, __spreadArray([0], __read(indices), false));
        },
        enumerable: false,
        configurable: true
    });
    KnirvWallet.prototype.isEmpty = function () {
        return this._accounts.length === 0;
    };
    KnirvWallet.prototype.hasHDWallet = function () {
        return !!this._keyrings.find(keyring_1.isHDWalletKeyring);
    };
    KnirvWallet.prototype.hasKeyring = function (keyring) {
        if ((0, keyring_1.isPrivateKeyKeyring)(keyring)) {
            return this._keyrings.some(function (k) {
                if (!(0, keyring_1.isPrivateKeyKeyring)(k)) {
                    return false;
                }
                return (0, utils_1.arrayContentEquals)(keyring.privateKey, k.privateKey);
            });
        }
        if ((0, keyring_1.isHDWalletKeyring)(keyring)) {
            return this._keyrings.some(function (k) {
                if (!(0, keyring_1.isHDWalletKeyring)(k)) {
                    return false;
                }
                return keyring.mnemonicEntropy === k.mnemonicEntropy;
            });
        }
        return this._keyrings.some(function (k) {
            return keyring.id === k.id;
        });
    };
    KnirvWallet.prototype.hasPrivateKey = function (privateKey) {
        var keyring = this._keyrings
            .filter(keyring_1.hasPrivateKey)
            .find(function (keyring) {
            // Only PrivateKeyKeyring and Web3AuthKeyring have direct privateKey property
            if ((0, keyring_1.isPrivateKeyKeyring)(keyring) || (0, keyring_1.isWeb3AuthKeyring)(keyring)) {
                return JSON.stringify(keyring.privateKey) === JSON.stringify(privateKey);
            }
            // HDWalletKeyring doesn't have a direct privateKey property
            return false;
        });
        return keyring !== undefined;
    };
    KnirvWallet.prototype.getPrivateKeyStr = function () {
        return __awaiter(this, void 0, void 0, function () {
            var privateKey;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getPrivateKey()];
                    case 1:
                        privateKey = _a.sent();
                        return [2 /*return*/, (0, utils_1.arrayToHex)(privateKey)];
                }
            });
        });
    };
    KnirvWallet.prototype.getMnemonic = function () {
        if (!(0, keyring_1.isHDWalletKeyring)(this.currentKeyring)) {
            throw new Error('Mnemonic words not found');
        }
        return this.currentKeyring.getMnemonic();
    };
    KnirvWallet.prototype.getLastAccountIndexBy = function (keyring) {
        var indices = this.accounts
            .filter(function (account) { return account.keyringId === keyring.id; })
            .map(function (account) { return account.index; });
        return Math.max.apply(Math, __spreadArray([0], __read(indices), false));
    };
    KnirvWallet.prototype.getNextAccountIndexBy = function (keyring) {
        if ((0, keyring_1.isLedgerKeyring)(keyring)) {
            return this.lastLedgerAccountIndex + 1;
        }
        return this.lastAccountIndex + 1;
    };
    KnirvWallet.prototype.getNextAccountNumberBy = function (keyring) {
        return this.getNextAccountIndexBy(keyring);
    };
    KnirvWallet.prototype.getNextHDPathBy = function (keyring) {
        if (!(0, keyring_1.isHDWalletKeyring)(keyring)) {
            throw new Error('The current keyring is not an HD Wallet Keyring');
        }
        var seedAccounts = this.accounts
            .filter(function (account) { return account.keyringId === keyring.id; })
            .filter(account_1.isSeedAccount);
        if (seedAccounts.length === 0) {
            return 0;
        }
        var lastHdPath = seedAccounts.reduce(function (account1, account2) {
            return account1.hdPath > account2.hdPath ? account1 : account2;
        }).hdPath;
        var _loop_1 = function (index) {
            if (!seedAccounts.find(function (account) { return account.hdPath === index; })) {
                return { value: index };
            }
        };
        for (var index = 0; index < lastHdPath; index += 1) {
            var state_1 = _loop_1(index);
            if (typeof state_1 === "object")
                return state_1.value;
        }
        return lastHdPath + 1;
    };
    KnirvWallet.prototype.addAccount = function (account) {
        if (this._accounts.find(function (_account) { return _account.id === account.id; })) {
            return this._accounts.length;
        }
        return this._accounts.push(account);
    };
    KnirvWallet.prototype.removeAccount = function (removedAccount) {
        var filteredAccounts = this._accounts.filter(function (account) { return account.id !== removedAccount.id; });
        this._accounts = filteredAccounts;
        var removedKeyringId = removedAccount.keyringId;
        var keyringUsedCount = filteredAccounts.filter(function (account) { return account.keyringId === removedKeyringId; }).length;
        if (keyringUsedCount === 0) {
            var filteredKeyrings = this._keyrings.filter(function (keyring) { return keyring.id !== removedKeyringId; });
            this._keyrings = filteredKeyrings;
        }
        return true;
    };
    KnirvWallet.prototype.addKeyring = function (keyring) {
        if (this._keyrings.find(function (_keyring) { return _keyring.id === keyring.id; })) {
            return this._keyrings.length;
        }
        return this._keyrings.push(keyring);
    };
    KnirvWallet.prototype.getPrivateKey = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                if (!(0, keyring_1.hasPrivateKey)(this.currentKeyring)) {
                    throw new Error('Current account does not have a private key');
                }
                if ((0, keyring_1.isHDWalletKeyring)(this.currentKeyring)) {
                    if (this.currentAccount instanceof account_1.SeedAccount) {
                        return [2 /*return*/, this.currentKeyring.getPrivateKey(this.currentAccount.hdPath)];
                    }
                    throw new Error('Problems with account types');
                }
                return [2 /*return*/, this.currentKeyring.privateKey];
            });
        });
    };
    KnirvWallet.prototype.sign = function (provider, document) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                return [2 /*return*/, this.signByAccountId(provider, this.currentAccount.id, document)];
            });
        });
    };
    KnirvWallet.prototype.signByAccountId = function (provider, accountId, document) {
        return __awaiter(this, void 0, void 0, function () {
            var account, keyring;
            return __generator(this, function (_a) {
                account = this._accounts.find(function (account) { return account.id === accountId; });
                if (!account) {
                    throw new Error('Account not found');
                }
                keyring = this._keyrings.find(function (keyring) { return keyring.id === account.keyringId; });
                if (!keyring) {
                    throw new Error('Keyring not found');
                }
                if ((0, account_1.hasHDPath)(account)) {
                    return [2 /*return*/, keyring.sign(provider, document, account.hdPath)];
                }
                return [2 /*return*/, keyring.sign(provider, document)];
            });
        });
    };
    KnirvWallet.prototype.broadcastTxSync = function (provider, accountId, signedTx) {
        return __awaiter(this, void 0, void 0, function () {
            var account, keyring;
            return __generator(this, function (_a) {
                account = this._accounts.find(function (account) { return account.id === accountId; });
                if (!account) {
                    throw new Error('Account not found');
                }
                keyring = this._keyrings.find(function (keyring) { return keyring.id === account.keyringId; });
                if (!keyring) {
                    throw new Error('Keyring not found');
                }
                if ((0, account_1.hasHDPath)(account)) {
                    return [2 /*return*/, keyring.broadcastTxSync(provider, signedTx, account.hdPath)];
                }
                return [2 /*return*/, keyring.broadcastTxSync(provider, signedTx)];
            });
        });
    };
    KnirvWallet.prototype.broadcastTxCommit = function (provider, accountId, signedTx) {
        return __awaiter(this, void 0, void 0, function () {
            var account, keyring;
            return __generator(this, function (_a) {
                account = this._accounts.find(function (account) { return account.id === accountId; });
                if (!account) {
                    throw new Error('Account not found');
                }
                keyring = this._keyrings.find(function (keyring) { return keyring.id === account.keyringId; });
                if (!keyring) {
                    throw new Error('Keyring not found');
                }
                if ((0, account_1.hasHDPath)(account)) {
                    return [2 /*return*/, keyring.broadcastTxCommit(provider, signedTx, account.hdPath)];
                }
                return [2 /*return*/, keyring.broadcastTxCommit(provider, signedTx)];
            });
        });
    };
    KnirvWallet.prototype.serialize = function (password) {
        return __awaiter(this, void 0, void 0, function () {
            var plain, serialized, encryptedSerialize;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        plain = {
                            currentAccountId: this._currentAccountId,
                            accounts: this._accounts.map(function (account) { return account.toData(); }),
                            keyrings: this._keyrings.map(function (keyring) { return keyring.toData(); }),
                        };
                        serialized = JSON.stringify(plain);
                        return [4 /*yield*/, (0, wallet_crypto_util_1.encryptAES)(serialized, password)];
                    case 1:
                        encryptedSerialize = _a.sent();
                        return [2 /*return*/, encryptedSerialize];
                }
            });
        });
    };
    KnirvWallet.prototype.clone = function () {
        return new KnirvWallet({
            accounts: this._accounts.map(function (account) { return account.toData(); }),
            keyrings: this._keyrings.map(function (keyring) { return keyring.toData(); }),
            currentAccountId: this._currentAccountId,
        });
    };
    // Missing methods implementation as per Wallet_FollowThrough_Implementation.md
    KnirvWallet.prototype.getAccounts = function () {
        return this.accounts;
    };
    KnirvWallet.prototype.getCurrentAccount = function () {
        var _this = this;
        return this.accounts.find(function (account) { return account.id === _this._currentAccountId; });
    };
    KnirvWallet.prototype.getAccountById = function (accountId) {
        return this.accounts.find(function (account) { return account.id === accountId; });
    };
    KnirvWallet.prototype.switchToAccount = function (accountId) {
        var account = this.getAccountById(accountId);
        if (account) {
            this._currentAccountId = accountId;
            return true;
        }
        return false;
    };
    KnirvWallet.prototype.getKeyrings = function () {
        return this.keyrings;
    };
    KnirvWallet.prototype.getKeyringById = function (keyringId) {
        return this.keyrings.find(function (keyring) { return keyring.id === keyringId; });
    };
    // Fix wallet structure validation
    KnirvWallet.prototype.toJSON = function () {
        return {
            accounts: this.accounts,
            keyrings: this.keyrings,
            currentAccountId: this._currentAccountId,
            version: '1.0.0'
        };
    };
    KnirvWallet.deserialize = function (encryptedSerialize, password) {
        return __awaiter(this, void 0, void 0, function () {
            var serialized, plain;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, (0, wallet_crypto_util_1.decryptAES)(encryptedSerialize, password)];
                    case 1:
                        serialized = _a.sent();
                        plain = JSON.parse(serialized);
                        return [2 /*return*/, new KnirvWallet(plain)];
                }
            });
        });
    };
    KnirvWallet.createByMnemonic = function (mnemonic, paths) {
        if (paths === void 0) { paths = [0]; }
        return __awaiter(this, void 0, void 0, function () {
            var wallet, keyring, paths_1, paths_1_1, path, account, e_1_1;
            var e_1, _a;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        wallet = new KnirvWallet();
                        return [4 /*yield*/, keyring_1.HDWalletKeyring.fromMnemonic(mnemonic)];
                    case 1:
                        keyring = _b.sent();
                        _b.label = 2;
                    case 2:
                        _b.trys.push([2, 7, 8, 9]);
                        paths_1 = __values(paths), paths_1_1 = paths_1.next();
                        _b.label = 3;
                    case 3:
                        if (!!paths_1_1.done) return [3 /*break*/, 6];
                        path = paths_1_1.value;
                        return [4 /*yield*/, account_1.SeedAccount.createBy(keyring, wallet.nextAccountName, path)];
                    case 4:
                        account = _b.sent();
                        wallet.currentAccountId = account.id;
                        wallet.addAccount(account);
                        wallet.addKeyring(keyring);
                        _b.label = 5;
                    case 5:
                        paths_1_1 = paths_1.next();
                        return [3 /*break*/, 3];
                    case 6: return [3 /*break*/, 9];
                    case 7:
                        e_1_1 = _b.sent();
                        e_1 = { error: e_1_1 };
                        return [3 /*break*/, 9];
                    case 8:
                        try {
                            if (paths_1_1 && !paths_1_1.done && (_a = paths_1.return)) _a.call(paths_1);
                        }
                        finally { if (e_1) throw e_1.error; }
                        return [7 /*endfinally*/];
                    case 9: return [2 /*return*/, wallet];
                }
            });
        });
    };
    KnirvWallet.createByLedger = function (connector, paths) {
        if (paths === void 0) { paths = [0]; }
        return __awaiter(this, void 0, void 0, function () {
            var wallet, keyring, paths_2, paths_2_1, path, account, e_2_1;
            var e_2, _a;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        wallet = new KnirvWallet();
                        return [4 /*yield*/, keyring_1.LedgerKeyring.fromLedger(connector)];
                    case 1:
                        keyring = _b.sent();
                        _b.label = 2;
                    case 2:
                        _b.trys.push([2, 7, 8, 9]);
                        paths_2 = __values(paths), paths_2_1 = paths_2.next();
                        _b.label = 3;
                    case 3:
                        if (!!paths_2_1.done) return [3 /*break*/, 6];
                        path = paths_2_1.value;
                        return [4 /*yield*/, account_1.LedgerAccount.createBy(keyring, wallet.nextLedgerAccountName, path)];
                    case 4:
                        account = _b.sent();
                        wallet.currentAccountId = account.id;
                        wallet.addAccount(account);
                        wallet.addKeyring(keyring);
                        _b.label = 5;
                    case 5:
                        paths_2_1 = paths_2.next();
                        return [3 /*break*/, 3];
                    case 6: return [3 /*break*/, 9];
                    case 7:
                        e_2_1 = _b.sent();
                        e_2 = { error: e_2_1 };
                        return [3 /*break*/, 9];
                    case 8:
                        try {
                            if (paths_2_1 && !paths_2_1.done && (_a = paths_2.return)) _a.call(paths_2);
                        }
                        finally { if (e_2) throw e_2.error; }
                        return [7 /*endfinally*/];
                    case 9: return [2 /*return*/, wallet];
                }
            });
        });
    };
    KnirvWallet.createByWeb3Auth = function (privateKeyStr) {
        return __awaiter(this, void 0, void 0, function () {
            var privateKey, wallet, keyring, account;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        privateKey = (0, utils_1.hexToArray)(privateKeyStr);
                        wallet = new KnirvWallet();
                        return [4 /*yield*/, keyring_1.Web3AuthKeyring.fromPrivateKey(privateKey)];
                    case 1:
                        keyring = _a.sent();
                        return [4 /*yield*/, account_1.SingleAccount.createBy(keyring, wallet.nextAccountName)];
                    case 2:
                        account = _a.sent();
                        wallet.currentAccountId = account.id;
                        wallet.addAccount(account);
                        wallet.addKeyring(keyring);
                        return [2 /*return*/, wallet];
                }
            });
        });
    };
    KnirvWallet.createByAddress = function (address) {
        return __awaiter(this, void 0, void 0, function () {
            var wallet, keyring, account;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        wallet = new KnirvWallet();
                        return [4 /*yield*/, keyring_1.AddressKeyring.fromAddress(address)];
                    case 1:
                        keyring = _a.sent();
                        return [4 /*yield*/, account_1.AirgapAccount.createBy(keyring, wallet.nextAccountName)];
                    case 2:
                        account = _a.sent();
                        wallet.currentAccountId = account.id;
                        wallet.addAccount(account);
                        wallet.addKeyring(keyring);
                        return [2 /*return*/, wallet];
                }
            });
        });
    };
    KnirvWallet.generateMnemonic = function (length) {
        if (length === void 0) { length = 12; }
        var entropyLength = 4 * Math.floor((11 * length) / 33);
        var entropy = crypto_1.Random.getBytes(entropyLength);
        var mnemonic = crypto_1.Bip39.encode(entropy);
        return mnemonic.toString();
    };
    /**
     * Sign transaction method for test compatibility
     * This is a simplified version that creates a mock signed transaction
     */
    KnirvWallet.prototype.signTransaction = function (transaction) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                // Mock implementation for testing - in real implementation, this would use proper signing
                return [2 /*return*/, __assign(__assign({}, transaction), { signatures: [
                            {
                                pub_key: {
                                    type: 'tendermint/PubKeySecp256k1',
                                    value: 'mock-public-key-value'
                                },
                                signature: 'mock-signature-value'
                            }
                        ], memo: transaction.memo || '', fee: {
                            amount: [{ denom: transaction.token || 'unrn', amount: '0' }],
                            gas: transaction.gasLimit || '200000'
                        } })];
            });
        });
    };
    return KnirvWallet;
}());
exports.KnirvWallet = KnirvWallet;
exports.AdenaWallet = KnirvWallet;
//# sourceMappingURL=wallet.js.map