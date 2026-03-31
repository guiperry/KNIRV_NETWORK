import { Permission } from './types';
/**
 * Permission management utilities
 */
export declare class PermissionManager {
    /**
     * Validates if a set of permissions contains a required permission
     */
    static hasPermission(permissions: Permission[], required: Permission): boolean;
    /**
     * Validates if permissions are valid
     */
    static isValidPermissions(permissions: string[]): permissions is Permission[];
    /**
     * Converts string permissions to Permission enum
     */
    static toPermissions(permissions: string[]): Permission[];
    /**
     * Gets all available permissions
     */
    static getAllPermissions(): Permission[];
    /**
     * Checks if a permission level implies another
     */
    static implies(higher: Permission, lower: Permission): boolean;
}
//# sourceMappingURL=permissions.d.ts.map