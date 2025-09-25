"use strict";
/**
 * RxDB Service for Secure Local Database Storage
 * Manages encrypted RxDB instance for wallet data and sensitive information
 */
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.rxdbService = exports.RxDBService = void 0;
var rxdb_1 = require("rxdb");
var storage_dexie_1 = require("rxdb/plugins/storage-dexie");
var dev_mode_1 = require("rxdb/plugins/dev-mode");
var validate_ajv_1 = require("rxdb/plugins/validate-ajv");
// Add plugins for development mode
if (process.env.NODE_ENV === 'development' || process.env.NODE_ENV === 'test') {
    (0, rxdb_1.addRxPlugin)(dev_mode_1.RxDBDevModePlugin);
}
var RxDBService = /** @class */ (function () {
    function RxDBService(encryptionKey) {
        this.db = null;
        this.isInitialized = false;
        this.encryptionKey = encryptionKey || 'knirv-wallet-default-key-2025';
    }
    RxDBService.prototype.initialize = function () {
        return __awaiter(this, void 0, void 0, function () {
            var _a, error_1;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        if (this.isInitialized)
                            return [2 /*return*/];
                        _b.label = 1;
                    case 1:
                        _b.trys.push([1, 4, , 5]);
                        console.log('Initializing RxDB database...');
                        _a = this;
                        return [4 /*yield*/, (0, rxdb_1.createRxDatabase)({
                                name: 'knirv_wallet_db',
                                storage: (0, validate_ajv_1.wrappedValidateAjvStorage)({
                                    storage: (0, storage_dexie_1.getRxStorageDexie)()
                                })
                            })];
                    case 2:
                        _a.db = _b.sent();
                        return [4 /*yield*/, this.createCollections()];
                    case 3:
                        _b.sent();
                        this.isInitialized = true;
                        console.log('RxDB database initialized successfully');
                        return [3 /*break*/, 5];
                    case 4:
                        error_1 = _b.sent();
                        console.error('Failed to initialize RxDB:', error_1);
                        throw error_1;
                    case 5: return [2 /*return*/];
                }
            });
        });
    };
    RxDBService.prototype.createCollections = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.db)
                            throw new Error('Database not initialized');
                        return [4 /*yield*/, this.db.addCollections({
                                wallets: {
                                    schema: {
                                        title: 'wallet schema',
                                        version: 0,
                                        type: 'object',
                                        primaryKey: 'id',
                                        properties: {
                                            id: { type: 'string' },
                                            type: { type: 'string' },
                                            address: { type: 'string' },
                                            name: { type: 'string' },
                                            encryptedPrivateKey: { type: 'string' },
                                            publicKey: { type: 'string' },
                                            balance: { type: 'string' },
                                            usdcBalance: { type: 'string' },
                                            nrnBalance: { type: 'string' },
                                            lastSync: { type: 'number' },
                                            createdAt: { type: 'number' },
                                            updatedAt: { type: 'number' }
                                        },
                                        required: ['id', 'type', 'address', 'name', 'publicKey', 'balance', 'usdcBalance', 'nrnBalance']
                                    }
                                },
                                transactions: {
                                    schema: {
                                        title: 'transaction schema',
                                        version: 0,
                                        type: 'object',
                                        primaryKey: 'id',
                                        properties: {
                                            id: { type: 'string' },
                                            type: { type: 'string' },
                                            walletId: { type: 'string' },
                                            hash: { type: 'string' },
                                            from: { type: 'string' },
                                            to: { type: 'string' },
                                            amount: { type: 'string' },
                                            nrnAmount: { type: 'string' },
                                            status: { type: 'string' },
                                            timestamp: { type: 'number' },
                                            blockHeight: { type: 'number' },
                                            gasUsed: { type: 'number' },
                                            memo: { type: 'string' },
                                            category: { type: 'string' }
                                        },
                                        required: ['id', 'type', 'walletId', 'hash', 'from', 'to', 'amount', 'status', 'timestamp', 'category']
                                    }
                                },
                                conversions: {
                                    schema: {
                                        title: 'conversion schema',
                                        version: 0,
                                        type: 'object',
                                        primaryKey: 'id',
                                        properties: {
                                            id: { type: 'string' },
                                            type: { type: 'string' },
                                            walletId: { type: 'string' },
                                            transactionId: { type: 'string' },
                                            usdcAmount: { type: 'string' },
                                            nrnAmount: { type: 'string' },
                                            rate: { type: 'number' },
                                            timestamp: { type: 'number' },
                                            status: { type: 'string' },
                                            targetAddress: { type: 'string' }
                                        },
                                        required: ['id', 'type', 'walletId', 'transactionId', 'usdcAmount', 'nrnAmount', 'rate', 'timestamp', 'status']
                                    }
                                },
                                settings: {
                                    schema: {
                                        title: 'settings schema',
                                        version: 0,
                                        type: 'object',
                                        primaryKey: 'id',
                                        properties: {
                                            id: { type: 'string' },
                                            type: { type: 'string' },
                                            key: { type: 'string' },
                                            value: { type: 'string' },
                                            walletId: { type: 'string' },
                                            autoSync: { type: 'boolean' },
                                            biometricEnabled: { type: 'boolean' },
                                            notificationsEnabled: { type: 'boolean' },
                                            defaultNetwork: { type: 'string' },
                                            preferredCurrency: { type: 'string' },
                                            theme: { type: 'string' },
                                            timestamp: { type: 'number' },
                                            createdAt: { type: 'number' },
                                            updatedAt: { type: 'number' }
                                        },
                                        required: ['id', 'type', 'key', 'value', 'timestamp']
                                    }
                                },
                                graphs: {
                                    schema: {
                                        title: 'graph schema',
                                        version: 0,
                                        type: 'object',
                                        primaryKey: 'id',
                                        properties: {
                                            id: { type: 'string' },
                                            type: { type: 'string' },
                                            userId: { type: 'string' },
                                            nodes: { type: 'array', items: { type: 'object' } },
                                            edges: { type: 'array', items: { type: 'object' } },
                                            metadata: { type: 'object' },
                                            timestamp: { type: 'number' }
                                        },
                                        required: ['id', 'type', 'userId', 'nodes', 'edges', 'metadata', 'timestamp']
                                    }
                                }
                            })];
                    case 1:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    RxDBService.prototype.getDatabase = function () {
        if (!this.db)
            throw new Error('Database not initialized');
        return this.db;
    };
    RxDBService.prototype.isDatabaseInitialized = function () {
        return this.isInitialized;
    };
    RxDBService.prototype.destroy = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.db) return [3 /*break*/, 2];
                        return [4 /*yield*/, this.db.remove()];
                    case 1:
                        _a.sent();
                        this.db = null;
                        this.isInitialized = false;
                        _a.label = 2;
                    case 2: return [2 /*return*/];
                }
            });
        });
    };
    return RxDBService;
}());
exports.RxDBService = RxDBService;
exports.rxdbService = new RxDBService();
