import { Request, Response, NextFunction } from 'express';
import { TokenManager } from './token_manager';
import { Claims, AuthContext } from './types';
export declare const claimsKey: unique symbol;
/**
 * HTTP authentication middleware
 */
export declare class AuthMiddleware {
    private tokenManager;
    constructor(tokenManager: TokenManager);
    /**
     * Middleware function to authenticate HTTP requests
     */
    authenticate(): (req: Request, res: Response, next: NextFunction) => void;
    /**
     * Middleware to require specific permission
     */
    requirePermission(permission: string): (req: Request, res: Response, next: NextFunction) => void;
    /**
     * Extract claims from request
     */
    getClaims(req: Request): Claims | null;
    /**
     * Create authentication context from request
     */
    createAuthContext(req: Request): AuthContext | null;
}
/**
 * Utility function to get claims from request
 */
export declare function getClaims(req: Request): Claims | null;
/**
 * Utility function to check if current user has permission
 */
export declare function hasPermission(req: Request, permission: string): boolean;
//# sourceMappingURL=middleware.d.ts.map