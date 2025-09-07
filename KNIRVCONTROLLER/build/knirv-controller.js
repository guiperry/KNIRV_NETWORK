async function instantiate(module, imports = {}) {
  const adaptedImports = {
    env: Object.assign(Object.create(globalThis), imports.env || {}, {
      abort(message, fileName, lineNumber, columnNumber) {
        // ~lib/builtins/abort(~lib/string/String | null?, ~lib/string/String | null?, u32?, u32?) => void
        message = __liftString(message >>> 0);
        fileName = __liftString(fileName >>> 0);
        lineNumber = lineNumber >>> 0;
        columnNumber = columnNumber >>> 0;
        (() => {
          // @external.js
          throw Error(`${message} in ${fileName}:${lineNumber}:${columnNumber}`);
        })();
      },
      "console.log"(text) {
        // ~lib/bindings/dom/console.log(~lib/string/String) => void
        text = __liftString(text >>> 0);
        console.log(text);
      },
    }),
  };
  const { exports } = await WebAssembly.instantiate(module, adaptedImports);
  const memory = exports.memory || imports.env.memory;
  const adaptedExports = Object.setPrototypeOf({
    createAgentCore(id) {
      // assembly/index/createAgentCore(~lib/string/String) => bool
      id = __lowerString(id) || __notnull();
      return exports.createAgentCore(id) != 0;
    },
    initializeAgent() {
      // assembly/index/initializeAgent() => bool
      return exports.initializeAgent() != 0;
    },
    executeAgent(input, _context) {
      // assembly/index/executeAgent(~lib/string/String, ~lib/string/String) => ~lib/string/String
      input = __retain(__lowerString(input) || __notnull());
      _context = __lowerString(_context) || __notnull();
      try {
        return __liftString(exports.executeAgent(input, _context) >>> 0);
      } finally {
        __release(input);
      }
    },
    executeAgentTool(toolName, parameters, _context) {
      // assembly/index/executeAgentTool(~lib/string/String, ~lib/string/String, ~lib/string/String) => ~lib/string/String
      toolName = __retain(__lowerString(toolName) || __notnull());
      parameters = __retain(__lowerString(parameters) || __notnull());
      _context = __lowerString(_context) || __notnull();
      try {
        return __liftString(exports.executeAgentTool(toolName, parameters, _context) >>> 0);
      } finally {
        __release(toolName);
        __release(parameters);
      }
    },
    loadLoraAdapter(adapter) {
      // assembly/index/loadLoraAdapter(~lib/string/String) => bool
      adapter = __lowerString(adapter) || __notnull();
      return exports.loadLoraAdapter(adapter) != 0;
    },
    getAgentStatus() {
      // assembly/index/getAgentStatus() => ~lib/string/String
      return __liftString(exports.getAgentStatus() >>> 0);
    },
    createModel(type) {
      // assembly/index/createModel(~lib/string/String) => bool
      type = __lowerString(type) || __notnull();
      return exports.createModel(type) != 0;
    },
    loadModelWeights(_weightsPtr, weightsLen) {
      // assembly/index/loadModelWeights(usize, i32) => bool
      return exports.loadModelWeights(_weightsPtr, weightsLen) != 0;
    },
    runModelInference(input, _context) {
      // assembly/index/runModelInference(~lib/string/String, ~lib/string/String) => ~lib/string/String
      input = __retain(__lowerString(input) || __notnull());
      _context = __lowerString(_context) || __notnull();
      try {
        return __liftString(exports.runModelInference(input, _context) >>> 0);
      } finally {
        __release(input);
      }
    },
    getModelInfo() {
      // assembly/index/getModelInfo() => ~lib/string/String
      return __liftString(exports.getModelInfo() >>> 0);
    },
    getWasmVersion() {
      // assembly/index/getWasmVersion() => ~lib/string/String
      return __liftString(exports.getWasmVersion() >>> 0);
    },
    getSupportedFeatures() {
      // assembly/index/getSupportedFeatures() => ~lib/string/String
      return __liftString(exports.getSupportedFeatures() >>> 0);
    },
    allocateString(str) {
      // assembly/index/allocateString(~lib/string/String) => usize
      str = __lowerString(str) || __notnull();
      return exports.allocateString(str) >>> 0;
    },
  }, exports);
  function __liftString(pointer) {
    if (!pointer) return null;
    const
      end = pointer + new Uint32Array(memory.buffer)[pointer - 4 >>> 2] >>> 1,
      memoryU16 = new Uint16Array(memory.buffer);
    let
      start = pointer >>> 1,
      string = "";
    while (end - start > 1024) string += String.fromCharCode(...memoryU16.subarray(start, start += 1024));
    return string + String.fromCharCode(...memoryU16.subarray(start, end));
  }
  function __lowerString(value) {
    if (value == null) return 0;
    const
      length = value.length,
      pointer = exports.__new(length << 1, 2) >>> 0,
      memoryU16 = new Uint16Array(memory.buffer);
    for (let i = 0; i < length; ++i) memoryU16[(pointer >>> 1) + i] = value.charCodeAt(i);
    return pointer;
  }
  const refcounts = new Map();
  function __retain(pointer) {
    if (pointer) {
      const refcount = refcounts.get(pointer);
      if (refcount) refcounts.set(pointer, refcount + 1);
      else refcounts.set(exports.__pin(pointer), 1);
    }
    return pointer;
  }
  function __release(pointer) {
    if (pointer) {
      const refcount = refcounts.get(pointer);
      if (refcount === 1) exports.__unpin(pointer), refcounts.delete(pointer);
      else if (refcount) refcounts.set(pointer, refcount - 1);
      else throw Error(`invalid refcount '${refcount}' for reference '${pointer}'`);
    }
  }
  function __notnull() {
    throw TypeError("value must not be null");
  }
  return adaptedExports;
}
export const {
  memory,
  __new,
  __pin,
  __unpin,
  __collect,
  __rtti_base,
  createAgentCore,
  initializeAgent,
  executeAgent,
  executeAgentTool,
  loadLoraAdapter,
  getAgentStatus,
  createModel,
  loadModelWeights,
  runModelInference,
  getModelInfo,
  getWasmVersion,
  getSupportedFeatures,
  allocateString,
  deallocateString,
  wasmInit,
} = await (async url => instantiate(
  await (async () => {
    const isNodeOrBun = typeof process != "undefined" && process.versions != null && (process.versions.node != null || process.versions.bun != null);
    if (isNodeOrBun) { return globalThis.WebAssembly.compile(await (await import("node:fs/promises")).readFile(url)); }
    else { return await globalThis.WebAssembly.compileStreaming(globalThis.fetch(url)); }
  })(), {
  }
))(new URL("knirv-controller.wasm", import.meta.url));
