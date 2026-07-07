"use strict";
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
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
exports.strToSignedTx = exports.txToDocument = exports.documentToDefaultTx = exports.documentToTx = exports.decodeTxMessages = exports.KNIRVMessageTypes = void 0;
// KNIRV Network transaction types and utilities
var encoding_1 = require("../encoding");
// KNIRV Message Types
exports.KNIRVMessageTypes = {
    MSG_SEND: '/knirv.transaction.v1.MsgSend',
    MSG_CALL: '/knirv.transaction.v1.MsgCall',
    MSG_ADD_PACKAGE: '/knirv.transaction.v1.MsgAddPackage',
    MSG_RUN: '/knirv.transaction.v1.MsgRun',
    MSG_MCP_INVOKE: '/knirv.mcp.v1.MsgInvoke',
    MSG_MCP_REGISTER: '/knirv.mcp.v1.MsgRegister',
};
var decodeTxMessages = function (messages) {
    return messages.map(function (m) {
        // For KNIRV, we assume the value is already JSON-encoded or can be parsed directly
        try {
            var valueStr = new TextDecoder().decode(m.value);
            var parsedValue = JSON.parse(valueStr);
            return __assign({ '@type': m.type_url }, parsedValue);
        }
        catch (error) {
            // If parsing fails, return the raw value
            return {
                '@type': m.type_url,
                value: m.value,
            };
        }
    });
};
exports.decodeTxMessages = decodeTxMessages;
function createMemPackage(memPackage) {
    return {
        name: memPackage.name,
        path: memPackage.path,
        files: memPackage.files.map(function (file) { return ({
            name: file.name,
            body: file.body,
        }); }),
    };
}
function encodeMessageValue(message) {
    // For KNIRV, we encode the message value as JSON
    var jsonValue = JSON.stringify(message.value);
    var encodedValue = new TextEncoder().encode(jsonValue);
    return {
        type_url: message.type,
        value: encodedValue,
    };
}
function documentToTx(document) {
    var messages = document.msgs.map(encodeMessageValue);
    return {
        body: {
            messages: messages,
            memo: document.memo,
            timeout_height: '0',
            extension_options: [],
            non_critical_extension_options: [],
        },
        auth_info: {
            signer_infos: [],
            fee: {
                amount: document.fee.amount,
                gas: document.fee.gas,
                granter: document.fee.granter,
                payer: document.fee.payer,
            },
        },
        signatures: [],
    };
}
exports.documentToTx = documentToTx;
function documentToDefaultTx(document) {
    var messages = document.msgs.map(encodeMessageValue);
    return {
        body: {
            messages: messages,
            memo: document.memo,
            timeout_height: '0',
            extension_options: [],
            non_critical_extension_options: [],
        },
        auth_info: {
            signer_infos: [
                {
                    public_key: {
                        key: '',
                    },
                    mode_info: {
                        single: {
                            mode: 1,
                        },
                    },
                    sequence: '0',
                },
            ],
            fee: {
                amount: document.fee.amount,
                gas: document.fee.gas,
                granter: document.fee.granter,
                payer: document.fee.payer,
            },
        },
        signatures: [''],
    };
}
exports.documentToDefaultTx = documentToDefaultTx;
function txToDocument(tx) {
    return {
        chain_id: '',
        account_number: '0',
        sequence: '0',
        fee: tx.auth_info.fee,
        msgs: tx.body.messages.map(function (msg) { return ({
            type: msg.type_url,
            value: JSON.parse(new TextDecoder().decode(msg.value)),
        }); }),
        memo: tx.body.memo,
    };
}
exports.txToDocument = txToDocument;
/**
 * Change transaction json string to a Signed Tx.
 *
 * @param str
 * @returns Tx | null
 */
var strToSignedTx = function (str) {
    var rawTx = null;
    try {
        rawTx = JSON.parse(str);
    }
    catch (e) {
        console.error(e);
    }
    if (rawTx === null)
        return null;
    try {
        var document_1 = rawTx;
        var messages = document_1.msg
            .map(function (msg) { return ({
            type: msg['@type'],
            value: __assign({}, msg),
        }); })
            .map(encodeMessageValue);
        return {
            body: {
                messages: messages,
                memo: document_1.memo,
                timeout_height: '0',
                extension_options: [],
                non_critical_extension_options: [],
            },
            auth_info: {
                signer_infos: document_1.signatures.map(function (signature) {
                    var _a;
                    var publicKeyBytes = (0, encoding_1.fromBase64)(((_a = signature === null || signature === void 0 ? void 0 : signature.pub_key) === null || _a === void 0 ? void 0 : _a.value) || '');
                    return {
                        public_key: {
                            key: btoa(String.fromCharCode.apply(String, __spreadArray([], __read(publicKeyBytes), false))),
                        },
                        mode_info: {
                            single: {
                                mode: 1,
                            },
                        },
                        sequence: '0',
                    };
                }),
                fee: {
                    amount: [],
                    gas: document_1.fee.gas_wanted || '0',
                    granter: undefined,
                    payer: undefined,
                },
            },
            signatures: document_1.signatures.map(function (signature) {
                return (signature === null || signature === void 0 ? void 0 : signature.signature) || '';
            }),
        };
    }
    catch (e) {
        console.error(e);
        return null;
    }
};
exports.strToSignedTx = strToSignedTx;
//# sourceMappingURL=messages.js.map