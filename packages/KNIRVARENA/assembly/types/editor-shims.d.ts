// Editor-only type shims for AssemblyScript primitives and builtins.
// This file is consumed by the TypeScript language service for IDE tooling
// and is ignored by the AssemblyScript compiler (asc).

declare type bool = boolean;
declare type i32 = number;
declare type f32 = number;
declare type usize = number;

// AssemblyScript builtin helper
declare function changetype<T>(value: any): T;
