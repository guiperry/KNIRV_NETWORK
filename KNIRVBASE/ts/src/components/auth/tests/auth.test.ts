import { createAuthSystem, defaultAuthSystem } from '../index';
import { TokenManager, AuthMiddleware, PermissionManager } from '../index';

describe('Auth System', () => {
  describe('createAuthSystem', () => {
    it('should create a complete authentication system', () => {
      const authSystem = createAuthSystem('test-secret', 7200);
      
      expect(authSystem.tokenManager).toBeInstanceOf(TokenManager);
      expect(authSystem.authMiddleware).toBeInstanceOf(AuthMiddleware);
      expect(authSystem.permissionManager).toBe(PermissionManager);
    });

    it('should use provided secret and duration', () => {
      const authSystem = createAuthSystem('custom-secret', 1800);
      
      const token = authSystem.tokenManager.generateToken('user1', 'wallet1', ['read' as any]);
      const claims = authSystem.tokenManager.validateToken(token);
      
      const now = Math.floor(Date.now() / 1000);
      expect(claims.exp).toBeLessThanOrEqual(now + 1800);
    });
  });

  describe('defaultAuthSystem', () => {
    it('should create a default authentication system', () => {
      expect(defaultAuthSystem.tokenManager).toBeInstanceOf(TokenManager);
      expect(defaultAuthSystem.authMiddleware).toBeInstanceOf(AuthMiddleware);
      expect(defaultAuthSystem.permissionManager).toBe(PermissionManager);
    });
  });
});