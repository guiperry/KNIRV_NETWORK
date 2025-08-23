let i;
function S(t) {
  const e = i.__externref_table_alloc();
  return i.__wbindgen_export_2.set(e, t), e;
}
function T(t, e) {
  try {
    return t.apply(this, e);
  } catch (n) {
    const r = S(n);
    i.__wbindgen_exn_store(r);
  }
}
const x = typeof TextDecoder < "u" ? new TextDecoder("utf-8", { ignoreBOM: true, fatal: true }) : { decode: () => {
  throw Error("TextDecoder not available");
} };
typeof TextDecoder < "u" && x.decode();
let b = null;
function u() {
  return (b === null || b.byteLength === 0) && (b = new Uint8Array(i.memory.buffer)), b;
}
function l(t, e) {
  return t = t >>> 0, x.decode(u().subarray(t, t + e));
}
function p(t) {
  const e = typeof t;
  if (e == "number" || e == "boolean" || t == null) return `${t}`;
  if (e == "string") return `"${t}"`;
  if (e == "symbol") {
    const _ = t.description;
    return _ == null ? "Symbol" : `Symbol(${_})`;
  }
  if (e == "function") {
    const _ = t.name;
    return typeof _ == "string" && _.length > 0 ? `Function(${_})` : "Function";
  }
  if (Array.isArray(t)) {
    const _ = t.length;
    let s = "[";
    _ > 0 && (s += p(t[0]));
    for (let o = 1; o < _; o++) s += ", " + p(t[o]);
    return s += "]", s;
  }
  const n = /\[object ([^\]]+)\]/.exec(toString.call(t));
  let r;
  if (n && n.length > 1) r = n[1];
  else return toString.call(t);
  if (r == "Object") try {
    return "Object(" + JSON.stringify(t) + ")";
  } catch {
    return "Object";
  }
  return t instanceof Error ? `${t.name}: ${t.message}
${t.stack}` : r;
}
let f = 0;
const y = typeof TextEncoder < "u" ? new TextEncoder("utf-8") : { encode: () => {
  throw Error("TextEncoder not available");
} }, W = typeof y.encodeInto == "function" ? function(t, e) {
  return y.encodeInto(t, e);
} : function(t, e) {
  const n = y.encode(t);
  return e.set(n), { read: t.length, written: n.length };
};
function g(t, e, n) {
  if (n === void 0) {
    const c = y.encode(t), a = e(c.length, 1) >>> 0;
    return u().subarray(a, a + c.length).set(c), f = c.length, a;
  }
  let r = t.length, _ = e(r, 1) >>> 0;
  const s = u();
  let o = 0;
  for (; o < r; o++) {
    const c = t.charCodeAt(o);
    if (c > 127) break;
    s[_ + o] = c;
  }
  if (o !== r) {
    o !== 0 && (t = t.slice(o)), _ = n(_, r, r = o + t.length * 3, 1) >>> 0;
    const c = u().subarray(_ + o, _ + r), a = W(t, c);
    o += a.written, _ = n(_, r, o, 1) >>> 0;
  }
  return f = o, _;
}
let d = null;
function m() {
  return (d === null || d.buffer.detached === true || d.buffer.detached === void 0 && d.buffer !== i.memory.buffer) && (d = new DataView(i.memory.buffer)), d;
}
function M(t, e) {
  const n = e(t.length * 1, 1) >>> 0;
  return u().set(t, n / 1), f = t.length, n;
}
function j() {
  i.main();
}
function O(t, e, n, r) {
  i.closure22_externref_shim(t, e, n, r);
}
const h = typeof FinalizationRegistry > "u" ? { register: () => {
}, unregister: () => {
} } : new FinalizationRegistry((t) => i.__wbg_hrmcognitive_free(t >>> 0, 1));
class D {
  __destroy_into_raw() {
    const e = this.__wbg_ptr;
    return this.__wbg_ptr = 0, h.unregister(this), e;
  }
  free() {
    const e = this.__destroy_into_raw();
    i.__wbg_hrmcognitive_free(e, 0);
  }
  constructor() {
    const e = i.hrmcognitive_new();
    return this.__wbg_ptr = e >>> 0, h.register(this, this.__wbg_ptr, this), this;
  }
  initialize_modules(e, n) {
    i.hrmcognitive_initialize_modules(this.__wbg_ptr, e, n);
  }
  process_cognitive_input(e) {
    let n, r;
    try {
      const _ = g(e, i.__wbindgen_malloc, i.__wbindgen_realloc), s = f, o = i.hrmcognitive_process_cognitive_input(this.__wbg_ptr, _, s);
      return n = o[0], r = o[1], l(o[0], o[1]);
    } finally {
      i.__wbindgen_free(n, r, 1);
    }
  }
  get_model_info() {
    let e, n;
    try {
      const r = i.hrmcognitive_get_model_info(this.__wbg_ptr);
      return e = r[0], n = r[1], l(r[0], r[1]);
    } finally {
      i.__wbindgen_free(e, n, 1);
    }
  }
  load_weights(e) {
    const n = M(e, i.__wbindgen_malloc), r = f;
    return i.hrmcognitive_load_weights(this.__wbg_ptr, n, r) !== 0;
  }
  load_weights_from_url(e) {
    const n = g(e, i.__wbindgen_malloc, i.__wbindgen_realloc), r = f;
    return i.hrmcognitive_load_weights_from_url(this.__wbg_ptr, n, r);
  }
  get_weights_info() {
    let e, n;
    try {
      const r = i.hrmcognitive_get_weights_info(this.__wbg_ptr);
      return e = r[0], n = r[1], l(r[0], r[1]);
    } finally {
      i.__wbindgen_free(e, n, 1);
    }
  }
  connect_to_desktop(e) {
    const n = g(e, i.__wbindgen_malloc, i.__wbindgen_realloc), r = f;
    return i.hrmcognitive_connect_to_desktop(this.__wbg_ptr, n, r) !== 0;
  }
  send_host_message(e, n) {
    let r, _;
    try {
      const s = g(e, i.__wbindgen_malloc, i.__wbindgen_realloc), o = f, c = g(n, i.__wbindgen_malloc, i.__wbindgen_realloc), a = f, w = i.hrmcognitive_send_host_message(this.__wbg_ptr, s, o, c, a);
      return r = w[0], _ = w[1], l(w[0], w[1]);
    } finally {
      i.__wbindgen_free(r, _, 1);
    }
  }
  get_pending_messages() {
    let e, n;
    try {
      const r = i.hrmcognitive_get_pending_messages(this.__wbg_ptr);
      return e = r[0], n = r[1], l(r[0], r[1]);
    } finally {
      i.__wbindgen_free(e, n, 1);
    }
  }
  set_personality_metric(e, n) {
    const r = g(e, i.__wbindgen_malloc, i.__wbindgen_realloc), _ = f;
    i.hrmcognitive_set_personality_metric(this.__wbg_ptr, r, _, n);
  }
  get_personality_profile() {
    let e, n;
    try {
      const r = i.hrmcognitive_get_personality_profile(this.__wbg_ptr);
      return e = r[0], n = r[1], l(r[0], r[1]);
    } finally {
      i.__wbindgen_free(e, n, 1);
    }
  }
  update_user_feedback(e, n) {
    const r = g(e, i.__wbindgen_malloc, i.__wbindgen_realloc), _ = f;
    i.hrmcognitive_update_user_feedback(this.__wbg_ptr, r, _, n);
  }
  get_cognitive_state() {
    let e, n;
    try {
      const r = i.hrmcognitive_get_cognitive_state(this.__wbg_ptr);
      return e = r[0], n = r[1], l(r[0], r[1]);
    } finally {
      i.__wbindgen_free(e, n, 1);
    }
  }
  set_processing_mode(e) {
    const n = g(e, i.__wbindgen_malloc, i.__wbindgen_realloc), r = f;
    i.hrmcognitive_set_processing_mode(this.__wbg_ptr, n, r);
  }
  clear_memory_buffer() {
    i.hrmcognitive_clear_memory_buffer(this.__wbg_ptr);
  }
  get_memory_summary() {
    let e, n;
    try {
      const r = i.hrmcognitive_get_memory_summary(this.__wbg_ptr);
      return e = r[0], n = r[1], l(r[0], r[1]);
    } finally {
      i.__wbindgen_free(e, n, 1);
    }
  }
}
async function E(t, e) {
  if (typeof Response == "function" && t instanceof Response) {
    if (typeof WebAssembly.instantiateStreaming == "function") try {
      return await WebAssembly.instantiateStreaming(t, e);
    } catch (r) {
      if (t.headers.get("Content-Type") != "application/wasm") console.warn("`WebAssembly.instantiateStreaming` failed because your server does not serve Wasm with `application/wasm` MIME type. Falling back to `WebAssembly.instantiate` which is slower. Original error:\n", r);
      else throw r;
    }
    const n = await t.arrayBuffer();
    return await WebAssembly.instantiate(n, e);
  } else {
    const n = await WebAssembly.instantiate(t, e);
    return n instanceof WebAssembly.Instance ? { instance: n, module: t } : n;
  }
}
function A() {
  const t = {};
  return t.wbg = {}, t.wbg.__wbg_call_7cccdd69e0791ae2 = function() {
    return T(function(e, n, r) {
      return e.call(n, r);
    }, arguments);
  }, t.wbg.__wbg_log_bc9908cbe3049010 = function(e, n) {
    console.log(l(e, n));
  }, t.wbg.__wbg_new_23a2665fac83c611 = function(e, n) {
    try {
      var r = { a: e, b: n }, _ = (o, c) => {
        const a = r.a;
        r.a = 0;
        try {
          return O(a, r.b, o, c);
        } finally {
          r.a = a;
        }
      };
      return new Promise(_);
    } finally {
      r.a = r.b = 0;
    }
  }, t.wbg.__wbg_now_807e54c39636c349 = function() {
    return Date.now();
  }, t.wbg.__wbindgen_debug_string = function(e, n) {
    const r = p(n), _ = g(r, i.__wbindgen_malloc, i.__wbindgen_realloc), s = f;
    m().setInt32(e + 4, s, true), m().setInt32(e + 0, _, true);
  }, t.wbg.__wbindgen_init_externref_table = function() {
    const e = i.__wbindgen_export_2, n = e.grow(4);
    e.set(0, void 0), e.set(n + 0, void 0), e.set(n + 1, null), e.set(n + 2, true), e.set(n + 3, false);
  }, t.wbg.__wbindgen_throw = function(e, n) {
    throw new Error(l(e, n));
  }, t;
}
function v(t, e) {
  return i = t.exports, R.__wbindgen_wasm_module = e, d = null, b = null, i.__wbindgen_start(), i;
}
function k(t) {
  if (i !== void 0) return i;
  typeof t < "u" && (Object.getPrototypeOf(t) === Object.prototype ? { module: t } = t : console.warn("using deprecated parameters for `initSync()`; pass a single object instead"));
  const e = A();
  t instanceof WebAssembly.Module || (t = new WebAssembly.Module(t));
  const n = new WebAssembly.Instance(t, e);
  return v(n, t);
}
async function R(t) {
  if (i !== void 0) return i;
  typeof t < "u" && (Object.getPrototypeOf(t) === Object.prototype ? { module_or_path: t } = t : console.warn("using deprecated parameters for the initialization function; pass a single object instead")), typeof t > "u" && (t = new URL("/assets/knirv_cortex_wasm_bg-DcMOnuJA.wasm", import.meta.url));
  const e = A();
  (typeof t == "string" || typeof Request == "function" && t instanceof Request || typeof URL == "function" && t instanceof URL) && (t = fetch(t));
  const { instance: n, module: r } = await E(await t, e);
  return v(n, r);
}
export {
  D as HRMCognitive,
  R as default,
  k as initSync,
  j as main
};
