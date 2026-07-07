"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Decimal = void 0;
var bn_js_1 = require("bn.js");
// Too large values lead to massive memory usage. Limit to something sensible.
// The largest value we need is 18 (Ether).
var maxFractionalDigits = 100;
/**
 * A type for arbitrary precision, non-negative decimals.
 *
 * Instances of this class are immutable.
 */
var Decimal = /** @class */ (function () {
    function Decimal(atomics, fractionalDigits) {
        if (!atomics.match(/^[0-9]+$/)) {
            throw new Error('Invalid string format. Only non-negative integers in decimal representation supported.');
        }
        this.data = {
            atomics: new bn_js_1.default(atomics),
            fractionalDigits: fractionalDigits,
        };
    }
    Decimal.fromUserInput = function (input, fractionalDigits) {
        Decimal.verifyFractionalDigits(fractionalDigits);
        var badCharacter = input.match(/[^0-9.]/);
        if (badCharacter) {
            throw new Error("Invalid character at position ".concat(badCharacter.index + 1));
        }
        var whole;
        var fractional;
        if (input === '') {
            whole = '0';
            fractional = '';
        }
        else if (input.search(/\./) === -1) {
            // integer format, no separator
            whole = input;
            fractional = '';
        }
        else {
            var parts = input.split('.');
            switch (parts.length) {
                case 0:
                case 1:
                    throw new Error('Fewer than two elements in split result. This must not happen here.');
                case 2:
                    if (!parts[1])
                        throw new Error('Fractional part missing');
                    whole = parts[0];
                    fractional = parts[1].replace(/0+$/, '');
                    break;
                default:
                    throw new Error('More than one separator found');
            }
        }
        if (fractional.length > fractionalDigits) {
            throw new Error('Got more fractional digits than supported');
        }
        var quantity = "".concat(whole).concat(fractional.padEnd(fractionalDigits, '0'));
        return new Decimal(quantity, fractionalDigits);
    };
    Decimal.fromAtomics = function (atomics, fractionalDigits) {
        Decimal.verifyFractionalDigits(fractionalDigits);
        return new Decimal(atomics, fractionalDigits);
    };
    /**
     * Creates a Decimal with value 0.0 and the given number of fractial digits.
     *
     * Fractional digits are not relevant for the value but needed to be able
     * to perform arithmetic operations with other decimals.
     */
    Decimal.zero = function (fractionalDigits) {
        Decimal.verifyFractionalDigits(fractionalDigits);
        return new Decimal('0', fractionalDigits);
    };
    /**
     * Creates a Decimal with value 1.0 and the given number of fractial digits.
     *
     * Fractional digits are not relevant for the value but needed to be able
     * to perform arithmetic operations with other decimals.
     */
    Decimal.one = function (fractionalDigits) {
        Decimal.verifyFractionalDigits(fractionalDigits);
        return new Decimal('1' + '0'.repeat(fractionalDigits), fractionalDigits);
    };
    Decimal.verifyFractionalDigits = function (fractionalDigits) {
        if (!Number.isInteger(fractionalDigits))
            throw new Error('Fractional digits is not an integer');
        if (fractionalDigits < 0)
            throw new Error('Fractional digits must not be negative');
        if (fractionalDigits > maxFractionalDigits) {
            throw new Error("Fractional digits must not exceed ".concat(maxFractionalDigits));
        }
    };
    Decimal.compare = function (a, b) {
        if (a.fractionalDigits !== b.fractionalDigits)
            throw new Error('Fractional digits do not match');
        return a.data.atomics.cmp(new bn_js_1.default(b.atomics));
    };
    Object.defineProperty(Decimal.prototype, "atomics", {
        get: function () {
            return this.data.atomics.toString();
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Decimal.prototype, "fractionalDigits", {
        get: function () {
            return this.data.fractionalDigits;
        },
        enumerable: false,
        configurable: true
    });
    /** Creates a new instance with the same value */
    Decimal.prototype.clone = function () {
        return new Decimal(this.atomics, this.fractionalDigits);
    };
    /** Returns the greatest decimal <= this which has no fractional part (rounding down) */
    Decimal.prototype.floor = function () {
        var factor = new bn_js_1.default(10).pow(new bn_js_1.default(this.data.fractionalDigits));
        var whole = this.data.atomics.div(factor);
        var fractional = this.data.atomics.mod(factor);
        if (fractional.isZero()) {
            return this.clone();
        }
        else {
            return Decimal.fromAtomics(whole.mul(factor).toString(), this.fractionalDigits);
        }
    };
    /** Returns the smallest decimal >= this which has no fractional part (rounding up) */
    Decimal.prototype.ceil = function () {
        var factor = new bn_js_1.default(10).pow(new bn_js_1.default(this.data.fractionalDigits));
        var whole = this.data.atomics.div(factor);
        var fractional = this.data.atomics.mod(factor);
        if (fractional.isZero()) {
            return this.clone();
        }
        else {
            return Decimal.fromAtomics(whole.addn(1).mul(factor).toString(), this.fractionalDigits);
        }
    };
    Decimal.prototype.toString = function () {
        var factor = new bn_js_1.default(10).pow(new bn_js_1.default(this.data.fractionalDigits));
        var whole = this.data.atomics.div(factor);
        var fractional = this.data.atomics.mod(factor);
        if (fractional.isZero()) {
            return whole.toString();
        }
        else {
            var fullFractionalPart = fractional.toString().padStart(this.data.fractionalDigits, '0');
            var trimmedFractionalPart = fullFractionalPart.replace(/0+$/, '');
            return "".concat(whole.toString(), ".").concat(trimmedFractionalPart);
        }
    };
    /**
     * Returns an approximation as a float type. Only use this if no
     * exact calculation is required.
     */
    Decimal.prototype.toFloatApproximation = function () {
        var out = Number(this.toString());
        if (Number.isNaN(out))
            throw new Error('Conversion to number failed');
        return out;
    };
    /**
     * a.plus(b) returns a+b.
     *
     * Both values need to have the same fractional digits.
     */
    Decimal.prototype.plus = function (b) {
        if (this.fractionalDigits !== b.fractionalDigits)
            throw new Error('Fractional digits do not match');
        var sum = this.data.atomics.add(new bn_js_1.default(b.atomics));
        return new Decimal(sum.toString(), this.fractionalDigits);
    };
    /**
     * a.minus(b) returns a-b.
     *
     * Both values need to have the same fractional digits.
     * The resulting difference needs to be non-negative.
     */
    Decimal.prototype.minus = function (b) {
        if (this.fractionalDigits !== b.fractionalDigits)
            throw new Error('Fractional digits do not match');
        var difference = this.data.atomics.sub(new bn_js_1.default(b.atomics));
        if (difference.ltn(0))
            throw new Error('Difference must not be negative');
        return new Decimal(difference.toString(), this.fractionalDigits);
    };
    /**
     * a.multiply(b) returns a*b.
     *
     * We only allow multiplication by unsigned integers to avoid rounding errors.
     */
    Decimal.prototype.multiply = function (b) {
        var product = this.data.atomics.mul(new bn_js_1.default(b.toString()));
        return new Decimal(product.toString(), this.fractionalDigits);
    };
    Decimal.prototype.equals = function (b) {
        return Decimal.compare(this, b) === 0;
    };
    Decimal.prototype.isLessThan = function (b) {
        return Decimal.compare(this, b) < 0;
    };
    Decimal.prototype.isLessThanOrEqual = function (b) {
        return Decimal.compare(this, b) <= 0;
    };
    Decimal.prototype.isGreaterThan = function (b) {
        return Decimal.compare(this, b) > 0;
    };
    Decimal.prototype.isGreaterThanOrEqual = function (b) {
        return Decimal.compare(this, b) >= 0;
    };
    return Decimal;
}());
exports.Decimal = Decimal;
//# sourceMappingURL=decimal.js.map