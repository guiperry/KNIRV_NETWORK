// KNIRV Controller Relay WASM Module
// AssemblyScript — stable ABI v1 exports.
// Heavy validation logic lives in the bundled TypeScript adapter.
// This module provides deterministic canonicalization, digest placeholder,
// and sequence replay protection inside the WASM sandbox.

type bool = boolean;
type i32 = number;
type f32 = number;
type usize = number;

// ─── Module identity ────────────────────────────────────────────────────

export function abi_version(): i32 {
  return 1;
}

export function module_kind(): i32 {
  return 3;
}

// ─── Memory helpers ─────────────────────────────────────────────────────

const allocated: Uint8Array[] = [];

export function allocate(length: usize): usize {
  const bytes = new Uint8Array(<i32>length);
  allocated.push(bytes);
  return changetype<usize>(bytes);
}

export function deallocate(_ptr: usize, _length: usize): void {
}

export function zeroize(): void {
  for (let i = 0; i < <i32>allocated.length; i++) {
    const bytes = allocated[i];
    for (let j = 0; j < <i32>bytes.length; j++) {
      bytes[j] = 0;
    }
  }
  allocated.length = 0;
}

// ─── Internal state ─────────────────────────────────────────────────────
// Monotonic sequence counters per (user, device, dve) tuple.

const sequenceMap = new Map<string, i32>();

// ─── Self-test ──────────────────────────────────────────────────────────

export function self_test(_input_pointer: usize, _input_length: usize): i32 {
  if (abi_version() != 1 || module_kind() != 3) {
    return -1;
  }
  return __put_result(__to_utf8("{\"status\":\"ok\"}"));
}

// ─── Result handles ─────────────────────────────────────────────────────

const resultPool: Uint8Array[] = [];

export function result_pointer(handle: i32): usize {
  if (handle < 0 || handle >= <i32>resultPool.length) return 0;
  return changetype<usize>(resultPool[<i32>handle]);
}

export function result_length(handle: i32): usize {
  if (handle < 0 || handle >= <i32>resultPool.length) return 0;
  return resultPool[<i32>handle].length;
}

export function free_result(handle: i32): void {
  if (handle >= 0 && handle < <i32>resultPool.length) {
    resultPool[<i32>handle] = new Uint8Array(0);
  }
}

// ─── Canonical relay operations ─────────────────────────────────────────

export function invoke(
  operation_pointer: usize,
  operation_length: usize,
  input_pointer: usize,
  input_length: usize,
): i32 {
  const opBytes = __read_bytes(operation_pointer, operation_length);
  const op = String.UTF8.decode(opBytes);

  if (op == "validate_envelope") {
    return __validate_envelope(input_pointer, input_length);
  }
  if (op == "canonicalize") {
    return __canonicalize(input_pointer, input_length);
  }
  if (op == "compute_digest") {
    return __compute_digest(input_pointer, input_length);
  }
  if (op == "check_sequence") {
    return __check_sequence(input_pointer, input_length);
  }

  return __put_result(__to_utf8("{\"error\":\"unknown_operation\"}"));
}

// ─── validate_envelope ──────────────────────────────────────────────────
// Validates a relay envelope's structural and temporal fields.
// Input: canonical JSON bytes.
// Output: JSON result handle.

function __validate_envelope(ptr: usize, len: usize): i32 {
  const raw = __read_bytes(ptr, len);
  const json = String.UTF8.decode(raw);

  let requestId = "";
  let userSubject = "";
  let deviceId = "";
  let dveId = "";
  let targetType = "";
  let targetId = "";
  let capability = "";
  let sequence: i32 = 0;
  let issuedAtUnix: i32 = 0;
  let expiresAtUnix: i32 = 0;
  let payloadDigest = "";

  const pairs = __json_kv_pairs(json);
  for (let i = 0; i < <i32>pairs.length; i += 2) {
    const key = pairs[i];
    const value = pairs[i + 1];
    if (key == "request_id") requestId = value;
    else if (key == "user_subject") userSubject = value;
    else if (key == "device_id") deviceId = value;
    else if (key == "dve_id") dveId = value;
    else if (key == "target_type") targetType = value;
    else if (key == "target_id") targetId = value;
    else if (key == "capability") capability = value;
    else if (key == "sequence") sequence = __parse_i32(value);
    else if (key == "issued_at_unix") issuedAtUnix = __parse_i32(value);
    else if (key == "expires_at_unix") expiresAtUnix = __parse_i32(value);
    else if (key == "payload_digest") payloadDigest = value;
  }

  if (requestId.length == 0 || userSubject.length == 0) {
    return __put_result(__to_utf8("{\"valid\":false,\"reason\":\"missing_identity\"}"));
  }
  if (targetType != "dve_expert_advisor" && targetType != "cli_supervisor") {
    return __put_result(__to_utf8("{\"valid\":false,\"reason\":\"invalid_target_type\"}"));
  }
  if (targetId.length == 0) {
    return __put_result(__to_utf8("{\"valid\":false,\"reason\":\"missing_target_id\"}"));
  }
  if (capability.length == 0) {
    return __put_result(__to_utf8("{\"valid\":false,\"reason\":\"missing_capability\"}"));
  }

  const now = <i32>Math.floor(Date.now() / 1000);
  if (now < issuedAtUnix - 60 || now > expiresAtUnix) {
    return __put_result(__to_utf8("{\"valid\":false,\"reason\":\"expired\"}"));
  }

  const seqKey = userSubject + ":" + deviceId + ":" + dveId;
  const lastSeq = sequenceMap.get(seqKey) || 0;
  if (sequence <= lastSeq) {
    return __put_result(__to_utf8("{\"valid\":false,\"reason\":\"sequence_replay\"}"));
  }
  sequenceMap.set(seqKey, sequence);

  if (payloadDigest.length != 71 || !payloadDigest.startsWith("sha256:")) {
    return __put_result(__to_utf8("{\"valid\":false,\"reason\":\"invalid_digest_format\"}"));
  }

  return __put_result(__to_utf8("{\"valid\":true}"));
}

// ─── canonicalize ───────────────────────────────────────────────────────

function __canonicalize(ptr: usize, len: usize): i32 {
  const raw = __read_bytes(ptr, len);
  const json = String.UTF8.decode(raw);
  const canonical = __canonical_json(json);
  return __put_result(__to_utf8(canonical));
}

function __canonical_json(input: string): string {
  const pairs = __json_kv_pairs(input);
  const entries: string[] = [];
  for (let i = 0; i < <i32>pairs.length; i += 2) {
    const k = pairs[i];
    const v = pairs[i + 1];
    entries.push("\"" + k + "\":" + v);
  }
  return "{" + entries.join(",") + "}";
}

// ─── compute_digest ─────────────────────────────────────────────────────
// Adapter computes SHA-256 via Web Crypto. WASM marks operation accepted.

function __compute_digest(_ptr: usize, _len: usize): i32 {
  return __put_result(__to_utf8("{\"delegated\":true}"));
}

// ─── check_sequence ─────────────────────────────────────────────────────

function __check_sequence(ptr: usize, len: usize): i32 {
  const raw = __read_bytes(ptr, len);
  const json = String.UTF8.decode(raw);
  const pairs = __json_kv_pairs(json);
  let user = "";
  let device = "";
  let dve = "";
  let seq: i32 = 0;
  for (let i = 0; i < <i32>pairs.length; i += 2) {
    const key = pairs[i];
    const value = pairs[i + 1];
    if (key == "user_subject") user = value;
    else if (key == "device_id") device = value;
    else if (key == "dve_id") dve = value;
    else if (key == "sequence") seq = __parse_i32(value);
  }
  const key = user + ":" + device + ":" + dve;
  const last = sequenceMap.get(key) || 0;
  const accepted = seq > last ? 1 : 0;
  if (accepted) sequenceMap.set(key, seq);
  return __put_result(__to_utf8("{\"accepted\":" + accepted.toString() + "}"));
}

// ─── Minimal JSON key/value parser ──────────────────────────────────────
// Returns flat array: [key1, value1, key2, value2, ...]

class JsonSlice {
  str: string;
  end: usize;
  constructor(str: string, end: usize) {
    this.str = str;
    this.end = end;
  }
}

function __json_kv_pairs(json: string): string[] {
  const out: string[] = [];
  let i = 0;
  while (i < json.length) {
    while (i < json.length && json.charCodeAt(i) <= 32) i++;
    if (i >= json.length) break;
    if (json.charCodeAt(i) == 0x22) {
      const key = __json_string(json, i);
      out.push(key.str);
      i = key.end;
      while (i < json.length && json.charCodeAt(i) != 0x3A) i++;
      i++;
      const val = __json_value(json, i);
      out.push(val.str);
      i = val.end;
    } else {
      break;
    }
  }
  return out;
}

function __json_string(s: string, start: usize): JsonSlice {
  let i = start + 1;
  const parts: string[] = [];
  while (i < s.length && s.charCodeAt(i) != 0x22) {
    if (s.charCodeAt(i) == 0x5C) {
      i++;
      if (i < s.length) parts.push(s.charAt(i));
    } else {
      parts.push(s.charAt(i));
    }
    i++;
  }
  return new JsonSlice(parts.join(""), i + 1);
}

function __json_value(s: string, start: usize): JsonSlice {
  let i = start;
  while (i < s.length && s.charCodeAt(i) <= 32) i++;
  if (i >= s.length) return new JsonSlice("", i);
  const ch = <i32>s.charCodeAt(i);
  if (ch == 0x22) return __json_string(s, i);
  if (ch == 0x7B || ch == 0x5B) {
    const close = ch == 0x7B ? 0x7D : 0x5D;
    let depth = 1;
    i++;
    while (i < s.length && depth > 0) {
      const c = <i32>s.charCodeAt(i);
      if (c == 0x22) { const r = __json_string(s, i); i = r.end; continue; }
      if (c == close) depth--;
      else if (c == close) depth++;
      i++;
    }
    return new JsonSlice(s.substring(start, i), i);
  }
  const startPrim = i;
  while (i < s.length && s.charCodeAt(i) > 32 && s.charCodeAt(i) != 0x2C && s.charCodeAt(i) != 0x7D && s.charCodeAt(i) != 0x5D) {
    i++;
  }
  return new JsonSlice(s.substring(startPrim, i), i);
}

function __parse_i32(s: string): i32 {
  return parseInt(s, 10);
}

// ─── Module housekeeping ────────────────────────────────────────────────

export function wasmInit(): void {
  sequenceMap.clear();
  resultPool.length = 0;
}

// ─── Private helpers ────────────────────────────────────────────────────

function __put_result(bytes: Uint8Array): i32 {
  const handle = resultPool.length;
  resultPool.push(bytes);
  return handle;
}

function __read_bytes(ptr: usize, len: usize): Uint8Array {
  if (ptr == 0 || len == 0) return new Uint8Array(0);
  const view = new Uint8Array(<i32>len);
  for (let i = 0; i < <i32>len; i++) {
    view[i] = load<u8>(ptr + i);
  }
  return view;
}

function __to_utf8(str: string): Uint8Array {
  const ab = String.UTF8.encode(str);
  const view = new Uint8Array(ab.byteLength);
  const u8 = new Uint8Array(ab);
  for (let i = 0; i < <i32>view.length; i++) {
    view[i] = u8[i];
  }
  return view;
}
