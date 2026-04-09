import jwt from 'jsonwebtoken';
import { Permission } from './types';
export class TokenManager {
    constructor(options) {
        this.secretKey = options.secretKey;
        this.tokenDuration = options.tokenDuration || 3600; // 1 hour default
    }
    /**
     * Creates a new JWT token
     */
    generateToken(userId, walletAddr, permissions) {
        const now = Math.floor(Date.now() / 1000);
        const claims = {
            user_id: userId,
            wallet_addr: walletAddr,
            permissions: permissions,
            iat: now,
            exp: now + this.tokenDuration,
            nbf: now
        };
        return jwt.sign(claims, this.secretKey, { algorithm: 'HS256' });
    }
    /**
     * Verifies and parses a JWT token
     */
    validateToken(tokenString) {
        try {
            const decoded = jwt.verify(tokenString, this.secretKey, { algorithms: ['HS256'] });
            return decoded;
        }
        catch (error) {
            throw new Error(`Failed to validate token: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }
    }
    /**
     * Generates a new token with extended expiration
     */
    refreshToken(oldToken) {
        const claims = this.validateToken(oldToken);
        return this.generateToken(claims.user_id, claims.wallet_addr, claims.permissions);
    }
    /**
     * Checks if claims contain required permission
     */
    hasPermission(claims, required) {
        return claims.permissions.some(p => p === required || p === Permission.Admin);
    }
    /**
     * Generates a hybrid token that embeds a session token (from external auth provider)
     * alongside JWT claims. Enables hybrid auth where blockchain wallet session is
     * validated alongside standard JWT.
     */
    generateHybridToken(userId, walletAddr, sessionToken, permissions) {
        const now = Math.floor(Date.now() / 1000);
        const claims = {
            user_id: userId,
            wallet_addr: walletAddr,
            permissions: permissions,
            session_token: sessionToken,
            iat: now,
            exp: now + this.tokenDuration,
            nbf: now
        };
        return jwt.sign(claims, this.secretKey, { algorithm: 'HS256' });
    }
}
//# sourceMappingURL=token_manager.js.map