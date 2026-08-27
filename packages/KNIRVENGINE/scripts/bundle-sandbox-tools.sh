#!/usr/bin/env bash
#
# bundle-sandbox-tools.sh
#
# Copies bubblewrap, Xvfb, and x11vnc (and their transitive shared-library
# dependencies) into a `tools/` directory next to the engine binary, so the
# engine can launch a sandbox without the end user installing those packages
# system-wide.
#
# The engine resolves <exeDir>/tools/<name> before PATH (see api/sandbox_tools.go)
# and sets LD_LIBRARY_PATH=<tools>/lib when it spawns a bundled binary, so the
# shipped libraries are found at runtime. glibc itself is intentionally NOT
# bundled (the host's loader/glibc is used), matching the AppImage convention.
#
# Usage: ./scripts/bundle-sandbox-tools.sh [OUT_DIR]
#   OUT_DIR defaults to ./tools (relative to the engine package root).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Sandbox tools are Linux-only; bwrap does not exist on macOS/Windows.
if [ "$(uname -s)" != "Linux" ]; then
	echo "[bundle-sandbox-tools] skipping: sandbox tools are Linux-only (OS: $(uname -s))"
	exit 0
fi

OUT_DIR="${1:-tools}"
BIN_DIR="$OUT_DIR"
LIB_DIR="$OUT_DIR/lib"

# Native Linux executables used by the sandbox and the real tool consoles.
# Missing entries are deliberately non-fatal: the engine will show its normal
# dependency error and can acquire the corresponding package at runtime.
TOOLS=("bwrap" "Xvfb" "x11vnc" "nsenter" "java" "dotnet" "bpftrace" "tshark" "zeek" "afl-fuzz" "rizin" "proxychains4" "frida-server")

# pip-installed consoles live in the managed venv rather than the system
# Python. If a build environment has provisioned this venv, keep it in the
# shipped bundle and walk the executable dependencies too.
PYTHON_TOOLS=("semgrep" "jwt_tool.py" "frida")
DOTNET_TOOLS=("ilspycmd")

# glibc core libraries are provided by the host; do not bundle them.
GLIBC_CORE_RE='^(ld-linux|libc|libm|libpthread|libdl|librt|libutil|libresolv|libmvec|libnsl|libBrokenLocale|libanl|libthread_db)\.so'

mkdir -p "$LIB_DIR"
cp -f scripts/frida-bridge.py "$BIN_DIR/frida-bridge.py"
chmod 0755 "$BIN_DIR/frida-bridge.py"

# Queue of ELF files whose dependencies still need to be walked.
declare -a QUEUE=()
declare -A SEEN=()

enqueue() {
	local f="$1"
	[ -n "${SEEN[$f]:-}" ] && return
	SEEN[$f]=1
	QUEUE+=("$f")
}

# 1. Locate and copy each top-level tool binary.
missing=()
for tool in "${TOOLS[@]}"; do
	src="$(command -v "$tool" 2>/dev/null || true)"
	if [ -z "$src" ]; then
		echo "[bundle-sandbox-tools] WARNING: '$tool' not found on PATH; skipping (engine will fall back to PATH/auto-install at runtime)." >&2
		missing+=("$tool")
		continue
	fi
	cp -fL "$src" "$BIN_DIR/$tool"
	chmod 0755 "$BIN_DIR/$tool"
	enqueue "$src"
	echo "[bundle-sandbox-tools] bundled $tool -> $BIN_DIR/$tool"
done

# 1b. Preserve an already-provisioned managed Python venv. The directory is
# intentionally copied as a unit because entrypoint scripts reference the venv
# interpreter by relative path. Do not fail a normal native-only build if it is
# absent.
for tool in "${PYTHON_TOOLS[@]}"; do
	src="$OUT_DIR/pyenv/bin/$tool"
	if [ -x "$src" ]; then
		enqueue "$src"
		echo "[bundle-sandbox-tools] bundled managed Python tool $tool"
	else
		echo "[bundle-sandbox-tools] WARNING: managed Python tool '$tool' not found in $OUT_DIR/pyenv/bin" >&2
	fi
done

# ILSpy is a managed dotnet global tool. Preserve its tool-path directory in
# exactly the same way as the Python venv; the backend resolves it before PATH.
for tool in "${DOTNET_TOOLS[@]}"; do
	src="$OUT_DIR/dotnettools/$tool"
	if [ -x "$src" ]; then
		enqueue "$src"
		echo "[bundle-sandbox-tools] bundled managed .NET tool $tool"
	else
		echo "[bundle-sandbox-tools] WARNING: managed .NET tool '$tool' not found in $OUT_DIR/dotnettools" >&2
	fi
done

if [ "${#missing[@]}" -gt 0 ]; then
	echo "[bundle-sandbox-tools] NOTE: ${missing[*]} were not bundled (not installed on this build host)." >&2
fi

# 2. Walk the transitive dependency closure, copying every non-glibc .so.
while [ ${#QUEUE[@]} -gt 0 ]; do
	cur="${QUEUE[0]}"
	QUEUE=("${QUEUE[@]:1}")

	# Managed Python/.NET launchers are scripts, not ELF files. They are still
	# copied with their managed directory, but have no shared-library closure to
	# walk. Keep an empty ldd result non-fatal under `set -euo pipefail`.
	deps="$(ldd "$cur" 2>/dev/null | awk '/=>/ {print $3}' | grep -E '^/' || true)"
	for dep in $deps; do
		[ -e "$dep" ] || continue
		base="$(basename "$dep")"
		if printf '%s' "$base" | grep -Eq "$GLIBC_CORE_RE"; then
			continue
		fi
		[ -n "${SEEN[$dep]:-}" ] && continue
		cp -fL "$dep" "$LIB_DIR/$base"
		enqueue "$dep"
	done
done

# 3. Optionally make the binaries self-locating via patchelf (best effort; the
#    engine also sets LD_LIBRARY_PATH at spawn, so this is a hardening extra).
if command -v patchelf >/dev/null 2>&1; then
	for tool in "${TOOLS[@]}"; do
		patchelf --set-rpath '$ORIGIN/lib' "$BIN_DIR/$tool" 2>/dev/null || true
	done
	echo "[bundle-sandbox-tools] applied rpath via patchelf"
fi

# 4. Mirror the bundle into dist/ so the cross-compiled binaries (dist/knirv-engine-*)
#    also find their tools directory at runtime (exeDir == dist/).
if [ -d dist ]; then
	rm -rf dist/tools
	cp -a "$OUT_DIR" dist/tools
	echo "[bundle-sandbox-tools] mirrored bundle to dist/tools"
fi

# Namespace entry, eBPF tracing, and Frida's process-attach are the only
# runtime helpers that need kernel capabilities.
#
# Two things make this block more than a one-line `if`:
#
# 1. Every step above re-copies these binaries fresh from PATH, which drops
#    any capability a previous `make install-sandbox-privileges` run granted.
#    A marker file records that grant was requested so an ordinary `make
#    build` reapplies it automatically instead of silently shipping a
#    capability-less binary (the historic bug: `nsenter -C` failing with
#    "Operation not permitted" after any rebuild that followed a working
#    install-sandbox-privileges run).
# 2. `security.capability` is a privileged xattr: an unprivileged `cp -a`
#    (the mirror step above) cannot carry it, so dist/tools' copies need
#    their own setcap pass rather than inheriting it from tools/.
#
# KNIRVENGINE_SET_FILE_CAPS=1 is the explicit, interactive first grant
# (`make install-sandbox-privileges`); the marker makes subsequent grants
# non-interactive and safe to skip loudly (not silently) when they fail.
MARKER="$OUT_DIR/.setcap-enabled"
explicit_grant=0
if [ "${KNIRVENGINE_SET_FILE_CAPS:-0}" = "1" ]; then
	explicit_grant=1
fi

if [ "$explicit_grant" = "1" ] || [ -f "$MARKER" ]; then
	if ! command -v setcap >/dev/null 2>&1; then
		echo "[bundle-sandbox-tools] ERROR: setcap is required to grant sandbox capabilities" >&2
		exit 1
	fi

	if [ "$explicit_grant" = "1" ]; then
		sudo_cmd=(sudo)
	else
		sudo_cmd=(sudo -n)
	fi

	grant_failed=0
	grant_one() {
		local target="$1"
		shift
		if [ -x "$target" ]; then
			if "${sudo_cmd[@]}" setcap "$@" "$target"; then
				echo "[bundle-sandbox-tools] granted capabilities to $target"
			else
				grant_failed=1
			fi
		fi
	}
	grant_caps() {
		local dir="$1"
		grant_one "$dir/nsenter" cap_sys_admin,cap_sys_ptrace,cap_sys_chroot+ep
		grant_one "$dir/bpftrace" cap_bpf,cap_perfmon,cap_sys_admin+ep
		grant_one "$dir/frida-server" cap_sys_ptrace+ep
	}

	grant_caps "$BIN_DIR"
	if [ -d dist/tools ]; then
		grant_caps "dist/tools"
	fi

	if [ "$grant_failed" = "1" ]; then
		echo "[bundle-sandbox-tools] WARNING: could not reapply sandbox capabilities non-interactively." >&2
		echo "[bundle-sandbox-tools] WARNING: nsenter/bpftrace/frida-server attach will fail (e.g. 'Operation not permitted') until you re-run: make install-sandbox-privileges" >&2
	else
		touch "$MARKER"
	fi
fi

# 5. Report.
lib_count="$(find "$LIB_DIR" -maxdepth 1 -type f | wc -l)"
bin_size="$(du -sh "$BIN_DIR" 2>/dev/null | cut -f1)"
echo "[bundle-sandbox-tools] done: $BIN_DIR ($bin_size, $lib_count shared libs)"
