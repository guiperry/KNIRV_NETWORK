"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.sha512 = exports.Sha512 = exports.sha256 = exports.Sha256 = void 0;
var sha256_1 = require("@noble/hashes/sha256");
var sha512_1 = require("@noble/hashes/sha512");
var utils_1 = require("./utils");
var Sha256 = /** @class */ (function () {
    function Sha256(firstData) {
        this.blockSize = 512 / 8;
        this.impl = sha256_1.sha256.create();
        if (firstData) {
            this.update(firstData);
        }
    }
    Sha256.prototype.update = function (data) {
        this.impl.update((0, utils_1.toRealUint8Array)(data));
        return this;
    };
    Sha256.prototype.digest = function () {
        return this.impl.digest();
    };
    return Sha256;
}());
exports.Sha256 = Sha256;
/** Convenience function equivalent to `new Sha256(data).digest()` */
function sha256(data) {
    return new Sha256(data).digest();
}
exports.sha256 = sha256;
var Sha512 = /** @class */ (function () {
    function Sha512(firstData) {
        this.blockSize = 1024 / 8;
        this.impl = sha512_1.sha512.create();
        if (firstData) {
            this.update(firstData);
        }
    }
    Sha512.prototype.update = function (data) {
        this.impl.update((0, utils_1.toRealUint8Array)(data));
        return this;
    };
    Sha512.prototype.digest = function () {
        return this.impl.digest();
    };
    return Sha512;
}());
exports.Sha512 = Sha512;
/** Convenience function equivalent to `new Sha512(data).digest()` */
function sha512(data) {
    return new Sha512(data).digest();
}
exports.sha512 = sha512;
//# sourceMappingURL=sha.js.map