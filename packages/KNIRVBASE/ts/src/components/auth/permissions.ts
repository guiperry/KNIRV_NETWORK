import { Permission } from './types';

/**
 * Permission management utilities
 */
export class PermissionManager {
  /**
   * Validates if a set of permissions contains a required permission
   */
  static hasPermission(permissions: Permission[], required: Permission): boolean {
    return permissions.some(p => p === required || p === Permission.Admin);
  }

  /**
   * Validates if permissions are valid
   */
  static isValidPermissions(permissions: string[]): permissions is Permission[] {
    return permissions.every(p => Object.values(Permission).includes(p as Permission));
  }

  /**
   * Converts string permissions to Permission enum
   */
  static toPermissions(permissions: string[]): Permission[] {
    return permissions.filter(p => Object.values(Permission).includes(p as Permission)) as Permission[];
  }

  /**
   * Gets all available permissions
   */
  static getAllPermissions(): Permission[] {
    return Object.values(Permission);
  }

  /**
   * Checks if a permission level implies another
   */
  static implies(higher: Permission, lower: Permission): boolean {
    if (higher === Permission.Admin) return true;
    if (higher === Permission.ReadWrite && lower !== Permission.Admin) return true;
    if (higher === Permission.ReadOnly && lower === Permission.ReadOnly) return true;
    return false;
  }
}