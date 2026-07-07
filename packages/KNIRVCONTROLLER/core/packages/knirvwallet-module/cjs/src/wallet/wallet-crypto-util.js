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
exports.decryptAES = exports.encryptAES = exports.encryptSha256 = exports.makeCryptKey = exports.executeKdf = void 0;
var crypto_js_1 = require("crypto-js");
var crypto_1 = require("../crypto");
var encoding_1 = require("../encoding");
function executeKdf(salt, password, configuration) {
    return __awaiter(this, void 0, void 0, function () {
        var hexKey, bytes, i;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    // Validate configuration
                    if (!configuration || !configuration.algorithm) {
                        throw new Error('Invalid KDF configuration: missing algorithm');
                    }
                    // Only support argon2id for backward compatibility, but use KNIRV crypto implementation
                    if (configuration.algorithm !== 'argon2id') {
                        throw new Error("Unsupported KDF algorithm: ".concat(configuration.algorithm));
                    }
                    return [4 /*yield*/, (0, exports.makeCryptKey)(password)];
                case 1:
                    hexKey = _a.sent();
                    bytes = new Uint8Array(hexKey.length / 2);
                    for (i = 0; i < hexKey.length; i += 2) {
                        bytes[i / 2] = parseInt(hexKey.substr(i, 2), 16);
                    }
                    return [2 /*return*/, bytes];
            }
        });
    });
}
exports.executeKdf = executeKdf;
var makeCryptKey = function (password) { return __awaiter(void 0, void 0, void 0, function () {
    var SALT_KEY;
    var _a;
    return __generator(this, function (_b) {
        SALT_KEY = (_a = process.env.SALT_KEY) !== null && _a !== void 0 ? _a : 'knirv-default-salt';
        return [2 /*return*/, crypto_js_1.default.PBKDF2(password, SALT_KEY, { keySize: 256 / 32, iterations: 1000 }).toString()];
    });
}); };
exports.makeCryptKey = makeCryptKey;
var encryptSha256 = function (password) { return __awaiter(void 0, void 0, void 0, function () {
    var cryptKey, hash;
    return __generator(this, function (_a) {
        switch (_a.label) {
            case 0: return [4 /*yield*/, (0, exports.makeCryptKey)(password)];
            case 1:
                cryptKey = _a.sent();
                hash = (0, crypto_1.sha256)(new TextEncoder().encode(cryptKey));
                return [2 /*return*/, (0, encoding_1.toHex)(hash)];
        }
    });
}); };
exports.encryptSha256 = encryptSha256;
var encryptAES = function (value, password) { return __awaiter(void 0, void 0, void 0, function () {
    return __generator(this, function (_a) {
        return [2 /*return*/, crypto_js_1.default.AES.encrypt(value, password).toString()];
    });
}); };
exports.encryptAES = encryptAES;
var decryptAES = function (encryptedValue, password) { return __awaiter(void 0, void 0, void 0, function () {
    var bytes;
    return __generator(this, function (_a) {
        bytes = crypto_js_1.default.AES.decrypt(encryptedValue, password);
        return [2 /*return*/, bytes.toString(crypto_js_1.default.enc.Utf8)];
    });
}); };
exports.decryptAES = decryptAES;
//# sourceMappingURL=wallet-crypto-util.js.map