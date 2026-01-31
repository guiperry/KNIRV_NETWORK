export enum Permission {
  ReadOnly = 'read',
  ReadWrite = 'write',
  Admin = 'admin'
}

export interface Claims {
  user_id: string;
  wallet_addr: string;
  permissions: Permission[];
  iat?: number;
  exp?: number;
  nbf?: number;
}

export interface TokenManagerOptions {
  secretKey: string;
  tokenDuration?: number; // in seconds, default 1 hour
}

export interface AuthContext {
  claims: Claims;
  request: any; // HTTP request object
}