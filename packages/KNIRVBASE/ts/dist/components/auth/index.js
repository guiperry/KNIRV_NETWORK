import { TokenManager } from './token_manager';
import { AuthMiddleware } from './middleware';
import { PermissionManager } from './permissions';
import { Permission } from './types';
export { TokenManager, AuthMiddleware, PermissionManager, Permission };
/**
 * Factory function to create a complete authentication system
 */
export function createAuthSystem(secretKey, tokenDuration) {
    const tokenManager = new TokenManager({ secretKey, tokenDuration });
    const authMiddleware = new AuthMiddleware(tokenManager);
    return {
        tokenManager,
        authMiddleware,
        permissionManager: PermissionManager
    };
}
/**
 * Default authentication system instance
 */
export const defaultAuthSystem = createAuthSystem(process.env.JWT_SECRET || 'default-secret-key', parseInt(process.env.JWT_DURATION || '3600', 10));
//# sourceMappingURL=index.js.map