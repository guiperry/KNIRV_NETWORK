"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Bech32 = exports.normalizeBech32 = exports.fromBech32 = exports.toBech32 = void 0;
var bech32_1 = require("bech32");
function toBech32(prefix, data, limit) {
    var address = bech32_1.bech32.encode(prefix, bech32_1.bech32.toWords(data), limit);
    return address;
}
exports.toBech32 = toBech32;
function fromBech32(address, limit) {
    if (limit === void 0) { limit = Infinity; }
    var decodedAddress = bech32_1.bech32.decode(address, limit);
    return {
        prefix: decodedAddress.prefix,
        data: new Uint8Array(bech32_1.bech32.fromWords(decodedAddress.words)),
    };
}
exports.fromBech32 = fromBech32;
/**
 * Takes a bech32 address and returns a normalized (i.e. lower case) representation of it.
 *
 * The input is validated along the way, which makes this significantly safer than
 * using `address.toLowerCase()`.
 */
function normalizeBech32(address) {
    var _a = fromBech32(address), prefix = _a.prefix, data = _a.data;
    return toBech32(prefix, data);
}
exports.normalizeBech32 = normalizeBech32;
/**
 * @deprecated This class is deprecated and will be removed soon. Please use fromBech32() and toBech32() instead. For more details please refer to https://github.com/cosmos/cosmjs/issues/1053.
 */
var Bech32 = /** @class */ (function () {
    function Bech32() {
    }
    /**
     * @deprecated This class is deprecated and will be removed soon. Please use fromBech32() and toBech32() instead. For more details please refer to https://github.com/cosmos/cosmjs/issues/1053.
     */
    Bech32.encode = function (prefix, data, limit) {
        return toBech32(prefix, data, limit);
    };
    /**
     * @deprecated This class is deprecated and will be removed soon. Please use fromBech32() and toBech32() instead. For more details please refer to https://github.com/cosmos/cosmjs/issues/1053.
     */
    Bech32.decode = function (address, limit) {
        if (limit === void 0) { limit = Infinity; }
        return fromBech32(address, limit);
    };
    return Bech32;
}());
exports.Bech32 = Bech32;
//# sourceMappingURL=bech32.js.map