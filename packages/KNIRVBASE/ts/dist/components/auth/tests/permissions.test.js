import { PermissionManager } from '../permissions';
describe('PermissionManager', () => {
    describe('hasPermission', () => {
        it('should return true for existing permission', () => {
            const permissions = ['read', 'write'];
            expect(PermissionManager.hasPermission(permissions, 'read')).toBe(true);
            expect(PermissionManager.hasPermission(permissions, 'write')).toBe(true);
        });
        it('should return false for non-existing permission', () => {
            const permissions = ['read'];
            expect(PermissionManager.hasPermission(permissions, 'admin')).toBe(false);
        });
        it('should return true for admin permission for any request', () => {
            const adminPermissions = ['admin'];
            expect(PermissionManager.hasPermission(adminPermissions, 'read')).toBe(true);
            expect(PermissionManager.hasPermission(adminPermissions, 'write')).toBe(true);
            expect(PermissionManager.hasPermission(adminPermissions, 'admin')).toBe(true);
        });
    });
    describe('isValidPermissions', () => {
        it('should return true for valid permissions', () => {
            const permissions = ['read', 'write', 'admin'];
            expect(PermissionManager.isValidPermissions(permissions)).toBe(true);
        });
        it('should return false for invalid permissions', () => {
            const permissions = ['read', 'invalid'];
            expect(PermissionManager.isValidPermissions(permissions)).toBe(false);
        });
    });
    describe('toPermissions', () => {
        it('should filter and convert valid permissions', () => {
            const permissions = ['read', 'invalid', 'write'];
            const result = PermissionManager.toPermissions(permissions);
            expect(result).toEqual(['read', 'write']);
        });
    });
    describe('getAllPermissions', () => {
        it('should return all available permissions', () => {
            const all = PermissionManager.getAllPermissions();
            expect(all).toContain('read');
            expect(all).toContain('write');
            expect(all).toContain('admin');
        });
    });
    describe('implies', () => {
        it('should check permission implications correctly', () => {
            expect(PermissionManager.implies('admin', 'read')).toBe(true);
            expect(PermissionManager.implies('admin', 'write')).toBe(true);
            expect(PermissionManager.implies('admin', 'admin')).toBe(true);
            expect(PermissionManager.implies('write', 'read')).toBe(true);
            expect(PermissionManager.implies('write', 'write')).toBe(true);
            expect(PermissionManager.implies('write', 'admin')).toBe(false);
            expect(PermissionManager.implies('read', 'read')).toBe(true);
            expect(PermissionManager.implies('read', 'write')).toBe(false);
            expect(PermissionManager.implies('read', 'admin')).toBe(false);
        });
    });
});
//# sourceMappingURL=permissions.test.js.map