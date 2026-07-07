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
exports.pbkdf2Sha512 = exports.pbkdf2Sha512Noble = exports.pbkdf2Sha512Crypto = exports.pbkdf2Sha512Subtle = exports.getSubtle = exports.getCryptoModule = void 0;
var utils_1 = require("../utils");
var pbkdf2_1 = require("@noble/hashes/pbkdf2");
var sha512_1 = require("@noble/hashes/sha512");
var crypto_1 = require("crypto");
/**
 * Returns the Node.js crypto module when available and `undefined`
 * otherwise.
 *
 * Detects an unimplemented fallback module from Webpack 5 and returns
 * `undefined` in that case.
 */
function getCryptoModule() {
    return __awaiter(this, void 0, void 0, function () {
        return __generator(this, function (_a) {
            try {
                // We get `Object{default: Object{}}` as a fallback when using
                // `crypto: false` in Webpack 5, which we interprete as unavailable.
                if (typeof crypto_1.default === 'object' && Object.keys(crypto_1.default).length <= 1) {
                    return [2 /*return*/, undefined];
                }
                return [2 /*return*/, crypto_1.default];
            }
            catch (_b) {
                return [2 /*return*/, undefined];
            }
            return [2 /*return*/];
        });
    });
}
exports.getCryptoModule = getCryptoModule;
function getSubtle() {
    return __awaiter(this, void 0, void 0, function () {
        var g, subtle, crypto_2;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    g = globalThis;
                    subtle = g.crypto && g.crypto.subtle;
                    if (!!subtle) return [3 /*break*/, 2];
                    return [4 /*yield*/, getCryptoModule()];
                case 1:
                    crypto_2 = _a.sent();
                    if (crypto_2 && crypto_2.webcrypto && crypto_2.webcrypto.subtle) {
                        subtle = crypto_2.webcrypto.subtle;
                    }
                    _a.label = 2;
                case 2: return [2 /*return*/, subtle];
            }
        });
    });
}
exports.getSubtle = getSubtle;
function pbkdf2Sha512Subtle(subtle, secret, salt, iterations, keylen) {
    return __awaiter(this, void 0, void 0, function () {
        return __generator(this, function (_a) {
            (0, utils_1.assert)(subtle, 'Argument subtle is falsy');
            (0, utils_1.assert)(typeof subtle === 'object', 'Argument subtle is not of type object');
            (0, utils_1.assert)(typeof subtle.importKey === 'function', 'subtle.importKey is not a function');
            (0, utils_1.assert)(typeof subtle.deriveBits === 'function', 'subtle.deriveBits is not a function');
            return [2 /*return*/, subtle
                    .importKey('raw', secret, { name: 'PBKDF2' }, false, ['deriveBits'])
                    .then(function (key) {
                    return subtle
                        .deriveBits({
                        name: 'PBKDF2',
                        salt: salt,
                        iterations: iterations,
                        hash: { name: 'SHA-512' },
                    }, key, keylen * 8)
                        .then(function (buffer) { return new Uint8Array(buffer); });
                })];
        });
    });
}
exports.pbkdf2Sha512Subtle = pbkdf2Sha512Subtle;
function pbkdf2Sha512Crypto(crypto, secret, salt, iterations, keylen) {
    return __awaiter(this, void 0, void 0, function () {
        return __generator(this, function (_a) {
            (0, utils_1.assert)(crypto, 'Argument crypto is falsy');
            (0, utils_1.assert)(typeof crypto === 'object', 'Argument crypto is not of type object');
            (0, utils_1.assert)(typeof crypto.pbkdf2 === 'function', 'crypto.pbkdf2 is not a function');
            return [2 /*return*/, new Promise(function (resolve, reject) {
                    crypto.pbkdf2(secret, salt, iterations, keylen, 'sha512', function (error, result) {
                        if (error) {
                            reject(error);
                        }
                        else {
                            resolve(Uint8Array.from(result));
                        }
                    });
                })];
        });
    });
}
exports.pbkdf2Sha512Crypto = pbkdf2Sha512Crypto;
function pbkdf2Sha512Noble(secret, salt, iterations, keylen) {
    return __awaiter(this, void 0, void 0, function () {
        return __generator(this, function (_a) {
            return [2 /*return*/, (0, pbkdf2_1.pbkdf2Async)(sha512_1.sha512, secret, salt, { c: iterations, dkLen: keylen })];
        });
    });
}
exports.pbkdf2Sha512Noble = pbkdf2Sha512Noble;
/**
 * A pbkdf2 implementation for BIP39. This is not exported at package level and thus a private API.
 */
function pbkdf2Sha512(secret, salt, iterations, keylen) {
    return __awaiter(this, void 0, void 0, function () {
        var subtle, crypto_3;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0: return [4 /*yield*/, getSubtle()];
                case 1:
                    subtle = _a.sent();
                    if (!subtle) return [3 /*break*/, 2];
                    return [2 /*return*/, pbkdf2Sha512Subtle(subtle, secret, salt, iterations, keylen)];
                case 2: return [4 /*yield*/, getCryptoModule()];
                case 3:
                    crypto_3 = _a.sent();
                    if (crypto_3) {
                        return [2 /*return*/, pbkdf2Sha512Crypto(crypto_3, secret, salt, iterations, keylen)];
                    }
                    else {
                        return [2 /*return*/, pbkdf2Sha512Noble(secret, salt, iterations, keylen)];
                    }
                    _a.label = 4;
                case 4: return [2 /*return*/];
            }
        });
    });
}
exports.pbkdf2Sha512 = pbkdf2Sha512;
//# sourceMappingURL=pbkdf2.js.map