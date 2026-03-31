import { Permission, Claims, TokenManagerOptions } from './types';
export declare class TokenManager {
    private secretKey;
    private tokenDuration;
    constructor(options: TokenManagerOptions);
    /**
     * Creates a new JWT token
     */
    generateToken(userId: string, walletAddr: string, permissions: Permission[]): string;
    /**
     * Verifies and parses a JWT token
     */
    validateToken(tokenString: string): Claims;
    /**
     * Generates a new token with extended expiration
     */
    refreshToken(oldToken: string): string;
    /**
     * Checks if claims contain required permission
     */
    hasPermission(claims: Claims, required: Permission): boolean;
}
//# sourceMappingURL=token_manager.d.ts.map