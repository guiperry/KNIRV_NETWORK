import jwt from 'jsonwebtoken';
import { Permission, Claims, TokenManagerOptions } from './types';

export class TokenManager {
  private secretKey: string;
  private tokenDuration: number;

  constructor(options: TokenManagerOptions) {
    this.secretKey = options.secretKey;
    this.tokenDuration = options.tokenDuration || 3600; // 1 hour default
  }

  /**
   * Creates a new JWT token
   */
  generateToken(
    userId: string,
    walletAddr: string,
    permissions: Permission[]
  ): string {
    const now = Math.floor(Date.now() / 1000);
    
    const claims: Claims = {
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
  validateToken(tokenString: string): Claims {
    try {
      const decoded = jwt.verify(tokenString, this.secretKey, { algorithms: ['HS256'] }) as Claims;
      return decoded;
    } catch (error) {
      throw new Error(`Failed to validate token: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Generates a new token with extended expiration
   */
  refreshToken(oldToken: string): string {
    const claims = this.validateToken(oldToken);
    return this.generateToken(claims.user_id, claims.wallet_addr, claims.permissions);
  }

  /**
   * Checks if claims contain required permission
   */
  hasPermission(claims: Claims, required: Permission): boolean {
    return claims.permissions.some(p => p === required || p === Permission.Admin);
  }
}