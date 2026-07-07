"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toRfc3339 = exports.fromRfc3339 = void 0;
var rfc3339Matcher = /^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2}):(\d{2})(\.\d{1,9})?((?:[+-]\d{2}:\d{2})|Z)$/;
function padded(integer, length) {
    if (length === void 0) { length = 2; }
    return integer.toString().padStart(length, '0');
}
function fromRfc3339(str) {
    var matches = rfc3339Matcher.exec(str);
    if (!matches) {
        throw new Error('Date string is not in RFC3339 format');
    }
    var year = +matches[1];
    var month = +matches[2];
    var day = +matches[3];
    var hour = +matches[4];
    var minute = +matches[5];
    var second = +matches[6];
    // fractional seconds match either undefined or a string like ".1", ".123456789"
    var milliSeconds = matches[7] ? Math.floor(+matches[7] * 1000) : 0;
    var tzOffsetSign;
    var tzOffsetHours;
    var tzOffsetMinutes;
    // if timezone is undefined, it must be Z or nothing (otherwise the group would have captured).
    if (matches[8] === 'Z') {
        tzOffsetSign = 1;
        tzOffsetHours = 0;
        tzOffsetMinutes = 0;
    }
    else {
        tzOffsetSign = matches[8].substring(0, 1) === '-' ? -1 : 1;
        tzOffsetHours = +matches[8].substring(1, 3);
        tzOffsetMinutes = +matches[8].substring(4, 6);
    }
    var tzOffset = tzOffsetSign * (tzOffsetHours * 60 + tzOffsetMinutes) * 60; // seconds
    var timestamp = Date.UTC(year, month - 1, day, hour, minute, second, milliSeconds) - tzOffset * 1000;
    return new Date(timestamp);
}
exports.fromRfc3339 = fromRfc3339;
function toRfc3339(date) {
    var year = date.getUTCFullYear();
    var month = padded(date.getUTCMonth() + 1);
    var day = padded(date.getUTCDate());
    var hour = padded(date.getUTCHours());
    var minute = padded(date.getUTCMinutes());
    var second = padded(date.getUTCSeconds());
    var ms = padded(date.getUTCMilliseconds(), 3);
    return "".concat(year, "-").concat(month, "-").concat(day, "T").concat(hour, ":").concat(minute, ":").concat(second, ".").concat(ms, "Z");
}
exports.toRfc3339 = toRfc3339;
//# sourceMappingURL=rfc3339.js.map