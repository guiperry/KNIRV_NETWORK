"use strict";
var __values = (this && this.__values) || function(o) {
    var s = typeof Symbol === "function" && Symbol.iterator, m = s && o[s], i = 0;
    if (m) return m.call(o);
    if (o && typeof o.length === "number") return {
        next: function () {
            if (o && i >= o.length) o = void 0;
            return { value: o && o[i++], done: !o };
        }
    };
    throw new TypeError(s ? "Object is not iterable." : "Symbol.iterator is not defined.");
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.mergeUintArray = exports.toRealUint8Array = void 0;
// See https://github.com/paulmillr/noble-hashes/issues/25 for why this is needed
function toRealUint8Array(data) {
    if (data instanceof Uint8Array)
        return data;
    else
        return Uint8Array.from(data);
}
exports.toRealUint8Array = toRealUint8Array;
function mergeUintArray() {
    var e_1, _a;
    var arraies = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        arraies[_i] = arguments[_i];
    }
    var mappedUintArray = arraies.map(function (array) {
        if (array instanceof Uint8Array)
            return array;
        return Uint8Array.from([array]);
    });
    var totalLength = mappedUintArray.reduce(function (accumulator, current) { return current.length + accumulator; }, 0);
    var result = new Uint8Array(totalLength);
    var index = 0;
    try {
        for (var mappedUintArray_1 = __values(mappedUintArray), mappedUintArray_1_1 = mappedUintArray_1.next(); !mappedUintArray_1_1.done; mappedUintArray_1_1 = mappedUintArray_1.next()) {
            var array = mappedUintArray_1_1.value;
            result.set(array, index);
            index += array.length;
        }
    }
    catch (e_1_1) { e_1 = { error: e_1_1 }; }
    finally {
        try {
            if (mappedUintArray_1_1 && !mappedUintArray_1_1.done && (_a = mappedUintArray_1.return)) _a.call(mappedUintArray_1);
        }
        finally { if (e_1) throw e_1.error; }
    }
    return result;
}
exports.mergeUintArray = mergeUintArray;
//# sourceMappingURL=utils.js.map