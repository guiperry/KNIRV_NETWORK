import { TokenManager } from './token_manager';
export const claimsKey = Symbol('claims');
/**
 * HTTP authentication middleware
 */
export class AuthMiddleware {
    constructor(tokenManager) {
        this.tokenManager = tokenManager;
    }
    /**
     * Middleware function to authenticate HTTP requests
     */
    authenticate() {
        return (req, res, next) => {
            const authHeader = req.headers.authorization;
            if (!authHeader) {
                res.status(401).json({ error: 'missing authorization header' });
                return;
            }
            if (authHeader.length < 7 || authHeader.substring(0, 7) !== 'Bearer ') {
                res.status(401).json({ error: 'invalid authorization format' });
                return;
            }
            const tokenString = authHeader.substring(7);
            try {
                const claims = this.tokenManager.validateToken(tokenString);
                req[claimsKey] = claims;
                next();
            }
            catch (error) {
                res.status(401).json({ error: 'invalid token' });
                return;
            }
        };
    }
    /**
     * Middleware to require specific permission
     */
    requirePermission(permission) {
        return (req, res, next) => {
            const claims = this.getClaims(req);
            if (!claims) {
                res.status(401).json({ error: 'authentication required' });
                return;
            }
            if (!this.tokenManager.hasPermission(claims, permission)) {
                res.status(403).json({ error: 'insufficient permissions' });
                return;
            }
            next();
        };
    }
    /**
     * Extract claims from request
     */
    getClaims(req) {
        return req[claimsKey] || null;
    }
    /**
     * Create authentication context from request
     */
    createAuthContext(req) {
        const claims = this.getClaims(req);
        if (!claims)
            return null;
        return {
            claims,
            request: req
        };
    }
}
/**
 * Utility function to get claims from request
 */
export function getClaims(req) {
    return req[claimsKey] || null;
}
/**
 * Utility function to check if current user has permission
 */
export function hasPermission(req, permission) {
    const claims = getClaims(req);
    if (!claims)
        return false;
    const tokenManager = new TokenManager({ secretKey: process.env.JWT_SECRET || 'default-secret' });
    return tokenManager.hasPermission(claims, permission);
}
//# sourceMappingURL=middleware.js.map