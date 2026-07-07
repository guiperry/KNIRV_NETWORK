"use strict";
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
exports.Random = void 0;
var Random = /** @class */ (function () {
    function Random() {
    }
    /**
     * Returns `count` cryptographically secure random bytes
     */
    Random.getBytes = function (count) {
        try {
            var globalObject = typeof window === 'object' ? window : self;
            var cryptoApi = typeof globalObject.crypto !== 'undefined' ? globalObject.crypto : globalObject.msCrypto;
            var out = new Uint8Array(count);
            cryptoApi.getRandomValues(out);
            return out;
        }
        catch (_a) {
            try {
                var crypto_1 = require('crypto');
                return new Uint8Array(__spreadArray([], __read(crypto_1.randomBytes(count)), false));
            }
            catch (_b) {
                throw new Error('No secure random number generator found');
            }
        }
    };
    return Random;
}());
exports.Random = Random;
//# sourceMappingURL=random.js.map