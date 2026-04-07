export declare enum Permission {
    ReadOnly = "read",
    ReadWrite = "write",
    Admin = "admin"
}
export interface Claims {
    user_id: string;
    wallet_addr: string;
    permissions: Permission[];
    session_token?: string;
    iat?: number;
    exp?: number;
    nbf?: number;
}
export interface TokenManagerOptions {
    secretKey: string;
    tokenDuration?: number;
}
export interface AuthContext {
    claims: Claims;
    request: unknown;
}
//# sourceMappingURL=types.d.ts.map