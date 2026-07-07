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
exports.SeedAccount = void 0;
var uuid_1 = require("uuid");
var address_1 = require("../../utils/address");
var keyring_1 = require("../keyring");
var SeedAccount = /** @class */ (function () {
    function SeedAccount(_a) {
        var id = _a.id, index = _a.index, keyringId = _a.keyringId, publicKey = _a.publicKey, type = _a.type, name = _a.name, hdPath = _a.hdPath;
        this.id = id !== null && id !== void 0 ? id : (0, uuid_1.v4)();
        this._index = index;
        this.type = type;
        this.keyringId = keyringId;
        this.hdPath = hdPath !== null && hdPath !== void 0 ? hdPath : 0;
        this.publicKey = Uint8Array.from(publicKey);
        this._name = name;
    }
    Object.defineProperty(SeedAccount.prototype, "index", {
        get: function () {
            return this._index;
        },
        set: function (_index) {
            this._index = _index;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(SeedAccount.prototype, "name", {
        get: function () {
            return this._name;
        },
        set: function (_name) {
            this._name = _name;
        },
        enumerable: false,
        configurable: true
    });
    SeedAccount.prototype.getAddress = function (prefix) {
        return (0, address_1.publicKeyToAddress)(this.publicKey, prefix);
    };
    SeedAccount.prototype.toData = function () {
        return {
            id: this.id,
            index: this._index,
            type: this.type,
            keyringId: this.keyringId,
            hdPath: this.hdPath,
            publicKey: Array.from(this.publicKey),
            name: this._name,
        };
    };
    SeedAccount.createBy = function (keyring, name, hdPath, index) {
        if (index === void 0) { index = 1; }
        return __awaiter(this, void 0, void 0, function () {
            var publicKey, keyringId, type;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!(0, keyring_1.isHDWalletKeyring)(keyring)) {
                            throw new Error('Invalid account type');
                        }
                        return [4 /*yield*/, keyring.getPublicKey(hdPath)];
                    case 1:
                        publicKey = _a.sent();
                        keyringId = keyring.id, type = keyring.type;
                        return [2 /*return*/, new SeedAccount({
                                keyringId: keyringId,
                                index: index,
                                type: type,
                                publicKey: Array.from(publicKey),
                                name: name,
                                hdPath: hdPath,
                            })];
                }
            });
        });
    };
    SeedAccount.createByWallet = function (wallet) {
        return __awaiter(this, void 0, void 0, function () {
            var hdPath;
            return __generator(this, function (_a) {
                if (!wallet.currentKeyring || !(0, keyring_1.isHDWalletKeyring)(wallet.currentKeyring)) {
                    throw new Error('The current keyring is not an HD Wallet Keyring');
                }
                hdPath = wallet.getNextHDPathBy(wallet.currentKeyring);
                return [2 /*return*/, this.createBy(wallet.currentKeyring, wallet.nextAccountName, hdPath)];
            });
        });
    };
    SeedAccount.fromData = function (accountInfo) {
        return new SeedAccount({
            id: accountInfo.id,
            index: accountInfo.index,
            type: accountInfo.type,
            keyringId: accountInfo.keyringId,
            hdPath: accountInfo.hdPath,
            publicKey: accountInfo.publicKey,
            name: accountInfo.name,
        });
    };
    return SeedAccount;
}());
exports.SeedAccount = SeedAccount;
//# sourceMappingURL=seed-account.js.map