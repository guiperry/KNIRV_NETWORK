"""ctypes adapter for the KNIRV C ABI.

Set KNIRV_SDK_LIBRARY to a platform-specific libknirv_sdk artifact. The class
has explicit close() semantics; context-manager use is preferred.
"""
from __future__ import annotations
import ctypes as _ctypes
import json as _json
import os as _os
from pathlib import Path as _Path

class _Bytes(_ctypes.Structure):
    _fields_ = [("ptr", _ctypes.POINTER(_ctypes.c_uint8)), ("len", _ctypes.c_size_t)]

class KnirvError(RuntimeError): pass
class InvalidArgumentError(KnirvError): pass
class AuthenticationError(KnirvError): pass
class TimeoutError(KnirvError): pass
class TransportError(KnirvError): pass
class ApiError(KnirvError): pass
class CryptoError(KnirvError): pass

_STATUS_ERRORS = {1: InvalidArgumentError, 2: AuthenticationError, 3: TimeoutError,
                  4: TransportError, 5: ApiError, 6: CryptoError}

class Client:
    def __init__(self, config: dict | None = None, library: str | None = None):
        library = library or _os.environ.get("KNIRV_SDK_LIBRARY")
        if not library: raise KnirvError("set KNIRV_SDK_LIBRARY or pass library=")
        self._lib = _ctypes.CDLL(str(_Path(library)))
        self._lib.knirv_engine_new.argtypes = [_Bytes, _ctypes.POINTER(_ctypes.c_void_p)]
        self._lib.knirv_engine_call.argtypes = [_ctypes.c_void_p, _Bytes, _ctypes.POINTER(_Bytes)]
        self._lib.knirv_engine_free.argtypes = [_ctypes.c_void_p]
        self._lib.knirv_bytes_free.argtypes = [_Bytes]
        self._lib.knirv_module_bytes.argtypes = [_Bytes, _ctypes.POINTER(_Bytes)]
        raw = _json.dumps(config or {}).encode(); self._config = _ctypes.create_string_buffer(raw)
        self._engine = _ctypes.c_void_p()
        if self._lib.knirv_engine_new(_Bytes(_ctypes.cast(self._config, _ctypes.POINTER(_ctypes.c_uint8)), len(raw)), _ctypes.byref(self._engine)) != 0: raise KnirvError("engine creation failed")
    def close(self) -> None:
        if self._engine: self._lib.knirv_engine_free(self._engine); self._engine = _ctypes.c_void_p()
    def __enter__(self): return self
    def __exit__(self, *_): self.close()
    def __del__(self):
        # Fallback only; callers should use close() or the context manager.
        self.close()
    def call(self, operation: str, payload: dict | None = None) -> dict:
        if not self._engine: raise KnirvError("client is closed")
        raw = _json.dumps({"version": 1, "operation": operation, "payload": payload or {}}).encode(); request = _ctypes.create_string_buffer(raw); out = _Bytes()
        status = self._lib.knirv_engine_call(self._engine, _Bytes(_ctypes.cast(request, _ctypes.POINTER(_ctypes.c_uint8)), len(raw)), _ctypes.byref(out))
        if status != 0: raise _STATUS_ERRORS.get(status, KnirvError)(f"engine call failed with status {status}")
        try: return _json.loads(_ctypes.string_at(out.ptr, out.len))
        finally: self._lib.knirv_bytes_free(out)
    def module_bytes(self, name: str) -> bytes:
        raw = name.encode()
        request = _ctypes.create_string_buffer(raw)
        out = _Bytes()
        status = self._lib.knirv_module_bytes(_Bytes(_ctypes.cast(request, _ctypes.POINTER(_ctypes.c_uint8)), len(raw)), _ctypes.byref(out))
        if status != 0: raise _STATUS_ERRORS.get(status, KnirvError)(f"module lookup failed with status {status}")
        try: return _ctypes.string_at(out.ptr, out.len)
        finally: self._lib.knirv_bytes_free(out)
    def materialize_module(self, name: str, path: str | _Path) -> None:
        _Path(path).write_bytes(self.module_bytes(name))
