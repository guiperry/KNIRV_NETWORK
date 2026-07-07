"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.fromBase64 = exports.toBase64 = void 0;
var base64js = require("base64-js");
function toBase64(data) {
    return base64js.fromByteArray(data);
}
exports.toBase64 = toBase64;
function fromBase64(base64String) {
    if (!base64String.match(/^[a-zA-Z0-9+/]*={0,2}$/)) {
        throw new Error('Invalid base64 string format');
    }
    return base64js.toByteArray(base64String);
}
exports.fromBase64 = fromBase64;
//# sourceMappingURL=base64.js.map