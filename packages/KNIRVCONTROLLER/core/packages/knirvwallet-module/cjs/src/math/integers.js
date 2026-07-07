"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Uint64 = exports.Uint53 = exports.Int53 = exports.Uint32 = void 0;
var bn_js_1 = require("bn.js");
var uint64MaxValue = new bn_js_1.default('18446744073709551615', 10, 'be');
// interface IntegerStatic<T> {
//   readonly fromString: (str: string) => T;
// }
// interface FixedLengthIntegerStatic<T> {
//   readonly fromBytes: (bytes: ArrayLike<number>, endianess: 'be' | 'le') => T;
// }
var Uint32 = /** @class */ (function () {
    function Uint32(input) {
        if (Number.isNaN(input)) {
            throw new Error('Input is not a number');
        }
        if (!Number.isInteger(input)) {
            throw new Error('Input is not an integer');
        }
        if (input < 0 || input > 4294967295) {
            throw new Error('Input not in uint32 range: ' + input.toString());
        }
        this.data = input;
    }
    /** @deprecated use Uint32.fromBytes */
    Uint32.fromBigEndianBytes = function (bytes) {
        return Uint32.fromBytes(bytes);
    };
    /**
     * Creates a Uint32 from a fixed length byte array.
     *
     * @param bytes a list of exactly 4 bytes
     * @param endianess defaults to big endian
     */
    Uint32.fromBytes = function (bytes, endianess) {
        if (endianess === void 0) { endianess = 'be'; }
        if (bytes.length !== 4) {
            throw new Error('Invalid input length. Expected 4 bytes.');
        }
        for (var i = 0; i < bytes.length; ++i) {
            if (!Number.isInteger(bytes[i]) || bytes[i] > 255 || bytes[i] < 0) {
                throw new Error('Invalid value in byte. Found: ' + bytes[i]);
            }
        }
        var beBytes = endianess === 'be' ? bytes : Array.from(bytes).reverse();
        // Use mulitiplication instead of shifting since bitwise operators are defined
        // on SIGNED int32 in JavaScript and we don't want to risk surprises
        return new Uint32(beBytes[0] * Math.pow(2, 24) + beBytes[1] * Math.pow(2, 16) + beBytes[2] * Math.pow(2, 8) + beBytes[3]);
    };
    Uint32.fromString = function (str) {
        if (!str.match(/^[0-9]+$/)) {
            throw new Error('Invalid string format');
        }
        return new Uint32(Number.parseInt(str, 10));
    };
    Uint32.prototype.toBytesBigEndian = function () {
        // Use division instead of shifting since bitwise operators are defined
        // on SIGNED int32 in JavaScript and we don't want to risk surprises
        return new Uint8Array([
            Math.floor(this.data / Math.pow(2, 24)) & 0xff,
            Math.floor(this.data / Math.pow(2, 16)) & 0xff,
            Math.floor(this.data / Math.pow(2, 8)) & 0xff,
            Math.floor(this.data / Math.pow(2, 0)) & 0xff,
        ]);
    };
    Uint32.prototype.toBytesLittleEndian = function () {
        // Use division instead of shifting since bitwise operators are defined
        // on SIGNED int32 in JavaScript and we don't want to risk surprises
        return new Uint8Array([
            Math.floor(this.data / Math.pow(2, 0)) & 0xff,
            Math.floor(this.data / Math.pow(2, 8)) & 0xff,
            Math.floor(this.data / Math.pow(2, 16)) & 0xff,
            Math.floor(this.data / Math.pow(2, 24)) & 0xff,
        ]);
    };
    Uint32.prototype.toNumber = function () {
        return this.data;
    };
    Uint32.prototype.toBigInt = function () {
        return BigInt(this.toNumber());
    };
    Uint32.prototype.toString = function () {
        return this.data.toString();
    };
    return Uint32;
}());
exports.Uint32 = Uint32;
var Int53 = /** @class */ (function () {
    function Int53(input) {
        if (Number.isNaN(input)) {
            throw new Error('Input is not a number');
        }
        if (!Number.isInteger(input)) {
            throw new Error('Input is not an integer');
        }
        if (input < Number.MIN_SAFE_INTEGER || input > Number.MAX_SAFE_INTEGER) {
            throw new Error('Input not in int53 range: ' + input.toString());
        }
        this.data = input;
    }
    Int53.fromString = function (str) {
        if (!str.match(/^-?[0-9]+$/)) {
            throw new Error('Invalid string format');
        }
        return new Int53(Number.parseInt(str, 10));
    };
    Int53.prototype.toNumber = function () {
        return this.data;
    };
    Int53.prototype.toBigInt = function () {
        return BigInt(this.toNumber());
    };
    Int53.prototype.toString = function () {
        return this.data.toString();
    };
    return Int53;
}());
exports.Int53 = Int53;
var Uint53 = /** @class */ (function () {
    function Uint53(input) {
        var signed = new Int53(input);
        if (signed.toNumber() < 0) {
            throw new Error('Input is negative');
        }
        this.data = signed;
    }
    Uint53.fromString = function (str) {
        var signed = Int53.fromString(str);
        return new Uint53(signed.toNumber());
    };
    Uint53.prototype.toNumber = function () {
        return this.data.toNumber();
    };
    Uint53.prototype.toBigInt = function () {
        return BigInt(this.toNumber());
    };
    Uint53.prototype.toString = function () {
        return this.data.toString();
    };
    return Uint53;
}());
exports.Uint53 = Uint53;
var Uint64 = /** @class */ (function () {
    function Uint64(data) {
        if (data.isNeg()) {
            throw new Error('Input is negative');
        }
        if (data.gt(uint64MaxValue)) {
            throw new Error('Input exceeds uint64 range');
        }
        this.data = data;
    }
    /** @deprecated use Uint64.fromBytes */
    Uint64.fromBytesBigEndian = function (bytes) {
        return Uint64.fromBytes(bytes);
    };
    /**
     * Creates a Uint64 from a fixed length byte array.
     *
     * @param bytes a list of exactly 8 bytes
     * @param endianess defaults to big endian
     */
    Uint64.fromBytes = function (bytes, endianess) {
        if (endianess === void 0) { endianess = 'be'; }
        if (bytes.length !== 8) {
            throw new Error('Invalid input length. Expected 8 bytes.');
        }
        for (var i = 0; i < bytes.length; ++i) {
            if (!Number.isInteger(bytes[i]) || bytes[i] > 255 || bytes[i] < 0) {
                throw new Error('Invalid value in byte. Found: ' + bytes[i]);
            }
        }
        var beBytes = endianess === 'be' ? Array.from(bytes) : Array.from(bytes).reverse();
        return new Uint64(new bn_js_1.default(beBytes));
    };
    Uint64.fromString = function (str) {
        if (!str.match(/^[0-9]+$/)) {
            throw new Error('Invalid string format');
        }
        return new Uint64(new bn_js_1.default(str, 10, 'be'));
    };
    Uint64.fromNumber = function (input) {
        if (Number.isNaN(input)) {
            throw new Error('Input is not a number');
        }
        if (!Number.isInteger(input)) {
            throw new Error('Input is not an integer');
        }
        var bigint;
        try {
            bigint = new bn_js_1.default(input);
        }
        catch (_a) {
            throw new Error('Input is not a safe integer');
        }
        return new Uint64(bigint);
    };
    Uint64.prototype.toBytesBigEndian = function () {
        return Uint8Array.from(this.data.toArray('be', 8));
    };
    Uint64.prototype.toBytesLittleEndian = function () {
        return Uint8Array.from(this.data.toArray('le', 8));
    };
    Uint64.prototype.toString = function () {
        return this.data.toString(10);
    };
    Uint64.prototype.toBigInt = function () {
        return BigInt(this.toString());
    };
    Uint64.prototype.toNumber = function () {
        return this.data.toNumber();
    };
    return Uint64;
}());
exports.Uint64 = Uint64;
// Assign classes to unused variables in order to verify static interface conformance at compile time.
// Workaround for https://github.com/microsoft/TypeScript/issues/33892
// const _int53Class: IntegerStatic<Int53> = Int53;
// const _uint53Class: IntegerStatic<Uint53> = Uint53;
// const _uint32Class: IntegerStatic<Uint32> & FixedLengthIntegerStatic<Uint32> = Uint32;
// const _uint64Class: IntegerStatic<Uint64> & FixedLengthIntegerStatic<Uint64> = Uint64;
//# sourceMappingURL=integers.js.map