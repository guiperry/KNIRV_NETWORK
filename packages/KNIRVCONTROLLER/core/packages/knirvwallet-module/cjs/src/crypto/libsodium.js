"use strict";
// Keep all classes requiring libsodium-js in one file as having multiple
// requiring of the libsodium-wrappers module currently crashes browsers
//
// libsodium.js API: https://gist.github.com/webmaster128/b2dbe6d54d36dd168c9fabf441b9b09c
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
exports.Xchacha20poly1305Ietf = exports.xchacha20NonceLength = exports.Ed25519 = exports.Ed25519Keypair = exports.Argon2id = exports.isArgon2idOptions = void 0;
var libsodium_wrappers_sumo_1 = require("libsodium-wrappers-sumo");
var utils_1 = require("../utils");
function isArgon2idOptions(thing) {
    if (!(0, utils_1.isNonNullObject)(thing))
        return false;
    if (typeof thing.outputLength !== 'number')
        return false;
    if (typeof thing.opsLimit !== 'number')
        return false;
    if (typeof thing.memLimitKib !== 'number')
        return false;
    return true;
}
exports.isArgon2idOptions = isArgon2idOptions;
var Argon2id = /** @class */ (function () {
    function Argon2id() {
    }
    Argon2id.execute = function (password, salt, options) {
        return __awaiter(this, void 0, void 0, function () {
            var adjustedSalt;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, libsodium_wrappers_sumo_1.default.ready];
                    case 1:
                        _a.sent();
                        adjustedSalt = salt;
                        if (salt.length !== 16) {
                            adjustedSalt = new Uint8Array(16);
                            if (salt.length > 16) {
                                adjustedSalt.set(salt.slice(0, 16));
                            }
                            else {
                                adjustedSalt.set(salt);
                                // Fill remaining bytes with zeros (already done by new Uint8Array)
                            }
                        }
                        return [2 /*return*/, libsodium_wrappers_sumo_1.default.crypto_pwhash(options.outputLength, password, adjustedSalt, // Use adjusted salt
                            options.opsLimit, options.memLimitKib * 1024, libsodium_wrappers_sumo_1.default.crypto_pwhash_ALG_ARGON2ID13)];
                }
            });
        });
    };
    return Argon2id;
}());
exports.Argon2id = Argon2id;
var Ed25519Keypair = /** @class */ (function () {
    function Ed25519Keypair(privkey, pubkey) {
        this.privkey = privkey;
        this.pubkey = pubkey;
    }
    // a libsodium privkey has the format `<ed25519 privkey> + <ed25519 pubkey>`
    Ed25519Keypair.fromLibsodiumPrivkey = function (libsodiumPrivkey) {
        if (libsodiumPrivkey.length !== 64) {
            throw new Error("Unexpected key length ".concat(libsodiumPrivkey.length, ". Must be 64."));
        }
        return new Ed25519Keypair(libsodiumPrivkey.slice(0, 32), libsodiumPrivkey.slice(32, 64));
    };
    Ed25519Keypair.prototype.toLibsodiumPrivkey = function () {
        var privateKeyArray = new Uint8Array(this.privkey.length + this.pubkey.length);
        privateKeyArray.set(this.privkey);
        privateKeyArray.set(this.pubkey, this.privkey.length);
        return privateKeyArray;
    };
    return Ed25519Keypair;
}());
exports.Ed25519Keypair = Ed25519Keypair;
var Ed25519 = /** @class */ (function () {
    function Ed25519() {
    }
    /**
     * Generates a keypair deterministically from a given 32 bytes seed.
     *
     * This seed equals the Ed25519 private key.
     * For implementation details see crypto_sign_seed_keypair in
     * https://download.libsodium.org/doc/public-key_cryptography/public-key_signatures.html
     * and diagram on https://blog.mozilla.org/warner/2011/11/29/ed25519-keys/
     */
    Ed25519.makeKeypair = function (seed) {
        return __awaiter(this, void 0, void 0, function () {
            var keypair;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, libsodium_wrappers_sumo_1.default.ready];
                    case 1:
                        _a.sent();
                        keypair = libsodium_wrappers_sumo_1.default.crypto_sign_seed_keypair(seed);
                        return [2 /*return*/, Ed25519Keypair.fromLibsodiumPrivkey(keypair.privateKey)];
                }
            });
        });
    };
    Ed25519.createSignature = function (message, keyPair) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, libsodium_wrappers_sumo_1.default.ready];
                    case 1:
                        _a.sent();
                        return [2 /*return*/, libsodium_wrappers_sumo_1.default.crypto_sign_detached(message, keyPair.toLibsodiumPrivkey())];
                }
            });
        });
    };
    Ed25519.verifySignature = function (signature, message, pubkey) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, libsodium_wrappers_sumo_1.default.ready];
                    case 1:
                        _a.sent();
                        return [2 /*return*/, libsodium_wrappers_sumo_1.default.crypto_sign_verify_detached(signature, message, pubkey)];
                }
            });
        });
    };
    return Ed25519;
}());
exports.Ed25519 = Ed25519;
/**
 * Nonce length in bytes for all flavours of XChaCha20.
 *
 * @see https://libsodium.gitbook.io/doc/advanced/stream_ciphers/xchacha20#notes
 */
exports.xchacha20NonceLength = 24;
var Xchacha20poly1305Ietf = /** @class */ (function () {
    function Xchacha20poly1305Ietf() {
    }
    Xchacha20poly1305Ietf.encrypt = function (message, key, nonce) {
        return __awaiter(this, void 0, void 0, function () {
            var additionalData;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, libsodium_wrappers_sumo_1.default.ready];
                    case 1:
                        _a.sent();
                        additionalData = null;
                        return [2 /*return*/, libsodium_wrappers_sumo_1.default.crypto_aead_xchacha20poly1305_ietf_encrypt(message, additionalData, null, // secret nonce: unused and should be null (https://download.libsodium.org/doc/secret-key_cryptography/aead/chacha20-poly1305/xchacha20-poly1305_construction)
                            nonce, key)];
                }
            });
        });
    };
    Xchacha20poly1305Ietf.decrypt = function (ciphertext, key, nonce) {
        return __awaiter(this, void 0, void 0, function () {
            var additionalData;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, libsodium_wrappers_sumo_1.default.ready];
                    case 1:
                        _a.sent();
                        additionalData = null;
                        return [2 /*return*/, libsodium_wrappers_sumo_1.default.crypto_aead_xchacha20poly1305_ietf_decrypt(null, // secret nonce: unused and should be null (https://download.libsodium.org/doc/secret-key_cryptography/aead/chacha20-poly1305/xchacha20-poly1305_construction)
                            ciphertext, additionalData, nonce, key)];
                }
            });
        });
    };
    return Xchacha20poly1305Ietf;
}());
exports.Xchacha20poly1305Ietf = Xchacha20poly1305Ietf;
//# sourceMappingURL=libsodium.js.map