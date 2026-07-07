"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.fromHex = exports.toHex = void 0;
function toHex(data) {
    var out = '';
    var bytes = data.entries();
    var byte = bytes.next();
    while (!byte.done) {
        out += ('0' + byte.value[1].toString(16)).slice(-2);
        byte = bytes.next();
    }
    return out;
}
exports.toHex = toHex;
function fromHex(hexstring) {
    if (hexstring.length % 2 !== 0) {
        throw new Error('hex string length must be a multiple of 2');
    }
    var out = new Uint8Array(hexstring.length / 2);
    for (var i = 0; i < out.length; i++) {
        var j = 2 * i;
        var hexByteAsString = hexstring.slice(j, j + 2);
        if (!hexByteAsString.match(/[0-9a-f]{2}/i)) {
            throw new Error('hex string contains invalid characters');
        }
        out[i] = parseInt(hexByteAsString, 16);
    }
    return out;
}
exports.fromHex = fromHex;
//# sourceMappingURL=hex.js.map