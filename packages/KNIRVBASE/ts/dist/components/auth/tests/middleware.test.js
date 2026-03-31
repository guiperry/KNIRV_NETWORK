import { AuthMiddleware } from '../middleware';
import { TokenManager } from '../token_manager';
import { Permission } from '../types';
describe('AuthMiddleware', () => {
    let authMiddleware;
    let tokenManager;
    const testSecret = 'test-secret-key';
    beforeEach(() => {
        tokenManager = new TokenManager({ secretKey: testSecret });
        authMiddleware = new AuthMiddleware(tokenManager);
    });
    const createMockRequest = (authHeader) => ({
        headers: authHeader ? { authorization: authHeader } : {}
    });
    const createMockResponse = () => {
        const res = {};
        res.status = jest.fn().mockReturnValue(res);
        res.json = jest.fn().mockReturnValue(res);
        return res;
    };
    describe('constructor', () => {
        it('should create AuthMiddleware with token manager', () => {
            expect(authMiddleware).toBeInstanceOf(AuthMiddleware);
        });
    });
    describe('authenticate middleware', () => {
        it('should pass with valid Bearer token', (done) => {
            const token = tokenManager.generateToken('user123', 'wallet456', [Permission.ReadOnly]);
            const req = createMockRequest(`Bearer ${token}`);
            const res = createMockResponse();
            const next = jest.fn();
            authMiddleware.authenticate()(req, res, next);
            expect(next).toHaveBeenCalled();
            expect(res.status).not.toHaveBeenCalled();
            done();
        });
        it('should reject missing authorization header', (done) => {
            const req = createMockRequest();
            const res = createMockResponse();
            const next = jest.fn();
            authMiddleware.authenticate()(req, res, next);
            expect(next).not.toHaveBeenCalled();
            expect(res.status).toHaveBeenCalledWith(401);
            expect(res.json).toHaveBeenCalledWith({ error: 'missing authorization header' });
            done();
        });
        it('should reject invalid authorization format', (done) => {
            const req = createMockRequest('InvalidFormat token');
            const res = createMockResponse();
            const next = jest.fn();
            authMiddleware.authenticate()(req, res, next);
            expect(next).not.toHaveBeenCalled();
            expect(res.status).toHaveBeenCalledWith(401);
            expect(res.json).toHaveBeenCalledWith({ error: 'invalid authorization format' });
            done();
        });
        it('should reject invalid token', (done) => {
            const req = createMockRequest('Bearer invalid-token');
            const res = createMockResponse();
            const next = jest.fn();
            authMiddleware.authenticate()(req, res, next);
            expect(next).not.toHaveBeenCalled();
            expect(res.status).toHaveBeenCalledWith(401);
            expect(res.json).toHaveBeenCalledWith({ error: 'invalid token' });
            done();
        });
    });
    describe('requirePermission middleware', () => {
        it('should pass with required permission', (done) => {
            const token = tokenManager.generateToken('user123', 'wallet456', [Permission.ReadWrite]);
            const req = createMockRequest(`Bearer ${token}`);
            const res = createMockResponse();
            const next = jest.fn();
            // First authenticate
            authMiddleware.authenticate()(req, res, () => {
                // Then check permission
                authMiddleware.requirePermission('write')(req, res, next);
            });
            expect(next).toHaveBeenCalled();
            done();
        });
        it('should reject without required permission', (done) => {
            const token = tokenManager.generateToken('user123', 'wallet456', [Permission.ReadOnly]);
            const req = createMockRequest(`Bearer ${token}`);
            const res = createMockResponse();
            const next = jest.fn();
            // First authenticate
            authMiddleware.authenticate()(req, res, () => {
                // Then check permission
                authMiddleware.requirePermission('write')(req, res, next);
            });
            expect(next).not.toHaveBeenCalled();
            expect(res.status).toHaveBeenCalledWith(403);
            expect(res.json).toHaveBeenCalledWith({ error: 'insufficient permissions' });
            done();
        });
        it('should reject unauthenticated request', (done) => {
            const req = createMockRequest();
            const res = createMockResponse();
            const next = jest.fn();
            authMiddleware.requirePermission('read')(req, res, next);
            expect(next).not.toHaveBeenCalled();
            expect(res.status).toHaveBeenCalledWith(401);
            expect(res.json).toHaveBeenCalledWith({ error: 'authentication required' });
            done();
        });
    });
    describe('getClaims', () => {
        it('should return claims from authenticated request', () => {
            const token = tokenManager.generateToken('user123', 'wallet456', [Permission.ReadOnly]);
            const req = createMockRequest(`Bearer ${token}`);
            const res = createMockResponse();
            const next = jest.fn();
            authMiddleware.authenticate()(req, res, () => {
                const claims = authMiddleware.getClaims(req);
                expect(claims).toBeTruthy();
                expect(claims.user_id).toBe('user123');
            });
        });
        it('should return null for unauthenticated request', () => {
            const req = createMockRequest();
            const claims = authMiddleware.getClaims(req);
            expect(claims).toBeNull();
        });
    });
    describe('createAuthContext', () => {
        it('should create auth context for authenticated request', () => {
            const token = tokenManager.generateToken('user123', 'wallet456', [Permission.ReadOnly]);
            const req = createMockRequest(`Bearer ${token}`);
            const res = createMockResponse();
            const next = jest.fn();
            authMiddleware.authenticate()(req, res, () => {
                const context = authMiddleware.createAuthContext(req);
                expect(context).toBeTruthy();
                expect(context.claims.user_id).toBe('user123');
                expect(context.request).toBe(req);
            });
        });
        it('should return null for unauthenticated request', () => {
            const req = createMockRequest();
            const context = authMiddleware.createAuthContext(req);
            expect(context).toBeNull();
        });
    });
});
//# sourceMappingURL=middleware.test.js.map