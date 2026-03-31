import { Permission } from './types';
/**
 * Permission management utilities
 */
export class PermissionManager {
    /**
     * Validates if a set of permissions contains a required permission
     */
    static hasPermission(permissions, required) {
        return permissions.some(p => p === required || p === Permission.Admin);
    }
    /**
     * Validates if permissions are valid
     */
    static isValidPermissions(permissions) {
        return permissions.every(p => Object.values(Permission).includes(p));
    }
    /**
     * Converts string permissions to Permission enum
     */
    static toPermissions(permissions) {
        return permissions.filter(p => Object.values(Permission).includes(p));
    }
    /**
     * Gets all available permissions
     */
    static getAllPermissions() {
        return Object.values(Permission);
    }
    /**
     * Checks if a permission level implies another
     */
    static implies(higher, lower) {
        if (higher === Permission.Admin)
            return true;
        if (higher === Permission.ReadWrite && lower !== Permission.Admin)
            return true;
        if (higher === Permission.ReadOnly && lower === Permission.ReadOnly)
            return true;
        return false;
    }
}
//# sourceMappingURL=permissions.js.map