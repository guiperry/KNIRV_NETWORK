/**
 * TypeScript types for KNIRV Gateway SDK
 */
export function defaultClientOptions() {
    return {
        baseURL: process.env.KNIRVGATEWAY_BASE_URL ||
            process.env.GATEWAY_SERVICE_URL ||
            'http://localhost:8000',
        economicsURL: process.env.ECONOMICS_SERVICE_URL ||
            'http://localhost:8090',
        apiKey: process.env.KNIRVGATEWAY_API_KEY || '',
        nrnContract: process.env.NRN_CONTRACT || '',
        timeout: 30000,
        retries: 3,
        retryDelay: 1000,
        environment: process.env.NODE_ENV || 'development',
        debug: process.env.KNIRV_DEBUG === 'true',
        verbose: process.env.KNIRV_VERBOSE === 'true',
        serviceURLs: {
            knirvchain: process.env.KNIRVCHAIN_URL || 'http://localhost:8080',
            knirvserver: process.env.KNIRVNEXUS_URL || 'http://localhost:8081',
            knirvoracle: process.env.KNIRVORACLE_URL || 'http://localhost:8082',
            knirvgraph: process.env.KNIRVGRAPH_URL || 'http://localhost:8083',
        },
    };
}
// Error Types
export class KNIRVGatewayError extends Error {
    constructor(message, statusCode, response) {
        super(message);
        this.statusCode = statusCode;
        this.response = response;
        this.name = 'KNIRVGatewayError';
    }
}
export class EconomicsServiceError extends KNIRVGatewayError {
    constructor(message, statusCode, response) {
        super(message, statusCode, response);
        this.name = 'EconomicsServiceError';
    }
}
export class GatewayServiceError extends KNIRVGatewayError {
    constructor(message, statusCode, response) {
        super(message, statusCode, response);
        this.name = 'GatewayServiceError';
    }
}
// Configuration Validation
export function validateClientOptions(options) {
    const errors = [];
    if (options.baseURL && !isValidURL(options.baseURL)) {
        errors.push('Invalid baseURL format');
    }
    if (options.economicsURL && !isValidURL(options.economicsURL)) {
        errors.push('Invalid economicsURL format');
    }
    if (options.timeout && (options.timeout < 0 || options.timeout > 300000)) {
        errors.push('Timeout must be between 0 and 300000ms');
    }
    if (options.retries && (options.retries < 0 || options.retries > 10)) {
        errors.push('Retries must be between 0 and 10');
    }
    return errors;
}
function isValidURL(url) {
    try {
        new URL(url);
        return true;
    }
    catch {
        return false;
    }
}
