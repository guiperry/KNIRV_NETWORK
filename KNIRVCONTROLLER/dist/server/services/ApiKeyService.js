"use strict";
/**
 * API Key Service
 * Manages API key generation, validation, and authentication for KNIRVCONTROLLER
 */
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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype);
    return g.next = verb(0), g["throw"] = verb(1), g["return"] = verb(2), typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
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
exports.apiKeyService = void 0;
var RxDBService_1 = require("./RxDBService");
// Browser-compatible crypto implementation
var browserCrypto = {
    randomBytes: function (size) {
        var array = new Uint8Array(size);
        if (typeof window !== 'undefined' && window.crypto && window.crypto.getRandomValues) {
            window.crypto.getRandomValues(array);
        }
        else {
            // Fallback for older browsers
            for (var i = 0; i < size; i++) {
                array[i] = Math.floor(Math.random() * 256);
            }
        }
        return {
            toString: function (encoding) {
                if (encoding === 'hex') {
                    return Array.from(array).map(function (b) { return b.toString(16).padStart(2, '0'); }).join('');
                }
                return array.toString();
            }
        };
    }
};
var ApiKeyService = /** @class */ (function () {
    function ApiKeyService() {
        this.defaultRateLimit = {
            requestsPerMinute: 60,
            requestsPerHour: 1000,
            requestsPerDay: 10000
        };
        this.availablePermissions = [
            'read:agents',
            'write:agents',
            'read:graph',
            'write:graph',
            'read:cortex',
            'write:cortex',
            'read:wallet',
            'write:wallet',
            'read:skills',
            'write:skills',
            'read:analytics',
            'admin:all'
        ];
    }
    /**
     * Generate a new API key
     */
    ApiKeyService.prototype.createApiKey = function (request) {
        return __awaiter(this, void 0, void 0, function () {
            var keyId, apiKey, newKey;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        keyId = this.generateId();
                        apiKey = this.generateApiKey();
                        newKey = {
                            id: keyId,
                            key: apiKey,
                            name: request.name,
                            description: request.description,
                            permissions: this.validatePermissions(request.permissions),
                            createdAt: new Date(),
                            expiresAt: request.expiresAt,
                            isActive: true,
                            usageCount: 0,
                            rateLimit: __assign(__assign({}, this.defaultRateLimit), request.rateLimit),
                            metadata: {}
                        };
                        return [4 /*yield*/, this.saveApiKey(newKey)];
                    case 1:
                        _a.sent();
                        return [2 /*return*/, newKey];
                }
            });
        });
    };
    /**
     * Validate an API key and return key information
     */
    ApiKeyService.prototype.validateApiKey = function (key) {
        return __awaiter(this, void 0, void 0, function () {
            var apiKeys, apiKey, error_1;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 5, , 6]);
                        return [4 /*yield*/, this.getApiKeys()];
                    case 1:
                        apiKeys = _a.sent();
                        apiKey = apiKeys.find(function (k) { return k.key === key; });
                        if (!apiKey) {
                            return [2 /*return*/, null];
                        }
                        // Check if key is active
                        if (!apiKey.isActive) {
                            return [2 /*return*/, null];
                        }
                        if (!(apiKey.expiresAt && new Date() > apiKey.expiresAt)) return [3 /*break*/, 3];
                        return [4 /*yield*/, this.deactivateApiKey(apiKey.id)];
                    case 2:
                        _a.sent();
                        return [2 /*return*/, null];
                    case 3: 
                    // Update last used timestamp
                    return [4 /*yield*/, this.updateLastUsed(apiKey.id)];
                    case 4:
                        // Update last used timestamp
                        _a.sent();
                        return [2 /*return*/, apiKey];
                    case 5:
                        error_1 = _a.sent();
                        console.error('Failed to validate API key:', error_1);
                        return [2 /*return*/, null];
                    case 6: return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Check if API key has required permission
     */
    ApiKeyService.prototype.hasPermission = function (apiKey, requiredPermission) {
        // Admin permission grants all access
        if (apiKey.permissions.includes('admin:all')) {
            return true;
        }
        return apiKey.permissions.includes(requiredPermission);
    };
    /**
     * Check rate limits for API key
     */
    ApiKeyService.prototype.checkRateLimit = function (apiKey) {
        return __awaiter(this, void 0, void 0, function () {
            var now, usage, minuteAgo_1, minuteUsage, hourAgo_1, hourUsage, dayAgo_1, dayUsage, error_2;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        now = new Date();
                        return [4 /*yield*/, this.getApiKeyUsage(apiKey.id)];
                    case 1:
                        usage = _a.sent();
                        minuteAgo_1 = new Date(now.getTime() - 60 * 1000);
                        minuteUsage = usage.filter(function (u) { return u.timestamp > minuteAgo_1; }).length;
                        if (minuteUsage >= apiKey.rateLimit.requestsPerMinute) {
                            return [2 /*return*/, { allowed: false, resetTime: new Date(minuteAgo_1.getTime() + 60 * 1000) }];
                        }
                        hourAgo_1 = new Date(now.getTime() - 60 * 60 * 1000);
                        hourUsage = usage.filter(function (u) { return u.timestamp > hourAgo_1; }).length;
                        if (hourUsage >= apiKey.rateLimit.requestsPerHour) {
                            return [2 /*return*/, { allowed: false, resetTime: new Date(hourAgo_1.getTime() + 60 * 60 * 1000) }];
                        }
                        dayAgo_1 = new Date(now.getTime() - 24 * 60 * 60 * 1000);
                        dayUsage = usage.filter(function (u) { return u.timestamp > dayAgo_1; }).length;
                        if (dayUsage >= apiKey.rateLimit.requestsPerDay) {
                            return [2 /*return*/, { allowed: false, resetTime: new Date(dayAgo_1.getTime() + 24 * 60 * 60 * 1000) }];
                        }
                        return [2 /*return*/, { allowed: true }];
                    case 2:
                        error_2 = _a.sent();
                        console.error('Failed to check rate limit:', error_2);
                        return [2 /*return*/, { allowed: true }]; // Allow on error to prevent service disruption
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Record API key usage
     */
    ApiKeyService.prototype.recordUsage = function (usage) {
        return __awaiter(this, void 0, void 0, function () {
            var db, error_3;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 3, , 4]);
                        db = RxDBService_1.rxdbService.getDatabase();
                        return [4 /*yield*/, db.settings.upsert({
                                id: "api_usage_".concat(usage.keyId, "_").concat(Date.now()),
                                type: 'settings',
                                key: 'api_usage',
                                value: JSON.stringify(usage),
                                timestamp: Date.now()
                            })];
                    case 1:
                        _a.sent();
                        // Increment usage count
                        return [4 /*yield*/, this.incrementUsageCount(usage.keyId)];
                    case 2:
                        // Increment usage count
                        _a.sent();
                        return [3 /*break*/, 4];
                    case 3:
                        error_3 = _a.sent();
                        console.error('Failed to record API usage:', error_3);
                        return [3 /*break*/, 4];
                    case 4: return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Get all API keys
     */
    ApiKeyService.prototype.getApiKeys = function () {
        return __awaiter(this, void 0, void 0, function () {
            var db, settings, error_4;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        db = RxDBService_1.rxdbService.getDatabase();
                        return [4 /*yield*/, db.settings.findOne({ selector: { key: 'api_keys' } }).exec()];
                    case 1:
                        settings = _a.sent();
                        if (settings) {
                            return [2 /*return*/, JSON.parse(settings.value)];
                        }
                        return [2 /*return*/, []];
                    case 2:
                        error_4 = _a.sent();
                        console.error('Failed to load API keys:', error_4);
                        return [2 /*return*/, []];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Get API key usage history
     */
    ApiKeyService.prototype.getApiKeyUsage = function (keyId) {
        return __awaiter(this, void 0, void 0, function () {
            var db, usageRecords, usage, _i, usageRecords_1, record, usageData, error_5;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        db = RxDBService_1.rxdbService.getDatabase();
                        return [4 /*yield*/, db.settings.find({
                                selector: { key: 'api_usage' }
                            }).exec()];
                    case 1:
                        usageRecords = _a.sent();
                        usage = [];
                        for (_i = 0, usageRecords_1 = usageRecords; _i < usageRecords_1.length; _i++) {
                            record = usageRecords_1[_i];
                            try {
                                usageData = JSON.parse(record.value);
                                if (usageData.keyId === keyId) {
                                    usage.push(usageData);
                                }
                            }
                            catch (_b) {
                                // Skip invalid records
                            }
                        }
                        return [2 /*return*/, usage.sort(function (a, b) { return b.timestamp.getTime() - a.timestamp.getTime(); })];
                    case 2:
                        error_5 = _a.sent();
                        console.error('Failed to load API usage:', error_5);
                        return [2 /*return*/, []];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Deactivate an API key
     */
    ApiKeyService.prototype.deactivateApiKey = function (keyId) {
        return __awaiter(this, void 0, void 0, function () {
            var apiKeys, updatedKeys;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getApiKeys()];
                    case 1:
                        apiKeys = _a.sent();
                        updatedKeys = apiKeys.map(function (key) {
                            return key.id === keyId ? __assign(__assign({}, key), { isActive: false }) : key;
                        });
                        return [4 /*yield*/, this.saveApiKeys(updatedKeys)];
                    case 2:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Delete an API key
     */
    ApiKeyService.prototype.deleteApiKey = function (keyId) {
        return __awaiter(this, void 0, void 0, function () {
            var apiKeys, updatedKeys;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getApiKeys()];
                    case 1:
                        apiKeys = _a.sent();
                        updatedKeys = apiKeys.filter(function (key) { return key.id !== keyId; });
                        return [4 /*yield*/, this.saveApiKeys(updatedKeys)];
                    case 2:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    /**
     * Get available permissions
     */
    ApiKeyService.prototype.getAvailablePermissions = function () {
        return __spreadArray([], this.availablePermissions, true);
    };
    /**
     * Private helper methods
     */
    ApiKeyService.prototype.generateApiKey = function () {
        var prefix = 'knirv_';
        var randomBytes = browserCrypto.randomBytes(32).toString('hex');
        return prefix + randomBytes;
    };
    ApiKeyService.prototype.generateId = function () {
        return browserCrypto.randomBytes(16).toString('hex');
    };
    ApiKeyService.prototype.validatePermissions = function (permissions) {
        var _this = this;
        return permissions.filter(function (p) { return _this.availablePermissions.includes(p); });
    };
    ApiKeyService.prototype.saveApiKey = function (apiKey) {
        return __awaiter(this, void 0, void 0, function () {
            var apiKeys;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getApiKeys()];
                    case 1:
                        apiKeys = _a.sent();
                        apiKeys.push(apiKey);
                        return [4 /*yield*/, this.saveApiKeys(apiKeys)];
                    case 2:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    ApiKeyService.prototype.saveApiKeys = function (apiKeys) {
        return __awaiter(this, void 0, void 0, function () {
            var db, error_6;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        db = RxDBService_1.rxdbService.getDatabase();
                        return [4 /*yield*/, db.settings.upsert({
                                id: 'api_keys',
                                type: 'settings',
                                key: 'api_keys',
                                value: JSON.stringify(apiKeys),
                                timestamp: Date.now()
                            })];
                    case 1:
                        _a.sent();
                        return [3 /*break*/, 3];
                    case 2:
                        error_6 = _a.sent();
                        console.error('Failed to save API keys:', error_6);
                        throw error_6;
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    ApiKeyService.prototype.updateLastUsed = function (keyId) {
        return __awaiter(this, void 0, void 0, function () {
            var apiKeys, updatedKeys;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getApiKeys()];
                    case 1:
                        apiKeys = _a.sent();
                        updatedKeys = apiKeys.map(function (key) {
                            return key.id === keyId ? __assign(__assign({}, key), { lastUsed: new Date() }) : key;
                        });
                        return [4 /*yield*/, this.saveApiKeys(updatedKeys)];
                    case 2:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    ApiKeyService.prototype.incrementUsageCount = function (keyId) {
        return __awaiter(this, void 0, void 0, function () {
            var apiKeys, updatedKeys;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getApiKeys()];
                    case 1:
                        apiKeys = _a.sent();
                        updatedKeys = apiKeys.map(function (key) {
                            return key.id === keyId ? __assign(__assign({}, key), { usageCount: key.usageCount + 1 }) : key;
                        });
                        return [4 /*yield*/, this.saveApiKeys(updatedKeys)];
                    case 2:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    return ApiKeyService;
}());
exports.apiKeyService = new ApiKeyService();
