import { Request, Response, NextFunction } from 'express';
import { TokenManager } from './token_manager';
import { Claims, AuthContext } from './types';

export const claimsKey = Symbol('claims');

/**
 * HTTP authentication middleware
 */
export class AuthMiddleware {
  private tokenManager: TokenManager;

  constructor(tokenManager: TokenManager) {
    this.tokenManager = tokenManager;
  }

  /**
   * Middleware function to authenticate HTTP requests
   */
  authenticate() {
    return (req: Request, res: Response, next: NextFunction): void => {
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
        (req as any)[claimsKey] = claims;
        next();
      } catch (error) {
        res.status(401).json({ error: 'invalid token' });
        return;
      }
    };
  }

  /**
   * Middleware to require specific permission
   */
  requirePermission(permission: string) {
    return (req: Request, res: Response, next: NextFunction): void => {
      const claims = this.getClaims(req);
      
      if (!claims) {
        res.status(401).json({ error: 'authentication required' });
        return;
      }

      if (!this.tokenManager.hasPermission(claims, permission as any)) {
        res.status(403).json({ error: 'insufficient permissions' });
        return;
      }

      next();
    };
  }

  /**
   * Extract claims from request
   */
  getClaims(req: Request): Claims | null {
    return (req as any)[claimsKey] || null;
  }

  /**
   * Create authentication context from request
   */
  createAuthContext(req: Request): AuthContext | null {
    const claims = this.getClaims(req);
    if (!claims) return null;
    
    return {
      claims,
      request: req
    };
  }
}

/**
 * Utility function to get claims from request
 */
export function getClaims(req: Request): Claims | null {
  return (req as any)[claimsKey] || null;
}

/**
 * Utility function to check if current user has permission
 */
export function hasPermission(req: Request, permission: string): boolean {
  const claims = getClaims(req);
  if (!claims) return false;
  
  const tokenManager = new TokenManager({ secretKey: process.env.JWT_SECRET || 'default-secret' });
  return tokenManager.hasPermission(claims, permission as any);
}