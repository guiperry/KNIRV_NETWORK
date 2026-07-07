"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.hexToArray = exports.arrayToHex = void 0;
function arrayToHex(data) {
    return Buffer.from(data).toString('hex');
}
exports.arrayToHex = arrayToHex;
function hexToArray(hex) {
    return Uint8Array.from(Buffer.from(hex, 'hex'));
}
exports.hexToArray = hexToArray;
//# sourceMappingURL=data.js.map