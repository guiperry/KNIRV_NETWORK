import { PermissionManager } from '../permissions';

describe('PermissionManager', () => {
  describe('hasPermission', () => {
    it('should return true for existing permission', () => {
      const permissions = ['read', 'write'];
      expect(PermissionManager.hasPermission(permissions as any, 'read' as any)).toBe(true);
      expect(PermissionManager.hasPermission(permissions as any, 'write' as any)).toBe(true);
    });

    it('should return false for non-existing permission', () => {
      const permissions = ['read'];
      expect(PermissionManager.hasPermission(permissions as any, 'admin' as any)).toBe(false);
    });

    it('should return true for admin permission for any request', () => {
      const adminPermissions = ['admin'];
      expect(PermissionManager.hasPermission(adminPermissions as any, 'read' as any)).toBe(true);
      expect(PermissionManager.hasPermission(adminPermissions as any, 'write' as any)).toBe(true);
      expect(PermissionManager.hasPermission(adminPermissions as any, 'admin' as any)).toBe(true);
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
      expect(PermissionManager.implies('admin' as any, 'read' as any)).toBe(true);
      expect(PermissionManager.implies('admin' as any, 'write' as any)).toBe(true);
      expect(PermissionManager.implies('admin' as any, 'admin' as any)).toBe(true);
      
      expect(PermissionManager.implies('write' as any, 'read' as any)).toBe(true);
      expect(PermissionManager.implies('write' as any, 'write' as any)).toBe(true);
      expect(PermissionManager.implies('write' as any, 'admin' as any)).toBe(false);
      
      expect(PermissionManager.implies('read' as any, 'read' as any)).toBe(true);
      expect(PermissionManager.implies('read' as any, 'write' as any)).toBe(false);
      expect(PermissionManager.implies('read' as any, 'admin' as any)).toBe(false);
    });
  });
});