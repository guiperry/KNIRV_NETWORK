import { TokenManager } from '../token_manager';
import { Permission } from '../types';
describe('TokenManager', () => {
    const testSecret = 'test-secret-key';
    let tokenManager;
    beforeEach(() => {
        tokenManager = new TokenManager({ secretKey: testSecret, tokenDuration: 3600 });
    });
    describe('constructor', () => {
        it('should create TokenManager with provided options', () => {
            const tm = new TokenManager({ secretKey: testSecret });
            expect(tm).toBeInstanceOf(TokenManager);
        });
        it('should use default token duration when not provided', () => {
            const tm = new TokenManager({ secretKey: testSecret });
            const token = tm.generateToken('user1', 'wallet1', [Permission.ReadOnly]);
            const claims = tm.validateToken(token);
            const now = Math.floor(Date.now() / 1000);
            expect(claims.exp).toBeGreaterThan(now);
            expect(claims.exp).toBeLessThanOrEqual(now + 3600);
        });
    });
    describe('generateToken', () => {
        it('should generate a valid JWT token', () => {
            const token = tokenManager.generateToken('user123', 'wallet456', [Permission.ReadOnly, Permission.ReadWrite]);
            expect(token).toBeTruthy();
            expect(typeof token).toBe('string');
        });
        it('should generate tokens with different content', () => {
            const token1 = tokenManager.generateToken('user1', 'wallet1', [Permission.ReadOnly]);
            const token2 = tokenManager.generateToken('user2', 'wallet2', [Permission.ReadWrite]);
            expect(token1).not.toBe(token2);
        });
    });
    describe('validateToken', () => {
        it('should validate a valid token and return correct claims', () => {
            const permissions = [Permission.ReadOnly, Permission.ReadWrite];
            const token = tokenManager.generateToken('user123', 'wallet456', permissions);
            const claims = tokenManager.validateToken(token);
            expect(claims.user_id).toBe('user123');
            expect(claims.wallet_addr).toBe('wallet456');
            expect(claims.permissions).toEqual(permissions);
            expect(claims.iat).toBeTruthy();
            expect(claims.exp).toBeTruthy();
            expect(claims.nbf).toBeTruthy();
        });
        it('should throw error for invalid token', () => {
            expect(() => {
                tokenManager.validateToken('invalid-token');
            }).toThrow('Failed to validate token');
        });
        it('should throw error for token with wrong secret', () => {
            const wrongManager = new TokenManager({ secretKey: 'wrong-secret' });
            const token = tokenManager.generateToken('user123', 'wallet456', [Permission.ReadOnly]);
            expect(() => {
                wrongManager.validateToken(token);
            }).toThrow('Failed to validate token');
        });
    });
    describe('refreshToken', () => {
        it('should refresh a valid token', () => {
            const permissions = [Permission.ReadOnly];
            const oldToken = tokenManager.generateToken('user123', 'wallet456', permissions);
            const newToken = tokenManager.refreshToken(oldToken);
            expect(newToken).toBeTruthy();
            expect(typeof newToken).toBe('string');
            const claims = tokenManager.validateToken(newToken);
            expect(claims.user_id).toBe('user123');
            expect(claims.wallet_addr).toBe('wallet456');
            expect(claims.permissions).toEqual(permissions);
        });
        it('should throw error when refreshing invalid token', () => {
            expect(() => {
                tokenManager.refreshToken('invalid-token');
            }).toThrow('Failed to validate token');
        });
    });
    describe('hasPermission', () => {
        it('should return true for existing permission', () => {
            const claims = {
                user_id: 'user1',
                wallet_addr: 'wallet1',
                permissions: [Permission.ReadOnly, Permission.ReadWrite]
            };
            expect(tokenManager.hasPermission(claims, Permission.ReadOnly)).toBe(true);
            expect(tokenManager.hasPermission(claims, Permission.ReadWrite)).toBe(true);
        });
        it('should return false for non-existing permission', () => {
            const claims = {
                user_id: 'user1',
                wallet_addr: 'wallet1',
                permissions: [Permission.ReadOnly]
            };
            expect(tokenManager.hasPermission(claims, Permission.Admin)).toBe(false);
        });
        it('should return true for admin permission for any request', () => {
            const adminClaims = {
                user_id: 'admin',
                wallet_addr: 'admin-wallet',
                permissions: [Permission.Admin]
            };
            expect(tokenManager.hasPermission(adminClaims, Permission.ReadOnly)).toBe(true);
            expect(tokenManager.hasPermission(adminClaims, Permission.ReadWrite)).toBe(true);
            expect(tokenManager.hasPermission(adminClaims, Permission.Admin)).toBe(true);
        });
    });
});
//# sourceMappingURL=token_manager.test.js.map