import { TokenManager } from './token_manager';
import { AuthMiddleware } from './middleware';
import { PermissionManager } from './permissions';
import { Permission, Claims, TokenManagerOptions, AuthContext } from './types';
export { TokenManager, AuthMiddleware, PermissionManager, Permission, Claims, TokenManagerOptions, AuthContext };
/**
 * Factory function to create a complete authentication system
 */
export declare function createAuthSystem(secretKey: string, tokenDuration?: number): {
    tokenManager: TokenManager;
    authMiddleware: AuthMiddleware;
    permissionManager: typeof PermissionManager;
};
/**
 * Default authentication system instance
 */
export declare const defaultAuthSystem: {
    tokenManager: TokenManager;
    authMiddleware: AuthMiddleware;
    permissionManager: typeof PermissionManager;
};
//# sourceMappingURL=index.d.ts.map