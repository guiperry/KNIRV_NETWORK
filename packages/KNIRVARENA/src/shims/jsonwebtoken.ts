/**
 * Browser shim for 'jsonwebtoken'.
 * knirvbase-ts exports its auth module (TokenManager etc.) which imports jsonwebtoken,
 * but the arena never calls any JWT functions in the browser.
 * These stubs prevent the Node.js stream-based internals from crashing at module init.
 */

export function sign(..._args: unknown[]): never {
  throw new Error('jsonwebtoken.sign is not available in browser environments');
}

export function verify(..._args: unknown[]): never {
  throw new Error('jsonwebtoken.verify is not available in browser environments');
}

export function decode(..._args: unknown[]): null {
  return null;
}

export const JsonWebTokenError = class JsonWebTokenError extends Error {};
export const TokenExpiredError = class TokenExpiredError extends Error {};
export const NotBeforeError = class NotBeforeError extends Error {};

export default { sign, verify, decode, JsonWebTokenError, TokenExpiredError, NotBeforeError };
